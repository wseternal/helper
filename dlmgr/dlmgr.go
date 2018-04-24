package dlmgr

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"sync"

	"context"
	"time"

	"helper"
	"helper/iohelper/filter"
	"helper/iohelper/pump"
	"helper/iohelper/sink"
	"helper/iohelper/source"
	"helper/logger"
)

type DLCallBack func(dl *ContentDL, fullpath string, err error)

type ContentDL struct {
	Hashsum string
	URL     string
	Name    string
	// Timeout download timeout, in unit of second
	Timeout  int64
	cb       DLCallBack
	basename string
}

type Downloader struct {
	Q       []*ContentDL
	Dir     string
	c       chan *ContentDL
	dlQueue *sync.Cond
	workers []*dlWorker
}

type dlWorker struct {
	Mgr    *Downloader
	Job    *ContentDL
	S      *sync.Cond
	ID     string
	Cancel context.CancelFunc
}

const (
	DefaultDownloadPath = "/var/downloads/"
)

func defaultDLCallBack(dl *ContentDL, fullpath string, err error) {
	logger.LogI("defaultDLCallBack: %s from %s is downloaded to %s, err %v\n", dl.basename, dl.URL, fullpath, err)
}

func (o *Downloader) ClearQueue() {
	o.dlQueue.L.Lock()
	logger.LogD("ClearQueue: clear %d entries in queue\n", len(o.Q))
	o.Q = o.Q[0:0:len(o.Q)]
	o.dlQueue.L.Unlock()
}

func (o *Downloader) StopAllWorker() {
	for _, w := range o.workers {
		if w.Job != nil {
			if w.Cancel != nil {
				w.Cancel()
			} else {
				logger.LogW("StopAll: entry %s is not cancellable\n", w.Job.Name)
			}
		}
	}
}

func (o *Downloader) CancelDownload(name string) error {
	var existed *ContentDL
	var idx int
	o.dlQueue.L.Lock()
	for k, e := range o.Q {
		if e.Name == name {
			existed = e
			idx = k
			break
		}
	}
	if existed != nil {
		copy(o.Q[idx:], o.Q[idx+1:])
		o.Q = o.Q[:len(o.Q)-1]
	}
	o.dlQueue.L.Unlock()
	if existed != nil {
		return nil
	}

	var worker *dlWorker
	for _, w := range o.workers {
		if w.Job != nil && w.Job.Name == name {
			worker = w
			existed = w.Job
			break
		}
	}
	if existed != nil {
		if worker.Cancel != nil {
			worker.Cancel()
			return nil
		} else {
			return fmt.Errorf("CancelDownload: downloading entry %s is not cancellable\n", name)
		}
	}
	return fmt.Errorf("CancelDownload: can not find downloading entry: %s\n", name)
}

func (o *Downloader) Download(v *ContentDL, cb DLCallBack) error {
	var existed *ContentDL
	o.dlQueue.L.Lock()
	for _, e := range o.Q {
		if e.Name == v.Name {
			existed = e
			break
		}
	}
	o.dlQueue.L.Unlock()
	if existed != nil {
		return fmt.Errorf("downloader: entry %+v is already existed in download queue\n", existed)
	}
	if cb == nil {
		v.cb = defaultDLCallBack
	} else {
		v.cb = cb
	}
	u, err := url.Parse(v.URL)
	if err != nil {
		return fmt.Errorf("parse URL %s failed, error: %s", v.URL, err)
	}
	v.basename = path.Base(u.Path)
	if v.basename == "." || v.basename == "/" {
		logger.LogW("no basename component in url: %s, use download entry name %s instead\n", v.URL, v.Name)
		v.basename = v.Name
	}
	o.c <- v
	return nil
}

func (o *Downloader) queueLoop() {
	for {
		select {
		case dl := <-o.c:
			o.EnQueue(dl)
		}
	}
}

func (o *Downloader) newDLWorker(id string) *dlWorker {
	return &dlWorker{
		Mgr: o,
		S:   o.dlQueue,
		ID:  id,
	}
}

