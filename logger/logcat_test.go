package logger

import "testing"

func TestLogcat(t *testing.T) {
	SetLogger(INFO, NewLogcatLogger(AndroidLogInfo, "test"))
	LogI("test android logcat output")
}
