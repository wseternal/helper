package stats

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestCounter(t *testing.T) {
	counter := NewCounter("test", TermSecond)
	var err error
	if err = counter.AddTerm(time.Second * 2); err != nil {
		t.Fatal(err)
	}
	if err = counter.AddTerm(time.Second * 5); err != nil {
		t.Fatal(err)
	}
	counter.Start()
	tick := time.NewTicker(time.Second * 1)
	max := 10
	for max > 0 {
		select {
		case <-tick.C:
			counter.Add(1, int64(rand.Intn(5)+5))
		}
		max--
		fmt.Printf(".")
	}
	data, err := json.MarshalIndent(counter.Terms, "", "    ")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("\ncounter statistics is \n%s\n", string(data))
}
