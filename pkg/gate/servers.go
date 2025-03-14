package gate

import (
	"reflect"
	"time"

	"mango/api/center"
	"mango/pkg/conf"
	"mango/pkg/conf/apollo"
	"mango/pkg/log"
	n "mango/pkg/network"
	"mango/pkg/util"

	"github.com/golang/protobuf/proto"
)

type agentServer struct {
	tcpClient *n.TCPClient
	conn      n.Conn
	info      n.BaseAgentInfo
}

func newServerItem(info n.BaseAgentInfo, autoReconnect bool, pendingWriteNum int) {
	if info.ListenOnAddr == "" {
		log.Warning("agentServer", "警告,没地址怎么连接?,info=%v,autoReconnect=%v,pendingWriteNum=%v",
			info, autoReconnect, pendingWriteNum)
		return
	}

	tcpClient := new(n.TCPClient)
	tcpClient.Addr = info.ListenOnAddr
	tcpClient.PendingWriteNum = pendingWriteNum
	tcpClient.AutoReconnect = autoReconnect
	tcpClient.NewAgent = func(conn *n.TCPConn) n.AgentServer {
		a := &agentServer{tcpClient: tcpClient, conn: conn, info: info}
		log.Debug("agentServer", "连接成功,%v", util.PrintStructFields(a.info))
		sendRegAppReq(a)
		timeInterval := 20 * time.Second
		timerHeartbeat := time.NewTimer(timeInterval)
		go func(t *time.Timer) {
			for {
				<-t.C
				req := center.HeartBeatReq{
					PulseTime:        time.Now().Unix(),
					ServiceState:     int32(ServiceState),
					StateDescription: GetStateDescription(),
					HttpAddress:      apollo.GetConfig("http监听地址", ""),
					RpcAddress:       apollo.GetConfig("rpc监听地址", ""),
				}
				a.SendData(n.AppCenter, uint32(center.CMDCenter_IDHeartBeatReq), &req)

				t.Reset(timeInterval)
			}
		}(timerHeartbeat)

		mxServers.Lock()
		servers[util.MakeUint64FromUint32(info.AppType, info.AppId)] = a
		mxServers.Unlock()

		if info.AppType == n.AppConfig {
			tcpClient.AutoReconnect = true
			apollo.SetNetAgent(a)
		}
		return a
	}

	log.Debug("agentServer", "开始连接,%v", util.PrintStructFields(info))

	tcpClient.Start()
}

