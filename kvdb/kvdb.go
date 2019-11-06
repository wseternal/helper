package kvdb

import (
	"context"
	"strconv"
)

type KeyValueImpl interface {
	Put(set, key, val []byte) error
	Get(set, key []byte) (val []byte, err error)
	Delete(set, key []byte) error
	Exist(set, key []byte) bool

	// Seek return the iterator whose key greater than or equal to given key
	// iterator.Valid() is false if no matching key found
	Seek(set, key []byte) Iterator
	First(set []byte) Iterator
	Last(set []byte) Iterator

	// ctx could be a cancellable context used to cancel the long time range function
	// if start is nil, iterate from the first key in bucket
	// if end is nil, iterate to the end key in the bucket
	Range(ctx context.Context, set, start, end []byte, f func(iterator Iterator)) error

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
	DefaultSet       = []byte("default")
)

func PutInt64(impl KeyValueImpl, set, key []byte, val int64) error {
	return impl.Put(set, key, []byte(strconv.FormatInt(val, 10)))
}

func GetInt64(impl KeyValueImpl, set, key []byte) (int64, error) {
	data, err := impl.Get(set, key)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(string(data), 10, 64)
}
