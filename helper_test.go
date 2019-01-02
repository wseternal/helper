package helper

import (
	"flag"
	"fmt"
	"testing"
)

var fn string

func init() {
	flag.StringVar(&fn, "f", "", "file name to test")
}

func TestFileMode(t *testing.T) {
	fmt.Printf("IsFile: %t\n", IsFile(fn))
	fmt.Printf("IsDir: %t\n", IsDir(fn))
	fmt.Printf("IsSymbolLink: %t\n", IsSymbolLink(fn))
	fmt.Printf("IsUnixSocket: %t\n", IsUnixSocket(fn))
}

func TestSameSliceBackend(t *testing.T) {
	bk := make([]byte, 32)
	bk2 := make([]byte, 8)
	s1 := bk[1:5]
	s2 := bk[:0]
	fmt.Printf("bk[1:5] bk[:0] the same backend: %v\n", SameSliceBackend(s1, s2))
	fmt.Printf("bk[1:5] bk the same backend: %v\n", SameSliceBackend(s1, bk))
	fmt.Printf("bk[:0] bk2 the same backend: %v\n", SameSliceBackend(s2, bk2))
	fmt.Printf("bk[:0] bk2 the same backend: %v\n", SameSliceBackend(s2, bk2))
}

func TestUnixDate(t *testing.T) {
	fmt.Printf("unix timestamp of tody: %d\n", UnixDate(0))
}