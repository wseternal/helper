package boltdb

import (
	"errors"

	"github.com/boltdb/bolt"
)

type BoltDB struct {
	h    *bolt.DB
	Path string
}

var (
	ErrBucketNotExist = errors.New("bucket is not existed")
	DefaultBucket     = []byte("default")
)

func init() {
}

func NewBoltDB(fn string, readonly bool) (*BoltDB, error) {
	opt := *bolt.DefaultOptions
	opt.ReadOnly = readonly

	db, err := bolt.Open(fn, 0600, &opt)
	return &BoltDB{
		h:    db,
		Path: fn,
	}, err
}
