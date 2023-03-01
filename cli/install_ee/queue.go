package install_ee

import (
	"fmt"
	"sync"
)

const emptyQueueError = "empty queue"

// stringQueue is a queue of strings.
type stringQueue struct {
	mtx *sync.Mutex
	buf []string
}

// Insert inserts the passed value into the queue.
func (q *stringQueue) Insert(data string) {
	q.mtx.Lock()
	q.buf = append(q.buf, data)
	q.mtx.Unlock()
}

// InsertBatch inserts the passed slice of values into the queue.
func (q *stringQueue) InsertBatch(data []string) {
	q.mtx.Lock()
	q.buf = append(q.buf, data...)
	q.mtx.Unlock()
}

// Pop takes the first element from the queue and removes it.
func (q *stringQueue) Pop() (string, error) {
	q.mtx.Lock()
	defer q.mtx.Unlock()
	if len(q.buf) == 0 {
		return "", fmt.Errorf(emptyQueueError)
	}
	el := q.buf[0]
	q.buf = q.buf[1:]
	return el, nil
}

// GetAll returns all elements of queue.
func (q *stringQueue) GetAll() ([]string, error) {
	q.mtx.Lock()
	defer q.mtx.Unlock()
	if len(q.buf) == 0 {
		return nil, fmt.Errorf(emptyQueueError)
	}
	return q.buf, nil
}

// Len returns the length of queue.
func (q *stringQueue) Len() int {
	q.mtx.Lock()
	el := len(q.buf)
	defer q.mtx.Unlock()
	return el
}
