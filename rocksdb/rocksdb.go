package rocksdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/wseternal/gorocksdb"
)

type BackupInfo struct {
	NumFiles  int32
	Size      int64
	Timestamp int64
	ID        int64
}

type CFInfo struct {
	Name              string
	EstimateNumKeys   string
	TotalSstFilesSize string
	LevelStats        string `json:",omitempty"`
}

type RDB struct {
	*gorocksdb.DB
	CFHs      CFHandles
	WriteOpts *gorocksdb.WriteOptions
	ReadOpts  *gorocksdb.ReadOptions

	secondaryPath string
}

type CFOptions map[string]*gorocksdb.Options
type CFHandles map[string]*gorocksdb.ColumnFamilyHandle
type WriteBatches map[string]*gorocksdb.WriteBatch

type Property string
type PropertyPrefix string

type KVJson struct {
	Key   string
	Value json.RawMessage
}

type KVRaw struct {
	Key, Value string
}

type RangeOption struct {
	// all entries with the the key falls in range [StartKey, EndKey]
	StartKey, EndKey string

	// all entries with the ts field falls in range [startTS, endTS]
	StartTS, EndTS int64
	// the ts field index in the key after splitted
	TSFieldIndex int
	// separator used to split the key
	KeySeparator string

	// if Key existed, get entry with given Key
	Key string

	// column family
	CF string

	// limit returned entry count
	Limit int64

	// IsJsonValue: if true, use json.RawMessage for value when encoded key/value to the output
	IsJsonValue bool

	// range option could be terminated by cancel the context
	Ctx    context.Context
	Cancel context.CancelFunc

	// output each object per-line, it's set when output parameter is specified
	streamOutput bool
}

type RDBIterator struct {
	*gorocksdb.Iterator
}

type Iterator = gorocksdb.Iterator

type HijackTsInKey func(ts string, key []byte) (newKey []byte, nextKey []byte)

// in RangeFunc, key/value data of iterator shall be deep copied if you want to store them.
type RangeFunc func(iter *gorocksdb.Iterator)

const (
	rocksdbPrefix = "rocksdb."
)

var (
	DefaultWriteOption   = gorocksdb.NewDefaultWriteOptions()
	DefaultReadOption    = gorocksdb.NewDefaultReadOptions()
	DefaultDBOption      = NewDBOptions()
	DefaultRestoreOption = gorocksdb.NewRestoreOptions()
	DefaultFlushOption   = gorocksdb.NewDefaultFlushOptions()

	// please refer to https://github.com/facebook/rocksdb/wiki/Rate-Limiter
	DefaultRateLimiter = gorocksdb.NewRateLimiter(10<<20, 100000, 10)
)

const (
	DefaultBloomFilterBit = 10

	HugeWriteBufferSize = DefaultWriteBufferSize << 2
	HugeBlockCacheSize  = DefaultBlockCacheSize << 2

	DefaultWriteBufferSize = 32 << 20
	DefaultBlockCacheSize  = 64 << 20

	TinyWriteBufferSize = DefaultWriteBufferSize >> 2
	TinyBlockCacheSize  = DefaultBlockCacheSize >> 2

	DefaultColumnFamilyName = "default"

	// please refer to struct Properties in include/rocksdb/db.H
	KStats                     Property = "stats" // kCFStats plus KDBStats
	KSSTables                  Property = "sstables"
	KLevelStats                Property = "levelstats"
	KEstimateNumKeys           Property = "estimate-num-keys"
	KBackgroundErrors          Property = "background-errors"
	KEstimateLiveDataSize      Property = "estimate-live-data-size"
	KNumSnapshots              Property = "num-snapshots"
	KOldestSnapshotTime        Property = "oldest-snapshot-time"
	KNumLiveVersions           Property = "num-live-versions"
	KCurrentSuperVersionNumber Property = "current-super-version-number"
	KTotalSstFilesSize         Property = "total-sst-files-size"
	KAggregatedTableProperties Property = "aggregated-table-properties"

	KNumFilesAtLevelPrefix            PropertyPrefix = "num-files-at-level"
	KCompressionRatioAtLevelPrefix    PropertyPrefix = "compression-ratio-at-level"
	KAggregatedTablePropertiesAtLevel PropertyPrefix = "aggregated-table-properties-at-level"
)

func init() {
	// disable Write ahead log by default, it's strangely that lots of
	// small wal log files (allocated with much more storage) are left on the system.
	DefaultWriteOption.DisableWAL(true)
	// set sync to true, if external tool such as ldb need be used to read the data in realtime
	DefaultWriteOption.SetSync(false)
	DefaultFlushOption.SetWait(true)
}

func NewRangeOption() *RangeOption {
	opt := &RangeOption{
		CF:           DefaultColumnFamilyName,
		Limit:        -1,
		TSFieldIndex: 1,
		KeySeparator: ",",
	}
	return opt
}

