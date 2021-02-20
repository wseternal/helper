package sqlimpl

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"testing"

	"github.com/wseternal/helper/fastjson"
)

var (
	dsn    = "websocket:cloudfi@tcp(127.0.0.1:3306)/websocket"
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
	dsn = "root:weixiaoxin123@tcp(121.42.157.74:3316)/wifiadx"
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
	dsn = "root:Zkzx3411@tcp(www.cloudfi.cn:3306)/wifiadx"
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
	t.Logf("%v %v\n", obj, err)
}
