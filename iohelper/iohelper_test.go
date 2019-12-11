package iohelper

import (
	"crypto"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestValidFilesum(t *testing.T) {
	fmt.Printf("%v\n", ValidFileSum("./shared_preference.go", "d28ac38a050061cd6989ece30813f870", crypto.MD5))
}

func TestSP(t *testing.T) {
	str := `<map>
    <string name="bearer">0</string>
    <boolean name="carrier_enabled" value="true" />
</map>`
	sp := &SharedPreference{}
	xml.Unmarshal([]byte(str), sp)

	for _, elem := range sp.SElems {
		fmt.Printf("%v\n", *elem)
	}
	for _, elem := range sp.BElems {
		fmt.Printf("%v\n", *elem)
	}
}

func TestWriteWithRate(t *testing.T) {
	var err error
	for i := 0; i < 10; i++ {
		err = ErrorfWithRate(5, "%s\n", time.Now().String())
		if errors.Is(err, ErrRateLimited) {
			time.Sleep(time.Second * 1)
			continue
		}
		fmt.Printf("%s\n", err)
		time.Sleep(time.Second * 1)
	}
	for _, elem := range RateLimitCache.Keys() {
		v, err := RateLimitCache.Get(elem)
		fmt.Printf("%s %v\n", v, err)
	}

	for i := 0; i < 10; i++ {
		WriteWithRate(os.Stderr, 5, "%s\n", time.Now().String())
		time.Sleep(time.Second * 1)
	}
	for _, elem := range RateLimitCache.Keys() {
		v, err := RateLimitCache.Get(elem)
		fmt.Printf("%s %v\n", v, err)
	}
}
