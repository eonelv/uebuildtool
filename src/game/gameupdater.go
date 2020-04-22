// gameupdater
package game

import (
	"bufio"
	. "core"
	"crypto/md5"
	"encoding/json"
	"errors"
	. "file"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

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
FString GameVersion::EncryptKey = TEXT("%s");`

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

	versionCppFilePath    string
	projectEncryptIniPath string

	sourceGameConfigPath     string
	cookGameConfigContent    string
	dynamicUpdateJsonContent string

	outAppFileName string
	outZipFileName string

	beginTime time.Time

	//0x01 cook失败
	//0x02 App失败
	//0x04 zip失败
	result int
}

func (this *GameUpdater) DoUpdate() {
	defer func() {
		if err := recover(); err != nil {
			LogError(err) //这里的err其实就是panic传入的内容
			LogError("Process Exit")
		}
	}()

	defer this.clear()
	defer this.sendReport()

	LogInfo("")
	LogInfo("---------------------------------")
	LogInfo("---------Begin Build------------")
	LogInfo("---------time:", time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05"))
	LogInfo("---------------------------------")
	this.beginTime = time.Now()

	//0. 读取配置文件
	err := this.readConfig()
	if err != nil {
		LogError("Build Failed time:", time.Now())
		return
	}

	//1. 更新SVN
	this.checkOut()

	SVNDatabase := &SVNDatabase{}
	SVNDatabase.ProjectPath = this.config.ProjectHomePath
	this.version = SVNDatabase.ReadSVNVersion()
	if this.version == 0 {
		LogError("SVN Version is zero!!!", time.Now())
		return
	}
	//************先出App****************
	this.readProjectGameSetting()
	WriteFile([]byte(this.dynamicUpdateJsonContent), this.config.ProjectHomePath+"/Content/json/dynamiclist.json")

	//6. 写入代码版本号到C++（这里还需要读取Sqlite的功能，最后再加吧）
	this.writeVersionCPP()

	EncryptAndCompressAll(this.config.JsonHome)
	EncryptAndCompressAll(this.config.LuaHome)

	if this.config.IsApp {
		//在C++代码被修改之后，特别是蓝图父类，会丢失蓝图，必须重新check一次代码，所以更新完马上编译
		//这种情况必须要重编App
		this.buildFirst()
		//7. 生成App
		okApp := this.buildApp()
		if !okApp {
			return
		}
	}

	//***************再出资源************
	//2. Cook Data
	this.cookDatas()

	//3. 对比输出需要打包的文件（读取旧的文件MD5, 计算新的MD5）
	oldJson, err := ReadJson(this.config.configHome + "/version.json")
	if err != nil {
		LogError("Read Old Json Failed!")
	}
	this.calcNewMD5()
	Result := this.merge(oldJson)
	this.copyFiles(Result)

	//4. 根据对比结果生成pak
	this.writeCrypto()
	this.processReslist()

	this.buildPak()

	//文件拆分
	f := &FileSpliter{}
	f.Execute(this.config.tempPakPath, this.config.ZipSourcePath)

	CopyDir(this.config.ResOutputContentPath+"/Script", this.config.ZipSourcePath+"/Script")
	CopyDir(this.config.ResOutputContentPath+"/json", this.config.ZipSourcePath+"/json")

	//遍历pak目录，计算pak的MD5
	//如果是外网版本，增加一行记录
	//如果是内网版本，替换掉现有记录
	pakmd5 := &PakMD5{}
	//计算原始文件MD5
	pakmd5.CalcMD5(this.Reslist, this.numCPU, this.config.tempPakPath, this.config.IsPatch, this.version)
	//计算拆分文件MD5
	pakmd5.CalcMD5(this.Reslist, this.numCPU, this.config.ZipSourcePath, this.config.IsPatch, this.version)

	//输出reslist文件
	errWriteReslist := this.Reslist.Flush(this.version)
	if errWriteReslist != nil {
		this.clearWhenError()
		return
	}

	//打包pak到输出目录
	okZip := this.zipSpPackage()
	if !okZip {
		this.Reslist.Reverse()
		this.clearWhenError()
		return
	}

	//5. 写入Version.json文件
	okVersion := this.writeVersion(oldJson)
	if !okVersion {
		this.Reslist.Reverse()
		this.clearWhenError()
	}
}

func (this *GameUpdater) zipSpPackage() bool {
	zipFilePath := fmt.Sprintf("%s/%s", this.config.OutputPath, this.today)
	this.outZipFileName = fmt.Sprintf("%s/res_%s_v%d.zip", zipFilePath, this.today, this.version)

	if this.checkAvalibleZipFile(this.config.ZipSourcePath) != 0 {
		err := Zip(this.config.ZipSourcePath, this.outZipFileName)
		if err != nil {
			this.result |= 0x04
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
	this.Reslist.configHome = this.config.configHome
	this.Reslist.ZipSourcePakPath = this.config.ZipSourcePath
	this.Reslist.IsPatch = this.config.IsPatch

	this.Reslist.ReadData()
	return nil
}

func (this *GameUpdater) readConfig() error {
	this.numCPU = runtime.NumCPU()
	this.config = &Config{}
	err := this.config.readConfig()
	this.today = this.config.today
	return err
}

func (this *GameUpdater) buildFirst() {
	LogInfo("**********Begin checkout svn code**********")
	ProjectFileParam := fmt.Sprintf(`-Project=%s/%s.uproject`, this.config.ProjectHomePath, this.config.ProjectName)
	Exec(this.config.unrealBuildTool, "Development", "Win64", ProjectFileParam, "-TargetType=Editor", "-Progress", "-NoHotReloadFromIDE")
	LogInfo("**********BuildFirst Complete!**********")
}

func (this *GameUpdater) writeCrypto() {
	cryptoJsonPath := fmt.Sprintf("%s/Config/DefaultCrypto.json", this.config.ProjectHomePath)
	this.projectEncryptIniPath = fmt.Sprintf("%s/Config/DefaultCrypto.ini", this.config.ProjectHomePath)

	file, err := os.OpenFile(this.projectEncryptIniPath, os.O_RDONLY, os.ModeAppend)

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
	WriteFile([]byte(encryptLine), this.projectEncryptIniPath)

	this.isEncryptPak = true
	tempMsg := fmt.Sprintf("{\n\"EncryptionKey\":{\"Key\":\"%s\"}\n}", pakEncryptKey)
	WriteFile([]byte(tempMsg), cryptoJsonPath)
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
	LogInfo("**********Begin buildPak**********")
	if ok, _ := PathExists(this.config.tempPakPath); !ok {
		os.MkdirAll(this.config.tempPakPath, os.ModePerm)
	} else {
		os.RemoveAll(this.config.tempPakPath)
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
	defer this.wG.Done()
	for {
		select {
		case s := <-this.chanPakPath:
			this.buildSinglePak(s)
		case <-time.After(1 * time.Second):
			return
		}
	}
}

func (this *GameUpdater) buildSinglePak(pakSrcPath *SKeyValue) {
	outputPak := fmt.Sprintf("%s/%s_p_%d.pak", this.config.tempPakPath, pakSrcPath.Value, this.Reslist.GetPakIndex(pakSrcPath.Value))

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
			if strings.HasSuffix(s.relName, ".json") || strings.HasSuffix(s.relName, ".lua") {
				CopyFileAndCompress(s.sourceParentPath+"/"+s.relName, this.config.ResOutputContentPath+"/"+s.relName)
			} else {
				CopyFile(s.sourceParentPath+"/"+s.relName, this.config.ResOutputContentPath+"/"+s.relName)
			}
		case <-time.After(1 * time.Second):
			return
		}
	}
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
			oldJson.Set(Key, NewMD5.md5)
		}
	}
	return ResultList
}

func (this *GameUpdater) calcNewMD5() {
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
	defer LogInfo("calcSingle Complete")
	projectContentPath := this.config.ProjectHomePath + "/Content"
	var parentSourcePath string = projectContentPath
	for {
		select {
		case s := <-this.chanFileName:
			parentSourcePath = projectContentPath

			md5 := CalcFileMD5(s)
			var RelName string
			if strings.Contains(s, this.config.CookedPath) {
				RelName = string(s[strings.Count(this.config.CookedPath, ""):])
				parentSourcePath = this.config.CookedPath
			} else if strings.Contains(s, "json") {
				RelName = string(s[strings.Count(projectContentPath, ""):])
			} else if strings.Contains(s, "Script") {
				RelName = string(s[strings.Count(projectContentPath, ""):])
			}
			this.chanMD5 <- &SMD5{RelName, parentSourcePath, md5}
		case <-time.After(1 * time.Second):
			return
		}
	}
}

func (this *GameUpdater) readAll() {
	this.readFiles(this.config.CookedPath)
	LogInfo("**********Check Cooked Path End**********", this.config.CookedPath)
	this.readFiles(this.config.JsonHome)
	LogInfo("**********Check Json Path End**********", this.config.JsonHome)
	this.readFiles(this.config.LuaHome)
	//this.isReadAll = true
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
			LogDebug("begin calc", Name)
			this.chanFileName <- Name
		}
	}
}

func (this *GameUpdater) readProjectGameSetting() {
	this.sourceGameConfigPath = fmt.Sprintf("%s/Config/DefaultGame.ini", this.config.ProjectHomePath)

	file, err := os.OpenFile(this.sourceGameConfigPath, os.O_RDONLY, os.ModeAppend)

	if err != nil {
		LogError("Read DefaultGame.ini failed", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	this.cookGameConfigContent = ""
	this.dynamicUpdateJsonContent = "{"
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		LogInfo("Dynamic Update Config Source:", line)
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
			this.result |= 0x01
			panic(err)
		}
	}()
	WriteFile([]byte(this.cookGameConfigContent), this.sourceGameConfigPath)
	LogInfo("**********Begin Cook Content**********")
	ProjectFile := fmt.Sprintf("%s/%s.uproject", this.config.ProjectHomePath, this.config.ProjectName)
	LogFile := fmt.Sprintf("%s/log/Cook-2020.txt", this.config.ProjectHomePath)

	var pLogFormatted string = fmt.Sprintf("-abslog=%s", LogFile)
	var pPlatformFormatted string = fmt.Sprintf("-TargetPlatform=%s", this.config.CookPlatformType)

	if ok, _ := PathExists(this.config.OutputPath); !ok {
		os.MkdirAll(this.config.OutputPath, os.ModePerm)
	}

	err := ExecCookCmd(this.config.UE_EXE, ProjectFile, pPlatformFormatted, pLogFormatted)
	if err != nil {
		LogError("Build App failed.", err.Error())
	}
	LogInfo("**********Cook Content Complete**********")
}

func (this *GameUpdater) writeVersionCPP() {
	code := fmt.Sprintf(codetemp, time.Now(), this.version, pakEncryptKey)
	this.versionCppFilePath = fmt.Sprintf("%s/Source/%s/GameVersion.cpp", this.config.ProjectHomePath, this.config.ProjectName)
	WriteFile([]byte(code), this.versionCppFilePath)
}

func (this *GameUpdater) buildApp() bool {
	LogInfo("**********Begin buildApp**********")
	var tempBuildFile string = fmt.Sprintf("%s/TempBuild.cmd", this.config.BuilderHome)
	var achieveDir string
	if this.config.targetPlatform == "Android" {
		achieveDir = fmt.Sprintf("%s/%s_%s", this.config.OutputPath, this.config.targetPlatform, this.config.cookflavor)

		a1 := fmt.Sprintf("-ScriptsForProject=%s/%s.uproject", this.config.ProjectHomePath, this.config.ProjectName)
		a2 := "BuildCookRun"
		a3 := fmt.Sprintf("-project=%s/%s.uproject", this.config.ProjectHomePath, this.config.ProjectName)
		a4 := fmt.Sprintf("-archivedirectory=%s", this.config.OutputPath)

		a5 := fmt.Sprintf("-cookflavor=%s", this.config.cookflavor)
		var a6 string
		if !this.config.IsRelease {
			a6 = "-clientconfig=DebugGame"
		} else {
			a6 = "-clientconfig=Shipping"
		}

		a7 := fmt.Sprintf(`-ue4exe="%s"`, this.config.UE_EXE)
		a8 := fmt.Sprintf("-targetplatform=%s", this.config.targetPlatform)
		var argAll []string
		argAll = append(argAll, a1)
		argAll = append(argAll, a2)
		argAll = append(argAll, a3)
		argAll = append(argAll, a4)
		argAll = append(argAll, a5)
		argAll = append(argAll, a6)
		argAll = append(argAll, a7)
		argAll = append(argAll, a8)

		argAll = append(argAll, "-nocompileeditor", "-nop4", "-cook",
			"-stage", "-archive", "-package", "-compressed",
			"-SkipCookingEditorContent",
			"-pak", "-prereqs", "-nodebuginfo", "-build",
			"-utf8output", "-compile")
		var cmdString = "\"" + this.config.automationTool + "\""
		for _, a := range argAll {
			cmdString += " "
			cmdString += a
		}
		WriteFile([]byte(cmdString), tempBuildFile)
		err := ExecApp(tempBuildFile)
		if err != nil {
			LogError("Build App failed.", err.Error())
		}
	} else {
		achieveDir = fmt.Sprintf("%s/%s", this.config.OutputPath, this.config.targetPlatform)

		a1 := fmt.Sprintf("-ScriptsForProject=%s/%s.uproject", this.config.ProjectHomePath, this.config.ProjectName)
		a2 := "BuildCookRun"
		a3 := fmt.Sprintf("-project=%s/%s.uproject", this.config.ProjectHomePath, this.config.ProjectName)
		a4 := fmt.Sprintf("-archivedirectory=%s", this.config.OutputPath)

		var a5 string
		if !this.config.IsRelease {
			a5 = "-clientconfig=DebugGame"
		} else {
			a5 = "-clientconfig=Shipping"
		}

		a6 := fmt.Sprintf(`-ue4exe="%s"`, this.config.UE_EXE)
		a7 := fmt.Sprintf("-targetplatform=%s", this.config.targetPlatform)

		var argAll []string
		argAll = append(argAll, a1)
		argAll = append(argAll, a2)
		argAll = append(argAll, a3)
		argAll = append(argAll, a4)
		argAll = append(argAll, a5)
		argAll = append(argAll, a6)
		argAll = append(argAll, a7)
		argAll = append(argAll, "-nocompileeditor", "-nop4", "-cook",
			"-stage", "-archive", "-package", "-compressed",
			"-SkipCookingEditorContent",
			"-pak", "-prereqs", "-nodebuginfo", "-build",
			"-utf8output", "-compile")
		var cmdString = "\"" + this.config.automationTool + "\""
		for _, a := range argAll {
			cmdString += " "
			cmdString += a
		}
		WriteFile([]byte(cmdString), tempBuildFile)
		err := ExecApp(tempBuildFile)
		if err != nil {
			LogError("Build App failed.", err.Error())
		}
	}
	zipFilePath := fmt.Sprintf("%s/%s", this.config.OutputPath, this.today)

	rd, err := ioutil.ReadDir(achieveDir)
	if err != nil {
		os.Remove(tempBuildFile)
		this.result |= 0x02
		LogInfo("**********buildApp Failed!**********")
		return false
	}
	this.outAppFileName = ""
	for _, fi := range rd {
		if fi.IsDir() {
			continue
		}
		name := fi.Name()
		if strings.Contains(name, ".apk") || strings.Contains(name, ".ipa") {
			index := strings.LastIndex(name, ".")

			this.outAppFileName = fmt.Sprintf("%s/%s_v%d.%s", zipFilePath, name[:index], this.version, name[index+1:])
			CopyFile(achieveDir+"/"+fi.Name(), this.outAppFileName)
		}
	}

	os.Remove(tempBuildFile)
	os.RemoveAll(achieveDir)

	if ok, _ := PathExists(this.outAppFileName); ok {
		LogInfo("**********buildApp Success!**********")
		return true
	}

	LogInfo("**********buildApp Complete!**********")
	return false
}

func (this *GameUpdater) checkOut() {
	LogInfo("**********Begin checkout svn code**********")
	go this.svnCheckout()
	this.sysChan = make(chan string)
	defer close(this.sysChan)
	for {
		select {
		case this.svnMsg = <-this.sysChan:
			if this.svnMsg == "end" {
				return
			} else if this.svnMsg == "ok" {
				LogInfo("Build Application. update svn complete!\r\n")
				return
			}
		case <-time.After(5 * time.Second):
			fmt.Println("Now updating svn ..." + this.svnMsg)
		}
	}

}

func (this *GameUpdater) svnCheckout() {
	LogInfo("The next step is to update code")
	this.sysChan <- "updating code"

	ok, _ := PathExists(this.config.ProjectName)
	if !ok {
		ExecSVNCmd("svn", "checkout", this.config.svnCode, this.config.ProjectName)
	}

	ExecSVNCmd("svn", "cleanup", this.config.ProjectName)
	ExecSVNCmd("svn", "update", "--force", this.config.ProjectName, "--accept", "theirs-full")
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
	errWrite := WriteFile(Bytes, this.config.configHome+"/version.json")
	if errWrite != nil {
		LogError("Write version.json Error!", errWrite)
		return false
	}
	return true
}

func (this *GameUpdater) clear() {
	//删除缓存文件
	os.RemoveAll(this.config.ResOutputContentPath)
	os.RemoveAll(this.config.tempPakPath)
	os.RemoveAll(this.config.ZipSourcePath)
	os.RemoveAll(this.config.JsonHome)
	os.RemoveAll(this.config.LuaHome)

	os.Remove(this.projectEncryptIniPath)
	os.Remove(this.versionCppFilePath)
	os.Remove(this.sourceGameConfigPath)

	timePassed := time.Now().Unix() - this.beginTime.Unix()

	LogInfo("---------------------------------")
	LogInfo("-------Build Complete!!!---------")
	LogInfo("-------time:", time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05"))
	LogInfo("-------Total seconds:", timePassed)
	LogInfo("---------------------------------")
}

func (this *GameUpdater) sendReport() {
	tempURL := `http://192.168.0.10/zentaopms/www/sendmsg.php?user=%s&msg=%s`

	ip, err := getLocalIP()
	var msgtemp string
	msgtemp = fmt.Sprintf("%s:%s-%s. 参数: 外网包=%v, 发布版=%v.", ip, this.config.ProjectName, this.config.targetPlatform,
		this.config.IsPatch, this.config.IsRelease)
	if this.outAppFileName != "" {
		msgtemp += fmt.Sprintf(" App:%s", this.outAppFileName[len(this.config.BuilderHome)+1:])
	}
	if this.outZipFileName != "" {
		msgtemp += fmt.Sprintf(" zip:%s", this.outZipFileName[len(this.config.BuilderHome)+1:])
	} else {
		msgtemp += fmt.Sprintf(" zip:%s", "没有")
	}

	if this.result&0x01 == 0x01 {
		msgtemp += fmt.Sprintf(" Error", "Cook失败")
	} else if this.result&0x02 == 0x02 {
		msgtemp += fmt.Sprintf(" Error", "App失败")
	} else if this.result&0x02 == 0x04 {
		msgtemp += fmt.Sprintf(" Error", "Zip失败")
	}

	timePassed := time.Now().Unix() - this.beginTime.Unix()
	msgtemp += fmt.Sprintf(". 耗时:%ds", timePassed)
	//生成client 参数为默认
	client := &http.Client{}

	//生成要访问的url
	url := fmt.Sprintf(tempURL, this.config.teamMembers, msgtemp)

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
}