func (opt *RangeOption) SetupCancelContext(ctx context.Context) {
	// cancel the previous context if any
	if opt.Cancel != nil {
		opt.Cancel()

	}
	if ctx == nil {
		ctx = context.Background()
	}
	opt.Ctx, opt.Cancel = context.WithCancel(ctx)
}

func (opt *RangeOption) Done() {
	if opt.Cancel != nil {
		opt.Cancel()
	}
}

func GenHijackTsInKeyByIndex(idx int, sep []byte) HijackTsInKey {
	return func(ts string, key []byte) ([]byte, []byte) {
		fields := bytes.Split(key, sep)
		if len(fields) == 1 {
			return key, nil
		}
		if len(fields) < (idx + 1) {
			return key, nil
		}
		// do not override the original key
		replaced := make([]byte, len(fields[idx]))
		var nextKey []byte
		if idx >= 1 {
			nextKey = bytes.Join(fields[0:idx], sep)
			for i := len(nextKey) - 1; i >= 0; i-- {
				if nextKey[i] != 0xff {
					nextKey[i] = nextKey[i] + 1
					break
				}
			}
		}
		copy(replaced, ts)
		fields[idx] = replaced
		return bytes.Join(fields, sep), nextKey
	}
}

func (cfOpts CFOptions) GetKVPair() ([]string, []*gorocksdb.Options) {
	optLen := len(cfOpts)
	keys := make([]string, optLen)
	values := make([]*gorocksdb.Options, optLen)
	idx := 0
	for k, v := range cfOpts {
		keys[idx] = k
		values[idx] = v
		idx++
	}
	return keys, values
}

func (cfOpts CFOptions) AddDefaultCF() {
	if _, found := cfOpts[DefaultColumnFamilyName]; !found {
		cfOpts[DefaultColumnFamilyName] = NewCFOptions(TinyWriteBufferSize, TinyBlockCacheSize, DefaultBloomFilterBit)
	}
}

func Exist(fn string) bool {
	opts := gorocksdb.NewDefaultOptions()
	tmp, err := gorocksdb.OpenDbForReadOnly(opts, fn, false)
	if err == nil {
		tmp.Close()
		return true
	}
	return false
}

func setDefault(opts *gorocksdb.Options) {
	opts.SetKeepLogFileNum(1)
	opts.SetInfoLogLevel(gorocksdb.WarnInfoLogLevel)
	opts.SetMaxLogFileSize(128 << 20)
	opts.SetMaxTotalWalSize(128 << 20)
	opts.SetWALTtlSeconds(60)
	opts.SetWalSizeLimitMb(8)
	opts.SetRateLimiter(DefaultRateLimiter)
}

