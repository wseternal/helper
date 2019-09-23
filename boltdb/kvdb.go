package boltdb

import (
	"bytes"
	"context"

	"github.com/boltdb/bolt"
	"github.com/wseternal/helper/kvdb"
)

const (
	cursorFuncFirst = "first"
	cursorFuncLast  = "last"
	cursorFuncSeek  = "seek"
)

type BoltDBIterator struct {
	key, value []byte
	valid      bool
}

func (it *BoltDBIterator) Key() []byte {
	return it.key
}

func (it *BoltDBIterator) Value() []byte {
	return it.value
}

func (it *BoltDBIterator) Valid() bool {
	return it.valid
}

func (it *BoltDBIterator) Close() error {
	it.valid = false
	it.key = nil
	it.value = nil
	return nil
}

func (db *BoltDB) Close() error {
	return db.h.Close()
}

func (db *BoltDB) Get(bucket, key []byte) ([]byte, error) {
	var res []byte
	var err error
	if bucket == nil {
		bucket = DefaultBucket
	}
	if err = db.h.View(func(t *bolt.Tx) error {
		if b := t.Bucket(bucket); b != nil {
			res = append(res, b.Get(key)...)
			return nil
		}
		return ErrBucketNotExist
	}); err != nil {
		return nil, err
	}
	return res, nil
}

func (db *BoltDB) Put(bucket, key, value []byte) error {
	if bucket == nil {
		bucket = DefaultBucket
	}
	return db.h.Update(func(t *bolt.Tx) error {
		b, err := t.CreateBucketIfNotExists(bucket)
		if err != nil {
			return err
		}
		return b.Put(key, value)
	})
}

func (db *BoltDB) Delete(bucket, key []byte) error {
	if bucket == nil {
		bucket = DefaultBucket
	}
	return db.h.Update(func(t *bolt.Tx) error {
		if b := t.Bucket(bucket); b != nil {
			return b.Delete(key)
		}
		return ErrBucketNotExist
	})
}

func (db *BoltDB) Exist(bucket, key []byte) bool {
	if bucket == nil {
		bucket = DefaultBucket
	}
	exist := false
	_ = db.h.View(func(t *bolt.Tx) error {
		if b := t.Bucket(bucket); b != nil {
			if b.Get(key) != nil {
				exist = true
			}
		}
		return nil
	})
	return exist
}

func (db *BoltDB) cursorFunc(bucket []byte, name string, key []byte) kvdb.Iterator {
	if bucket == nil {
		bucket = DefaultBucket
	}
	it := &BoltDBIterator{}
	_ = db.h.View(func(t *bolt.Tx) error {
		if b := t.Bucket(bucket); b != nil {
			c := b.Cursor()
			var k, v []byte
			switch name {
			case cursorFuncFirst:
				k, v = c.First()
			case cursorFuncLast:
				k, v = c.Last()
			case cursorFuncSeek:
				k, v = c.Seek(key)
			}
			if k == nil {
				return nil
			}
			// deep copy the key value
			it.key = append(it.key, k...)
			it.value = append(it.value, v...)
			it.valid = true
		}
		return nil
	})
	return it
}

func (db *BoltDB) Seek(bucket, key []byte) kvdb.Iterator {
	return db.cursorFunc(bucket, cursorFuncSeek, key)
}

func (db *BoltDB) First(bucket []byte) kvdb.Iterator {
	return db.cursorFunc(bucket, cursorFuncFirst, nil)
}

func (db *BoltDB) Last(bucket []byte) kvdb.Iterator {
	return db.cursorFunc(bucket, cursorFuncLast, nil)
}

func (db *BoltDB) Range(ctx context.Context, bucket, start, end []byte, f func(iterator kvdb.Iterator)) error {
	if bucket == nil {
		bucket = DefaultBucket
	}
	return db.h.View(func(t *bolt.Tx) error {
		b := t.Bucket(bucket)
		if b == nil {
			return nil
		}
		c := b.Cursor()
		for k, v := c.Seek(start); k != nil && bytes.Compare(k, end) <= 0; k, v = c.Next() {
			if ctx != nil && ctx.Err() != nil {
				break
			}
			iter := &BoltDBIterator{
				key:   k,
				value: v,
				valid: true,
			}
			f(iter)
		}
		return nil
	})
}
