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

func (arr *JSONArray) Get(index int) interface{} {
	if index >= arr.Size() {
		return nil
	}
	return arr.entries[index]
}

func (arr *JSONArray) GetIntValue(index int) int64 {
	return GetIntValue(arr.Get(index))
}

func (arr *JSONArray) GetFloatValue(index int) float64 {
	return GetFloatValue(arr.Get(index))
}

func (arr *JSONArray) GetString(index int) string {
	return GetString(arr.Get(index))
}

func (arr *JSONArray) GetBoolValue(index int) bool {
	return GetBoolValue(arr.Get(index))
}

func (arr *JSONArray) GetJSONObject(index int) *JSONObject {
	i := arr.Get(index)
	if i == nil {
		return nil
	}
	if v, ok := i.(map[string]interface{}); ok {
		return ObjectFrom(v)
	}
	return nil
}

func (arr *JSONArray) Size() int {
	return len(arr.entries)
}

func (arr *JSONArray) MarshalJSON() ([]byte, error) {
	return json.Marshal(arr.entries)
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