func NewCFOptions(writeBufferSize int, blockCacheSize int, bloomFilterBit int) *gorocksdb.Options {
	// OptimizeForSmallDb func, Use this if your DB is very small (like under 1GB)
	// OptimizeForPointLookup func,  don't need to keep the data sorted, i.e., you'll never use an iterator.
	// OptimizeLevelStyleCompaction func
	// OptimizeUniversalStyleCompaction func
	// comparator Comparator used to define the order of keys in the table, default, byte-wise ordering
	// merge_operator if Merge operation needs to be accessed.
	// compaction_filter A single CompactionFilter instance to call into during compaction. Allows an application to modify/delete a key-value during background compaction.
	// compaction_filter_factory  a factory that provides compaction filter objects
	// write_buffer_size Amount of data to build up in memory (backed by an unsorted log on disk) before converting to a sorted on-disk file.
	// 		default 64MB, Up to max_write_buffer_number write buffers may be held in memory at the same time.
	// compression (CompressionType) default kSnappyCompression
	// bottommost_compression (CompressionType) Compression algorithm that will be used for the bottommost level, default kDisableCompressionOption
	// compression_opts different options for compression algorithms
	// level0_file_num_compaction_trigger default 4, Number of files to trigger level-0 compaction
	// prefix_extractor If non-nullptr, use the specified function to determine the prefixes for keys. These prefixes will be placed in the filter.
	// 		1) key.starts_with(prefix(key))
	// 		2) Compare(prefix(key), key) <= 0.
	// 		3) If Compare(k1, k2) <= 0, then Compare(prefix(k1), prefix(k2)) <= 0
	// 		4) prefix(prefix(key)) == prefix(key)
	// max_bytes_for_level_base the max total for level-1. default 256MB
	// max_bytes_for_level_multiplier if level-1 is 200MB, and multiplier is 10, level-2 will be 2GB
	// disable_auto_compactions default false, Disable automatic compactions, Manual compactions can still be issued on this column family

	// BlockBasedTableOptions
	// cache_index_and_filter_blocks default false, Indicating if we'd put index/filter blocks to the block cache.
	// cache_index_and_filter_blocks_with_high_priority default false, if true, index and filter blocks may be less likely to be evicted than data blocks.
	// pin_l0_filter_and_index_blocks_in_cache default false, if true, a reference to the filter and index blocks are held in the "table reader" object,
	// 		so the blocks are pinned and only evicted from cache when table reader is freed.
	// index_type default kBinarySearch
	// checksum default kCRC32c
	// no_block_cache default false, disable block cache if set to true.
	// block_cache default 8MB internal cache
	// block_size default 4K
	// block_size_deviation default 10, If the percentage of free space in the current block is less than this specified number and adding a new record to the block will
	// 		exceed the configured block size, then this block will be closed and the new record will be written to the next block.
	// block_restart_interval default 16, Number of keys between restart points for delta encoding of keys.
	// index_block_restart_interval default 1, Same as block_restart_interval but used for the index block.
	// use_delta_encoding default true, Use delta encoding to compress keys in blocks. ReadOptions::pin_data requires this option to be disabled.
	// filter_policy If non-nullptr, use the specified filter policy to reduce disk reads. Many applications will benefit from passing the result of NewBloomFilterPolicy() here.
	// whole_key_filtering  default true, If true, place whole keys in the filter (not just prefixes).
	// verify_compression default false, Verify that decompressing the compressed block gives back the input.
	// it's a verification mode that we use to detect bugs in compression algorithms.
	// read_amp_bytes_per_bit default 0 (disabled), This number must be a power of 2
	opts := gorocksdb.NewDefaultOptions()
	setDefault(opts)

	opts.OptimizeLevelStyleCompaction(uint64(writeBufferSize << 2))

	bbto := gorocksdb.NewDefaultBlockBasedTableOptions()
	if bloomFilterBit > 0 {
		filter := gorocksdb.NewBloomFilter(bloomFilterBit)
		bbto.SetFilterPolicy(filter)
	}
	if blockCacheSize <= 0 {
		blockCacheSize = DefaultBlockCacheSize
	}
	bbto.SetBlockCache(gorocksdb.NewLRUCache(uint64(blockCacheSize)))

	if writeBufferSize <= 0 {
		writeBufferSize = DefaultWriteBufferSize
	}
	// use 64K block size
	bbto.SetBlockSize(64 << 10)

	opts.SetWriteBufferSize(writeBufferSize)
	opts.SetBlockBasedTableFactory(bbto)

	opts.SetCompactionStyle(gorocksdb.LevelCompactionStyle)
	// use maximum 1024M for L1
	opts.SetMaxBytesForLevelBase(1024 << 20)
	opts.SetMaxBytesForLevelMultiplier(10)
	// use 512M for L1
	opts.SetTargetFileSizeBase(512 << 20)
	opts.SetTargetFileSizeMultiplier(10)

	return opts
}

