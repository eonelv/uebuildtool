//创建并行的加密任务
//继承BaseMultiThreadTask
//采用统一的多协程任务处理模板 - 见multitask.go
package game

import (
	"file"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "ngcod.com/core"
)

type EncryptJsonTask struct {
	BaseMultiThreadTask
	channel chan string
}

func (this *EncryptJsonTask) CreateChan() {
	LogDebug("create channel by EncryptJsonTask")
	this.channel = make(chan string)
}

func (this *EncryptJsonTask) CloseChan() {
	LogDebug("close channel by EncryptJsonTask")
	close(this.channel)
}

func (this *EncryptJsonTask) WriteToChannel(SrcFileDir string) {
	filepath.Walk(SrcFileDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if err != nil {
			return err
		}
		path = strings.ReplaceAll(path, `\`, "/")

		if !info.IsDir() {
			this.channel <- path
		}
		return err
	})
}

func (this *EncryptJsonTask) ProcessTask(DestFileDir string) {
	//LogDebug("processTask by EncryptJsonTask")
	for {
		select {
		case s := <-this.channel:
			file.EncryptFile(s)
		case <-time.After(1 * time.Second):
			return
		}
	}
}
