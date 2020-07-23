// msg_Build
package message

import (
	. "core"
	. "def"
	"reflect"
)

func registerNetMsgBuild() {
	isSuccess := RegisterMsgFunc(CMD_BUILD, createNetMsgBuild)
	LogInfo("Registor message", CMD_BUILD)
	if !isSuccess {
		LogError("Registor CMD_BUILD faild")
	}

	isSuccess = RegisterMsgFunc(CMD_BUILD_INFO, createNetMsgBuildInfo)
	LogInfo("Registor message", CMD_BUILD_INFO)
	if !isSuccess {
		LogError("Registor CMD_BUILD_INFO faild")
	}
}

func createNetMsgBuild(cmdData *Command) NetMsg {
	netMsg := &MsgBuild{}
	netMsg.CreateByBytes(cmdData.Message.([]byte))
	return netMsg
}

func createNetMsgBuildInfo(cmdData *Command) NetMsg {
	netMsg := &MsgBuildInfo{}
	netMsg.CreateByBytes(cmdData.Message.([]byte))
	return netMsg
}

const (
	QUERY uint16 = 1
	BUILD uint16 = 2
)

const (
	ServerStateNone     int32 = 0
	ServerStateIdle     int32 = 1
	ServerStateBuilding int32 = 2
)

type MsgBuild struct {
	UserID ObjectID
	Action uint16
	//"isPath"是tag, 在反射里会用到. 如果需要处理反射，可以添加tag
	IsPatch bool "isPath"
	//"是否编译App"
	IsBuildApp bool
	//"测试版，发布版"
	IsRelease bool
	//"Android才有(ETC2...)"
	Cookflavor [64]byte
	//"Android or iOS"
	TargetPlatform [64]byte
	//"总的Project数量"
	Num uint16
	//"所有Project"
	PData []byte
}

type Project struct {
	ID          ObjectID
	Name        [255]byte
	ProjectName [255]byte
	Host        [255]byte
	Account     [255]byte
	Member      [255]byte
	SVN         [255]byte
	ServerState int32
}

type MsgBuildInfo struct {
	ID          ObjectID
	UserID      ObjectID
	Name        [255]byte
	ProjectName [255]byte
	Host        [255]byte
	ServerState int32
}

func (this *MsgBuild) GetNetBytes() ([]byte, bool) {
	return GenNetBytes(uint16(CMD_BUILD), reflect.ValueOf(this))
}

func (this *MsgBuild) CreateByBytes(bytes []byte) (bool, int) {
	return Byte2Struct(reflect.ValueOf(this), bytes)
}

func (this *MsgBuildInfo) GetNetBytes() ([]byte, bool) {
	return GenNetBytes(uint16(CMD_BUILD_INFO), reflect.ValueOf(this))
}

func (this *MsgBuildInfo) CreateByBytes(bytes []byte) (bool, int) {
	return Byte2Struct(reflect.ValueOf(this), bytes)
}