func NewDBOptions() *gorocksdb.Options {
	// https://github.com/facebook/rocksdb/blob/master/include/rocksdb/options.h
	// create_if_missing
	// create_missing_column_families
	// error_if_exists
	// paranoid_checks, aggressively check consistency of the data. switch to ro mode when writing error.
	// Env Use the specified object to interact with the environment, e.g.: to read/write files, schedule background work
	// rate_limiter control write rate of flush and compaction
	// sst_file_manager (track SST files and control their file deletion rate.)
	// info_log
	// info_log_level
	// max_open_files default -1 (files opened are always kept open)
	// max_file_opening_threads
	// max_total_wal_size, if 0, [sum of all write_buffer_size * max_write_buffer_number] * 4
	// statistics collect metrics about database operations
	// use_fsync default false (use fdatasync), if true, use fsync
	// db_paths A list of paths where SST files can be put into, with its target size, e.g.: [{"/flash_path", 10GB}, {"/hard_drive", 2TB}]
	// db_log_dir
	// wal_dir
	// delete_obsolete_files_period_micros
	// max_background_jobs (compactions and flushes)
	// for compatible, max_background_jobs = max_background_compactions + max_background_flushes
	// max_log_file_size default 0 (all logs write to the same file)
	// log_file_time_to_roll default 0 (disiabled)
	// keep_log_file_num default 1000
	// recycle_log_file_num If non-zero, we will reuse previously written log files for new
	// 		logs, overwriting the old data
	// max_manifest_file_size: default MAX_INT, so roll-over does not take place.
	// table_cache_numshardbits default 6
	// table_cache_remove_scan_count_limit not supported any more
	// WAL_ttl_seconds default 0
	// WAL_size_limit_MB default 0
	// 		If both set to 0, logs will be deleted asap and will not get into
	// manifest_preallocation_size default 4MB
	// allow_mmap_reads default false, Allow the OS to mmap file for reading sst tables.
	// allow_mmap_writes default false, Allow the OS to mmap file for writing., DB::SyncWAL() only works if this is set to false.
	// use_direct_reads default false, Files will be opened in "direct I/O" mode which means that data r/w from the disk will not be cached or buffered
	// use_direct_io_for_flush_and_compaction default false, Use O_DIRECT for both reads and writes in background flush and compactions
	// allow_fallocate, default true
	// is_fd_close_on_exec, default true, Disable child process inherit open files.
	// skip_log_error_on_recovery: NOT SUPPORTED ANYMORE -- this options is no longer used
	// stats_dump_period_sec default 600, dump rocksdb.stats to LOG every stats_dump_period_sec
	// advise_random_on_open default true, hint the file access pattern of underlying file system is random
	// db_write_buffer_size: Amount of data to build up in memtables across all column families before writing to disk.
	// 		This is distinct from write_buffer_size, which enforces a limit for a single memtable. default is disabled.
	// write_buffer_manager default disabled, The memory usage of memtable will report to this object
	// access_hint_on_compaction_start Specify the file access pattern once a compaction is started
	// 		It will be applied to all input files of a compaction.
	// new_table_reader_for_compaction_inputs default false, If true, always create a new file descriptor and new table reader for compaction inputs.
	// compaction_readahead_size we perform bigger reads when doing compaction. set to at least 2MB for spinning disks
	// random_access_max_buffer_size maximum buffer size that is used by WinMmapReadableFile in unbuffered disk I/O mode.
	// 		honored only on windows, default 1MB
	// writable_file_max_buffer_size maximum buffer size that is used by WritableFileWriter. default 1MB
	// use_adaptive_mutex default false, which spins in the user space before resorting to kernel, This could reduce context switch when the mutex is not
	// 		heavily contended, However, if the mutex is hot, we could end up wasting spin time
	// bytes_per_sync Allows OS to incrementally sync files to disk while they are being written, asynchronously, in the background.
	// 		When rate limiter is enabled, it automatically enables bytes_per_sync to 1MB, default 0
	// wal_bytes_per_sync Same as bytes_per_sync, but applies to WAL files
	// listeners A vector of EventListeners which call-back functions will be called when specific RocksDB event happens.
	// enable_thread_tracking default false, status of the threads involved in this DB will be available through GetThreadList() API
	// delayed_write_rate default 0, if 0, infer a value from `rater_limiter` value if it's non empty, or 16MB if its' empty
	// enable_pipelined_write default false, if true, true, separate write thread queue is maintained for WAL write and memtable write.
	// 		Enabling the feature may improve rite throughput and reduce latency of the prepare phase of two-phase commit.
	// allow_concurrent_memtable_write If true, allow multi-writers to update mem tables in parallel, default true.
	// 		Concurrent memtable writes are not compatible with inplace_update_support or filter_deletes.
	// 		It is strongly recommended to set enable_write_thread_adaptive_yield if you are going to use this feature.
	// enable_write_thread_adaptive_yield default true, If true, threads synchronizing with the write batch group leader will
	// 		wait for up to write_thread_max_yield_usec before blocking on a mutex.
	// write_thread_max_yield_usec The maximum number of microseconds that a write operation will use a yielding spin loop to coordinate with other write threads before
	// 		blocking on a mutex. default 100. increasing this value is likely to increase RocksDB throughput at the expense of increased CPU usage.
	// write_thread_slow_yield_usec The latency in microseconds after which a std::this_thread::yield call (sched_yield on Linux) is considered to be a signal that
	// 		other processes or threads would like to use the current core.Increasing this makes writer threads more likely to take CPU by spinning,
	//  	which will show up as an increase in the number of involuntary context switches.
	// skip_stats_update_on_db_open default false,  If true, then DB::Open() will not update the statistics
	// wal_recovery_mode default kPointInTimeRecovery
	// allow_2pc default false, if set to false then recovery will fail when a prepared transaction is encountered in the WAL
	// row_cache A global cache for table-level rows, default nullptr (disabled)
	// wal_filter A filter object supplied to be invoked while processing WAL during recovery. default nullptr
	// fail_if_options_file_error default false, If true, then DB::Open / CreateColumnFamily / DropColumnFamily / SetOptions will fail if options file is not detected or properly persisted.
	// dump_malloc_stats default false, print malloc stats together with rockdsdb.stats when printing to LOG
	// avoid_flush_during_recovery default false, By default RocksDB replay WAL logs and flush them on DB open
	// avoid_flush_during_shutdown By default RocksDB will flush all memtables on DB close if there are unpersisted data (i.e. with WAL disabled) The flush can be skip to speedup
	// 		DB close. Unpersisted data WILL BE LOST. default false. Dynamically changeable through SetDBOptions() API.
	// allow_ingest_behind default false, Set this option to true during creation of database if you want to be able to ingest behind.
	// concurrent_prepare default false, If enabled it uses two queues for writes
	// manual_wal_flush: default false, If true WAL is not flushed automatically after each write. Instead it relies on manual invocation of FlushWAL to write the WAL buffer to its file.
	// TODO add dump_malloc_stats

	opts := gorocksdb.NewDefaultOptions()
	setDefault(opts)

	opts.SetCreateIfMissing(true)
	opts.SetCreateIfMissingColumnFamilies(true)

	return opts
}

