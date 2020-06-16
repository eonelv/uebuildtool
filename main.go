// uebuildtool project main.go
package main

import (
	. "core"
	"mynet"
	"runtime"
	"strings"
	"utils"
)

/*
type FileMD5 struct {
	Key   string
	Value string
}
*/

var Sender *TCPSender

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	//var gameUpdater *game.GameUpdater = &game.GameUpdater{}
	//gameUpdater.DoUpdate()
	start()
}

func start() {
	macAddress := utils.GetMacAddrs()
	var isAuth bool
	for _, Address := range macAddress {
		UpperAddress := strings.ToUpper(Address)
		UpperAddress = strings.ReplaceAll(UpperAddress, ":", "-")
		LogError(UpperAddress)
		if _, ok := utils.AllowMacAddress[UpperAddress]; ok {
			isAuth = true
			break
		}
	}
	if !isAuth {
		LogError("No Authorization")
		return
	}

	mynet.Connect()
}
