package syncmap

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

type Map struct {
	elems              reflect.Value
	keyType, valueType reflect.Type
	sync.RWMutex       `json:"-"`
}

var (
	ErrNilKey = errors.New("non-nil key is required")
)

func New(keyType, valueType reflect.Type) (*Map, error) {
	if keyType == nil || valueType == nil {
		return nil, errors.New("non-nil keyType and valueType are both required")
	}
	if !keyType.Comparable() {
		return nil, fmt.Errorf("keyType: %s is not comparable", keyType.String())
	}
	m := &Map{
		keyType:   keyType,
		valueType: valueType,
	}
	m.elems = reflect.MakeMap(reflect.MapOf(keyType, valueType))
	return m, nil
}

func (m *Map) Delete(key interface{}) error {
	return m.Add(key, nil)
}

// Add given key value pair. if value is nil, the corresponding key will be deleted
func (m *Map) Add(key, value interface{}) error {
	if key == nil {
		return ErrNilKey
	}
	k := reflect.ValueOf(key)
	if k.Type() != m.keyType {
		return fmt.Errorf("wrong type of key: %T, require %s", key, m.keyType.String())
	}

	v := reflect.ValueOf(value)
	if value != nil {
		if v.Type() != m.valueType {
			return fmt.Errorf("wrong type of value: %T, require %s", value, m.valueType.String())
		}
	}
	m.Lock()
	m.elems.SetMapIndex(k, v)
	m.Unlock()
	return nil
}

func (m *Map) _get(key interface{}) reflect.Value {
	if key == nil {
		return reflect.ValueOf(nil)
	}
	k := reflect.ValueOf(key)
	if k.Type() != m.keyType {
		return reflect.ValueOf(nil)
	}
	m.RLock()
	defer m.RUnlock()
	elem := m.elems.MapIndex(k)
	return elem
}

func (m *Map) Has(key interface{}) bool {
	v := m._get(key)
	return v.IsValid()
}

// Get return the value with given element type at created, str := m.Get(key).(string)
func (m *Map) Get(key interface{}) interface{} {
	v := m._get(key)
	if !v.IsValid() {
		return nil
	}
	return v.Interface()
}

// ValueSlice return slice of values with given element type, e.g: arr := m.ValueSlice().([]string)
func (m *Map) ValueSlice() interface{} {
	s := reflect.MakeSlice(reflect.SliceOf(m.valueType), 0, 0)
	m.RLock()
	iter := m.elems.MapRange()
	for iter.Next() {
		s = reflect.Append(s, iter.Value())
	}
	m.RUnlock()
	return s.Interface()
}

func (m *Map) ForEach(cb func(key, value interface{})) {
	m.RLock()
	iter := m.elems.MapRange()
	for iter.Next() {
		cb(iter.Key().Interface(), iter.Value().Interface())
	}
	m.RUnlock()
}