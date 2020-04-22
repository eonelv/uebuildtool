// jsonandlua
package game

import (
	"file"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

var wG *sync.WaitGroup

var chanWattingEncyptFileName chan string

func EncryptAndCompressAll(SrcFile string) {
	chanWattingEncyptFileName = make(chan string, runtime.NumCPU())
	defer close(chanWattingEncyptFileName)

	go writeCompressFileToChannel(SrcFile)

	wG = &sync.WaitGroup{}
	wG.Add(runtime.NumCPU())
	for i := 0; i < runtime.NumCPU(); i++ {
		go go_EncryptAndCompressFile(SrcFile)
	}
	wG.Wait()
}

func writeCompressFileToChannel(SrcFile string) {
	filepath.Walk(SrcFile, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if err != nil {
			return err
		}
		path = strings.ReplaceAll(path, `\`, "/")

		if !info.IsDir() {
			chanWattingEncyptFileName <- path
		}
		return err
	})
}

func go_EncryptAndCompressFile(DestFile string) {
	defer wG.Done()
	for {
		select {
		case s := <-chanWattingEncyptFileName:
			file.EncryptFile(s)
		case <-time.After(1 * time.Second):
			return
		}
	}
}
