package HttpAPI

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

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

func (c *APIContext) Get(key string) interface{} {
	m := c.Value(apiContextKey).(*apiContextValue)
	m.RLock()
	defer m.RUnlock()

	return m.elems[key]
}

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
	requestID := fmt.Sprintf("apictx_%d_%p", time.Now().Unix(), ctx)
	apiCtx.Set(ContextKeyRequestID, requestID)
	return apiCtx
}

func (c *APIContext) EnableAPIDebug() {
	c.Set(ContextKeyDebug, true)
}

func (c *APIContext) IsAPIDebug() bool {
	return c.Get(ContextKeyDebug) != nil
}

func (c *APIContext) GetRequestObject() interface{} {
	return c.Get(ContextKeyReqObj)
}

func (c *APIContext) GetRequestClient() interface{} {
	return c.Get(ContextKeyReqClient)
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

func NewAPIContextWithCloseNotifier(w http.ResponseWriter, req *http.Request) (apiCtx *APIContext, cancel context.CancelFunc) {
	var ctx context.Context
	ctx, cancel = context.WithCancel(context.Background())
	apiCtx = NewAPIContext(ctx)

	requestID := req.FormValue(ContextKeyRequestID)
	if len(requestID) > 0 {
		apiCtx.Set(ContextKeyRequestID, requestID)
	}

	if len(req.FormValue(ContextKeyDebug)) > 0 {
		apiCtx.EnableAPIDebug()
	}

	apiCtx.Set(ContextKeyRequestPath, req.URL.Path)
	apiCtx.Set(ContextKeyRequestForm, json.RawMessage(codec.JsonMarshal(req.Form)))

	// cancel the operation if client disconnected
	c := w.(http.CloseNotifier).CloseNotify()
	go func() {
		select {
		case <-c:
			fmt.Fprintf(os.Stderr, "client closed the connection, abort %s\n", apiCtx.Get(ContextKeyRequestID).(string))
			cancel()
		case <-apiCtx.Done():
			break
		}
	}()
	return
}

func (c *APIContext) String() string {
	m := c.Value(apiContextKey).(*apiContextValue)
	m.RLock()
	defer m.RUnlock()
	return codec.JsonMarshal(m.elems)
}
