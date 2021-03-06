// Cook之后的文件：
// 1. 经过MD5对比,将需要更新的文件复制到ResOutputContentPath
// 2. 将ResOutputContentPath内的文件打包pak到TempPakPath
// 3. 拆分pak文件到ZipSourcePath
// 4. 复制Lua & Json到ZipSourcePath
// 5. 生成压缩包
package cfg

import (
	. "def"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	. "ngcod.com/core"
	"ngcod.com/utils"
)

var config string = `{
	"svncode":"",
	"projectName":"ENGGame",
	"ue_exe":"C:/eonegame/UnrealEngine/Engine/Binaries/Win64/UE4Editor.exe",
	"unrealBuildTool":"C:/eonegame/UnrealEngine//Engine/Binaries/DotNET/UnrealBuildTool.exe",
	"automationTool":"C:/eonegame/UnrealEngine//Engine/Build/BatchFiles/RunUAT.bat",
	"UnrealPak":"C:/eonegame/UnrealEngine//Engine/Binaries/Win64/UnrealPak.exe",
	"TeamMembers":"liwei",
	"isPatch":1,
	"isDebugTool":0,
	"isEncrypt":0
}`

var BuildAndroid string = `uebuildtool.exe -Release=false -Patch=true -cookflavor=ETC2 -targetPlatform=Android -BuildApp=true`

var BuildAndroidRes string = `uebuildtool.exe -Release=false -Patch=true -cookflavor=ETC2 -targetPlatform=Android`

var BuildIOS string = `uebuildtool.exe -Release=false -Patch=true -targetPlatform=iOS -BuildApp=true`

const svnCore string = `svn://192.168.0.24/client/ue4/ENGCore/Plugins/ENGCore`
const svnUnLua string = `svn://192.168.0.24/client/ue4/UnLua424/UnLua`

type Config struct {
	//配置参数
	//Unreal Editor full path
	UE_EXE string
	//UnrealPak工具路径
	UnrealPak       string
	AutomationTool  string
	UnrealBuildTool string

	//项目代码SVN路径
	svnCode string

	//编译程序运行目录
	BuilderHome string
	//运行过程中用到的临时源文件目录
	TempFileHome string
	//配置文件目录
	ConfigHome         string
	ProjectHomePath    string
	ProjectContentPath string

	//项目文件名称 - xxx.uproject
	ProjectName string

	//项目中json文件目录 - 位于Content
	JsonHome string

	//项目中Lua文件目录 - 位于Content名字为Script
	LuaHome string

	//加密之前的原始资源临时缓存目录，编译完之后复制回项目的content目录
	TempJsonHome string
	TempLuaHome  string

	//Android_ETC2
	CookPlatformType string

	//编译参数
	//是否外网包
	IsPatch bool
	//是否发布版
	IsRelease bool
	//是否编译App
	IsApp bool
	//控制是否调用BuildFirst(现在都需要调用.虽然会让编译时间变长,但可以解决修改了C++父类导致蓝图不能编译的问题)
	IsDebugTool bool
	//是否加密 - 由于demo项目与正式项目代码不同，采用不加密方式
	isEncrypt bool

	//Android / iOS
	targetPlatform string
	//ETC2
	cookflavor string

	//输出目录
	//Cook之后文件的目录
	CookedPath string
	//编译结果目录 - Android和iOS不一样
	OutputPath string

	//Cook之后, 所有需要更新的资源经过MD5对比都放到这里
	//需要打包成pak的资源、json、lua
	ResOutputContentPath string

	//Pak文件输出目录 - 用作更新文件拆分的源目录
	TempPakPath string

	//需要打包的文件目录 - 待更新的json或lua、Pak拆分文件
	ZipSourcePath string

	Today string

	//编译通知成员
	teamMembers string

	//ENGCore插件目录 - 在项目的Plugin下
	PluginCodePath string
	//ENGCore插件SVN更新的临时目录 - 更新完之后复制到项目插件目录进行编译
	TempPluginCodePath string

	//UnLua插件目录 - 在项目的Plugin下
	PluginUnLuaPath string
	//UnLua插件SVN更新的临时目录 - 更新完之后复制到项目插件目录进行编译
	TempPluginUnLuaPath string

	SVNCore  string
	SVNUnLua string

	ProjectEncryptIniPath string
	VersionCppFilePath    string
	SourceGameConfigPath  string
}

func (this *Config) SetMembers(v string) {
	this.teamMembers = v
}

func (this *Config) GetMembers() string {
	return this.teamMembers
}

func (this *Config) SetTargetPlatform(v string) {
	this.targetPlatform = v
	if this.targetPlatform == IOS {
		this.CookPlatformType = IOS
	} else if this.targetPlatform == Win64 {
		this.CookPlatformType = WinNoEditor
	} else {
		this.CookPlatformType = fmt.Sprintf("%s_%s", this.targetPlatform, this.cookflavor)
	}
}

