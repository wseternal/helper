package iohelper

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
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

func JsonUnmarshalFromFile(fn string, val interface{}) error {
	data, err := ioutil.ReadFile(fn)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, val)
}

func XmlUnmarshalFromFile(fn string, val interface{}) error {
	data, err := ioutil.ReadFile(fn)
	if err != nil {
		return err
	}
	return xml.Unmarshal(data, val)
}
