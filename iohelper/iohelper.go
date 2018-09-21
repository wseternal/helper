package iohelper

import (
	"bytes"
	"crypto"
	"fmt"
	"io/ioutil"
	"sync"

	"bitbucket.org/wseternal/helper"
	"bitbucket.org/wseternal/helper/iohelper/filter"
	"bitbucket.org/wseternal/helper/iohelper/pump"
	"bitbucket.org/wseternal/helper/iohelper/sink"
	"bitbucket.org/wseternal/helper/iohelper/source"

	_ "crypto/md5"
	_ "crypto/sha1"
	_ "crypto/sha256"
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

func HashsumFromFile(fn string, h crypto.Hash) ([]byte, error) {
	if !h.Available() {
		return nil, fmt.Errorf("hash function (%d) is not existed", h)
	}
	if !helper.IsFile(fn) {
		return nil, fmt.Errorf("file %s is not existed", fn)
	}
	src, err := source.NewFile(fn)
	if err != nil {
		return nil, err
	}
	sum := filter.NewHash(h.New(), true)
	src.Chain(sum)
	snk := sink.NewDiscard()
	if _, err = pump.All(src, snk, true); err != nil {
		return nil, err
	}
	return sum.Sum(nil), nil
}
