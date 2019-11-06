package boltdb

import (
	"fmt"
	"os"
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

	iter := db.First(nil)
	fmt.Printf("first: %s, %s\n", string(iter.Key()), string(iter.Value()))
}

func TestNextSequence(t *testing.T) {
	os.Remove("./test.db")
	db, err := NewBoltDB("./test.db", false)
	if err != nil {
		t.Fatal(err)
	}
	var seq uint64

	for i := 1; i <= 20; i++ {
		if seq, err = db.GetNextSequence(nil); err != nil {
			t.Fatal(err)
		}
		str := []byte(fmt.Sprintf("%04d", seq))
		db.Put(nil, []byte(str), []byte(str))
	}
	db.Range(nil, nil, db.First(nil).Value(), db.Last(nil).Value(), func(iter kvdb.Iterator) {
		fmt.Printf("%s => %s\n", string(iter.Key()), string(iter.Value()))
	})

	db.Range(nil, nil, nil, nil, func(iter kvdb.Iterator) {
		fmt.Printf("%s => %s\n", string(iter.Key()), string(iter.Value()))
	})

	stats, err := db.GetBucketStats(nil)
	fmt.Printf("stats: %+v %v\n", stats, err)
}
