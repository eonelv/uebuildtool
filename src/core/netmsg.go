package core

import (
	"errors"
	"fmt"
	"reflect"
	"time"
)

type CreateNetMsgFunc func(cmdData *Command) NetMsg

type NetMsg interface {
	GetNetBytes() ([]byte, bool)
	CreateByBytes(bytes []byte) (bool, int)
	Process(p interface{})
}

func GenNetBytes(cmd uint16, values reflect.Value) ([]byte, bool) {
	datas, ok := Struct2Bytes(values)
	if !ok {
		return nil, false
	}
	length := uint16(HEADER_LENGTH + uint16(len(datas)))
	header := &PackHeader{TAG, VERSION, length, cmd}
	headerDatas, okh := Struct2Bytes(reflect.ValueOf(header))
	if !okh {
		return nil, false
	}
	return append(headerDatas, datas...), true
}

func SendCommand(ch chan *Command, cmd *Command, timeoutSec time.Duration) error {
	if ch == nil {
		return errors.New("SendCommand nil chan")
	}
	if cmd == nil {
		return errors.New("SendCommand nil cmd")
	}

	defer func() {
		if x := recover(); x != nil {
			LogError("SendCommand panic :", x, "cmd:", cmd.Cmd)
		}
	}()

	select {
	case ch <- cmd:
	case <-time.After(timeoutSec * time.Second):
		return errors.New(fmt.Sprintf("SendCommand. push chan time out.cmd:%d", cmd.Cmd))
	}

	return nil
}
