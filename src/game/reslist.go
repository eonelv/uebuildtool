// reslist
package game

import (
	. "core"
	. "file"
	"fmt"
	"sync"
	"utils"

	simplejson "github.com/bitly/go-simplejson"
)

type Reslist struct {
	configHome       string
	ZipSourcePakPath string
	IsPatch          bool
	IsEncrypt        bool
	ReslistData      *simplejson.Json
	ReslistMap       map[string]interface{}
	PakVersionData   *simplejson.Json
	PakVersionMap    map[string]interface{}
	mutex            sync.RWMutex
}

func (this *Reslist) ReadData() {
	this.mutex = sync.RWMutex{}

	reslistJson, err := utils.ReadJson(this.configHome + "/reslist.json")
	if err != nil {
		reslistJson = simplejson.New()
	}
	pakVersionJson, err := utils.ReadJson(this.configHome + "/pakversion.json")
	if err != nil {
		pakVersionJson = simplejson.New()
	}
	this.ReslistData = reslistJson
	this.ReslistMap = reslistJson.MustMap()
	this.PakVersionData = pakVersionJson
	this.PakVersionMap = pakVersionJson.MustMap()
}

func (this *Reslist) GetPakIndex(key string) int64 {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	index := utils.GetInt(this.PakVersionMap, key)

	if index == 0 {
		index = 5
	}
	if this.IsPatch {
		index++
	}
	this.PakVersionData.Set(key, index)
	this.PakVersionMap[key] = index
	return index
}

func (this *Reslist) Flush(version int64) error {
	//TODO 测试阶段都用这个版本号
	version = 24841
	Bytes, err := this.ReslistData.MarshalJSON()
	if err != nil {
		LogError("Read Json Data Error!", err)
		return err
	}
	//跟随外网包一起发布到资源服务器

	OutBytes := CloneBytes(Bytes)
	copy(Bytes, OutBytes)
	//加密
	if this.IsEncrypt {
		encrypt := &Encrypt{}
		encrypt.InitEncrypt(183, 46, 15, 43, 0, 88, 232, 90)
		encrypt.Encrypt(OutBytes, 0, len(OutBytes), true)
	}

	err = WriteFile(OutBytes, fmt.Sprintf("%s/reslist_%d.json", this.ZipSourcePakPath, version))
	if err != nil {
		LogError("Write reslist.json Error!", err)
		return err
	}
	//本地缓存
	//先备份
	reslistFileName := fmt.Sprintf("%s/reslist.json", this.configHome)
	backupReslistFileName := fmt.Sprintf("%s/reslist_back.json", this.configHome)
	CopyFile(reslistFileName, backupReslistFileName)

	err = WriteFile(Bytes, reslistFileName)
	if err != nil {
		LogError("Writereslist.json Error!", err)
		return err
	}

	Bytes, err1 := this.PakVersionData.MarshalJSON()
	if err1 != nil {
		LogError("Read Json Data Error!", err)
		return err1
	}
	//跟随外网包一起发布到资源服务器-游戏拉取
	err = WriteFile(Bytes, fmt.Sprintf("%s/pakversion_%d.json", this.ZipSourcePakPath, version))
	if err != nil {
		LogError("Write reslist.json Error!", err)
		return err
	}
	//本地缓存
	pakversionFileName := fmt.Sprintf("%s/pakversion.json", this.configHome)
	backuppakversionName := fmt.Sprintf("%s/pakversion_back.json", this.configHome)
	CopyFile(pakversionFileName, backuppakversionName)

	err = WriteFile(Bytes, pakversionFileName)
	if err != nil {
		LogError("Write reslist.json Error!", err)
		return err
	}
	return nil
}

func (this *Reslist) Reverse() {
	reslistFileName := fmt.Sprintf("%s/reslist.json", this.configHome)
	backupReslistFileName := fmt.Sprintf("%s/reslist_back.json", this.configHome)
	CopyFile(backupReslistFileName, reslistFileName)

	pakversionFileName := fmt.Sprintf("%s/pakversion.json", this.configHome)
	backuppakversionName := fmt.Sprintf("%s/pakversion_back.json", this.configHome)
	CopyFile(backuppakversionName, pakversionFileName)
}
