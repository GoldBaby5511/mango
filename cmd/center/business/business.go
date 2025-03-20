package business

import (
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	"mango/api/center"
	lconf "mango/pkg/conf"
	"mango/pkg/conf/apollo"
	g "mango/pkg/gate"
	"mango/pkg/log"
	n "mango/pkg/network"
	"mango/pkg/util"
	"math/rand"
	"time"
)

var (
	appConnData = make(map[n.AgentClient]*connectionData)
	appRegData  = make(map[uint64]*connectionData)
)

//连接数据
type connectionData struct {
	a                  n.AgentClient
	appInfo            lconf.BaseInfo
	appState           int
	regToken           string
	stateDescription   string
	totalHearDelayTime int64
	lastHeartbeat      int64
	httpAddr           string
	rpcAddr            string
}

func init() {
	g.MsgRegister(&center.RegisterAppReq{}, n.AppCenter, uint16(center.CMDCenter_IDAppRegReq), handleRegisterAppReq)
	g.MsgRegister(&center.HeartBeatReq{}, n.AppCenter, uint16(center.CMDCenter_IDHeartBeatReq), handleHeartBeatReq)
	g.EventRegister(g.ConnectSuccess, connectSuccess)
	g.EventRegister(g.Disconnect, disconnect)
	g.CallBackRegister(g.CbConfigChangeNotify, configChangeNotify)
}

func connectSuccess(args []interface{}) {
	a := args[g.AgentIndex].(n.AgentClient)
	log.Info("连接", "来了老弟,当前连接数=%d,name=%v", len(appConnData), a.AgentInfo().AppName)
	if v, ok := appConnData[a]; ok {
		log.Error("连接", "异常,重复连接?,%d,%d", v.appInfo.Type, v.appInfo.Id)
		a.Close()
		return
	}
	appConnData[a] = &connectionData{a: a, appInfo: lconf.BaseInfo{Name: a.AgentInfo().AppName}}
}

func disconnect(args []interface{}) {
	log.Info("连接", "告辞中,当前连接数=%d", len(appConnData))
	a := args[g.AgentIndex].(n.AgentClient)
	if v, ok := appConnData[a]; ok {
		regKey := util.MakeUint64FromUint32(v.appInfo.Type, v.appInfo.Id)
		log.Info("连接", "再见,appType=%d,appId=%d,regKey=%d", v.appInfo.Type, v.appInfo.Id, regKey)
		broadcastAppState(v.appInfo.Type, v.appInfo.Id, lconf.AppStateOffline)
		delete(appConnData, a)
		delete(appRegData, regKey)
	} else {
		log.Error("连接", "异常,没有注册的连接?")
	}
}

func configChangeNotify(args []interface{}) {
	key := args[apollo.KeyIndex].(apollo.ConfKey)
	//value := args[apollo.ValueIndex].(apollo.ConfValue)

	switch key.Key {
	case "服务列表":
		log.Debug("", "收到服务列表")
	default:
		break
	}
}

