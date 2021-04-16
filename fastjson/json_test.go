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

func TestUnmarshal(t *testing.T) {
	str := `{"result":[{"BFP":37.3,"BMC":4.0,"BMI":26.3,"BMR":1422,"BWP":45.8,"FC":-13.3,"MA":57,"MC":-2.3,"PP":11.6,"SBC":57,"SBW":60.4,"SLM":43.6,"SMM":25.7,"VFR":8.0,"WC":-15.6},{"BFP":25.5,"BMC":3.3,"BMI":26.3,"BMR":1575,"BWP":52.9,"FC":-8.8,"MA":45,"MC":-4.9,"PP":17.2,"SBC":57,"SBW":62.3,"SLM":53.3,"SMM":28.1,"VFR":13.5,"WC":-13.7}]}`
	// str := `{"result":[123,234]}`
	obj, _ := ParseObject(str)
	arr, err := obj.GetJSONArray("result")
	fmt.Printf("%v %v\n", arr.GetJSONObject(0).GetString("WC"), err)
}