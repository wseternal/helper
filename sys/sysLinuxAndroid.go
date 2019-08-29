// +build linux android

package sys

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/wseternal/helper"
	"github.com/wseternal/helper/iohelper"
	"github.com/wseternal/helper/logger"
)

const (
	SysFSNet         = "/sys/class/net"
	SysFSIEEE80211   = "/sys/class/ieee80211"
	OperStateUp      = "up"
	OperStateDown    = "down"
	OperStateUnknown = "unknown"
)

func GetUptime() (int64, error) {
	info_t := &syscall.Sysinfo_t{}
	err := syscall.Sysinfo(info_t)
	if err != nil {
		return 0, err
	}
	// convert to int64 as in 32bit arch, type of Sysinfo_t.Uptime is int32.
	return int64(info_t.Uptime), nil
}

func GetNetStatistics(dev string) *NetStatistics {
	out, err := helper.ShellCommand("cd %s/%s/statistics && cat rx_bytes rx_packets rx_errors tx_bytes tx_packets tx_errors", SysFSNet, dev).CombinedOutput()
	if err != nil {
		logger.LogW("GetNetStatistics for dev %s failed, out: %v, error: %s\n", dev, string(out), err)
		return nil
	}
	out = bytes.TrimSpace(out)
	fields := strings.Split(string(out), "\n")
	if len(fields) != 6 {
		logger.LogW("GetNetStatistics: invliad parsed result %v for dev %s\n", fields, dev)
		return nil
	}
	return &NetStatistics{
		RxBytes:   helper.AtouDef(fields[0], 10, 0),
		RxPackets: helper.AtouDef(fields[1], 10, 0),
		RxErrors:  helper.AtouDef(fields[2], 10, 0),
		TxBytes:   helper.AtouDef(fields[3], 10, 0),
		TxPackets: helper.AtouDef(fields[4], 10, 0),
		TxErrors:  helper.AtouDef(fields[5], 10, 0),
		Device:    dev,
	}
}

func SysNetDevExist(name string) error {
	if !helper.IsDir(SysFSNet) {
		return fmt.Errorf("%s is not existed", SysFSNet)
	}
	fp := filepath.Join(SysFSNet, name)
	if !helper.IsSymbolLink(fp) {
		return fmt.Errorf("%s is not a symbol link", fp)
	}
	var err error
	var link string
	if link, err = os.Readlink(fp); err != nil {
		return fmt.Errorf("follow sysnet link %s failed, %s", fp, err)
	}
	if !helper.IsDir(filepath.Join(SysFSNet, link)) {
		return fmt.Errorf("%s is not a valid directory", link)
	}
	return nil
}

func GetSysNetDevOperState(dev string) string {
	return iohelper.CatTextFile(filepath.Join(SysFSNet, dev, "operstate"), OperStateUnknown)
}

func GetAllNetStatistics(operstate string) []*NetStatistics {
	entries, err := ioutil.ReadDir(SysFSNet)
	if err != nil {
		logger.LogW("GetAllNetStatistics: list directory %s failed, error: %s\n", SysFSNet, err)
		return nil
	}
	res := make([]*NetStatistics, 0)
	for _, entry := range entries {
		if (entry.Mode() & os.ModeSymlink) == 0 {
			continue
		}
		state := iohelper.CatTextFile(filepath.Join(SysFSNet, entry.Name(), "operstate"), OperStateUnknown)
		if state == operstate {
			if stat := GetNetStatistics(entry.Name()); stat != nil {
				res = append(res, stat)
			}
		}
	}
	return res
}

func GetSysNetAddr(name string, allowVirtual bool) (string, error) {
	var err error
	var addr string
	if !helper.IsDir(SysFSNet) {
		return addr, fmt.Errorf("%s is not existed", SysFSNet)
	}
	fp := filepath.Join(SysFSNet, name)
	if !helper.IsSymbolLink(fp) {
		return addr, fmt.Errorf("%s is not a symbol link", fp)
	}
	var link string
	if link, err = os.Readlink(fp); err != nil {
		return addr, err
	}
	if strings.Contains(link, "virtual") && !allowVirtual {
		return addr, fmt.Errorf("ignore virtual device %s under sysnet\n", name)
	}
	fp = filepath.Join(SysFSNet, name, "address")
	return strings.Replace(iohelper.CatTextFile(fp, AllZeroMAC), ":", "", -1), nil
}

// TODO: add capability set get interface

func GetMachineID() string {
	var fp string
	var elems []os.FileInfo
	var err error

	if !helper.IsDir(SysFSIEEE80211) {
		goto sysnet
	}
	elems, err = ioutil.ReadDir(SysFSIEEE80211)
	if err != nil {
		goto sysnet
	}
	for _, elem := range elems {
		if (elem.Mode() & os.ModeSymlink) == 0 {
			continue
		}
		fp = filepath.Join(SysFSIEEE80211, elem.Name(), "macaddress")
		if !helper.IsFile(fp) {
			continue
		}
		return strings.Replace(iohelper.CatTextFile(fp, AllZeroMAC), ":", "", -1)
	}

sysnet:
	if !helper.IsDir(SysFSNet) {
		goto out
	}
	elems, err = ioutil.ReadDir(SysFSNet)
	if err != nil {
		goto out
	}
	for _, elem := range elems {
		var link string
		if (elem.Mode() & os.ModeSymlink) == 0 {
			continue
		}
		fp = filepath.Join(SysFSNet, elem.Name())
		if link, err = os.Readlink(fp); err != nil {
			goto out
		}
		if strings.Contains(link, "virtual") {
			logger.LogD("GetMachineID: ignore virtual device %s under sysnet\n", elem.Name())
			continue
		}
		fp = filepath.Join(SysFSNet, elem.Name(), "address")
		return strings.Replace(iohelper.CatTextFile(fp, AllZeroMAC), ":", "", -1)
	}
out:
	return strings.Replace(AllZeroMAC, ":", "", -1)
}

func GetFileInfo(fn string) (*syscall.Stat_t, error) {
	info, err := os.Stat(fn)
	if err != nil {
		return nil, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if ok {
		return stat, nil
	}
	return nil, fmt.Errorf("not *syscall.Stat_t, it's %+v %[1]T", info.Sys())
}
