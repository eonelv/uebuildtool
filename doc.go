// uebuildtool project doc.go

/*
uebuildtool document
*/
package main

/*
go多线程复制数据
channel1

第一个协程往channel1写入数据
go this.writeCopyFileToChannel(...)

创建WaitGroup用于同步
this.wGFile = &sync.WaitGroup{}
this.wGFile.Add(runtime.NumCPU())

创建CPU数量的协程用于处理数据
for i := 0; i < runtime.NumCPU(); i++ {
	go this.go_writeFile(DestFileDir)
}
等待所有协程处理完毕
this.wGFile.Wait()

go_writeFile执行完就调用一次
this.wGFile.Done()

defer this.wGFile.Done()
for {
	select {
	case mi := <-this.chanSubFileInfo://从channel1读取数据
		this.writeFile(DestFileDir, mi)
	case <-time.After(2 * time.Second):
		return
	}
}

------------
2020.06.03
------------
关于多线程任务，最新版已经修改了处理方式
使用MultiThreadTask统一处理，逻辑不变
1. 使用的时候定义自己的channel
type EncryptJsonTask struct {
	BaseMultiThreadTask
	channel chan string
}

2. 实现接口的函数
func (this *EncryptJsonTask) WriteToChannel(SrcFileDir string)
func (this *EncryptJsonTask) ProcessTask(DestFileDir string)

func (this *EncryptJsonTask) CreateChan()
func (this *EncryptJsonTask) CloseChan()
*/

/*
基于UE版本号4.24.3
*/
