package codec

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"reflect"
)

func GobEncode(val ...interface{}) (out []byte, err error) {
	buf := new(bytes.Buffer)
	if err = GobEncodeTo(buf, val); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func GobEncodeTo(w io.Writer, val ...interface{}) error {
	var err error
	enc := gob.NewEncoder(w)
	i := 0
	for _, v := range val {
		i++
		if err = enc.Encode(v); err != nil {
			return fmt.Errorf("encode error at %d/%d, error: %s", i, len(val), err)
		}
	}
	return nil
}

func GobEncodeToFile(fn string, val ...interface{}) error {
	f, err := os.OpenFile(fn, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0660)
	if err != nil {
		return err
	}
	if err = GobEncodeTo(f, val); err != nil {
		return err
	}
	return nil
}

func GobDecode(data []byte, val ...interface{}) (err error) {
	return GobDecodeFrom(bytes.NewReader(data), val)
}

func GobDecodeFrom(r io.Reader, val ...interface{}) (err error) {
	dec := gob.NewDecoder(r)
	i := 0
	for _, v := range val {
		i++
		if err = dec.Decode(v); err != nil {
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
