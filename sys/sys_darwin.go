package sys

/*
#include <time.h>
#include <sys/sysctl.h>
*/
import "C"

import (
	"fmt"
	"net"
	"strings"
	"time"
	"unsafe"
)

func GetUptime() (int64, error) {
	var ts C.struct_timeval
	var len C.size_t = C.sizeof_struct_timeval
	ctlName := make([]C.int, 2)
	ctlName[0] = C.CTL_KERN
	ctlName[1] = C.KERN_BOOTTIME
	ret := C.sysctl((*C.int)(unsafe.Pointer(&ctlName[0])), 2, unsafe.Pointer(&ts), &len, nil, 0)
	if ret != 0 {
		return 0, fmt.Errorf("sysctl return: %d\n", ret)
	}
	return time.Now().Unix() - int64(ts.tv_sec), nil
}

func GetMachineID() string {
	intfs, err := net.Interfaces()
	if err != nil {
		return strings.Replace(AllZeroMAC, ":", "", -1)
	}
	for _, intf := range intfs {
		if (intf.Flags & (net.FlagLoopback | net.FlagPointToPoint)) != 0 {
			continue
		}
		hwAddr := intf.HardwareAddr.String()
		if len(hwAddr) >= 17 && intf.MTU >= 1500 {
			return strings.Replace(hwAddr, ":", "", -1)
		}
	}
	return strings.Replace(AllZeroMAC, ":", "", -1)
}

func GetAllNetStatistics(operstate string) []*NetStatistics {
	return nil
}
