package install_ee

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tarantool/tt/cli/config"
	"github.com/tarantool/tt/cli/util"
	"github.com/tarantool/tt/cli/version"
)

const (
	eeMacosOldPrefix = "/enterprise-macos/"
	eeCommonPrefix   = "/enterprise/"
	eeSource         = "https://download.tarantool.io/"
	eeReleasePrefix  = "/enterprise/release/"
	eeDebugPrefix    = "/enterprise/debug/"
	eeDevPrefix      = "/enterprise/dev/"
)

// SearchEECtx contains information for packages searching.
type SearchEECtx struct {
	// Dbg is set if debug builds of tarantool-ee must be included in the result of search.
	Dbg bool
	// Dev is set if dev builds of tarantool-ee must be included in the result of search.
	Dev bool
}

// compileVersionRegexp compiles a regular expression for SDK bundle names.
func compileVersionRegexp() (*regexp.Regexp, error) {
	matchRe := ""

	arch, err := util.GetArch()
	if err != nil {
		return nil, err
	}

	osType, err := util.GetOs()
	if err != nil {
		return nil, err
	}

	switch osType {
	case util.OsLinux:
		matchRe = "(.+\\/)(tarantool-enterprise-(?:sdk|bundle)-(?:(?:no)?gc64)?(?:-)?" +
			"(.*r[0-9]{3})?(?:(?:(?:-|\\.)(?:no)?gc64)?(?:.linux.)?" +
			arch + ")?.tar.gz)"
	case util.OsMacos:
		matchRe = "(.+\\/)(tarantool-enterprise-(?:sdk|bundle)-(?:(?:no)?gc64)?(?:-)?" +
			"(.*r[0-9]{3})(?:-|\\.)macos(?:x)?(?:-|\\.)" +
			arch + "\\.tar\\.gz)"
	}

	re := regexp.MustCompile(matchRe)

	return re, nil
}

// getVersions collects a list of all available tarantool-ee
// versions for the host architecture.
func getVersions(data *[]byte) ([]EEVersion, error) {
	versions := []EEVersion{}

	re, err := compileVersionRegexp()
	if err != nil {
		return nil, err
	}

	parsedData := re.FindAllStringSubmatch(strings.TrimSpace(string(*data)), -1)
	if len(parsedData) == 0 {
		return nil, fmt.Errorf("no packages for this OS")
	}

	for _, entry := range parsedData {
		version, err := version.GetVersionDetails(entry[3])
		if err != nil {
			return nil, err
		}
		version.Tarball = entry[2]
		eeVersion := EEVersion{VersionInfo: version, Prefix: entry[1]}
		versions = append(versions, eeVersion)
	}

	SortEEVersions(versions)

	return versions, nil
}

func FetchVersionsLocal(files []string) ([]EEVersion, error) {
	versions := []EEVersion{}

	re, err := compileVersionRegexp()
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		parsedData := re.FindStringSubmatch(file)
		if len(parsedData) == 0 {
			continue
		}

		version, err := version.GetVersionDetails(parsedData[1])
		if err != nil {
			return nil, err
		}

		version.Tarball = file
		eeVer := EEVersion{VersionInfo: version}
		versions = append(versions, eeVer)
	}

	SortEEVersions(versions)

	return versions, nil
}

// FetchVersions returns all available tarantool-ee versions.
// The result will be sorted in ascending order.
func FetchVersions(searchCtx SearchEECtx, cliOpts *config.CliOpts) ([]EEVersion,
	error) {
	credentials, err := getCreds(cliOpts)
	if err != nil {
		return nil, err
	}

	bundleReferences, err := collectBundleReferences(searchCtx, eeSource, credentials)
	if err != nil {
		return nil, err
	}
	resBody := []byte(strings.Join(bundleReferences, "\n"))

	versions, err := getVersions(&resBody)
	if err != nil {
		return nil, err
	}

	return versions, nil
}

// getShortVersionFromBundleName parses the major and minor version from tarantool-sdk-bundle.
func getShortVersionFromBundleName(bundleName string) (string, error) {
	matchRe := "(\\d\\.\\d+)"
	reg := regexp.MustCompile(matchRe)
	res := reg.FindString(bundleName)
	if res == "" {
		return "", fmt.Errorf("no version found")
	}
	return res, nil
}

// GetTarantoolEE downloads given tarantool-ee bundle into directory.
func GetTarantoolEE(cliOpts *config.CliOpts, bundleName, bundleSource string, dst string) error {
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		return fmt.Errorf("directory doesn't exist: %s", dst)
	}
	if !util.IsDir(dst) {
		return fmt.Errorf("incorrect path: %s", dst)
	}
	credentials, err := getCreds(cliOpts)
	if err != nil {
		return err
	}

	source, err := url.Parse(eeSource)
	if err != nil {
		return err
	}
	source.Path = bundleSource

	source.Path = filepath.Join(source.Path, bundleName)
	client := http.Client{Timeout: 0}
	req, err := http.NewRequest(http.MethodGet, source.String(), http.NoBody)
	if err != nil {
		return err
	}

	req.SetBasicAuth(credentials.username, credentials.password)

	res, err := client.Do(req)
	if err != nil {
		return err
	} else if res.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP request error: %s", http.StatusText(res.StatusCode))
	}

	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	file, err := os.Create(filepath.Join(dst, bundleName))
	if err != nil {
		return err
	}
	file.Write(resBody)
	file.Close()

	return nil
}