func (this *Config) GetTargetPlatform() string {
	return this.targetPlatform
}

func (this *Config) SetCookflavor(v string) {
	this.cookflavor = v
	if this.targetPlatform == IOS {
		this.CookPlatformType = IOS
	} else if this.targetPlatform == Win64 {
		this.CookPlatformType = WinNoEditor
	} else {
		this.CookPlatformType = fmt.Sprintf("%s_%s", this.targetPlatform, this.cookflavor)
	}
}

func (this *Config) GetCookflavor() string {
	return this.cookflavor
}

func (this *Config) SetSVNCode(v string) {
	this.svnCode = v
}

func (this *Config) GetSVNCode() string {
	return this.svnCode
}

func (this *Config) IsEncrypt() bool {
	return this.isEncrypt && this.IsPatch && this.IsRelease
}

//读取config配置文件
//其中读取命令行参数已经无效 - 通过消息设置编译参数
func (this *Config) ReadConfig() error {

	t := time.Now()
	this.Today = fmt.Sprintf("%d%02d%02d", t.Year(), t.Month(), t.Day())

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	LogDebug("exe folder:", filepath.Dir(os.Args[0]))
	if err == nil {
		this.BuilderHome = dir
	} else {
		this.BuilderHome = "E:/golang/uebuildtool"
	}

	this.TempFileHome = fmt.Sprintf("%s/TempFile", this.BuilderHome)
	this.ConfigHome = fmt.Sprintf("%s/config", this.BuilderHome)
	utils.PathExistAndCreate(this.ConfigHome)
	configFileName := this.ConfigHome + "/config.json"

	oldJson, err := utils.ReadJson(configFileName)
	if err != nil {
		LogError("Read config Json Failed! 1")

		utils.WriteFile([]byte(config), configFileName)

		//使用远程编译， 不用生成脚本文件
		//WriteFile([]byte(BuildAndroid), this.BuilderHome+"/BuildAndroid.cmd")
		//WriteFile([]byte(BuildAndroidRes), this.BuilderHome+"/BuildAndroid-Res.cmd")
		//WriteFile([]byte(BuildIOS), this.BuilderHome+"/BuildIOS.cmd")
		return err
	}

	oldJson, err = utils.ReadJson(configFileName)
	if err != nil {
		LogError("Read config Json Failed! 2")
		return err
	}

	ConfigDatas := oldJson.MustMap()
	this.svnCode = utils.GetString(ConfigDatas, "svncode")
	this.ProjectName = utils.GetString(ConfigDatas, "projectName")
	this.UE_EXE = utils.GetString(ConfigDatas, "ue_exe")
	this.UnrealBuildTool = utils.GetString(ConfigDatas, "unrealBuildTool")
	this.AutomationTool = utils.GetString(ConfigDatas, "automationTool")
	this.UnrealPak = utils.GetString(ConfigDatas, "UnrealPak")
	this.teamMembers = utils.GetString(ConfigDatas, "TeamMembers")

	//如果改成网络或者参数传入 修改下面代码
	this.IsPatch = utils.GetInt(ConfigDatas, "isPatch") == 1
	this.IsDebugTool = utils.GetInt(ConfigDatas, "isDebugTool") == 1
	this.isEncrypt = utils.GetInt(ConfigDatas, "isEncrypt") == 1

	//外部传入
	this.IsRelease = false
	this.IsApp = false
	this.CookPlatformType = "Android_ETC2"
	this.targetPlatform = Android
	this.cookflavor = "ETC2"

	this.targetPlatform = IOS
	this.CookPlatformType = IOS

	lenParam := len(os.Args)

	for i := 1; i < lenParam; i++ {
		line := os.Args[i]
		lineParam := strings.Split(line, "=")
		if len(lineParam) < 2 {
			LogError("Unkown Param:", line)
			continue
		}
		Key := strings.TrimSpace(lineParam[0])
		Value := strings.TrimSpace(lineParam[1])

		if Key == "-cookflavor" {
			this.cookflavor = Value
		} else if Key == "-targetPlatform" {
			this.targetPlatform = Value
		} else if Key == "-Release" {
			this.IsRelease, _ = strconv.ParseBool(Value)
		} else if Key == "-Patch" {
			this.IsPatch, _ = strconv.ParseBool(Value)
		} else if Key == "-BuildApp" {
			this.IsApp, _ = strconv.ParseBool(Value)
		} else {
			LogError("Unkown Param:", line)
		}
	}
	if this.targetPlatform == IOS {
		this.CookPlatformType = IOS
	} else if this.targetPlatform == Win64 {
		this.CookPlatformType = Win64
	} else {
		this.CookPlatformType = fmt.Sprintf("%s_%s", this.targetPlatform, this.cookflavor)
	}

	return nil
}

