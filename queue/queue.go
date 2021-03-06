package queue

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/wseternal/helper/logger"
)

type Queue struct {
	sync.RWMutex
	elemType    reflect.Type
	elems       []interface{}
	flushCap    int
	flushTicker *time.Ticker
	f           FlushFunc

	abortChan  chan struct{}
	threadDone chan struct{}
}

type FlushFunc func(elems []interface{})

// New create a queue, elements can be added to the queue,
// if t is not nil, every element added to queue must be the type specified.
// a capability count or timeout could be specified with queue's method
// StartFlushThread, the flush function will be called against the queue
// if capability or timeout condition satisfied.
func New(t reflect.Type) *Queue {
	return &Queue{
		elemType:   t,
		abortChan:  make(chan struct{}),
		threadDone: make(chan struct{}),
		elems:      make([]interface{}, 0),
	}
}

// Add add a value to the queue
// will trigger the FlushFunc immediately is cap exceeded
func (q *Queue) Add(val interface{}) error {
	if val == nil && q.elemType != nil {
		return fmt.Errorf("QueueAdd: pass nil value while require element be specific type: %v\n", q.elemType)
	}
	t := reflect.TypeOf(val)
	if q.elemType != nil && q.elemType != t {
		return fmt.Errorf("QueueAdd: require element type be type: %v, got %v\n", q.elemType, t)
	}
	q.Lock()
	q.elems = append(q.elems, val)
	if q.flushCap > 0 && len(q.elems) >= q.flushCap {
		q.flushQueue()
	}
	q.Unlock()
	return nil
}

func (q *Queue) Flush() {
	q.Lock()
	defer q.Unlock()
	q.flushQueue()
}

// Elements return an interface value to the slice of underline elements;
// if elemType is specified at Queue.New, the return value can be convert to []elemType.
// otherwise, the return value is []interface{}
func (q *Queue) Elements() interface{} {
	if q.elemType == nil {
		return q.elems
	}
	v := reflect.MakeSlice(reflect.SliceOf(q.elemType), len(q.elems), cap(q.elems))
	for i, elem := range q.elems {
		v.Index(i).Set(reflect.ValueOf(elem))
	}
	return v.Interface()
}

// flushQueue must be called after locked
func (q *Queue) flushQueue() {
	count := len(q.elems)
	if count == 0 {
		return
	}
	if q.f != nil {
		q.f(q.elems)
	}
	q.elems = make([]interface{}, 0)
	logger.LogD("queue with %d entries is flushed successfully\n", count)
}

func (q *Queue) flushThread() {
forLooP:
	for {
		select {
		case <-q.abortChan:
			break forLooP
		case <-q.flushTicker.C:
			if q.f != nil {
				q.Lock()
				q.flushQueue()
				q.Unlock()
			}
		}
	}
	if q.flushTicker != nil {
		q.flushTicker.Stop()
	}
	q.threadDone <- struct{}{}
}

// StopFlushThread stop the flush thread corresponding with queue
// all left items will be flushed
func (q *Queue) StopFlushThread() {
	q.Lock()
	defer q.Unlock()

	// todo release the lock quickly, call the flush with cached elements
	if q.f != nil {
		q.flushQueue()
		q.abortChan <- struct{}{}
		<-q.threadDone
		logger.LogI("StopFlushThread: current flush thread ended successfully\n")
	}
	q.f = nil
}

// StartFlushThread star the flush thread corresponding with queue
// cap: if larger than zero, queue will be flushed if cap items added to queue
// t: if larger than zero, queue will be flushed if duration t elpased
func (q *Queue) StartFlushThread(f FlushFunc, cap int, t time.Duration) {
	if f == nil {
		return
	}

	q.StopFlushThread()

	q.Lock()
	defer q.Unlock()

	q.f = f
	if cap > 0 {
		q.flushCap = cap
	}
	if t > 0 {
		q.flushTicker = time.NewTicker(t)
	}
	go q.flushThread()
}
