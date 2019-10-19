package redisw

import (
	"fmt"
	"testing"
	"time"
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

func TestFailover(t *testing.T) {
	conf := *DefaultFailoverOption
	conf.MasterName = "redis_m1"
	c, err := NewFailoverClient(&conf)
	if err != nil {
		t.Fatalf("create failover client failed, %s\n", err)
	}
	ticker := time.NewTicker(time.Second)

	for {
		<-ticker.C
		if err = c.Ping().Err(); err != nil {
			fmt.Printf("ping failed, %s\n", err)
		}
		if err = c.Incr("pinged").Err(); err != nil {
			fmt.Printf("increase pinged failed, %s\n", err)
		}
		var msg string
		pinged := c.GetInt64("pinged", 0, &msg)
		fmt.Printf("pinged: %d, err: %v\n", pinged, msg)
	}
}
