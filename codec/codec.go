package codec

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/wseternal/helper/jsonrpc"
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

// JsonMarhsal return {"error":"err returned by json.Marhsal"} on error
func JsonMarshal(i interface{}) string {
	out, err := json.Marshal(i)
	if err != nil {
		return ToJsonError(err)
	}
	return string(out)
}

func ToJsonError(err error) string {
	obj := struct {
		Error string `json:"error"`
	}{
		Error: err.Error(),
	}
	return JsonMarshal(obj) + "\n"
}

func ToJsonResult(val interface{}) string {
	obj := &struct {
		Result interface{} `json:"result"`
	}{
		Result: val,
	}
	return JsonMarshal(obj) + "\n"
}

func WriteJsonRpcString(w io.Writer, res interface{}, err error) {
	if err == nil {
		if res == nil {
			res = "ok"
		}
		_, _ = io.WriteString(w, ToJsonResult(res))
		return
	}
	_, _ = io.WriteString(w, ToJsonError(err))
}

func WriteString(w io.Writer, i interface{}) {
	switch v := i.(type) {
	case string:
		_, _ = io.WriteString(w, v)
	case []byte:
		_, _ = w.Write(v)
	default:
		if tmp, ok := i.(fmt.Stringer); ok {
			_, _ = io.WriteString(w, tmp.String())
			break
		}
		WriteJsonString(w, i)
	}
}

func WriteJsonString(w io.Writer, res interface{}) {
	_, _ = io.WriteString(w, JsonMarshal(res))
}

// resp will be consumed and closed
func HttpError(resp *http.Response, err error) error {
	if resp == nil {
		return err
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("resp: %d:%s, content: %s, error: %s", resp.StatusCode, resp.Status, string(data), err)
	}

	// err == nil, check repsponse object wheter are json object with error field
	// for potential json error object, at east 11 bytes, as: {"error":x}
	if len(data) < 11 {
		return nil
	}
	res := &jsonrpc.Response{}
	err = json.Unmarshal(data, res)
	if err == nil && res.Error != nil {
		return fmt.Errorf("resp: %d:%s, content: %s, error: %s", resp.StatusCode, resp.Status, string(data), err)
	}
	return nil
}
