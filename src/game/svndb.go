// svndb
package game

import (
	. "core"
	. "db/sqlitedb"
	"fmt"
)

type SVNDatabase struct {
	ProjectPath string
	isInit      bool
}

func (this *SVNDatabase) ReadSVNVersion() int64 {
	if !this.isInit {
		CreateDBMgr(fmt.Sprintf("%s/.svn/wc.db", this.ProjectPath))
		this.isInit = true
	}
	rows, err := DBMgr.Query("select revision from NODES;")
	if err != nil || len(rows) == 0 {
		LogError("get version form db error", err)
		return 0
	}
	for _, row := range rows {
		return row.GetInt64("revision")
	}
	return 0
}
