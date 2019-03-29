// +build linux,cgo android,cgo darwin,cgo

package helper

import (
	"fmt"
	"github.com/wseternal/helper/logger"
	"os/user"
	"strconv"
	"syscall"
)

func SetProcessFileLimit(soft, max uint64) error {
	err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &syscall.Rlimit{Cur: soft, Max: max})
	if err != nil {
		err = fmt.Errorf("set rlimit for NOFILE failed: %s", err)
		return err
	}
	return nil
}

func SetProcessUser(userName string) error {
	u, err := user.Lookup(userName)
	if err != nil {
		return err
	}
	logger.LogD("set process user: find user %s: %v\n", userName, u)
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return fmt.Errorf("invalid uid %s found for user %s", u.Uid, userName)
	}
	if syscall.Getuid() == uid {
		return nil
	}
	logger.LogD("find uid %d for user %s\n", uid, userName)
	if err = syscall.Setuid(uid); err != nil {
		l.Printf("change uid to %s(%s) failed: %s, trying use CGO setuid\n", userName, u.Uid, err)
		if err = Setuid((uid)); err != nil {
			return fmt.Errorf("CGO setuid %d failed, erro: %s, quit...", uid, err)
		}
	}
	logger.LogI("set process user to %s successfully", userName)
	return nil
}