func (this *Config) BuildPath() {
	PackHome := fmt.Sprintf("%s/%s", this.BuilderHome, Output_Dir_IOS)
	utils.PathExistAndCreate(PackHome)

	PackHome = fmt.Sprintf("%s/%s", this.BuilderHome, Output_Dir_Android)
	utils.PathExistAndCreate(PackHome)

	PackHome = fmt.Sprintf("%s/%s", this.BuilderHome, Output_Dir_Win64)
	utils.PathExistAndCreate(PackHome)

	this.ProjectHomePath = fmt.Sprintf("%s/%s", this.BuilderHome, this.ProjectName)
	this.ProjectHomePath = strings.ReplaceAll(this.ProjectHomePath, `\`, "/")
	this.ProjectContentPath = fmt.Sprintf("%s/%s", this.ProjectHomePath, "Content")

	this.JsonHome = fmt.Sprintf("%s/Content/json", this.ProjectHomePath)
	this.LuaHome = fmt.Sprintf("%s/Content/Script", this.ProjectHomePath)
	this.TempJsonHome = fmt.Sprintf("%s/json", this.TempFileHome)
	this.TempLuaHome = fmt.Sprintf("%s/Script", this.TempFileHome)

	this.CookedPath = fmt.Sprintf("%s/Saved/Cooked/%s/%s/Content", this.ProjectHomePath, this.CookPlatformType, this.ProjectName)
	if this.targetPlatform == IOS {
		this.OutputPath = fmt.Sprintf("%s/%s", this.BuilderHome, Output_Dir_IOS)
		this.ConfigHome = fmt.Sprintf("%s/config/iOS", this.BuilderHome)
	} else if this.targetPlatform == Win64 {
		this.OutputPath = fmt.Sprintf("%s/%s", this.BuilderHome, Output_Dir_Win64)
		this.ConfigHome = fmt.Sprintf("%s/config/Win64", this.BuilderHome)
		this.CookedPath = fmt.Sprintf("%s/Saved/Cooked/%s/%s/Content", this.ProjectHomePath, WinNoEditor, this.ProjectName)
	} else {
		this.OutputPath = fmt.Sprintf("%s/%s", this.BuilderHome, Output_Dir_Android)
		this.ConfigHome = fmt.Sprintf("%s/config/Android", this.BuilderHome)
	}

	utils.PathExistAndCreate(this.ConfigHome)

	this.TempPakPath = fmt.Sprintf("%s/paks", this.TempFileHome)
	this.ZipSourcePath = fmt.Sprintf("%s/tempFiles", this.TempFileHome)

	this.ResOutputContentPath = fmt.Sprintf("%s/Content", this.TempFileHome)

	zipFilePath := fmt.Sprintf("%s/%s", this.OutputPath, this.Today)
	utils.PathExistAndCreate(zipFilePath)
	utils.PathExistAndCreate(this.ZipSourcePath)

	this.PluginCodePath = fmt.Sprintf("%s/Plugins/ENGCore", this.ProjectHomePath)
	this.TempPluginCodePath = fmt.Sprintf("%s/ENGCore", this.TempFileHome)
	this.SVNCore = svnCore

	this.PluginUnLuaPath = fmt.Sprintf("%s/Plugins/UnLua", this.ProjectHomePath)
	this.TempPluginUnLuaPath = fmt.Sprintf("%s/UnLua", this.TempFileHome)
	this.SVNUnLua = svnUnLua

	this.SourceGameConfigPath = fmt.Sprintf("%s/Config/DefaultGame.ini", this.ProjectHomePath)
	this.VersionCppFilePath = fmt.Sprintf("%s/Source/%s/GameVersion.cpp", this.ProjectHomePath, this.ProjectName)
	this.ProjectEncryptIniPath = fmt.Sprintf("%s/Config/DefaultCrypto.ini", this.ProjectHomePath)
}

func (this *Config) PrintParams() {
	LogInfo("**********************Params*********************")
	LogInfo("svn code:", this.svnCode)
	LogInfo("project name:", this.ProjectName)

	LogInfo("CookPlatformType:", this.CookPlatformType)
	LogInfo("targetPlatform:", this.targetPlatform)

	LogInfo("Members:", this.teamMembers)

	LogInfo("IsPatch:", this.IsPatch)
	LogInfo("IsRelease:", this.IsRelease)
	LogInfo("IsApp:", this.IsApp)

	LogInfo("-------------------------------------")
	LogInfo("ue exe:", this.UE_EXE)
	LogInfo("unrealBuildTool:", this.UnrealBuildTool)
	LogInfo("automationTool:", this.AutomationTool)
	LogInfo("UnrealPak:", this.UnrealPak)

	LogInfo("*************************************************")
}
