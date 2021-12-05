package business

import (
	"encoding/json"
	"fmt"
	"google.golang.org/protobuf/proto"
	"math/rand"
	"reflect"
	"time"
	lconf "xlddz/core/conf"
	"xlddz/core/conf/apollo"
	g "xlddz/core/gate"
	"xlddz/core/log"
	"xlddz/core/module"
	n "xlddz/core/network"
	"xlddz/core/network/protobuf"
	"xlddz/protocol/center"
	"xlddz/servers/center/conf"
)

var (
	skeleton                                      = module.NewSkeleton(conf.GoLen, conf.TimerDispatcherLen, conf.AsynCallLen, conf.ChanRPCLen)
	appConnData map[n.AgentClient]*connectionData = make(map[n.AgentClient]*connectionData)
	appRegData  map[uint64]*connectionData        = make(map[uint64]*connectionData)
	processor                                     = protobuf.NewProcessor()
)

const (
	connected        int = 1
	registered       int = 2
	underMaintenance int = 3
)

//连接数据
type appRegInfo struct {
	appType   uint32
	appId     uint32
	regToken  string
	appName   string
	address   string
	curStatus int
}

type connectionData struct {
	a             n.AgentClient
	regInfo       appRegInfo
	lastHeartbeat int64
}

func init() {
	//消息注册
	chanRPC := skeleton.ChanRPCServer
	processor.Register(&center.RegisterAppReq{}, n.CMDCenter, uint16(center.CMDID_Center_IDAppRegReq), chanRPC)
	processor.Register(&center.AppStateNotify{}, n.CMDCenter, uint16(center.CMDID_Center_IDAppState), chanRPC)
	processor.Register(&center.DataTransferReq{}, n.CMDCenter, uint16(center.CMDID_Center_IDDataMessageReq), chanRPC)
	processor.Register(&center.AppPulseNotify{}, n.CMDCenter, uint16(center.CMDID_Center_IDPulseNotify), chanRPC)
	processor.Register(&center.AppOfflineReq{}, n.CMDCenter, uint16(center.CMDID_Center_IDAppOfflineReq), chanRPC)
	processor.Register(&center.AppUpdateReq{}, n.CMDCenter, uint16(center.CMDID_Center_IDAppUpdateReq), chanRPC)

	chanRPC.Register(g.ConnectSuccess, connectSuccess)
	chanRPC.Register(g.Disconnect, disconnect)
	chanRPC.Register(reflect.TypeOf(&center.RegisterAppReq{}), handleRegisterAppReq)
	chanRPC.Register(reflect.TypeOf(&center.AppStateNotify{}), handleAppStateNotify)
	chanRPC.Register(reflect.TypeOf(&center.DataTransferReq{}), handleDataTransferReq)
	chanRPC.Register(reflect.TypeOf(&center.AppPulseNotify{}), handleAppPulseNotify)
	chanRPC.Register(reflect.TypeOf(&center.AppOfflineReq{}), handleAppOfflineReq)
	chanRPC.Register(reflect.TypeOf(&center.AppUpdateReq{}), handleAppUpdateReq)

	apollo.RegPublicCB(configChangeNotify)
}

type Gate struct {
	*g.Gate
}

