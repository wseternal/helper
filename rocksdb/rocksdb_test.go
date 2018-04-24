package rocksdb

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/tecbot/gorocksdb"
)

var (
	cfOpts = CFOptions{
		"report": NewCFOptions(DefaultWriteBufferSize, DefaultBlockCacheSize, DefaultBloomFilterBit),
		"test":   NewCFOptions(DefaultWriteBufferSize, DefaultBlockCacheSize, DefaultBloomFilterBit),
	}
)

func initDB(dbpath string) (rdb *RDB, err error) {
	if !Exist(dbpath) {
		fmt.Printf("db %s is not existed, create it\n", dbpath)
		if err = Create(dbpath, nil, nil); err != nil {
			return nil, err
		}
	}
	rdb, err = Open(dbpath, nil, nil, false)
	if err != nil {
		return nil, err
	}
	fmt.Printf("open %s successfully: %v\n", dbpath, rdb)
	return rdb, nil
}

func TestCreateOpen(t *testing.T) {
	db, err := initDB("./r.db")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("estimate num keys: %s\n", db.GetProperty(KEstimateNumKeys))
	fmt.Printf("etstimate live data size: %s\n", db.GetProperty(KEstimateLiveDataSize))
	fmt.Printf("background errors: %s\n", db.GetProperty(KBackgroundErrors))
	fmt.Printf("stats: %s\n", db.GetProperty(KStats))
	fmt.Printf("level stats: %s\n", db.GetProperty(KLevelStats))
	fmt.Printf("sstable : %s\n", db.GetProperty(KSSTables))
	fmt.Printf("total sst file size : %s\n", db.GetProperty(KTotalSstFilesSize))
	fmt.Printf("aggregated table properties %s", db.GetProperty(KAggregatedTableProperties))

	iter := db.NewIteratorCF(gorocksdb.NewDefaultReadOptions(), db.CFHs[DefaultColumnFamilyName])
	num := 0
	iter.SeekToFirst()
	for iter.Valid() {
		num++
		iter.Next()
	}
	fmt.Printf("\nenumerate %d entries\n", num)
	db.Close()
}

func BenchmarkWrite(b *testing.B) {
	rdb, err := Open("./r.db", nil, nil, false)
	if err != nil {
		b.Fatal(err)
	}
	data := []byte("test")
	opt := gorocksdb.NewDefaultWriteOptions()
	b.ResetTimer()
	fmt.Printf("start benchmark\n")
	for i := 0; i < b.N; i++ {
		rdb.Put(opt, []byte(strconv.Itoa(i)), data)
	}
	rdb.Close()
}

func BenchmarkIter(b *testing.B) {
	rdb, err := Open("./r.db", nil, nil, false)
	if err != nil {
		b.Fatal(err)
	}
	opt := gorocksdb.NewDefaultReadOptions()
	idx := 0
	iter := rdb.NewIteratorCF(opt, rdb.CFHs[DefaultColumnFamilyName])
	iter.SeekToFirst()
	b.ResetTimer()
	fmt.Printf("start benchmark\n")
	for i := 0; i < b.N; i++ {
		idx++
		if iter.Valid() {
			iter.Next()
		}
	}
	fmt.Printf("enumerate %d keys\n", idx)
	iter.Close()
	rdb.Close()
}

func BenchmarkRead(b *testing.B) {
	rdb, err := Open("./r.db", nil, nil, false)
	if err != nil {
		b.Fatal(err)
	}
	opt := gorocksdb.NewDefaultReadOptions()
	b.ResetTimer()
	fmt.Printf("start benchmark\n")
	for i := 0; i < b.N; i++ {
		rdb.Get(opt, []byte(strconv.Itoa(i)))
	}
	rdb.Close()
}
