package sqlitedb

import (
	SQL "database/sql"
	"io/ioutil"
	"os"
	"strings"

	"ngcod.com/db/sqlitedb"

	. "ngcod.com/core"
)

var DBMgr sqlitedb.DataBaseMgr

func CreateDBMgr(path string) bool {
	LogInfo("db path:", path)
	db, err := SQL.Open("sqlite3", path)
	//	db, err := SQL.Open("mysql", "ouyang:ouyang@tcp(192.168.0.10:3306)/zentao?charset=utf8")
	if err != nil {
		LogError("DataBase Connect Error %s \n", err.Error())
		return CreateDB()
	}
	DBMgr = sqlitedb.DataBaseMgr{}
	DBMgr.DBServer = db
	return true
}

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
