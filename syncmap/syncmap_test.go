package syncmap

import (
	"fmt"
	"reflect"
	"testing"
)

func TestSandbox(t *testing.T) {
	m, err := New(reflect.TypeOf(""), reflect.TypeOf(""))
	if err != nil {
		t.Fatal(err)
	}
	m.Add("k1", "v1")
	m.Add("k2", "v2")
	m.Add("k1", nil)
	m.Add("k3", "v3")
	m.Add("k4", "v4")
	if err = m.Add(1, nil); err != nil {
		fmt.Printf("err: %s\n", err)
	}
	if err = m.Add("k3", 3); err != nil {
		fmt.Printf("err: %s\n", err)
	}
	fmt.Printf("%t %t\n", m.Has("k1"), m.Has("k2"))
	fmt.Printf("%s\n", m.Get("k2").(string))
	fmt.Printf("%+v\n", m.ValueSlice().([]string))
}
