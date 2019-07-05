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

	"github.com/wseternal/helper/logger"
)

type SigAction struct {
	Funcs []func()
}

var (
	l             *log.Logger
	muxSigActions sync.Mutex
	sigActions    map[syscall.Signal]*SigAction
	sigChan       chan os.Signal

	LocationCST = time.FixedZone("ChinaCST", 8*3600)
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
		return -1, err
	}
	if stat.IsDir() {
		return -2, fmt.Errorf("%s is a directory", path)
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

// kill terminate signal to current process/group, wait given time ,
// if process is not quit, send kill signal
func Terminate(t time.Duration) {
	// reset SIGTERM handler to default
	signal.Reset(syscall.SIGTERM)
	signal.Reset(syscall.SIGINT)

	var err error
	pid := syscall.Getpid()
	// send TERM to process group and current process to exit
	if err = syscall.Kill(-pid, syscall.SIGTERM); err != nil {
		fmt.Fprintf(os.Stderr, "send SIGTERM to -%d failed, %s\n", pid, err)
	}
	if err = syscall.Kill(pid, syscall.SIGTERM); err != nil {
		fmt.Fprintf(os.Stderr, "send SIGTERM to %d failed, %s\n", pid, err)
	}

	time.Sleep(t)
	fmt.Fprintf(os.Stderr, "wait too long (%s) for application (%d) exit, kill it", t.String(), pid)

	if err = syscall.Kill(-pid, syscall.SIGKILL); err != nil {
		fmt.Fprintf(os.Stderr, "send SIGKILL to -%d failed, %s\n", pid, err)
	}
	if err = syscall.Kill(pid, syscall.SIGKILL); err != nil {
		fmt.Fprintf(os.Stderr, "send SIGKILL to %d failed, %s\n", pid, err)
	}
	log.Panicf("OOps, send SIGTERM/SIGKILL to %d failed...\n", pid)
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

// UnixDate return unix timestamp of 00:00:00 (in CST) respect to offDay
// offDay 0: 00:00:00 of today, -1 00:00:00 of yesterday,  1: 00:00:00 tomorrow
func UnixDate(offDay int) int64 {
	return UnixDateLocation(offDay, LocationCST)
}

// return unix timestamp of 00:00:00 today
// if location is nil, use time.Local
// use time.FixedZone(name, offset) to generate customized location.
func UnixTodayZero(tsUtcNow int64, location *time.Location) int64 {
	utcTime := time.Unix(tsUtcNow, 0)
	if location == nil {
		location = time.Local
	}
	t := utcTime.In(location)
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location()).Unix()
}

// return unix timestamp of 00:00:00 against offDay
// offDay: 0 (today), +1 (tomorrow), -1 (yesterday), etc
// use time.FixedZone(name, offset) to generate customized location.
func UnixDateLocation(offDay int, location *time.Location) int64 {
	todayZero := UnixTodayZero(time.Now().Unix(), location)
	return todayZero + int64(offDay)*86400
}

// check whether local timezone is as expected as given value, e.g.: for china, +8
func IsTimeZone(zone int) error {
	z, off := time.Now().Zone()
	if off == 3600*zone {
		return nil
	}
	return fmt.Errorf("local timezone %s (offset: %d) is not in the expected time zone: %+d", z, off, zone)
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
		switch tmp.Kind() {
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
