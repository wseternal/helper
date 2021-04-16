package sqlimpl

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/wseternal/helper/fastjson"
)

var (
	dsn    = "user:pass@tcp(127.0.0.1:3306)/db"
	tables = []*DataTable{
		&DataTable{
			Name: "test",
			CreateStatement: `CREATE TABLE IF NOT EXISTS test (
				id INT AUTO_INCREMENT, mac CHAR(12) NOT NULL,
				content TEXT,
				PRIMARY KEY(mac),
				KEY id(id)
			) ENGINE=INNODB;`,
		},
	}
)

func TestDataTableActions(t *testing.T) {
	impl, err := ConnectDB("mysql", dsn)
	if err != nil {
		t.Fatalf("connect to db %s failed, error: %s\n", dsn, err)
	}
	defer impl.Close()

	err = impl.InitTables(tables)
	if err != nil {
		t.Fatalf("initialize tables failed: %s\n", err)
	}
	var mac string
	condition := `where mac="00c000112233"`
	err = tables[0].Find("mac", condition, &mac)
	if err == nil {
		fmt.Printf("found mac: %s\n", mac)
		buf := make([]byte, 32)
		io.ReadFull(rand.Reader, buf)
		buffer := new(bytes.Buffer)
		enc := base64.NewEncoder(base64.StdEncoding, buffer)
		enc.Write(buf)
		enc.Close()
		res, err := tables[0].Update(condition, []string{"content"}, string(buffer.Bytes()))
		if err != nil {
			t.Fatalf("update failed: %s\n", err)
		}
		rows, err := res.RowsAffected()
		if err == nil {
			fmt.Printf("rows affected is %#v\n", rows)
		}
	} else {
		t.Logf("find mac failed: %s\n", err)
		res, err := tables[0].Insert([]string{"mac", "content"}, "00c000112233", "test content")
		if err != nil {
			t.Fatalf("insert failed: %s\n", err)
		}
		fmt.Printf("res is %#v\n", res)
	}
}

func TestGetTable(t *testing.T) {
	dsn = "root:weixiaoxin123@tcp(127.0.0.1:3306)/sandbox"
	impl, err := ConnectDB("mysql", dsn)
	if err != nil {
		t.Fatalf("connect to db %s failed, error: %s\n", dsn, err)
	}
	defer impl.Close()

	var dt *DataTable
	dt, err = impl.GetTable("wifi_ch_userinfo")
	fmt.Printf("%v %v\n", dt, err)
}

func TestScanJSONObject(t *testing.T) {
	dsn = "root:weixiaoxin123@tcp(127.0.0.1:3306)/sandbox"
	impl, err := ConnectDB("mysql", dsn)
	if err != nil {
		t.Fatalf("connect to db %s failed, error: %s\n", dsn, err)
	}
	defer impl.Close()

	var ret []*fastjson.JSONObject
	if ret, err = impl.FetchAll("select * from wxwork_login"); err != nil {
		t.Fatalf("fetch all failed, %s\n", err)
	}

	t.Logf("found %d entries\n", len(ret))
	for _, elem := range ret {
		t.Logf("%s\n", elem.String())
	}

	var obj *fastjson.JSONObject
	obj, err = impl.FetchOne("select * from wxwork_app limit 1")
	t.Logf("%v %v\n", obj.String(), err)
	t.Logf("%d %d\n", obj.GetIntValue("updatets"), obj.GetIntValue("expiresin"))

	obj, err = impl.FetchOne(`select * from wifi_ch_record_body as b right join wifi_ch_records as r on b.record_id=r.id where r.id=?`, 82306418)
	t.Logf("impedance is null: %t\n", obj.Get("impedance") == nil)
	t.Logf("%v %v\n", obj, err)
}

func TestInsertJsonObject(t *testing.T) {
	dsn = "root:Zkzx3411@tcp(www.cloudfi.cn:3306)/wifiadx"
	impl, err := ConnectDB("mysql", dsn)
	if err != nil {
		t.Fatalf("connect to db %s failed, error: %s\n", dsn, err)
	}
	defer impl.Close()

	obj := fastjson.NewObject()
	obj.Put("mac", "112233445566")
	obj.Put("ts", time.Now().Unix())
	obj.Put("location", "test location")
	obj.Put("lat", "123")
	obj.Put("lng", "234")
	res, err := impl.Insert("scale_location", true, obj)
	fmt.Printf("%v %v\n", res, err)
}

