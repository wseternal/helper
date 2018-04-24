package codec

import (
	"fmt"
	"testing"
)

type info struct {
	A, B int
}

func TestGobCodec(t *testing.T) {
	o := &info{
		A: 1,
		B: 2,
	}
	out, err := GobEncode(o)
	if err != nil {
		t.Fatal(err)
	}
	o1 := &info{}
	err = GobDecode(out, o1)
	if err != nil {
		t.Fatalf("gob decode failed, error: %s\n", err)
	}
	fmt.Printf("o1 is %#v\n", o1)
}
