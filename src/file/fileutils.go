package file

import (
	"archive/zip"
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ngcod.com/utils"

	. "ngcod.com/core"
)

type CopyDirTask struct {
	BaseMultiThreadTask
	channel           chan *MKeyValue
	TargetNamePostfix string
}

type RenameDirTask struct {
	BaseMultiThreadTask
	channel           chan string
	TargetNamePostfix string
}

func (this *CopyDirTask) CreateChan() {
	this.channel = make(chan *MKeyValue)
	LogDebug("create channel by CopyDirTask")
}

func (this *CopyDirTask) CloseChan() {
	close(this.channel)
	LogDebug("close channel by CopyDirTask")
}

func (this *CopyDirTask) WriteToChannel(SrcFileDir string) {
	filepath.Walk(SrcFileDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if err != nil {
			return err
		}
		path = strings.ReplaceAll(path, `\`, "/")

		if !info.IsDir() {
			RelName := path[strings.Count(SrcFileDir, ""):]
			this.channel <- &MKeyValue{path, RelName}
		}
		return err
	})
}

func (this *CopyDirTask) ProcessTask(DestFileDir string) {
	for {
		select {
		case s := <-this.channel:
			utils.CopyFile(s.Key, DestFileDir+"/"+s.Value+this.TargetNamePostfix)
		case <-time.After(1 * time.Second):
			return
		}
	}
}

func (this *RenameDirTask) CreateChan() {
	this.channel = make(chan string)
}

func (this *RenameDirTask) CloseChan() {
	close(this.channel)
}

func (this *RenameDirTask) WriteToChannel(SrcFileDir string) {
	filepath.Walk(SrcFileDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if err != nil {
			return err
		}
		path = strings.ReplaceAll(path, `\`, "/")

		if !info.IsDir() {
			this.channel <- path
		}
		return err
	})
}

func (this *RenameDirTask) ProcessTask(DestFileDir string) {
	for {
		select {
		case s := <-this.channel:
			os.Rename(s, s+this.TargetNamePostfix)
		case <-time.After(1 * time.Second):
			return
		}
	}
}

func Zip(srcFile string, destZip string) error {
	zipfile, err := os.Create(destZip)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	filepath.Walk(srcFile, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		path = strings.ReplaceAll(path, `\`, "/")
		header.Name = strings.TrimPrefix(path, filepath.Dir(srcFile)+"/")

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		header.Name = header.Name[len(srcFile):]
		if header.Name[0] == '/' {
			header.Name = header.Name[1:]
		}
		if header.Name == "" {
			return err
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(writer, file)
		}
		return err
	})

	return err
}

func ReadJsonFile(SrcFile string) {
	datas, err := ioutil.ReadFile(SrcFile)
	if err != nil {
		return
	}

	b := bytes.NewReader(datas)
	var out bytes.Buffer
	r, _ := zlib.NewReader(b)
	io.Copy(&out, r)

	datas = out.Bytes()
	encrypt := &Encrypt{}
	encrypt.InitEncrypt(183, 46, 15, 43, 0, 88, 232, 90)
	encrypt.Encrypt(datas, 0, len(datas), true)
	LogDebug(string(datas[:]))
}

func EncryptFile(SrcFile string) error {
	datas, err := ioutil.ReadFile(SrcFile)
	if err != nil {
		return err
	}

	//加密
	encrypt := &Encrypt{}
	encrypt.InitEncrypt(183, 46, 15, 43, 0, 88, 232, 90)
	encrypt.Encrypt(datas, 0, len(datas), true)

	os.Remove(SrcFile)

	//创建目标文件
	fileWrite, err := os.OpenFile(SrcFile, os.O_WRONLY|os.O_CREATE, os.ModePerm)

	if err != nil {
		fmt.Println("Create err:", err)
		return err
	}
	defer fileWrite.Close()
	fileWrite.Write(datas)
	return err
}

func CompressFile(SrcFile string) error {
	datas, err := ioutil.ReadFile(SrcFile)
	if err != nil {
		return err
	}

	var in bytes.Buffer

	writer := zlib.NewWriter(&in)
	writer.Write(datas)
	writer.Close()

	//创建目标文件
	fileWrite, err := os.OpenFile(SrcFile, os.O_WRONLY|os.O_CREATE, os.ModePerm)

	if err != nil {
		fmt.Println("Create err:", err)
		return err
	}
	defer fileWrite.Close()
	fileWrite.Write(in.Bytes())
	return err
}

func CopyFileAndCompress(SrcFile string, DestFile string) error {
	DestFile = strings.ReplaceAll(DestFile, "\\", "/")
	index := strings.LastIndex(DestFile, "/")
	ParentPath := DestFile[:index]
	os.MkdirAll(ParentPath, os.ModePerm)

	datas, err := ioutil.ReadFile(SrcFile)
	if err != nil {
		return err
	}

	var in bytes.Buffer

	writer := zlib.NewWriter(&in)
	writer.Write(datas)
	writer.Close()

	//创建目标文件
	fileWrite, err := os.OpenFile(DestFile, os.O_WRONLY|os.O_CREATE, os.ModePerm)

	if err != nil {
		fmt.Println("Create err:", err)
		return err
	}
	defer fileWrite.Close()
	fileWrite.Write(in.Bytes())
	return err
}

func ExecCookCmd(cmdStr string, args ...string) error {
	args = append(args, "-run=Cook", "-fileopenlog", "-unversioned", "-skipeditorcontent",
		"-stdout", "-CrashForUAT", "-unattended", "-NoLogTimes", "-UTF8Output")

	var testString string = cmdStr

	for _, a := range args {
		testString += " "
		testString += a
	}
	fmt.Println(testString)

	err := utils.Exe_Cmd(cmdStr, false, args...)
	return err
}

func ExecSVNCmd(cmdStr string, args ...string) error {
	args = append(args, "--username=liwei", "--password=liwei!@#")

	err := utils.Exe_Cmd(cmdStr, true, args...)
	return err
}

func ExecPakCmd(cmdStr string, args ...string) error {
	args = append(args, "-encrypt", "-encryptindex", "-compress")

	err := utils.Exe_Cmd(cmdStr, false, args...)
	return err
}

func ExecApp(cmdStr string, args ...string) error {

	//cmd := exec.Command(cmdStr, args...)
	//cmd.Stdout = os.Stdout
	//err := cmd.Run()
	//return err

	err := utils.Exe_Cmd(cmdStr, false, args...)
	return err
}
