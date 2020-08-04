// uebuildtool project main.go
package main

import (
	. "core"
	"fmt"
	"mynet"
	"net/http"
	"os"
	"path/filepath"
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

	go startFileServer()
	start()
}

func startFileServer() {
	p, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	http.Handle("/log/", http.FileServer(http.Dir(p)))

	http.Handle("/APack_Android/", http.FileServer(http.Dir(p)))
	http.Handle("/APack_iOS/", http.FileServer(http.Dir(p)))

	err := http.ListenAndServe(":5009", nil)
	if err != nil {
		fmt.Println(err)
	}
}

func start() {
	macAddress := utils.GetMacAddrs()
	var isAuth bool
	for _, Address := range macAddress {
		UpperAddress := strings.ToUpper(Address)
		UpperAddress = strings.ReplaceAll(UpperAddress, ":", "-")

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
