// serverconfig
package cfg

import (
	. "core"
	. "file"
	"fmt"
	"os"
	"path/filepath"
	"utils"
)

type ServerConfig struct {
	BuilderHome string
	Host        string
}

const serverconfig string = `{
	"host":"192.168.1.19"
}`

func (this *ServerConfig) ReadServerConfig() error {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err == nil {
		this.BuilderHome = dir
	} else {
		this.BuilderHome = "E:/golang/uebuildtool"
	}

	configHome := fmt.Sprintf("%s/config", this.BuilderHome)
	PathExistAndCreate(configHome)
	configFileName := configHome + "/serverconfig.json"

	oldJson, err := utils.ReadJson(configFileName)
	if err != nil {
		LogError("Read config Json Failed! 1.")
		WriteFile([]byte(serverconfig), configFileName)
		return err
	}
	ConfigDatas := oldJson.MustMap()
	this.Host = utils.GetString(ConfigDatas, "host")
	return nil
}
