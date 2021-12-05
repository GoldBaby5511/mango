package gate

import (
	"google.golang.org/protobuf/proto"
	"net"
	"reflect"
	"time"
	"xlddz/core/chanrpc"
	"xlddz/core/conf"
	"xlddz/core/conf/apollo"
	"xlddz/core/log"
	n "xlddz/core/network"
	"xlddz/protocol/center"
	"xlddz/protocol/config"
	"xlddz/protocol/gate"
	"xlddz/protocol/logger"
)

//网络事件
const (
	ConnectSuccess  string = "ConnectSuccess"
	Disconnect      string = "Disconnect"
	RouterConnected string = "RouterConnected"
	RouterRegResult string = "RouterRegResult"
)

const (
	AgentIndex = 0
	IdIndex    = 1
)

var (
	cbRouterDisconnect []func() //router断开回调
	tcpLog             *n.TCPClient
	servers            map[uint64]*ServerItem = make(map[uint64]*ServerItem)
	AgentChanRPC       *chanrpc.Server
	Processor          n.Processor
)

func init() {
	if tcpLog == nil {
		tcpLog = new(n.TCPClient)
	}
	cbRouterDisconnect = append(cbRouterDisconnect, apollo.RouterDisconnect)
	apollo.RegPublicCB(ApolloNotify)
}

func ApolloNotify(k apollo.ConfKey, v apollo.ConfValue) {
	//得到日志服务
	if conf.AppType != n.AppLogger && k.Key == "日志服务器地址" && v.Value != "" &&
		v.RspCount == 1 && tcpLog != nil && !tcpLog.IsRunning() {
		LogAddr := v.Value
		tcpLog.Addr = LogAddr
		tcpLog.AutoReconnect = true
		tcpLog.NewAgent = func(conn *n.TCPConn) n.AgentServer {
			a := &agentServer{conn: conn}
			a.s = &ServerItem{tcpClient: tcpLog, a: a, AppName: "logger", AppType: n.AppLogger, AppID: 0, Address: LogAddr}
			log.Info("gate", "日志服务器连接成功,Addr=%v", LogAddr)
			log.Info("gate", "服务启动完成,阔以开始了... ...")

			log.SetCallback(func(i log.LogInfo) {
				var logReq logger.LogReq
				logReq.FileName = proto.String(i.File)
				logReq.LineNo = proto.Uint32(uint32(i.Line))
				logReq.SrcApptype = proto.Uint32(conf.AppType)
				logReq.SrcAppid = proto.Uint32(conf.AppID)
				logReq.Content = []byte(i.LogStr)
				logReq.ClassName = []byte(i.Classname)
				logReq.LogLevel = proto.Uint32(uint32(i.Level))
				logReq.TimeMs = proto.Uint64(i.TimeMs)
				logReq.SrcAppname = proto.String(conf.AppName)
				cmd := n.TCPCommand{MainCmdID: uint16(n.CMDLogger), SubCmdID: uint16(logger.CMDID_Logger_IDLogReq)}
				bm := n.BaseMessage{MyMessage: &logReq, Cmd: cmd}
				a.SendMessage(bm)
			})

			return a
		}

		tcpLog.Start()
	}
}

//Gate 服务端网关
type Gate struct {
	MaxConnNum      int
	PendingWriteNum int
	MaxMsgLen       uint32

	// websocket
	WSAddr      string
	HTTPTimeout time.Duration
	CertFile    string
	KeyFile     string

	// tcp
	TCPAddr      string
	LenMsgLen    int
	LittleEndian bool

	//tcpRouterClient
	TCPClientAddr string
}

