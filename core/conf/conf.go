package conf

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"xlddz/core/util"
)

var (
	LenStackBuf = 4096

	// skeleton conf
	GoLen              = 10000
	TimerDispatcherLen = 10000
	AsynCallLen        = 10000
	ChanRPCLen         = 10000

	//服务基础属性
	AppInfo BaseInfo
	//AppName         string
	//AppID           uint32
	//AppType         uint32
	//ListenOnAddress string
	//CenterAddr      string
)

type BaseInfo struct {
	AppName         string
	AppID           uint32
	AppType         uint32
	ListenOnAddress string
	CenterAddr      string
}

func ParseCmdArgs() {
	if AppInfo.AppName != "" {
		data, err := ioutil.ReadFile(fmt.Sprintf("conf/%v.json", AppInfo.AppName))
		if err == nil {
			err = json.Unmarshal(data, &AppInfo)
		}
	}
	if v, ok := util.ParseArgs("/AppID"); ok {
		AppInfo.AppID = v
	}
	if v, ok := util.ParseArgs("/AppType"); ok {
		AppInfo.AppType = v
	}
	if v, ok := util.ParseArgs("/DockerRun"); ok && v == 1 {
		addr := strings.Split(AppInfo.CenterAddr, ":")
		if len(addr) == 2 {
			AppInfo.CenterAddr = "center:" + addr[1]
		}
	}
}
