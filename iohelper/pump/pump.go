package pump

import (
	"io"

	"helper/iohelper"
	"helper/iohelper/sink"
	"helper/iohelper/source"
)

// Step pump data from r to w once
func Step(r *source.Source, w *sink.Sink, bufp *[]byte) (n int, err error) {
	n, err = r.Read(*bufp)
	switch {
	case err == io.EOF:
		if n > 0 {
			n, err = w.Write((*bufp)[:n])
			if err != nil {
				return n, err
			}
		}
		return n, io.EOF
	case err != nil:
		return n, err
	default:
		return w.Write((*bufp)[:n])
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
