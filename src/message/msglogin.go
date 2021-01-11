// msglogin
package message

import (
	. "def"
	"reflect"

	. "ngcod.com/core"
)

type MsgLogin struct {
	//"MAC地址"
	Account NAME_STRING
	//"MsgConnection返回的ID"
	UserID ObjectID
	//"是编译服务器还是用户"
	IsServer bool
}

func registerNetMsgLogin() {
	isSuccess := RegisterMsgFunc(CMD_LOGIN, createNetMsgLogin)
	LogInfo("Registor message", CMD_LOGIN)
	if !isSuccess {
		LogError("Registor CMD_LOGIN faild")
	}
}

func createNetMsgLogin(cmdData *Command) NetMsg {
	netMsg := &MsgLogin{}
	netMsg.CreateByBytes(cmdData.Message.([]byte))
	return netMsg
}

func (this *MsgLogin) GetNetBytes() ([]byte, bool) {
	return GenNetBytes(uint16(CMD_LOGIN), reflect.ValueOf(this))
}

func (this *MsgLogin) CreateByBytes(bytes []byte) (bool, int) {
	return Byte2Struct(reflect.ValueOf(this), bytes)
}

func (this *MsgLogin) Process(p interface{}) {

	Sender := p.(*TCPSender)

	const isServer bool = true
	if isServer {
		//如果是服务器-注册
		msgBindServer := &MsgBindServer{}
		msgBindServer.UserID = this.UserID
		msgBindServer.Account = this.Account

		Sender.Send(msgBindServer)
	} else {
		//客户端-查询
		msgBuild := &MsgBuild{}
		msgBuild.Action = QUERY
		Sender.Send(msgBuild)
	}
}
