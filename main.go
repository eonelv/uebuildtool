// uebuildtool project main.go
package main

import (
	. "game"
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
	//f := &FileSpliter{}
	//f.Execute(`E:\golang\uebuildtool\APackages\1`, `E:\golang\uebuildtool\APackages\2`)
}
