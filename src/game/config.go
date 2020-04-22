// config
package game

import (
	. "core"
	. "file"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var config string = `{
	"svncode":"svn://192.168.0.24/client/ue4/ENGGame",
	"projectName":"ENGGame",
	"ue_exe":"C:/Program Files/Epic Games/UE_4.24/Engine/Binaries/Win64/UE4Editor.exe",
	"unrealBuildTool":"C:/Program Files/Epic Games/UE_4.24/Engine/Binaries/DotNET/UnrealBuildTool.exe",
	"automationTool":"C:/Program Files/Epic Games/UE_4.24/Engine/Build/BatchFiles/RunUAT.bat",
	"UnrealPak":"C:/Program Files/Epic Games/UE_4.24/Engine/Binaries/Win64/UnrealPak.exe",
	"TeamMembers":"liwei,simb",
	"isPatch":1
}`

type Config struct {
	//配置参数
	UE_EXE string
	//UnrealPak工具路径
	UnrealPak       string
	automationTool  string
	unrealBuildTool string

	svnCode string

	BuilderHome     string
	configHome      string
	ProjectHomePath string
	ProjectName     string

	JsonHome string
	LuaHome  string

	CookPlatformType string //Android_ETC2

	//"外网包"
	IsPatch   bool
	IsRelease bool
	IsApp     bool

	targetPlatform string //Android / iOS
	cookflavor     string //ETC2

	//输出目录
	CookedPath           string
	OutputPath           string
	ResOutputContentPath string
	tempPakPath          string
	ZipSourcePath        string

	today       string
	teamMembers string
}

func (this *Config) readConfig() error {

	t := time.Now()
	this.today = fmt.Sprintf("%d%02d%02d", t.Year(), t.Month(), t.Day())

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err == nil {
		this.BuilderHome = dir
	} else {
		this.BuilderHome = "E:/golang/uebuildtool"
	}

	TempPackHome := fmt.Sprintf("%s/APack_iOS", this.BuilderHome)
	PathExistAndCreate(TempPackHome)

	TempPackHome = fmt.Sprintf("%s/APack_Android", this.BuilderHome)
	PathExistAndCreate(TempPackHome)

	this.configHome = fmt.Sprintf("%s/config", this.BuilderHome)
	PathExistAndCreate(this.configHome)
	configFileName := this.configHome + "/config.json"

	oldJson, err := ReadJson(configFileName)
	if err != nil {
		LogError("Read config Json Failed!")

		WriteFile([]byte(config), configFileName)
		return err
	}

	ConfigDatas := oldJson.MustMap()
	this.svnCode = GetString(ConfigDatas, "svncode")
	this.ProjectName = GetString(ConfigDatas, "projectName")
	this.UE_EXE = GetString(ConfigDatas, "ue_exe")
	this.unrealBuildTool = GetString(ConfigDatas, "unrealBuildTool")
	this.automationTool = GetString(ConfigDatas, "automationTool")
	this.UnrealPak = GetString(ConfigDatas, "UnrealPak")
	this.teamMembers = GetString(ConfigDatas, "TeamMembers")

	//如果改成网络或者参数传入 修改下面代码
	this.IsPatch = GetInt(ConfigDatas, "isPatch") == 1
	LogInfo("**********************Params*********************")
	LogInfo("svn code:", this.svnCode)
	LogInfo("project name:", this.ProjectName)
	LogInfo("ue exe:", this.UE_EXE)
	LogInfo("unrealBuildTool:", this.unrealBuildTool)
	LogInfo("automationTool:", this.automationTool)
	LogInfo("UnrealPak:", this.UnrealPak)
	LogInfo("isPatch:", this.IsPatch)
	LogInfo("*************************************************")

	this.ProjectHomePath = fmt.Sprintf("%s/%s", this.BuilderHome, this.ProjectName)
	this.ProjectHomePath = strings.ReplaceAll(this.ProjectHomePath, `\`, "/")

	this.JsonHome = fmt.Sprintf("%s/Content/json", this.ProjectHomePath)
	this.LuaHome = fmt.Sprintf("%s/Content/Script", this.ProjectHomePath)

	//外部传入
	this.IsRelease = false
	this.IsApp = false
	this.CookPlatformType = "Android_ETC2"
	this.targetPlatform = "Android"
	this.cookflavor = "ETC2"

	this.targetPlatform = "iOS"
	this.CookPlatformType = "iOS"

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
	if this.targetPlatform == "iOS" {
		this.CookPlatformType = "iOS"
	} else {
		this.CookPlatformType = fmt.Sprintf("%s_%s", this.targetPlatform, this.cookflavor)
	}
	LogInfo("Program Param's", "-CookPlatformType=", this.CookPlatformType,
		"-targetPlatform=", this.targetPlatform, "-Release=", this.IsRelease, "-Patch=", this.IsPatch)

	this.CookedPath = fmt.Sprintf("%s/Saved/Cooked/%s/%s/Content", this.ProjectHomePath, this.CookPlatformType, this.ProjectName)
	if this.targetPlatform == "iOS" {
		this.OutputPath = fmt.Sprintf("%s/APack_iOS", this.BuilderHome)
		this.configHome = fmt.Sprintf("%s/config/iOS", this.BuilderHome)
	} else {
		this.OutputPath = fmt.Sprintf("%s/APack_Android", this.BuilderHome)
		this.configHome = fmt.Sprintf("%s/config/Android", this.BuilderHome)
	}

	PathExistAndCreate(this.configHome)

	this.tempPakPath = fmt.Sprintf("%s/paks", this.OutputPath)
	this.ZipSourcePath = fmt.Sprintf("%s/tempFiles", this.OutputPath)

	this.ResOutputContentPath = fmt.Sprintf("%s/Content", this.OutputPath)

	zipFilePath := fmt.Sprintf("%s/%s", this.OutputPath, this.today)
	PathExistAndCreate(zipFilePath)
	PathExistAndCreate(this.ZipSourcePath)
	return nil
}
