package jsonrpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"time"

	"bitbucket.org/wseternal/helper/logger"
)

type Request struct {
	ID      *int32      `json:"id,omitempty"`
	JsonRpc string      `json:"jsonrpc,omitempty"`
	Method  string      `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
}

type JsonError struct {
	Code    int         `json:"code,omitempty"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type Response struct {
	ID      int32       `json:"id,omitempty"`
	JsonRpc string      `json:"jsonrpc,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	Result  interface{} `json:"result,omitempty"`
}

type JsonObject = map[string]interface{}

const (
	RPCVersion = "2.0"
)

var (
	client = &http.Client{Timeout: time.Second * 10}
)

func (resp *Response) Data() []byte {
	data, _ := json.Marshal(resp)
	return data
}

func (resp *Response) String() string {
	data, _ := json.Marshal(resp)
	return string(data)
}

func (req *Request) UnmarshalFrom(data []byte) error {
	return json.Unmarshal(data, req)
}

func (req *Request) CreateResponse() *Response {
	res := &Response{
		JsonRpc: RPCVersion,
	}
	if req.ID != nil {
		res.ID = *req.ID
	}
	return res
}

func HttpRequest(url, method string, params interface{}, resUnmarshal interface{}) (res *Response, err error) {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	id := int32(rand.Intn(math.MaxInt32))
	enc.Encode(Request{
		ID:      &id,
		JsonRpc: "2.0",
		Method:  method,
		Params:  params,
	})
	res = &Response{
		Result: resUnmarshal,
	}
	var resp *http.Response
	resp, err = client.Post(url, "application/json", buf)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var data []byte
	if data, err = ioutil.ReadAll(resp.Body); err != nil {
		return nil, fmt.Errorf("read data from resp failed, error: %s", err)
	}
	if err = json.Unmarshal(data, res); err != nil {
		return nil, err
	}
	logger.LogD("jsonrpc response for %s %s, %+v\n", method, url, res)
	return res, nil
}

func GetField(obj interface{}, field string) interface{} {
	if obj == nil {
		return nil
	}
	var m JsonObject
	var ok bool

	if m, ok = obj.(JsonObject); ok {
		return m[field]
	}
	return nil
}

func GetStringField(obj interface{}, field string) *string {
	var tmp string
	var ok bool
	v := GetField(obj, field)
	if v == nil {
		return nil
	}
	if tmp, ok = v.(string); ok {
		return &tmp
	}
	return nil
}

func GetIntegerField(obj interface{}, field string) *float64 {
	var tmp float64
	var ok bool
	v := GetField(obj, field)
	if v == nil {
		return nil
	}
	if tmp, ok = v.(float64); ok {
		return &tmp
	}
	return nil
}

func GetBoolField(obj interface{}, field string) *bool {
	var tmp bool
	var ok bool
	v := GetField(obj, field)
	if v == nil {
		return nil
	}
	if tmp, ok = v.(bool); ok {
		return &tmp
	}
	return nil
}
