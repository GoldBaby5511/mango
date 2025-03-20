package gate

import (
	"net"
	"reflect"

	"mango/api/center"
	"mango/api/gateway"
	"mango/pkg/conf"
	"mango/pkg/log"
	n "mango/pkg/network"
	"mango/pkg/util"

	"github.com/golang/protobuf/proto"
)

type agentClient struct {
	conn n.Conn
	info n.BaseAgentInfo
}

func (a *agentClient) Run() {
	for {
		bm, msgData, err := a.conn.ReadMsg()
		if err != nil {
			//服务间连接
			if a.info.AppType == n.CommonServer {
				log.Error("agentClient", "异常,网关读取消息失败,info=%v,err=%v", a.info, err)
				continue
			} else {
				log.Warning("agentClient", "异常,网关读取消息失败,info=%v,err=%v", a.info, err)
				break
			}
		}
		if processor == nil {
			log.Error("agentClient", "异常,解析器为nil断开连接,cmd=%v", &bm.Cmd)
			break
		}

		if conf.AppInfo.Type != n.AppCenter && bm.Cmd.AppType == uint16(n.AppCenter) {
			if bm.Cmd.CmdId == uint16(center.CMDCenter_IDAppRegReq) {
				var m center.RegisterAppReq
				_ = proto.Unmarshal(msgData, &m)
				a.info = n.BaseAgentInfo{AgentType: n.CommonServer, AppName: m.GetAppName(), AppType: m.GetAppType(), AppId: m.GetAppId()}
				if agentChanRPC != nil {
					agentChanRPC.Call0(CommonServerReg, a, a.info)
				}
				log.Debug("", "相互注册,%v", a.info)
				mxClients.Lock()
				clients[util.MakeUint64FromUint32(a.info.AppType, a.info.AppId)] = a
				mxClients.Unlock()
				continue
			} else if bm.Cmd.CmdId == uint16(center.CMDCenter_IDHeartBeatReq) {
				//TODO  其他服务传来的心跳暂不处理
				continue
			}
		}

		unmarshalCmd := bm.Cmd
		var cmd, msg, dataReq interface{}
		if bm.Cmd.AppType == uint16(n.AppGate) && bm.Cmd.CmdId == uint16(gateway.CMDGateway_IDTransferDataReq) && conf.AppInfo.Type != n.AppGate {
			var m gateway.TransferDataReq
			_ = proto.Unmarshal(msgData, &m)
			unmarshalCmd = n.TCPCommand{AppType: uint16(m.GetDataApptype()), CmdId: uint16(m.GetDataCmdid())}
			msgData = m.GetData()
			dataReq = &m
			bm.AgentInfo = n.BaseAgentInfo{AgentType: n.NormalUser, AppName: "NormalUser", AppType: util.GetHUint32FromUint64(m.GetGateconnid()), AppId: util.GetLUint32FromUint64(m.GetGateconnid())}
		} else {
			bm.AgentInfo = a.info
			dataReq = a.info
		}

		cmd, msg, err = processor.Unmarshal(unmarshalCmd.AppType, unmarshalCmd.CmdId, msgData)
		if err != nil {
			log.Warning("agentClient", "异常,agentClient反序列化,headCmd=%v,error: %v", bm.Cmd, err)
			continue
		}
		err = processor.Route(n.BaseMessage{MyMessage: msg, AgentInfo: bm.AgentInfo, TraceId: bm.TraceId}, a, cmd, dataReq)
		if err != nil {
			log.Warning("agentClient", "client agentClient route message error: %v,cmd=%v", err, cmd)
			continue
		}
	}
}

func (a *agentClient) OnClose() {
	if agentChanRPC != nil {
		err := agentChanRPC.Call0(Disconnect, a)
		if err != nil {
			log.Warning("agentClient", "agentClient OnClose err=%v", err)
		}
	}

	mxClients.Lock()
	delete(clients, util.MakeUint64FromUint32(a.info.AppType, a.info.AppId))
	mxClients.Unlock()
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

func (a *agentClient) SendData(appType, cmdId uint32, m proto.Message) {
	data, err := proto.Marshal(m)
	if err != nil {
		log.Error("agentClient", "异常,proto.Marshal %v error: %v", reflect.TypeOf(m), err)
		return
	}

	//超长判断
	if len(data) > int(MaxMsgLen-1024) {
		log.Error("agentClient", "异常,消息体超长,type=%v,appType=%v,cmdId=%v,len=%v,max=%v", reflect.TypeOf(m), appType, cmdId, len(data), int(MaxMsgLen-1024))
		return
	}

	err = a.conn.WriteMsg(uint16(appType), uint16(cmdId), data, nil)
	if err != nil {
		log.Error("agentClient", "write message %v error: %v", reflect.TypeOf(m), err)
	}
}

func (a *agentClient) AgentInfo() *n.BaseAgentInfo {
	return &a.info
}
