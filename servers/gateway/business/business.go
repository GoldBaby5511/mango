package business

import (
	"errors"
	"google.golang.org/protobuf/proto"
	"reflect"
	"sync"
	g "xlddz/core/gate"
	"xlddz/core/log"
	"xlddz/core/module"
	n "xlddz/core/network"
	"xlddz/core/network/protobuf"
	"xlddz/protocol/client"
	gcmd "xlddz/protocol/gate"
	"xlddz/protocol/router"
	"xlddz/servers/gateway/conf"
)

var (
	skeleton             = module.NewSkeleton(conf.GoLen, conf.TimerDispatcherLen, conf.AsynCallLen, conf.ChanRPCLen)
	routeAgent   n.Agent = nil
	mutexConnId  sync.Mutex
	userConnData map[uint64]*connectionData = make(map[uint64]*connectionData)
	processor                               = protobuf.NewProcessor()
)

//连接数据
type connectionData struct {
	a           n.Agent
	userId      uint64
	connId      uint64
	hasSayhello bool
}

func init() {
	//消息注册
	chanRPC := skeleton.ChanRPCServer
	processor.Register(&gcmd.PulseReq{}, n.CMDGate, uint16(gcmd.CMDID_Gate_IDPulseReq), chanRPC)
	processor.Register(&gcmd.TransferDataReq{}, n.CMDGate, uint16(gcmd.CMDID_Gate_IDTransferDataReq), chanRPC)
	processor.Register(&gcmd.AuthInfo{}, n.CMDGate, uint16(gcmd.CMDID_Gate_IDAuthInfo), chanRPC)
	processor.Register(&gcmd.HelloReq{}, n.CMDGate, uint16(gcmd.CMDID_Gate_IDHelloReq), chanRPC)

	processor.RegHandle(&router.DataTransferRsp{}, n.CMDRouter, uint16(router.CMDID_Router_IDDataMessageReq), routerDataMessageReq)

	chanRPC.Register(g.ConnectSuccess, connectSuccess)
	chanRPC.Register(g.Disconnect, disconnect)
	chanRPC.Register(g.RouterConnected, routerConnected)
	chanRPC.Register(reflect.TypeOf(&gcmd.PulseReq{}), handlePulseReq)
	chanRPC.Register(reflect.TypeOf(&gcmd.TransferDataReq{}), handleTransferDataReq)
	chanRPC.Register(reflect.TypeOf(&gcmd.AuthInfo{}), handleAuthInfo)
	chanRPC.Register(reflect.TypeOf(&gcmd.HelloReq{}), handleHelloReq)

}

type Gate struct {
	*g.Gate
}

func (m *Gate) OnInit() {
	m.Gate = &g.Gate{
		AgentChanRPC:  skeleton.ChanRPCServer,
		Processor:     processor,
		TCPAddr:       conf.Server.TCPAddr,
		TCPClientAddr: conf.Server.TCPClientAddr,
	}
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
	mutexConnId.Lock()
	defer mutexConnId.Unlock()
	connId := args[g.IdIndex].(uint64)
	userConnData[connId] = &connectionData{a: args[g.AgentIndex].(n.Agent), connId: connId, hasSayhello: false}

	log.Debug("module", "来了老弟,connId=%v,当前连接数=%d", connId, len(userConnData))
}

func disconnect(args []interface{}) {
	if len(args) != 2 {
		return
	}
	mutexConnId.Lock()
	defer mutexConnId.Unlock()
	connId := args[g.IdIndex].(uint64)
	if v, ok := userConnData[connId]; ok {
		//发送退出
		var logout client.LogoutReq
		logout.UserId = proto.Uint64(v.userId)
		routeAgent.SendData2App(n.AppLogin, n.Send2AnyOne, n.CMDClient, uint32(client.CMDID_Client_IDLogoutReq), &logout)

		log.Debug("module", "走了老弟,userId=%v,connId=%v,当前连接数=%d", v.userId, v.connId, len(userConnData))
		delete(userConnData, connId)
	} else {
		log.Warning("module", "一个没有注册过的连接?,当前连接数=%d", len(userConnData))
	}
}

func routerConnected(args []interface{}) {
	routeAgent = args[g.AgentIndex].(n.Agent)
}

func handlePulseReq(args []interface{}) {

}

func handleTransferDataReq(args []interface{}) {
	m := args[n.DATA_INDEX].(*gcmd.TransferDataReq)
	a := args[n.AGENT_INDEX].(n.Agent)

	connData, err := getUserConnData(a)
	if err != nil {
		log.Error("module", "异常,消息转发,木有找到的连接发的消息,len=%v", len(args))
		return
	}

	log.Debug("agent", "n.CMDGate,消息转发,type=%v,appid=%v,kind=%v,sub=%v,connId=%v",
		m.GetAttApptype(), m.GetAttAppid(), m.GetDataCmdKind(), m.GetDataCmdSubid(), connData.connId)

	var dataReq router.DataTransferReq
	dataReq.SrcAppid = proto.Uint32(conf.Server.AppID)
	dataReq.SrcApptype = proto.Uint32(conf.Server.AppType)
	dataReq.DestAppid = proto.Uint32(m.GetAttAppid())
	dataReq.DestApptype = proto.Uint32(m.GetAttApptype())
	dataReq.DataCmdkind = proto.Uint32(m.GetDataCmdKind())
	dataReq.DataCmdsubid = proto.Uint32(m.GetDataCmdSubid())
	dataReq.DataBuff = m.GetData()
	dataReq.DataDirection = proto.Uint32(uint32(router.EnuDataDirection_DT_App2App))
	dataReq.AttGateid = proto.Uint32(conf.Server.AppID)
	dataReq.AttGateconnid = proto.Uint64(makeGateConnID(connData.connId))
	routeAgent.SendData(n.CMDRouter, uint32(router.CMDID_Router_IDDataMessageReq), &dataReq)
}

func handleAuthInfo(args []interface{}) {

}

func handleHelloReq(args []interface{}) {
	m := args[n.DATA_INDEX].(*gcmd.HelloReq)
	a := args[n.AGENT_INDEX].(n.Agent)

	connData, err := getUserConnData(a)
	if err != nil {
		log.Error("gate消息", "异常,没有连接的hello消息")
		return
	}

	connData.hasSayhello = true

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

func getUserConnData(a n.Agent) (*connectionData, error) {
	mutexConnId.Lock()
	defer mutexConnId.Unlock()
	for _, v := range userConnData {
		if v.a == a {
			return v, nil
		}
	}

	return nil, errors.New("没有找到啊")
}

func getUserAgent(gateConnId uint64) (n.Agent, error) {
	mutexConnId.Lock()
	defer mutexConnId.Unlock()
	connData, ok := userConnData[gateConnId&0xFFFFFFFF]
	if !ok {
		return nil, errors.New("真没找到啊")
	}

	return connData.a, nil
}

func routerDataMessageReq(args []interface{}) {
	b := args[n.DATA_INDEX].(n.BaseMessage)
	m := (b.MyMessage).(*router.DataTransferReq)

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
