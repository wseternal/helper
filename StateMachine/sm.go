package StateMachine

import (
	"bitbucket.org/wseternal/helper/logger"
	"sync"
	"time"
)

type MessageType int

const (
	MTUnknown MessageType = iota
	MTSetSMState
	MTAddSMAction
)

type ActionFunc func()

type Action struct {
	ID, Description, FireInState string
	Delay                        time.Duration `json:",string"`
	T                            *time.Timer   `json:"-"`
	F                            ActionFunc    `json:"-"`
	Exit                         chan int      `json:"-"`
	Queued                       time.Time
}

type StateMessage struct {
	State string
	Priv  interface{}
}

type Message struct {
	MsgType MessageType
	Data    interface{}
}

type DefaultStateHandler struct {
}

type StateObject struct {
	Name    string
	Handler StateHandler `json:"-"`
	Priv    interface{}  `json:"-"`
}

type StateHandler interface {
	Enter(obj *SMObject)
	Leave(obj *SMObject)
}

type SMObject struct {
	Name                            string
	StateNext, StateCurr, StatePrev *StateObject

	// State transition (SetState), AddAction and FireAction are all handled in
	// the same stateLoop, to avoid unnecessary multi-thread protection.
	msgCtrl chan *Message `json:"-"`
	actCtrl chan *Action  `json:"-"`

	StateList struct {
		Elems        map[string]*StateObject
		sync.RWMutex `json:"-"`
	}

	ActionList struct {
		Elems        map[string]*Action
		sync.RWMutex `json:"-"`
	}
}

func (act *Action) ActionThread(obj *SMObject) {
	var triggerAction = false
	select {
	case _, ok := <-act.T.C:
		if ok {
			triggerAction = true
		}
	case <-act.Exit:
		break
	}
	obj.ActionList.Lock()
	delete(obj.ActionList.Elems, act.ID)
	obj.ActionList.Unlock()

	if triggerAction {
		logger.LogD("FireAction: %s, delay: %s, description: %s", act.ID, act.Delay.String(), act.Description)
		obj.actCtrl <- act
	}
}

func (act *Action) Stop() {
	act.T.Stop()
	act.Exit <- 0
	logger.LogD("action %s is stopped\n", act.ID)
}

func (h *DefaultStateHandler) Enter(obj *SMObject) {
	var prevStateStr string
	if obj.StatePrev != nil {
		prevStateStr = obj.StatePrev.Name
	} else {
		prevStateStr = "__NA__"
	}
	logger.LogD("%s: end-state-transition %s ==> %s\n", obj.Name, prevStateStr, obj.StateCurr.Name)
}

func (h *DefaultStateHandler) Leave(obj *SMObject) {
	logger.LogD("%s: pre-state-transition %s ==> %s \n", obj.Name, obj.StateCurr.Name, obj.StateNext.Name)
}

func (obj *SMObject) GetState(name string) *StateObject {
	obj.StateList.RLock()
	defer obj.StateList.RUnlock()
	if v, ok := obj.StateList.Elems[name]; ok {
		return v
	}
	return nil
}

func (obj *SMObject) GetActions() []*Action {
	obj.ActionList.RLock()
	defer obj.ActionList.RUnlock()
	acts := make([]*Action, len(obj.ActionList.Elems))
	idx := 0
	for _, v := range obj.ActionList.Elems {
		acts[idx] = v
		idx++
	}
	return acts
}

func NewStateMachine(name string) *SMObject {
	obj := SMObject{
		Name: name,
	}
	obj.msgCtrl = make(chan *Message, 8)
	obj.actCtrl = make(chan *Action, 8)
	obj.StateList.Elems = make(map[string]*StateObject, 0)
	obj.ActionList.Elems = make(map[string]*Action, 0)
	return &obj
}

func (obj *SMObject) RegisterState(state string, handler StateHandler) {
	obj.StateList.Lock()
	defer obj.StateList.Unlock()
	obj.StateList.Elems[state] = &StateObject{
		Name:    state,
		Handler: handler,
	}
}

func (obj *SMObject) ClearActions() {
	logger.LogD("%s: ClearActions, current state is %s\n", obj.Name, obj.StateCurr.Name)
	obj.ActionList.Lock()
	for _, act := range obj.ActionList.Elems {
		if act.FireInState != "" {
			act.Stop()
		}
	}
	obj.ActionList.Unlock()
}

