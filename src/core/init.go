package core

import (
	"fmt"
)

func init() {
	fmt.Println("core.init")
	initLog()
	initRegisterMsgFunc()
}