func (o *Downloader) EnQueue(dl *ContentDL) {
	o.dlQueue.L.Lock()
	o.Q = append(o.Q, dl)
	o.dlQueue.Signal()
	o.dlQueue.L.Unlock()
	logger.LogD("downloader: Enqueue %s %s %s\n", dl.Name, dl.basename, dl.URL)
}

func (w *dlWorker) get(dl *ContentDL, fp string) (err error) {
	var src *source.Source
	var snk *sink.Sink
	var h *filter.Hash

	var resp *http.Response
	var req *http.Request

	if helper.IsFile(fp) {
		if src, err = source.NewFile(fp); err != nil {
			return err
		}
		h = filter.NewHash(md5.New(), false)
		snk = sink.NewBuffer()
		if _, err = pump.All(src, snk, true); err != nil {
			return err
		}
		checksum := hex.EncodeToString(snk.Bytes())
		if checksum == dl.Hashsum {
			logger.LogI("%s with checksum %s is already downloaded\n", fp, dl.Hashsum)
			return nil
		}
	}
	req, err = http.NewRequest(http.MethodGet, dl.URL, nil)
	if err != nil {
		return err
	}
	var ctx context.Context
	if dl.Timeout == 0 {
		ctx, w.Cancel = context.WithCancel(context.Background())
	} else {
		ctx, w.Cancel = context.WithTimeout(context.Background(), time.Second*time.Duration(dl.Timeout))
	}
	defer w.Cancel()

	req.WithContext(ctx)

	go func() {
		resp, err = http.DefaultClient.Do(req)
		w.Cancel()
	}()

	select {
	case <-ctx.Done():
		break
	}
	// pay attention: the client.Do may not finish returning when context timed out and
	// the program flow reaches here. so, the err may be nil.
	if err != nil {
		return err
	}
	if resp == nil {
		return ctx.Err()
	}
	snk, err = sink.NewFile(fp)
	if err != nil {
		return err
	}
	src = source.New(resp.Body)
	if len(dl.Hashsum) != 0 {
		h = filter.NewHash(md5.New(), true)
		src.Chain(h)
	}
	_, err = pump.All(src, snk, true)
	if err == nil && len(dl.Hashsum) != 0 {
		checksum := hex.EncodeToString(h.Sum(nil))
		if checksum != dl.Hashsum {
			err = fmt.Errorf("%s: computed checksum %s is not equal to expected %s", fp, checksum, dl.Hashsum)
		}
	}
	return err
}

func (w *dlWorker) do() {
	dl := w.Job
	fp := path.Join(w.Mgr.Dir, dl.basename)
	logger.LogD("%s: begin downloading %s (%s) from %s\n", w.ID, dl.Name, dl.basename, dl.URL)
	err := w.get(dl, fp)
	logger.LogD("%s: finish downloading %s (%s) from %s, error: %v\n", w.ID, dl.Name, dl.basename, dl.URL, err)
	dl.cb(dl, fp, err)
	w.Job = nil
	w.Cancel = nil
}

func (w *dlWorker) Work() {
	for {
		w.S.L.Lock()
		for len(w.Mgr.Q) == 0 {
			w.S.Wait()
		}
		w.Job = w.Mgr.Q[0]
		w.Mgr.Q = w.Mgr.Q[1:]
		w.S.L.Unlock()
		w.do()
	}
}

func (o *Downloader) Start() {
	go o.queueLoop()
	for _, w := range o.workers {
		go w.Work()
	}
}

func NewDownloader(dlPath string) *Downloader {
	o := &Downloader{
		Q: make([]*ContentDL, 0),
		c: make(chan *ContentDL, 8),
	}
	if helper.IsDir(dlPath) {
		o.Dir = dlPath
	} else {
		logger.LogI("specified directory %s is invalid or not existed, using default %s\n", dlPath, DefaultDownloadPath)
		o.Dir = DefaultDownloadPath
		if err := os.MkdirAll(o.Dir, 0755); err != nil {
			temp := os.TempDir()
			logger.LogW("mkdir %s error: %s, use system temporary directory %s\n", o.Dir, err, temp)
			o.Dir = temp
		}
	}
	o.dlQueue = sync.NewCond(&sync.Mutex{})
	o.workers = make([]*dlWorker, 1)
	o.workers[0] = o.newDLWorker("DefaultDLWorker")
	return o
}
