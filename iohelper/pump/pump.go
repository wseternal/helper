package pump

import (
	"fmt"
	"io"

	"bitbucket.org/wseternal/helper/iohelper"
	"bitbucket.org/wseternal/helper/iohelper/sink"
	"bitbucket.org/wseternal/helper/iohelper/source"
)

var (
	DEBUG = false
)

// Step pump data from r to w once
func Step(r *source.Source, w *sink.Sink, bufp *[]byte) (n int, err error) {
	if DEBUG {
		defer func() {
			fmt.Printf("step: write %d, err: %v\n", n, err)
		}()
	}
	n, err = r.Read(*bufp)
	if DEBUG {
		fmt.Printf("step: read(%s) %d, err: %v\n", r.Name, n, err)
	}

	var toWrite []byte
	switch err {
	case source.DataModifiedNotInPlace:
		err = nil
		toWrite = r.DataFiltered
	case source.EOFDataModifiedNotInPlace:
		err = io.EOF
		toWrite = r.DataFiltered
	default:
		toWrite = (*bufp)[:n]
	}
	switch {
	case err == io.EOF:
		if n > 0 {
			n, err = w.Write(toWrite)
			if err != nil {
				return n, err
			}
		}
		return n, io.EOF
	case err != nil:
		return n, err
	default:
		return w.Write(toWrite)
	}
}

// All pump data form r to w, until EOF or error occurred
// A successful call returns err == nil, not err == EOF,
// that is, it doesn't treat an EOF as error to be reported.
//
// if closeWhenDone is true, the r and w will be closed if
// it's closable.
func All(r *source.Source, w *sink.Sink, closeWhenDone bool) (total int, err error) {
	var n int

	bufp := iohelper.DefaultGetBuffer()
	defer iohelper.DefaultPutBuffer(bufp)
	for {
		n, err = Step(r, w, bufp)
		total += n
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			if closeWhenDone {
				r.Close()
				w.Close()
			}
			return total, err
		}
	}
}
