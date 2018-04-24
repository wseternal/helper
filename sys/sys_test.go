package sys

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestSysinfo(t *testing.T) {
	if avg, err := GetLoadAVG(); err != nil {
		t.Fatal(err)
	} else {
		t.Logf("avg is %v\n", avg)
	}
	if uptime, err := GetUptime(); err != nil {
		t.Fatal(err)
	} else {
		t.Logf("uptime is %v\n", uptime)
	}
}

func TestGetNetStatistics(t *testing.T) {
	fmt.Printf("net statistics is %v\n", GetAllNetStatistics("up"))
}

func TestUint64(t *testing.T) {
	stat := &NetStatistics{
		RxBytes: 2495422313,
		TxBytes: 122495422313,
	}
	fmt.Printf("%v\n", stat)

	data, _ := json.Marshal(stat)

	fmt.Printf("%s\n", string(data))
}

func TestUptime(t *testing.T) {
	uptime, err := GetUptime()
	fmt.Printf("uptime: %v, err:  %v\n", uptime, err)
}

func TestMachineID(t *testing.T) {
	fmt.Printf("machine id %s\n", GetMachineID())
}
