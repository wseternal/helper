package HttpAPI

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wseternal/helper"
	"github.com/wseternal/helper/codec"
)

type RequestClient struct {
	*http.Client

	// tr is used as the Transport for client
	tr *http.Transport
}

// ctx: the context passed by api.Do
type NewRequestFunc func(ctx *APIContext) (*http.Request, error)

// resValuePtr: ptr to the response object, resValuePtr.Elem() will be the struct object
type ResponseSanityCheckFunc func(ctx *APIContext, resValuePtr reflect.Value) error

type API struct {
	RequestObjectType, ResponseObjectType reflect.Type
	ReqF                                  NewRequestFunc
	ResCheckF                             ResponseSanityCheckFunc
	Name                                  string
}

type ErrorCallback func(ctx *APIContext, err error) CallbackResult
type CallbackResult int

const (
	ContextKeyCancelFunc  = "__api_cancel_func"
	ContextKeyRequestDesc = "__api_request_desc"
	ContextKeyRequestID   = "__api_request_id"
	ContextKeyRequestPath = "__api_request_path"
	ContextKeyRequestForm = "__api_request_form"

	ContextKeyErrorCallback = "__api_error_callback"
	ContextKeyRetryCount    = "__api_retry_count"
	ContextKeyReqObj        = "__api_req_obj"
	ContextKeyReqClient     = "__api_req_client"
	ContextKeySpent         = "__api_spent"
	ContextKeyDebug         = "__api_debug"

	//following context key will be set if debug
	ContextKeyDebugResData = "__api_debug_response_data"
	ContextKeyDebugAPI     = "__api_debug_api"
)

const (
	// skip left error callbacks, fail the API request without retry
	CallbackResultAbort CallbackResult = iota

	// skip left error callbacks, retry the API request
	CallbackResultRetry

	// continue invoke left error callbacks
	CallbackResultContinue
)

var (
	apis struct {
		elem map[string]*API
		sync.RWMutex
	}

	OnGoingAPIs struct {
		Elems map[string]*APIContext
		sync.RWMutex
	}
)

func init() {
	apis.elem = make(map[string]*API)
	OnGoingAPIs.Elems = make(map[string]*APIContext)
}

// request type must be type to a valid struct
// response type shall be type to a valid struct ( or struct ptr)
func RegisterAPI(name string, f NewRequestFunc, resCheckF ResponseSanityCheckFunc, reqType, responseType reflect.Type) (*API, error) {
	if responseType == nil || helper.ValidStructType(responseType, nil, 1) != nil {
		return nil, fmt.Errorf("reponse type %v is nil or not a valid struct (or ptr to struct)", responseType)
	}

	if reqType != nil && helper.ValidStructType(reqType, nil, 1) != nil {
		return nil, fmt.Errorf("request type %s is a valid struct (or ptr to struct)", reqType)
	}

	apis.Lock()
	defer apis.Unlock()
	if _, found := apis.elem[name]; found {
		return nil, fmt.Errorf("API name: %s is already registerd", name)
	} else {
		apis.elem[name] = &API{
			Name:               name,
			RequestObjectType:  reqType,
			ResponseObjectType: responseType,
			ReqF:               f,
			ResCheckF:          resCheckF,
		}
	}
	return apis.elem[name], nil
}

// GetAPI: return API object with given name, if not found, return dummy API
// that will return error on any further action
func GetAPI(name string) *API {
	apis.RLock()
	defer apis.RUnlock()
	return apis.elem[name]
}

func (api *API) _do(ctx *APIContext) (interface{}, error) {
	var err error

	// generate the request and use given/default client to send out request
	var req *http.Request
	var res *http.Response

	client := DefaultClient
	if ctx.Get(ContextKeyReqClient) != nil {
		var ok bool
		if client, ok = ctx.Get(ContextKeyReqClient).(*RequestClient); !ok {
			return nil, fmt.Errorf("request client in context is not a valid type *RequestClient, it's %T", ctx.Get(ContextKeyReqClient))
		}
	}
	reqObj := ctx.GetRequestObject()
	if req, err = api.ReqF(ctx); err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	t1 := time.Now()

	api.preDo(ctx, req)
	defer api.postDo(ctx)

	if res, err = client.Do(req); err != nil {
		return nil, err
	}
	defer res.Body.Close()

	ctx.Set(ContextKeySpent, time.Now().Sub(t1).Nanoseconds())

	// unmarshal response data to response object
	t := api.ResponseObjectType
	// response type could be a struct or ptr to struct
	var v reflect.Value
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	v = reflect.New(t)
	if ctx.IsAPIDebug() {
		data, _ := ioutil.ReadAll(res.Body)
		ctx.Set(ContextKeyDebugAPI, api.Name)
		ctx.Set(ContextKeyDebugResData, data)

		defer func() {
			if err == nil {
				return
			}
			fmt.Fprintf(os.Stderr, "error context: apiname: %s, request: %s\n", api.Name, req.URL.String())
			fmt.Fprintf(os.Stderr, "error context: request object: %+v\n", reqObj)
			fmt.Fprintf(os.Stderr, "error context: request object content: %s\n", codec.JsonMarshal(reqObj))
			fmt.Fprintf(os.Stderr, "error context: response content: %s\n", string(data))

		}()
		if err = json.Unmarshal(data, v.Interface()); err != nil {
			return nil, fmt.Errorf("unmarshal response %s to %s failed, %s", string(data), t.String(), err)
		}
	} else {
		if err = codec.JsonUnmarshalFrom(res.Body, v.Interface()); err != nil {
			return nil, fmt.Errorf("unmarshal response to %s failed, %s", t.String(), err)
		}
	}

	if api.ResCheckF != nil {
		if err = api.ResCheckF(ctx, v); err != nil {
			fmt.Fprintf(os.Stderr, "%s: request obj %+v, sanity check the response failed, %s\n", api.Name, reqObj, err)
			return nil, err
		}
	}
	return v.Interface(), nil
}

