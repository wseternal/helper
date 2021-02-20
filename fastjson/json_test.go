package fastjson

import (
	"encoding/json"
	"fmt"
	"testing"
)

var (
	jsonStr = `{"s1": "123", "i1": 1234, "o1": {"s1":"12345", "i1":12345}, "a1":[1,2,3,4,5]}`
)

func TestParseObject(t *testing.T) {
	obj, err := ParseObject(jsonStr)
	fmt.Printf("%+v %v\n", obj, err)
	fmt.Printf("%d %d\n", obj.GetIntValue("s1"), obj.GetIntValue("i1"))
	fmt.Printf("%s\n", obj.GetString("a1"))
	data, err := json.Marshal(obj)
	fmt.Printf("%s %v\n", string(data), err)
}
