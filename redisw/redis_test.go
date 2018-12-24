package redisw

import (
	"fmt"
	"testing"
)

func TestRedis(t *testing.T) {
	c, err := NewClient(nil)
	if err != nil {
		t.Fatalf("create client failed, %s\n", err)
	}
	fmt.Printf("%v\n", c)

	var errStr string
	val := c.GetInt64("kkk", -1, &errStr)

	fmt.Printf("%v %v\n", val, errStr)
}

