package boltdb

import (
	"fmt"
	"testing"
)

func TestNew(t *testing.T) {
	db, err := New("./test.db", false)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Put(nil, []byte("k1"), []byte("val1")); err != nil {
		t.Fatalf("put failed, %s\n", err)
	}
	val, err := db.Get(nil, []byte("k1"))
	if err != nil {
		t.Fatalf("get failed, %s\n", err)
	}
	fmt.Printf("val is %s\n", string(val))
}