//Run module实现
func (gate *Gate) Run(closeSig chan bool) {

	log.Info("gate", "网关执行,TCPAddr=%v,RouterAddr=%v",
		gate.TCPAddr, gate.TCPClientAddr)

	var wsServer *n.WSServer
	if gate.WSAddr != "" {
		wsServer = new(n.WSServer)
		wsServer.Addr = gate.WSAddr
		wsServer.MaxConnNum = gate.MaxConnNum
		wsServer.PendingWriteNum = gate.PendingWriteNum
		wsServer.MaxMsgLen = gate.MaxMsgLen
		wsServer.HTTPTimeout = gate.HTTPTimeout
		wsServer.CertFile = gate.CertFile
		wsServer.KeyFile = gate.KeyFile
		wsServer.NewAgent = func(conn *n.WSConn) n.AgentClient {
			a := &agent{conn: conn, gate: gate}
			if AgentChanRPC != nil {
				AgentChanRPC.Go(ConnectSuccess, a)
			}
			return a
		}
	}

	var tcpServer *n.TCPServer
	if gate.TCPAddr != "" {
		tcpServer = new(n.TCPServer)
		tcpServer.Addr = gate.TCPAddr
		tcpServer.MaxConnNum = gate.MaxConnNum
		tcpServer.PendingWriteNum = gate.PendingWriteNum
		tcpServer.LenMsgLen = gate.LenMsgLen
		tcpServer.MaxMsgLen = gate.MaxMsgLen
		tcpServer.LittleEndian = gate.LittleEndian
		tcpServer.GetConfig = apollo.GetConfigAsInt64
		tcpServer.NewAgent = func(conn *n.TCPConn, agentId uint64) n.AgentClient {
			a := &agent{id: agentId, conn: conn, gate: gate, info: n.BaseAgentInfo{AgentType: n.NormalUser}}
			if AgentChanRPC != nil {
				AgentChanRPC.Go(ConnectSuccess, a, agentId)
			}
			return a
		}
	}

	//router连接
	var tcpRouterClient *n.TCPClient
	if gate.TCPClientAddr != "" {
		tcpRouterClient = new(n.TCPClient)
		tcpRouterClient.Addr = gate.TCPClientAddr
		tcpRouterClient.PendingWriteNum = gate.PendingWriteNum
		tcpRouterClient.AutoReconnect = true
		tcpRouterClient.NewAgent = func(conn *n.TCPConn) n.AgentServer {
			a := &agentServer{conn: conn}
			a.s = &ServerItem{tcpClient: tcpLog, a: a, AppName: "router", AppType: n.AppCenter, AppID: 0, Address: gate.TCPClientAddr}
			servers[(uint64(n.AppCenter)<<32 | uint64(0))] = a.s
			if AgentChanRPC != nil {
				AgentChanRPC.Go(RouterConnected, a)
			}

			log.Info("agent", "Router连接成功,Addr=%v", gate.TCPClientAddr)

			//连接成功发送注册命令
			sendRegAppReq(a)

			//启动心跳
			timeInterval := 30 * time.Second
			timerHeartbeat := time.NewTimer(timeInterval)
			go func(t *time.Timer) {
				for {
					<-t.C
					var appPulseNotify center.AppPulseNotify
					appPulseNotify.Action = (*center.AppPulseNotify_PulseAction)(proto.Int32(int32(center.AppPulseNotify_HeartBeatReq)))
					appPulseNotify.PulseData = proto.Uint64(uint64(time.Now().Unix()))
					a.SendData(n.CMDCenter, uint32(center.CMDID_Center_IDPulseNotify), &appPulseNotify)

					t.Reset(timeInterval)
				}
			}(timerHeartbeat)
			return a
		}
	}

	if wsServer != nil {
		wsServer.Start()
	}
	if tcpServer != nil {
		tcpServer.Start()
	}

	if tcpRouterClient != nil {
		tcpRouterClient.Start()
	}
	<-closeSig

	if wsServer != nil {
		wsServer.Close()
	}
	if tcpServer != nil {
		tcpServer.Close()
	}
	if tcpRouterClient != nil {
		tcpRouterClient.Close()
	}
	if tcpLog != nil {
		tcpLog.Close()
	}
}

//OnDestroy module实现
func (gate *Gate) OnDestroy() {}

