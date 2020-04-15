package core

import "sync"

var netMsgs map[uint16]CreateNetMsgFunc

var mutex1 sync.RWMutex

func initRegisterMsgFunc() {
	netMsgs = make(map[uint16]CreateNetMsgFunc)
	mutex1 = sync.RWMutex{}
}

func RegisterMsgFunc(cmd uint16, msgFunc CreateNetMsgFunc) bool {
	mutex1.Lock()
	defer mutex1.Unlock()
	_, ok := netMsgs[cmd]
	 if ok {
		 return false
	 }
	netMsgs[cmd] = msgFunc
	return true
}

func CreateNetMsg(msg *Command) NetMsg {
	createFunc, _ := netMsgs[msg.Cmd]

	if createFunc == nil {
		LogError("UnKnown Command", msg.Cmd)
		return nil
	}
	return createFunc(msg)
}