func (this *GameUpdater) clearWhenError() {
	os.Remove(this.outAppFileName)
	os.Remove(this.outZipFileName)
}

func GetInt(Data map[string]interface{}, Key string) int64 {
	TempStr, ok := Data[Key]
	if !ok {
		return 0
	}
	sValue, ok1 := TempStr.(json.Number)
	if ok1 {
		result, _ := sValue.Int64()
		return result
	}

	sValue1, ok2 := TempStr.(int64)
	if ok2 {
		return sValue1
	}
	return 0
}

func GetString(Data map[string]interface{}, Key string) string {
	TempStr, ok := Data[Key]
	if !ok {
		return ""
	}
	return TempStr.(string)
}

func MD5(pData []byte) string {
	md5 := md5.Sum(pData)
	return string(md5[:])
}

func ReadFileData(filePath string) ([]byte, error) {
	datas, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return datas, nil
}

func CalcFileMD5(filePath string) string {
	bytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return ""
	}
	md5 := md5.Sum(bytes)
	return fmt.Sprintf("%x", md5)
}

func ReadJson(filePath string) (*simplejson.Json, error) {
	bytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Printf("%v\n", err)
		return simplejson.New(), err
	}
	result, err := simplejson.NewJson(bytes)

	if err != nil {
		fmt.Printf("%v\n", err)
		return simplejson.New(), err
	}
	return result, nil
}

func getLocalIP() (ipv4 string, err error) {
	var (
		addrs   []net.Addr
		addr    net.Addr
		ipNet   *net.IPNet // IP地址
		isIpNet bool
	)
	// 获取所有网卡
	if addrs, err = net.InterfaceAddrs(); err != nil {
		return
	}
	// 取第一个非lo的网卡IP
	for _, addr = range addrs {
		// 这个网络地址是IP地址: ipv4, ipv6

		if ipNet, isIpNet = addr.(*net.IPNet); isIpNet && !ipNet.IP.IsLoopback() {
			// 跳过IPV6
			if ipNet.IP.To4() != nil {
				ipv4 = ipNet.IP.String() // 192.168.1.1

				LogInfo(ipv4)
				if strings.HasPrefix(ipv4, "192.168.") {
					return
				}
			}
		}
	}

	err = errors.New("No IP")
	return
}
