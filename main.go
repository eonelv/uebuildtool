// uebuildtool project main.go
package main

import (
	//"core"
	"game"

	//"file"
	"runtime"
)

type FileMD5 struct {
	Key   string
	Value string
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	var gameUpdater *game.GameUpdater = &game.GameUpdater{}
	gameUpdater.DoUpdate()

	//var task core.MultiThreadTask = &file.CopyDirTask{}
	//core.ExecTask(task, `C:\UE4\Projects\ENGGame\Content\json`, `D:\lv\1`)

	//ReadJsonFile(`E:\golang\uebuildtool\dynamiclist.json`)
	//CopyFileAndCompress(`C:\UE4\Projects\ENGGame\Content\json\dynamiclist.json`, `E:\golang\uebuildtool\dynamiclist.json`)
}
