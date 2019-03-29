package rocksdb

import (
	"encoding/json"
	"fmt"
	"github.com/wseternal/helper/iohelper/sink"
	"github.com/wseternal/helper/jsonrpc"
	"io"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"time"
)

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

func (rdb *RDB) handleRangeActioin(method string, params interface{}, closeChan <- chan bool) ([]byte, error) {
	opt := NewRangeOption()
	var output string
	var snk *sink.Sink
	var err error

	var p *string
	var f *float64
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
	if closeChan != nil {
		opt.abortChan = closeChan
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
		err = rdb.DeleteRange(opt, snk)
	}
	if err != nil {
		return nil, err
	}
	if len(output) > 0 {
		msg := fmt.Sprintf(`"write result to %s successfully"`, output)
		return []byte(msg), nil
	}
	return snk.Bytes(), nil
}

func (rdb *RDB) HandleRPC(req *jsonrpc.Request, closeChan <- chan bool) (*jsonrpc.Response, error) {
	var data []byte
	var err error
	var resp = &jsonrpc.Response{
		JsonRpc: jsonrpc.RPCVersion,
	}
	if req.ID != nil {
		resp.ID = *req.ID
	}

	switch req.Method {
	case "get", "delete":
		data, err = rdb.handleRangeActioin(req.Method, req.Params, closeChan)
	case "info":
		data, err = rdb.handleInfo(req.Params)
	default:
		err = fmt.Errorf("unsupported method: %s", req.Method)
	}
	if err != nil {
		return nil, err
	}
	resp.Result = json.RawMessage(data)
	return resp, nil
}

func (rdb *RDB) HandleHttpRequest(w http.ResponseWriter, req *http.Request) {
	var err error
	var res []byte

	notify := w.(http.CloseNotifier).CloseNotify()
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		err = fmt.Errorf("method %s is not supported, only support post", req.Method)
		goto out
	}
	switch req.URL.Path {
	case "/api/rpc":
		var jsonReq = &jsonrpc.Request{}
		var resp *jsonrpc.Response
		var data []byte
		if data, err = ioutil.ReadAll(req.Body); err != nil {
			goto out
		}
		if err = jsonReq.UnmarshalFrom(data); err != nil {
			goto out
		}
		resp, err = rdb.HandleRPC(jsonReq, notify)
		res = []byte(resp.String())
	default:
		err = fmt.Errorf("request URL %s is not supported", req.URL.Path)
	}
out:
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, err.Error())
	} else {
		w.Write(res)
	}
	return
}
