package gate

import (
	"google.golang.org/protobuf/proto"
	"net"
	"reflect"
	"sync"
	"time"
	"xlddz/core/chanrpc"
	"xlddz/core/conf"
	"xlddz/core/conf/apollo"
	"xlddz/core/log"
	n "xlddz/core/network"
	"xlddz/protocol/logger"
	"xlddz/protocol/router"
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
	routerMsgChan      []chan []interface{} //router消息并发通道
	cbRouterDisconnect []func()             //router断开回调
	tcpLog             *n.TCPClient
)

func init() {
	if tcpLog == nil {
		tcpLog = new(n.TCPClient)
	}
	cbRouterDisconnect = append(cbRouterDisconnect, apollo.RouterDisconnect)
	apollo.RegPublicCB(ApolloNotify)
}

//apollo配置第一次获取成功回调
func ApolloNotify(k apollo.ConfKey, v apollo.ConfValue) {
	//得到日志服务
	if conf.AppType != n.AppLogger && k.Key == "日志服务器地址" && v.Value != "" &&
		v.RspCount == 1 && tcpLog != nil && !tcpLog.IsRunning() {
		LogAddr := v.Value
		tcpLog.Addr = LogAddr
		tcpLog.NewAgent = func(conn *n.TCPConn) n.Agent {
			a := &agent{conn: conn, agentType: AGENT_TYPE_LOGGER}
			log.Info("gate", "日志服务器连接成功,Addr=%v", LogAddr)
			log.Info("gate", "服务启动完成,阔以开始了... ...")

			callChan := make(chan log.LogInfo)
			//这里边不能再有log调用否则就是死循环
			go func() {
				for {
					logInfo := <-callChan

					var logReq logger.LogReq
					logReq.FileName = proto.String(logInfo.File)
					logReq.LineNo = proto.Uint32(uint32(logInfo.Line))
					logReq.SrcApptype = proto.Uint32(conf.AppType)
					logReq.SrcAppid = proto.Uint32(conf.AppID)
					logReq.Content = []byte(logInfo.LogStr)
					logReq.ClassName = []byte(logInfo.Classname)
					logReq.LogLevel = proto.Uint32(uint32(logInfo.Level))
					logReq.TimeMs = proto.Uint64(logInfo.TimeMs)
					logReq.SrcAppname = proto.String(conf.AppName)
					a.SendData(n.CMDLogger, uint32(logger.CMDID_Logger_IDLogReq), &logReq)
				}
			}()

			log.SetLogCallBack(callChan)
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
	Processor       n.Processor
	AgentChanRPC    *chanrpc.Server

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
		wsServer.NewAgent = func(conn *n.WSConn) n.Agent {
			a := &agent{conn: conn, gate: gate, agentType: AGENT_TYPE_SERVER}
			if gate.AgentChanRPC != nil {
				gate.AgentChanRPC.Go(ConnectSuccess, a)
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
		tcpServer.NewAgent = func(conn *n.TCPConn, agentId uint64) n.Agent {
			a := &agent{id: agentId, conn: conn, gate: gate, agentType: AGENT_TYPE_SERVER}
			if gate.AgentChanRPC != nil {
				gate.AgentChanRPC.Go(ConnectSuccess, a, agentId)
			}
			return a
		}
	}

	//router连接
	var wg sync.WaitGroup
	closeMsg := make(chan bool)
	var tcpRouterClient *n.TCPClient
	if gate.TCPClientAddr != "" {
		tcpRouterClient = new(n.TCPClient)
		tcpRouterClient.Addr = gate.TCPClientAddr
		tcpRouterClient.PendingWriteNum = gate.PendingWriteNum
		tcpRouterClient.NewAgent = func(conn *n.TCPConn) n.Agent {
			a := &agent{conn: conn, gate: gate, agentType: AGENT_TYPE_ROUTER}
			if gate.AgentChanRPC != nil {
				gate.AgentChanRPC.Go(RouterConnected, a)
			}

			apollo.SetRouterAgent(a)
			log.Info("agent", "Router连接成功,Addr=%v", a.gate.TCPClientAddr)

			//连接成功发送注册命令
			var registerReq router.RegisterAppReq
			registerReq.AuthKey = proto.String("2")
			registerReq.AppType = proto.Uint32(conf.AppType)
			registerReq.AppId = proto.Uint32(conf.AppID)
			a.SendData(n.CMDRouter, uint32(router.CMDID_Router_IDAppRegReq), &registerReq)

			//启动心跳
			timeInterval := 30 * time.Second
			timerHeartbeat := time.NewTimer(timeInterval)
			go func(t *time.Timer) {
				for {
					<-t.C
					var appPulseNotify router.AppPulseNotify
					appPulseNotify.Action = (*router.AppPulseNotify_PulseAction)(proto.Int32(int32(router.AppPulseNotify_HeartBeatReq)))
					appPulseNotify.PulseData = proto.Uint64(uint64(time.Now().Unix()))
					a.SendData(n.CMDRouter, uint32(router.CMDID_Router_IDPulseNotify), &appPulseNotify)

					t.Reset(timeInterval)
				}
			}(timerHeartbeat)
			return a
		}

		//创建异步处理协程 默认创建1000个
		if conf.RouterGoroutineNum < 0 || conf.RouterGoroutineNum > 1000 {
			conf.RouterGoroutineNum = 1000
		}
		//重连情况下不需要重建,运行时改变 conf.RouterGoroutineNum可能发生泄漏
		if len(routerMsgChan) != conf.RouterGoroutineNum {
			for i := 0; i < conf.RouterGoroutineNum; i++ {
				wg.Add(1)
				mc := make(chan []interface{})
				go func(mc chan []interface{}) {
					defer wg.Done()
					for {
						select {
						case args := <-mc:
							err := gate.Processor.Route(args...)
							if err != nil {
								log.Error("agent", "client agent route message error: %v", err)
								continue
							}
						case <-closeMsg:
							return
						}
					}
				}(mc)
				routerMsgChan = append(routerMsgChan, mc)
			}
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
	for i := 0; i < len(routerMsgChan); i++ {
		closeMsg <- true
	}
	wg.Wait()

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

//代理类型
const (
	AGENT_TYPE_SERVER int = 0 //服务
	AGENT_TYPE_ROUTER int = 1 //router
	AGENT_TYPE_LOGGER int = 2 //log
)

//代理
type agent struct {
	id        uint64
	conn      n.Conn
	gate      *Gate
	agentType int //代理类型
}

func (a *agent) Run() {
	for {
		bm, msgData, err := a.conn.ReadMsg()
		if err != nil {
			log.Warning("gate", "异常,网关读取消息失败,err=%v", err)
			break
		}

		//构造参数，全新改造中，暂时这么用着
		headCmd := &bm.Cmd
		if !a.preDealFrameMsg(headCmd, msgData) {
			if a.gate.Processor == nil {
				log.Error("agent", "异常,解析器为nil断开连接,cmd=%v", headCmd)
				break
			}

			//类型判断
			if a.agentType == AGENT_TYPE_SERVER {
				cmd, msg, err := a.gate.Processor.Unmarshal(headCmd.MainCmdID, headCmd.SubCmdID, msgData)
				if err != nil {
					log.Error("agent", "unmarshal message,agentType=%v,headCmd=%v,error: %v", a.agentType, headCmd, err)
					continue
				}

				err = a.gate.Processor.Route(msg, a, cmd, msgData)
				if err != nil {
					log.Error("agent", "client agent route message error: %v,cmd=%v", err, cmd)
					continue
				}
			} else if a.agentType == AGENT_TYPE_ROUTER {
				//理论上只有这个消息
				if headCmd.SubCmdID != uint16(router.CMDID_Router_IDDataMessageReq) {
					log.Error("agent", "这里理论上是不可能出现的!!!,出现未知消息,cmd=%v", headCmd)
					continue
				}

				var m router.DataTransferReq
				_ = proto.Unmarshal(msgData, &m)
				var cmd, msg, err interface{} = nil, nil, nil
				//Gateway服务是一个特例，发往客户端的消息需要外层逻辑处理，暂时以对外监听地址做区别，以当前服务拓扑结构连router且对外的只有网关
				if m.GetDataDirection() == uint32(router.EnuDataDirection_DT_App2Client) && a.gate.TCPAddr != "" {
					msg = &m
					cmd = &n.TCPCommand{MainCmdID: uint16(n.CMDRouter), SubCmdID: uint16(router.CMDID_Router_IDDataMessageReq)}
				} else {
					cmd, msg, err = a.gate.Processor.Unmarshal(uint16(m.GetDataCmdkind()), uint16(m.GetDataCmdsubid()), m.GetDataBuff())
					if err != nil {
						log.Error("agent", "unmarshal message,agentType=%v,headCmd=%v,error: %v", a.agentType, headCmd, err)
						continue
					}
				}

				//异步到协程内处理，优先用户的connid，默认第一个了
				indexChan := 0
				if m.GetAttGateconnid() != 0 {
					indexChan = int(m.GetAttGateconnid()) % len(routerMsgChan)
				} else if m.GetSrcAppid() != 0 {
					indexChan = int(m.GetSrcAppid()) % len(routerMsgChan)
				}

				//投递消息
				if indexChan >= 0 && indexChan < len(routerMsgChan) {
					baseMsg := n.BaseMessage{MyMessage: msg, TraceId: bm.TraceId}
					routerMsgChan[indexChan] <- []interface{}{baseMsg, a, cmd, &m}
				} else {
					log.Error("agent", "agent异步消息时出错,cmd=%v,indexChan=%v,len(routerMsgChan)=%v", cmd, indexChan, len(routerMsgChan))
					continue
				}

			} else {
				log.Error("agent", "快跑吧！这是日志服务器都给你消息了吗？,agentType=%v,headCmd=%v", a.agentType, headCmd)
				break
			}
		}
	}
}

func (a *agent) OnClose() {
	if a.agentType == AGENT_TYPE_LOGGER {
		log.SetLogCallBack(nil)
		log.Info("agent", "日志服务器断开")
	} else {
		if a.gate.AgentChanRPC != nil {
			err := a.gate.AgentChanRPC.Call0(Disconnect, a, a.id)
			if err != nil {
				log.Error("agent", "chanrpc error: %v", err)
			}
		}

		//连接关闭了
		if a.agentType == AGENT_TYPE_ROUTER {
			//router断了世界应该被重启一次
			log.Error("agent", "异常,与router连接断开,世界需要重启... ...")
			for _, cb := range cbRouterDisconnect {
				cb()
			}
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

//框架消息处理,返回true则不在丢到业务层处理
func (a *agent) preDealFrameMsg(cmd *n.TCPCommand, data []byte) bool {

	//对外监听但又不连接router的那肯定就是router服务本身了
	if a.agentType == AGENT_TYPE_SERVER && a.gate.TCPClientAddr == "" {
		return false
	}

	//消息处理
	if cmd.MainCmdID == uint16(n.CMDRouter) {
		switch cmd.SubCmdID {
		case uint16(router.CMDID_Router_IDAppRegRsp): //router注册消息
			var m router.RegisterAppRsp
			_ = proto.Unmarshal(data, &m)

			if m.GetRegResult() == 0 {
				log.Info("agent", "注册成功,regToken=%v,RouterId=%v",
					m.GetReregToken(), m.GetRouterId())

				//获取配置
				if n.AppConfig != conf.AppType {
					apollo.RegisterConfig("", conf.AppType, conf.AppID, nil)
				}
			} else {
				log.Warning("agent", "注册失败,RouterId=%v,原因=%v", m.GetRouterId(), m.GetReregToken())
			}
			if a.gate.AgentChanRPC != nil {
				a.gate.AgentChanRPC.Call0(RouterRegResult, m.GetRegResult(), m.GetRouterId())
			}
		case uint16(router.CMDID_Router_IDAppState): //app状态改变
			var m router.AppStateNotify
			_ = proto.Unmarshal(data, &m)
			log.Debug("agent", "app状态改变 AppState=%v,RouterId=%v,AppType=%v,AppId=%v",
				m.GetAppState(), m.GetRouterId(), m.GetAppType(), m.GetAppId())
		case uint16(router.CMDID_Router_IDDataMessageReq): //普通消息
			var m router.DataTransferReq
			_ = proto.Unmarshal(data, &m)
			//配置中心消息
			if m.GetDataCmdkind() == n.CMDConfig && n.AppConfig != conf.AppType {
				apollo.ProcessReq(&m)
				return true
			}
			return false
		case uint16(router.CMDID_Router_IDPulseNotify): //心跳
		default:
			log.Error("agent", "n.CMDRouter,异常,还未处理消息,%v", cmd)
		}
		return true
	}

	return false
}

func (a *agent) SendMessage(bm n.BaseMessage) {
	m := bm.MyMessage.(proto.Message)
	data, err := proto.Marshal(m)
	if err != nil {
		log.Error("agent", "异常,proto.Marshal %v error: %v", reflect.TypeOf(m), err)
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
		log.Error("agent", "写信息失败 %v error: %v", reflect.TypeOf(m), err)
	}
}

func (a *agent) SendMessage2App(destAppType, destAppid uint32, bm n.BaseMessage) {
	dataReq := a.getTranData(destAppid, destAppType, uint32(bm.Cmd.MainCmdID), uint32(bm.Cmd.SubCmdID), uint32(router.EnuDataDirection_DT_App2App))
	dataReq.DataBuff, _ = proto.Marshal(bm.MyMessage.(proto.Message))
	cmd := n.TCPCommand{MainCmdID: uint16(n.CMDRouter), SubCmdID: uint16(router.CMDID_Router_IDDataMessageReq)}
	transBM := n.BaseMessage{MyMessage: &dataReq, Cmd: cmd, TraceId: bm.TraceId}
	a.SendMessage(transBM)
}

func (a *agent) SendMessage2Client(bm n.BaseMessage, userID, gateConnID, sessionID uint64) {
	dataReq := a.getTranData(uint32(gateConnID>>32), n.AppGate, uint32(bm.Cmd.MainCmdID), uint32(bm.Cmd.SubCmdID), uint32(router.EnuDataDirection_DT_App2Client))
	dataReq.DataBuff, _ = proto.Marshal(bm.MyMessage.(proto.Message))
	dataReq.AttUserid = proto.Uint64(userID)
	dataReq.AttGateconnid = proto.Uint64(gateConnID)
	dataReq.AttSessionid = proto.Uint64(sessionID)
	cmd := n.TCPCommand{MainCmdID: uint16(n.CMDRouter), SubCmdID: uint16(router.CMDID_Router_IDDataMessageReq)}
	transBM := n.BaseMessage{MyMessage: &dataReq, Cmd: cmd, TraceId: bm.TraceId}
	a.SendMessage(transBM)
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

func (a *agent) SendData2App(destAppType, destAppid, mainCmdID, subCmdID uint32, m proto.Message) {
	dataReq := a.getTranData(destAppid, destAppType, mainCmdID, subCmdID, uint32(router.EnuDataDirection_DT_App2App))
	dataReq.DataBuff, _ = proto.Marshal(m)
	cmd := n.TCPCommand{MainCmdID: uint16(n.CMDRouter), SubCmdID: uint16(router.CMDID_Router_IDDataMessageReq)}
	bm := n.BaseMessage{MyMessage: &dataReq, Cmd: cmd}
	a.SendMessage(bm)
}

func (a *agent) getTranData(destAppid, destAppType, dataKind, dataSubId, direction uint32) router.DataTransferReq {
	var dataReq router.DataTransferReq
	dataReq.SrcAppid = proto.Uint32(conf.AppID)
	dataReq.SrcApptype = proto.Uint32(conf.AppType)
	dataReq.DestAppid = proto.Uint32(destAppid)
	dataReq.DestApptype = proto.Uint32(destAppType)
	dataReq.DataCmdkind = proto.Uint32(dataKind)
	dataReq.DataCmdsubid = proto.Uint32(dataSubId)
	dataReq.DataDirection = proto.Uint32(direction)
	return dataReq
}
