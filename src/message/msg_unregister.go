// msg_unregister.go
package message

import (
	"cfg"
	. "def"
	"fmt"
	"os"
	"reflect"

	. "ngcod.com/core"
)

type MsgUnRegister struct {
	ID     ObjectID
	UserID ObjectID
	State  byte
}

func registerNetMsgUnRegister() {
	isSuccess := RegisterMsgFunc(CMD_UNREGISTER_SERSVER, createNetMsgUnRegister)
	LogInfo("Registor message", CMD_UNREGISTER_SERSVER)
	if !isSuccess {
		LogError("Registor CMD_UNREGISTER_SERSVER faild")
	}
}

func createNetMsgUnRegister(cmdData *Command) NetMsg {
	netMsg := &MsgUnRegister{}
	netMsg.CreateByBytes(cmdData.Message.([]byte))
	return netMsg
}

func (this *MsgUnRegister) GetNetBytes() ([]byte, bool) {
	return GenNetBytes(uint16(CMD_UNREGISTER_SERSVER), reflect.ValueOf(this))
}

func (this *MsgUnRegister) CreateByBytes(bytes []byte) (bool, int) {
	return Byte2Struct(reflect.ValueOf(this), bytes)
}

func (this *MsgUnRegister) Process(p interface{}) {
	LogDebug("Reveive MsgUnRegister uebuilder", this.State)
	Sender, ok := p.(*TCPSender)
	if !ok {
		return
	}
	if this.State == 1 { //删除所有文件
		config := &cfg.Config{}
		config.ReadConfig()
		config.BuildPath()

		os.RemoveAll(config.TempFileHome)
		os.RemoveAll(config.ProjectHomePath)

		PackHome := fmt.Sprintf("%s/APack_Android", config.BuilderHome)
		os.RemoveAll(PackHome)
		PackHome = fmt.Sprintf("%s/APack_iOS", config.BuilderHome)
		os.RemoveAll(PackHome)
		PackHome = fmt.Sprintf("%s/config", config.BuilderHome)
		os.RemoveAll(PackHome)

		this.State = 2
		Sender.Send(this)
	} else if this.State == 3 {
		os.Exit(100)
	}

}
