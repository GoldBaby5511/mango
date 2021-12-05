package gate

import (
	"google.golang.org/protobuf/proto"
	"reflect"
	"xlddz/core/conf"
	"xlddz/core/conf/apollo"
	"xlddz/core/log"
	n "xlddz/core/network"
	"xlddz/protocol/center"
)

type ServerItem struct {
	tcpClient *n.TCPClient
	a         *agentServer
	AppName   string
	AppID     uint32
	AppType   uint32
	Address   string
}

func NewServerItem(name, addr string, appType, id uint32) *ServerItem {
	s := new(ServerItem)
	s.tcpClient = new(n.TCPClient)
	s.a = nil
	s.AppName = name
	s.AppID = id
	s.AppType = appType
	s.Address = addr

	if s.Address != "" {
		s.tcpClient = new(n.TCPClient)
		s.tcpClient.Addr = s.Address
		s.tcpClient.AutoReconnect = false
		s.tcpClient.NewAgent = func(conn *n.TCPConn) n.AgentServer {
			s.a = &agentServer{conn: conn, s: s}
			log.Debug("", "互联成功,%v,%v,%v,%v", s.AppName, s.AppType, s.AppID, s.Address)
			sendRegAppReq(s.a)
			if n.AppConfig == s.AppType {
				apollo.SetRouterAgent(s.a)
				apollo.RegisterConfig("", conf.AppType, conf.AppID, nil)
			}

			servers[uint64(appType)<<32|uint64(id)] = s
			return s.a
		}

		log.Debug("", "开始互联,%v,%v,%v,%v", s.AppName, s.AppType, s.AppID, s.Address)

		s.tcpClient.Start()
	} else {
		log.Warning("", "没有地址?,%v,%v,%v,%v", s.AppName, s.AppType, s.AppID, s.Address)
	}
	return s
}

func (s *ServerItem) Close() {
	if s.tcpClient != nil {
		s.tcpClient.Close()
	}
}

type agentServer struct {
	conn    n.Conn
	s       *ServerItem
	AppName string
	AppID   uint32
	AppType uint32
	Address string
}

func (a *agentServer) Run() {
	for {
		bm, msgData, err := a.conn.ReadMsg()
		if err != nil {
			log.Warning("agentServer", "异常,网关读取消息失败,id=%v,agentServerType=%v,err=%v", a.AppID, a.AppType, err)
			break
		}

		if bm.Cmd.MainCmdID != uint16(n.CMDCenter) {
			log.Warning("", "不可能出现非router消息")
			break
		}

		//构造参数，全新改造中，暂时这么用着
		//log.Debug("", "成功了一半,收到消息,%v", headCmd)

		switch bm.Cmd.SubCmdID {
		case uint16(center.CMDID_Center_IDAppRegRsp): //router注册消息
			var m center.RegisterAppRsp
			_ = proto.Unmarshal(msgData, &m)

			if m.GetRegResult() == 0 {
				log.Info("agent", "注册成功,regToken=%v,RouterId=%v,%v,%v,%v,%v",
					m.GetReregToken(), m.GetRouterId(), m.GetAppName(), m.GetAppType(), m.GetAppId(), m.GetAppAddress())

				//获取配置
				_, ok := servers[uint64(m.GetAppType())<<32|uint64(m.GetAppId())]
				if !(conf.AppType == m.GetAppType() && conf.AppID == m.GetAppId()) && !ok {
					NewServerItem(m.GetAppName(), m.GetAppAddress(), m.GetAppType(), m.GetAppId())
				}

				if conf.AppType == n.AppConfig {
					if _, ok := servers[uint64(n.AppCenter)<<32|uint64(0)]; ok {
						servers[uint64(n.AppCenter)<<32|uint64(0)].AppID = m.GetRouterId()
					}
				}

			} else {
				log.Warning("agent", "注册失败,RouterId=%v,原因=%v", m.GetRouterId(), m.GetReregToken())
			}
			if AgentChanRPC != nil {
				AgentChanRPC.Call0(RouterRegResult, m.GetRegResult(), m.GetRouterId())
			}
		case uint16(center.CMDID_Center_IDAppState): //app状态改变
			var m center.AppStateNotify
			_ = proto.Unmarshal(msgData, &m)
			log.Debug("agent", "app状态改变 AppState=%v,RouterId=%v,AppType=%v,AppId=%v",
				m.GetAppState(), m.GetRouterId(), m.GetAppType(), m.GetAppId())
		case uint16(center.CMDID_Center_IDPulseNotify): //心跳
		default:
			log.Error("agent", "n.CMDCenter,异常,还未处理消息,%v", bm.Cmd)
		}
	}
}

func (a *agentServer) OnClose() {
	if a.s != nil {
		if a.AppType == n.AppLogger {
			log.SetCallback(nil)
			log.Info("agent", "日志服务器断开")
		} else if a.AppType == n.AppCenter {
			log.Warning("agent", "异常,与router连接断开,世界需要重启... ...")
			for _, cb := range cbRouterDisconnect {
				cb()
			}
		}
		log.Debug("", "server 断开了，%v,%v", a.s.AppType, a.s.AppID)
		delete(servers, uint64(uint64(a.s.AppType)<<32|uint64(a.s.AppID)))
	} else {
		log.Warning("", "有一个没有s的连接断开了")
	}

}

func (a *agentServer) SendMessage(bm n.BaseMessage) {
	m := bm.MyMessage.(proto.Message)
	data, err := proto.Marshal(m)
	if err != nil {
		log.Error("agentServer", "异常,proto.Marshal %v error: %v", reflect.TypeOf(m), err)
		return
	}
	//追加TraceId
	otherData := make([]byte, 0, n.TraceIdLen+1)
	if bm.TraceId != "" {
		otherData = append(otherData, n.FlagOtherTraceId)
		otherData = append(otherData, []byte(bm.TraceId)...)
	}
	err = a.conn.WriteMsg(bm.Cmd.MainCmdID, bm.Cmd.SubCmdID, data, otherData)
	if err != nil {
		log.Error("agentServer", "写信息失败 %v error: %v", reflect.TypeOf(m), err)
	}
}

func (a *agentServer) SendData(mainCmdID, subCmdID uint32, m proto.Message) {
	data, err := proto.Marshal(m)
	if err != nil {
		log.Error("agentServer", "异常,proto.Marshal %v error: %v", reflect.TypeOf(m), err)
		return
	}
	err = a.conn.WriteMsg(uint16(mainCmdID), uint16(subCmdID), data, nil)
	if err != nil {
		log.Error("agentServer", "write message %v error: %v", reflect.TypeOf(m), err)
	}
}

func (a *agentServer) getTranData(destAppid, destAppType, dataKind, dataSubId, direction uint32) center.DataTransferReq {
	var dataReq center.DataTransferReq
	dataReq.SrcAppid = proto.Uint32(conf.AppID)
	dataReq.SrcApptype = proto.Uint32(conf.AppType)
	dataReq.DestAppid = proto.Uint32(destAppid)
	dataReq.DestApptype = proto.Uint32(destAppType)
	dataReq.DataCmdkind = proto.Uint32(dataKind)
	dataReq.DataCmdsubid = proto.Uint32(dataSubId)
	dataReq.DataDirection = proto.Uint32(direction)
	return dataReq
}

func (a *agentServer) Close()   {}
func (a *agentServer) Destroy() {}
