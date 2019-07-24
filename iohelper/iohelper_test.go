package iohelper

import (
	"crypto"
	"encoding/xml"
	"fmt"
	"testing"
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