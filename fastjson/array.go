package fastjson

import "encoding/json"

type JSONArray struct {
	entries []interface{}
}

func NewArray() *JSONArray {
	arr := &JSONArray{
		entries: make([]interface{}, 0),
	}
	return arr
}

func ArrayFrom(arr []interface{}) *JSONArray {
	return &JSONArray{
		entries: arr,
	}
}

func (arr *JSONArray) Size() int {
	return len(arr.entries)
}

func (arr *JSONArray) UnmarshalJSON(bytes []byte) error {
	if arr.entries == nil {
		arr.entries = make([]interface{}, 0)
	}
	return json.Unmarshal(bytes, &arr.entries)
}

func (arr *JSONArray) String() string {
	if data, err := json.Marshal(arr.entries); err != nil {
		return err.Error()
	} else {
		return string(data)
	}
}

func ParseArray(data string) (*JSONArray, error) {
	arr := NewArray()
	err := arr.UnmarshalJSON([]byte(data))
	return arr, err
}