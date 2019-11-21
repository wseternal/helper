package rocksdb

import (
	"context"
	"fmt"
	"os"

	"github.com/wseternal/gorocksdb"
	"github.com/wseternal/helper/kvdb"
)

func (rdb *RDB) Put(set, key, val []byte) error {
	return rdb.PutCF(rdb.CFHs[string(set)], key, val)
}

func (rdb *RDB) Get(set, key []byte) (val []byte, err error) {
	return rdb.GetCF(rdb.CFHs[string(set)], key)
}

func (rdb *RDB) Delete(set, key []byte) error {
	return rdb.DeleteCF(rdb.CFHs[string(set)], key)
}

func (rdb *RDB) Exist(set, key []byte) bool {
	return rdb.KeyExist(rdb.CFHs[string(set)], key)
}

func (rdb *RDB) Seek(set, key []byte) kvdb.Iterator {
	it := rdb.NewRDBIterator(string(set))
	it.Iterator.Seek(key)
	return it
}

func (rdb *RDB) First(set []byte) kvdb.Iterator {
	it := rdb.NewRDBIterator(string(set))
	it.Iterator.SeekToFirst()
	return it
}

func (rdb *RDB) Last(set []byte) kvdb.Iterator {
	it := rdb.NewRDBIterator(string(set))
	it.Iterator.SeekToLast()
	return it
}

func (rdb *RDB) Range(ctx context.Context, set, start, end []byte, f func(iterator kvdb.Iterator)) error {
	opt := &RangeOption{
		StartKey: string(start),
		EndKey:   string(end),
		CF:       string(set),
	}
	opt.SetupCancelContext(ctx)

	return rdb.RangeForeach(opt, func(iter *Iterator) {
		f(NewRDBIteratorFrom(iter))
	})
}

func (rdb *RDB) Close() error {
	if rdb.DB != nil {
		rdb.DB.Close()
		if len(rdb.secondaryPath) > 0 {
			if err := gorocksdb.DestroyDb(rdb.secondaryPath, DefaultDBOption); err != nil {
				fmt.Fprintf(os.Stderr, "destroy secondary db %s failed, %s\n", rdb.secondaryPath, err)
			}
		}
		rdb.DB = nil
	}
	return nil
}