func (obj *SMObject) onAddActionMessage(msg *Message) {
	var m *Action
	var ok bool

	if m, ok = msg.Data.(*Action); !ok {
		logger.LogE("%s: onAddActionMessage, msg.Data is not type *Action, it's %v, %[1]T\n", obj.Name, msg.Data)
		return
	}
	now := time.Now()
	act := obj.FindAction(m.ID)
	if act != nil {
		if !act.T.Stop() {
			if len(act.T.C) > 0 {
				<-act.T.C
			}
		}
		act.T.Reset(m.Delay)
		act.Queued = now
		act.Description = m.Description
		act.Delay = m.Delay
		act.FireInState = m.FireInState
		act.F = m.F
	} else {
		m.Queued = now
		m.T = time.NewTimer(m.Delay)
		m.Exit = make(chan int)

		obj.ActionList.Lock()
		obj.ActionList.Elems[m.ID] = m
		obj.ActionList.Unlock()

		go m.ActionThread(obj)
	}
}

func (obj *SMObject) onStateMessage(msg *Message) {

	var nextStateObj *StateObject

	var m *StateMessage
	var ok bool

	if m, ok = msg.Data.(*StateMessage); !ok {
		logger.LogE("%s: onStateMessage, msg.Data is not type *StateMessage, it's %v, %[1]T\n", obj.Name, msg.Data)
		return
	}

	if len(obj.StateList.Elems) == 0 {
		logger.LogW("%s: no states registerd, ignore state: %s\n", obj.Name, m.State)
		return
	}
	logger.LogD("%s: onStateMessage: %#v\n", obj.Name, m)
	nextStateObj = obj.GetState(m.State)
	if obj.StateCurr == nextStateObj {
		logger.LogW("%s: current state is %s already, no state transition will be occurred\n", obj.Name, m.State)
		return
	}

	if nextStateObj != nil {
		nextStateObj.Priv = m.Priv
		if obj.StateCurr != nil {
			obj.StateNext = nextStateObj
			obj.StateCurr.Handler.Leave(obj)
		}
		obj.StateNext = nil
		obj.StateCurr, obj.StatePrev = nextStateObj, obj.StateCurr

		// purge all pending action when state changes
		obj.ClearActions()

		obj.StateCurr.Handler.Enter(obj)

		obj.ActionList.RLock()
		if len(obj.ActionList.Elems) == 0 && len(obj.msgCtrl) == 0 {
			logger.LogD("%s: entered state %s without any pending actions and messages", obj.Name, obj.StateCurr.Name)
		}
		obj.ActionList.RUnlock()
	} else {
		logger.LogE("%s: unsupported state: %s\n", obj.Name, m.State)
	}
}

func (obj *SMObject) stateLoop() {
	for {
		select {
		case msg := <-obj.msgCtrl:
			switch msg.MsgType {
			case MTSetSMState:
				obj.onStateMessage(msg)
			case MTAddSMAction:
				obj.onAddActionMessage(msg)
				break
			}
		case act := <-obj.actCtrl:
			if len(act.FireInState) > 0 {
				if obj.StateCurr == nil {
					logger.LogE("%s: current state object is nil when firing action: %s\n", obj.Name, act.ID)
					break
				}
				if obj.StateCurr.Name != act.FireInState {
					logger.LogW("%s: %s won't be fired in %s, it's required to be fired in %s\n", obj.Name, act.ID, obj.StateCurr.Name, act.FireInState)
					break
				}
			}
			logger.LogD("%s: FireAction: %s, delay: %s, description: %s", obj.Name, act.ID, act.Delay.String(), act.Description)
			act.F()
		}
	}
}

func (obj *SMObject) Start() {
	if len(obj.StateList.Elems) == 0 {
		logger.LogW("%s: no state registered to state machine\n", obj.Name)
	}
	go obj.stateLoop()
}

func (obj *SMObject) SendMessage(msg *Message) {
	obj.msgCtrl <- msg
}

func (obj *SMObject) SetState(state string, priv interface{}) {
	msg := &Message{
		MsgType: MTSetSMState,
	}

	msg.Data = &StateMessage{
		State: state,
		Priv:  priv,
	}
	obj.msgCtrl <- msg
}

func (obj *SMObject) FindAction(id string) *Action {
	obj.ActionList.RLock()
	defer obj.ActionList.RUnlock()
	if act, ok := obj.ActionList.Elems[id]; ok {
		return act
	}
	return nil
}

func (obj *SMObject) AddAction(id string, f ActionFunc, delay time.Duration, desc string, fireInState string) {
	logger.LogD("%s: AddAction: %s, delay: %s, description: %s", obj.Name, id, delay.String(), desc)
	act := &Action{
		ID:          id,
		Description: desc,
		FireInState: fireInState,
		F:           f,
		Delay:       delay,
	}
	if delay > 0 {
		msg := &Message{
			MsgType: MTAddSMAction,
			Data:    act,
		}
		obj.msgCtrl <- msg
	} else {
		obj.actCtrl <- act
	}
}
