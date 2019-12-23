package iohelper

import (
	"bytes"
	"crypto"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/wseternal/helper/codec"

	"github.com/wseternal/helper"
	"github.com/wseternal/helper/iohelper/filter"
	"github.com/wseternal/helper/iohelper/pump"
	"github.com/wseternal/helper/iohelper/sink"
	"github.com/wseternal/helper/iohelper/source"

	_ "crypto/md5"
	_ "crypto/sha1"
	_ "crypto/sha256"

	lru "github.com/hashicorp/golang-lru"
)

type RateLimitedInfo struct {
	Key string
	// last do something timestamp
	LastDoTS int64
	DoCount  int64
	// total count trying to do something
	TotalCount int64

	PFL *helper.PCFileLine
}

var (
	bufPool = sync.Pool{
		New: func() interface{} {
			b := make([]byte, 8192)
			return &b
		},
	}
	RateLimitCache *lru.TwoQueueCache

	ErrRateLimited = errors.New("suppressed by rate limit")
)

func init() {
	var err error
	RateLimitCache, err = lru.New2Q(2048)
	if err != nil {
		log.Panicf("initialize rate limited log cache failed, %s\n", err)
	}
}

func DefaultGetBuffer() *[]byte {
	return bufPool.Get().(*[]byte)
}

func DefaultPutBuffer(bufp *[]byte) {
	bufPool.Put(bufp)
}

// CatTextFile return file context as string with leading/suffix spaces stripped,
// if file is not read successfully, the defult value is returned instead.
func CatTextFile(fn string, dfl string) string {
	if data, err := ioutil.ReadFile(fn); err == nil {
		data = bytes.TrimSpace(data)
		return string(data)
	}
	return dfl
}

func HashsumFromFile(fn string, h crypto.Hash) ([]byte, error) {
	if !h.Available() {
		return nil, fmt.Errorf("hash function (%d) is not existed", h)
	}
	if !helper.IsFile(fn) {
		return nil, fmt.Errorf("file %s is not existed", fn)
	}
	src, err := source.NewFile(fn)
	if err != nil {
		return nil, err
	}
	sum := filter.NewHash(h.New(), false)
	src.Chain(sum)
	snk := sink.NewBuffer()
	if _, err = pump.All(src, snk, true); err != nil {
		return nil, err
	}
	return snk.Bytes(), nil
}

func ValidFileSum(fn string, expectSum string, hash crypto.Hash) error {
	data, err := HashsumFromFile(fn, hash)
	if err != nil {
		return err
	}
	sum := hex.EncodeToString(data)
	if sum != expectSum {
		return fmt.Errorf("computed sum: %s <> expected sum: %s", sum, expectSum)
	}
	return nil
}

func HttpDownload(url string, dst string, expectedSum string) error {
	var err error
	if err = ValidFileSum(dst, expectedSum, crypto.MD5); err == nil {
		fmt.Printf("%s (%s) is already downloaded", dst, expectedSum)
		return nil
	}
	var resp *http.Response
	resp, err = http.Get(url)
	if err != nil {
		return fmt.Errorf("http get %s failed, %s", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http get %s: %d %s", url, resp.StatusCode, resp.Status)
	}
	src := source.New(resp.Body)
	h := filter.NewHash(md5.New(), true)
	src.Chain(h)

	snk, err := sink.NewFile(dst)
	if err != nil {
		return fmt.Errorf("create download destination file %s failed, %s\n", dst, err)
	}
	defer snk.Close()

	if _, err = pump.All(src, snk, false); err != nil {
		return fmt.Errorf("save to file %s failed, %s\n", dst, err)
	}
	sum := hex.EncodeToString(h.Sum(nil))
	if len(expectedSum) > 0 && sum != expectedSum {
		return fmt.Errorf("invalid sum %s for file %s, expect %s", sum, dst, expectedSum)
	}
	return nil
}

func CopyFile(dst, src string) (int, error) {
	source, err := source.NewFile(src)
	if err != nil {
		return 0, err
	}
	var snk *sink.Sink
	snk, err = sink.NewFile(dst)
	if err != nil {
		return 0, err
	}
	return pump.All(source, snk, true)
}

func WriteWithRate(w io.Writer, intervalTS int64, key string, format string, args ...interface{}) {
	pfl := helper.GetPCFileLine(1)
	now := time.Now().Unix()

	_key := fmt.Sprintf("%s_%d", key, pfl.PC)
	v, ok := RateLimitCache.Get(_key)
	var info *RateLimitedInfo
	if ok {
		info = v.(*RateLimitedInfo)
		info.TotalCount++
		if now-info.LastDoTS < intervalTS {
			return
		}
	} else {
		info = &RateLimitedInfo{
			Key:        key,
			PFL:        pfl,
			TotalCount: 1,
		}
		RateLimitCache.Add(_key, info)
	}
	info.LastDoTS = now
	info.DoCount++
	fmt.Fprintf(w, format, args...)
}

func ErrorfWithRate(intervalTS int64, key string, format string, args ...interface{}) error {
	pfl := helper.GetPCFileLine(1)
	now := time.Now().Unix()

	err := fmt.Errorf(format, args...)

	_key := fmt.Sprintf("%s_%d", key, pfl.PC)
	v, ok := RateLimitCache.Get(_key)
	var info *RateLimitedInfo
	if ok {
		info = v.(*RateLimitedInfo)
		info.TotalCount++
		if now-info.LastDoTS < intervalTS {
			return fmt.Errorf("%w:%s", ErrRateLimited, err)
		}
	} else {
		info = &RateLimitedInfo{
			Key:        key,
			PFL:        pfl,
			TotalCount: 1,
		}
		RateLimitCache.Add(_key, info)
	}
	info.LastDoTS = now
	info.DoCount++
	return err
}

func (info *RateLimitedInfo) String() string {
	return codec.JsonMarshal(info)
}
