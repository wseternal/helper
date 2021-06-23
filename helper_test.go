package helper

import (
	"flag"
	"fmt"
	"reflect"
	"testing"
	"time"
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
	zone, off := time.Now().Zone()
	fmt.Printf("unix timestamp of tody: %d, in zone: %s, zoneoff: %d\n", UnixDate(0), zone, off)
	loc := time.FixedZone("zone+4", 4*3600)
	fmt.Printf("unix timestamp of today of zone 4: %d\n", UnixDateLocation(0, loc))
	fmt.Printf("unix timestamp of today of zone 8: %d\n", UnixDateLocation(0, LocationCST))
}

type TT struct {
	Name string
}

func TestValidStructType(t *testing.T) {
	a := TT{
		Name: "123",
	}

	b := &a
	c := &b
	d := &c

	err := ValidStructType(b, reflect.TypeOf(TT{}), 1)
	if err != nil {
		t.Logf("%s\n", err)
	}

	err = ValidStructType(b, reflect.TypeOf(TT{}), 1)
	if err != nil {
		t.Logf("%s\n", err)
	}
	err = ValidStructType(c, reflect.TypeOf(TT{}), 2)
	if err != nil {
		t.Logf("%s\n", err)
	}
	err = ValidStructType(d, reflect.TypeOf(TT{}), 2)
	if err != nil {
		t.Fatalf("%s\n", err)
	}
}

func TestNextKey(t *testing.T) {
	key := "120f3485fff"
	s, err := NextKey(key)
	fmt.Printf("nextkey of %s %s, %v\n", key, s, err)
}

func TestGetPCFileLine(t *testing.T) {
	fmt.Printf("%v\n", GetPCFileLine())
	fmt.Printf("%v\n", GetPCFileLine())
}