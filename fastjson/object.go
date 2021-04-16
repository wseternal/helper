package fastjson

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type JSONObject struct {
	entries map[string]interface{}
}

func ObjectFrom(m map[string]interface{}) *JSONObject {
	return &JSONObject{
		entries: m,
	}
}

func NewObject() *JSONObject {
	obj := &JSONObject{}
	obj.entries = make(map[string]interface{})
	return obj
}

func (obj *JSONObject) MarshalJSON() ([]byte, error) {
	return json.Marshal(obj.entries)
}

func (obj *JSONObject) UnmarshalJSON(bytes []byte) error {
	if obj.entries == nil {
		obj.entries = make(map[string]interface{})
	}
	return json.Unmarshal(bytes, &obj.entries)
}

func (obj *JSONObject) String() string {
	if data, err := json.Marshal(obj.entries); err != nil {
		return err.Error()
	} else {
		return string(data)
	}
}

func (obj *JSONObject) ContainsKey(key string) bool {
	_, found := obj.entries[key]
	return found
}

func (obj *JSONObject) Put(key string, value interface{}) {
	obj.entries[key] = value
}

// Remove the entry with given key, return the entry removed.
// if no entry with given key existed, return nil
func (obj *JSONObject) Remove(key string) interface{} {
	if elem, found := obj.entries[key]; found {
		return elem
	} else {
		return nil
	}
}

func (obj *JSONObject) GetJSONArray(key string) (*JSONArray, error)  {
	elem := obj.Get(key)
	if elem == nil {
		return NewArray(), nil
	}

	arr, ok := elem.([]interface{})
	if !ok {
		return nil, fmt.Errorf("%v(%[1]T) can not be converted to json array", elem)
	}
	return ArrayFrom(arr), nil
}

// Get would dereference the value found if it's a pointer
func (obj *JSONObject) Get(key string) interface{} {
	i, found := obj.entries[key]
	if !found {
		return nil
	}

	if i == nil {
		return nil
	}

	v := reflect.ValueOf(i)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v.Interface()
}

func (obj *JSONObject) GetIntValue(key string) int64 {
	i := obj.Get(key)
	return GetIntValue(i)
}

func (obj *JSONObject) GetFloatValue(key string) float64 {
	i := obj.Get(key)
	return GetFloatValue(i)
}

func (obj *JSONObject) GetString(key string) string {
	v := obj.Get(key)
	return GetString(v)
}

func (obj *JSONObject) GetBoolValue(key string) bool {
	i, found := obj.entries[key]
	if !found {
		return false
	}
	return GetBoolValue(i)
}

func ParseObject(data string) (*JSONObject, error) {
	obj := NewObject()
	err := obj.UnmarshalJSON([]byte(data))
	return obj, err
}

func (obj *JSONObject) Entries() (keys []string, values []interface{}) {
	if obj.entries == nil {
		return
	}
	for k, v := range obj.entries {
		keys = append(keys, k)
		values = append(values, v)
	}
	return keys, values
}

func (obj *JSONObject) Keys() []string {
	var ret []string
	if obj.entries == nil {
		return ret
	}

	for k, _ := range obj.entries {
		ret = append(ret, k)
	}
	return ret
}

func (obj *JSONObject) Values() []interface{} {
	var ret []interface{}
	if obj.entries == nil {
		return ret
	}

	for _, v := range obj.entries {
		ret = append(ret, v)
	}
	return ret
}
