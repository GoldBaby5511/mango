package business

import (
	"fmt"
	"math/rand"
	"os"
	"path"
	"time"
	g "xlddz/core/gate"
	"xlddz/core/log"
	n "xlddz/core/network"
	"xlddz/core/util"
	"xlddz/protocol/logger"
)

var (
	appConnData map[n.AgentClient]*connectionData = make(map[n.AgentClient]*connectionData)
)

const (
	connected  int = 1
	registered int = 2
)

//连接数据
type appRegInfo struct {
	appType  uint32
	appId    uint32
	regToken string
	appName  string
	curStep  int
}

type connectionData struct {
	a             n.AgentClient
	regInfo       appRegInfo
	lastHeartbeat int64
	baseFile      *os.File
	pathname      string
}

func init() {
	g.MsgRegister(&logger.LogReq{}, n.CMDLogger, uint16(logger.CMDID_Logger_IDLogReq), handleLogReq)
	g.MsgRegister(&logger.LogFlush{}, n.CMDLogger, uint16(logger.CMDID_Logger_IDLogFlush), handleLogFlush)
	g.EventRegister(g.ConnectSuccess, connectSuccess)
	g.EventRegister(g.Disconnect, disconnect)
}

func connectSuccess(args []interface{}) {
	log.Info("连接", "来了老弟,当前连接数=%d", len(appConnData))
	a := args[g.AgentIndex].(n.AgentClient)
	if v, ok := appConnData[a]; ok {
		log.Error("连接", "异常,重复连接?,%d,%d", v.regInfo.appType, v.regInfo.appId)
		a.Close()
		return
	}
	appConnData[a] = &connectionData{a: a, regInfo: appRegInfo{curStep: connected}}
}

func disconnect(args []interface{}) {
	log.Info("连接", "告辞中,当前连接数=%d", len(appConnData))
	a := args[g.AgentIndex].(n.AgentClient)
	if v, ok := appConnData[a]; ok {
		regKey := makeRegKey(v.regInfo.appType, v.regInfo.appId)
		log.Info("连接", "再见,appName=%v,appType=%d,appId=%d,regKey=%d",
			v.regInfo.appName, v.regInfo.appType, v.regInfo.appId, regKey)

		//关闭文件
		if v.baseFile != nil {
			v.baseFile.Close()
		}
		delete(appConnData, a)
	} else {
		log.Error("连接", "异常,没有注册的连接?")
	}
}

func handleLogReq(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*logger.LogReq)
	//m := args[n.DataIndex].(*logger.LogReq)
	a := args[n.AgentIndex].(n.AgentClient)

	//连接存在判断
	if _, ok := appConnData[a]; !ok {
		log.Error("连接", "异常,没有注册的连接?")
		a.Close()
		return
	}

	log.Debug("写日志", "收到,Level=%v,Appname=%v,Content=%s", m.GetLogLevel(), m.GetSrcAppname(), string(m.GetContent()))

	if appConnData[a].baseFile == nil {
		pathname := ""
		curPath, err := util.GetCurrentPath()
		if err == nil {
			pathname = curPath + "log/" + m.GetSrcAppname() + "/"
			_, err := os.Stat(pathname)
			if err != nil && os.IsNotExist(err) {
				err = os.MkdirAll(pathname, os.ModePerm)
				if err != nil {
					pathname = ""
				}
			}
		}
		if pathname == "" {
			return
		}

		file, err := createNewLogFile(pathname, m.GetSrcAppname(), m.GetSrcApptype(), m.GetSrcAppid())
		if err != nil {
			return
		}

		appConnData[a].baseFile = file
		appConnData[a].pathname = pathname

		token := fmt.Sprintf("gb%x%x%x", rand.Int(), time.Now().UnixNano(), rand.Int())
		appConnData[a].regInfo = appRegInfo{m.GetSrcApptype(), m.GetSrcAppid(), token, m.GetSrcAppname(), registered}

	} else {
		//60M分割文件 1024*1024*60
		fi, err := appConnData[a].baseFile.Stat()
		if err == nil && fi.Size() >= 1024*1024*60 {
			file, err := createNewLogFile(appConnData[a].pathname, m.GetSrcAppname(), m.GetSrcApptype(), m.GetSrcAppid())
			if err == nil {
				appConnData[a].baseFile.Close()
				appConnData[a].baseFile = file
			}
		}
	}

	//再次判断
	if appConnData[a].baseFile == nil {
		return
	}

	//构造内容
	now := time.Now()
	timeStr := fmt.Sprintf("[local:%v-%02d-%02d %02d:%02d:%02d.%09d]",
		now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond())
	now = time.Unix(0, int64(m.GetTimeMs()*1000000))
	timeStr = timeStr + fmt.Sprintf("[remote:%v-%02d-%02d %02d:%02d:%02d.%09d]\t",
		now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond())
	className := fmt.Sprintf("[%s]", string(m.GetClassName()))
	fileInfo := fmt.Sprintf("\t[%s][lineNO:%d]", m.GetFileName(), m.GetLineNo())
	logStr := timeStr + log.GetLogLevelStr(int(m.GetLogLevel())) + className + string(m.GetContent()) + fileInfo

	appConnData[a].baseFile.WriteString(logStr + "\n")
}

func handleLogFlush(args []interface{}) {

}

func makeRegKey(appType, appId uint32) uint64 {
	return uint64(appType)<<32 | uint64(appId)
}

func createNewLogFile(pathName, appName string, appType, appId uint32) (*os.File, error) {
	now := time.Now()
	filename := fmt.Sprintf("%s_%d_%d_%d%02d%02d_%02d_%02d_%02d.log",
		appName,
		appType,
		appId,
		now.Year(),
		now.Month(),
		now.Day(),
		now.Hour(),
		now.Minute(),
		now.Second())
	file, err := os.Create(path.Join(pathName, filename))
	if err != nil {
		return nil, err
	}
	return file, nil
}
