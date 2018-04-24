package StateMachine

import (
	"fmt"
	"helper/logger"
	"testing"
	"time"
)

func TestStateMachine(t *testing.T) {
	logger.SetLevel(logger.DEBUG)
	obj := NewStateMachine("test")
	obj.RegisterState("connect", &DefaultStateHandler{})
	obj.RegisterState("keepalive", &DefaultStateHandler{})
	obj.Start()
	obj.SetState("connect", "conn priv")
	time.Sleep(time.Second)
	obj.AddAction("act1", func() { fmt.Printf("action 1 triggered after two second\n") }, time.Second*2, "2s", "")
	obj.AddAction("act2", func() { fmt.Printf("action 2 triggered immediately\n") }, 0, "immediately", "")
	obj.AddAction("act3", func() { fmt.Printf("action 3 triggered after five second\n") }, time.Second*5, "5s", "")
	time.Sleep(time.Second * 3)
	act := obj.FindAction("act3")
	if act != nil {
		act.Stop()
	}
	obj.SetState("keepalive", "keealive priv")
	time.Sleep(time.Second * 5)
}
