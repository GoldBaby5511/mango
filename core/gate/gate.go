package gate

import (
	"google.golang.org/protobuf/proto"
	"sync"
	"time"
	"xlddz/core/chanrpc"
	"xlddz/core/conf"
	"xlddz/core/conf/apollo"
	"xlddz/core/log"
	n "xlddz/core/network"
	"xlddz/protocol/center"
	"xlddz/protocol/gate"
	"xlddz/protocol/logger"
)

//网络事件
const (
	ConnectSuccess  string = "ConnectSuccess"
	Disconnect      string = "Disconnect"
	CenterConnected string = "CenterConnected"
	CenterRegResult string = "CenterRegResult"
)

const (
	AgentIndex = 0
	IdIndex    = 1
)

var (
	cbCenterDisconnect []func()
	tcpLog             *n.TCPClient
	mxServers          sync.Mutex
	servers            map[uint64]*agentServer = make(map[uint64]*agentServer)
	AgentChanRPC       *chanrpc.Server
	Processor          n.Processor
)

func init() {
	if tcpLog == nil {
		tcpLog = new(n.TCPClient)
	}
	cbCenterDisconnect = append(cbCenterDisconnect, apollo.CenterDisconnect)
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
			a := &agentServer{tcpClient: tcpLog, conn: conn, info: n.BaseAgentInfo{AgentType: n.CommonServer, AppName: "logger", AppType: n.AppLogger, AppID: 0, ListenOnAddress: LogAddr}}
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

	TCPClientAddr string
}

//Run module实现
func (gate *Gate) Run(closeSig chan bool) {

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
			a := &agentClient{conn: conn, gate: gate}
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
			a := &agentClient{id: agentId, conn: conn, gate: gate, info: n.BaseAgentInfo{AgentType: n.NormalUser}}
			if AgentChanRPC != nil {
				AgentChanRPC.Go(ConnectSuccess, a, agentId)
			}
			return a
		}
	}

	if gate.TCPClientAddr != "" {
		newServerItem(n.BaseAgentInfo{AgentType: n.CommonServer, AppName: "center", AppType: n.AppCenter, ListenOnAddress: gate.TCPClientAddr}, true, gate.PendingWriteNum)
	}

	if wsServer != nil {
		wsServer.Start()
	}
	if tcpServer != nil {
		tcpServer.Start()
	}

	<-closeSig

	if wsServer != nil {
		wsServer.Close()
	}
	if tcpServer != nil {
		tcpServer.Close()
	}
	if tcpLog != nil {
		tcpLog.Close()
	}
}

//OnDestroy module实现
func (gate *Gate) OnDestroy() {}

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

func SendMessage2Client(bm n.BaseMessage, gateConnID, sessionID uint64) {
	var dataReq gate.TransferDataReq
	dataReq.AttApptype = proto.Uint32(n.AppGate)
	dataReq.AttAppid = proto.Uint32(uint32(gateConnID >> 32))
	dataReq.DataCmdKind = proto.Uint32(uint32(bm.Cmd.MainCmdID))
	dataReq.DataCmdSubid = proto.Uint32(uint32(bm.Cmd.SubCmdID))
	dataReq.Data, _ = proto.Marshal(bm.MyMessage.(proto.Message))
	dataReq.Gateconnid = proto.Uint64(gateConnID)
	dataReq.AttSessionid = proto.Uint64(sessionID)
	cmd := n.TCPCommand{MainCmdID: uint16(n.CMDGate), SubCmdID: uint16(gate.CMDID_Gate_IDTransferDataReq)}
	transBM := n.BaseMessage{MyMessage: &dataReq, Cmd: cmd, TraceId: bm.TraceId}
	destAgents := getDestAppInfo(n.AppGate, uint32(gateConnID>>32))
	for _, a := range destAgents {
		a.SendMessage(transBM)
	}
}

func getDestAppInfo(destAppType, destAppid uint32) []*agentServer {
	mxServers.Lock()
	defer mxServers.Unlock()
	var destAgent []*agentServer
	destTypeAppCount := func() int {
		destCount := 0
		for _, v := range servers {
			if v.info.AppType == destAppType {
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
				if v.info.AppType == destAppType {
					destAgent = append(destAgent, v)
				}
			}
			sendResult = true
		case n.Send2AnyOne:
			for _, v := range servers {
				if v.info.AppType == destAppType {
					destAgent = append(destAgent, v)
					sendResult = true
					break
				}
			}
		default:
			for _, v := range servers {
				if v.info.AppType == destAppType && v.info.AppID == destAppid {
					destAgent = append(destAgent, v)
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
