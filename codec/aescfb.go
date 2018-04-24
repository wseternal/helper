package codec

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

const (
	DefaultCryptoHash = crypto.SHA256
)

// CFBCrypto implements filter.Filter interface
type CFBCrypto struct {
	IV        []byte
	Key       []byte
	BlockSize int
	cipher.Stream
}

// NewCFBCrytpo return *CFBCrypto to do encrypt/decrypt
func NewCFBCrytpo(key, iv []byte) (*CFBCrypto, error) {
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

	cfb := &CFBCrypto{
		IV:        iv,
		Key:       key,
		BlockSize: block.BlockSize(),
	}
	if isEncrypter {
		cfb.Stream = cipher.NewCFBEncrypter(block, iv)
	} else {
		cfb.Stream = cipher.NewCFBDecrypter(block, iv)
	}
	return cfb, err
}

// Process implements the filter.Filter
func (codec *CFBCrypto) Process(p []byte, eof bool) (out []byte, err error) {
	out = make([]byte, len(p))
	if len(p) >= 0 {
		codec.XORKeyStream(out, p)
	}
	if eof {
		err = io.EOF
	}
	return out, err
}

// AesKey return keys suites to do aes encryption,
// if len(p) != 16,24,32, then use sha256 sum of p
func AesKey(nonce, key []byte) []byte {
	length := len(nonce) + len(key)
	switch length {
	case 16, 24, 32:
		data := make([]byte, length)
		copy(data, nonce)
		copy(data[len(nonce):], key)
		return data
	default:
		h := DefaultCryptoHash.New()
		h.Write(nonce)
		h.Write(key)
		return h.Sum(nil)
	}
}

func GetNonce() ([]byte, error) {
	out := make([]byte, aes.BlockSize)
	_, err := io.ReadFull(rand.Reader, out)
	return out, err
}
