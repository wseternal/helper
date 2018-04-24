package helper

import (
	"flag"
	"fmt"
	"testing"
)

var fn string

func init() {
	flag.StringVar(&fn, "f", "", "file name to test")
}

func TestIP2Mac(t *testing.T) {
	mac, err := IP2Mac("1.1.1.1", false)
	fmt.Printf("mac %v, err: %v\n", mac, err)
}

func TestGetDefaultRoute(t *testing.T) {
	ip, dev, err := GetDefaultRoute()
	fmt.Printf("ip: %v, dev: %v, err: %v\n", ip, dev, err)
}

func TestFileMode(t *testing.T) {
	fmt.Printf("IsFile: %t\n", IsFile(fn))
	fmt.Printf("IsDir: %t\n", IsDir(fn))
	fmt.Printf("IsSymbolLink: %t\n", IsSymbolLink(fn))
	fmt.Printf("IsUnixSocket: %t\n", IsUnixSocket(fn))
}
