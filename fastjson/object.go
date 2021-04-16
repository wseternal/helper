package fastjson

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
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

func (obj *JSONObject) GetBoolValue(key string) bool {
	i, found := obj.entries[key]
	if !found {
		return false
	}
	if ret, ok := i.(bool); ok {
		return ret
	}
	return false
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
	if i == nil {
		return 0
	}
	v := reflect.ValueOf(i)
	k := v.Kind()
	if k == reflect.Float32 || k == reflect.Float64 {
		return int64(v.Float())
	}
	if k == reflect.String {
		ret, _ := strconv.ParseInt(v.String(), 10, 64)
		return ret
	}
	if k >= reflect.Int && k <= reflect.Float64 {
		return v.Int()
	}
	if k >= reflect.Uint && k <= reflect.Uint64 {
		return int64(v.Uint())
	}
	return 0
}

func (obj *JSONObject) GetFloatValue(key string) float64 {
	i := obj.Get(key)
	if i == nil {
		return 0
	}

	v := reflect.ValueOf(i)
	k := v.Type().Kind()
	if k >= reflect.Int && k <= reflect.Int64 {
		return float64(v.Int())
	}
	if k >= reflect.Uint && k <= reflect.Uint64 {
		return float64(v.Uint())
	}
	if k == reflect.Float32 || k == reflect.Float64 {
		return v.Float()
	}
	return 0
}

func (obj *JSONObject) GetString(key string) string {
	v := obj.Get(key)

	switch t := v.(type) {
	case nil:
		return "null"
	case JSONObject:
		return t.String()
	case JSONArray:
		return t.String()
	case map[string]interface{}:
		return ObjectFrom(t).String()
	case []interface{}:
		return ArrayFrom(t).String()
	case string:
		return t
	case int8, uint8, int16, uint16, int32, uint32, int64, uint64, int, uint:
		return fmt.Sprintf("%d", t)
	case float32, float64:
		return fmt.Sprintf("%f", t)
	case bool:
		return fmt.Sprintf("%t", t)
	default:
		return fmt.Sprintf("%v", t)
	}
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
