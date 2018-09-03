package logger

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"runtime"
)

// Filter process every log message, return true if continue the logging, otherwise,
// return false to drop the message
type Filter func(level LogLevel, format string, v ...interface{}) bool

type LogLevel int

const (
	EMERGENCY LogLevel = iota
	ALERT
	CRITICAL
	ERROR
	WARNING
	NOTICE
	INFO
	DEBUG
)

const (
	LoggerNum = int(DEBUG) + 1
)

var (
	LogLevelString = []string{
		"Emergency",
		"Alert",
		"Critical",
		"Error",
		"Warning",
		"Notice",
		"Info",
		"Debug",
	}
	logLevel      LogLevel = INFO
	discardLogger          = log.New(ioutil.Discard, "Discard: ", 0)
	loggers       []*log.Logger
	filters       map[string]Filter
)

func init() {
	loggers = make([]*log.Logger, LoggerNum)
	for i := 0; i < LoggerNum; i++ {
		if i > int(ERROR) {
			loggers[i] = log.New(os.Stdout, fmt.Sprintf("%s: ", LogLevelString[i]), 0)
		} else {
			loggers[i] = log.New(os.Stderr, fmt.Sprintf("%s: ", LogLevelString[i]), 0)
		}
	}
	filters = make(map[string]Filter, 0)
}

func RegisterFilter(name string, f Filter) {
	filters[name] = f
}

func Log(level LogLevel, format string, v ...interface{}) {
	if len(filters) > 0 {
		for _, f := range filters {
			if !f(level, format, v...) {
				return
			}
		}
	}
	if level <= logLevel {
		if level <= ERROR {
			log := loggers[level]
			_, f, l, ok := runtime.Caller(2)
			if ok {
				str := fmt.Sprintf("<%s:%d> %s", f, l, fmt.Sprintf(format, v...))
				log.Printf("%s", str)
				return
			}
		}
		loggers[level].Printf(format, v...)
	} else {
		discardLogger.Printf(format, v...)
	}
}

func Write(w io.Writer, format string, v ...interface{}) {
	io.WriteString(w, fmt.Sprintf(format, v...))
}

// ERROR is always outputed
func LogE(format string, v ...interface{}) {
	Log(ERROR, format, v...)
}

func LogD(format string, v ...interface{}) {
	if logLevel < DEBUG {
		return
	}
	Log(DEBUG, format, v...)
}

func LogI(format string, v ...interface{}) {
	if logLevel < INFO {
		return
	}
	Log(INFO, format, v...)
}

func LogW(format string, v ...interface{}) {
	if logLevel < WARNING {
		return
	}
	Log(WARNING, format, v...)
}

// Caller skip 0: the func calling this helper function
// 1: parent of the func calling this helper function, etc...
func Caller(skip int) string {
	if _, f, l, ok := runtime.Caller(skip); ok {
		return fmt.Sprintf("%s:%d", f, l)
	} else {
		return ""
	}
}

func Panicf(format string, v ...interface{}) {
	_, f, l, ok := runtime.Caller(1)
	if ok {
		str := fmt.Sprintf("<%s:%d> %s", f, l, fmt.Sprintf(format, v...))
		log.Panicf("%s", str)
	} else {
		log.Panicf(format, v...)
	}
}

func SetLevel(level LogLevel) {
	if logLevel != level {
		logLevel = level
	}
	LogI("set log level to %v\n", level)
}

func SetFlagForAll(flag int) {
	for _, l := range loggers {
		l.SetFlags(flag)
	}
}

func GetLogger(level LogLevel) *log.Logger {
	if int(level) > LoggerNum {
		return nil
	}
	return loggers[level]
}

func SetLogger(level LogLevel, l *log.Logger) {
	if int(level) >= LoggerNum {
		return
	}
	loggers[level] = l
}

func Compact(obj interface{}, maxLen int) string {
	msg := fmt.Sprintf("%+v", obj)
	return CompactString(msg, maxLen)
}

func CompactString(msg string, maxLen int) string {
	if maxLen <= 0 {
		return msg
	}
	l := len(msg)
	if l <= maxLen || l < 5 {
		// at least, we need string lenght >= 5, so that it can
		// show 1st letter + "..." + last letter
		return msg
	}
	cnt := (maxLen - 3) / 2
	return fmt.Sprintf("%s...%s", msg[0:cnt], msg[l-cnt:l])
}
