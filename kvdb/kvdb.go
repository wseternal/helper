package kvdb

import "context"

type KeyValueImpl interface {
	Put(set, key, val []byte) error
	Get(set, key []byte) (val []byte, err error)
	Delete(set, key[]byte) error
	Exist(set, key[]byte) bool

	// Seek return the iterator whose key greater than or equal to given key
	// iterator.Valid() is false if no matching key found
	Seek(set, key[]byte) Iterator
	First(set []byte) Iterator
	Last(set []byte) Iterator

	// ctx could be a cancellable context used to cancel the long time range function
	Range(ctx context.Context, set, start, end []byte, f func(iterator Iterator) )

	Close() error
}

type Iterator interface {
	Key() []byte
	Value() []byte
	Valid() bool
	Close() error
}

var (
	ErrKeyNotExisted = "given key is not existed"
	ErrSetNotExisted = "given set is not existed"
	DefaultSet = "default"
)