func handleRegisterAppReq(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*center.RegisterAppReq)
	a := args[n.AgentIndex].(n.AgentClient)

	//连接存在判断
	if _, ok := appConnData[a]; !ok {
		log.Error("连接", "异常,没有注册的连接?")
		a.Close()
		return
	}

	regKey := util.MakeUint64FromUint32(m.GetAppType(), m.GetAppId())
	if v, ok := appRegData[regKey]; ok {
		if v.regToken != m.GetReregToken() {

			resultMsg := fmt.Sprintf("该服务已注册,appType=%v,appId=%v,regKey=%v",
				m.GetAppType(), m.GetAppId(), regKey)
			log.Warning("连接", resultMsg)

			var rsp center.RegisterAppRsp
			rsp.RegResult = proto.Uint32(1)
			rsp.ReregToken = proto.String(resultMsg)
			rsp.CenterId = proto.Uint32(lconf.AppInfo.Id)
			a.SendData(n.AppCenter, uint32(center.CMDCenter_IDAppRegRsp), &rsp)

			a.Close()
			return
		} else {
			log.Info("注册", "关闭旧的")
			v.a.Close()
		}
	} else {
		appRegData[regKey] = appConnData[a]
	}
	//信息存储
	token := fmt.Sprintf("gb%x%x%x", rand.Int(), time.Now().UnixNano(), rand.Int())
	appRegData[regKey].appInfo = lconf.BaseInfo{
		Type:         m.GetAppType(),
		Id:           m.GetAppId(),
		Name:         m.GetAppName(),
		ListenOnAddr: m.GetMyAddress(),
	}
	appRegData[regKey].regToken = token
	appRegData[regKey].appState = lconf.AppStateRunning

	log.Debug("注册", "服务注册,appType=%v,appId=%v,regKey=%v,addr=%v",
		m.GetAppType(), m.GetAppId(), regKey, m.GetMyAddress())

	sendRsp := func(a n.AgentClient, i lconf.BaseInfo) {
		var rsp center.RegisterAppRsp
		rsp.RegResult = proto.Uint32(0)
		rsp.ReregToken = proto.String(token)
		rsp.CenterId = proto.Uint32(lconf.AppInfo.Id)
		rsp.AppName = proto.String(i.Name)
		rsp.AppType = proto.Uint32(i.Type)
		rsp.AppId = proto.Uint32(i.Id)
		rsp.AppAddress = proto.String(i.ListenOnAddr)
		a.SendData(n.AppCenter, uint32(center.CMDCenter_IDAppRegRsp), &rsp)
	}

	//自己注册成功
	sendRsp(a, appRegData[regKey].appInfo)

	//相互广播相互连接
	for k, v := range appRegData {
		if k == regKey {
			continue
		}
		//daemon判断
		if m.GetAppType() == n.AppDaemon {
			if v.appInfo.Type != n.AppConfig {
				continue
			}
			sendRsp(a, v.appInfo)
			sendRsp(v.a, appRegData[regKey].appInfo)
		} else {
			if v.appInfo.Type == n.AppDaemon {
				continue
			}
			sendRsp(a, v.appInfo)
			sendRsp(v.a, appRegData[regKey].appInfo)
		}
	}
}

func handleHeartBeatReq(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*center.HeartBeatReq)
	a := args[n.AgentIndex].(n.AgentClient)

	//非法判断
	if _, ok := appConnData[a]; !ok {
		log.Warning("心跳", "莫名的心跳?")
		return
	}
	app := appConnData[a]

	//log.Trace("", "心跳,aInfo=%v,state=%v,desc=%v,http=%v,rpc=%v",
	//	app.appInfo, m.GetServiceState(), m.GetStateDescription(), m.GetHttpAddress(), m.GetRpcAddress())

	app.totalHearDelayTime += time.Now().UnixNano() - m.GetPulseTime()
	app.lastHeartbeat = time.Now().UnixNano()
	if m.GetServiceState() != lconf.AppStateNone {
		app.appState = int(m.GetServiceState())
	}
	app.stateDescription = m.GetStateDescription()
	app.httpAddr = m.GetHttpAddress()
	app.rpcAddr = m.GetRpcAddress()

	var rsp center.HeartBeatRsp
	rsp.PulseTime = proto.Int64(time.Now().Unix())
	a.SendData(n.AppCenter, uint32(center.CMDCenter_IDHeartBeatRsp), &rsp)
}

func broadcastAppState(appType, appId uint32, state int32) {
	for a, v := range appConnData {
		if v.appInfo.Type == appType && v.appInfo.Id == appId {
			continue
		}
		var rsp center.AppStateNotify
		rsp.AppState = proto.Int32(state)
		rsp.CenterId = proto.Uint32(lconf.AppInfo.Id)
		rsp.AppType = proto.Uint32(appType)
		rsp.AppId = proto.Uint32(appId)
		a.SendData(n.AppCenter, uint32(center.CMDCenter_IDAppState), &rsp)
	}
}

type configServer struct {
	DaemonId uint32
	Alias    string
	lconf.BaseInfo
}

func getConfigServerList() []configServer {
	sList := make([]configServer, 0)
	v := apollo.GetConfig("服务列表", "")
	if err := json.Unmarshal([]byte(v), &sList); err != nil {
		log.Warning("", "反序列化服务列表出错,err=%v", err)
	}
	return sList
}

func getBaseInfoFromConfigList(appType, appId uint32) *configServer {
	sList := getConfigServerList()
	for _, v := range sList {
		if v.Type == appType && v.Id == appId {
			return &v
		}
	}
	return nil
}
