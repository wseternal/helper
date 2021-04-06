package codec

import (
	"crypto/aes"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
)

type info struct {
	A, B int
}

func TestGobCodec(t *testing.T) {
	o := &info{
		A: 1,
		B: 2,
	}
	out, err := GobEncode(o)
	if err != nil {
		t.Fatal(err)
	}
	o1 := &info{}
	err = GobDecode(out, o1)
	if err != nil {
		t.Fatalf("gob decode failed, error: %s\n", err)
	}
	fmt.Printf("o1 is %#v\n", o1)
}

func TestWrite(t *testing.T) {
	var res interface{}

	res = "string literal\n"
	WriteString(os.Stderr, res)

	res = []byte("[]byte literal\n")
	WriteString(os.Stderr, res)

	res = &info{
		A: 1,
		B: 2,
	}
	WriteString(os.Stderr, res)
}

func TestAesCBC(t *testing.T) {
	key, _ := hex.DecodeString("6368616e676520746869732070617373")
	data, _ := hex.DecodeString("73c86d43a9d700a253a96c85b0f6b03ac9792e0e757f869cca306bd3cba1c62b")

	decrypted, err := Decrypt(data[aes.BlockSize:], key, data[:aes.BlockSize])
	fmt.Printf("%v %v\n", string(decrypted), err)
}
