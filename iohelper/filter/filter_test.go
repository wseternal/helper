// filter_test is the external test for filter, to avoid recursive import
// as source imports filter

package filter_test

import (
	"crypto/sha256"
	"fmt"
	. "bitbucket.org/wseternal/helper/iohelper/filter"
	"bitbucket.org/wseternal/helper/iohelper/pump"
	"bitbucket.org/wseternal/helper/iohelper/sink"
	"bitbucket.org/wseternal/helper/iohelper/source"
	"io"
	"reflect"
	"testing"
)

func TestHashFilter(t *testing.T) {
	input := "hello world"
	sha256SumString := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	src := source.NewString(input).Chain(NewHash(sha256.New(), false), ToHex)
	snk := sink.NewBuffer()
	pump.All(src, snk, true)
	out := string(snk.Bytes())
	fmt.Printf("%v -> sha256sum -> ToHex -> %v\n", input, out)
	if out != sha256SumString {
		t.Errorf("computed sum %v is not as expected: %v\n", out, sha256SumString)
	}
}

func TestOnTheFlyHashFilter(t *testing.T) {
	input := "hello world"
	sha256SumString := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"

	h := NewHash(sha256.New(), true)
	src := source.NewString(input).Chain(h)
	snk := sink.NewDiscard()
	pump.All(src, snk, true)

	src = source.NewBytes(h.Sum(nil))
	snk = sink.NewBuffer().Chain(ToHex)
	pump.All(src, snk, true)
	out := string(snk.Bytes())
	fmt.Printf("%v -> on-the-fly sha256sum -> ToHex -> %v\n", input, out)
	if out != sha256SumString {
		t.Errorf("computed sum %v is not as expected: %v\n", out)
	}
}

func inc1(p []byte, eof bool) (out []byte, err error) {
	for i, v := range p {
		p[i] = v + 1
	}
	if eof {
		err = io.EOF
	}
	return p, err
}

func dec1(p []byte, eof bool) (out []byte, err error) {
	for i, v := range p {
		p[i] = v - 1
	}
	if eof {
		err = io.EOF
	}
	return p, err
}

var data = []byte{1, 2, 3, 4}

func TestIncDecFilter(t *testing.T) {
	snk := sink.NewBuffer()
	src := source.NewBytes(data)
	pump.All(src.Chain(Func(inc1), Func(dec1)), snk, true)
	out := snk.Bytes()
	fmt.Printf("%v -> inc1 -> dec1 -> %v\n", data, out)
	if !reflect.DeepEqual(data, snk.Bytes()) {
		t.Errorf("%v is not equal to %v\n", data, out)
	}
}

func discard(p []byte, eof bool) (out []byte, err error) {
	if eof {
		err = io.EOF
	}
	return nil, err
}

func TestNullFilter(t *testing.T) {
	snk := sink.NewBuffer()
	src := source.NewBytes(data).Chain(Func(inc1))
	pump.All(src.Chain(Func(discard)), snk, true)
	out := snk.Bytes()
	fmt.Printf("%v -> inc1 -> null -> %v\n", data, out)
	if len(snk.Bytes()) != 0 {
		t.Errorf("error, null filter shall return empty [] instead of %v for any input\n", out)
	}
}
