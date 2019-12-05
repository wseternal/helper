package HttpAPI

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/wseternal/helper/codec"
)

// context with Set/Get for multiple context values
type APIContext struct {
	context.Context
}

const (
	DefaultRetryCount = 1
)

var (
	apiContextKey = new(int)
)

type apiContextValue struct {
	elems map[string]interface{}
	sync.RWMutex
}

func (c *APIContext) Delete(key string) {
	m := c.Value(apiContextKey).(*apiContextValue)
	m.Lock()
	delete(m.elems, key)
	m.Unlock()
}

func (c *APIContext) Set(key string, val interface{}) {
	m := c.Value(apiContextKey).(*apiContextValue)
	m.Lock()
	m.elems[key] = val
	m.Unlock()
}

func (c *APIContext) Inc(key string, val int) {
	m := c.Value(apiContextKey).(*apiContextValue)
	m.Lock()
	if _, found := m.elems[key]; found {
		m.elems[key] = val + m.elems[key].(int)
	} else {
		m.elems[key] = val
	}
	m.Unlock()
}

// SetNX: set the key/val only if key is not existed.
func (c *APIContext) SetNX(key string, val interface{}) {
	m := c.Value(apiContextKey).(*apiContextValue)
	m.Lock()
	if _, found := m.elems[key]; !found {
		m.elems[key] = val
	}
	m.Unlock()
}

func (c *APIContext) GetString(key, def string) string {
	v := c.Get(key)
	if v == nil {
		return def
	}
	return v.(string)
}

func (c *APIContext) GetInt(key string, def int) int {
	v := c.Get(key)
	if v == nil {
		return def
	}
	return v.(int)
}

func (c *APIContext) Get(key string) interface{} {
	m := c.Value(apiContextKey).(*apiContextValue)
	m.RLock()
	defer m.RUnlock()

	return m.elems[key]
}

// NewAPIContext create API context with key/value map
func NewAPIContext(ctx context.Context) *APIContext {
	m := &apiContextValue{
		elems: make(map[string]interface{}),
	}
	if ctx == nil {
		ctx = context.Background()
	}
	apiCtx := &APIContext{
		Context: context.WithValue(ctx, apiContextKey, m),
	}
	return apiCtx
}

func NewAPIContextCancelable(parent context.Context) (*APIContext, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	apiCtx := NewAPIContext(ctx)
	apiCtx.SetCancelFunc(cancel)
	return apiCtx, cancel
}

// NewAPIContextCancelableWith create context could be canceled either by cancel func returned,
// or by cancel event of req.Context
func NewAPIContextCancelableWith(req *http.Request) (apiCtx *APIContext, cancel context.CancelFunc) {
	apiCtx, cancel = NewAPIContextCancelable(req.Context())

	requestID := req.FormValue(ContextKeyRequestID)
	if len(requestID) > 0 {
		apiCtx.SetRequestID(requestID)
	}
	if len(req.FormValue(ContextKeyDebug)) > 0 {
		apiCtx.EnableAPIDebug()
	}
	apiCtx.Set(ContextKeyRequestPath, req.URL.Path)
	apiCtx.Set(ContextKeyRequestForm, json.RawMessage(codec.JsonMarshal(req.Form)))
	return
}

func (c *APIContext) GetRequestID() string {
	return c.GetString(ContextKeyRequestID, "")
}

func (c *APIContext) SetRequestID(id string) {
	c.Set(ContextKeyRequestID, id)
}

func (c *APIContext) GetRequestPath() string {
	return c.GetString(ContextKeyRequestPath, "")
}

func (c *APIContext) SetRequestDesc(desc string) {
	c.Set(ContextKeyRequestDesc, desc)
}

func (c *APIContext) GetRequestDesc() string {
	return c.GetString(ContextKeyRequestDesc, "")
}

func (c *APIContext) GetRequestForm() json.RawMessage {
	v := c.Get(ContextKeyRequestForm)
	if v == nil {
		return nil
	}
	return v.(json.RawMessage)
}

func (c *APIContext) EnableAPIDebug() {
	c.Set(ContextKeyDebug, true)
}

func (c *APIContext) IsAPIDebug() bool {
	return c.Get(ContextKeyDebug) != nil
}

func (c *APIContext) SetCancelFunc(f context.CancelFunc) {
	c.Set(ContextKeyCancelFunc, f)
}

func (c *APIContext) GetCancelFunc() context.CancelFunc {
	v := c.Get(ContextKeyCancelFunc)
	if v == nil {
		return nil
	}
	return v.(context.CancelFunc)
}

func (c *APIContext) GetRequestObject() interface{} {
	return c.Get(ContextKeyReqObj)
}

func (c *APIContext) SetRequestClient(client *RequestClient) {
	c.Set(ContextKeyReqClient, client)
}

func (c *APIContext) GetRequestClient() *RequestClient {
	v := c.Get(ContextKeyReqClient)
	if v == nil {
		return nil
	}
	client, ok := v.(*RequestClient)
	if !ok {
		return nil
	}
	return client
}

func (c *APIContext) GetErrorCallback() []ErrorCallback {
	v := c.Get(ContextKeyErrorCallback)
	if v == nil {
		return nil
	}
	return v.([]ErrorCallback)
}

func (c *APIContext) SetErrorCallback(cb ...ErrorCallback) {
	c.Set(ContextKeyErrorCallback, cb)
}

func (c *APIContext) SetRetryCount(val int64) {
	c.Set(ContextKeyRetryCount, val)
}

func (c *APIContext) GetRetryCount() int64 {
	v := c.Get(ContextKeyRetryCount)
	if v == nil {
		return DefaultRetryCount
	}
	res, ok := v.(int64)
	if ok {
		return res
	}
	return DefaultRetryCount
}

func (c *APIContext) GetLastAPIResponse() []byte {
	if c.Get(ContextKeyDebug) != nil {
		if data := c.Get(ContextKeyDebugResData); data != nil {
			return data.([]byte)
		}
	}
	return nil
}

func (c *APIContext) String() string {
	m := c.Value(apiContextKey).(*apiContextValue)
	m.RLock()
	defer m.RUnlock()
	return codec.JsonMarshal(m.elems)
}
