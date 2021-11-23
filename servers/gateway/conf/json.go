package conf

import (
	"encoding/json"
	"io/ioutil"
	lconf "xlddz/core/conf"
	"xlddz/core/log"
)

var Server struct {
	AppName       string
	AppType       uint32
	AppID         uint32
	TCPAddr       string
	TCPClientAddr string
}

func init() {
	data, err := ioutil.ReadFile("conf/gateway.json")
	if err != nil {
		log.Fatal("jsonconf", "QQQ%v", err)
	}
	err = json.Unmarshal(data, &Server)
	if err != nil {
		log.Fatal("jsonconf", "%v", err)
	}

	lconf.AppName = Server.AppName
	lconf.AppType = Server.AppType
	lconf.AppID = Server.AppID
	log.Info("jsonconf", "配置文件载入成功%v", Server)
}
