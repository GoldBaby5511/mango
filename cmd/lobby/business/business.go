package business

import (
	"fmt"
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
	g.MsgRegister(&gateway.NetworkDisconnected{}, n.AppGate, uint16(gateway.CMDGateway_IDNetworkDisconnected), handleGatewayNetworkDisconnected)
	g.MsgRegister(&lobby.LoginReq{}, n.AppLobby, uint16(lobby.CMDLobby_IDLoginReq), handleLoginReq)
	g.MsgRegister(&lobby.LogoutReq{}, n.AppLobby, uint16(lobby.CMDLobby_IDLogoutReq), handleLogoutReq)
	g.MsgRegister(&lobby.QueryUserInfoReq{}, n.AppLobby, uint16(lobby.CMDLobby_IDQueryUserInfoReq), handleQueryUserInfoReq)
	g.MsgRegister(&lobby.SyncUserStatus{}, n.AppLobby, uint16(lobby.CMDLobby_IDSyncUserStatus), handleSyncUserStatus)
	g.MsgRegister(&property.QueryPropertyRsp{}, n.AppProperty, uint16(property.CMDProperty_IDQueryPropertyRsp), handleQueryPropertyRsp)
	g.CallBackRegister(g.CbAppControlNotify, appControlNotify)
	g.EventRegister(g.ConnectSuccess, connectSuccess)
	g.EventRegister(g.Disconnect, disconnect)
}

func connectSuccess(args []interface{}) {

}

func disconnect(args []interface{}) {
	a := args[g.AgentIndex].(n.AgentClient)

	log.Debug("", "网络断开,AppType=%v,AppId=%v", a.AgentInfo().AppType, a.AgentInfo().AppId)

	if a.AgentInfo().AppType == n.AppRoom {
		for _, u := range userList {
			if u.GetRoomConnId() == util.MakeUint64FromUint32(a.AgentInfo().AppType, a.AgentInfo().AppId) {
				log.Debug("", "网络断开,清除房间,user=%v", u)
				u.RoomConnId = 0
			}
		}
	}
}

func appControlNotify(args []interface{}) {

}

// 网络断开
func handleGatewayNetworkDisconnected(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*gateway.NetworkDisconnected)

	//log.Debug("", "网络断开,userId=%v", m.GetUserId())

	if u, ok := userList[m.GetUserId()]; ok {
		log.Debug("", "网络断开,user=%v", u)
		if u.GetRoomConnId() != 0 {
			g.SendData2App(n.AppRoom, util.GetLUint32FromUint64(u.GetRoomConnId()), n.AppLobby, uint32(lobby.CMDLobby_IDSyncUserStatus),
				&lobby.SyncUserStatus{
					UserInfo: &types.BaseUserInfo{
						UserId: m.GetUserId(),
						Status: types.BaseUserInfo_offline,
					},
				})
		} else {
			delete(userList, m.GetUserId())
		}
	} else {
		log.Warning("", "网络断开,没找到用户,userId=%v", m.GetUserId())
	}
}

func handleLoginReq(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*lobby.LoginReq)
	srcData := b.AgentInfo
	gateConnId := util.MakeUint64FromUint32(srcData.AppType, srcData.AppId)

	//存在判断
	var userId uint64 = 0
	for _, v := range userList {
		if v.GetAccount() == m.GetAccount() {
			userId = v.GetUserId()
			v.GateConnId = *proto.Uint64(gateConnId)
		}
	}
	if userId == 0 {
		var err error
		userId, err = dbUserLogin(gateConnId, m)
		if err != nil {
			log.Error("", "数据库登录失败,err=%v", err)
			return
		}
	}

	log.Debug("登录", "收到登录,AppType=%v,AppID=%v,Account=%v,gateConnId=%d,userId=%d",
		b.AgentInfo.AppType, b.AgentInfo.AppId, m.GetAccount(), gateConnId, userId)

	if userId == 0 {
		respondUserLogin(userId, gateConnId, int32(lobby.LoginRsp_NOTEXIST), "用户不存在")
		return
	}

	//查询财富
	req := property.QueryPropertyReq{
		UserId:     userId,
		GateConnId: gateConnId,
	}
	g.SendData2App(n.AppProperty, n.Send2AnyOne, n.AppProperty, uint32(property.CMDProperty_IDQueryPropertyReq), &req)
}

// 注销登录
func handleLogoutReq(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*lobby.LogoutReq)

	log.Debug("注销", "注销登录,userId=%v,%v", m.GetUserId(), util.PrintStructFields(b.AgentInfo))

	//if u, ok := userList[m.GetUserId()]; ok {
	//	log.Debug("", "收到注销,userId=%v,gateConnId=%v", m.GetUserId(), u.GetGateConnId())
	//	if u.GetRoomConnId() != 0 {
	//		g.SendData2App(n.AppRoom, util.GetLUint32FromUint64(u.GetRoomConnId()), n.AppGate, uint32(lobby.CMDLobby_IDSyncUserStatus), &lobby.SyncUserStatus{
	//			UserInfo: &types.BaseUserInfo{
	//				UserId: m.GetUserId(),
	//				Status: types.BaseUserInfo_offline,
	//			},
	//		})
	//	}
	//	//delete(userList, m.GetUserId())
	//} else {
	//	log.Warning("", "收到注销,userId=%v,gateConnId=%v", m.GetUserId(), m.GetGateConnId())
	//}
}