//代理
type agent struct {
	id   uint64
	conn n.Conn
	gate *Gate
	info n.BaseAgentInfo
}

func (a *agent) Run() {

	for {
		bm, msgData, err := a.conn.ReadMsg()
		if err != nil {
			log.Warning("agent", "异常,网关读取消息失败,id=%v,err=%v", a.id, err)
			break
		}
		if Processor == nil {
			log.Error("agent", "异常,解析器为nil断开连接,cmd=%v", &bm.Cmd)
			break
		}

		//构造参数，全新改造中，暂时这么用着
		if bm.Cmd.MainCmdID == uint16(n.CMDConfig) && bm.Cmd.SubCmdID == uint16(config.CMDID_Config_IDApolloCfgRsp) {
			log.Debug("", "收到配置中心消息2")
			apollo.ProcessReq(&bm.Cmd, msgData)
			continue
		}

		if conf.AppType != n.AppCenter && bm.Cmd.MainCmdID == uint16(n.CMDCenter) && bm.Cmd.SubCmdID == uint16(center.CMDID_Center_IDAppRegReq) {
			var m center.RegisterAppReq
			_ = proto.Unmarshal(msgData, &m)
			a.info = n.BaseAgentInfo{AgentType: n.CommonServer, AppName: m.GetAppName(), AppType: m.GetAppType(), AppID: m.GetAppId()}
			log.Debug("", "相互注册,%v", a.info)
			continue
		}

		if bm.Cmd.MainCmdID == uint16(n.CMDGate) && bm.Cmd.SubCmdID == uint16(gate.CMDID_Gate_IDTransferDataReq) {
			if conf.AppType == n.AppGate {
				cmd, msg, err := Processor.Unmarshal(bm.Cmd.MainCmdID, bm.Cmd.SubCmdID, msgData)
				if err != nil {
					log.Error("agent", "unmarshal message,headCmd=%v,error: %v", bm.Cmd, err)
					continue
				}

				baseMsg := n.BaseMessage{MyMessage: msg, TraceId: bm.TraceId}
				err = Processor.Route(baseMsg, a, cmd, msgData)
				if err != nil {
					log.Error("agent", "client agent route message error: %v,cmd=%v", err, cmd)
					continue
				}
			} else {
				log.Debug("", "gate二次解密,%v,%v", bm.Cmd, a.info.AgentType)
				var m gate.TransferDataReq
				_ = proto.Unmarshal(msgData, &m)

				cmd, msg, err := Processor.Unmarshal(uint16(m.GetDataCmdKind()), uint16(m.GetDataCmdSubid()), m.GetData())
				if err != nil {
					log.Error("agent", "unmarshal message,headCmd=%v,error: %v", bm.Cmd, err)
					continue
				}

				baseMsg := n.BaseMessage{MyMessage: msg, TraceId: bm.TraceId}
				err = Processor.Route(baseMsg, a, cmd, &m)
				if err != nil {
					log.Error("agent", "路由失败, err=%v,cmd=%v", err, cmd)
					continue
				}
			}
			continue
		}

		cmd, msg, err := Processor.Unmarshal(bm.Cmd.MainCmdID, bm.Cmd.SubCmdID, msgData)
		if err != nil {
			log.Error("agent", "unmarshal message,headCmd=%v,error: %v", bm.Cmd, err)
			continue
		}

		baseMsg := n.BaseMessage{MyMessage: msg, TraceId: bm.TraceId}
		err = Processor.Route(baseMsg, a, cmd, msgData)
		if err != nil {
			log.Error("agent", "client agent route message error: %v,cmd=%v", err, cmd)
			continue
		}

		//if bm.Cmd.SubCmdID != uint16(gate.CMDID_Gate_IDTransferDataReq) {
		//	cmd, msg, err := Processor.Unmarshal(bm.Cmd.MainCmdID, bm.Cmd.SubCmdID, msgData)
		//	if err != nil {
		//		log.Error("agent", "unmarshal message,headCmd=%v,error: %v", bm.Cmd, err)
		//		continue
		//	}
		//
		//	baseMsg := n.BaseMessage{MyMessage: msg, TraceId: bm.TraceId}
		//	err = Processor.Route(baseMsg, a, cmd, msgData)
		//	if err != nil {
		//		log.Error("agent", "client agent route message error: %v,cmd=%v", err, cmd)
		//		continue
		//	}
		//} else {
		//	log.Debug("", "gate二次解密,%v,%v", bm.Cmd, a.info.AgentType)
		//	var m gate.TransferDataReq
		//	_ = proto.Unmarshal(msgData, &m)
		//	var cmd, msg, err interface{} = nil, nil, nil
		//	//Gateway服务是一个特例，发往客户端的消息需要外层逻辑处理，暂时以对外监听地址做区别，以当前服务拓扑结构连router且对外的只有网关
		//	if m.GetDataDirection() == uint32(center.EnuDataDirection_DT_App2Client) && conf.AppType == n.AppGate {
		//		msg = &m
		//		cmd = &n.TCPCommand{MainCmdID: uint16(n.CMDCenter), SubCmdID: uint16(center.CMDID_Center_IDDataMessageReq)}
		//	} else {
		//		cmd, msg, err = Processor.Unmarshal(uint16(bm.Cmd.MainCmdID), uint16(bm.Cmd.SubCmdID), msgData)
		//		if err != nil {
		//			log.Error("agent", "unmarshal message,headCmd=%v,error: %v", bm.Cmd, err)
		//			continue
		//		}
		//	}
		//	baseMsg := n.BaseMessage{MyMessage: msg, TraceId: bm.TraceId}
		//	err = Processor.Route(baseMsg, a, cmd, &m)
		//	if err != nil {
		//		log.Error("agent", "路由失败, err=%v,cmd=%v", err, cmd)
		//		continue
		//	}
		//}
	}
}

