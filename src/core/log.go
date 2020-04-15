package core

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

type ELogger struct {
}

var (
	logPath = "log"
)

func initLog() {
	var logger ELogger
	log.SetOutput(&logger)

	os.MkdirAll(logPath+"/I", os.ModeDir)
	os.MkdirAll(logPath+"/D", os.ModeDir)
	os.MkdirAll(logPath+"/E", os.ModeDir)
}

func LogInfo(v ...interface{}) {
	output("I", v...)
}

func LogDebug(v ...interface{}) {

	output("D", v...)
}

func LogError(v ...interface{}) {
	output("E", v...)
}

func output(keyworlds string, v ...interface{}) {
	_, file, line, ok := runtime.Caller(2)
	var shortFile string
	if !ok {
		shortFile = "???"
		line = 0
	} else {
		index := strings.LastIndex(file, "/")
		shortFile = string([]byte(file)[index+1:])
	}
	log.Printf("%v [%v(%d)] : %v \n", keyworlds, shortFile, line, strings.TrimRight(fmt.Sprintln(v...), "\n"))
}

func getLogFile(t time.Time, name string) string {
	return fmt.Sprintf("%s/%s/%d-%d-%d.log", logPath, string(name), t.Year(), t.Month(), t.Day())
}

func (this *ELogger) Write(p []byte) (int, error) {

	os.Stdout.Write(p)

	subName := string(p[20:21])

	fileName := getLogFile(time.Now(), subName)
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, os.FileMode(0644))
	if err != nil {
		os.Stdout.Write([]byte(fmt.Sprintf("can not open log file:%s r:%v\n", fileName, err)))
		return 0, err
	}

	return file.Write(p)
}
