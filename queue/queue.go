package queue

import (
	"sync"
)

type qState int

const (
	empty qState = iota + 1
	full
	middle
)

// Queue is queue
type Queue struct {
	state qState
	head  int
	tail  int
	size  int
	items []interface{}
	lock  sync.Mutex
}

// NewQueue return new queue
func NewQueue(size int) *Queue {
	if size == 0 {
		return nil
	}
	return &Queue{
		state: empty,
		head:  0,
		tail:  0,
		size:  size,
		items: make([]interface{}, size),
	}
}

// Get gets value from queue
func (q *Queue) Get() (v interface{}) {
	for {
		q.lock.Lock()
		if q.state != empty {
			break
		}
		q.lock.Unlock()
	}

	v = q.items[q.head]
	q.head = (q.head + 1) % q.size
	if q.tail == q.head {
		q.state = empty
	} else {
		q.state = middle
	}
	q.lock.Unlock()
	return
}

// Put puts value to queue
func (q *Queue) Put(v interface{}) {
	for {
		q.lock.Lock()
		if q.state != full {
			break
		}
		q.lock.Unlock()
	}

	q.items[q.tail] = v
	q.tail = (q.tail + 1) % q.size
	if q.tail == q.head {
		q.state = full
	} else {
		q.state = middle
	}
	q.lock.Unlock()
}

/*
//Get gets value from queue
func (q *Queue) Get() (v interface{}, found bool) {
	if q.state == empty {
		return
	}
	v = q.items[q.head]
	q.head = (q.head + 1) % q.size
	if q.tail == q.head {
		q.state = empty
	} else {
		q.state = middle
	}
	found = true
	return
}

//Put puts value to queue
func (q *Queue) Put(v interface{}) bool {
	if q.state == full {
		return false
	}
	q.items[q.tail] = v
	q.tail = (q.tail + 1) % q.size
	if q.tail == q.head {
		q.state = full
	} else {
		q.state = middle
	}
	return true
}
*/
