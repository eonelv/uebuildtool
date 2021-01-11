// msgfileinfo
package message

import (
	. "def"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	. "ngcod.com/core"
)

func registerNetMsgFileInfo() {
	isSuccess := RegisterMsgFunc(CMD_FILE, createNetMsgNetFileInfo)
	LogInfo("Registor message", CMD_FILE)
	if !isSuccess {
		LogError("Registor CMD_FILE faild")
	}
}

func createNetMsgNetFileInfo(cmdData *Command) NetMsg {
	netMsg := &MsgFile{}
	netMsg.CreateByBytes(cmdData.Message.([]byte))
	return netMsg
}

const (
	QUERY_FILE  uint16 = 1
	LIST_FILE   uint16 = 2
	REMOVE_FILE uint16 = 3
)

type MsgFileInfo struct {
	IsDir    bool
	Size     int64
	FileName [1024]byte
}

type MsgFile struct {
	ProjectID ObjectID
	Action    uint16
	Num       uint16
	PData     []byte //MsgFileInfo
}

func (this *MsgFile) GetNetBytes() ([]byte, bool) {
	return GenNetBytes(uint16(CMD_FILE), reflect.ValueOf(this))
}

func (this *MsgFile) CreateByBytes(bytes []byte) (bool, int) {
	return Byte2Struct(reflect.ValueOf(this), bytes)
}

func (this *MsgFile) Process(p interface{}) {
	Sender, ok := p.(*TCPSender)
	if !ok {
		return
	}

	if this.Action == QUERY_FILE {
		this.query(Sender)
	} else if this.Action == REMOVE_FILE {
		this.remove(Sender)
	}
}

func (this *MsgFile) query(Sender *TCPSender) {
	msgFileInfo := &MsgFileInfo{}
	Byte2Struct(reflect.ValueOf(msgFileInfo), this.PData)
	if !msgFileInfo.IsDir {
		return
	}

	fileName := Byte2String(msgFileInfo.FileName[:])
	fileName = strings.TrimSpace(fileName)

	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	dir = strings.ReplaceAll(dir, `\`, `/`)
	if fileName == "null" {
		this.sendDefault(Sender)
		return
	}

	count := 0
	var totalData []byte = []byte{}

	fileName = dir + "/" + fileName
	rd, _ := ioutil.ReadDir(fileName)

	for _, fi := range rd {
		count++
		msgFileInfoResult := &MsgFileInfo{}
		fullName := fileName + "/" + fi.Name()

		fullName = fullName[len(dir)+1:]
		CopyArray(reflect.ValueOf(&msgFileInfoResult.FileName), []byte(fullName))
		if fi.IsDir() {
			msgFileInfoResult.IsDir = true
		} else {
			msgFileInfoResult.IsDir = false
		}
		msgFileInfoResult.Size = fi.Size()

		data, _ := Struct2Bytes(reflect.ValueOf(msgFileInfoResult))
		totalData = append(totalData, data...)
	}
	this.PData = totalData
	this.Action = LIST_FILE
	this.Num = uint16(count)
	Sender.Send(this)
}

func (this *MsgFile) sendDefault(Sender *TCPSender) {
	count := 3
	var totalData []byte = []byte{}

	msgFileInfoResult := &MsgFileInfo{}
	CopyArray(reflect.ValueOf(&msgFileInfoResult.FileName), []byte("APack_Android"))
	msgFileInfoResult.IsDir = true
	data, _ := Struct2Bytes(reflect.ValueOf(msgFileInfoResult))
	totalData = append(totalData, data...)

	msgFileInfoResult = &MsgFileInfo{}
	CopyArray(reflect.ValueOf(&msgFileInfoResult.FileName), []byte("APack_iOS"))
	msgFileInfoResult.IsDir = true
	data, _ = Struct2Bytes(reflect.ValueOf(msgFileInfoResult))
	totalData = append(totalData, data...)

	msgFileInfoResult = &MsgFileInfo{}
	CopyArray(reflect.ValueOf(&msgFileInfoResult.FileName), []byte("log"))
	msgFileInfoResult.IsDir = true
	data, _ = Struct2Bytes(reflect.ValueOf(msgFileInfoResult))
	totalData = append(totalData, data...)

	this.PData = totalData
	this.Action = LIST_FILE
	this.Num = uint16(count)
	if count != 0 {
		Sender.Send(this)
	}
}

func (this *MsgFile) remove(Sender *TCPSender) {
	num := int(this.Num)
	index := 0
	var isNeedReQuery bool
	var parentFolder string
	var parentFileInfo *MsgFileInfo
	for i := 0; i < num; i++ {
		msgFileInfo := &MsgFileInfo{}
		ok, idx := Byte2Struct(reflect.ValueOf(msgFileInfo), this.PData[index:])
		if !ok {
			return
		}

		index += idx
		fileName := Byte2String(msgFileInfo.FileName[:])

		if parentFolder == "" {
			indexParent := strings.LastIndex(fileName, `/`)
			parentFolder = fileName[:indexParent]
		}

		if msgFileInfo.IsDir {
			err := os.RemoveAll(fileName)
			if err == nil {
				isNeedReQuery = true
			}
		} else {
			err := os.Remove(fileName)
			if err == nil {
				isNeedReQuery = true
			}
		}
	}
	if isNeedReQuery {
		parentFileInfo = &MsgFileInfo{}
		CopyArray(reflect.ValueOf(&parentFileInfo.FileName), []byte(parentFolder))
		parentFileInfo.IsDir = true

		this.PData, _ = Struct2Bytes(reflect.ValueOf(parentFileInfo))
		this.Num = 1
		this.Action = 1
		this.query(Sender)
	}
}
