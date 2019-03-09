package helper

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"reflect"
	"runtime/debug"
	"strconv"
	"sync"
	"syscall"
	"time"

	"path/filepath"

	"bitbucket.org/wseternal/helper/logger"
)

type SigAction struct {
	Funcs []func()
}

var (
	l             *log.Logger
	muxSigActions sync.Mutex
	sigActions    map[syscall.Signal]*SigAction
	sigChan       chan os.Signal
)

// 2006-01-02 15:04:05"
const (
	YYYY     = "2006"
	MM       = "01"
	DD       = "02"
	H        = "03"
	M        = "04"
	S        = "05"
	HH_M_S   = "15:04:05"
	H_M_S_PM = "03:04:05 PM"
)

func init() {
	l = log.New(os.Stderr, "", log.Ldate|log.Ltime)
}

func Recover(msg string, stackTrace, panic bool) {
	if except := recover(); except != nil {
		if stackTrace {
			debug.PrintStack()
		}
		str := fmt.Sprintf("recover: %s found exception: %s", msg, except)
		if panic {
			l.Panic(str)
		} else {
			l.Print(str)
		}
	}
}

// ShellCommand run specific command using sh -c, meanwhile, set the Setpgid flag
// by default, pgid is 0, which make the created process as a new process group,
// set the Cmd.SysProcAttr.Pgid manually if it not as expected.
func ShellCommand(format string, arg ...interface{}) *exec.Cmd {
	var cmd string
	if len(arg) > 0 {
		cmd = fmt.Sprintf(format, arg...)
	} else {
		cmd = format
	}
	logger.LogD("ShellCommand: %s\n", cmd)
	c := exec.Command("sh", "-c", cmd)
	c.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	return c
}

func IsDir(path string) bool {
	ret, _ := testFileMode(path, os.ModeDir, true)
	return ret
}

func IsFile(path string) bool {
	ret, err := testFileMode(path, os.ModeType, true)
	if err != nil {
		ret = false
	} else {
		ret = !ret
	}
	return ret
}

func FileSize(path string) (int64, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	if stat.IsDir() {
		return 0, fmt.Errorf("%s is a directory", path)
	}
	return stat.Size(), nil
}

func testFileMode(path string, m os.FileMode, followSymlink bool) (bool, error) {
	var finfo os.FileInfo
	var err error
	if followSymlink {
		finfo, err = os.Stat(path)
	} else {
		finfo, err = os.Lstat(path)
	}
	if err != nil {
		return false, err
	}
	return (finfo.Mode() & m) != 0, nil
}

func IsUnixSocket(path string) bool {
	ret, _ := testFileMode(path, os.ModeSocket, true)
	return ret
}

func IsSymbolLink(path string) bool {
	ret, _ := testFileMode(path, os.ModeSymlink, false)
	return ret
}

func AtoiDef(s string, base int, def int64) int64 {
	if val, err := strconv.ParseInt(s, base, 64); err != nil {
		return def
	} else {
		return val
	}
}

func AtouDef(s string, base int, def uint64) uint64 {
	if val, err := strconv.ParseUint(s, base, 64); err != nil {
		return def
	} else {
		return val
	}
}

func signalCaptureThread() {
	for {
		select {
		case s := <-sigChan:
			sig, _ := s.(syscall.Signal)
			logger.LogI("invoking actions for signal %d (%s) for %d subscribers\n", sig, s.String(), len(sigActions[sig].Funcs))
			for _, f := range sigActions[sig].Funcs {
				f()
			}
		}
	}
}

func OnSignal(f func(), sigs ...syscall.Signal) {
	muxSigActions.Lock()
	defer muxSigActions.Unlock()
	if sigActions == nil {
		sigActions = make(map[syscall.Signal]*SigAction)
		sigChan = make(chan os.Signal, 1)
		go signalCaptureThread()
	}
	for _, sig := range sigs {
		_, found := sigActions[sig]
		if !found {
			sigActions[sig] = &SigAction{
				Funcs: make([]func(), 0),
			}
			signal.Notify(sigChan, os.Signal(sig))
		}
		sigActions[sig].Funcs = append(sigActions[sig].Funcs, f)
	}
}

func SameSliceBackend(a, b interface{}) bool {
	if a == nil || b == nil {
		return false
	}
	v1 := reflect.ValueOf(a)
	v2 := reflect.ValueOf(b)
	if v1.Type() != v2.Type() {
		return false
	}
	if v1.Kind() != reflect.Slice || v2.Kind() != reflect.Slice {
		return false
	}
	if v1.IsNil() || v2.IsNil() {
		return false
	}
	elemSize := int(v1.Type().Elem().Size())
	p1 := int(v1.Pointer()) + v1.Cap()*elemSize
	p2 := int(v2.Pointer()) + v2.Cap()*elemSize
	return p1 == p2
}

func GetDirectorySize(dir string) (size int64, err error) {
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		size += info.Size()
		return nil
	})
	return size, err
}

// UnixDate return unix timestamp of 00:00:00 day with given offset
// 0: 00:00:00 of today, -1 00:00:00 of yesterday,  1: 00:00:00 tomorrow
func UnixDate(offDay int) int64 {
	now := time.Now()
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, now.Location()).Unix() + int64(offDay * 86400)
}

// zoneOff: timezone offset, postive for east, negative for west, 0 for UTC
// e.g.: +8 for CST
func UnixDateWithZone(offDay int, zoneOff int8) int64 {
	_, off := time.Now().Zone()
	return UnixDate(offDay) + int64(zoneOff)*3600 - int64(off)
}

// valid whether obj is in given struct type, dereference: the dereference depth
// if t is a pointer, if < 0: infinite dereference; if >= 0: only deference given times
// if expected is nil, only check whether obj is a struct
func ValidStructType(obj interface{}, expected reflect.Type, dereference int) error {
	if obj == nil {
		return errors.New("nil object")
	}
	if expected != nil && expected.Kind() != reflect.Struct {
		return errors.New("parameter 'expected' must be a valid struct type")
	}
	tmp := reflect.TypeOf(obj)
	count := dereference
	for {
		switch(tmp.Kind()) {
		case reflect.Ptr:
			if count == 0 {
				return fmt.Errorf("%v(%[1]T) is not a valid struct type after dereference given %d times", obj, dereference)
			}
			if count > 0 {
				count--
			}
			tmp = tmp.Elem()
		case reflect.Struct:
			if expected == nil {
				return nil
			} else {
				if expected == tmp {
					return nil
				}
				return fmt.Errorf("obj is in type: %s, not equal to expected: %s", tmp, expected)
			}
		default:
			return fmt.Errorf("%v is not a valid struct type, it's underline type is: %s", obj, tmp.Kind())
		}
	}
}

