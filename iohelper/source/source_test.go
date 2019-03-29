package source_test

import (
	"fmt"
	"github.com/wseternal/helper/iohelper/pump"
	"github.com/wseternal/helper/iohelper/sink"
	. "github.com/wseternal/helper/iohelper/source"
	"reflect"
	"testing"
)

func TestSimpleSource(t *testing.T) {
	var data = []byte{1, 2, 3, 4}
	snk := sink.NewBuffer()
	pump.All(NewBytes(data), snk, true)
	out := snk.Bytes()
	fmt.Printf("%v -> raw cp -> %v\n", data, out)
	if !reflect.DeepEqual(data, out) {
		t.Errorf("error, %v is not equal to %v\n", data, out)
	}
}
