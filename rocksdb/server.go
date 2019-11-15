package rocksdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/wseternal/helper/codec"
	"github.com/wseternal/helper/iohelper/sink"
	"github.com/wseternal/helper/jsonrpc"
)

func (rdb *RDB) handleSet(params interface{}) error {
	var p *string
	if p = jsonrpc.GetStringField(params, "key"); p == nil {
		return fmt.Errorf("key is required")
	}
	key := *p
	if p = jsonrpc.GetStringField(params, "value"); p == nil {
		return fmt.Errorf("value is required")
	}
	val := *p
	cf := DefaultColumnFamilyName
	if p = jsonrpc.GetStringField(params, "cf"); p != nil {
		cf = *p
	}
	return rdb.PutCF(rdb.CFHs[cf], []byte(key), []byte(val))
}

func (rdb *RDB) handleBackupInfo(params interface{}) (interface{}, error) {
	p := jsonrpc.GetStringField(params, "path")
	if p == nil || len(*p) == 0 {
		return nil, errors.New("path is required in params")
	}
	return GetBackupInfo(*p)
}

func (rdb *RDB) handleBackup(params interface{}) ([]byte, error) {
	p := jsonrpc.GetStringField(params, "path")
	if p == nil || len(*p) == 0 {
		return nil, errors.New("path is required in params")
	}
	return nil, rdb.Backup(*p)
}

func (rdb *RDB) handleInfo(params interface{}) ([]byte, error) {
	var err error
	var verbose bool

	var p *bool
	if p = jsonrpc.GetBoolField(params, "verbose"); p != nil {
		verbose = *p
	}

	snk := sink.NewBuffer()
	err = rdb.Info(snk, verbose)
	if err != nil {
		return nil, err
	}

	return snk.Bytes(), err
}

func (rdb *RDB) handleRangeAction(method string, params interface{}, ctx context.Context) ([]byte, error) {
	opt := NewRangeOption()
	var output string
	var snk *sink.Sink
	var err error

	var p *string
	var f *float64
	var t *bool

	if p = jsonrpc.GetStringField(params, "cf"); p != nil {
		opt.CF = *p
	}
	if p = jsonrpc.GetStringField(params, "key"); p != nil {
		opt.Key = *p
	}
	if p = jsonrpc.GetStringField(params, "startkey"); p != nil {
		opt.StartKey = *p
	}
	if p = jsonrpc.GetStringField(params, "endkey"); p != nil {
		opt.EndKey = *p
	}
	if t = jsonrpc.GetBoolField(params, "jsonval"); t != nil {
		opt.IsJsonValue = *t
	}
	if p = jsonrpc.GetStringField(params, "output"); p != nil {
		output = *p
		snk, err = sink.NewFile(output)
		if err != nil {
			return nil, err
		}
		opt.streamOutput = true
	} else {
		snk = sink.NewBuffer()
	}
	if f = jsonrpc.GetIntegerField(params, "limit"); f != nil {
		opt.Limit = int64(*f)
	}
	if p = jsonrpc.GetStringField(params, "sep"); p != nil {
		opt.KeySeparator = *p
	}
	if f = jsonrpc.GetIntegerField(params, "tsindex"); f != nil {
		opt.TSFieldIndex = int(*f)
		if opt.TSFieldIndex < 0 {
			return nil, fmt.Errorf("invalid tsindex: %d", opt.TSFieldIndex)
		}
	}
	if ctx != nil {
		opt.Ctx, opt.Cancel = context.WithCancel(ctx)
	}
	now := time.Now().Unix()
	if f = jsonrpc.GetIntegerField(params, "startts"); f != nil {
		opt.StartTS = int64(*f)
		if opt.StartTS < 0 {
			opt.StartTS = now + opt.StartTS
		}
	}
	if f = jsonrpc.GetIntegerField(params, "endts"); f != nil {
		opt.EndTS = int64(*f)
		if opt.EndTS < 0 {
			opt.EndTS = now + opt.EndTS
		} else if opt.EndTS == 0 {
			opt.EndTS = now
		}
	}
	switch method {
	case "get":
		err = rdb.GetRange(opt, snk)
	case "delete":
		if err = rdb.DeleteRange(opt); err == nil {
			_, _ = io.WriteString(snk, `{"result":"ok"}`)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("handleRangeAction failed, opt: %+v, %s", opt, err)
	}
	if len(output) > 0 {
		msg := fmt.Sprintf(`"write result to %s successfully"`, output)
		return []byte(msg), nil
	}
	return snk.Bytes(), nil
}

func (rdb *RDB) HandleRPC(req *jsonrpc.Request, ctx context.Context) (*jsonrpc.Response, error) {
	var res interface{}
	var err error
	var resp = &jsonrpc.Response{
		JsonRpc: jsonrpc.RPCVersion,
	}
	if req.ID != nil {
		resp.ID = *req.ID
	}

	switch req.Method {
	case "get", "delete":
		res, err = rdb.handleRangeAction(req.Method, req.Params, ctx)
	case "set":
		if err = rdb.handleSet(req.Params); err == nil {
			res = "set ok"
		}
	case "info":
		res, err = rdb.handleInfo(req.Params)
	case "flush":
		rdb.Flush()
		res = "flush ok"
	case "compact":
		rdb.Flush()
		for _, v := range rdb.CFHs {
			rdb.CompactCF(v)
		}
		res = "compact ok"
	case "backup":
		res, err = rdb.handleBackup(req.Params)
	case "backup_info":
		res, err = rdb.handleBackupInfo(req.Params)
	default:
		err = fmt.Errorf("unsupported method: %s", req.Method)
	}
	if err != nil {
		return nil, err
	}
	if buf, ok := res.([]byte); ok {
		resp.Result = json.RawMessage(buf)
	} else {
		resp.Result = res
	}
	return resp, nil
}

func (rdb *RDB) HandleHttpRequest(w http.ResponseWriter, req *http.Request) {
	var err error
	var res interface{}

	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		err = fmt.Errorf("method %s is not supported, only support post", req.Method)
		goto out
	}
	switch req.URL.Path {
	case "/api/rpc":
		var jsonReq = &jsonrpc.Request{}
		var data []byte
		if data, err = ioutil.ReadAll(req.Body); err != nil {
			goto out
		}
		if err = jsonReq.UnmarshalFrom(data); err != nil {
			goto out
		}
		res, err = rdb.HandleRPC(jsonReq, req.Context())
	default:
		err = fmt.Errorf("request URL %s is not supported", req.URL.Path)
	}
out:
	codec.WriteHttpResult(w, res, err)
}
