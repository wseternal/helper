package source

import (
"bytes"
	"encoding/json"
"fmt"
"io"
"os"
"path"






"bitbucket.org/wseternal/helper/iohelper/filter"


)

// Source encapsulate the original io.Reader with filters
type Source struct {
	io.Reader
	filters []filter.Filter
	Name    string
}

var (
	DEBUG = false
)

func (src *Source) Close() (err error) {
	if c, ok := src.Reader.(io.Closer); ok {
		err = c.Close()
	}
	for _, f := range src.filters {
		if c, ok := f.(io.Closer); ok {
			tmp := c.Close()
			if err != nil {
				fmt.Printf("source filter close error: %s\n", tmp)
			}
		}
	}
	return err
}

// Read implements the io.Reader interface
func (src *Source) Read(buffer []byte) (n int, err error) {
	n, err = src.Reader.Read(buffer)
	data := buffer[:n]
	if DEBUG {
		fmt.Printf("source: read(%s) original reader %d, err: %v\n", src.Name, n, err)
	}
	if len(src.filters) == 0 {
		return n, err
	}
	rEOF := false
	if err == io.EOF {
		rEOF = true
	}
	for _, f := range src.filters {
		data, err = f.Process(data, rEOF)
		n = len(data)
		if !(err == nil || err == io.EOF) {
			return n, err
		}
		// a filter may return (0, nil) to indicates that current input data is consumed,
		// and more input data is need for generating output
		if n == 0 && err == nil {
			return 0, nil
		}
	}
	if rEOF {
		err = io.EOF
	}
	copy(buffer, data)
	return n, err
}

// Chain append filters to the source object
func (src *Source) Chain(f ...filter.Filter) *Source {
	if src.filters == nil {
		src.filters = make([]filter.Filter, len(f))
		copy(src.filters, f)
	} else {
		src.filters = append(src.filters, f...)
	}
	return src
}

// NewString return a source with given string
func NewString(p string) *Source {
	return &Source{
		Reader: bytes.NewReader([]byte(p)),
	}
}

// NewBytes return a source with given bytes
func NewBytes(p []byte) *Source {
	return &Source{
		Reader: bytes.NewReader(p),
	}
}

// NewFile return a source with given file
func NewFile(fn string) (*Source, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	return &Source{
		Reader: f,
		Name:   path.Base(fn),
	}, nil
}

// New return a source encapsulates the given io.Reader
func New(r io.Reader) *Source {
	return &Source{
		Reader: r,
	}
}

func NewJsonFrom(o interface{}) (*Source, error) {
	data, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}
	return NewBytes(data), nil
}
