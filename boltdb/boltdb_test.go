package boltdb

import (
	"fmt"
	"testing"
	"github.com/wseternal/helper/kvdb"
)

func TestNew(t *testing.T) {
	db, err := NewBoltDB("./test.db", false)
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

func TestEnumerate(t *testing.T) {
	db, err := NewBoltDB("./test.db", false)
	if err != nil {
		t.Fatal(err)
	}
	db.Put(nil, []byte("k,1"), []byte("val1"))
	db.Put(nil, []byte("k,2"), []byte("val1"))
	db.Put(nil, []byte("k,4"), []byte("val1"))
	db.Put(nil, []byte("k,8"), []byte("val1"))

	db.Range(nil, nil, []byte("k"), []byte("l"), func(iter kvdb.Iterator) {
		fmt.Printf("%s: %s\n", string(iter.Key()), string(iter.Value()))
	})
}
