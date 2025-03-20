package conf

import (
	"encoding/json"
	aConfig "github.com/apolloconfig/agollo/v4/env/config"
	"io/ioutil"
	"mango/pkg/log"
)

const (
	DefaultConfigFile string = "configs/config/config.json"
)

var Server struct {
	UseApollo     bool `default:"false" json:"UseApollo"`
	LoggerAddr    string
	Config        aConfig.AppConfig
	CommonServers []ApolloConfig
}

type ApolloConfig struct {
	Appid      string `json:"appID"`
	Cluster    string `json:"cluster"`
	Ns         string `json:"namespaceName"`
	Ip         string `json:"ip"`
	ServerType uint32 `json:"serverType"`
	ServerId   uint32 `json:"serverID"`
}

func init() {
	data, err := ioutil.ReadFile(DefaultConfigFile)
	if err != nil {
		log.Fatal("", "%v", err)
	}
	err = json.Unmarshal(data, &Server)
	if err != nil {
		log.Fatal("", "%v", err)
	}

	log.Info("", "配置文件载入成功%v", Server)
}
