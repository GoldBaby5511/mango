package gate

import (
	"github.com/golang/protobuf/proto"
	"reflect"
	"strings"
	"sync"
	"time"
	"xlddz/core/chanrpc"
	"xlddz/core/conf"
	"xlddz/core/conf/apollo"
	"xlddz/core/log"
	"xlddz/core/module"
	n "xlddz/core/network"
	"xlddz/core/network/protobuf"
	"xlddz/core/util"
	"xlddz/protocol/center"
	"xlddz/protocol/gate"
	"xlddz/protocol/logger"
)

//网络事件
const (
	ConnectSuccess   string = "ConnectSuccess"
	Disconnect       string = "Disconnect"
	CenterConnected  string = "CenterConnected"
	CenterDisconnect string = "CenterDisconnect"
	CenterRegResult  string = "CenterRegResult"
)

const (
	AgentIndex = 0
	IdIndex    = 1
)

var (
	cbCenterDisconnect []func()
	tcpLog             *n.TCPClient = nil
	mxServers          sync.Mutex
	wg                 sync.WaitGroup
	servers            map[uint64]*agentServer = make(map[uint64]*agentServer)
	agentChanRPC       *chanrpc.Server
	Skeleton           = module.NewSkeleton(conf.GoLen, conf.TimerDispatcherLen, conf.AsynCallLen, conf.ChanRPCLen)
	processor          = protobuf.NewProcessor()
	MaxConnNum         int
	PendingWriteNum    int
	MaxMsgLen          uint32

	// websocket
	WSAddr      string
	HTTPTimeout time.Duration
	CertFile    string
	KeyFile     string

	// tcp
	LenMsgLen int
	closeSig  chan bool
)

func init() {
	tcpLog = new(n.TCPClient)
	cbCenterDisconnect = append(cbCenterDisconnect, apollo.CenterDisconnect)
	apollo.RegPublicCB(ApolloNotify)
	agentChanRPC = Skeleton.ChanRPCServer
	closeSig = make(chan bool, 1)
}

func Start() {

	if conf.AppInfo.AppType == n.AppCenter {
		apollo.RegisterConfig("", conf.AppInfo.AppType, conf.AppInfo.AppID, nil)
	}

	wg.Add(2)
	go func() {
		Skeleton.Run()
		wg.Done()
	}()

	go func() {
		Run()
		wg.Done()
	}()
}

func Stop() {
	defer util.TryE(conf.AppInfo.AppName)
	Skeleton.Close()
	closeSig <- true
	wg.Wait()
}

func MsgRegister(m proto.Message, mainCmdId uint32, subCmdId uint16, f interface{}) {
	chanRPC := Skeleton.ChanRPCServer
	processor.Register(m, mainCmdId, subCmdId, chanRPC)
	chanRPC.Register(reflect.TypeOf(m), f)
}

func EventRegister(id interface{}, f interface{}) {
	Skeleton.ChanRPCServer.Register(id, f)
}