func (a *agent) OnClose() {
	if AgentChanRPC != nil {
		err := AgentChanRPC.Call0(Disconnect, a, a.id)
		if err != nil {
			log.Error("agent", "chanrpc error: %v", err)
		}
	}
}

func (a *agent) LocalAddr() net.Addr {
	return a.conn.LocalAddr()
}

func (a *agent) RemoteAddr() net.Addr {
	return a.conn.RemoteAddr()
}

func (a *agent) Close() {
	a.conn.Close()
}

func (a *agent) Destroy() {
	a.conn.Destroy()
}

func (a *agent) SendData(mainCmdID, subCmdID uint32, m proto.Message) {
	data, err := proto.Marshal(m)
	if err != nil {
		log.Error("agent", "异常,proto.Marshal %v error: %v", reflect.TypeOf(m), err)
		return
	}
	err = a.conn.WriteMsg(uint16(mainCmdID), uint16(subCmdID), data, nil)
	if err != nil {
		log.Error("agent", "write message %v error: %v", reflect.TypeOf(m), err)
	}
}

func (a *agent) AgentInfo() n.BaseAgentInfo {
	return a.info
}

func sendRegAppReq(a *agentServer) {
	var registerReq center.RegisterAppReq
	registerReq.AuthKey = proto.String("GoldBaby")
	registerReq.AppName = proto.String(conf.AppName)
	registerReq.AppType = proto.Uint32(conf.AppType)
	registerReq.AppId = proto.Uint32(conf.AppID)
	registerReq.MyAddress = proto.String(conf.ListenOnAddress)
	a.SendData(n.CMDCenter, uint32(center.CMDID_Center_IDAppRegReq), &registerReq)
}

func SendData2App(destAppType, destAppid, mainCmdID, subCmdID uint32, m proto.Message) {
	cmd := n.TCPCommand{MainCmdID: uint16(mainCmdID), SubCmdID: uint16(subCmdID)}
	bm := n.BaseMessage{MyMessage: m, Cmd: cmd}
	destAgents := getDestAppInfo(destAppType, destAppid)
	for _, a := range destAgents {
		a.SendMessage(bm)
	}
}

