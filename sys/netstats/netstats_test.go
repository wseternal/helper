package netstats

import (
	"fmt"
	"testing"
)

func TestGetMobileDataUsage(t *testing.T) {
	res, err := GetTotalDataUsageOfCurrentMonth()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("totoal used mobile data: %+v\n", res)
}
