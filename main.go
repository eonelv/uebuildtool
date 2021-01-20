// uebuildtool project main.go
package main

import (
	"fmt"
	"mynet"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	. "ngcod.com/core"

	"ngcod.com/utils"
)

/*
type FileMD5 struct {
	Key   string
	Value string
}
*/

var Sender *TCPSender

var localIP string

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	//var gameUpdater *game.GameUpdater = &game.GameUpdater{}
	//gameUpdater.DoUpdate()
	localIP, _ = utils.GetLocalIP()
	go startFileServer()
	start()
}

func startFileServer() {

	http.HandleFunc("/", HomeHandler)
	p, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	http.Handle("/log/", mynet.FileServer(http.Dir(p)))

	http.Handle("/APack_Android/", mynet.FileServer(http.Dir(p)))
	http.Handle("/APack_iOS/", mynet.FileServer(http.Dir(p)))

	err := http.ListenAndServe(":80", nil)
	if err != nil {
		fmt.Println(err)
	}
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	webContext :=
		`<font size="6"><h1 align="center">%s编译输出</h1>
		<ul>
		<h1><li><a href="http://%s/APack_Android">Android包</a></li></h1>
		<h1><li><a href="http://%s/APack_iOS">iOS包</a></li></h1>
		<h1><li><a href="http://%s/log">日志</a></li></h1>
		</ul></font">`

	w.Write([]byte(fmt.Sprintf(webContext, localIP, localIP, localIP, localIP)))
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
