package helper

import (
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
			logger.LogI("invoking signal actions (%d registered) for %s\n", len(sigActions[sig].Funcs), s.String())
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
