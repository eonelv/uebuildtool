package core

import (
	"sync"
)

var chanDatas map[ObjectID]chan *Command
var mutex sync.RWMutex

func CreateChanMgr() {
	chanDatas = make(map[ObjectID]chan *Command)
	mutex = sync.RWMutex{}
}

func GetChanByID(id ObjectID) chan *Command {
	channel, ok := chanDatas[id]
	if !ok {
		return nil
	}
	return channel
}

func RegisterChan(id ObjectID, ch chan *Command) bool {
	_, ok := chanDatas[id]
	if ok {
		LogError("注册chan失败", id)
		return false
	}
	mutex.Lock()
	defer mutex.Unlock()
	chanDatas[id] = ch
	return true
}

func UnRegisterChan(id ObjectID) bool {
	_, ok := chanDatas[id]
	if !ok {
		return false
	}
	mutex.Lock()
	defer mutex.Unlock()
	delete(chanDatas, id)
	return true
}
