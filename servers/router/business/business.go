package business

import (
	"fmt"
	"google.golang.org/protobuf/proto"
	"math/rand"
	"reflect"
	"sync"
	"time"
	"xlddz/core/conf/apollo"
	g "xlddz/core/gate"
	"xlddz/core/log"
	"xlddz/core/module"
	n "xlddz/core/network"
	"xlddz/core/network/protobuf"
	"xlddz/protocol/router"
	"xlddz/servers/router/conf"
)

var (
	skeleton      = module.NewSkeleton(conf.GoLen, conf.TimerDispatcherLen, conf.AsynCallLen, conf.ChanRPCLen)
	mutexConnData sync.Mutex
	mutexRegData  sync.Mutex
	appConnData   map[n.Agent]*connectionData = make(map[n.Agent]*connectionData)
	appRegData    map[uint64]*connectionData  = make(map[uint64]*connectionData)
	processor                                 = protobuf.NewProcessor()
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
	a             n.Agent
	regInfo       appRegInfo
	lastHeartbeat int64
}

func init() {
	//消息注册
	chanRPC := skeleton.ChanRPCServer
	processor.Register(&router.RegisterAppReq{}, n.CMDRouter, uint16(router.CMDID_Router_IDAppRegReq), chanRPC)
	processor.Register(&router.AppStateNotify{}, n.CMDRouter, uint16(router.CMDID_Router_IDAppState), chanRPC)
	processor.Register(&router.DataTransferReq{}, n.CMDRouter, uint16(router.CMDID_Router_IDDataMessageReq), chanRPC)
	processor.Register(&router.AppPulseNotify{}, n.CMDRouter, uint16(router.CMDID_Router_IDPulseNotify), chanRPC)
	processor.Register(&router.AppOfflineReq{}, n.CMDRouter, uint16(router.CMDID_Router_IDAppOfflineReq), chanRPC)
	processor.Register(&router.AppUpdateReq{}, n.CMDRouter, uint16(router.CMDID_Router_IDAppUpdateReq), chanRPC)

	chanRPC.Register(g.ConnectSuccess, connectSuccess)
	chanRPC.Register(g.Disconnect, disconnect)

	chanRPC.Register(reflect.TypeOf(&router.RegisterAppReq{}), handleRegisterAppReq)
	chanRPC.Register(reflect.TypeOf(&router.AppStateNotify{}), handleAppStateNotify)
	chanRPC.Register(reflect.TypeOf(&router.DataTransferReq{}), handleDataTransferReq)
	chanRPC.Register(reflect.TypeOf(&router.AppPulseNotify{}), handleAppPulseNotify)
	chanRPC.Register(reflect.TypeOf(&router.AppOfflineReq{}), handleAppOfflineReq)
	chanRPC.Register(reflect.TypeOf(&router.AppUpdateReq{}), handleAppUpdateReq)
}

type Gate struct {
	*g.Gate
}

func (m *Gate) OnInit() {
	m.Gate = &g.Gate{
		AgentChanRPC: skeleton.ChanRPCServer,
		Processor:    processor,
		TCPAddr:      conf.Server.TCPAddr,
	}
}

func (m *Gate) OnDestroy() {}

type Module struct {
	*module.Skeleton
}

func (m *Module) OnInit() {
	m.Skeleton = skeleton
}

func (m *Module) OnDestroy() {}

func connectSuccess(args []interface{}) {
	mutexConnData.Lock()
	defer mutexConnData.Unlock()
	log.Info("连接", "来了老弟,当前连接数=%d", len(appConnData))
	a := args[g.AgentIndex].(n.Agent)
	if v, ok := appConnData[a]; ok {
		log.Error("连接", "异常,重复连接?,%d,%d", v.regInfo.appType, v.regInfo.appId)
		a.Close()
		return
	}
	appConnData[a] = &connectionData{a: a, regInfo: appRegInfo{curStep: connected}}
}

func disconnect(args []interface{}) {
	mutexConnData.Lock()
	defer mutexConnData.Unlock()
	log.Info("连接", "告辞中,当前连接数=%d", len(appConnData))
	a := args[g.AgentIndex].(n.Agent)
	if v, ok := appConnData[a]; ok {
		regKey := makeRegKey(v.regInfo.appType, v.regInfo.appId)
		log.Info("连接", "再见,appType=%d,appId=%d,regKey=%d", v.regInfo.appType, v.regInfo.appId, regKey)
		delete(appConnData, a)
		mutexRegData.Lock()
		delete(appRegData, regKey)
		mutexRegData.Unlock()
	} else {
		log.Error("连接", "异常,没有注册的连接?")
	}
}

