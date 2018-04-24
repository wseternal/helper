// +build cgo

package sys

/*
#include <stdlib.h>
#include <string.h>
#include <sys/time.h>

void assign_timeval(struct timeval *val, long sec, long usec)
{
	val->tv_sec = (time_t)sec;
	val->tv_usec = (time_t)usec;
}
*/
import "C"

func SetTimeOfDay(sec int64) error {
	tv := &C.struct_timeval{}
	// assing_timeval helper function is used to initialize the timeval struct.
	// as in glibc, the filed tv_sec is in type __time_t, however, in musl,
	// the type is time_t. use a wrap function to avoid the type mismatch error.
	C.assign_timeval(tv, C.long(sec), 0)
	ret, err := C.settimeofday(tv, nil)
	if ret != 0 {
		return err
	}
	return nil
}