func (api *API) preDo(ctx *APIContext, req *http.Request) {
	reqID := ctx.GetRequestID()
	if len(reqID) == 0 {
		reqID = fmt.Sprintf("%s_%d_%p ", api.Name, time.Now().Unix(), ctx)
	}
	OnGoingAPIs.Lock()
	ctx.SetRequestID(reqID)
	OnGoingAPIs.Elems[reqID] = ctx
	OnGoingAPIs.Unlock()
}

func (api *API) postDo(ctx *APIContext) {
	OnGoingAPIs.Lock()
	delete(OnGoingAPIs.Elems, ctx.GetRequestID())
	OnGoingAPIs.Unlock()
}

// ContextKeyReqObj must be set if RequestObjectType is not nil
func (api *API) Do(ctx *APIContext) (interface{}, error) {
	var err error

	// check request object
	reqObj := ctx.GetRequestObject()
	if api.RequestObjectType != nil {
		if err = helper.ValidStructType(reqObj, api.RequestObjectType, 1); err != nil {
			return nil, fmt.Errorf("invalid request obj: %v(%[1]T), %s", reqObj, err)
		}
	}
	var res interface{}

	var tried int64 = 0
	maxRetry := ctx.GetRetryCount()
	for {
		tried++
		res, err = api._do(ctx)
		// no error occurred, break
		if err == nil {
			break
		}

		needRetry := false
		fmt.Fprintf(os.Stderr, "API(%s).Do failed, %s\n", api.Name, err)
		// error occurred, invoke error callbacks
		if cbs := ctx.GetErrorCallback(); cbs != nil {
			for _, cb := range cbs {
				cbRes := cb(ctx, err)
				if cbRes == CallbackResultAbort {
					fmt.Printf("API(%s).Do erorr callback result is abort, return...\n", api.Name)
					goto out
				}
				if cbRes == CallbackResultRetry {
					fmt.Printf("API(%s).Do erorr callback result is retry, retrying...\n", api.Name)
					needRetry = true
					break
				}
			}
		}
		if !needRetry {
			break
		}

		fmt.Printf("API(%s).Do: tried: %d, max retry: %d\n", api.Name, tried, maxRetry)
		// check retry count
		if tried > maxRetry {
			break
		}
	}
out:
	return res, err
}

func DoAPI(ctx *APIContext, name string, reqObj interface{}) (interface{}, error) {
	api := GetAPI(name)
	if api == nil {
		return nil, fmt.Errorf("API %s is not registerd", name)
	}
	if ctx == nil {
		ctx = NewAPIContext(nil)
	}
	ctx.Set(ContextKeyReqObj, reqObj)
	res, err := api.Do(ctx)
	if err != nil {
		return nil, err
	}
	if ctx.IsAPIDebug() {
		fmt.Printf("API %s invoked successfully, spent %v\n", name, ctx.Get(ContextKeySpent))
	}
	return res, nil
}

// return http request path fields separated by "/"
// e.g.: /a/b/c => ["a", "b", "c"]
func SplitRequestPath(req *http.Request) []string {
	return strings.Split(strings.Trim(req.URL.Path, "/"), "/")
}

func GetBooleanFormValue(req *http.Request, key string, def bool) bool {
	str := req.FormValue(key)
	if len(str) == 0 || len(str) > 5 {
		return def
	}
	str = strings.ToLower(str)
	switch str {
	case "false":
		return false
	case "true":
		return true
	default:
		return def
	}
}

func GetStringFormValue(req *http.Request, key string, def string) string {
	str := req.FormValue(key)
	if len(str) == 0 {
		return def
	}
	return str
}

func GetInt64FormValue(req *http.Request, key string, def int64) int64 {
	str := req.FormValue(key)
	if len(str) == 0 {
		return def
	}
	v, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetInt64FormValue: convert %s to int64 failed, %s", str, err)
		return def
	}
	return v
}

func ErrCallbackDisableRetry(ctx *APIContext, err error) CallbackResult {
	return CallbackResultAbort
}
