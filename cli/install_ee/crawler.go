package install_ee

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/containerd/containerd/pkg/atomic"
	"github.com/tarantool/tt/cli/util"
	"golang.org/x/net/html"
)

// getHref is a helper function to pull the href attribute from a Token.
func getHref(token html.Token) (string, bool) {
	for _, a := range token.Attr {
		if a.Key == "href" {
			return a.Val, true
		}
	}
	return "", false
}

// findReferences scans the returned content from the passed URL, parses it and returns URLs
// that contains bundles, URLs that can be crawled or an error if occurred.
func findReferences(searchCtx SearchEECtx, sourceUrl *url.URL,
	req *http.Request, client *http.Client) ([]string, []string, error) {
	var err error

	req.URL = sourceUrl
	res, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	} else if res.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("HTTP request error: %s", http.StatusText(res.StatusCode))
	}

	body := res.Body
	defer body.Close()

	downloadUrls := make([]string, 0)
	traverseUrls := make([]string, 0)

	var avoidOs string
	osID, err := util.GetOs()
	if err != nil {
		return nil, nil, err
	}
	if osID == util.OsLinux {
		avoidOs = "macos"
	} else {
		avoidOs = "linux"
	}

	tokenizer := html.NewTokenizer(body)
	for {
		currentTokenizer := tokenizer.Next()
		switch currentTokenizer {
		case html.ErrorToken:
			return downloadUrls, traverseUrls, nil
		case html.StartTagToken:
			token := tokenizer.Token()
			if !(token.Data == "a") {
				continue
			}

			// Extract the href value.
			hrefValue, ok := getHref(token)
			if !ok {
				continue
			}

			isBackLink := strings.Contains(hrefValue, "../")
			isSameLink := strings.Contains(hrefValue, "./")
			isBundleLink := strings.Contains(hrefValue, ".tar.gz")
			isWrongOsLink := strings.Contains(hrefValue, fmt.Sprintf("/%s/", avoidOs))
			isSHA256Link := strings.Contains(hrefValue, ".sha256")
			isDevPrefix := strings.Contains(hrefValue, "/dev/")
			isDebugPrefix := strings.Contains(hrefValue, "/debug/")

			if (!searchCtx.Dev && isDevPrefix) || (!searchCtx.Dbg && isDebugPrefix) {
				continue
			}

			if !isBundleLink && !isBackLink &&
				!isWrongOsLink && !isSameLink {
				traverseUrls = append(traverseUrls, hrefValue)
			} else if isBundleLink && !isBackLink &&
				!isWrongOsLink && !isSHA256Link && !isSameLink {
				downloadUrls = append(downloadUrls, hrefValue)
			}
		}
	}
}

// collectBundleReferences crawls all urls from the passed bundle source, collects references
// containing sdk bundles for the host system and returns them in a slice of strings.
func collectBundleReferences(searchCtx SearchEECtx, baseUrl string,
	credentials userCredentials) ([]string, error) {
	bundleRefsQueue := &stringQueue{mtx: &sync.Mutex{}, buf: make([]string, 0)}
	prefixQueue := &stringQueue{mtx: &sync.Mutex{}, buf: make([]string, 0)}

	osType, err := util.GetOs()
	if err != nil {
		return nil, err
	}
	if osType == util.OsMacos {
		prefixQueue.InsertBatch([]string{eeReleasePrefix, eeDebugPrefix, eeDevPrefix,
			eeMacosOldPrefix})
	} else {
		prefixQueue.Insert(eeCommonPrefix)
	}

	eeUrl, err := url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}

	vProcCount := runtime.GOMAXPROCS(0) * 5

	// errChan is a channel for receiving errors from crawling goroutines.
	errChan := make(chan error, vProcCount)

	// freeAtomicSl is a slice of atomic values for controlling the state of workers.
	freeAtomicSl := make([]atomic.Bool, vProcCount)
	for i := range freeAtomicSl {
		freeAtomicSl[i] = atomic.NewBool(true)
	}

	// prefixChanSl is a slice of channels with relative urls that is used by
	// workers goroutines for receiving a new url for crawling.
	prefixChanSl := make([]chan string, vProcCount)
	for i := range prefixChanSl {
		prefixChanSl[i] = make(chan string, 1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for i := 0; i < vProcCount; i++ {
		go crawlWorker(ctx, searchCtx, prefixChanSl[i], prefixQueue, bundleRefsQueue,
			eeUrl.String(), errChan, freeAtomicSl[i], credentials)
	}

	// This loop is needed for managing workers. Any worker puts found
	// references to the prefix queue and waits for a new prefix to be
	// received from the prefix channel. This loop implements the logic
	// of moving prefixes from the queue to the channels of free workers.
	// If all workers have done their work and there is no more prefixes
	// in the queue, the loop will stop all the goroutines by canceling
	// the context.
	for {
		select {
		case err := <-errChan:
			cancel()
			return nil, err
		default:
		}
		countFree := 0
		for i := range freeAtomicSl {
			if freeAtomicSl[i].IsSet() {
				prefix, err := prefixQueue.Pop()
				if err != nil {
					countFree++
					if countFree == vProcCount {
						cancel()
						return bundleRefsQueue.GetAll()
					}
				} else {
					freeAtomicSl[i].Unset()
					prefixChanSl[i] <- prefix
				}
			}
		}
	}
}

// crawlWorker is a worker function that gets a new relative url from the prefix channel,
// scans it and fills a queue with found prefixes and a queue with found bundle references.
func crawlWorker(ctx context.Context, searchCtx SearchEECtx, prefixChan chan string,
	prefixQueue, resultQueue *stringQueue, source string, errChan chan error,
	freeAtomic atomic.Bool, credentials userCredentials) {
	for {
		select {
		case <-ctx.Done():
			return
		case prefix := <-prefixChan:
			client := &http.Client{Timeout: 25 * time.Second}

			eeUrl, err := url.Parse(source)
			if err != nil {
				errChan <- err
				return
			}

			eeUrl.Path = prefix
			newReq, err := http.NewRequest(http.MethodGet, eeSource, http.NoBody)
			if err != nil {
				errChan <- err
				return
			}
			newReq.URL = eeUrl
			newReq.SetBasicAuth(credentials.username, credentials.password)

			foundReferences, crawlUrls, err := findReferences(searchCtx, eeUrl, newReq, client)
			if err != nil {
				errChan <- err
				return
			}
			prefixQueue.InsertBatch(crawlUrls)
			resultQueue.InsertBatch(foundReferences)

			freeAtomic.Set()
		}
	}
}