func NewSecondary(master, secondary string, opts *gorocksdb.Options, cfOpts CFOptions) (rdb *RDB, err error) {
	if opts == nil {
		opts = NewDBOptions()
	}
	if cfOpts == nil {
		cfOpts = make(CFOptions, 1)
	}
	cfOpts.AddDefaultCF()

	var cfsDB []string
	if Exist(master) {
		if cfsDB, err = gorocksdb.ListColumnFamilies(opts, master); err != nil {
			return nil, err
		}
	}
	for _, v := range cfsDB {
		if _, found := cfOpts[v]; !found {
			cfOpts[v] = NewCFOptions(DefaultWriteBufferSize, DefaultBlockCacheSize, DefaultBloomFilterBit)
		}
	}

	keys, values := cfOpts.GetKVPair()

	// duplicate
	readOpts := *DefaultReadOption
	writeOpts := *DefaultWriteOption

	rdb = &RDB{
		ReadOpts:      &readOpts,
		WriteOpts:     &writeOpts,
		secondaryPath: secondary,
	}

	defer func() {
		if err != nil && rdb.DB != nil {
			rdb.Close()
			rdb.DB = nil
		}
	}()

	var handles []*gorocksdb.ColumnFamilyHandle

	if rdb.DB, handles, err = gorocksdb.OpenDbAsSecondaryColumnFamilies(opts, master, secondary, keys, values); err != nil {
		return rdb, err
	}
	rdb.CFHs = make(CFHandles, len(keys))
	for i, v := range keys {
		rdb.CFHs[v] = handles[i]
	}
	return rdb, nil
}

// New if fn does not exist when opening, CreateIfMissing could be set to avoid opening error
func New(fn string, opts *gorocksdb.Options, cfOpts CFOptions, readonly bool) (rdb *RDB, err error) {
	if opts == nil {
		opts = NewDBOptions()
	}
	if cfOpts == nil {
		cfOpts = make(CFOptions, 1)
	}
	cfOpts.AddDefaultCF()

	var cfsDB []string
	if Exist(fn) {
		if cfsDB, err = gorocksdb.ListColumnFamilies(opts, fn); err != nil {
			return nil, err
		}
	}
	for _, v := range cfsDB {
		if _, found := cfOpts[v]; !found {
			cfOpts[v] = NewCFOptions(DefaultWriteBufferSize, DefaultBlockCacheSize, DefaultBloomFilterBit)
		}
	}

	keys, values := cfOpts.GetKVPair()

	// duplicate
	readOpts := *DefaultReadOption
	writeOpts := *DefaultWriteOption

	rdb = &RDB{
		ReadOpts:  &readOpts,
		WriteOpts: &writeOpts,
	}

	defer func() {
		if err != nil && rdb.DB != nil {
			rdb.Close()
			rdb.DB = nil
		}
	}()

	var handles []*gorocksdb.ColumnFamilyHandle
	if readonly {
		if rdb.DB, handles, err = gorocksdb.OpenDbForReadOnlyColumnFamilies(opts, fn, keys, values, false); err != nil {
			return rdb, err
		}
	} else {
		if rdb.DB, handles, err = gorocksdb.OpenDbColumnFamilies(opts, fn, keys, values); err != nil {
			return rdb, err
		}
	}
	rdb.CFHs = make(CFHandles, len(keys))
	for i, v := range keys {
		rdb.CFHs[v] = handles[i]
	}
	return rdb, nil
}

func (rdb *RDB) GetProperty(name Property) string {
	return rdb.DB.GetProperty(fmt.Sprintf("%s%s", rocksdbPrefix, name))
}

func (rdb *RDB) GetPropertyCF(name Property, cf *gorocksdb.ColumnFamilyHandle) string {
	return rdb.DB.GetPropertyCF(fmt.Sprintf("%s%s", rocksdbPrefix, name), cf)
}

func (rdb *RDB) GetPropertyWithPrefix(prefix int, name PropertyPrefix) string {
	return rdb.DB.GetProperty(fmt.Sprintf("%s%s%d", rocksdbPrefix, name, prefix))
}

func (rdb *RDB) Flush() error {
	return rdb.DB.Flush(DefaultFlushOption)
}

func (rdb *RDB) CompactCF(cf *gorocksdb.ColumnFamilyHandle) {
	rdb.DB.CompactRangeCF(cf, gorocksdb.Range{})
}

func (rdb *RDB) Backup(fn string) error {
	engine, err := GetBackupEngine(fn)
	if err != nil {
		return fmt.Errorf("BackupDB: open backupengine for %s failed, error: %s", fn, err)
	}
	defer engine.Close()

	return engine.CreateNewBackup(rdb.DB)
}

