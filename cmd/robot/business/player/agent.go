package player

import (
	"github.com/golang/protobuf/proto"
	"mango/api/gateway"
	"mango/pkg/conf"
	"mango/pkg/log"
	n "mango/pkg/network"
	"reflect"
)

type agentPlayer struct {
	tcpClient *n.TCPClient
	p         *Player
	conn      n.Conn
	info      n.BaseAgentInfo
}

func (a *agentPlayer) Run() {
	for {
		bm, msgData, err := a.conn.ReadMsg()
		if err != nil {
			log.Warning("agentPlayer", "异常,网关读取消息失败,info=%v,err=%v", a.info, err)
			break
		}

		if a.p.processor == nil {
			log.Warning("", "processor==nil,cmd=%v", bm.Cmd)
			break
		}

		unmarshalCmd := bm.Cmd
		var cmd, msg, dataReq interface{}
		if bm.Cmd.AppType == uint16(n.AppGate) && bm.Cmd.CmdId == uint16(gateway.CMDGateway_IDTransferDataReq) && conf.AppInfo.Type != n.AppGate {
			var m gateway.TransferDataReq
			_ = proto.Unmarshal(msgData, &m)
			unmarshalCmd = n.TCPCommand{AppType: uint16(m.GetDataApptype()), CmdId: uint16(m.GetDataCmdid())}
			msgData = m.GetData()
			dataReq = &m
		} else {
			dataReq = a.info
		}

		cmd, msg, err = a.p.processor.Unmarshal(unmarshalCmd.AppType, unmarshalCmd.CmdId, msgData)
		if err != nil {
			log.Error("agentClient", "unmarshal message,headCmd=%v,error: %v", bm.Cmd, err)
			continue
		}
		err = a.p.processor.Route(n.BaseMessage{MyMessage: msg, TraceId: bm.TraceId}, a, cmd, dataReq)
		if err != nil {
			log.Error("agentClient", "client agentClient route message error: %v,cmd=%v", err, cmd)
			continue
		}
	}
}

func (a *agentPlayer) OnClose() {
	log.Debug("", "服务间连接断开了,info=%v", a.info)
}

func (a *agentPlayer) SendMessage(bm n.BaseMessage) {
	m := bm.MyMessage.(proto.Message)
	data, err := proto.Marshal(m)
	if err != nil {
		log.Error("agentPlayer", "异常,proto.Marshal %v error: %v", reflect.TypeOf(m), err)
		return
	}
	//追加TraceId
	otherData := make([]byte, 0, n.TraceIdLen+1)
	if bm.TraceId != "" {
		otherData = append(otherData, n.FlagOtherTraceId)
		otherData = append(otherData, []byte(bm.TraceId)...)
	}
	err = a.conn.WriteMsg(bm.Cmd.AppType, bm.Cmd.CmdId, data, otherData)
	if err != nil {
		log.Error("agentPlayer", "写信息失败 %v error: %v", reflect.TypeOf(m), err)
	}
}

func (a *agentPlayer) SendData(appType, cmdId uint32, m proto.Message) {
	data, err := proto.Marshal(m)
	if err != nil {
		log.Error("agentPlayer", "异常,proto.Marshal %v error: %v", reflect.TypeOf(m), err)
		return
	}
	err = a.conn.WriteMsg(uint16(appType), uint16(cmdId), data, nil)
	if err != nil {
		log.Error("agentPlayer", "write message %v error: %v", reflect.TypeOf(m), err)
	}
}

func (a *agentPlayer) Close() {
	a.conn.Close()
}
func (a *agentPlayer) Destroy() {
	a.conn.Destroy()
}
