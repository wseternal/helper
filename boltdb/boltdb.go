package boltdb

import (
	"github.com/boltdb/bolt"
	"github.com/wseternal/helper/kvdb"
)

type BoltDB struct {
	h    *bolt.DB
	Path string
}

var (
	DefaultBucket = kvdb.DefaultSet
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
func (db *BoltDB) GetNextSequence(bucket []byte) (uint64, error) {
	if bucket == nil {
		bucket = DefaultBucket
	}
	tx, err := db.h.Begin(true)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var b *bolt.Bucket
	if b, err = tx.CreateBucketIfNotExists(bucket); err != nil {
		return 0, err
	}
	var seq uint64
	seq, err = b.NextSequence()
	if err != nil {
		return 0, err
	}
	return seq, tx.Commit()
}

func (db *BoltDB) GetBucketStats(bucket []byte) (*bolt.BucketStats, error) {
	if bucket == nil {
		bucket = DefaultBucket
	}
	tx, err := db.h.Begin(false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var b *bolt.Bucket
	if b = tx.Bucket(bucket); b == nil {
		return nil, bolt.ErrBucketNotFound
	}
	stats := b.Stats()
	return &stats, nil
}
