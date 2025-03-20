package gate

import (
	"encoding/json"
	"fmt"
	"mango/pkg/amqp"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"mango/api/center"
	"mango/api/config"
	"mango/api/gateway"
	"mango/api/logger"
	"mango/pkg/chanrpc"
	"mango/pkg/conf"
	"mango/pkg/conf/apollo"
	"mango/pkg/console"
	"mango/pkg/database"
	"mango/pkg/log"
	"mango/pkg/module"
	n "mango/pkg/network"
	"mango/pkg/network/protobuf"
	"mango/pkg/util"

	"github.com/GoldBaby5511/go-simplejson"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
)

const (
	//事件
	ConnectSuccess  string = "ConnectSuccess"
	Disconnect      string = "Disconnect"
	CenterRegResult string = "CenterRegResult"
	CommonServerReg string = "CommonServerReg"

	//回调
	CbBeforeServiceStart    string = "CbBeforeServiceStart"
	CbAfterServiceStart     string = "CbAfterServiceStart"
	CbConfigChangeNotify    string = "CbConfigChangeNotify"
	CbAppControlNotify      string = "CbAppControlNotify"
	CbRabbitMQMessageNotify string = "CbRabbitMQMessageNotify"

	AgentIndex = 0
)

var (
	cbFunctions                   = make(map[string]interface{})
	tcpLog           *n.TCPClient = nil
	mxServers        sync.Mutex
	mxClients        sync.Mutex
	wg               sync.WaitGroup
	servers                           = make(map[uint64]*agentServer)
	clients                           = make(map[uint64]*agentClient)
	agentChanRPC     *chanrpc.Server  = nil
	Skeleton         *module.Skeleton = nil
	processor                         = protobuf.NewProcessor()
	ServiceState                      = conf.AppStateNone
	StateDescription sync.Map
	MaxConnNum       int
	PendingWriteNum  int
	MinMsgLen        uint32
	MaxMsgLen        uint32 = 16 * 1024
	HttpServer              = n.NewHttpServer()
	RpcServer               = n.NewRpcServer()

	// websocket
	WSAddr      string
	HTTPTimeout time.Duration
	CertFile    string
	KeyFile     string

	closeSig chan bool
)

func init() {
	tcpLog = new(n.TCPClient)
	Skeleton = module.NewSkeleton(conf.GoLen, conf.TimerDispatcherLen, conf.AsynCallLen, conf.ChanRPCLen)
	agentChanRPC = Skeleton.ChanRPCServer
	apollo.MsgRouter = Skeleton.ChanRPCServer
	amqp.MsgRouter = Skeleton.ChanRPCServer
	closeSig = make(chan bool, 0)
	MsgRegister(&config.ConfigRsp{}, n.AppConfig, uint16(config.CMDConfig_IDConfigRsp), apollo.HandleConfigRsp)
	MsgRegister(&config.ItemRspState{}, n.AppConfig, uint16(config.CMDConfig_IDItemRspState), apollo.HandleItemRspState)
	MsgRegister(&center.AppControlReq{}, n.AppCenter, uint16(center.CMDCenter_IDAppControlReq), appControlReq)
	//s := &serverInfo{}
	//RpcServiceRegister(s.ServiceRegistrar)
	Skeleton.RegisterChanRPC(apollo.ConfigChangeNotifyId, configChangeNotify)
	Skeleton.RegisterChanRPC(amqp.RabbitMqMessageNotifyId, rabbitMQMessageNotify)
}

func Start(appName string) {
	ServiceState = conf.AppStateStarting
	conf.LoadBaseConfig(appName)
	// logger
	err := log.New(conf.ApplogDir)
	if err != nil {
		panic(err)
	}

	//默认订阅
	apollo.RegisterConfig("", conf.AppInfo.Type, conf.AppInfo.Id)

	//callback
	eventCallBack(CbBeforeServiceStart)

	wg.Add(2)
	go func() {
		Skeleton.Run()
		wg.Done()
	}()

	go func() {
		Run()
		wg.Done()
	}()

	//callback
	eventCallBack(CbAfterServiceStart)

	ServiceState = conf.AppStateRunning
	AddStateDescription("启动时间", fmt.Sprintf("%s", time.Now().Format("2006-01-02 15:04:05")))

	// close
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	sig := <-c
	log.Info("主流程", "服务器关闭 (signal: %v)", sig)
	Stop()
}

func Stop() {
	defer util.TryE(conf.ApplogDir)
	for k, v := range servers {
		v.Close()
		delete(servers, k)
	}
	closeSig <- true
	time.Sleep(time.Second / 2)
	amqp.Close()
	Skeleton.Close()
	console.Destroy()
	wg.Wait()
}

