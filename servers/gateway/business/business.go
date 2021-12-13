package business

import (
	"errors"
	"google.golang.org/protobuf/proto"
	"reflect"
	"time"
	lconf "xlddz/core/conf"
	"xlddz/core/conf/apollo"
	g "xlddz/core/gate"
	"xlddz/core/log"
	"xlddz/core/module"
	n "xlddz/core/network"
	"xlddz/core/network/protobuf"
	"xlddz/protocol/client"
	gcmd "xlddz/protocol/gate"
	"xlddz/servers/gateway/conf"
)

var (
	skeleton                                = module.NewSkeleton(conf.GoLen, conf.TimerDispatcherLen, conf.AsynCallLen, conf.ChanRPCLen)
	userConnData map[uint64]*connectionData = make(map[uint64]*connectionData)
	processor                               = protobuf.NewProcessor()
)

//连接数据
type connectionData struct {
	a           n.AgentClient
	userId      uint64
	connId      uint64
	hasHello    bool
	lastPulseTk int64
}

func init() {
	//消息注册
	chanRPC := skeleton.ChanRPCServer
	processor.Register(&gcmd.PulseReq{}, n.CMDGate, uint16(gcmd.CMDID_Gate_IDPulseReq), chanRPC)
	processor.Register(&gcmd.TransferDataReq{}, n.CMDGate, uint16(gcmd.CMDID_Gate_IDTransferDataReq), chanRPC)
	processor.Register(&gcmd.AuthInfo{}, n.CMDGate, uint16(gcmd.CMDID_Gate_IDAuthInfo), chanRPC)
	processor.Register(&gcmd.HelloReq{}, n.CMDGate, uint16(gcmd.CMDID_Gate_IDHelloReq), chanRPC)

	chanRPC.Register(g.ConnectSuccess, connectSuccess)
	chanRPC.Register(g.Disconnect, disconnect)
	chanRPC.Register(g.CenterConnected, routerConnected)
	chanRPC.Register(reflect.TypeOf(&gcmd.PulseReq{}), handlePulseReq)
	chanRPC.Register(reflect.TypeOf(&gcmd.TransferDataReq{}), handleTransferDataReq)
	chanRPC.Register(reflect.TypeOf(&gcmd.AuthInfo{}), handleAuthInfo)
	chanRPC.Register(reflect.TypeOf(&gcmd.HelloReq{}), handleHelloReq)
}

type Gate struct {
	*g.Gate
}

func (m *Gate) OnInit() {
	g.AgentChanRPC = skeleton.ChanRPCServer
	g.Processor = processor
	m.Gate = &g.Gate{
		TCPAddr:       conf.Server.TCPAddr,
		TCPClientAddr: lconf.CenterAddr,
		MaxConnNum:    20000,
	}

	skeleton.AfterFunc(30*time.Second, checkConnectionAlive)
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
	if len(args) != 2 {
		return
	}
	connId := args[g.IdIndex].(uint64)
	userConnData[connId] = &connectionData{
		a:           args[g.AgentIndex].(n.AgentClient),
		connId:      connId,
		hasHello:    false,
		lastPulseTk: time.Now().Unix(),
	}

	log.Debug("module", "来了老弟,connId=%v,当前连接数=%d", connId, len(userConnData))
}

func disconnect(args []interface{}) {
	if len(args) != 2 {
		return
	}
	connId := args[g.IdIndex].(uint64)
	if v, ok := userConnData[connId]; ok {
		log.Debug("agent1", "走了老弟,userId=%v,connId=%v,当前连接数=%d", v.userId, v.connId, len(userConnData))

		//发送退出
		var logout client.LogoutReq
		logout.UserId = proto.Uint64(v.userId)
		g.SendData2App(n.AppLogin, n.Send2AnyOne, n.CMDClient, uint32(client.CMDID_Client_IDLogoutReq), &logout)

		delete(userConnData, connId)
	} else {
		log.Warning("agent1", "一个没有注册过的连接?,当前连接数=%d", len(userConnData))
	}
}

func routerConnected(args []interface{}) {
}