func PurgeOldBackups(fn string, numBackupsToKeep uint32) error {
	engine, err := GetBackupEngine(fn)
	if err != nil {
		return fmt.Errorf("PurgeOldBackups: open backupengine for %s failed, error: %s", fn, err)
	}
	defer engine.Close()

	return engine.PurgeOldBackups(numBackupsToKeep)
}

func GetBackupEngine(fn string) (*gorocksdb.BackupEngine, error) {
	engine, err := gorocksdb.OpenBackupEngine(DefaultDBOption, fn)
	if err != nil {
		return nil, fmt.Errorf("OpenBackupEngine fail: %s", err)
	}
	return engine, nil
}

func Restore(backupPath, restorePath string) error {
	engine, err := GetBackupEngine(backupPath)
	if err != nil {
		return err
	}
	defer engine.Close()

	return engine.RestoreDBFromLatestBackup(restorePath, restorePath, DefaultRestoreOption)
}

func GetBackupInfo(fn string) ([]*BackupInfo, error) {
	engine, err := GetBackupEngine(fn)
	if err != nil {
		return nil, err
	}
	defer engine.Close()

	b := engine.GetInfo()
	defer b.Destroy()

	count := b.GetCount()
	if count == 0 {
		return nil, fmt.Errorf("backup count is 0 for %s", fn)
	}
	res := make([]*BackupInfo, count)
	for i := 0; i < count; i++ {
		res[i] = &BackupInfo{
			ID:        b.GetBackupId(i),
			Timestamp: b.GetTimestamp(i),
			NumFiles:  b.GetNumFiles(i),
			Size:      b.GetSize(i),
		}
	}
	return res, nil
}

func (rdb *RDB) KeyExist(cf *gorocksdb.ColumnFamilyHandle, key []byte) bool {
	iter := rdb.NewIteratorCF(DefaultReadOption, cf)
	defer iter.Close()
	iter.Seek(key)

	// as Slice, map, and function values are not comparable, convert byte slice to string
	if iter.Valid() && string(iter.Key().Data()) == string(key) {
		return true
	}
	return false
}

// SeekForPrevKey return the key less than or equal to given key
// return nil if no matching key found
func (rdb *RDB) SeekForPrevKey(cf *gorocksdb.ColumnFamilyHandle, key []byte) []byte {
	iter := rdb.NewIteratorCF(DefaultReadOption, cf)
	defer iter.Close()
	iter.SeekForPrev(key)
	if iter.Valid() {
		return append([]byte(nil), iter.Key().Data()...)
	}
	return nil
}

// SeekKeyUpperBound return the first key >= key and the last key <= upper bound
func (rdb *RDB) SeekKeyUpperBound(cf *gorocksdb.ColumnFamilyHandle, key, upperBound []byte) (first, last []byte) {
	opt := *DefaultReadOption
	opt.SetIterateUpperBound(upperBound)

	iter := rdb.NewIteratorCF(&opt, cf)
	defer iter.Close()

	// find the first key  >= key
	iter.Seek(key)
	if !iter.Valid() {
		return nil, nil
	}
	first = append([]byte(nil), iter.Key().Data()...)

	// find the last key <= upper bound
	iter.SeekForPrev(upperBound)
	last = append([]byte(nil), iter.Key().Data()...)

	return first, last
}

func (rdb *RDB) NewWriteBatch() *gorocksdb.WriteBatch {
	return gorocksdb.NewWriteBatch()
}

func (rdb *RDB) DeleteCF(cf *gorocksdb.ColumnFamilyHandle, key []byte) error {
	return rdb.DB.DeleteCF(rdb.WriteOpts, cf, key)
}

func (rdb *RDB) PutCF(cf *gorocksdb.ColumnFamilyHandle, key, value []byte) error {
	return rdb.DB.PutCF(rdb.WriteOpts, cf, key, value)
}

func (rdb *RDB) GetCF(cf *gorocksdb.ColumnFamilyHandle, key []byte) ([]byte, error) {
	data, err := rdb.DB.GetCF(rdb.ReadOpts, cf, key)
	if err != nil {
		return nil, err
	}
	if data.Data() == nil {
		return nil, fmt.Errorf("no value found for key: %s", string(key))
	}
	defer data.Free()

	return append([]byte(nil), data.Data()...), nil
}

func (rdb *RDB) WriteTo(cf *gorocksdb.ColumnFamilyHandle, key []byte, w io.Writer) error {
	iter := rdb.NewIteratorCF(DefaultReadOption, cf)
	defer iter.Close()
	iter.Seek(key)
	if !(iter.Valid() && string(iter.Key().Data()) == string(key)) {
		return fmt.Errorf("can not find key: %s", string(key))
	}

	kv := NewKVRaw(iter)
	var data []byte
	data, _ = json.Marshal(&kv)
	_, _ = w.Write(data)

	return nil
}

