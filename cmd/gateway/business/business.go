package business

import (
	"errors"
	"github.com/golang/protobuf/proto"
	"mango/api/gateway"
	"mango/api/lobby"
	"mango/pkg/conf"
	"mango/pkg/conf/apollo"
	g "mango/pkg/gate"
	"mango/pkg/log"
	n "mango/pkg/network"
	"mango/pkg/timer"
	"mango/pkg/util"
	"time"
)

var (
	connectionId uint32 = 0
	userConnData        = make(map[uint32]*connectionData)
)

type connectionData struct {
	a           n.AgentClient
	userId      uint64
	connId      uint32
	hasHello    bool
	lastPulseTk int64
}

func init() {
	g.MsgRegister(&gateway.PulseReq{}, n.AppGate, uint16(gateway.CMDGateway_IDPulseReq), handlePulseReq)
	g.MsgRegister(&gateway.TransferDataReq{}, n.AppGate, uint16(gateway.CMDGateway_IDTransferDataReq), handleTransferDataReq)
	g.MsgRegister(&gateway.AuthInfo{}, n.AppGate, uint16(gateway.CMDGateway_IDAuthInfo), handleAuthInfo)
	g.MsgRegister(&gateway.HelloReq{}, n.AppGate, uint16(gateway.CMDGateway_IDHelloReq), handleHelloReq)
	g.EventRegister(g.ConnectSuccess, connectSuccess)
	g.EventRegister(g.Disconnect, disconnect)

	g.Skeleton.LoopFunc(30*time.Second, checkConnectionAlive, timer.LoopForever)
}

func connectSuccess(args []interface{}) {
	connectionId++
	connId := connectionId
	userConnData[connId] = &connectionData{
		a:           args[g.AgentIndex].(n.AgentClient),
		connId:      connId,
		hasHello:    false,
		lastPulseTk: time.Now().Unix(),
	}
	userConnData[connId].a.AgentInfo().AppType = connId
	userConnData[connId].a.AgentInfo().AppId = conf.AppInfo.Id

	log.Debug("module", "来了老弟,connId=%v,当前连接数=%d,gateConnId=%v,info=%v",
		connId, len(userConnData), util.MakeUint64FromUint32(connId, conf.AppInfo.Id), *userConnData[connId].a.AgentInfo())
}

func disconnect(args []interface{}) {
	if a, err := getUserConnData(args[g.AgentIndex].(n.AgentClient)); err == nil {
		log.Debug("module", "走了老弟,userId=%v,connId=%v,当前连接数=%d,info=%v", a.userId, a.connId, len(userConnData), a.a.AgentInfo())

		var logout lobby.LogoutReq
		logout.UserId = proto.Uint64(a.userId)
		g.SendData2App(n.AppLobby, n.Send2AnyOne, n.AppLobby, uint32(lobby.CMDLobby_IDLogoutReq), &logout)

		delete(userConnData, a.connId)
	} else {
		log.Warning("module", "一个没有注册过的连接?,当前连接数=%d", len(userConnData))
	}
}

func handlePulseReq(args []interface{}) {
	a := args[n.AgentIndex].(n.AgentClient)

	connData, err := getUserConnData(a)
	if err != nil {
		log.Warning("gate消息", "异常,没有连接的hello消息")
		return
	}

	log.Debug("hello", "收到心跳消息,userId=%v,connId=%d", connData.userId, connData.connId)

	connData.lastPulseTk = time.Now().Unix()
}

func handleTransferDataReq(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*gateway.TransferDataReq)
	a := args[n.AgentIndex].(n.AgentClient)

	connData, err := getUserConnData(a)
	if err != nil {
		log.Warning("module", "异常,消息转发,木有找到的连接发的消息,len=%v", len(args))
		return
	}

	log.Debug("module", "n.AppGate,消息转发,type=%v,appid=%v,kind=%v,sub=%v,connId=%v,gateConnId=%v,AgentType=%v",
		m.GetDestApptype(), m.GetDestAppid(), m.GetDataApptype(), m.GetDataCmdid(), connData.connId, m.GetGateconnid(), a.AgentInfo().AgentType)

	if m.GetGateconnid() != 0 && a.AgentInfo().AgentType == n.CommonServer {
		a, err := getUserAgent(m.GetGateconnid())
		if err != nil {
			log.Warning("消息转发", "为找到可能已下线,"+
				"AttGateconnid=%v,connId=%v",
				m.GetGateconnid(),
				util.GetHUint32FromUint64(m.GetGateconnid()))
			return
		}
		a.SendData(n.AppGate, uint32(gateway.CMDGateway_IDTransferDataReq), m)
	} else {
		m.Gateid = proto.Uint32(conf.AppInfo.Id)
		m.Gateconnid = proto.Uint64(util.MakeUint64FromUint32(connData.connId, conf.AppInfo.Id))
		m.UserId = proto.Uint64(connData.userId)
		g.SendData2App(m.GetDestApptype(), m.GetDestAppid(), n.AppGate, uint32(gateway.CMDGateway_IDTransferDataReq), m)
	}
}

func handleAuthInfo(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*gateway.AuthInfo)
	srcApp := args[n.OtherIndex].(n.BaseAgentInfo)

	log.Debug("", "认证消息,appID=%d,userID=%d", srcApp.AppId, m.GetUserId())
	connData, ok := userConnData[util.GetHUint32FromUint64(m.GetGateconnid())]
	if !ok {
		return
	}
	connData.userId = m.GetUserId()
}

func handleHelloReq(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*gateway.HelloReq)
	a := args[n.AgentIndex].(n.AgentClient)

	connData, err := getUserConnData(a)
	if err != nil {
		log.Warning("gate消息", "异常,没有连接的hello消息")
		return
	}

	connData.hasHello = true

	log.Debug("hello", "收到hello消息,connId=%d", connData.connId)

	//加密方式暂不考虑
	var rsp gateway.HelloRsp
	flag := gateway.HelloRsp_LoginToken
	rsp.RspFlag = proto.Uint32(uint32(flag))
	if m.GetGuid() != "" {
		rsp.Guid = proto.String(m.GetGuid())
	}
	a.SendData(n.AppGate, uint32(gateway.CMDGateway_IDHelloRsp), &rsp)
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
	connData, ok := userConnData[util.GetHUint32FromUint64(gateConnId)]
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
}
