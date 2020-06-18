package message

import (
	. "core"
	. "def"
	"reflect"
)

type MsgServerRegister struct {
	Host        NAME_STRING "IP地址"
	Account     NAME_STRING "服务器所在的目录名字"
	UserID      ObjectID    "MsgConnection返回的ID"
	SVN         [1024]byte  "SVN地址"
	Member      [1024]byte  "通知的用户列表"
	ProjectName NAME_STRING "项目名称"
	IsServer    bool        "是编译服务器还是用户"
}

func registerNetMsgRegisterServer() {
	isSuccess := RegisterMsgFunc(CMD_REGISTER_SERVER, createNetMsgRegisterServer)
	LogInfo("Registor message", CMD_REGISTER_SERVER)
	if !isSuccess {
		LogError("Registor CMD_BUILD faild")
	}
}

func createNetMsgRegisterServer(cmdData *Command) NetMsg {
	netMsg := &MsgServerRegister{}
	netMsg.CreateByBytes(cmdData.Message.([]byte))
	return netMsg
}

func (this *MsgServerRegister) GetNetBytes() ([]byte, bool) {
	return GenNetBytes(uint16(CMD_REGISTER_SERVER), reflect.ValueOf(this))
}

func (this *MsgServerRegister) CreateByBytes(bytes []byte) (bool, int) {
	return Byte2Struct(reflect.ValueOf(this), bytes)
}

func (this *MsgServerRegister) Process(p interface{}) {

}
