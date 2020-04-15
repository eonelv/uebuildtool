// pakmd5
package game

import (
	. "core"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	simplejson "github.com/bitly/go-simplejson"
)

type ResFileInfo struct {
	name       string
	pname      string
	md5        string
	size       int64
	pakVersion int64
	ResVesion  int64
}

type SFileInfo struct {
	name string
	size int64
}

type ResFilePair struct {
	Key      string
	FileInfo *ResFileInfo
}

type PakMD5 struct {
	numCPU  int
	path    string
	Reslist *Reslist
	isPatch bool
	version int64

	MD5          map[string]*ResFileInfo
	chanMD5      chan *ResFilePair
	chanFileName chan *SFileInfo
	wG           *sync.WaitGroup
	isInit       bool
}

func (this *PakMD5) CalcMD5(Reslist *Reslist, numCPU int, path string, isPatch bool, verison int64) {
	LogInfo("**********Begin calcNewMD5**********", path)
	this.numCPU = numCPU
	this.Reslist = Reslist
	this.path = path
	this.isPatch = isPatch
	this.version = verison

	if !this.isInit {
		this.MD5 = make(map[string]*ResFileInfo)
	}
	this.chanFileName = make(chan *SFileInfo, numCPU)
	defer close(this.chanFileName)

	go this.readAll(this.path)

	this.chanMD5 = make(chan *ResFilePair, this.numCPU)
	defer close(this.chanMD5)
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
	LogInfo("**********CalcNewMD5 Complete**********")

	this.writeReslist()
}

func (this *PakMD5) writeReslist() {
	var pname string
	var gJson *simplejson.Json
	var pJson *simplejson.Json
	//var taskJson []interface{}
	var tempTaskJson *simplejson.Json
	var ok bool
	for Key := range this.MD5 {
		pname = ""
		partIndex := strings.LastIndex(Key, "_part")
		var grandName string
		var pnameIndex int
		if partIndex != -1 {
			d := this.MD5[Key]
			itemC := simplejson.New()
			itemC.Set("size", d.size)
			itemC.Set("md5", d.md5)
			itemC.Set("name", Key)

			pname = Key[:partIndex]
			pnameIndex = strings.LastIndex(Key, "_p_")
			if pnameIndex != -1 {
				grandName = Key[:pnameIndex]
			} else {
				grandName = Key
			}
			gJson, ok = this.Reslist.ReslistData.CheckGet(grandName)
			//没找到最上级的json
			if !ok {
				gJson = simplejson.New()
				pJson = simplejson.New()
				taskJson := []*simplejson.Json{}
				pJson.Set("tasks", taskJson)

				tempTaskJson, ok = pJson.CheckGet("tasks")
				taskJson1 := tempTaskJson.MustArray()
				taskJson1 = append(taskJson1, itemC)
				pJson.Set("tasks", taskJson1)

				gJson.Set(pname, pJson)

				LogDebug("1. Task Length", len(taskJson), Key)
				this.Reslist.ReslistData.Set(grandName, gJson)
			} else {
				pJson, ok = gJson.CheckGet(pname)
				if !ok {
					pJson = simplejson.New()

					taskJson := []*simplejson.Json{}
					pJson.Set("tasks", taskJson)

					tempTaskJson, ok = pJson.CheckGet("tasks")
					taskJson1 := tempTaskJson.MustArray()
					taskJson1 = append(taskJson1, itemC)
					pJson.Set("tasks", taskJson1)

					LogDebug("2. Task Length", len(taskJson1), Key)
					gJson.Set(pname, pJson)
				} else {
					tempTaskJson, ok = pJson.CheckGet("tasks")
					if !ok {
						taskJson := []*simplejson.Json{}
						pJson.Set("tasks", taskJson)

						tempTaskJson, ok = pJson.CheckGet("tasks")
						taskJson1 := tempTaskJson.MustArray()
						taskJson1 = append(taskJson1, itemC)
						pJson.Set("tasks", taskJson1)

						LogDebug("3. Task Length", len(taskJson1), Key)
					} else {
						taskJson := tempTaskJson.MustArray()
						if len(taskJson) == 0 {
							taskJson = append(taskJson, itemC)
						} else {
							taskJson = append(taskJson, itemC)
						}
						pJson.Set("tasks", taskJson)
						LogDebug("4. Task Length", len(taskJson), Key)
					}
				}
			}

		} else {
			pnameIndex = strings.LastIndex(Key, "_p_")
			if pnameIndex != -1 {
				pname = Key[:pnameIndex]
			} else {
				pname = Key
			}
			gJson, ok = this.Reslist.ReslistData.CheckGet(pname)

			item := simplejson.New()
			d := this.MD5[Key]
			item.Set("name", d.name)

			//TODO 好像没什么用
			item.Set("pname", d.pname)
			item.Set("version", d.ResVesion)
			if d.pakVersion != 0 {
				item.Set("pakversion", d.pakVersion)
			}

			item.Set("size", d.size)
			item.Set("md5", d.md5)
			//如果没有找到pName
			if !ok {
				gJson = simplejson.New()
			}
			gJson.Set(d.name, item.MustMap())
			this.Reslist.ReslistData.Set(pname, gJson.MustMap())
		}
		/*
			pnameIndex = strings.LastIndex(Key, "_p_")
			if pnameIndex != -1 {
				pname = Key[:pnameIndex]
			} else {
				pname = Key
			}
			json, ok := this.Reslist.ReslistData.CheckGet(pname)
			item := simplejson.New()
			d := this.MD5[Key]
			item.Set("name", d.name)

			//TODO 好像没什么用
			item.Set("pname", d.pname)
			item.Set("version", d.ResVesion)
			if d.pakVersion != 0 {
				item.Set("pakversion", d.pakVersion)
			}

			item.Set("size", d.size)
			item.Set("md5", d.md5)
			//如果没有找到pName
			if !ok {
				json = &simplejson.Json{}
				this.Reslist.ReslistData.Set(pname, pJson)
			} else {
				json.Set(d.name, item)
				this.Reslist.ReslistData.Set(pname, json)
			}
			if !ok {
				itemArray := []*simplejson.Json{item}
				this.Reslist.ReslistData.Set(pname, itemArray)
			} else {
				itemArray := json.MustArray()
				if this.isPatch && d.pakVersion != 0 {
					itemArray = append(itemArray, item)
				} else {
					itemArray[0] = item
				}
				this.Reslist.ReslistData.Set(pname, itemArray)
			}
		*/
	}
}

