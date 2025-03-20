package conf

import (
	"encoding/json"
	"errors"
	"fmt"
	"mango/pkg/log"
	n "mango/pkg/network"
	"mango/pkg/util"
	"io/ioutil"
	"os"
	"strconv"
)

const (
	ArgAppName         string = "-Name"
	ArgAppType         string = "-Type"
	ArgAppId           string = "-Id"
	ArgCenterAddr      string = "-CenterAddr"
	ArgListenOnAddr    string = "-ListenOnAddr"
	ArgDockerRun       string = "-DockerRun"
	ArgDefaultBasePort string = "-DefaultBasePort"

	//服务状态
	AppStateNone              = 0
	AppStateStarting          = 1
	AppStateRunning           = 2
	AppStateMaintenance       = 4
	AppStateMaintenanceFinish = 8
	AppStateOffline           = 16
)

var (
	LenStackBuf               = 4096
	GoLen                     = 10000
	TimerDispatcherLen        = 10000
	AsynCallLen               = 10000
	ChanRPCLen                = 10000
	DefaultBasePort    uint32 = 10000
	AppInfo            BaseInfo
	ApplogDir          string
)

type BaseInfo struct {
	Name         string
	Type         uint32
	Id           uint32
	ListenOnAddr string
	CenterAddr   string
}

func LoadBaseConfig(name string) {
	AppInfo.Name = name
	if AppInfo.Name != "" {
		data, err := ioutil.ReadFile(fmt.Sprintf("configs/%s/%s.json", AppInfo.Name, AppInfo.Name))
		if err == nil {
			err = json.Unmarshal(data, &AppInfo)
		}
	}
	args := os.Args
	if v, ok := util.ParseArgsString(ArgAppName, args); ok {
		AppInfo.Name = v
	}
	if v, ok := util.ParseArgsUint32(ArgAppType, args); ok {
		AppInfo.Type = v
	}
	if v, ok := util.ParseArgsUint32(ArgAppId, args); ok {
		AppInfo.Id = v
	}
	if v, ok := util.ParseArgsString(ArgCenterAddr, args); ok {
		AppInfo.CenterAddr = v
	}
	if v, ok := util.ParseArgsString(ArgListenOnAddr, args); ok {
		AppInfo.ListenOnAddr = v
	}
	if v, ok := util.ParseArgsUint32(ArgDefaultBasePort, args); ok {
		DefaultBasePort = v
	}
	if AppInfo.ListenOnAddr == "" {
		AppInfo.ListenOnAddr = fmt.Sprintf("0.0.0.0:%d", DefaultBasePort+AppInfo.Id)
	}
	if AppInfo.CenterAddr == "" && AppInfo.Type != n.AppCenter && AppInfo.Type != n.AppLogger {
		AppInfo.CenterAddr = fmt.Sprintf("127.0.0.1:%v", DefaultBasePort+50)
		log.Debug("", "使用默认地址,CenterAddr=%v", AppInfo.CenterAddr)
	}
	if RunInLocalDocker() {
		AppInfo.CenterAddr = "center:" + strconv.Itoa(util.GetPortFromIPAddress(AppInfo.CenterAddr))
	}

	if util.PortInUse(util.GetPortFromIPAddress(AppInfo.ListenOnAddr)) {
		log.Fatal("初始化", "端口[%v]已被占用,请检查运行环境", util.GetPortFromIPAddress(AppInfo.ListenOnAddr))
		return
	}

	if AppInfo.Name == "" || AppInfo.Type == 0 || AppInfo.Id == 0 || AppInfo.ListenOnAddr == "" ||
		(AppInfo.CenterAddr == "" && AppInfo.Type != n.AppCenter && AppInfo.Type != n.AppLogger) {
		log.Fatal("初始化", "初始参数异常,请检查,AppInfo=%v", AppInfo)
		return
	}

	//创建日志目录
	if err := makeAppLogDir(); err != nil {
		log.Fatal("初始化", "创建日志目录失败,err=%v", err)
		return
	}

	log.Debug("", "基础属性,%v,log目录=%v", AppInfo, ApplogDir)
}

func RunInLocalDocker() bool {
	args := os.Args
	if v, ok := util.ParseArgsUint32(ArgDockerRun, args); ok && v == 1 {
		return true
	}
	return false
}

func makeAppLogDir() error {
	curPath, err := util.GetCurrentPath()
	if err != nil {
		return errors.New("获取当前路径失败")
	}
	pathName := fmt.Sprintf("%slog/%s%d/", curPath, AppInfo.Name, AppInfo.Id)
	err = os.MkdirAll(pathName, os.ModePerm)
	if err != nil {
		return errors.New("文件路径创建失败")
	}
	ApplogDir = pathName
	return nil
}
