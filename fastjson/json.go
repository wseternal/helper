package fastjson

import (
	"fmt"
	"reflect"
	"strconv"
)

func GetString(v interface{}) string {
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

func GetFloatValue(i interface{}) float64 {
	if i == nil {
		return 0
	}

	v := reflect.ValueOf(i)
	k := v.Type().Kind()
	if k == reflect.String {
		ret, _ := strconv.ParseFloat(v.String(), 64)
		return ret
	}
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

func GetIntValue(i interface{}) int64 {
	if i == nil {
		return 0
	}
	v := reflect.ValueOf(i)
	k := v.Kind()
	if k == reflect.Float32 || k == reflect.Float64 {
		return int64(v.Float())
	}
	if k == reflect.String {
		ret, _ := strconv.ParseFloat(v.String(), 64)
		return int64(ret)
	}
	if k >= reflect.Int && k <= reflect.Float64 {
		return v.Int()
	}
	if k >= reflect.Uint && k <= reflect.Uint64 {
		return int64(v.Uint())
	}
	return 0
}

func GetBoolValue(i interface{}) bool {
	if i == nil {
		return false
	}
	if ret, ok := i.(bool); ok {
		return ret
	}
	return false
}