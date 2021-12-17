package business

import (
	"errors"
	"github.com/golang/protobuf/proto"
	"time"
	"xlddz/core/conf"
	"xlddz/core/conf/apollo"
	g "xlddz/core/gate"
	"xlddz/core/log"
	n "xlddz/core/network"
	"xlddz/protocol/client"
	gcmd "xlddz/protocol/gate"
)

var (
	userConnData map[uint64]*connectionData = make(map[uint64]*connectionData)
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
	g.MsgRegister(&gcmd.PulseReq{}, n.CMDGate, uint16(gcmd.CMDID_Gate_IDPulseReq), handlePulseReq)
	g.MsgRegister(&gcmd.TransferDataReq{}, n.CMDGate, uint16(gcmd.CMDID_Gate_IDTransferDataReq), handleTransferDataReq)
	g.MsgRegister(&gcmd.AuthInfo{}, n.CMDGate, uint16(gcmd.CMDID_Gate_IDAuthInfo), handleAuthInfo)
	g.MsgRegister(&gcmd.HelloReq{}, n.CMDGate, uint16(gcmd.CMDID_Gate_IDHelloReq), handleHelloReq)
	g.EventRegister(g.ConnectSuccess, connectSuccess)
	g.EventRegister(g.Disconnect, disconnect)
	g.EventRegister(g.CenterConnected, centerConnected)

	g.Skeleton.AfterFunc(30*time.Second, checkConnectionAlive)
}

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

func centerConnected(args []interface{}) {
}

func handlePulseReq(args []interface{}) {
	//m := args[n.DataIndex].(*gcmd.PulseReq)
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
	m := (b.MyMessage).(*gcmd.TransferDataReq)
	a := args[n.AgentIndex].(n.AgentClient)

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
		m.Gateid = proto.Uint32(conf.AppInfo.AppID)
		m.Gateconnid = proto.Uint64(makeGateConnID(connData.connId))
		g.SendData2App(m.GetAttApptype(), m.GetAttAppid(), n.CMDGate, uint32(gcmd.CMDID_Gate_IDTransferDataReq), m)
	}

}

func handleAuthInfo(args []interface{}) {

}

func handleHelloReq(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*gcmd.HelloReq)
	a := args[n.AgentIndex].(n.AgentClient)

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
	return uint64(conf.AppInfo.AppID)<<32 + connId
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

	g.Skeleton.AfterFunc(30*time.Second, checkConnectionAlive)
}
