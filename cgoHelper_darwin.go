package helper

/*
#include <string.h>
#include <unistd.h>
#include <errno.h>
*/
import "C"
import (
	"fmt"
)

func Setuid(uid int) error {
	res := C.setuid(C.uid_t(uid))
	if res != 0 {
		return fmt.Errorf("setuid(%d) failed, ret: %d", uid, res)
	}
	return nil
}
