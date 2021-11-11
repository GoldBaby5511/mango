package conf

import (
	"encoding/json"
	"io/ioutil"
	"xlddz/core/log"
)

var Server struct {
	TCPAddr       string
	TCPClientAddr string
	LogServerAddr string
	AppType       uint32
	AppID         uint32
	AppName       string
	MaxConnNum    int
	ConsolePort   int
	WorkDB        string
	ScreenPrint   bool
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
	log.SetScreenPrint(Server.ScreenPrint)
	log.Info("jsonconf", "配置文件载入成功%v", Server)
}