func getScaleRecord(impl *SQLImpl, rid int64) (*fastjson.JSONObject, error) {
	obj, err := impl.FetchOne("select * from wifi_ch_record_body as b right join wifi_ch_records as r on b.record_id=r.id where r.id=?", rid)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, fmt.Errorf("no record exists for record_id: %d", rid)
	}

	ret := fastjson.NewObject()

	impedance := obj.GetIntValue("impedance")
	age := obj.GetIntValue("age")
	height := obj.GetIntValue("height")
	weight := obj.GetIntValue("weight")
	bmi := obj.GetFloatValue("bmi")
	ret.Put("height", height)
	ret.Put("weight", weight)
	ret.Put("bmi", bmi)
	ret.Put("nickname", obj.Get("nickname"))
	ret.Put("add_time", obj.Get("add_time"))
	if age == 0 {
		age = 30
	}

	gender := obj.GetString("gender")

	if impedance > 0 {
		// prepare bodyfat data
		var resp *http.Response
		var bodyfatRequest *http.Request

		body := new(bytes.Buffer)
		part := multipart.NewWriter(body)
		part.WriteField("weight", fmt.Sprintf("%d", weight*10))
		part.WriteField("height", fmt.Sprintf("%d", height))
		part.WriteField("impedance", fmt.Sprintf("%d", impedance))
		part.WriteField("age", fmt.Sprintf("%d", age))

		part.Close()

		bodyfatRequest, err = http.NewRequest(http.MethodPost, "http://srv.cloudfi.cn/api/bodyfat/get", body)
		bodyfatRequest.Header.Add("Content-Type", part.FormDataContentType())
		if resp, err = http.DefaultClient.Do(bodyfatRequest); err != nil {
			fmt.Fprintf(os.Stderr, "bodyfat request with body %s failed, %s\n", body.String(), err)
			goto out
		}
		var data []byte
		if data, err = ioutil.ReadAll(resp.Body); err != nil {
			fmt.Fprintf(os.Stderr, "read bodyfat response failed, %s\n", err)
			goto out
		}

		var bodyfat *fastjson.JSONObject
		if bodyfat, err = fastjson.ParseObject(string(data)); err != nil {
			fmt.Fprintf(os.Stderr, "unmarshal %s to json object failed, %s\n", string(data), err)
			goto out
		}
		if bodyfat.ContainsKey("error") {
			fmt.Fprintf(os.Stderr, "error set in bodyfat response: %s\n", bodyfat.GetString("error"))
			goto out
		}

		var result *fastjson.JSONArray
		if result, err = bodyfat.GetJSONArray("result"); err != nil {
			fmt.Fprintf(os.Stderr, "convert bodyfat result %s to json array failed, %s\n", string(data), err)
			goto out
		}
		if result.Size() != 2 {
			fmt.Fprintf(os.Stderr, "invalid size for bodyfat result: %s\n", result)
			goto out
		}

		// result[0] is for female; result[1] is for male
		switch gender {
		case "1":
			ret.Put("bodyfat", result.GetJSONObject(1))
		case "0":
			ret.Put("bodyfat", result.GetJSONObject(0))
		default:
			// if gender is unknown, regard it as male
			ret.Put("bodyfat", result.GetJSONObject(1))

		}
		resp.Body.Close()
	}
out:
	return ret, nil
}

func TestSandbox(t *testing.T) {
	dsn = "root:Zkzx3411@tcp(www.cloudfi.cn:3306)/wifiadx"
	impl, err := ConnectDB("mysql", dsn)
	if err != nil {
		t.Fatalf("connect to db %s failed, error: %s\n", dsn, err)
	}
	defer impl.Close()
	res, err := getScaleRecord(impl, 128846051)
	fmt.Printf("%v %v\n", res, err)
}
