// +build linux

package helper

import (
	"fmt"
	"testing"
)

func TestIP2Mac(t *testing.T) {
	mac, err := IP2Mac("1.1.1.1", false)
	fmt.Printf("mac %v, err: %v\n", mac, err)
}

func TestGetDefaultRoute(t *testing.T) {
	ip, dev, err := GetDefaultRoute()
	fmt.Printf("ip: %v, dev: %v, err: %v\n", ip, dev, err)
}
