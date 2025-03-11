package business

import (
	"errors"
	"github.com/golang/protobuf/proto"
	"mango/api/gateway"
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
	g.MsgRegister(&gateway.ShutDownSocket{}, n.AppGate, uint16(gateway.CMDGateway_IDShutDownSocket), handleShutDownSocket)
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

	log.Debug("module", "来了老弟,connId=%v,当前连接数=%d,gateConnId=%v,%v",
		connId, len(userConnData), util.MakeUint64FromUint32(connId, conf.AppInfo.Id), util.PrintStructFields(*userConnData[connId].a.AgentInfo()))
}

func disconnect(args []interface{}) {
	if a, err := getUserConnData(args[g.AgentIndex].(n.AgentClient)); err == nil {
		log.Debug("module", "走了老弟,userId=%v,connId=%v,当前连接数=%d,info=%v", a.userId, a.connId, len(userConnData), util.PrintStructFields(a.a.AgentInfo()))

		if a.a.AgentInfo().AgentType == n.NormalUser && a.userId != 0 {
			//通知lobby
			g.SendData2App(n.AppLobby, n.Send2All, n.AppGate, uint32(gateway.CMDGateway_IDNetworkDisconnected),
				&gateway.NetworkDisconnected{
					UserId: a.userId,
				})
		}

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

	log.Debug("module", "n.AppGate,消息转发,type=%v,appid=%v,mainCmd=%v,subCmd=%v,connId=%v,gateConnId=%v,AgentType=%v",
		m.GetDestApptype(), m.GetDestAppid(), m.GetMainCmdId(), m.GetSubCmdId(), connData.connId, m.GetGateConnId(), a.AgentInfo().AgentType)

	//App->Client 消息
	if m.GetGateConnId() != 0 && a.AgentInfo().AgentType == n.CommonServer {
		a, err := getUserAgent(m.GetGateConnId())
		if err != nil {
			log.Warning("消息转发", "未找到可能已下线,"+
				"Gateconnid=%v,connId=%v",
				m.GetGateConnId(),
				util.GetHUint32FromUint64(m.GetGateConnId()))
			return
		}
		a.SendData(n.AppGate, uint32(gateway.CMDGateway_IDTransferDataReq), m)
	} else {
		//Client->App 消息
		m.GateConnId = *proto.Uint64(util.MakeUint64FromUint32(connData.connId, conf.AppInfo.Id))
		m.UserId = *proto.Uint64(connData.userId)
		g.SendData2App(m.GetDestApptype(), m.GetDestAppid(), n.AppGate, uint32(gateway.CMDGateway_IDTransferDataReq), m)
	}
}

func handleAuthInfo(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*gateway.AuthInfo)
	srcApp := args[n.OtherIndex].(n.BaseAgentInfo)

	log.Debug("", "认证消息,appID=%d,userID=%d", srcApp.AppId, m.GetUserId())
	connData, ok := userConnData[util.GetHUint32FromUint64(m.GetGateConnId())]
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
	rsp := gateway.HelloRsp{
		RspFlag: uint32(gateway.HelloRsp_LoginToken),
		Guid:    m.GetGuid(),
	}

	a.SendData(n.AppGate, uint32(gateway.CMDGateway_IDHelloRsp), &rsp)
}

// 关闭网络
func handleShutDownSocket(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*gateway.ShutDownSocket)
	//a := args[n.AgentIndex].(n.AgentClient)

	connId := util.GetHUint32FromUint64(m.GetGateConnId())
	a, err := getUserAgent(m.GetGateConnId())
	if err != nil {
		log.Warning("关闭网络", "关闭网络,未找到可能已下线,gateConnId=%v,connId=%v",
			m.GetGateConnId(),
			connId)
		return
	}

	//清除绑定userid
	if connData, ok := userConnData[connId]; ok {
		if connData.a.AgentInfo().AgentType == n.NormalUser && connData.userId != 0 {
			log.Debug("", "收到关闭网络消息,清除绑定userid,userId=%v,connId=%v,gateConnId=%v,%v", m.GetUserId(), connId, m.GetGateConnId(), util.PrintStructFields(a.AgentInfo()))
			//清除绑定userid
			connData.userId = 0
		}
	}

	log.Debug("", "收到关闭网络消息,userId=%v,connId=%v,gateConnId=%v,%v", m.GetUserId(), connId, m.GetGateConnId(), util.PrintStructFields(a.AgentInfo()))

	a.Close()
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

	//关闭连接
	for _, c := range da {
		log.Debug("心跳", "心跳超时断开,userId=%v,connId=%v,info=%v", c.userId, c.connId, c.a.AgentInfo())
		if c.userId != 0 {
			//通知lobby
			g.SendData2App(n.AppLobby, n.Send2All, n.AppGate, uint32(gateway.CMDGateway_IDNetworkDisconnected),
				&gateway.NetworkDisconnected{
					UserId: c.userId,
				})
		}
		c.a.Close()
	}
}
