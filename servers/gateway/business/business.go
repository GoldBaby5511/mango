package business

import (
	"errors"
	"google.golang.org/protobuf/proto"
	"reflect"
	"time"
	"xlddz/core/conf/apollo"
	g "xlddz/core/gate"
	"xlddz/core/log"
	"xlddz/core/module"
	n "xlddz/core/network"
	"xlddz/core/network/protobuf"
	"xlddz/protocol/center"
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
	processor.Register(&center.DataTransferRsp{}, n.CMDCenter, uint16(center.CMDID_Center_IDDataMessageReq), chanRPC)

	chanRPC.Register(g.ConnectSuccess, connectSuccess)
	chanRPC.Register(g.Disconnect, disconnect)
	chanRPC.Register(g.RouterConnected, routerConnected)
	chanRPC.Register(reflect.TypeOf(&gcmd.PulseReq{}), handlePulseReq)
	chanRPC.Register(reflect.TypeOf(&gcmd.TransferDataReq{}), handleTransferDataReq)
	chanRPC.Register(reflect.TypeOf(&gcmd.AuthInfo{}), handleAuthInfo)
	chanRPC.Register(reflect.TypeOf(&gcmd.HelloReq{}), handleHelloReq)
	chanRPC.Register(reflect.TypeOf(&center.DataTransferRsp{}), handRouterDataMessageReq)
}

type Gate struct {
	*g.Gate
}

func (m *Gate) OnInit() {
	g.AgentChanRPC = skeleton.ChanRPCServer
	g.Processor = processor
	m.Gate = &g.Gate{
		TCPAddr:       conf.Server.TCPAddr,
		TCPClientAddr: conf.Server.TCPClientAddr,
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
		m.GetAttApptype(), m.GetAttAppid(), m.GetDataCmdKind(), m.GetDataCmdSubid(), connData.connId, m.GetAttGateconnid(), a.AgentInfo().AgentType)

	if m.GetAttGateconnid() != 0 && a.AgentInfo().AgentType == n.CommonServer {
		a, err := getUserAgent(m.GetAttGateconnid())
		if err != nil {
			log.Error("消息转发", "根本没找到用户连接,"+
				"AttUserid=%v,AttGateconnid=%v,connId=%v",
				m.GetAttUserid(), m.GetAttGateconnid(),
				m.GetAttGateconnid()&0xFFFFFFFF)
			return
		}
		a.SendData(n.CMDGate, uint32(gcmd.CMDID_Gate_IDTransferDataReq), m)
	} else {
		//var dataReq center.DataTransferReq
		//dataReq.SrcAppid = proto.Uint32(conf.Server.AppID)
		//dataReq.SrcApptype = proto.Uint32(lconf.AppType)
		//dataReq.DestAppid = proto.Uint32(m.GetAttAppid())
		//dataReq.DestApptype = proto.Uint32(m.GetAttApptype())
		//dataReq.DataCmdkind = proto.Uint32(m.GetDataCmdKind())
		//dataReq.DataCmdsubid = proto.Uint32(m.GetDataCmdSubid())
		//dataReq.DataBuff = m.GetData()
		//dataReq.DataDirection = proto.Uint32(uint32(center.EnuDataDirection_DT_App2App))
		//dataReq.AttGateid = proto.Uint32(conf.Server.AppID)
		//dataReq.AttGateconnid = proto.Uint64(makeGateConnID(connData.connId))
		//dataReq.DataDirection = proto.Uint32(uint32(center.EnuDataDirection_DT_App2App))
		m.AttGateid = proto.Uint32(conf.Server.AppID)
		m.AttGateconnid = proto.Uint64(makeGateConnID(connData.connId))
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

func handRouterDataMessageReq(args []interface{}) {
	b := args[n.DATA_INDEX].(n.BaseMessage)
	m := (b.MyMessage).(*center.DataTransferReq)

	a, err := getUserAgent(m.GetAttGateconnid())
	if err != nil {
		log.Error("消息转发", "根本没找到用户连接,"+
			"SrcApptype=%v,SrcAppid=%v,"+
			"DataCmdkind=%v,DataCmdsubid=%v,"+
			"AttUserid=%v,AttGateconnid=%v,connId=%v",
			m.GetSrcApptype(), m.GetSrcAppid(),
			m.GetDataCmdkind(), m.GetDataCmdsubid(),
			m.GetAttUserid(), m.GetAttGateconnid(),
			m.GetAttGateconnid()&0xFFFFFFFF)
		return
	}

	log.Debug("module", "消息回传,connId=%v,connId=%v", m.GetAttGateconnid(), m.GetAttGateconnid()&0xFFFFFFFF)

	var req gcmd.TransferDataReq
	req.DataCmdKind = proto.Uint32(m.GetDataCmdkind())
	req.DataCmdSubid = proto.Uint32(m.GetDataCmdsubid())
	req.AttApptype = proto.Uint32(m.GetSrcApptype())
	req.AttAppid = proto.Uint32(m.GetSrcAppid())
	req.Data = m.GetDataBuff()
	req.AttSessionid = proto.Uint64(m.GetAttSessionid())
	a.SendData(n.CMDGate, uint32(gcmd.CMDID_Gate_IDTransferDataReq), &req)
}

func checkConnectionAlive() {

	var da []connectionData
	for _, v := range userConnData {
		if time.Now().Unix()-v.lastPulseTk > apollo.GetConfigAsInt64("心跳间隔", 180) {
			da = append(da, *v)
		}
	}

	for _, c := range da {
		log.Debug("心跳", "心跳超时断开,userId=%v,connId=%v", c.userId, c.connId)
		c.a.Close()
	}

	skeleton.AfterFunc(30*time.Second, checkConnectionAlive)
}
