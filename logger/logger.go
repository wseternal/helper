package logger

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"runtime"
)

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
}

func Log(level LogLevel, format string, v ...interface{}) {
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
