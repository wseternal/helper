package HttpAPI

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
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

const (
	ContextKeyReqObj    = "__api_req_obj"
	ContextKeyReqClient = "__api_req_client"
	ContextKeySpent     = "__api_spent"
	ContextKeyDebug     = "__api_debug"
	//following context key will be set if debug
	ContextKeyDebugLastResData = "__api_debug_last_response_data"
	ContextKeyDebugLastReq     = "__api_debug_last_req"
	ContextKeyDebugLastAPI     = "__api_debug_last_api"
)

var (
	apis struct {
		elem map[string]*API
		sync.RWMutex
	}
)

func init() {
	apis.elem = make(map[string]*API)
}

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

func (api *API) Do(ctx *APIContext) (interface{}, error) {
	var err error
	// check request object
	reqObj := ctx.Get(ContextKeyReqObj)
	if api.RequestObjectType != nil {
		if err = helper.ValidStructType(reqObj, api.RequestObjectType, 1); err != nil {
			return nil, fmt.Errorf("invalid request obj: %v(%[1]T), %s", reqObj, err)
		}
	}
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
	if req, err = api.ReqF(ctx); err != nil {
		return nil, err
	}
	req = req.WithContext(ctx.Context)

	t1 := time.Now()
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
		ctx.Set(ContextKeyDebugLastAPI, api.Name)
		ctx.Set(ContextKeyDebugLastReq, reqObj)
		ctx.Set(ContextKeyDebugLastResData, data)

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
			fmt.Fprintf(os.Stderr, "%s: sanity check the response failed, %s\n", api.Name, err)
			return nil, err
		}
	}

	return v.Interface(), nil
}
