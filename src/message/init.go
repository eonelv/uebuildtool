package message

import (
	"fmt"
)

func init() {
	fmt.Println("message.init")
	registerNetMsgLogin()
	registerNetMsgBuild()
	registerNetMsgConnection()
	registerNetMsgRegisterServer()
	registerNetMsgBindServer()
	registerNetMsgNetReport()
	registerNetMsgFileInfo()
	registerNetMsgUnRegister()
}
