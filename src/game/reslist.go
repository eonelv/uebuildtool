// reslist
package game

import (
	"fmt"
	"sync"

	. "ngcod.com/core"
	"ngcod.com/utils"

	simplejson "github.com/bitly/go-simplejson"
)

type Reslist struct {
	configHome       string
	ZipSourcePakPath string
	reslistPath      string
	pakversionPath   string
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

	this.reslistPath = fmt.Sprintf("%s/reslist.json", this.configHome)
	this.pakversionPath = fmt.Sprintf("%s/pakversion.json", this.configHome)

	reslistJson, err := utils.ReadJson(this.reslistPath)
	if err != nil {
		reslistJson = simplejson.New()
	}
	pakVersionJson, err := utils.ReadJson(this.pakversionPath)
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
	} else {
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

	err = utils.WriteFile(OutBytes, fmt.Sprintf("%s/reslist_%d.json", this.ZipSourcePakPath, version))
	if err != nil {
		LogError("Write reslist.json Error!", err)
		return err
	}
	//本地缓存
	//先备份
	backupReslistFileName := fmt.Sprintf("%s/reslist_back.json", this.configHome)
	utils.CopyFile(this.reslistPath, backupReslistFileName)

	err = utils.WriteFile(Bytes, this.reslistPath)
	if err != nil {
		LogError("Write reslist.json Error!", err)
		return err
	}

	if !this.IsPatch {
		return nil
	}
	Bytes, err1 := this.PakVersionData.MarshalJSON()
	if err1 != nil {
		LogError("Read Json Data Error!", err)
		return err1
	}
	//跟随外网包一起发布到资源服务器-游戏拉取
	err = utils.WriteFile(Bytes, fmt.Sprintf("%s/pakversion_%d.json", this.ZipSourcePakPath, version))
	if err != nil {
		LogError("Write reslist.json Error!", err)
		return err
	}
	//本地缓存
	backuppakversionName := fmt.Sprintf("%s/pakversion_back.json", this.configHome)
	utils.CopyFile(this.pakversionPath, backuppakversionName)

	err = utils.WriteFile(Bytes, this.pakversionPath)
	if err != nil {
		LogError("Write reslist.json Error!", err)
		return err
	}
	return nil
}

func (this *Reslist) Reverse() {

	backupReslistFileName := fmt.Sprintf("%s/reslist_back.json", this.configHome)
	utils.CopyFile(backupReslistFileName, this.reslistPath)

	if this.IsPatch {
		backuppakversionName := fmt.Sprintf("%s/pakversion_back.json", this.configHome)
		utils.CopyFile(backuppakversionName, this.pakversionPath)
	}
}
