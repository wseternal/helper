package codec

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"io/ioutil"
	"os"
)

func JsonUnmarshalFrom(r io.Reader, val interface{}) error {
	dec := json.NewDecoder(r)
	return dec.Decode(val)
}

func JsonUnmarshalFromFile(fn string, val interface{}) error {
	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer f.Close()
	return JsonUnmarshalFrom(f, val)
}

func XmlUnmarshalFromFile(fn string, val interface{}) error {
	data, err := ioutil.ReadFile(fn)
	if err != nil {
		return err
	}
	return xml.Unmarshal(data, val)
}
