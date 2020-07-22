package message

import (
	. "core"
	"game"
	"reflect"
	"time"
)

func (this *MsgBuild) Process(p interface{}) {
	Sender, ok := p.(*TCPSender)
	if !ok {
		return
	}

	switch this.Action {
	case QUERY:
		this.query(Sender)
	case BUILD:
		this.build(Sender)
	}
}

func (this *MsgBuild) query(Sender *TCPSender) {

	index := 0
	for i := 0; i < int(this.Num); i++ {
		project := &Project{}
		_, index = Byte2Struct(reflect.ValueOf(project), this.PData[index:])
		LogDebug(project.ID, Byte2String(project.Name[:]), Byte2String(project.ProjectName[:]),
			Byte2String(project.Host[:]), Byte2String(project.Member[:]), project.ServerState)
	}
	this.Action = BUILD
	this.IsBuildApp = true
	this.IsPatch = true
	this.IsRelease = false
	CopyArray(reflect.ValueOf(&this.Cookflavor), []byte("ETC2"))
	CopyArray(reflect.ValueOf(&this.TargetPlatform), []byte("Android"))
	Sender.Send(this)
}

func (this *MsgBuild) build(Sender *TCPSender) {
	LogDebug("开始编译...")
	project := &Project{}
	Byte2Struct(reflect.ValueOf(project), this.PData)

	var gameUpdater *game.GameUpdater = &game.GameUpdater{}

	//0. 读取配置文件
	err := gameUpdater.ReadConfig()
	if err != nil {
		LogError("Build Failed time:", time.Now())
		return
	}

	config := gameUpdater.GetConfig()

	config.IsPatch = this.IsPatch
	config.IsRelease = this.IsRelease
	config.IsApp = this.IsBuildApp
	config.ProjectName = Byte2String(project.ProjectName[:])

	//LogDebug("set svn path:", Byte2String(project.SVN[:]))
	config.SetSVNCode(Byte2String(project.SVN[:]))
	config.SetMembers(Byte2String(project.Member[:]))
	config.SetTargetPlatform(Byte2String(this.TargetPlatform[:]))
	config.SetCookflavor(Byte2String(this.Cookflavor[:]))
	config.BuildPath()

	msgBuildInfo := &MsgBuildInfo{}

	CopyArray(reflect.ValueOf(&msgBuildInfo.Host), project.Host[:])
	CopyArray(reflect.ValueOf(&msgBuildInfo.Name), project.Name[:])
	CopyArray(reflect.ValueOf(&msgBuildInfo.ProjectName), project.ProjectName[:])
	msgBuildInfo.ID = project.ID
	msgBuildInfo.ServerState = 2

	Sender.Send(msgBuildInfo)

	gameUpdater.ProjectID = project.ID
	gameUpdater.DoUpdate()

	msgBuildInfo.ServerState = 1
	Sender.Send(msgBuildInfo)
}

func (this *MsgBuildInfo) Process(p interface{}) {

}
