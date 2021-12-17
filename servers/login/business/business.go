package business

import (
	"github.com/golang/protobuf/proto"
	g "xlddz/core/gate"
	"xlddz/core/log"
	n "xlddz/core/network"
	"xlddz/protocol/client"
	"xlddz/protocol/gate"
	"xlddz/protocol/types"
)

func init() {
	g.MsgRegister(&client.LoginReq{}, n.CMDClient, uint16(client.CMDID_Client_IDLoginReq), handleLoginReq)
	g.MsgRegister(&client.LogoutReq{}, n.CMDClient, uint16(client.CMDID_Client_IDLogoutReq), handleLogoutReq)
	g.EventRegister(g.ConnectSuccess, connectSuccess)
	g.EventRegister(g.Disconnect, disconnect)
}

func connectSuccess(args []interface{}) {
	log.Info("连接", "来了老弟,参数数量=%d", len(args))
}

func disconnect(args []interface{}) {
	log.Info("连接", "告辞中,参数数量=%d", len(args))
}

func handleLoginReq(args []interface{}) {
	//a := args[n.AgentIndex].(n.AgentClient)
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*client.LoginReq)
	srcData := args[n.OtherIndex].(*gate.TransferDataReq)

	log.Debug("登录", "收到登录,主渠道=%d,子渠道=%d", m.GetChannelId(), m.GetSiteId())

	sendLoginRsp(srcData.GetGateconnid(), "成功", int32(client.LoginRsp_SUCCESS))
}

func handleLogoutReq(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*client.LogoutReq)
	log.Debug("注销", "注销请求,userId=%v", m.GetUserId())
}

// 发送登录响应
func sendLoginRsp(gateConnId uint64, info string, code int32) {
	log.Info("登录", "发送登录响应,gateConnId=%v,info=%v,code=%v", gateConnId, info, code)

	var rsp client.LoginRsp
	rsp.LoginInfo = proto.String(info)
	rsp.LoginResult = (*client.LoginRsp_Result)(proto.Int32(code))
	rsp.BaseInfo = new(types.BaseUserInfo)
	rsp.BaseInfo.UserId = proto.Uint64(10001)
	rsp.BaseInfo.GameId = proto.Uint64(10001)

	rspBm := n.BaseMessage{MyMessage: &rsp, TraceId: ""}
	rspBm.Cmd = n.TCPCommand{MainCmdID: uint16(n.CMDClient), SubCmdID: uint16(client.CMDID_Client_IDLoginRsp)}
	g.SendMessage2Client(rspBm, gateConnId, 0)
}
