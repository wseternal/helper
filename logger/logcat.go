// +build android,cgo

package logger

/*
#cgo LDFLAGS: -llog
#include <android/log.h>
#include <stdlib.h>
*/
import "C"

import (
	"log"
	"unsafe"
)

type LogcatPriority string

type LogcatLogger struct {
	priority AndroidLogPriority
	tag      *C.char
}

type AndroidLogPriority int

const (
	AndroidLogUnknown AndroidLogPriority = iota
	AndroidLogDefault
	AndroidLogVerbose
	AndroidLogDebug
	AndroidLogInfo
	AndroidLogWarn
	AndroidLogError
	AndroidLogFatal
	AndroidLogSilent
)

func (l *LogcatLogger) Write(p []byte) (n int, err error) {
	cstr := C.CString(string(p))
	defer C.free(unsafe.Pointer(cstr))
	ret, err := C.__android_log_write(C.int(l.priority), l.tag, cstr)
	if err != nil {
		return int(ret), err
	}
	return len(p), nil
}

func NewLogcatLogger(priority AndroidLogPriority, tag string) *log.Logger {
	l := &LogcatLogger{
		priority: priority,
		tag:      C.CString(tag),
	}
	return log.New(l, "", 0)
}