func NewKVJson(iter *gorocksdb.Iterator) *KVJson {
	return &KVJson{
		Key:   string(iter.Key().Data()),
		Value: iter.Value().Data(),
	}
}

func NewKVRaw(iter *gorocksdb.Iterator) *KVRaw {
	return &KVRaw{
		Key:   string(iter.Key().Data()),
		Value: string(iter.Value().Data()),
	}
}

// RangeForeach enumerate all keys falls in range [startKey, endKey],
func (rdb *RDB) RangeForeach(opt *RangeOption, oper RangeFunc) error {
	if oper == nil {
		return fmt.Errorf("oper parameter is nil")
	}
	cf := rdb.CFHs[opt.CF]
	iter := rdb.NewIteratorCF(DefaultReadOption, cf)
	defer iter.Close()

	defer opt.Done()

	if len(opt.StartKey) > 0 {
		iter.Seek([]byte(opt.StartKey))
	} else {
		iter.SeekToFirst()
	}

	var cnt int64
	for cnt = 0; iter.Valid() && (len(opt.EndKey) == 0 || string(iter.Key().Data()) <= opt.EndKey); iter.Next() {
		cnt++
		oper(iter)
		if opt.Limit > 0 && cnt >= opt.Limit {
			break
		}
		if opt.Ctx != nil && opt.Ctx.Err() != nil {
			return opt.Ctx.Err()
		}
	}
	return nil
}

// RangeForeachByTS enumerate all entries with the ts field falls in range [startTS, endTS]
func (rdb *RDB) RangeForeachByTS(opt *RangeOption, f HijackTsInKey, oper RangeFunc) error {
	var err error
	var cnt int64 = 0
	var cntNextKeys int64 = 0
	cf := rdb.CFHs[opt.CF]
	iter := rdb.NewIteratorCF(DefaultReadOption, cf)
	defer iter.Close()

	defer opt.Done()

	if oper == nil || f == nil {
		err = fmt.Errorf("%s\n", "RangeForeachByTS: both f and oper shall not be nil")
		goto out
	}

	iter.SeekToFirst()
	if !iter.Valid() {
		goto out
	}
	for {
		tsStart := strconv.FormatInt(opt.StartTS, 10)
		tsEnd := strconv.FormatInt(opt.EndTS, 10)
		keyStart, nextKey := f(tsStart, iter.Key().Data())
		if nextKey == nil {
			break
		}
		cntNextKeys++
		keyEnd, _ := f(tsEnd, iter.Key().Data())

		iter.Seek(keyStart)
		for ; iter.Valid() && (len(keyEnd) == 0 || string(iter.Key().Data()) <= string(keyEnd)); iter.Next() {
			oper(iter)
			cnt++
			if opt.Limit > 0 && cnt >= opt.Limit {
				goto out
			}
			if opt.Ctx != nil && opt.Ctx.Err() != nil {
				err = opt.Ctx.Err()
				goto out
			}
		}
		iter.Seek(nextKey)
		if !iter.Valid() {
			break
		}
	}
out:
	return err
}

func (rdb *RDB) GetRangeByKey(opt *RangeOption, w io.Writer) error {
	var err error
	enc := json.NewEncoder(w)
	var elems []interface{}

	if !opt.streamOutput {
		elems = make([]interface{}, 0)
	}

	if err = rdb.RangeForeach(opt, func(iter *gorocksdb.Iterator) {
		var elem interface{}
		if opt.IsJsonValue {
			elem = NewKVJson(iter)
		} else {
			elem = NewKVRaw(iter)
		}
		if opt.streamOutput {
			if err = enc.Encode(elem); err != nil {
				fmt.Fprintf(os.Stderr, "GetRangeByKey: json encode %+v failed, %s\n", elem, err)
			}
		} else {
			elems = append(elems, elem)
		}
	}); err != nil {
		return err
	}
	if !opt.streamOutput {
		if err = enc.Encode(elems); err != nil {
			return fmt.Errorf("GetRangeByKey (%s %s): json marshal %d elements, failed, %s\n", opt.StartKey, opt.EndKey, len(elems), err)
		}
	}
	return nil
}

func (rdb *RDB) GetRangeByTS(opt *RangeOption, w io.Writer) error {
	var err error
	var elems []interface{}

	f := GenHijackTsInKeyByIndex(opt.TSFieldIndex, []byte(opt.KeySeparator))
	enc := json.NewEncoder(w)

	if !opt.streamOutput {
		elems = make([]interface{}, 0)
	}
	if err = rdb.RangeForeachByTS(opt, f, func(iter *gorocksdb.Iterator) {
		var elem interface{}
		if opt.IsJsonValue {
			elem = NewKVJson(iter)
		} else {
			elem = NewKVRaw(iter)
		}
		if opt.streamOutput {
			if err = enc.Encode(elem); err != nil {
				fmt.Fprintf(os.Stderr, "GetRangeByTS: json encode %+v failed, %s\n", elem, err)
			}
		} else {
			elems = append(elems, elem)
		}
	}); err != nil {
		return err
	}

	if !opt.streamOutput {
		if err = enc.Encode(elems); err != nil {
			err = fmt.Errorf("GetRangeByTS (%d %d): json marshal %d elements failed, %s\n", opt.StartTS, opt.EndTS, len(elems), err)
			return err
		}
	}
	return nil
}

