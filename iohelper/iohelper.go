package iohelper

import (
	"bytes"
	"io/ioutil"
	"sync"
)

var bufPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 8192)
		return &b
	},
}

func DefaultGetBuffer() *[]byte {
	return bufPool.Get().(*[]byte)
}

func DefaultPutBuffer(bufp *[]byte) {
	bufPool.Put(bufp)
}

// CatTextFile return file context as string with leading/suffix spaces stripped,
// if file is not read successfully, the defult value is returned instead.
func CatTextFile(fn string, dfl string) string {
	if data, err := ioutil.ReadFile(fn); err == nil {
		data = bytes.TrimSpace(data)
		return string(data)
	}
	return dfl
}