func handleRegisterAppReq(args []interface{}) {
	m := args[n.DATA_INDEX].(*router.RegisterAppReq)
	a := args[n.AGENT_INDEX].(n.Agent)

	mutexConnData.Lock()
	defer mutexConnData.Unlock()

	//连接存在判断
	if _, ok := appConnData[a]; !ok {
		log.Error("连接", "异常,没有注册的连接?")
		a.Close()
		return
	}

	//是否已注册
	if appConnData[a].regInfo.curStep == registered {
		return
	}

	regKey := makeRegKey(m.GetAppType(), m.GetAppId())
	mutexRegData.Lock()
	if v, ok := appRegData[regKey]; ok {
		if v.regInfo.regToken != m.GetReregToken() {
			mutexRegData.Unlock()
			resultMsg := fmt.Sprintf("该服务已注册,appType=%v,appId=%v,regKey=%v",
				m.GetAppType(), m.GetAppId(), regKey)
			log.Warning("连接", resultMsg)

			var rsp router.RegisterAppRsp
			rsp.RegResult = proto.Uint32(1)
			rsp.ReregToken = proto.String(resultMsg)
			rsp.RouterId = proto.Uint32(conf.Server.AppID)
			a.SendData(n.CMDRouter, uint32(router.CMDID_Router_IDAppRegRsp), &rsp)

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
	appRegData[regKey].regInfo = appRegInfo{m.GetAppType(), m.GetAppId(), token, m.GetAppName(), registered}
	mutexRegData.Unlock()

	log.Debug("注册", "服务注册,appType=%v,appId=%v,regKey=%v",
		m.GetAppType(), m.GetAppId(), regKey)

	var rsp router.RegisterAppRsp
	rsp.RegResult = proto.Uint32(0)
	rsp.ReregToken = proto.String(token)
	rsp.RouterId = proto.Uint32(conf.Server.AppID)
	a.SendData(n.CMDRouter, uint32(router.CMDID_Router_IDAppRegRsp), &rsp)
}

func handleAppStateNotify(args []interface{}) {

}

func handleDataTransferReq(args []interface{}) {
	m := args[n.DATA_INDEX].(*router.DataTransferReq)
	a := args[n.AGENT_INDEX].(n.Agent)

	mutexConnData.Lock()
	defer mutexConnData.Unlock()

	//连接存在判断
	if _, ok := appConnData[a]; !ok {
		log.Error("连接", "异常,么有连接的注册?")
		a.Close()
		return
	}

	if appConnData[a].regInfo.curStep != registered {
		log.Warning("转发", "不要急,还没注册完成")
		return
	}

	log.Debug("转发", "消息转发,"+
		"SrcApptype=%v,SrcAppid=%v,"+
		"DestApptype=%v,DestApptype=%v,"+
		"Cmdkind=%v,Cmdsubid=%v",
		m.GetSrcApptype(), m.GetSrcAppid(),
		m.GetDestApptype(), m.GetDestAppid(),
		m.GetDataCmdkind(), m.GetDataCmdsubid())

	//目的判断
	if m.GetDestApptype() == n.AppRouter {
		switch m.GetDataCmdkind() {
		case n.CMDConfig:
			apollo.ProcessReq(m)
		default:

		}
	} else {
		//转发只目标
		sendResult := false
		switch m.GetDestAppid() {
		case n.Send2All:
			for k, v := range appConnData {
				if v.regInfo.appType == m.GetDestApptype() {
					k.SendData(n.CMDRouter, uint32(router.CMDID_Router_IDDataMessageReq), m)
				}
			}
			sendResult = true
		case n.Send2AnyOne:
			for k, v := range appConnData {
				if v.regInfo.appType == m.GetDestApptype() {
					k.SendData(n.CMDRouter, uint32(router.CMDID_Router_IDDataMessageReq), m)
					sendResult = true
					break
				}
			}
		default:
			for k, v := range appConnData {
				if v.regInfo.appType == m.GetDestApptype() && v.regInfo.appId == m.GetDestAppid() {
					k.SendData(n.CMDRouter, uint32(router.CMDID_Router_IDDataMessageReq), m)
					sendResult = true
					break
				}
			}
		}

		if !sendResult {
			destAppCount := 0
			for _, v := range appConnData {
				if v.regInfo.appType == m.GetDestApptype() {
					destAppCount++
				}
			}
			log.Error("转发", "异常,消息转发失败,"+
				"SrcApptype=%v,SrcAppid=%v,"+
				"DestApptype=%v,DestApptype=%v,"+
				"Cmdkind=%v,Cmdsubid=%v,目标app数量=%d",
				m.GetSrcApptype(), m.GetSrcAppid(),
				m.GetDestApptype(), m.GetDestAppid(),
				m.GetDataCmdkind(), m.GetDataCmdsubid(), destAppCount)
		}
	}
}

func handleAppPulseNotify(args []interface{}) {
	m := args[n.DATA_INDEX].(*router.AppPulseNotify)
	a := args[n.AGENT_INDEX].(n.Agent)

	mutexConnData.Lock()
	defer mutexConnData.Unlock()

	//非法判断
	if _, ok := appConnData[a]; !ok {
		log.Warning("心跳", "莫名的心跳?")
		return
	}

	switch m.GetAction() {
	case router.AppPulseNotify_LogoutReq:
	case router.AppPulseNotify_HeartBeatReq:
		var rsp router.AppPulseNotify
		rsp.Action = (*router.AppPulseNotify_PulseAction)(proto.Int32(int32(router.AppPulseNotify_HeartBeatRsp)))
		a.SendData(n.CMDRouter, uint32(router.CMDID_Router_IDPulseNotify), &rsp)
		appConnData[a].lastHeartbeat = time.Now().UnixNano()
	}

}

func handleAppOfflineReq(args []interface{}) {

}

func handleAppUpdateReq(args []interface{}) {

}

func makeRegKey(appType, appId uint32) uint64 {
	return uint64(appType)<<32 | uint64(appId)
}