func (m *Gate) OnInit() {
	g.AgentChanRPC = skeleton.ChanRPCServer
	g.Processor = processor
	m.Gate = &g.Gate{
		TCPAddr: conf.Server.TCPAddr,
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
	log.Info("连接", "来了老弟,当前连接数=%d", len(appConnData))
	a := args[g.AgentIndex].(n.AgentClient)
	if v, ok := appConnData[a]; ok {
		log.Error("连接", "异常,重复连接?,%d,%d", v.regInfo.appType, v.regInfo.appId)
		a.Close()
		return
	}
	appConnData[a] = &connectionData{a: a, regInfo: appRegInfo{curStatus: connected}}
}

func disconnect(args []interface{}) {
	log.Info("连接", "告辞中,当前连接数=%d", len(appConnData))
	a := args[g.AgentIndex].(n.AgentClient)
	if v, ok := appConnData[a]; ok {
		regKey := makeRegKey(v.regInfo.appType, v.regInfo.appId)
		log.Info("连接", "再见,appType=%d,appId=%d,regKey=%d", v.regInfo.appType, v.regInfo.appId, regKey)
		delete(appConnData, a)
		delete(appRegData, regKey)
	} else {
		log.Error("连接", "异常,没有注册的连接?")
	}
}

func configChangeNotify(k apollo.ConfKey, v apollo.ConfValue) {

	key := apollo.ConfKey{AppType: lconf.AppType, AppId: lconf.AppID, Key: "服务维护"}
	if k == key {
		type appInfo struct {
			AppType uint32
			AppId   uint32
			OpType  uint32
		}
		var info appInfo
		err := json.Unmarshal([]byte(v.Value), &info)
		if err != nil {
			log.Error("配置", "%v", err)
			return
		}

		if _, ok := appRegData[makeRegKey(info.AppType, info.AppId)]; !ok {
			log.Warning("配置", "要维护的服务不存在啊,info=%v", info)
			return
		}
		appRegData[makeRegKey(info.AppType, info.AppId)].regInfo.curStatus = int(info.OpType)

		log.Debug("配置", "收到服务维护配置,%v", info)
	}
}

func handleRegisterAppReq(args []interface{}) {
	b := args[n.DATA_INDEX].(n.BaseMessage)
	m := (b.MyMessage).(*center.RegisterAppReq)
	a := args[n.AGENT_INDEX].(n.AgentClient)

	//连接存在判断
	if _, ok := appConnData[a]; !ok {
		log.Error("连接", "异常,没有注册的连接?")
		a.Close()
		return
	}

	//是否已注册
	if appConnData[a].regInfo.curStatus == registered {
		return
	}

	regKey := makeRegKey(m.GetAppType(), m.GetAppId())
	if v, ok := appRegData[regKey]; ok {
		if v.regInfo.regToken != m.GetReregToken() {

			resultMsg := fmt.Sprintf("该服务已注册,appType=%v,appId=%v,regKey=%v",
				m.GetAppType(), m.GetAppId(), regKey)
			log.Warning("连接", resultMsg)

			var rsp center.RegisterAppRsp
			rsp.RegResult = proto.Uint32(1)
			rsp.ReregToken = proto.String(resultMsg)
			rsp.RouterId = proto.Uint32(conf.Server.AppID)
			a.SendData(n.CMDCenter, uint32(center.CMDID_Center_IDAppRegRsp), &rsp)

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
	appRegData[regKey].regInfo = appRegInfo{m.GetAppType(), m.GetAppId(), token, m.GetAppName(), m.GetMyAddress(), registered}

	log.Debug("注册", "服务注册,appType=%v,appId=%v,regKey=%v,%v",
		m.GetAppType(), m.GetAppId(), regKey, m.GetMyAddress())

	var rsp center.RegisterAppRsp
	rsp.RegResult = proto.Uint32(0)
	rsp.ReregToken = proto.String(token)
	rsp.RouterId = proto.Uint32(conf.Server.AppID)
	rsp.AppName = proto.String(m.GetAppName())
	rsp.AppType = proto.Uint32(m.GetAppType())
	rsp.AppId = proto.Uint32(m.GetAppId())
	rsp.AppAddress = proto.String(m.GetMyAddress())
	for k, _ := range appConnData {
		k.SendData(n.CMDCenter, uint32(center.CMDID_Center_IDAppRegRsp), &rsp)
	}

	//广播已注册
	for _, v := range appConnData {
		if v.regInfo.appType == m.GetAppType() && v.regInfo.appId == m.GetAppId() {
			continue
		}
		var rsp center.RegisterAppRsp
		rsp.RegResult = proto.Uint32(0)
		rsp.ReregToken = proto.String(v.regInfo.regToken)
		rsp.RouterId = proto.Uint32(conf.Server.AppID)
		rsp.AppName = proto.String(v.regInfo.appName)
		rsp.AppType = proto.Uint32(v.regInfo.appType)
		rsp.AppId = proto.Uint32(v.regInfo.appId)
		rsp.AppAddress = proto.String(v.regInfo.address)
		a.SendData(n.CMDCenter, uint32(center.CMDID_Center_IDAppRegRsp), &rsp)
	}
}

func handleAppStateNotify(args []interface{}) {

}

func handleDataTransferReq(args []interface{}) {
	b := args[n.DATA_INDEX].(n.BaseMessage)
	m := (b.MyMessage).(*center.DataTransferReq)
	a := args[n.AGENT_INDEX].(n.AgentClient)

	//连接存在判断
	if _, ok := appConnData[a]; !ok {
		log.Error("连接", "异常,么有连接的注册?")
		a.Close()
		return
	}

	if appConnData[a].regInfo.curStatus != registered {
		log.Warning("转发", "兄弟,你状态有问题啊,"+
			"SrcApptype=%v,SrcAppid=%v,"+
			"DestApptype=%v,DestApptype=%v,"+
			"Cmdkind=%v,Cmdsubid=%v,regInfo=%v",
			m.GetSrcApptype(), m.GetSrcAppid(),
			m.GetDestApptype(), m.GetDestAppid(),
			m.GetDataCmdkind(), m.GetDataCmdsubid(), appConnData[a].regInfo)
		return
	}

	if m.GetDestApptype() != n.AppCenter {
		log.Warning("转发", "兄弟,你状态有问题啊,"+
			"SrcApptype=%v,SrcAppid=%v,"+
			"DestApptype=%v,DestApptype=%v,"+
			"Cmdkind=%v,Cmdsubid=%v,regInfo=%v",
			m.GetSrcApptype(), m.GetSrcAppid(),
			m.GetDestApptype(), m.GetDestAppid(),
			m.GetDataCmdkind(), m.GetDataCmdsubid(), appConnData[a].regInfo)
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
	if m.GetDestApptype() == n.AppCenter {
		switch m.GetDataCmdkind() {
		case n.CMDConfig:
			//apollo.ProcessReq(m)
		default:
		}
	} else {
		destTypeAppCount := func() int {
			destCount := 0
			for _, v := range appConnData {
				if v.regInfo.appType == m.GetDestApptype() {
					destCount++
				}
			}
			return destCount
		}
		sendResult := false
		if destTypeAppCount() != 0 {
			switch m.GetDestAppid() {
			case n.Send2All:
				for k, v := range appConnData {
					if v.regInfo.appType == m.GetDestApptype() && v.regInfo.curStatus != underMaintenance {
						k.SendData(n.CMDCenter, uint32(center.CMDID_Center_IDDataMessageReq), m)
					}
				}
				sendResult = true
			case n.Send2AnyOne:
				for k, v := range appConnData {
					if v.regInfo.appType == m.GetDestApptype() && v.regInfo.curStatus != underMaintenance {
						k.SendData(n.CMDCenter, uint32(center.CMDID_Center_IDDataMessageReq), m)
						sendResult = true
						break
					}
				}
			default:
				for k, v := range appConnData {
					if v.regInfo.appType == m.GetDestApptype() && v.regInfo.appId == m.GetDestAppid() && v.regInfo.curStatus != underMaintenance {
						k.SendData(n.CMDCenter, uint32(center.CMDID_Center_IDDataMessageReq), m)
						sendResult = true
						break
					}
				}
			}
		}

		if !sendResult {
			log.Error("转发", "异常,消息转发失败,"+
				"SrcApptype=%v,SrcAppid=%v,"+
				"DestApptype=%v,DestApptype=%v,"+
				"Cmdkind=%v,Cmdsubid=%v,目标app数量=%d",
				m.GetSrcApptype(), m.GetSrcAppid(),
				m.GetDestApptype(), m.GetDestAppid(),
				m.GetDataCmdkind(), m.GetDataCmdsubid(), destTypeAppCount())
		}
	}
}

func handleAppPulseNotify(args []interface{}) {
	b := args[n.DATA_INDEX].(n.BaseMessage)
	m := (b.MyMessage).(*center.AppPulseNotify)
	a := args[n.AGENT_INDEX].(n.AgentClient)

	//非法判断
	if _, ok := appConnData[a]; !ok {
		log.Warning("心跳", "莫名的心跳?")
		return
	}

	switch m.GetAction() {
	case center.AppPulseNotify_LogoutReq:
	case center.AppPulseNotify_HeartBeatReq:
		var rsp center.AppPulseNotify
		rsp.Action = (*center.AppPulseNotify_PulseAction)(proto.Int32(int32(center.AppPulseNotify_HeartBeatRsp)))
		a.SendData(n.CMDCenter, uint32(center.CMDID_Center_IDPulseNotify), &rsp)
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