func (a *agentServer) Run() {
	for {
		bm, msgData, err := a.conn.ReadMsg()
		if err != nil {
			log.Warning("agentServer", "Server读取消息失败,err=%v,%v", err, util.PrintStructFields(a.info))
			break
		}

		if bm.Cmd.MainCmdID != uint16(n.AppCenter) {
			log.Warning("agentServer", "不可能出现非center消息,cmd=%v", bm.Cmd)
			break
		}

		bm.AgentInfo = a.info
		switch center.CMDCenter(bm.Cmd.SubCmdID) {
		case center.CMDCenter_IDAppRegRsp:
			var m center.RegisterAppRsp
			_ = proto.Unmarshal(msgData, &m)

			log.Info("agentServer", "注册消息,regResult=%v,CenterId=%v,appName=%v,appType=%v,appId=%v,Addr=%v",
				m.GetRegResult(), m.GetCenterId(), m.GetAppName(), m.GetAppType(), m.GetAppId(), m.GetAppAddress())

			if m.GetRegResult() == 0 {
				mxServers.Lock()
				_, ok := servers[util.MakeUint64FromUint32(m.GetAppType(), m.GetAppId())]
				if conf.AppInfo.Type == m.GetAppType() && conf.AppInfo.Id == m.GetAppId() {
					//更新center信息,就一个不更新也没啥问题
					if _, ok := servers[util.MakeUint64FromUint32(n.AppCenter, 0)]; ok {
						servers[util.MakeUint64FromUint32(n.AppCenter, m.GetCenterId())] = servers[util.MakeUint64FromUint32(n.AppCenter, 0)]
						servers[util.MakeUint64FromUint32(n.AppCenter, m.GetCenterId())].info.AppId = m.GetCenterId()
						delete(servers, util.MakeUint64FromUint32(n.AppCenter, 0))
					}
				}
				mxServers.Unlock()

				if !(conf.AppInfo.Type == m.GetAppType() && conf.AppInfo.Id == m.GetAppId()) && !ok {
					info := n.BaseAgentInfo{AgentType: n.CommonServer, AppName: m.GetAppName(), AppType: m.GetAppType(), AppId: m.GetAppId(), ListenOnAddr: m.GetAppAddress()}
					newServerItem(info, false, 0)
				}
			}

			if agentChanRPC != nil {
				agentChanRPC.Call0(CenterRegResult, m.GetRegResult(), m.GetCenterId(), conf.BaseInfo{Name: m.GetAppName(), Type: m.GetAppType(), Id: m.GetAppId()})
			}
		case center.CMDCenter_IDAppState: //app状态改变
			var m center.AppStateNotify
			_ = proto.Unmarshal(msgData, &m)

			log.Debug("agentServer", "app状态改变 AppState=%v,CenterId=%v,AppType=%v,AppId=%v",
				m.GetAppState(), m.GetCenterId(), m.GetAppType(), m.GetAppId())

			//服务下线
			if m.GetAppState() == conf.AppStateOffline {
				mxServers.Lock()
				key := util.MakeUint64FromUint32(m.GetAppType(), m.GetAppId())
				if _, ok := servers[key]; ok {
					servers[key].Close()
				}
				mxServers.Unlock()
			}
		case center.CMDCenter_IDHeartBeatRsp:
			//TODO 测试消息
			//log.Trace("agentServer", "暂时是个检测消息")
		default:
			cmd, msg, err := processor.Unmarshal(bm.Cmd.MainCmdID, bm.Cmd.SubCmdID, msgData)
			if err != nil {
				log.Error("agentServer", "异常,agentServer反序列化,headCmd=%v,error: %v", bm.Cmd, err)
				continue
			}
			err = processor.Route(n.BaseMessage{MyMessage: msg, AgentInfo: bm.AgentInfo, TraceId: bm.TraceId}, a, cmd)
			if err != nil {
				log.Error("agentServer", "agentServer route message err=%v,cmd=%v", err, cmd)
				continue
			}
		}
	}
}

func (a *agentServer) OnClose() {
	switch a.info.AppType {
	case n.AppLogger:
		log.SetCallback(nil)
	default:
		break
	}

	log.Debug("agentServer", "服务间连接断开了,%v", util.PrintStructFields(a.info))

	if a.tcpClient != nil && !a.tcpClient.AutoReconnect {
		a.tcpClient.Close()

		mxServers.Lock()
		delete(servers, util.MakeUint64FromUint32(a.info.AppType, a.info.AppId))
		mxServers.Unlock()
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

	//超长判断
	if (len(data) + len(otherData)) > int(MaxMsgLen-1024) {
		log.Error("agentServer", "异常,消息体超长,type=%v,cmd=%v,len=%v,max=%v", reflect.TypeOf(m), bm.Cmd, len(data)+len(otherData), int(MaxMsgLen-1024))
		return
	}

	bm.AgentInfo.AppType = conf.AppInfo.Type
	bm.AgentInfo.AppId = conf.AppInfo.Id
	err = a.conn.WriteMsg(bm, data, otherData)
	if err != nil {
		log.Error("agentServer", "写信息失败,type=%v,cmd=%v,err=%v", reflect.TypeOf(m), bm.Cmd, err)
	}
}

func (a *agentServer) SendData(mainCmdID, subCmdID uint32, m proto.Message) {
	data, err := proto.Marshal(m)
	if err != nil {
		log.Error("agentServer", "异常,proto.Marshal %v error: %v", reflect.TypeOf(m), err)
		return
	}

	//超长判断
	if len(data) > int(MaxMsgLen-1024) {
		log.Error("agentServer", "异常,消息体超长,type=%v,mainCmdID=%v,subCmdID=%v,len=%v,max=%v", reflect.TypeOf(m), mainCmdID, subCmdID, len(data), int(MaxMsgLen-1024))
		return
	}

	bm := n.BaseMessage{}
	bm.AgentInfo.AppType = conf.AppInfo.Type
	bm.AgentInfo.AppId = conf.AppInfo.Id
	bm.Cmd.MainCmdID = uint16(mainCmdID)
	bm.Cmd.SubCmdID = uint16(subCmdID)

	//fmt.Println("SendData,bm=", bm)

	err = a.conn.WriteMsg(bm, data, nil)
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
