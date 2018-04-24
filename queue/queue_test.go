package queue

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func onFlush(elems []interface{}) {
	fmt.Printf("elems is: %v\n", elems)
}

func TestQueue_Add(t *testing.T) {
	var val int
	q := New(reflect.TypeOf(val))
	q.StartFlushThread(onFlush, 3, time.Second*3)
	q.Add(1)
	q.Add(2)
	q.Add(3)
	q.StartFlushThread(onFlush, 0, time.Second*2)
	q.Add(4)
	time.Sleep(time.Second * 3)
	q.Add(5)
	select {}
}
