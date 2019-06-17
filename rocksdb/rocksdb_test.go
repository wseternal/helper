package rocksdb

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/wseternal/gorocksdb"
)

var (
	cfOpts = CFOptions{
		"report": NewCFOptions(DefaultWriteBufferSize, DefaultBlockCacheSize, DefaultBloomFilterBit),
		"test":   NewCFOptions(DefaultWriteBufferSize, DefaultBlockCacheSize, DefaultBloomFilterBit),
	}
)

func initDB(dbpath string) (rdb *RDB, err error) {
	rdb, err = New(dbpath, nil, cfOpts, false)
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

func TestSecondary(t *testing.T) {
	db, err := New("./r.db", nil, cfOpts, false)
	if err != nil {
		t.Fatal(err)
	}
	db2, err := NewSecondary("./r.db", "r2.db", nil, cfOpts)
	if err != nil {
		t.Fatalf("open secondard rdb failed, %s\n", err)
	}

	key := strconv.FormatInt(time.Now().Unix(), 10)

	if err = db.PutCF(db.CFHs[DefaultColumnFamilyName], []byte(key), []byte(key)); err != nil {
		t.Fatalf("write db failed, %s\n", err)
	}
	fmt.Printf("set %s on db master\n", key)
	db.Flush()
	db2.TryCatchUpWithPrimary()
	var val []byte
	val, err = db2.GetCF(db.CFHs[DefaultColumnFamilyName], []byte(key))
	fmt.Printf("%v %v\n", val, err)
	db2.Close()
	db.Close()
}

func BenchmarkWrite(b *testing.B) {
	rdb, err := New("./r.db", nil, nil, false)
	if err != nil {
		b.Fatal(err)
	}
	data := []byte("test")
	b.ResetTimer()
	fmt.Printf("start benchmark\n")
	for i := 0; i < b.N; i++ {
		rdb.Put([]byte(DefaultColumnFamilyName), []byte(strconv.Itoa(i)), data)
	}
	rdb.Close()
}

func BenchmarkIter(b *testing.B) {
	rdb, err := New("./r.db", nil, nil, false)
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
	rdb, err := New("./r.db", nil, nil, false)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	fmt.Printf("start benchmark\n")
	for i := 0; i < b.N; i++ {
		rdb.WriteTo(rdb.CFHs[DefaultColumnFamilyName], []byte(strconv.Itoa(i)), os.Stdout)
	}
	rdb.Close()
}
