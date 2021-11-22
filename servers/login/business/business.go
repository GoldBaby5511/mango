package business

import (
	"google.golang.org/protobuf/proto"
	"xlddz/core/gate"
	"xlddz/core/log"
	n "xlddz/core/network"
	"xlddz/core/network/protobuf"
	"xlddz/protocol/client"
	"xlddz/protocol/router"
	"xlddz/protocol/types"
	"xlddz/servers/login/conf"
)

type Module struct {
	*gate.Gate
}

func (m *Module) OnInit() {

	log.Debug("Module", "登录服务器初始化")

	//消息注册
	p := protobuf.NewProcessor()
	p.RegHandle(&client.LoginReq{}, n.CMDClient, uint16(client.CMDID_Client_IDLoginReq), handleLoginReq)
	p.RegHandle(&client.LoginReq{}, n.CMDClient, uint16(client.CMDID_Client_IDLogoutReq), handleLogoutReq)

	m.Gate = &gate.Gate{
		Processor:     p,
		TCPClientAddr: conf.Server.TCPClientAddr,
	}
}

func handleLoginReq(args []interface{}) {
	a := args[n.AGENT_INDEX].(n.Agent)
	b := args[n.DATA_INDEX].(n.BaseMessage)
	m := (b.MyMessage).(*client.LoginReq)
	srcData := args[n.OTHER_INDEX].(*router.DataTransferReq)

	log.Debug("登录", "收到登录,主渠道=%d,子渠道=%d", m.GetChannelId(), m.GetSiteId())

	sendLoginRsp(a, srcData.GetAttGateconnid(), "成功", int32(client.LoginRsp_SUCCESS))
}

func handleLogoutReq(args []interface{}) {
	b := args[n.DATA_INDEX].(n.BaseMessage)
	m := (b.MyMessage).(*client.LoginReq)
	log.Debug("注销", "注销请求,userId=%v", m.GetUserId())
}

// 发送登录响应
func sendLoginRsp(a n.Agent, gateConnId uint64, info string, code int32) {
	log.Info("登录", "发送登录响应,gateConnId=%v,info=%v,code=%v", gateConnId, info, code)

	var rsp client.LoginRsp
	rsp.LoginInfo = proto.String(info)
	rsp.LoginResult = (*client.LoginRsp_Result)(proto.Int32(code))
	rsp.BaseInfo = new(types.BaseUserInfo)
	rsp.BaseInfo.UserId = proto.Uint64(10001)
	rsp.BaseInfo.GameId = proto.Uint64(10001)

	rspBm := n.BaseMessage{MyMessage: &rsp, TraceId: ""}
	rspBm.Cmd = n.TCPCommand{MainCmdID: uint16(n.CMDClient), SubCmdID: uint16(client.CMDID_Client_IDLoginRsp)}
	a.SendMessage2Client(rspBm, 0, gateConnId, 0)
}
