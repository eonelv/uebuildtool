// msgbindserver
package message

import (
	. "def"
	"reflect"

	. "ngcod.com/core"
)

type MsgBindServer struct {
	//"MAC地址"
	Account NAME_STRING
	//"MsgConnection返回的ID"
	UserID ObjectID
}

func registerNetMsgBindServer() {
	isSuccess := RegisterMsgFunc(CMD_BIND_SERVER, createNetMsgBindServer)
	LogInfo("Registor message", CMD_BIND_SERVER)
	if !isSuccess {
		LogError("Registor CMD_BIND_SERVER faild")
	}
}

func createNetMsgBindServer(cmdData *Command) NetMsg {
	netMsg := &MsgBindServer{}
	netMsg.CreateByBytes(cmdData.Message.([]byte))
	return netMsg
}

func (this *MsgBindServer) GetNetBytes() ([]byte, bool) {
	return GenNetBytes(uint16(CMD_BIND_SERVER), reflect.ValueOf(this))
}

func (this *MsgBindServer) CreateByBytes(bytes []byte) (bool, int) {
	return Byte2Struct(reflect.ValueOf(this), bytes)
}

func (this *MsgBindServer) Process(p interface{}) {

}
