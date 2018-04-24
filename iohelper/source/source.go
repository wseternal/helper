package source

import (
	"bytes"
	"io"

	"encoding/json"
	"fmt"
	"helper/iohelper/filter"
	"os"
	"path"
)

// Source encapsulate the original io.Reader with filters
type Source struct {
	io.Reader
	rEOF    bool
	filters []filter.Filter
	buf     bytes.Buffer
	Name    string
}

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
func (src *Source) Read(p []byte) (n int, err error) {
	n, err = src.Reader.Read(p)
	out := p[:n]
	if len(src.filters) == 0 {
		return n, err
	}
	/*
	* rEOF: whether the original r is read to EOF
	* when there are filters, any filter may consume
	* the input and return 0, nil to indicate that
	* it need more input to generate valid output.
	* However, when src.rEOF is true, all chained
	* filters must finish their job and generate the
	* final output.
	 */
	if !src.rEOF {
		if err == io.EOF {
			src.rEOF = true
		}
		for _, f := range src.filters {
			switch {
			case err == io.EOF:
				out, err = f.Process(out, true)
				n = len(out)
			case err != nil:
				// non-eof error occurred, return directly
				return n, err
			case err == nil:
				if n > 0 {
					out, err = f.Process(out, false)
					n = len(out)
				} else {
					/* a filter return 0, nil, which means the filter consumed all input bytes
					and still need more to generate output, no need to process more filters
					*/
					return 0, nil
				}
			}
		}
		if n == 0 {
			return 0, nil
		}
		src.buf.Write(out)
	}
	n, err = src.buf.Read(p)
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
