// filespliter
package file

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "ngcod.com/core"
	"ngcod.com/utils"
)

const SizeEach int64 = 5 * 1024 * 1024

type MFileInfo struct {
	Name    string
	RelName string
	Size    int64
}

type FileSplitInfo struct {
	name    string
	relName string
	start   int64
	length  int64
	part    int32
}

type FileSpliterTask struct {
	BaseMultiThreadTask
	channel chan *FileSplitInfo
}

func (this *FileSpliterTask) CreateChan() {
	this.channel = make(chan *FileSplitInfo)
}

func (this *FileSpliterTask) CloseChan() {
	close(this.channel)
}

func (this *FileSpliterTask) WriteToChannel(SrcFileDir string) {
	filepath.Walk(SrcFileDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if err != nil {
			return err
		}
		path = strings.ReplaceAll(path, `\`, "/")

		if !info.IsDir() {
			RelName := path[strings.Count(SrcFileDir, ""):]

			size := info.Size()
			subFileCount := int32(size / SizeEach)
			leftByts := size - int64(subFileCount)*SizeEach
			if leftByts > 512*1024 {
				subFileCount += 1
			} else if leftByts > 0 && subFileCount == 0 {
				subFileCount = 1
			}

			var i int32
			if subFileCount == 1 {
				spliteInfo := &FileSplitInfo{path, RelName, 0, 0, 0}

				spliteInfo.start = 0
				spliteInfo.length = size
				spliteInfo.part = -1

				this.channel <- spliteInfo
			} else {
				for i = 0; i < subFileCount; i++ {
					spliteInfo := &FileSplitInfo{path, RelName, 0, 0, 0}
					length := SizeEach
					if i == subFileCount-1 {
						length = size - SizeEach*int64(i)
					}
					spliteInfo.start = int64(i) * SizeEach
					spliteInfo.length = length
					spliteInfo.part = i

					this.channel <- spliteInfo
				}
			}

		}
		return err
	})
}

func (this *FileSpliterTask) ProcessTask(DestFileDir string) {
	for {
		select {
		case mi := <-this.channel:
			this.writeFile(DestFileDir, mi)
		case <-time.After(10 * time.Second):
			return
		}
	}
}

func (this *FileSpliterTask) writeFile(DestFileDir string, splitInfo *FileSplitInfo) error {
	fileRead, err := os.Open(splitInfo.name)
	if err != nil {
		fmt.Println("Open err:", err)
		return err
	}
	defer fileRead.Close()

	os.MkdirAll(DestFileDir, os.ModePerm)

	if splitInfo.part == -1 {
		return utils.CopyFile(splitInfo.name, fmt.Sprintf("%s/%s", DestFileDir, splitInfo.relName))
	}
	DestFile := fmt.Sprintf("%s/%s_part%d", DestFileDir, splitInfo.relName, splitInfo.part)
	//创建目标文件
	fileWrite, err := os.OpenFile(DestFile, os.O_WRONLY|os.O_CREATE, os.ModePerm)

	if err != nil {
		fmt.Println("Create err:", err)
		return err
	}
	defer fileWrite.Close()

	buf := make([]byte, 1024)
	var writeLength int64 = 0
	fileRead.Seek(splitInfo.start, 0)
	for {
		n, err := fileRead.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		if writeLength+int64(n) >= splitInfo.length {
			//LogDebug("最后的字节", "length=", splitInfo.length, "WriteLength=", writeLength, "n=", n)
			n = int(splitInfo.length - writeLength)

			if n <= 0 {
				return nil
			}
			if _, err := fileWrite.Write(buf[:n]); err != nil {
				return err
			}
			writeLength += int64(n)
			break
		} else {
			if _, err := fileWrite.Write(buf[:n]); err != nil {
				return err
			}
			writeLength += int64(n)
		}

	}
	return err
}
