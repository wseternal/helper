package helper

import (
	"errors"
)

// increase the hex string slice "key" by 1,
// e.g.: 01fe -> 01ff; 10ff -> 1100
// in case of error, the original key plus error are returned
func NextKey(key string) (string, error) {
	if len(key) == 0 {
		return key, errors.New("empty key")
	}
	s := []byte(key)
	i := len(s) - 1
	for ;i >= 0; i-- {
		if s[i] == 'f' || s[i] == 'F' {
			s[i] = '0'
			continue
		}
		s[i] = s[i]+1
		break
	}
	if i < 0 {
		return key, errors.New(`key is all 'ff'`)
	}
	return string(s), nil
}