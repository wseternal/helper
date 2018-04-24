package helper

import (
	"fmt"
	"helper/logger"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime/debug"
	"strconv"
	"sync"
	"syscall"
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

// ShellCommandn run specific command using sh -c, meanwhile, set the pgid for child process
func ShellCommand(format string, arg ...interface{}) *exec.Cmd {
	var cmd string
	if len(arg) > 0 {
		cmd = fmt.Sprintf(format, arg...)
	} else {
		cmd = format
	}
	logger.LogD("ShellCommand: %s\n", cmd)
	c := exec.Command("sh", "-c", cmd)
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
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
