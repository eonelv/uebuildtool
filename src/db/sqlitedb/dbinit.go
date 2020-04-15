package sqlitedb

import (
	. "core"
	"io/ioutil"
	"os"
	"strings"
)

func CreateDB() bool {
	LogInfo("Create DB")
	fileDest, errDest := os.OpenFile("config/data.db", os.O_WRONLY|os.O_CREATE, os.ModeAppend)
	if errDest != nil {
		LogError(errDest)
		return false
	}
	defer fileDest.Close()
	return InitDB()
}

func InitDB() bool {
	LogInfo("create database and init db")
	var result bool
	result = CreateDBMgr("data.db")
	result = initSQL()
	return result
}

func initSQL() bool {
	userFile := "config/init.sql"

	file, err := os.OpenFile(userFile, os.O_RDONLY, os.ModeAppend)
	if err != nil {
		return false
	}
	buffer, errFileData := ioutil.ReadAll(file)
	if errFileData != nil {
		LogError("read init.sql error")
		return false
	}
	var sqlall string = string(buffer)
	lineArray := strings.Split(sqlall, ";")

	var errSQL error
	var errorlog string = "SQL init error: "
	for _, line := range lineArray {
		_, errSQL = DBMgr.Execute(line)
		if errSQL != nil {
			errorlog += line
		}
	}
	LogError(errorlog)
	return true
}
