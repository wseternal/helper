package codec

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"github.com/wseternal/helper/iohelper/pump"
	"github.com/wseternal/helper/iohelper/sink"
	"github.com/wseternal/helper/iohelper/source"
	"io"
)

type CBCCrypto struct {
	IV        []byte
	Key       []byte
	BC        cipher.BlockMode
	BlockSize int
}

// NewCBCCrypto return *CBCCrypto to do encrypt/decrypt
func NewCBCCrypto(key, iv []byte) (*CBCCrypto, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	isEncrypter := false

	if iv == nil {
		iv = make([]byte, block.BlockSize())
		if _, err := io.ReadFull(rand.Reader, iv); err != nil {
			return nil, err
		}
		isEncrypter = true
	}

	cbc := &CBCCrypto{
		IV:  iv,
		Key: key,
	}
	if isEncrypter {
		cbc.BC = cipher.NewCBCEncrypter(block, iv)
	} else {
		cbc.BC = cipher.NewCBCDecrypter(block, iv)
	}
	cbc.BlockSize = cbc.BC.BlockSize()
	return cbc, err
}

// Process implements the filter.Filter
func (obj *CBCCrypto) Process(p []byte, eof bool) (out []byte, err error) {
	remainder := len(p) % obj.BlockSize
	// pkcs#7 padding
	if remainder != 0 {
		pad := obj.BlockSize - remainder
		p = append(p, bytes.Repeat([]byte{byte(pad)}, pad)...)
	}

	out = make([]byte, len(p))
	if len(p) > 0 {
		obj.BC.CryptBlocks(out, p)
	}
	if eof {
		err = io.EOF
	}
	return out, err
}

func Decrypt(encrypted, key, iv []byte) (decrypted []byte, err error) {

	inst, err := NewCBCCrypto(key, iv)
	if err != nil {
		return nil, err
	}

	src := source.NewBytes(encrypted)
	src.Chain(inst)
	snk := sink.NewBuffer()
	if _, err = pump.All(src, snk, false); err != nil {
		return nil, err
	}
	return snk.Bytes(), nil
}
