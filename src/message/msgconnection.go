package message

import (
	. "cfg"
	. "def"
	"reflect"
	"strings"

	. "ngcod.com/core"
	"ngcod.com/utils"
)

type MsgConnection struct {
	ID        ObjectID
	AccountID NAME_STRING
}

func registerNetMsgConnection() {
	isSuccess := RegisterMsgFunc(CMD_CONNECTION, createNetMsgConnection)
	LogInfo("Registor message", CMD_CONNECTION)
	if !isSuccess {
		LogError("Registor CMD_BUILD faild")
	}
}

func createNetMsgConnection(cmdData *Command) NetMsg {
	netMsg := &MsgConnection{}
	netMsg.CreateByBytes(cmdData.Message.([]byte))
	return netMsg
}

func (this *MsgConnection) GetNetBytes() ([]byte, bool) {
	return GenNetBytes(uint16(CMD_CONNECTION), reflect.ValueOf(this))
}

func (this *MsgConnection) CreateByBytes(bytes []byte) (bool, int) {
	return Byte2Struct(reflect.ValueOf(this), bytes)
}

func (this *MsgConnection) Process(p interface{}) {
	ip, err := utils.GetLocalIP()
	if err != nil {
		return
	}

	Sender := p.(*TCPSender)

	const isServer bool = true
	if isServer {
		msgRegisterServer := &MsgServerRegister{}

		var config *Config = &Config{}
		config.ReadConfig()

		LogInfo("config:", config.BuilderHome, config.GetSVNCode(), config.GetMembers())

		parentFile := strings.ReplaceAll(config.BuilderHome, `\`, `/`)
		index := strings.LastIndex(parentFile, `/`)
		if index != -1 {
			parentFile = Byte2String([]byte(parentFile)[index+1:])
		}
		msgRegisterServer.UserID = this.ID
		CopyArray(reflect.ValueOf(&msgRegisterServer.Host), []byte(ip))
		CopyArray(reflect.ValueOf(&msgRegisterServer.Account), []byte(parentFile))
		msgRegisterServer.IsServer = true
		CopyArray(reflect.ValueOf(&msgRegisterServer.Member), []byte(config.GetMembers()))
		CopyArray(reflect.ValueOf(&msgRegisterServer.ProjectName), []byte(config.ProjectName))
		CopyArray(reflect.ValueOf(&msgRegisterServer.SVN), []byte(config.GetSVNCode()))

		Sender.Send(msgRegisterServer)
	} else {
		msgLogin := &MsgLogin{}
		msgLogin.UserID = this.ID
		CopyArray(reflect.ValueOf(&msgLogin.Account), []byte(ip))
		msgLogin.IsServer = false
		Sender.Send(msgLogin)
	}
}
