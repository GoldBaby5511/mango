package conf

import (
	"encoding/json"
	"io/ioutil"
	lconf "xlddz/core/conf"
	"xlddz/core/log"
)

var Server struct {
	TCPClientAddr string
	TCPAddr       string
	AppType       uint32
	AppID         uint32
	AppName       string
}

func init() {
	data, err := ioutil.ReadFile("conf/login.json")
	if err != nil {
		log.Fatal("jsonconf", "%v", err)
	}
	err = json.Unmarshal(data, &Server)
	if err != nil {
		log.Fatal("jsonconf", "%v", err)
	}

	lconf.AppName = Server.AppName
	lconf.AppType = Server.AppType
	lconf.AppID = Server.AppID
	lconf.ListenOnAddress = Server.TCPAddr
	lconf.CenterAddr = Server.TCPClientAddr
	log.Info("jsonconf", "配置文件载入成功%v", Server)
}
