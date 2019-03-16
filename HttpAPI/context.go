package HttpAPI

import (
	"context"
	"sync"
)

// context with Set/Get for multiple context values
type APIContext struct {
	context.Context
}

var (
	apiContextKey = new(int)
)

type apiContextValue struct {
	elems map[string]interface{}
	sync.RWMutex
}

func (c *APIContext) Set(key string, val interface{}) {
	m := c.Value(apiContextKey).(*apiContextValue)
	m.Lock()
	m.elems[key] = val
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
		elems:make(map[string]interface{}),
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return &APIContext{
		Context: context.WithValue(ctx, apiContextKey, m),
	}
}
