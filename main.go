// uebuildtool project main.go
package main

import (
	. "game"
	//. "file"
	"runtime"
)

type FileMD5 struct {
	Key   string
	Value string
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	var gameUpdater *GameUpdater = &GameUpdater{}
	gameUpdater.DoUpdate()
	//ReadJsonFile(`E:\golang\uebuildtool\dynamiclist.json`)
	//CopyFileAndCompress(`C:\UE4\Projects\ENGGame\Content\json\dynamiclist.json`, `E:\golang\uebuildtool\dynamiclist.json`)
}
