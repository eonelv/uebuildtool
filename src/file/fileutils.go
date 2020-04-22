package file

import (
	"archive/zip"
	"bufio"
	"bytes"
	"compress/zlib"
	. "core"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type MKeyValue struct {
	Key   string
	Value string
}

var wG *sync.WaitGroup

var chanWattingCopyFileName chan *MKeyValue

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

func PathExistAndCreate(path string) {
	if ok, _ := PathExists(path); !ok {
		os.MkdirAll(path, os.ModePerm)
	}
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func WriteFile(data []byte, filePath string) error {
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	defer f.Close()

	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	_, err = f.Write(data)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	return nil
}

func CopyDir(SrcFile string, DestFile string) {
	chanWattingCopyFileName = make(chan *MKeyValue, runtime.NumCPU())
	defer close(chanWattingCopyFileName)

	go writeCopyFileToChannel(SrcFile)

	wG = &sync.WaitGroup{}
	wG.Add(runtime.NumCPU())
	for i := 0; i < runtime.NumCPU(); i++ {
		go go_CopyFile(DestFile)
	}
	wG.Wait()
}

func writeCopyFileToChannel(SrcFile string) {
	filepath.Walk(SrcFile, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if err != nil {
			return err
		}
		path = strings.ReplaceAll(path, `\`, "/")

		if !info.IsDir() {
			RelName := path[strings.Count(SrcFile, ""):]
			chanWattingCopyFileName <- &MKeyValue{path, RelName}
		}
		return err
	})
}

func go_CopyFile(DestFile string) {
	defer wG.Done()
	for {
		select {
		case s := <-chanWattingCopyFileName:
			CopyFile(s.Key, DestFile+"/"+s.Value)
		case <-time.After(1 * time.Second):
			return
		}
	}
}

func CopyFile(SrcFile string, DestFile string) error {
	fileRead, err := os.Open(SrcFile)
	if err != nil {
		fmt.Println("Open err:", err)
		return err
	}
	defer fileRead.Close()

	index := strings.LastIndex(DestFile, "/")
	ParentPath := DestFile[:index]
	os.MkdirAll(ParentPath, os.ModePerm)

	//创建目标文件
	fileWrite, err := os.OpenFile(DestFile, os.O_WRONLY|os.O_CREATE, os.ModePerm)

	if err != nil {
		fmt.Println("Create err:", err)
		return err
	}
	defer fileWrite.Close()

	buf := make([]byte, 1024)
	for {
		n, err := fileRead.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		if _, err := fileWrite.Write(buf[:n]); err != nil {
			return err
		}
	}
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

	err := exe_Inner(cmdStr, args...)
	return err
}

func ExecSVNCmd(cmdStr string, args ...string) error {
	args = append(args, "--username=liwei", "--password=liwei!@#")

	err := exe_Inner(cmdStr, args...)
	return err
}

func ExecPakCmd(cmdStr string, args ...string) error {
	args = append(args, "-encrypt", "-encryptindex", "-compress")

	err := exe_Inner(cmdStr, args...)
	return err
}

func ExecApp(cmdStr string, args ...string) error {

	//cmd := exec.Command(cmdStr, args...)
	//cmd.Stdout = os.Stdout
	//err := cmd.Run()
	//return err

	err := exe_Inner(cmdStr, args...)
	return err
}

func Exec(cmdStr string, args ...string) error {

	var testString string = cmdStr

	for _, a := range args {
		testString += " "
		testString += a
	}
	fmt.Println(testString)

	err := exe_Inner(cmdStr, args...)
	return err
}

func exe_Inner(cmdStr string, args ...string) error {
	cmd := exec.Command(cmdStr, args...)
	output := make(chan []byte, 10240)
	defer close(output)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	defer stdoutPipe.Close()

	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() { // 命令在执行的过程中, 实时地获取其输出
			LogInfo(string(scanner.Bytes()))
		}
	}()

	if err := cmd.Run(); err != nil {
		panic(err)
	}
	return err
}