func Run() {

	log.Debug("", "Run,ListenOnAddress=%v", conf.AppInfo.ListenOnAddress)

	var wsServer *n.WSServer
	if WSAddr != "" {
		wsServer = new(n.WSServer)
		wsServer.Addr = WSAddr
		wsServer.MaxConnNum = MaxConnNum
		wsServer.PendingWriteNum = PendingWriteNum
		wsServer.MaxMsgLen = MaxMsgLen
		wsServer.HTTPTimeout = HTTPTimeout
		wsServer.CertFile = CertFile
		wsServer.KeyFile = KeyFile
		wsServer.NewAgent = func(conn *n.WSConn) n.AgentClient {
			a := &agentClient{conn: conn}
			if agentChanRPC != nil {
				agentChanRPC.Go(ConnectSuccess, a)
			}
			return a
		}
	}

	var tcpServer *n.TCPServer
	if conf.AppInfo.ListenOnAddress != "" {
		tcpServer = new(n.TCPServer)
		tcpServer.Addr = conf.AppInfo.ListenOnAddress
		tcpServer.MaxConnNum = MaxConnNum
		tcpServer.PendingWriteNum = PendingWriteNum
		tcpServer.LenMsgLen = LenMsgLen
		tcpServer.MaxMsgLen = MaxMsgLen
		tcpServer.GetConfig = apollo.GetConfigAsInt64
		tcpServer.NewAgent = func(conn *n.TCPConn, agentId uint64) n.AgentClient {
			a := &agentClient{id: agentId, conn: conn, info: n.BaseAgentInfo{AgentType: n.NormalUser}}
			if agentChanRPC != nil {
				agentChanRPC.Go(ConnectSuccess, a, agentId)
			}
			return a
		}
	}

	if conf.AppInfo.CenterAddr != "" && conf.AppInfo.AppType != n.AppCenter {
		newServerItem(n.BaseAgentInfo{AgentType: n.CommonServer, AppName: "center", AppType: n.AppCenter, ListenOnAddress: conf.AppInfo.CenterAddr}, true, PendingWriteNum)
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

func ApolloNotify(k apollo.ConfKey, v apollo.ConfValue) {
	if conf.AppInfo.AppType != n.AppLogger && k.Key == "日志服务器地址" && v.Value != "" &&
		v.RspCount == 1 && tcpLog != nil && !tcpLog.IsRunning() {
		logAddr := v.Value
		if v, ok := util.ParseArgs("/DockerRun"); ok && v == 1 {
			addr := strings.Split(logAddr, "|")
			logAddr = ""
			for i, v := range addr {
				curConnAddr := v
				curAddr := strings.Split(curConnAddr, ":")
				if len(curAddr) == 2 {
					logAddr = logAddr + "logger:" + curAddr[1]
				}
				if i != len(addr)-1 {
					logAddr = logAddr + "|"
				}
			}
		}
		tcpLog.Addr = logAddr
		tcpLog.AutoReconnect = true
		tcpLog.NewAgent = func(conn *n.TCPConn) n.AgentServer {
			a := &agentServer{tcpClient: tcpLog, conn: conn, info: n.BaseAgentInfo{AgentType: n.CommonServer, AppName: "logger", AppType: n.AppLogger, AppID: 0, ListenOnAddress: logAddr}}
			log.Info("gate", "日志服务器连接成功,Addr=%v", logAddr)
			log.Info("gate", "服务启动完成,阔以开始了... ...")

			log.SetCallback(func(i log.LogInfo) {
				var logReq logger.LogReq
				logReq.FileName = proto.String(i.File)
				logReq.LineNo = proto.Uint32(uint32(i.Line))
				logReq.SrcApptype = proto.Uint32(conf.AppInfo.AppType)
				logReq.SrcAppid = proto.Uint32(conf.AppInfo.AppID)
				logReq.Content = []byte(i.LogStr)
				logReq.ClassName = []byte(i.Classname)
				logReq.LogLevel = proto.Uint32(uint32(i.Level))
				logReq.TimeMs = proto.Uint64(i.TimeMs)
				logReq.SrcAppname = proto.String(conf.AppInfo.AppName)
				cmd := n.TCPCommand{MainCmdID: uint16(n.CMDLogger), SubCmdID: uint16(logger.CMDID_Logger_IDLogReq)}
				bm := n.BaseMessage{MyMessage: &logReq, Cmd: cmd}
				a.SendMessage(bm)
			})

			return a
		}

		tcpLog.Start()
	}
}

func sendRegAppReq(a *agentServer) {
	var registerReq center.RegisterAppReq
	registerReq.AuthKey = proto.String("GoldBaby")
	registerReq.AppName = proto.String(conf.AppInfo.AppName)
	registerReq.AppType = proto.Uint32(conf.AppInfo.AppType)
	registerReq.AppId = proto.Uint32(conf.AppInfo.AppID)
	myAddress := conf.AppInfo.ListenOnAddress
	if v, ok := util.ParseArgs("/DockerRun"); ok && v == 1 {
		addr := strings.Split(conf.AppInfo.ListenOnAddress, ":")
		if len(addr) == 2 {
			myAddress = conf.AppInfo.AppName + ":" + addr[1]
		}
	}
	registerReq.MyAddress = proto.String(myAddress)
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
