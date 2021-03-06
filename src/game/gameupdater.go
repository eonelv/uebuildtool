// gameupdater
package game

import (
	"bufio"
	. "cfg"
	. "def"
	. "file"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	. "ngcod.com/core"
	"ngcod.com/utils"

	simplejson "github.com/bitly/go-simplejson"
)

var pakEncryptKey string = `I1RBcG+EjxMTqiLhkoI7GmRWvnAL983H4tSu80YHULU=`

var codetemp string = `//BuildTool Generate, Nerver change it by hand
//eonegame
//%v
#include "GameVersion.h"

GameVersion::GameVersion()
{
}

GameVersion::~GameVersion()
{
}

int GameVersion::Version = %d;
FString GameVersion::EncryptKey = TEXT("%s");
bool GameVersion::IsEncrypt = %v;
bool GameVersion::IsPatch = %v;`

const (
	RESULT_ERROR_CODE_COOK int32 = 0x01
	RESULT_ERROR_CODE_APP  int32 = 0x02
	RESULT_ERROR_CODE_ZIP  int32 = 0x04
)

const (
	UASSET_EXT string = ".uasset"
	UMAP_EXT   string = ".umap"
	UEXP_EXT   string = ".uexp"
	UBULK_EXT  string = ".ubulk"
	JSON_EXT   string = ".json"
	LUA_EXT    string = ".lua"
	EXE_EXT    string = ".exe"
	IPA_EXT    string = ".ipa"
	APK_EXT    string = ".apk"
)

type SMD5 struct {
	relName          string
	sourceParentPath string
	md5              string
}

type SKeyValue struct {
	Key   string
	Value string
}

type GameUpdater struct {
	config *Config
	today  string
	numCPU int

	//是否需要加密Pak
	isEncryptPak bool

	//SVN更新
	sysChan chan string
	svnMsg  string

	newMD5Data map[string]*SMD5

	//计算MD5协程使用
	chanFileName chan string
	isReadAll    bool
	wG           *sync.WaitGroup
	chanMD5      chan *SMD5

	//复制文件协程使用
	chanWattingCopyFileName chan *SMD5

	//Pak协程使用
	chanPakPath chan *SKeyValue

	//SVN版本号
	version int64

	Reslist *Reslist

	cookGameConfigContent    string
	fullGameConfigContent    string
	dynamicUpdateJsonContent string

	outAppFileName string
	outZipFileName string

	beginTime time.Time

	//0x01 cook失败
	//0x02 App失败
	//0x04 zip失败
	result int32

	ProjectID ObjectID
}

