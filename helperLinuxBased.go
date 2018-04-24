// +build linux android

package helper

import (
	"bytes"
	"fmt"
	"regexp"
)

func IP2Mac(ip string, withColon bool) (string, error) {
	if len(ip) < 7 {
		return "", fmt.Errorf("invalid IP address: %s", ip)
	}
	out, err := ShellCommand("cat /proc/net/arp |awk '/%s / {print $4}'", ip).Output()
	if err != nil {
		return "", fmt.Errorf("resolve IP %s to MAC failed, error: %s", ip, err)
	}
	out = bytes.TrimSpace(out)
	if len(out) < 17 {
		return "", fmt.Errorf("no ARP entry for IP %s", ip)
	}
	if !withColon {
		out = bytes.Replace(out, []byte(":"), nil, -1)
	}
	return string(out), nil
}

// GetDefaultRoute use "ip route list 0/0" to find the default route entry.
func GetDefaultRoute() (nextHop, dev string, err error) {
	var out []byte
	var reRoute *regexp.Regexp
	if out, err = ShellCommand("ip route list 0/0").Output(); err != nil {
		return "", "", err
	}
	if reRoute, err = regexp.Compile(fmt.Sprintf(`default via (\S+) dev (\S+) *.*`)); err != nil {
		return "", "", err
	}
	out = bytes.TrimSpace(out)
	m := reRoute.FindStringSubmatch(string(out))
	if len(m) == 3 {
		return m[1], m[2], nil
	} else {
		return "", "", fmt.Errorf("can not find the default route entry, regexp match result is %v", m)
	}
}
