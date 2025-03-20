package business

import (
	"github.com/golang/protobuf/proto"
	"mango/api/gateway"
	"mango/api/lobby"
	"mango/api/property"
	"mango/api/types"
	g "mango/pkg/gate"
	"mango/pkg/log"
	n "mango/pkg/network"
	"mango/pkg/util"
)

var (
	userList = make(map[uint64]*types.BaseUserInfo)
)

func init() {
	g.MsgRegister(&lobby.LoginReq{}, n.AppLobby, uint16(lobby.CMDLobby_IDLoginReq), handleLoginReq)
	g.MsgRegister(&lobby.LogoutReq{}, n.AppLobby, uint16(lobby.CMDLobby_IDLogoutReq), handleLogoutReq)
	g.MsgRegister(&property.QueryPropertyRsp{}, n.AppProperty, uint16(property.CMDProperty_IDQueryPropertyRsp), handleQueryPropertyRsp)
	g.CallBackRegister(g.CbAppControlNotify, appControlNotify)
}

func appControlNotify(args []interface{}) {

}

func handleLoginReq(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*lobby.LoginReq)
	srcData := b.AgentInfo
	gateConnId := util.MakeUint64FromUint32(srcData.AppType, srcData.AppId)

	log.Debug("登录", "收到登录,AppType=%v,AppID=%v,Account=%v,gateConnId=%d,子渠道=%d",
		b.AgentInfo.AppType, b.AgentInfo.AppId, m.GetAccount(), gateConnId, m.GetSiteId())

	var userId uint64 = 0
	for _, v := range userList {
		if v.GetAccount() == m.GetAccount() {
			userId = v.GetUserId()
			v.GateConnid = proto.Uint64(gateConnId)
		}
	}
	if userId == 0 {
		userId = uint64(10000 + len(userList))
		userList[userId] = new(types.BaseUserInfo)
		userList[userId].Account = proto.String(m.GetAccount())
		userList[userId].UserId = proto.Uint64(userId)
		userList[userId].GameId = proto.Uint64(userId)
		userList[userId].GateConnid = proto.Uint64(gateConnId)
	}
	var req property.QueryPropertyReq
	req.UserId = proto.Uint64(userId)
	g.SendData2App(n.AppProperty, n.Send2AnyOne, n.AppProperty, uint32(property.CMDProperty_IDQueryPropertyReq), &req)
}

func handleLogoutReq(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*lobby.LogoutReq)
	log.Debug("注销", "注销请求,userId=%v", m.GetUserId())
}

func handleQueryPropertyRsp(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*property.QueryPropertyRsp)

	if _, ok := userList[m.GetUserId()]; !ok {
		return
	}
	userList[m.GetUserId()].UserProps = append(userList[m.GetUserId()].UserProps, m.GetUserProps()...)

	log.Debug("", "财富查询,userId=%v,len=%v,gateConnId=%d", m.GetUserId(), len(m.GetUserProps()), userList[m.GetUserId()].GetGateConnid())

	var authRsp gateway.AuthInfo
	authRsp.UserId = proto.Uint64(m.GetUserId())
	authRsp.Gateconnid = proto.Uint64(userList[m.GetUserId()].GetGateConnid())
	authRsp.Result = proto.Uint32(uint32(lobby.LoginRsp_SUCCESS))
	g.SendData2App(n.AppGate, util.GetLUint32FromUint64(userList[m.GetUserId()].GetGateConnid()), n.AppGate, uint32(gateway.CMDGateway_IDAuthInfo), &authRsp)

	var rsp lobby.LoginRsp
	rsp.ErrInfo = new(types.ErrorInfo)
	rsp.ErrInfo.Info = proto.String("成功")
	rsp.ErrInfo.Code = proto.Int32(int32(lobby.LoginRsp_SUCCESS))
	rsp.BaseInfo = new(types.BaseUserInfo)
	rsp.BaseInfo = userList[m.GetUserId()]
	rspBm := n.BaseMessage{MyMessage: &rsp, TraceId: ""}
	rspBm.Cmd = n.TCPCommand{AppType: uint16(n.AppLobby), CmdId: uint16(lobby.CMDLobby_IDLoginRsp)}
	g.SendMessage2Client(rspBm, userList[m.GetUserId()].GetGateConnid(), 0)
}
