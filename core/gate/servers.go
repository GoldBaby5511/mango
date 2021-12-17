package gate

import (
	"github.com/golang/protobuf/proto"
	"reflect"
	"time"
	"xlddz/core/conf"
	"xlddz/core/conf/apollo"
	"xlddz/core/log"
	n "xlddz/core/network"
	"xlddz/protocol/center"
)

type agentServer struct {
	tcpClient *n.TCPClient
	conn      n.Conn
	info      n.BaseAgentInfo
}

func newServerItem(info n.BaseAgentInfo, autoReconnect bool, pendingWriteNum int) {
	if info.ListenOnAddress == "" {
		log.Warning("agentServer", "警告,没地址怎么连接?,info=%v,autoReconnect=%v,pendingWriteNum=%v",
			info, autoReconnect, pendingWriteNum)
		return
	}

	tcpClient := new(n.TCPClient)
	tcpClient.Addr = info.ListenOnAddress
	tcpClient.PendingWriteNum = pendingWriteNum
	tcpClient.AutoReconnect = autoReconnect
	tcpClient.NewAgent = func(conn *n.TCPConn) n.AgentServer {
		a := &agentServer{tcpClient: tcpClient, conn: conn, info: info}
		log.Debug("agentServer", "连接成功,%v,autoReconnect=%v", a.info, autoReconnect)
		sendRegAppReq(a)
		timeInterval := 30 * time.Second
		timerHeartbeat := time.NewTimer(timeInterval)
		go func(t *time.Timer) {
			for {
				<-t.C
				var pulse center.AppPulseNotify
				pulse.Action = (*center.AppPulseNotify_PulseAction)(proto.Int32(int32(center.AppPulseNotify_HeartBeatReq)))
				pulse.PulseData = proto.Uint64(uint64(time.Now().Unix()))
				a.SendData(n.CMDCenter, uint32(center.CMDID_Center_IDPulseNotify), &pulse)

				t.Reset(timeInterval)
			}
		}(timerHeartbeat)

		if n.AppConfig == info.AppType {
			apollo.SetNetAgent(a)
			apollo.RegisterConfig("", conf.AppInfo.AppType, conf.AppInfo.AppID, nil)
		}

		mxServers.Lock()
		servers[uint64(info.AppType)<<32|uint64(info.AppID)] = a
		mxServers.Unlock()
		return a
	}

	log.Debug("agentServer", "开始连接,%v,autoReconnect=%v", info, autoReconnect)

	if tcpClient != nil {
		tcpClient.Start()
	}
}

func (a *agentServer) Run() {
	for {
		bm, msgData, err := a.conn.ReadMsg()
		if err != nil {
			log.Warning("agentServer", "异常,网关读取消息失败,info=%v,err=%v", a.info, err)
			break
		}

		if bm.Cmd.MainCmdID != uint16(n.CMDCenter) {
			log.Warning("", "不可能出现非center消息,cmd=%v", bm.Cmd)
			break
		}

		switch bm.Cmd.SubCmdID {
		case uint16(center.CMDID_Center_IDAppRegRsp):
			var m center.RegisterAppRsp
			_ = proto.Unmarshal(msgData, &m)

			if m.GetRegResult() == 0 {
				log.Info("agentServer", "注册成功,regToken=%v,RouterId=%v,%v,%v,%v,%v",
					m.GetReregToken(), m.GetCenterId(), m.GetAppName(), m.GetAppType(), m.GetAppId(), m.GetAppAddress())

				//获取配置
				mxServers.Lock()
				_, ok := servers[uint64(m.GetAppType())<<32|uint64(m.GetAppId())]
				mxServers.Unlock()
				if !(conf.AppInfo.AppType == m.GetAppType() && conf.AppInfo.AppID == m.GetAppId()) && !ok {
					if m.GetAppAddress() != "" {
						info := n.BaseAgentInfo{AgentType: n.CommonServer, AppName: m.GetAppName(), AppType: m.GetAppType(), AppID: m.GetAppId(), ListenOnAddress: m.GetAppAddress()}
						newServerItem(info, false, 0)
					} else {
						log.Warning("agentServer", "没有地址?,%v,%v,%v,%v",
							m.GetAppName(), m.GetAppType(), m.GetAppId(), m.GetAppAddress())
					}
				}

				if conf.AppInfo.AppType == n.AppConfig {
					mxServers.Lock()
					if _, ok := servers[uint64(n.AppCenter)<<32|uint64(0)]; ok {
						servers[uint64(n.AppCenter)<<32|uint64(0)].info.AppID = m.GetCenterId()
					}
					mxServers.Unlock()
				}
			} else {
				log.Warning("agentServer", "注册失败,RouterId=%v,原因=%v", m.GetCenterId(), m.GetReregToken())
			}
			if agentChanRPC != nil {
				agentChanRPC.Call0(CenterRegResult, m.GetRegResult(), m.GetCenterId())
			}
		case uint16(center.CMDID_Center_IDAppState): //app状态改变
			var m center.AppStateNotify
			_ = proto.Unmarshal(msgData, &m)
			log.Debug("agentServer", "app状态改变 AppState=%v,RouterId=%v,AppType=%v,AppId=%v",
				m.GetAppState(), m.GetCenterId(), m.GetAppType(), m.GetAppId())

			mxServers.Lock()
			if _, ok := servers[uint64(m.GetAppType())<<32|uint64(m.GetAppId())]; ok {
				servers[uint64(m.GetAppType())<<32|uint64(m.GetAppId())].Close()
			}
			mxServers.Unlock()

		case uint16(center.CMDID_Center_IDPulseNotify):
		default:
			log.Error("agentServer", "n.CMDCenter,异常,还未处理消息,%v", bm.Cmd)
		}
	}
}

func (a *agentServer) OnClose() {
	log.Debug("", "服务间连接断开了,info=%v", a.info)
	if a.info.AppType == n.AppLogger {
		log.SetCallback(nil)
		log.Info("agentServer", "日志服务器断开")
	} else if a.info.AppType == n.AppCenter {
		log.Warning("agentServer", "异常,与center连接断开,世界需要重启... ...")
		for _, c := range cbCenterDisconnect {
			c()
		}
	}
	if a.tcpClient != nil && !a.tcpClient.AutoReconnect {
		a.tcpClient.Close()
	}
	mxServers.Lock()
	delete(servers, uint64(a.info.AppType)<<32|uint64(a.info.AppID))
	mxServers.Unlock()
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

func (a *agentServer) Close() {
	a.conn.Close()
}
func (a *agentServer) Destroy() {
	a.conn.Destroy()
}