func (this *PakMD5) writeNewMD5(completeChan chan bool) {
	for {
		select {
		case FileMD5 := <-this.chanMD5:
			this.MD5[FileMD5.Key] = FileMD5.FileInfo
		case <-completeChan:
			return
		}
	}
}

func (this *PakMD5) calcSingle() {
	defer this.wG.Done()
	for {
		select {
		case s := <-this.chanFileName:
			md5 := CalcFileMD5(s.name)
			RelName := string(s.name[strings.Count(this.path, ""):])
			index := strings.LastIndex(RelName, "_p_")
			if index == -1 {
				index = strings.Count(RelName, "") - 1
			}
			parentName := RelName[:index]

			fileInfo := &ResFileInfo{}
			fileInfo.name = RelName
			fileInfo.pname = parentName

			fileInfo.pakVersion = GetInt(this.Reslist.PakVersionMap, parentName)
			fileInfo.ResVesion = this.version
			fileInfo.md5 = md5
			fileInfo.size = s.size

			this.chanMD5 <- &ResFilePair{RelName, fileInfo}
		case <-time.After(2 * time.Second):
			return
		}
	}
}

func (this *PakMD5) readAll(path string) {
	rd, err := ioutil.ReadDir(path)
	if err != nil {
		LogError(err)
		return
	}
	for _, fi := range rd {
		if fi.IsDir() {
			this.readAll(path + "/" + fi.Name())
		} else {
			Name := path + "/" + fi.Name()
			this.chanFileName <- &SFileInfo{Name, fi.Size()}
		}
	}
}
