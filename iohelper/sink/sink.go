package sink

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"bitbucket.org/wseternal/helper/iohelper/filter"
)

// Sink encapsulates a io.Writer
type Sink struct {
	io.Writer
	filters []filter.Filter
	Name string
}

// Buffer provides interface of Bytes()
type Buffer interface {
	Bytes() []byte
}

// Chain prepend filters to the original io.Writer
// Pay careful attention: sink filters shall only process
// the data on the fly, as there is no EOF flag in the write process.
// EOF only exists in read process (source filters).
func (snk *Sink) Chain(f ...filter.Filter) *Sink {
	if snk.filters == nil {
		snk.filters = make([]filter.Filter, len(f))
		copy(snk.filters, f)
	} else {
		snk.filters = append(snk.filters, f...)
	}
	return snk
}

// Close encapsulate close logic for sink and related filters
func (snk *Sink) Close() (err error) {
	for _, f := range snk.filters {
		if c, ok := f.(io.Closer); ok {
			tmp := c.Close()
			if err != nil {
				fmt.Printf("sink filter close error: %s\n", tmp)
			}
		}
	}
	if c, ok := snk.Writer.(io.Closer); ok {
		err = c.Close()
	}
	return err
}

func (snk *Sink) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	out := p
	for _, f := range snk.filters {
		out, err = f.Process(out, false)
		switch {
		case err != nil:
			return len(out), err
		case err == nil:
			continue
		}
	}
	n, err = snk.Writer.Write(out)
	if len(p) != len(out) || (&out[0] != &p[0]) {
		n = len(p)
	}
	return n, err
}

func (snk *Sink) Bytes() []byte {
	if buffer, ok := snk.Writer.(Buffer); ok {
		return buffer.Bytes()
	}

	if r, ok := snk.Writer.(io.Reader); ok {
		data, err := ioutil.ReadAll(r)
		if err != nil {
			panic(err)
		}
		return data
	} else {
		return nil
	}
}

// NewBuffer return a sink encapsulates a bytes.Buffer
func NewBuffer() *Sink {
	return &Sink{
		Writer: new(bytes.Buffer),
	}
}

// NewFile return a sink with given file
func NewFile(fn string) (*Sink, error) {
	f, err := os.OpenFile(fn, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0640)
	if err != nil {
		return nil, err
	}
	return &Sink{
		Writer: f,
	}, err
}

func NewDiscard() *Sink {
	return New(ioutil.Discard)
}

// New return a sink encapsulates the given io.Writer
func New(w io.Writer) *Sink {
	return &Sink{
		Writer: w,
	}
}
