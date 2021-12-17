package gate

import (
	"github.com/golang/protobuf/proto"
	"net"
	"reflect"
	"xlddz/core/conf"
	"xlddz/core/conf/apollo"
	"xlddz/core/log"
	n "xlddz/core/network"
	"xlddz/protocol/center"
	"xlddz/protocol/config"
	"xlddz/protocol/gate"
)

//代理
type agentClient struct {
	id   uint64
	conn n.Conn
	info n.BaseAgentInfo
}

func (a *agentClient) Run() {
	for {
		bm, msgData, err := a.conn.ReadMsg()
		if err != nil {
			log.Warning("agentClient", "异常,网关读取消息失败,id=%v,err=%v", a.id, err)
			break
		}
		if processor == nil {
			log.Error("agentClient", "异常,解析器为nil断开连接,cmd=%v", &bm.Cmd)
			break
		}
		if bm.Cmd.MainCmdID == uint16(n.CMDConfig) && bm.Cmd.SubCmdID == uint16(config.CMDID_Config_IDApolloCfgRsp) {
			apollo.ProcessReq(&bm.Cmd, msgData)
			continue
		}
		if conf.AppInfo.AppType != n.AppCenter && bm.Cmd.MainCmdID == uint16(n.CMDCenter) {
			if bm.Cmd.SubCmdID == uint16(center.CMDID_Center_IDAppRegReq) {
				var m center.RegisterAppReq
				_ = proto.Unmarshal(msgData, &m)
				a.info = n.BaseAgentInfo{AgentType: n.CommonServer, AppName: m.GetAppName(), AppType: m.GetAppType(), AppID: m.GetAppId()}
				log.Debug("", "相互注册,%v", a.info)
			} else if bm.Cmd.SubCmdID == uint16(center.CMDID_Center_IDPulseNotify) {

			}
			continue
		}

		unmarshalCmd := bm.Cmd
		var cmd, msg, dataReq interface{}
		if bm.Cmd.MainCmdID == uint16(n.CMDGate) && bm.Cmd.SubCmdID == uint16(gate.CMDID_Gate_IDTransferDataReq) && conf.AppInfo.AppType != n.AppGate {
			var m gate.TransferDataReq
			_ = proto.Unmarshal(msgData, &m)
			unmarshalCmd = n.TCPCommand{MainCmdID: uint16(m.GetDataCmdKind()), SubCmdID: uint16(m.GetDataCmdSubid())}
			msgData = m.GetData()
			dataReq = &m
		}

		cmd, msg, err = processor.Unmarshal(unmarshalCmd.MainCmdID, unmarshalCmd.SubCmdID, msgData)
		if err != nil {
			log.Error("agentClient", "unmarshal message,headCmd=%v,error: %v", bm.Cmd, err)
			continue
		}
		err = processor.Route(n.BaseMessage{MyMessage: msg, TraceId: bm.TraceId}, a, cmd, dataReq)
		if err != nil {
			log.Error("agentClient", "client agentClient route message error: %v,cmd=%v", err, cmd)
			continue
		}
	}
}

func (a *agentClient) OnClose() {
	if agentChanRPC != nil {
		err := agentChanRPC.Call0(Disconnect, a, a.id)
		if err != nil {
			log.Error("agentClient", "chanrpc error: %v", err)
		}
	}
}

func (a *agentClient) LocalAddr() net.Addr {
	return a.conn.LocalAddr()
}

func (a *agentClient) RemoteAddr() net.Addr {
	return a.conn.RemoteAddr()
}

func (a *agentClient) Close() {
	a.conn.Close()
}

func (a *agentClient) Destroy() {
	a.conn.Destroy()
}

func (a *agentClient) SendData(mainCmdID, subCmdID uint32, m proto.Message) {
	data, err := proto.Marshal(m)
	if err != nil {
		log.Error("agentClient", "异常,proto.Marshal %v error: %v", reflect.TypeOf(m), err)
		return
	}
	err = a.conn.WriteMsg(uint16(mainCmdID), uint16(subCmdID), data, nil)
	if err != nil {
		log.Error("agentClient", "write message %v error: %v", reflect.TypeOf(m), err)
	}
}

func (a *agentClient) AgentInfo() n.BaseAgentInfo {
	return a.info
}
