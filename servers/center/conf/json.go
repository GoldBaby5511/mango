package conf

import (
	"encoding/json"
	"io/ioutil"
	lconf "xlddz/core/conf"
	"xlddz/core/log"
	n "xlddz/core/network"
)

var Server struct {
	AppName string
	AppID   uint32
	TCPAddr string
}

func init() {
	data, err := ioutil.ReadFile("conf/center.json")
	if err != nil {
		log.Fatal("jsonconf", "%v", err)
	}
	err = json.Unmarshal(data, &Server)
	if err != nil {
		log.Fatal("jsonconf", "%v", err)
	}

	lconf.AppName = Server.AppName
	lconf.AppType = n.AppCenter
	lconf.AppID = Server.AppID
	lconf.ListenOnAddress = Server.TCPAddr
	log.Info("jsonconf", "配置文件载入成功%v", Server)
}