func SendMessage2Client(bm n.BaseMessage, userID, gateConnID, sessionID uint64) {
	var dataReq gate.TransferDataReq
	dataReq.AttApptype = proto.Uint32(uint32(gateConnID >> 32))
	dataReq.AttAppid = proto.Uint32(n.AppGate)
	dataReq.DataCmdKind = proto.Uint32(uint32(bm.Cmd.MainCmdID))
	dataReq.DataCmdSubid = proto.Uint32(uint32(bm.Cmd.SubCmdID))
	dataReq.DataDirection = proto.Uint32(uint32(center.EnuDataDirection_DT_App2Client))
	dataReq.Data, _ = proto.Marshal(bm.MyMessage.(proto.Message))
	dataReq.AttUserid = proto.Uint64(userID)
	dataReq.AttGateconnid = proto.Uint64(gateConnID)
	dataReq.AttSessionid = proto.Uint64(sessionID)
	cmd := n.TCPCommand{MainCmdID: uint16(n.CMDGate), SubCmdID: uint16(gate.CMDID_Gate_IDTransferDataReq)}
	transBM := n.BaseMessage{MyMessage: &dataReq, Cmd: cmd, TraceId: bm.TraceId}
	destAgents := getDestAppInfo(n.AppGate, uint32(gateConnID>>32))
	for _, a := range destAgents {
		a.SendMessage(transBM)
	}
}

//func SendMessage2Client(bm n.BaseMessage, userID, gateConnID, sessionID uint64) {
//	var dataReq gate.TransferDataReq
//	dataReq.DestAppid = proto.Uint32(uint32(gateConnID >> 32))
//	dataReq.DestApptype = proto.Uint32(n.AppGate)
//	dataReq.DataCmdkind = proto.Uint32(uint32(bm.Cmd.MainCmdID))
//	dataReq.DataCmdsubid = proto.Uint32(uint32(bm.Cmd.SubCmdID))
//	dataReq.DataDirection = proto.Uint32(uint32(center.EnuDataDirection_DT_App2Client))
//	dataReq.DataBuff, _ = proto.Marshal(bm.MyMessage.(proto.Message))
//	dataReq.AttUserid = proto.Uint64(userID)
//	dataReq.AttGateconnid = proto.Uint64(gateConnID)
//	dataReq.AttSessionid = proto.Uint64(sessionID)
//	cmd := n.TCPCommand{MainCmdID: uint16(n.CMDCenter), SubCmdID: uint16(center.CMDID_Center_IDDataMessageReq)}
//	transBM := n.BaseMessage{MyMessage: &dataReq, Cmd: cmd, TraceId: bm.TraceId}
//	destAgents := getDestAppInfo(n.AppGate, uint32(gateConnID>>32))
//	for _, a := range destAgents {
//		a.SendMessage(transBM)
//	}
//}

func getDestAppInfo(destAppType, destAppid uint32) []*agentServer {
	var destAgent []*agentServer
	destTypeAppCount := func() int {
		destCount := 0
		for _, v := range servers {
			if v.AppType == destAppType {
				destCount++
			}
		}
		return destCount
	}
	sendResult := false
	if destTypeAppCount() != 0 {
		switch destAppid {
		case n.Send2All:
			for _, v := range servers {
				if v.AppType == destAppType {
					destAgent = append(destAgent, v.a)
				}
			}
			sendResult = true
		case n.Send2AnyOne:
			for _, v := range servers {
				if v.AppType == destAppType {
					destAgent = append(destAgent, v.a)
					sendResult = true
					break
				}
			}
		default:
			for _, v := range servers {
				if v.AppType == destAppType && v.AppID == destAppid {
					destAgent = append(destAgent, v.a)
					sendResult = true
					break
				}
			}
		}
	}

	if !sendResult {
		log.Error("转发", "异常,消息转发失败,%v,destAppType=%v,destAppid=%v",
			destTypeAppCount(), destAppType, destAppid)
	}

	return destAgent
}