func (rdb *RDB) DeleteRangeByTS(opt *RangeOption) {
	f := GenHijackTsInKeyByIndex(opt.TSFieldIndex, []byte(opt.KeySeparator))
	cf := rdb.CFHs[opt.CF]
	_ = rdb.RangeForeachByTS(opt, f, func(iter *gorocksdb.Iterator) {
		err := rdb.DeleteCF(cf, iter.Key().Data())
		if err != nil {
			fmt.Fprintf(os.Stderr, "DeleteCF: key %s cf %s failed, %s\n", string(iter.Key().Data()), opt.CF, err)
		}
	})
}

func (rdb *RDB) DeleteRangeByKey(opt *RangeOption) {
	cf := rdb.CFHs[opt.CF]
	_ = rdb.RangeForeach(opt, func(iter *gorocksdb.Iterator) {
		err := rdb.DeleteCF(cf, iter.Key().Data())
		if err != nil {
			fmt.Fprintf(os.Stderr, "DeleteCF: key %s cf %s failed, %s\n", string(iter.Key().Data()), opt.CF, err)
		}
	})
}

func (rdb *RDB) Info(w io.Writer, verbose bool) error {
	if rdb == nil {
		return fmt.Errorf("dbInfo: nil db pointer passed")
	}
	infos := make([]*CFInfo, 0)
	for k, cf := range rdb.CFHs {
		info := &CFInfo{
			Name:              k,
			EstimateNumKeys:   rdb.GetPropertyCF(KEstimateNumKeys, cf),
			TotalSstFilesSize: rdb.GetPropertyCF(KTotalSstFilesSize, cf),
		}
		if verbose {
			info.LevelStats = rdb.GetPropertyCF(KLevelStats, cf)
		}
		infos = append(infos, info)
	}
	data, err := json.Marshal(infos)
	if err != nil {
		return err
	}
	w.Write(data)
	return nil
}

// GetRange if key specified, get the value corresponding with the key
// if StartTS or EndTS specified, use GetRangeByTS
// otherwise, use GetRangeByKey
func (rdb *RDB) GetRange(opt *RangeOption, w io.Writer) error {
	var err error
	if opt == nil {
		return fmt.Errorf("%s", "range option is nil")
	}
	cf := rdb.CFHs[opt.CF]
	if cf == nil {
		return fmt.Errorf("invalid column family: %s\n", opt.CF)
	}
	if len(opt.Key) > 0 {
		err = rdb.WriteTo(cf, []byte(opt.Key), w)
		if err != nil {
			return fmt.Errorf("get using key %s failed: %s", opt.Key, err)
		}
		return nil
	}
	if opt.StartTS > 0 || opt.EndTS > 0 {
		if opt.EndTS == 0 {
			opt.EndTS = time.Now().Unix()
		}
		return rdb.GetRangeByTS(opt, w)
	}
	return rdb.GetRangeByKey(opt, w)
}

func (rdb *RDB) DeleteRange(opt *RangeOption) error {
	var err error
	if opt == nil {
		return fmt.Errorf("%s", "range option is nil")
	}
	cf := rdb.CFHs[opt.CF]
	if cf == nil {
		return fmt.Errorf("invalid column family: %s\n", opt.CF)
	}

	if len(opt.Key) > 0 {
		err = rdb.DB.DeleteCF(DefaultWriteOption, cf, []byte(opt.Key))
		if err != nil {
			return fmt.Errorf("delete key %s failed: %s", opt.Key, err)
		}
	} else {
		if opt.StartTS > 0 || opt.EndTS > 0 {
			if opt.EndTS == 0 {
				opt.EndTS = time.Now().Unix()
			}
			rdb.DeleteRangeByTS(opt)
		} else {
			rdb.DeleteRangeByKey(opt)
		}
	}
	return nil
}

func (rdb *RDB) NewRDBIterator(cf string) *RDBIterator {
	return &RDBIterator{
		Iterator: rdb.DB.NewIteratorCF(rdb.ReadOpts, rdb.CFHs[cf]),
	}
}

func (it *RDBIterator) Key() []byte {
	return it.Iterator.Key().Data()
}

func (it *RDBIterator) Close() error {
	it.Iterator.Close()
	return nil
}

func (it *RDBIterator) Value() []byte {
	return it.Iterator.Value().Data()
}

func NewRDBIteratorFrom(iter *Iterator) *RDBIterator {
	return &RDBIterator{
		Iterator: iter,
	}
}
