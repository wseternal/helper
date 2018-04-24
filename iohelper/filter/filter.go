package filter

import (
	"encoding/hex"
	"hash"
	"io"
)

// Filter interface for customized actions on each chunk of source or sink
// Process may return 0, nil to indicate that the input data is consumed,
// and more data is need to generate valid output.
// when eof parameter is true, which means the no more data would be provided,
// valid output must be generated in this case, the err returned must be io.EOF
// if no other internal error occurred,
type Filter interface {
	Process(p []byte, eof bool) (out []byte, err error)
}

// Func wrap a func to be used in where the Filter interface is required
type Func func(p []byte, eof bool) (out []byte, err error)

// Process is the method to satisfy the Filter interface requirement
func (f Func) Process(p []byte, eof bool) (out []byte, err error) {
	return f(p, eof)
}

// Hash wrap a hash.Hash to digest the read chunks
type Hash struct {
	hash.Hash
	onTheFly bool
}

// NewHash return a hash filter
// if onTheFly is true, the hash filter shall snoop the data in background,
// without consume them. otherwise, the filter consume all input data,
// and output the hashsum when eof of Process is True
func NewHash(h hash.Hash, onTheFly bool) *Hash {
	return &Hash{
		Hash:     h,
		onTheFly: onTheFly,
	}
}

// Process is the method to satisfy the Filter interface requirement
func (h *Hash) Process(p []byte, eof bool) (out []byte, err error) {
	if h.onTheFly {
		if len(p) > 0 {
			h.Write(p)
		}
		return p, nil
	}
	if eof {
		out = h.Sum(nil)
		return out, io.EOF
	}
	h.Write(p)
	return out, nil
}

func toHex(p []byte, eof bool) (out []byte, err error) {
	if eof {
		err = io.EOF
	}
	if len(p) > 0 {
		out = make([]byte, hex.EncodedLen(len(p)))
		hex.Encode(out, p)
	}
	return out, err
}

// ToHex filter to convert binary string to hex string
var ToHex = Func(toHex)
