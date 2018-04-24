// +build linux darwin
// +build cgo
// +build !android

package sys

/*
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"unsafe"
)

const (
	avgSample = 3
)

func GetLoadAVG() (avg []float32, err error) {
	tmp := (*C.double)(C.malloc(C.sizeof_double * avgSample))
	ret := C.getloadavg(tmp, avgSample)
	if ret != avgSample {
		return nil, fmt.Errorf("getloadavg return: %d\n", ret)
	}
	p := (*[avgSample]C.double)(unsafe.Pointer(tmp))
	avg = make([]float32, avgSample)
	for i := 0; i < avgSample; i++ {
		avg[i] = float32(p[i])
	}
	defer C.free(unsafe.Pointer(tmp))
	return avg, nil
}