func handleQueryUserInfoReq(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*lobby.QueryUserInfoReq)
	srcApp := b.AgentInfo
	userId := m.GetUserId()

	if _, ok := userList[userId]; !ok {
		log.Warning("", "查询用户人没有?,uId=%v", userId)

		g.SendData(srcApp, n.BaseMessage{MyMessage: &lobby.QueryUserInfoRsp{
			ErrInfo: &types.ErrorInfo{
				Code: types.ErrorInfo_failed,
				Info: "查询用户人没有?",
			},
		}, Cmd: n.TCPCommand{MainCmdID: uint16(n.AppLobby), SubCmdID: uint16(lobby.CMDLobby_IDQueryUserInfoRsp)}})
		return
	}
	log.Info("", "查询用户,uid=%d,cId=%v", m.GetUserId(), userList[userId].GetGateConnId())

	rsp := lobby.QueryUserInfoRsp{
		UserInfo: userList[m.GetUserId()],
		ErrInfo: &types.ErrorInfo{
			Code: types.ErrorInfo_success,
		},
	}
	g.SendData(srcApp, n.BaseMessage{MyMessage: &rsp, Cmd: n.TCPCommand{MainCmdID: uint16(n.AppLobby), SubCmdID: uint16(lobby.CMDLobby_IDQueryUserInfoRsp)}})
}
func handleQueryPropertyRsp(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*property.QueryPropertyRsp)

	userId := m.GetUserId()
	connId := m.GetGateConnId()

	if _, ok := userList[userId]; !ok {
		log.Warning("", "财富回来人没了?,uId=%v,cId=%v,code=%v,Info=%v", userId, connId, m.GetErrInfo().GetCode(), m.GetErrInfo().GetInfo())
		respondUserLogin(userId, connId, int32(lobby.LoginRsp_SERVERERROR), fmt.Sprintf("财富回来人没了?uId=%v", m.GetUserId()))
		return
	}

	if m.GetErrInfo().GetCode() != types.ErrorInfo_success {
		log.Warning("", "财富查询失败,uId=%v,cId=%v,code=%v,Info=%v", userId, connId, m.GetErrInfo().GetCode(), m.GetErrInfo().GetInfo())
		respondUserLogin(userId, connId, int32(lobby.LoginRsp_SERVERERROR), fmt.Sprintf("财富查询失败,uId=%v", m.GetUserId()))
		return
	}

	//更新财富
	for _, p := range m.GetUserProps() {
		found := false
		for _, up := range userList[userId].Props {
			if p.GetId() == up.GetId() {
				found = true
				up.Count = p.GetCount()
			}
		}
		if found == false {
			userList[userId].Props = append(userList[m.GetUserId()].Props, p)
		}
	}

	//更新状态
	if userList[userId].GetStatus() == types.BaseUserInfo_none {
		userList[userId].Status = types.BaseUserInfo_free
	}

	//userList[m.GetUserId()].Props = append(userList[m.GetUserId()].Props, m.GetUserProps()...)

	log.Debug("", "财富查询,userId=%v,len=%v,gateConnId=%d,Status=%v", m.GetUserId(), len(m.GetUserProps()), userList[m.GetUserId()].GetGateConnId(), userList[userId].GetStatus())

	authRsp := gateway.AuthInfo{
		UserId:     m.GetUserId(),
		GateConnId: userList[m.GetUserId()].GetGateConnId(),
		Result:     uint32(lobby.LoginRsp_SUCCESS),
	}
	g.SendData2App(n.AppGate, util.GetLUint32FromUint64(userList[m.GetUserId()].GetGateConnId()), n.AppGate, uint32(gateway.CMDGateway_IDAuthInfo), &authRsp)

	//发送回复
	respondUserLogin(userId, connId, int32(lobby.LoginRsp_SUCCESS), "登录成功")
}

// 状态同步
func handleSyncUserStatus(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*lobby.SyncUserStatus)
	srcApp := b.AgentInfo

	user := m.GetUserInfo()
	if _, ok := userList[user.GetUserId()]; !ok {
		log.Warning("", "状态同步人没有?,uId=%v,cId=%v,Status=%v", user.GetUserId(), user.GetGateConnId(), user.GetStatus())
		return
	}

	log.Debug("", "状态同步,uid=%v,Status=%v,TableId=%v,SeatId=%v,原状态,Status=%v,TableId=%v,SeatId=%v",
		user.GetUserId(), user.GetStatus(), user.GetTableId(), user.GetSeatId(),
		userList[user.GetUserId()].GetStatus(), userList[user.GetUserId()].GetTableId(), userList[user.GetUserId()].GetSeatId())

	userList[user.GetUserId()].TableId = user.TableId
	userList[user.GetUserId()].SeatId = user.SeatId
	userList[user.GetUserId()].Status = user.Status
	userList[user.GetUserId()].RoomConnId = util.MakeUint64FromUint32(srcApp.AppType, srcApp.AppId)

	//退出房间
	if user.GetStatus() == types.BaseUserInfo_none {
		userList[user.GetUserId()].RoomConnId = 0
	}
}

// 登录回复
func respondUserLogin(userId, connId uint64, errCode int32, errInfo string) {

	log.Debug("", "登录回复,uId=%v,cId=%v,code=%v,errInfo=%v", userId, connId, errCode, errInfo)

	rsp := lobby.LoginRsp{
		ErrInfo: &types.ErrorInfo{
			Info: errInfo,
			Code: types.ErrorInfo_ResultCode(errCode),
		},
	}
	rsp.UserInfo = new(types.BaseUserInfo)
	rsp.UserInfo = userList[userId]
	rspBm := n.BaseMessage{MyMessage: &rsp, Cmd: n.TCPCommand{MainCmdID: uint16(n.AppLobby), SubCmdID: uint16(lobby.CMDLobby_IDLoginRsp)}}
	g.SendMessage2Client(rspBm, userList[userId].GetGateConnId())
}
