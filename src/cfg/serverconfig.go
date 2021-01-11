// serverconfig
package cfg

import (
	"fmt"
	"os"
	"path/filepath"

	. "ngcod.com/core"
	"ngcod.com/utils"
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
	utils.PathExistAndCreate(configHome)
	configFileName := configHome + "/serverconfig.json"

	oldJson, err := utils.ReadJson(configFileName)
	if err != nil {
		LogError("Read config Json Failed! 1.")
		utils.WriteFile([]byte(serverconfig), configFileName)
		return err
	}
	ConfigDatas := oldJson.MustMap()
	this.Host = utils.GetString(ConfigDatas, "host")
	return nil
}