func (this *GameUpdater) DoUpdate() {
	defer func() {
		if err := recover(); err != nil {
			LogError(err) //这里的err其实就是panic传入的内容
			LogError("GameUpdater Exit")
		}
	}()
	defer this.sendReport()
	defer this.clear()
	defer utils.SetCmdTitle(APP_TITLE + "-Version:" + APP_VERSION)

	LogInfo("")
	LogInfo("---------------------------------")
	LogInfo("---------Begin Build------------")
	LogInfo("---------time:", time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05"))
	LogInfo("---------------------------------")
	this.beginTime = time.Now()

	/*
		//0. 读取配置文件
		err := this.readConfig()
		if err != nil {
			LogError("Build Failed time:", time.Now())
			return
		}
		this.clear()

		this.config.BuildPath()
	*/
	this.config.PrintParams()

	var multiThreadTask MultiThreadTask
	//1. 更新SVN
	isSvnOK := this.checkOut()
	if !isSvnOK {
		LogError("SVN Update Error!!!", time.Now())
		return
	}
	//2. 复制插件到项目目录
	multiThreadTask = &CopyDirTask{}
	ExecTask(multiThreadTask, this.config.TempPluginCodePath, this.config.PluginCodePath)
	ExecTask(multiThreadTask, this.config.TempPluginUnLuaPath, this.config.PluginUnLuaPath)

	this.netReport("备份Json & Lua")
	//3. 备份SVN中的Json & Lua
	ExecTask(multiThreadTask, this.config.JsonHome, this.config.TempJsonHome)
	ExecTask(multiThreadTask, this.config.LuaHome, this.config.TempLuaHome)

	//读取版本号
	SVNDatabase := &SVNDatabase{}
	SVNDatabase.ProjectPath = this.config.ProjectHomePath
	this.version = SVNDatabase.ReadSVNVersion()
	if this.version == 0 {
		LogError("SVN Version is zero!!!", time.Now())
		return
	}
	//************先出App****************
	//写入动态更新列表
	this.netReport("写入动态更新列表")
	this.readProjectGameSetting()

	dynamiclistFileNme := this.config.ProjectHomePath + "/Content/json/dynamiclist.json"
	utils.WriteFile([]byte(this.dynamicUpdateJsonContent), dynamiclistFileNme)

	//写入代码版本号到C++（这里还需要读取Sqlite的功能，最后再加吧）
	this.writeVersionCPP()

	//加密Json & Lua
	if this.config.IsEncrypt() {
		LogInfo("开始加密文件")
		this.netReport("加密文件")
		multiThreadTask = &EncryptJsonTask{}
		ExecTask(multiThreadTask, this.config.JsonHome, "")
		ExecTask(multiThreadTask, this.config.LuaHome, "")
	}

	//1. 在C++代码被修改之后，特别是蓝图父类，会丢失蓝图，必须重新check一次代码，所以更新完马上编译
	//这种情况必须要重编C++代码
	//2. ENGCore.UnLua插件是时时编译的，所以需要重编C++代码
	this.buildFirst()
	okApp := this.buildApp()
	if !okApp {
		return
	}

	//删除动态更新列表
	os.Remove(dynamiclistFileNme)
	//解密Json & Lua
	if this.config.IsEncrypt() {
		LogInfo("开始解密文件")
		multiThreadTask = &EncryptJsonTask{}
		ExecTask(multiThreadTask, this.config.JsonHome, "")
		ExecTask(multiThreadTask, this.config.LuaHome, "")
	}

	//***************再出资源************
	//Cook Data
	this.cookDatas()

	//对比输出需要打包的文件（读取旧的文件MD5, 计算新的MD5）
	oldJson, err := utils.ReadJson(this.config.ConfigHome + "/version.json")
	if err != nil {
		LogError("Read Old Json Failed!")
	}

	this.calcNewMD5()
	Result := this.merge(oldJson)

	if !this.config.IsPatch {
		oldInnerJson, err := utils.ReadJson(this.config.ConfigHome + "/versionInner.json")
		if err != nil {
			LogError("Read Old 'versionInner.json' Failed!")
		} else {
			Result = this.mergeInner(Result, oldInnerJson)
		}
	}
	this.copyFiles(Result)

	//4. 根据对比结果生成pak
	this.writeCrypto()
	this.processReslist()

	this.buildPak()

	//文件拆分
	LogInfo("开始拆分Pak文件-目标目录是ZipSourcePath")
	this.netReport("拆分Pak文件")
	multiThreadTask = &FileSpliterTask{}
	ExecTask(multiThreadTask, this.config.TempPakPath, this.config.ZipSourcePath)

	LogInfo("开始复制json&lua目录-目标目录是ZipSourcePath")
	this.netReport("复制Json & Lua到输出目录")
	multiThreadTask = &CopyDirTask{}
	ExecTask(multiThreadTask, this.config.ResOutputContentPath+"/Script", this.config.ZipSourcePath+"/Script")
	ExecTask(multiThreadTask, this.config.ResOutputContentPath+"/json", this.config.ZipSourcePath+"/json")

	//遍历pak目录，计算pak的MD5
	//如果是外网版本，增加一行记录
	//如果是内网版本，替换掉现有记录
	this.netReport("计算所有待输出文件的MD5")
	pakmd5 := &PakMD5{}
	//计算原始文件MD5
	pakmd5.CalcMD5(this.Reslist, this.numCPU, this.config.TempPakPath, this.config.IsPatch, this.version)
	//计算拆分文件MD5
	pakmd5.CalcMD5(this.Reslist, this.numCPU, this.config.ZipSourcePath, this.config.IsPatch, this.version)

	//输出reslist文件
	errWriteReslist := this.Reslist.Flush(this.version)
	if errWriteReslist != nil {
		this.clearWhenError()
		return
	}

	if this.config.IsPatch {
		renameJsonTask := &RenameDirTask{}
		//如果是发布版，修改文件名(这里只需要修改文件名就可以了，版本号在reslist.json里面有)
		//项目代码在更新的时候，URL文件名加载对应的版本号即可
		renameJsonTask.TargetNamePostfix = fmt.Sprintf("_v_%d", this.version)
		ExecTask(renameJsonTask, this.config.ZipSourcePath+"/Script", this.config.ZipSourcePath+"/Script")
		ExecTask(renameJsonTask, this.config.ZipSourcePath+"/json", this.config.ZipSourcePath+"/json")
	}
	//写入Version.json文件
	okVersion := this.writeVersion(oldJson)
	if !okVersion {
		this.Reslist.Reverse()
		this.clearWhenError()
	}

	okVersion = this.writeInnerVersion(Result)
	if !okVersion {
		this.Reslist.Reverse()
		this.clearWhenError()
	}

	//生成压缩包
	okZip := this.zipSpPackage()
	if !okZip {
		this.Reslist.Reverse()
		this.clearWhenError()
		return
	}
}

func (this *GameUpdater) zipSpPackage() bool {
	this.netReport("正在生成压缩包")
	zipFilePath := fmt.Sprintf("%s/%s", this.config.OutputPath, this.today)
	prefixPatch := ""
	if this.config.IsPatch {
		prefixPatch = "sp_"
	}

	this.outZipFileName = fmt.Sprintf("%s/%sres_%s_%s_v%d.zip", zipFilePath, prefixPatch, this.config.GetTargetPlatform(), this.today, this.version)

	if this.checkAvalibleZipFile(this.config.ZipSourcePath) != 0 {
		err := Zip(this.config.ZipSourcePath, this.outZipFileName)
		if err != nil {
			this.result |= RESULT_ERROR_CODE_ZIP
			return false
		}
	} else {
		this.outZipFileName = ""
	}
	return true
}

func (this *GameUpdater) checkAvalibleZipFile(path string) int {
	rd, err := ioutil.ReadDir(path)
	if err != nil {
		return 0
	}
	avalibleCount := 0
	for _, fi := range rd {
		if fi.IsDir() {
			avalibleCount += this.checkAvalibleZipFile(path + "/" + fi.Name())
		} else {
			Name := fi.Name()
			if !strings.Contains(Name, "pakversion_") && !strings.Contains(Name, "reslist_") {
				avalibleCount++
			}
		}
	}
	return avalibleCount
}

func (this *GameUpdater) processReslist() error {
	this.Reslist = &Reslist{}
	this.Reslist.configHome = this.config.ConfigHome
	this.Reslist.ZipSourcePakPath = this.config.ZipSourcePath
	this.Reslist.IsPatch = this.config.IsPatch
	this.Reslist.IsEncrypt = this.config.IsEncrypt()

	this.Reslist.ReadData()
	return nil
}

func (this *GameUpdater) ReadConfig() error {
	this.numCPU = runtime.NumCPU()
	this.config = &Config{}
	err := this.config.ReadConfig()
	this.today = this.config.Today
	return err
}

func (this *GameUpdater) GetConfig() *Config {
	return this.config
}

func (this *GameUpdater) buildFirst() {
	if this.config.IsDebugTool {
		return
	}
	LogInfo("Begin Build First!")
	this.netReport("预编译项目")
	LogInfo("**********Begin checkout svn code**********")
	ProjectFileParam := fmt.Sprintf(`-Project=%s/%s.uproject`, this.config.ProjectHomePath, this.config.ProjectName)
	utils.Exec(this.config.UnrealBuildTool, "Development", "Win64", ProjectFileParam, "-TargetType=Editor", "-Progress", "-NoHotReloadFromIDE")
	LogInfo("**********BuildFirst Complete!**********")
}

func (this *GameUpdater) writeCrypto() {
	cryptoJsonPath := fmt.Sprintf("%s/Config/DefaultCrypto.json", this.config.ProjectHomePath)

	file, err := os.OpenFile(this.config.ProjectEncryptIniPath, os.O_RDONLY, os.ModeAppend)

	if err != nil {
		this.isEncryptPak = false
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	scanner.Split(bufio.ScanLines)

	var encryptLine string = ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// expression match
		index := strings.Index(line, "=")
		if index == -1 {
			encryptLine += fmt.Sprintf("%s\n", line)
		} else {
			Key := strings.TrimSpace(line[:index])
			if Key == "EncryptionKey" {
				encryptLine += fmt.Sprintf("EncryptionKey=%s\n", pakEncryptKey)
			} else {
				encryptLine += fmt.Sprintf("%s\n", line)
			}
		}
	}
	utils.WriteFile([]byte(encryptLine), this.config.ProjectEncryptIniPath)

	this.isEncryptPak = true
	tempMsg := fmt.Sprintf("{\n\"EncryptionKey\":{\"Key\":\"%s\"}\n}", pakEncryptKey)
	utils.WriteFile([]byte(tempMsg), cryptoJsonPath)
}

//build pak 可以并行操作
func (this *GameUpdater) findPakContent() {
	//直接读取输出目录的文件生成pak
	rd, err := ioutil.ReadDir(this.config.ResOutputContentPath)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	for _, fi := range rd {
		if !fi.IsDir() {
			continue
		}
		if fi.Name() == "Script" || fi.Name() == "json" {
			continue
		}
		Name := fmt.Sprintf("%s/%s", this.config.ResOutputContentPath, fi.Name())
		this.chanPakPath <- &SKeyValue{Name, fi.Name()}
	}
}

func (this *GameUpdater) buildPak() {
	this.netReport("打包Pak")
	LogInfo("**********Begin buildPak**********")
	if ok, _ := utils.PathExists(this.config.TempPakPath); !ok {
		os.MkdirAll(this.config.TempPakPath, os.ModePerm)
	} else {
		os.RemoveAll(this.config.TempPakPath)
	}

	this.chanPakPath = make(chan *SKeyValue, this.numCPU)
	defer close(this.chanPakPath)
	go this.findPakContent()

	this.wG = &sync.WaitGroup{}
	this.wG.Add(this.numCPU)
	for i := 0; i < this.numCPU; i++ {
		go this.go_build()
	}
	this.wG.Wait()
	LogInfo("**********buildPak Complete!**********")
}

func (this *GameUpdater) go_build() {
	defer func() {
		if err := recover(); err != nil {
			LogError(err) //这里的err其实就是panic传入的内容
			LogError("GameUpdater Exit")
		}
	}()
	defer this.wG.Done()
	for {
		select {
		case s := <-this.chanPakPath:
			this.buildSinglePak(s)
		case <-time.After(10 * time.Second):
			return
		}
	}
}

func (this *GameUpdater) buildSinglePak(pakSrcPath *SKeyValue) {
	outputPak := fmt.Sprintf("%s/%s_p_%d.pak", this.config.TempPakPath, pakSrcPath.Value, this.Reslist.GetPakIndex(pakSrcPath.Value))

	LogInfo("Build pak file Name=", outputPak)
	sourceFile := fmt.Sprintf("-create=%s", pakSrcPath.Key)
	cryptoJsonKey := fmt.Sprintf("-cryptokeys=%s/Config/DefaultCrypto.json", this.config.ProjectHomePath)
	if this.isEncryptPak {
		LogDebug("Create Pak Encryption")
		ExecPakCmd(this.config.UnrealPak, outputPak, sourceFile, cryptoJsonKey)
	} else {
		ExecPakCmd(this.config.UnrealPak, outputPak, sourceFile)
	}
}

func (this *GameUpdater) copyFiles(Result map[string]*SMD5) {
	this.netReport("复制文件到Pak打包目录")
	LogInfo("**********Begin copyFiles**********")
	this.chanWattingCopyFileName = make(chan *SMD5, this.numCPU)
	defer close(this.chanWattingCopyFileName)
	go this.writeCopyFileToChannel(Result)

	this.wG = &sync.WaitGroup{}
	this.wG.Add(this.numCPU)
	for i := 0; i < this.numCPU; i++ {
		go this.go_CopyFile()
	}
	this.wG.Wait()
	LogInfo("**********copyFiles Complete**********")
}

func (this *GameUpdater) writeCopyFileToChannel(Result map[string]*SMD5) {
	for Key := range Result {
		this.chanWattingCopyFileName <- Result[Key]
	}
}

func (this *GameUpdater) go_CopyFile() {
	defer this.wG.Done()
	for {
		select {
		case s := <-this.chanWattingCopyFileName:
			fileName := s.sourceParentPath + "/" + s.relName
			targetFileName := this.config.ResOutputContentPath + "/" + s.relName
			utils.CopyFile(fileName, targetFileName)

			if strings.HasSuffix(targetFileName, UMAP_EXT) {
				targetFileName = strings.ReplaceAll(targetFileName, UMAP_EXT, UEXP_EXT)
				fileName = strings.ReplaceAll(fileName, UMAP_EXT, UEXP_EXT)
				utils.CopyFile(fileName, targetFileName)
			} else if strings.HasSuffix(targetFileName, UASSET_EXT) {
				targetFileName = strings.ReplaceAll(targetFileName, UASSET_EXT, UEXP_EXT)
				fileName = strings.ReplaceAll(fileName, UASSET_EXT, UEXP_EXT)
				utils.CopyFile(fileName, targetFileName)

				targetFileName = strings.ReplaceAll(targetFileName, UEXP_EXT, UBULK_EXT)
				fileName = strings.ReplaceAll(fileName, UEXP_EXT, UBULK_EXT)
				utils.CopyFile(fileName, targetFileName)
			}

			if this.config.IsEncrypt() &&
				(strings.HasSuffix(s.relName, JSON_EXT) || strings.HasSuffix(s.relName, LUA_EXT)) {
				EncryptFile(targetFileName)
				CompressFile(targetFileName)
			}
		case <-time.After(10 * time.Second):
			return
		}
	}
}

func (this *GameUpdater) mergeInner(OldResult map[string]*SMD5, innerJson *simplejson.Json) map[string]*SMD5 {
	innerMD5Data := innerJson.MustMap()

	for Key := range innerMD5Data {
		OldMD5, OK := OldResult[Key]
		_, InnerOK := innerMD5Data[Key]
		if !InnerOK {
			LogDebug("Merge Inner:", "Inner Json no Key=", Key)
			continue
		}
		NewMD5, NewOK := this.newMD5Data[Key]
		if !NewOK {
			LogDebug("Merge Inner:", "newMD5Data no Key=", Key)
			continue
		}
		if OK {
			OldMD5.md5 = NewMD5.md5
			LogDebug("Merge Inner:", "Update Md5", Key, NewMD5.md5)
		} else {
			OldMD5 = NewMD5
			LogDebug("Merge Inner:", "Set Md5", Key, NewMD5.md5)
		}
		OldResult[Key] = OldMD5
	}
	return OldResult
}

func (this *GameUpdater) merge(oldJson *simplejson.Json) map[string]*SMD5 {
	OldMD5Data := oldJson.MustMap()
	RemoveList := make(map[string]byte)

	for Key := range OldMD5Data {
		_, OldOK := OldMD5Data[Key]
		if !OldOK {
			continue
		}
		_, NewOK := this.newMD5Data[Key]
		if !NewOK {
			RemoveList[Key] = 0
		}
	}
	for Key := range RemoveList {
		LogDebug("merge: delete Version's Key=", Key)
		delete(OldMD5Data, Key)
	}

	//新的MD5值不等于旧的， 也有可能是新增的文件
	//TODO 如果更新模式发生改变，也要标记为差异文件
	//TODO 关于模式可能要用新的配置文件来标记
	//ResultList用来复制文件和生成Version.json
	ResultList := make(map[string]*SMD5)
	for Key := range this.newMD5Data {
		OldMD5 := OldMD5Data[Key]
		NewMD5 := this.newMD5Data[Key]
		if OldMD5 != NewMD5.md5 {
			ResultList[Key] = NewMD5
			LogDebug("MD5 is diffrent, Key=", Key, "Old=", OldMD5, "New=", NewMD5.md5)
			oldJson.Set(Key, NewMD5.md5)
		}
	}
	return ResultList
}

func (this *GameUpdater) calcNewMD5() {
	this.netReport("计算Cook资源的MD5")
	LogInfo("**********Begin calculate source file md5 **********")
	this.newMD5Data = make(map[string]*SMD5)
	this.chanFileName = make(chan string, this.numCPU)
	//this.isReadAll = false
	fmt.Println("======================================")

	go this.readAll()

	this.chanMD5 = make(chan *SMD5, this.numCPU)
	this.wG = &sync.WaitGroup{}
	this.wG.Add(this.numCPU)
	for i := 0; i < this.numCPU; i++ {
		go this.calcSingle()
	}
	completeChan := make(chan bool)
	defer close(completeChan)

	go this.writeNewMD5(completeChan)
	this.wG.Wait()
	completeChan <- true
	LogInfo("**********Calculate source file md5 complete**********")
}

func (this *GameUpdater) writeNewMD5(completeChan chan bool) {
	for {
		select {
		case FileMD5 := <-this.chanMD5:
			this.newMD5Data[FileMD5.relName] = FileMD5
		case <-completeChan:
			return
		}
	}
}

func (this *GameUpdater) calcSingle() {
	defer this.wG.Done()
	projectContentPath := this.config.ProjectContentPath
	var parentSourcePath string = projectContentPath
	for {
		select {
		case s := <-this.chanFileName:
			parentSourcePath = this.config.CookedPath
			md5 := utils.CalcFileMD5(s)
			if strings.HasSuffix(s, JSON_EXT) || strings.HasSuffix(s, LUA_EXT) {
				parentSourcePath = projectContentPath
			}

			RelName := string(s[strings.Count(projectContentPath, ""):])
			this.chanMD5 <- &SMD5{RelName, parentSourcePath, md5}
		case <-time.After(5 * time.Second):
			return
		}
	}
}

func (this *GameUpdater) readAll() {
	// 按Cook之后的文件计算差异
	// this.readFiles(this.config.CookedPath)
	// this.readFiles(this.config.JsonHome)
	// this.readFiles(this.config.LuaHome)
	this.readFiles(this.config.ProjectContentPath)
}

func (this *GameUpdater) readFiles(pathname string) {
	rd, err := ioutil.ReadDir(pathname)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	for _, fi := range rd {
		if fi.IsDir() {
			if fi.Name() == "Engine" {
				return
			}
			this.readFiles(pathname + "/" + fi.Name())
		} else {
			Name := pathname + "/" + fi.Name()
			this.chanFileName <- Name
		}
	}
}

func (this *GameUpdater) readProjectGameSetting() {

	file, err := os.OpenFile(this.config.SourceGameConfigPath, os.O_RDONLY, os.ModeAppend)

	if err != nil {
		LogError("Read DefaultGame.ini failed", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	this.cookGameConfigContent = ""
	this.fullGameConfigContent = ""
	this.dynamicUpdateJsonContent = "{"
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		//LogInfo("Dynamic Update Config Source:", line)
		this.fullGameConfigContent += fmt.Sprintf("%s\n", line)

		// expression match
		index := strings.Index(line, "=")

		if index == -1 {
			this.cookGameConfigContent += fmt.Sprintf("%s\n", line)
			continue
		}
		Key := strings.TrimSpace(line[:index])

		if Key != "+DirectoriesToNeverCook" {
			this.cookGameConfigContent += fmt.Sprintf("%s\n", line)
			continue
		}
		//记录动态更新的目录
		Value := strings.TrimSpace(line[index+1:])
		LogInfo("Dynamic Update Config:", Value)
		GameKey := `/Game/`
		index = strings.Index(Value, GameKey)
		if index == -1 {
			this.cookGameConfigContent += fmt.Sprintf("%s\n", line)
			continue
		}
		Value = Value[index+len(GameKey) : len(Value)-2]
		this.dynamicUpdateJsonContent += fmt.Sprintf(`"%s":1,`, Value)
	}
	lenDynamicCount := len(this.dynamicUpdateJsonContent)
	if lenDynamicCount != 1 {
		this.dynamicUpdateJsonContent = this.dynamicUpdateJsonContent[:lenDynamicCount-1]
	}
	this.dynamicUpdateJsonContent += "}"
	LogInfo("Dynamiclist", this.dynamicUpdateJsonContent)
}

func (this *GameUpdater) cookDatas() {
	defer func() {
		if err := recover(); err != nil {
			this.result |= RESULT_ERROR_CODE_COOK
			panic(err)
		}
	}()

	this.netReport("Cook所有资源")

	utils.WriteFile([]byte(this.cookGameConfigContent), this.config.SourceGameConfigPath)
	LogInfo("**********Begin Cook Content**********")
	ProjectFile := fmt.Sprintf("%s/%s.uproject", this.config.ProjectHomePath, this.config.ProjectName)
	LogFile := fmt.Sprintf("%s/log/Cook-2020.txt", this.config.ProjectHomePath)

	var pLogFormatted string = fmt.Sprintf("-abslog=%s", LogFile)
	var pPlatformFormatted string = fmt.Sprintf("-TargetPlatform=%s", this.config.CookPlatformType)

	if ok, _ := utils.PathExists(this.config.OutputPath); !ok {
		os.MkdirAll(this.config.OutputPath, os.ModePerm)
	}

	err := ExecCookCmd(this.config.UE_EXE, ProjectFile, pPlatformFormatted, pLogFormatted)
	if err != nil {
		LogError("Build App failed.", err.Error())
	}
	LogInfo("**********Cook Content Complete**********")
}

func (this *GameUpdater) writeVersionCPP() {
	code := fmt.Sprintf(codetemp, time.Now(), this.version, pakEncryptKey, this.config.IsEncrypt(), this.config.IsPatch)
	utils.WriteFile([]byte(code), this.config.VersionCppFilePath)
}

func (this *GameUpdater) buildApp() bool {

	defer func() {
		if err := recover(); err != nil {
			LogError("Build App Error:", err) //这里的err其实就是panic传入的内容
			this.result |= RESULT_ERROR_CODE_APP
		}
	}()

	if !this.config.IsApp {
		return true
	}
	this.netReport("开始编译App")

	LogInfo("**********Begin buildApp**********")
	var tempBuildFile string = fmt.Sprintf("%s/TempBuild.cmd", this.config.BuilderHome)

	//应用程序生成的根目录, 与参数-archivedirectory指定的目录可能不相同
	//Android是TargetPlarform_Cookflavor
	tempTargetPlatform := this.config.GetTargetPlatform()
	achieveDir := fmt.Sprintf("%s/%s", this.config.OutputPath, tempTargetPlatform)

	defer os.Remove(tempBuildFile)

	a1 := fmt.Sprintf("-ScriptsForProject=%s/%s.uproject", this.config.ProjectHomePath, this.config.ProjectName)
	a2 := "BuildCookRun"
	a3 := fmt.Sprintf("-project=%s/%s.uproject", this.config.ProjectHomePath, this.config.ProjectName)
	a4 := fmt.Sprintf("-archivedirectory=%s", this.config.OutputPath)
	a5 := ""

	a6 := ""
	if !this.config.IsRelease {
		a6 = "-clientconfig=DebugGame"
	} else {
		a6 = "-clientconfig=Shipping"
	}

	a7 := fmt.Sprintf(`-ue4exe="%s"`, this.config.UE_EXE)
	a8 := fmt.Sprintf("-targetplatform=%s", tempTargetPlatform)

	var fixedParams = []string{"-nocompileeditor", "-nop4", "-cook",
		"-stage", "-archive", "-package", "-compressed",
		"-SkipCookingEditorContent",
		"-pak", "-prereqs", "-nodebuginfo", "-build",
		"-utf8output", "-compile"}
	var argAll []string

	if tempTargetPlatform == Android {
		achieveDir = fmt.Sprintf("%s/%s_%s", this.config.OutputPath, tempTargetPlatform, this.config.GetCookflavor())
		a5 = fmt.Sprintf("-cookflavor=%s", this.config.GetCookflavor())
	} else {
		a4 = fmt.Sprintf("-archivedirectory=%s", achieveDir)
	}

	argAll = append(argAll, a1)
	argAll = append(argAll, a2)
	argAll = append(argAll, a3)
	argAll = append(argAll, a4)
	argAll = append(argAll, a5)
	argAll = append(argAll, a6)
	argAll = append(argAll, a7)
	argAll = append(argAll, a8)

	argAll = append(argAll, fixedParams...)
	var cmdString = "\"" + this.config.AutomationTool + "\""
	for _, a := range argAll {
		if a == "" {
			continue
		}
		cmdString += " "
		cmdString += a
	}
	utils.WriteFile([]byte(cmdString), tempBuildFile)

	defer os.RemoveAll(achieveDir)
	err := ExecApp(tempBuildFile)
	if err != nil {
		LogError("Build App failed.", err.Error())
	}
	targetZipFilePath := fmt.Sprintf("%s/%s", this.config.OutputPath, this.today)

	rd, err := ioutil.ReadDir(achieveDir)
	if err != nil {
		os.Remove(tempBuildFile)
		this.result |= RESULT_ERROR_CODE_APP
		LogInfo("**********buildApp Failed!**********")
		return false
	}

	this.outAppFileName = ""
	if this.config.GetTargetPlatform() == Win64 {
		this.copyWinPackage(rd, targetZipFilePath, achieveDir, achieveDir)
	} else {
		this.copyApp(rd, targetZipFilePath, achieveDir)
	}

	if ok, _ := utils.PathExists(this.outAppFileName); ok {
		LogInfo("**********buildApp Success!**********")
		return true
	} else {
		this.result |= RESULT_ERROR_CODE_APP
	}

	LogInfo("**********buildApp Complete!**********")
	return false
}

func (this *GameUpdater) copyApp(rd []os.FileInfo, targetZipFilePath string, parentDir string) bool {
	for _, fi := range rd {
		if fi.IsDir() {
			parentDir = parentDir + "/" + fi.Name()
			folders, _ := ioutil.ReadDir(parentDir)
			ok := this.copyApp(folders, targetZipFilePath, parentDir)
			if ok {
				return true
			}
			continue
		}

		name := fi.Name()
		LogInfo("**********复制apk or ipa**********", name)
		if strings.Contains(name, APK_EXT) || strings.Contains(name, IPA_EXT) {
			name = strings.ReplaceAll(name, "-IOS", "")
			name = strings.ReplaceAll(name, "-Android", "")
			name = strings.ReplaceAll(name, "-Shipping", "")
			name = strings.ReplaceAll(name, "DebugGame", "Debug")
			index := strings.LastIndex(name, ".")

			prefixPatch := ""
			if this.config.IsPatch {
				prefixPatch = "sp_"
			}
			this.outAppFileName = fmt.Sprintf("%s/%s%s_v%d.%s", targetZipFilePath, prefixPatch, name[:index], this.version, name[index+1:])
			utils.CopyFile(parentDir+"/"+fi.Name(), this.outAppFileName)
			return true
		}
	}
	return false
}
func (this *GameUpdater) copyWinPackage(rd []os.FileInfo, zipFilePath string, achieveDir string, parentDir string) bool {
	for _, fi := range rd {
		if fi.IsDir() {
			parentDir = parentDir + "/" + fi.Name()
			folders, _ := ioutil.ReadDir(parentDir)
			ok := this.copyWinPackage(folders, zipFilePath, achieveDir, parentDir)
			if ok {
				return true
			}
			continue
		}

		name := fi.Name()
		if strings.Contains(name, EXE_EXT) {
			prefixPatch := ""
			if this.config.IsPatch {
				prefixPatch = "sp_"
			}
			this.outAppFileName = fmt.Sprintf("%s/%s%s_v%d.zip", zipFilePath, prefixPatch, "WinGame", this.version)

			tempExePath := achieveDir + "/" + WinNoEditor
			if this.checkAvalibleZipFile(tempExePath) != 0 {
				err := Zip(tempExePath, this.outAppFileName)
				if err != nil {
					this.result |= RESULT_ERROR_CODE_APP
					return false
				}
			}
			return true
		}
	}
	return false
}

func (this *GameUpdater) checkOut() bool {
	this.netReport("更新SVN")
	LogInfo("**********Begin checkout svn code**********")

	this.sysChan = make(chan string)
	defer close(this.sysChan)

	go this.svnCheckout()

	for {
		select {
		case this.svnMsg = <-this.sysChan:
			if this.svnMsg == "error" {
				return false
			} else if this.svnMsg == "ok" {
				LogInfo("Build Application. update svn complete!\r\n")
				return true
			}
		case <-time.After(5 * time.Second):
			fmt.Println("Now updating svn ..." + this.svnMsg)
		}
	}
	return false
}

func (this *GameUpdater) svnCheckout() {
	defer func() {
		if err := recover(); err != nil {
			LogError("svn update error:", err) //这里的err其实就是panic传入的内容
			LogError("svnCheckout Exit")
			this.sysChan <- "error"
		}
	}()
	LogInfo("The next step is to update code", this.config.GetSVNCode(), this.config.ProjectName)
	this.sysChan <- "updating code"

	//内网才需要更新项目代码
	if !this.config.IsPatch {
		ok, _ := utils.PathExists(this.config.ProjectName)
		if !ok {
			ExecSVNCmd("svn", "checkout", this.config.GetSVNCode(), this.config.ProjectName)
		} else {
			//这里原来是在clear清理
			//现在外网包不更新代码了，放到这里清理
			os.Remove(this.config.ProjectEncryptIniPath)
			os.Remove(this.config.VersionCppFilePath)
			os.Remove(this.config.SourceGameConfigPath)

			//Lua & Json可能是加密过的，所以要删除. 下次编译重新更新
			os.RemoveAll(this.config.JsonHome)
			os.RemoveAll(this.config.LuaHome)
		}

		ExecSVNCmd("svn", "cleanup", this.config.ProjectName)
		ExecSVNCmd("svn", "update", "--force", this.config.ProjectName, "--accept", "theirs-full")
	}

	//更新插件源码
	ExecSVNCmd("svn", "checkout", this.config.SVNCore, this.config.TempPluginCodePath)
	//更新UnLua源码
	ExecSVNCmd("svn", "checkout", this.config.SVNUnLua, this.config.TempPluginUnLuaPath)

	//清除插件
	os.RemoveAll(this.config.PluginCodePath)
	os.RemoveAll(this.config.PluginUnLuaPath)
	this.sysChan <- "ok"
}

func (this *GameUpdater) writeVersion(oldJson *simplejson.Json) bool {
	if !this.config.IsPatch {
		return true
	}
	Bytes, err := oldJson.MarshalJSON()
	if err != nil {
		LogError("Read Json Data Error!", err)
		return false
	}
	errWrite := utils.WriteFile(Bytes, this.config.ConfigHome+"/version.json")
	if errWrite != nil {
		LogError("Write version.json Error!", errWrite)
		return false
	}
	errWrite = utils.WriteFile(Bytes, fmt.Sprintf("%s/version_%d.json", this.config.ZipSourcePath, this.version))
	if errWrite != nil {
		LogError("Write version.json to package Error!", errWrite)
		return false
	}
	return true
}

func (this *GameUpdater) writeInnerVersion(InnerVersion map[string]*SMD5) bool {
	if this.config.IsPatch {
		s := "{}"
		errWriteNil := utils.WriteFile([]byte(s), this.config.ConfigHome+"/versionInner.json")
		if errWriteNil != nil {
			LogError("Write versionInner.json Error!", errWriteNil)
			return false
		}
		return true
	}
	oldJson := simplejson.New()
	for Key := range InnerVersion {
		OldMD5, OK := InnerVersion[Key]
		if !OK {
			continue
		}
		oldJson.Set(Key, OldMD5.md5)
	}

	Bytes, err := oldJson.MarshalJSON()
	if err != nil {
		LogError("writeInnerVersion oldJson convert to byte Data Error!", err)
		return false
	}
	errWrite := utils.WriteFile(Bytes, this.config.ConfigHome+"/versionInner.json")
	if errWrite != nil {
		LogError("Write versionInner.json Error!", errWrite)
		return false
	}
	return true
}

func (this *GameUpdater) clear() {
	//删除缓存文件
	//os.RemoveAll(this.config.ResOutputContentPath)
	//os.RemoveAll(this.config.TempPakPath)
	//os.RemoveAll(this.config.ZipSourcePath)

	//Lua & Json可能是加密过的，所以要删除. 下次编译重新更新
	os.RemoveAll(this.config.JsonHome)
	os.RemoveAll(this.config.LuaHome)

	//插件使用完之后也要删除. 下次编译重新更新
	os.RemoveAll(this.config.PluginCodePath)

	os.RemoveAll(this.config.PluginUnLuaPath)

	//还原DefaultGame.ini
	//    如果下次是内网包，更新SVN时会删除现有的
	//    如果下次是外网包，使用还原的这份
	utils.WriteFile([]byte(this.fullGameConfigContent), this.config.SourceGameConfigPath)

	multiThreadTask := &CopyDirTask{}
	ExecTask(multiThreadTask, this.config.TempJsonHome, this.config.JsonHome)
	ExecTask(multiThreadTask, this.config.TempLuaHome, this.config.LuaHome)

	os.RemoveAll(this.config.TempFileHome)
}

func (this *GameUpdater) sendReport() {
	tempURL := `http://192.168.0.10/zentaopms/www/sendmsg.php?user=%s&msg=%s`

	ip, err := utils.GetLocalIP()
	var msgtemp string
	msgtemp = fmt.Sprintf("%s:%s-%s. 参数: 外网包=%v, 发布版=%v.", ip, this.config.ProjectName, this.config.GetTargetPlatform(),
		this.config.IsPatch, this.config.IsRelease)
	if this.outAppFileName != "" {
		ip, _ := utils.GetLocalIP()
		msgtemp += fmt.Sprintf(" [App]:http://%s/%s", ip, this.outAppFileName[len(this.config.BuilderHome)+1:])
	}
	if this.outZipFileName != "" {
		msgtemp += fmt.Sprintf(" [Zip]:http://%s/%s", ip, this.outZipFileName[len(this.config.BuilderHome)+1:])
	} else {
		msgtemp += fmt.Sprintf(" [Zip]:%s", "没有")
	}

	if this.result&RESULT_ERROR_CODE_COOK == RESULT_ERROR_CODE_COOK {
		msgtemp += fmt.Sprintf(" Error:%s", "Cook失败")
	}
	if this.result&RESULT_ERROR_CODE_APP == RESULT_ERROR_CODE_APP {
		msgtemp += fmt.Sprintf(" Error:%s", "App失败")
	}
	if this.result&RESULT_ERROR_CODE_ZIP == RESULT_ERROR_CODE_ZIP {
		msgtemp += fmt.Sprintf(" Error:%s", "Zip失败")
	}

	timePassed := time.Now().Unix() - this.beginTime.Unix()

	timeString := fmt.Sprintf(". 耗时:%dh:%dm:%ds(%ds).", timePassed/60/60, timePassed%3600/60, timePassed%3600%60, timePassed)
	msgtemp += timeString
	//生成client 参数为默认
	client := &http.Client{}

	notifyMem := strings.ReplaceAll(this.config.GetMembers(), "-", ",")
	//生成要访问的url
	url := fmt.Sprintf(tempURL, notifyMem, msgtemp)

	//提交请求
	reqest, err := http.NewRequest("GET", url, nil)

	if err != nil {
		panic(err)
	}

	//处理返回结果
	//response, _ := client.Do(reqest)
	client.Do(reqest)

	//将结果定位到标准输出 也可以直接打印出来 或者定位到其他地方进行相应的处理
	//stdout := os.Stdout
	//_, err = io.Copy(stdout, response.Body)

	//返回的状态码
	//status := response.StatusCode

	LogInfo("---------------------------------")
	LogInfo("-------Build Complete!!!---------")
	LogInfo("-------time:", time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05"))
	LogInfo("-------Total Time", timeString)
	LogInfo("---------------------------------")
}

func (this *GameUpdater) clearWhenError() {
	LogDebug("clearWhenError")
	os.Remove(this.outAppFileName)
	os.Remove(this.outZipFileName)
}

func (this *GameUpdater) netReport(msg string) {
	cmd := &Command{CMD_SYSTEM_NET_REPORT, msg, nil, this.ProjectID}

	channel := GetChanByID(SYSTEM_CHAN_ID)
	err := SendCommand(channel, cmd, 10)
	if err != nil {
		LogError(err)
	}
}

func ReadFileData(filePath string) ([]byte, error) {
	datas, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return datas, nil
}