func handlePulseReq(args []interface{}) {
	//m := args[n.DATA_INDEX].(*gcmd.PulseReq)
	a := args[n.AGENT_INDEX].(n.AgentClient)

	connData, err := getUserConnData(a)
	if err != nil {
		log.Warning("gate消息", "异常,没有连接的hello消息")
		return
	}

	log.Debug("hello", "收到心跳消息,userId=%v,connId=%d", connData.userId, connData.connId)

	connData.lastPulseTk = time.Now().Unix()
}

func handleTransferDataReq(args []interface{}) {
	b := args[n.DATA_INDEX].(n.BaseMessage)
	m := (b.MyMessage).(*gcmd.TransferDataReq)
	a := args[n.AGENT_INDEX].(n.AgentClient)

	connData, err := getUserConnData(a)
	if err != nil {
		log.Warning("module", "异常,消息转发,木有找到的连接发的消息,len=%v", len(args))
		return
	}

	log.Debug("module", "n.CMDGate,消息转发,type=%v,appid=%v,kind=%v,sub=%v,connId=%v,%v,%v",
		m.GetAttApptype(), m.GetAttAppid(), m.GetDataCmdKind(), m.GetDataCmdSubid(), connData.connId, m.GetGateconnid(), a.AgentInfo().AgentType)

	if m.GetGateconnid() != 0 && a.AgentInfo().AgentType == n.CommonServer {
		a, err := getUserAgent(m.GetGateconnid())
		if err != nil {
			log.Error("消息转发", "根本没找到用户连接,"+
				"AttGateconnid=%v,connId=%v",
				m.GetGateconnid(),
				m.GetGateconnid()&0xFFFFFFFF)
			return
		}
		a.SendData(n.CMDGate, uint32(gcmd.CMDID_Gate_IDTransferDataReq), m)
	} else {
		m.Gateid = proto.Uint32(conf.Server.AppID)
		m.Gateconnid = proto.Uint64(makeGateConnID(connData.connId))
		g.SendData2App(m.GetAttApptype(), m.GetAttAppid(), n.CMDGate, uint32(gcmd.CMDID_Gate_IDTransferDataReq), m)
	}

}

func handleAuthInfo(args []interface{}) {

}

func handleHelloReq(args []interface{}) {
	b := args[n.DATA_INDEX].(n.BaseMessage)
	m := (b.MyMessage).(*gcmd.HelloReq)
	a := args[n.AGENT_INDEX].(n.AgentClient)

	connData, err := getUserConnData(a)
	if err != nil {
		log.Warning("gate消息", "异常,没有连接的hello消息")
		return
	}

	connData.hasHello = true

	log.Debug("hello", "收到hello消息,connId=%d", connData.connId)

	//加密方式暂不考虑
	var rsp gcmd.HelloRsp
	flag := gcmd.HelloRsp_LoginToken
	rsp.RspFlag = proto.Uint32(uint32(flag))
	if m.GetGuid() != "" {
		rsp.Guid = proto.String(m.GetGuid())
	}
	a.SendData(n.CMDGate, uint32(gcmd.CMDID_Gate_IDHelloRsp), &rsp)
}

func makeGateConnID(connId uint64) uint64 {
	return uint64(conf.Server.AppID)<<32 + connId
}

func getUserConnData(a n.AgentClient) (*connectionData, error) {
	for _, v := range userConnData {
		if v.a == a {
			return v, nil
		}
	}

	return nil, errors.New("没有找到啊")
}

func getUserAgent(gateConnId uint64) (n.AgentClient, error) {
	connData, ok := userConnData[gateConnId&0xFFFFFFFF]
	if !ok {
		return nil, errors.New("真没找到啊")
	}

	return connData.a, nil
}

func checkConnectionAlive() {

	var da []connectionData
	for _, v := range userConnData {
		if time.Now().Unix()-v.lastPulseTk > apollo.GetConfigAsInt64("心跳间隔", 180) && v.a.AgentInfo().AgentType != n.CommonServer {
			da = append(da, *v)
		}
	}

	for _, c := range da {
		log.Debug("心跳", "心跳超时断开,userId=%v,connId=%v,info=%v", c.userId, c.connId, c.a.AgentInfo())
		c.a.Close()
	}

	skeleton.AfterFunc(30*time.Second, checkConnectionAlive)
}
