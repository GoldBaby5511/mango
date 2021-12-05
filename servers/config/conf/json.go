package conf

import (
	"encoding/json"
	"io/ioutil"
	lconf "xlddz/core/conf"
	"xlddz/core/log"
	n "xlddz/core/network"
	aConfig "xlddz/servers/config/agollo/env/config"
)

var Server struct {
	TCPClientAddr string
	TCPAddr       string
	AppID         uint32
	AppName       string
	MaxConnNum    int
	ConsolePort   int
	ScreenPrint   bool
	UseApollo     bool `default:"false" json:"UseApollo"`
	Config        aConfig.AppConfig
	CommonServers []ApolloConfig
}

type ApolloConfig struct {
	Appid      string `json:"appId"`
	Cluster    string `json:"cluster"`
	Ns         string `json:"namespaceName"`
	Ip         string `json:"ip"`
	ServerType uint32 `json:"servertype"`
	ServerId   uint32 `json:"serverid"`
}

func init() {
	data, err := ioutil.ReadFile("conf/config.json")
	if err != nil {
		log.Fatal("jsonconf", "%v", err)
	}
	err = json.Unmarshal(data, &Server)
	if err != nil {
		log.Fatal("jsonconf", "%v", err)
	}

	lconf.ConsolePort = Server.ConsolePort
	lconf.AppName = Server.AppName
	lconf.AppID = Server.AppID
	lconf.AppType = n.AppConfig
	lconf.ListenOnAddress = Server.TCPAddr
	log.Info("jsonconf", "配置文件载入成功%v", Server)
}
