package boltdb

import (
	"errors"

	"github.com/boltdb/bolt"
)

type BoltDB struct {
	h *bolt.DB

	// corresponding option when opening the db
	Option *bolt.Options
}

var (
	ErrBucketNotExist = errors.New("bucket is not existed")
	DefaultBucket     = []byte("default")
)

func init() {
}

func New(fn string, readonly bool) (*BoltDB, error) {
	opt := NewOptions()
	opt.ReadOnly = readonly
	db, err := bolt.Open(fn, 0600, opt)
	return &BoltDB{
		h:      db,
		Option: opt,
	}, err
}

func NewOptions() *bolt.Options {
	opt := *bolt.DefaultOptions
	return &opt
}