func MsgRegister(m proto.Message, appType uint32, cmdId uint16, f interface{}) {
	chanRPC := Skeleton.ChanRPCServer
	processor.Register(m, appType, cmdId, chanRPC)
	chanRPC.Register(reflect.TypeOf(m), f)
}

func EventRegister(id interface{}, f interface{}) {
	Skeleton.ChanRPCServer.Register(id, f)
}

func CallBackRegister(id string, cb interface{}) {
	if _, ok := cbFunctions[id]; ok {
		log.Warning("", "已经注册,id=%v", id)
		return
	}
	switch cb.(type) {
	case func([]interface{}):
	default:
		log.Warning("", "回调函数类型错误,id=%v", id)
		return
	}
	cbFunctions[id] = cb
}

func HttpRouteRegister(route string, handler func(http.ResponseWriter, *http.Request)) {
	if HttpServer == nil {
		return
	}
	HttpServer.AddRoute(route, handler)
}

func RpcServiceRegister(service func(gs *grpc.Server)) {
	if RpcServer == nil {
		return
	}
	RpcServer.AddService(service)
}

func Run() {
	log.Debug("", "Run,ListenOnAddr=%v", conf.AppInfo.ListenOnAddr)

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
	if conf.AppInfo.ListenOnAddr != "" {
		tcpServer = new(n.TCPServer)
		tcpServer.Addr = fmt.Sprintf("0.0.0.0:%d", util.GetPortFromIPAddress(conf.AppInfo.ListenOnAddr))
		tcpServer.MaxConnNum = MaxConnNum
		tcpServer.PendingWriteNum = PendingWriteNum
		tcpServer.MinMsgLen = MinMsgLen
		tcpServer.MaxMsgLen = MaxMsgLen
		tcpServer.GetConfig = apollo.GetConfigAsInt64
		tcpServer.NewAgent = func(conn *n.TCPConn) n.AgentClient {
			a := &agentClient{conn: conn, info: n.BaseAgentInfo{AgentType: n.NormalUser}}
			if agentChanRPC != nil {
				agentChanRPC.Go(ConnectSuccess, a)
			}
			return a
		}
	}

	if conf.AppInfo.CenterAddr != "" && conf.AppInfo.Type != n.AppCenter && conf.AppInfo.Type != n.AppLogger {
		newServerItem(n.BaseAgentInfo{AgentType: n.CommonServer, AppName: "center", AppType: n.AppCenter, ListenOnAddr: conf.AppInfo.CenterAddr}, true, PendingWriteNum)
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

func configChangeNotify(args []interface{}) {
	key := args[apollo.KeyIndex].(apollo.ConfKey)
	value := args[apollo.ValueIndex].(apollo.ConfValue)

	switch key.Key {
	case "数据库配置":
		dbConfig := value.Value
		if database.DBC == nil && dbConfig != "" {
			database.InitDBHelper(dbConfig)
		}
	case "日志服务器地址":
		logAddr := value.Value
		if logAddr != "" && tcpLog != nil && !tcpLog.IsRunning() {
			ConnectLogServer(logAddr)
		}
	case "扩展订阅":
		if c, err := simplejson.NewJson([]byte(value.Value)); err == nil {
			for _, item := range c.MustArray() {
				switch item.(type) {
				case map[string]interface{}:
					ex := item.(map[string]interface{})
					appType, _ := strconv.Atoi(ex["appType"].(json.Number).String())
					appId, _ := strconv.Atoi(ex["appId"].(json.Number).String())
					key := ex["key"].(string)
					apollo.RegisterConfig(key, uint32(appType), uint32(appId))
				}
			}
		}
	case "http监听地址":
		port := util.GetPortFromIPAddress(value.Value)
		if port != 0 && !HttpServer.IsStart() {
			HttpServer.Port = port
			HttpServer.Start()
		}
	case "rpc监听地址":
		port := util.GetPortFromIPAddress(value.Value)
		if port != 0 && !RpcServer.IsStart() {
			RpcServer.Port = port
			RpcServer.Start()
		}
	case "rabbitmq消费者":
		amqp.NewConsumer(value.Value)
	default:
		break
	}

	//callback
	eventCallBack(CbConfigChangeNotify, args...)
}

func rabbitMQMessageNotify(args []interface{}) {
	m := args[0].(amqp.RabbitMQMessage)
	log.Debug("", "收到了,len=%v,body=%v,time=%v", len(args), m.Body, time.Unix(m.Time, 0).Format("2006-01-02 15-04-05"))

	//callback
	eventCallBack(CbRabbitMQMessageNotify, args...)
}

func appControlReq(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*center.AppControlReq)
	srcAgent := b.AgentInfo

	log.Debug("", "控制消息,sType=%v,sId=%v,AppType=%v,AppId=%v,CtlId=%v",
		srcAgent.AppType, srcAgent.AppId, m.GetCtlId(), m.GetAppType(), m.GetAppId())

	//callback
	eventCallBack(CbAppControlNotify, args...)

	//TODO 停服暂时暴力统一控制一下
	if conf.AppInfo.Type == m.GetAppType() && conf.AppInfo.Id == m.GetAppId() && m.GetCtlId() == int32(center.CtlId_StopService) {
		log.Info("主流程", "收到了停服命令,服务器关闭")
		time.Sleep(time.Second / 2)
		os.Exit(1)
	}
}

func ConnectLogServer(logAddr string) {
	log.Info("gate", "连接日志服务器,Addr=%v", logAddr)
	if conf.AppInfo.Type != n.AppLogger && logAddr != "" && tcpLog != nil && !tcpLog.IsRunning() {
		if conf.RunInLocalDocker() {
			addr := strings.Split(logAddr, "|")
			logAddr = ""
			for i, v := range addr {
				curConnAddr := v
				logAddr = logAddr + "logger:" + strconv.Itoa(util.GetPortFromIPAddress(curConnAddr))
				if i != len(addr)-1 {
					logAddr = logAddr + "|"
				}
			}
		}
		tcpLog.Addr = logAddr
		tcpLog.AutoReconnect = true
		tcpLog.NewAgent = func(conn *n.TCPConn) n.AgentServer {
			a := &agentServer{tcpClient: tcpLog, conn: conn, info: n.BaseAgentInfo{AgentType: n.CommonServer, AppName: "logger", AppType: n.AppLogger, AppId: 0, ListenOnAddr: logAddr}}
			log.Info("gate", "日志服务器连接成功")

			log.SetCallback(func(i log.LogInfo) {
				var logReq logger.LogReq
				logReq.FileName = proto.String(i.File)
				logReq.LineNo = proto.Uint32(uint32(i.Line))
				logReq.SrcApptype = proto.Uint32(conf.AppInfo.Type)
				logReq.SrcAppid = proto.Uint32(conf.AppInfo.Id)
				logReq.Content = []byte(i.LogStr)
				logReq.ClassName = []byte(i.Classname)
				logReq.LogLevel = proto.Uint32(uint32(i.Level))
				logReq.TimeNs = proto.Int64(i.TimeNs)
				logReq.SrcAppname = proto.String(conf.AppInfo.Name)
				cmd := n.TCPCommand{AppType: uint16(n.AppLogger), CmdId: uint16(logger.CMDLogger_IDLogReq)}
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
	registerReq.AppName = proto.String(conf.AppInfo.Name)
	registerReq.AppType = proto.Uint32(conf.AppInfo.Type)
	registerReq.AppId = proto.Uint32(conf.AppInfo.Id)
	myAddress := conf.AppInfo.ListenOnAddr
	if conf.RunInLocalDocker() {
		myAddress = conf.AppInfo.Name + ":" + strconv.Itoa(util.GetPortFromIPAddress(conf.AppInfo.ListenOnAddr))
	}
	registerReq.MyAddress = proto.String(myAddress)
	a.SendData(n.AppCenter, uint32(center.CMDCenter_IDAppRegReq), &registerReq)
}

func eventCallBack(event string, args ...interface{}) {
	if _, ok := cbFunctions[event]; ok {
		cbFunctions[event].(func([]interface{}))(args)
	}
}

func SendData(dataSrc n.BaseAgentInfo, bm n.BaseMessage) error {
	if dataSrc.AgentType == n.CommonServer {
		return sendData(bm, dataSrc.AppType, dataSrc.AppId)
	}
	return SendMessage2Client(bm, util.MakeUint64FromUint32(dataSrc.AppType, dataSrc.AppId), 0)
}

func SendData2App(destAppType, destAppid, appType, cmdId uint32, m proto.Message) error {
	cmd := n.TCPCommand{AppType: uint16(appType), CmdId: uint16(cmdId)}
	bm := n.BaseMessage{MyMessage: m, Cmd: cmd}
	return sendData(bm, destAppType, destAppid)
}

func SendMessage2Client(bm n.BaseMessage, gateConnID, sessionID uint64) error {
	var dataReq gateway.TransferDataReq
	dataReq.DestApptype = proto.Uint32(n.AppGate)
	dataReq.DestAppid = proto.Uint32(util.GetLUint32FromUint64(gateConnID))
	dataReq.DataApptype = proto.Uint32(uint32(bm.Cmd.AppType))
	dataReq.DataCmdid = proto.Uint32(uint32(bm.Cmd.CmdId))
	dataReq.Data, _ = proto.Marshal(bm.MyMessage.(proto.Message))
	dataReq.Gateconnid = proto.Uint64(gateConnID)
	dataReq.AttSessionid = proto.Uint64(sessionID)
	cmd := n.TCPCommand{AppType: uint16(n.AppGate), CmdId: uint16(gateway.CMDGateway_IDTransferDataReq)}
	transBM := n.BaseMessage{MyMessage: &dataReq, Cmd: cmd, TraceId: bm.TraceId}
	return sendData(transBM, n.AppGate, util.GetLUint32FromUint64(gateConnID))
}

func sendData(bm n.BaseMessage, destAppType, destAppid uint32) error {
	destAgents := GetDestApp(destAppType, destAppid)
	if len(destAgents) == 0 {
		//丢失连接的消息暂时只存储到mongo
		mHelper := database.GetMongoHelper()
		if mHelper != nil {
			c := mHelper.GetDB().C("MissingMessage")
			if c != nil {
				msg := new(n.MissingMessage)
				msg.DestAppType = destAppType
				msg.DestAppId = destAppid
				msg.AgentInfo = bm.AgentInfo
				msg.Cmd = bm.Cmd
				msg.TraceId = bm.TraceId
				msg.Data, _ = proto.Marshal(bm.MyMessage.(proto.Message))
				msg.Time = time.Now().Unix()
				err := c.Insert(msg)
				if err != nil {
					log.Warning("", "存储丢失消息失败了,err=%v", err)
				}

				//TODO 暂时保留
				//var result []*MissingMessage
				//q := bson.M{"destAppType": destAppType}
				//f := c.Find(nil)
				//var rl []MissingMessage
				//f.All(&rl)
				//for _, v := range rl {
				//	log.Debug("", "d=%v,Id=%v", v.DestAppType, v.DestAppId)
				//
				//	for _, a := range destAgents {
				//		otherData := make([]byte, 0, n.TraceIdLen+1)
				//		if v.TraceId != "" {
				//			otherData = append(otherData, n.FlagOtherTraceId)
				//			otherData = append(otherData, []byte(v.TraceId)...)
				//		}
				//		err = a.conn.WriteMsg(v.Cmd.AppType, v.Cmd.CmdId, v.Data, otherData)
				//		if err != nil {
				//			log.Error("agentServer", "写信息失败 %v error: %v", reflect.TypeOf(m), err)
				//		}
				//	}
				//
				//}
				//
				//c.RemoveAll(nil)
				////c.RemoveAll(q)
				//return nil
			}
		}

		log.Warning("转发", "消息发送失败,appCount=%v,destAppType=%v,destAppid=%v,bm.Cmd=%v",
			destTypeAppCount(destAppType), destAppType, destAppid, bm.Cmd)

		return fmt.Errorf("目标没找到,destAppType=%d,destAppid=%d", destAppType, destAppid)
	}

	for _, a := range destAgents {
		a.SendMessage(bm)
	}
	return nil
}

func GetDestApp(destAppType, destAppid uint32) []*agentServer {
	mxServers.Lock()
	defer mxServers.Unlock()
	var destAgent []*agentServer
	if destTypeAppCount(destAppType) != 0 {
		switch destAppid {
		case n.Send2All:
			for _, v := range servers {
				if v.info.AppType == destAppType {
					destAgent = append(destAgent, v)
				}
			}
		case n.Send2AnyOne:
			for _, v := range servers {
				if v.info.AppType == destAppType {
					destAgent = append(destAgent, v)
					break
				}
			}
		default:
			if _, ok := servers[util.MakeUint64FromUint32(destAppType, destAppid)]; ok {
				destAgent = append(destAgent, servers[util.MakeUint64FromUint32(destAppType, destAppid)])
			}
		}
	}

	return destAgent
}

func destTypeAppCount(destAppType uint32) int {
	destCount := 0
	for _, v := range servers {
		if v.info.AppType == destAppType {
			destCount++
		}
	}
	return destCount
}

func AddStateDescription(k, v string) {
	StateDescription.Store(k, v)
}

func GetStateDescription() string {
	desc := ""
	StateDescription.Range(func(k, v interface{}) bool {
		desc += fmt.Sprintf("%v:%v,", k, v)
		return true
	})
	return desc
}

//TODO 暂时保留
//type serverInfo struct {
//	center.UnimplementedAppControlServer
//}
//
//func (s *serverInfo) ServiceRegistrar(gs *grpc.Server) {
//	center.RegisterAppControlServer(gs, &serverInfo{})
//}
//
//func (s *serverInfo) ControlReq(ctx context.Context, in *center.AppControlReq) (*center.AppControlRsp, error) {
//	log.Debug("", "Received: %v", in.GetAppType())
//	return &center.AppControlRsp{CtlId: proto.Int32(in.GetCtlId())}, nil
//}
