package codec

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"reflect"
)

func GobEncode(val ...interface{}) (out []byte, err error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	i := 0
	for _, v := range val {
		i++
		if err = enc.Encode(v); err != nil {
			return nil, fmt.Errorf("encode error at %d/%d, error: %s", i, len(val), err)
		}
	}
	return buf.Bytes(), nil
}

func GobDecode(data []byte, val ...interface{}) (err error) {
	enc := gob.NewDecoder(bytes.NewReader(data))
	i := 0
	for _, v := range val {
		i++
		if err = enc.Decode(v); err != nil {
			return fmt.Errorf("decode error at %d/%d, error: %s", i, len(val), err)
		}
	}
	return nil
}

func GobDecodeValue(data []byte, val ...reflect.Value) (err error) {
	enc := gob.NewDecoder(bytes.NewReader(data))
	i := 0
	for _, v := range val {
		i++
		if err = enc.DecodeValue(v); err != nil {
			return fmt.Errorf("decode error at %d/%d, error: %s", i, len(val), err)
		}
	}
	return nil
}
