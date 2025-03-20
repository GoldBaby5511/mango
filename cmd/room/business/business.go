package business

import (
	"github.com/golang/protobuf/proto"
	"mango/api/gateway"
	"mango/api/list"
	"mango/api/property"
	rCMD "mango/api/room"
	tCMD "mango/api/table"
	"mango/api/types"
	"mango/cmd/room/business/player"
	"mango/cmd/room/business/table"
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
	userList = make(map[uint64]*player.Player)
)

func init() {
	g.MsgRegister(&tCMD.ApplyRsp{}, n.AppTable, uint16(tCMD.CMDTable_IDApplyRsp), handleApplyRsp)
	g.MsgRegister(&tCMD.WriteGameScore{}, n.AppTable, uint16(tCMD.CMDTable_IDWriteGameScore), handleWriteGameScore)
	g.MsgRegister(&tCMD.GameOver{}, n.AppTable, uint16(tCMD.CMDTable_IDGameOver), handleGameOver)
	g.MsgRegister(&list.RoomRegisterRsp{}, n.AppList, uint16(list.CMDList_IDRoomRegisterRsp), handleRoomRegisterRsp)
	g.MsgRegister(&rCMD.JoinReq{}, n.AppRoom, uint16(rCMD.CMDRoom_IDJoinReq), handleJoinReq)
	g.MsgRegister(&rCMD.UserActionReq{}, n.AppRoom, uint16(rCMD.CMDRoom_IDUserActionReq), handleUserActionReq)
	g.MsgRegister(&rCMD.ExitReq{}, n.AppRoom, uint16(rCMD.CMDRoom_IDExitReq), handleExitReq)
	g.MsgRegister(&property.ModifyPropertyRsp{}, n.AppProperty, uint16(property.CMDProperty_IDModifyPropertyRsp), handleModifyPropertyRsp)

	g.Skeleton.LoopFunc(1*time.Second, checkMatchTable, timer.LoopForever)
	g.Skeleton.LoopFunc(3*time.Second, table.CheckApplyTable, timer.LoopForever)
}

func handleApplyRsp(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*tCMD.ApplyRsp)
	srcApp := args[n.OtherIndex].(n.BaseAgentInfo)

	for _, v := range m.GetTableIds() {
		table.NewTable(v)
	}
	log.Debug("", "收到桌子,ApplyCount=%d,AttAppid=%d,len=%d",
		m.GetApplyCount(), srcApp.AppId, table.GetTableCount(table.All))

	var req list.RoomRegisterReq
	req.Info = new(types.RoomInfo)
	req.Info.AppInfo = new(types.BaseAppInfo)
	req.Info.AppInfo.Name = proto.String(conf.AppInfo.Name)
	req.Info.AppInfo.Type = proto.Uint32(conf.AppInfo.Type)
	req.Info.AppInfo.Id = proto.Uint32(conf.AppInfo.Id)
	req.Info.Kind = proto.Uint32(200)
	req.Info.Type = (*types.RoomInfo_RoomType)(proto.Int32(int32(types.RoomInfo_Gold)))
	g.SendData2App(n.AppList, n.Send2AnyOne, n.AppList, uint32(list.CMDList_IDRoomRegisterReq), &req)
}

func handleWriteGameScore(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*tCMD.WriteGameScore)
	srcApp := args[n.OtherIndex].(n.BaseAgentInfo)

	log.Debug("", "收到写分,tableId=%d,AttAppid=%d,len=%d",
		m.GetTableId(), srcApp.AppId, table.GetTableCount(table.Free))

	var req property.ModifyPropertyReq
	p := new(types.PropItem)
	p.PropId = (*types.PropType)(proto.Int32(int32(types.PropType_Score)))
	p.PropCount = proto.Int64(100)
	req.UserProps = append(req.UserProps, p)
	g.SendData2App(n.AppProperty, n.Send2AnyOne, n.AppProperty, uint32(property.CMDProperty_IDModifyPropertyReq), &req)
}

func handleGameOver(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*tCMD.GameOver)
	srcApp := args[n.OtherIndex].(n.BaseAgentInfo)

	log.Debug("", "收到结束,table=%d,AttAppid=%d",
		m.GetTableId(), srcApp.AppId)

	for _, p := range userList {
		if p.TableId != m.GetTableId() {
			continue
		}
		setUserState(p, player.StandingInRoom)
	}
	table.GameOver(m.GetTableId())
}

func handleRoomRegisterRsp(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	log.Debug("", "注册返回,AttAppid=%d", b.AgentInfo.AppId)
}

func handleJoinReq(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	srcData := args[n.OtherIndex].(*gateway.TransferDataReq)

	log.Debug("", "进入房间,userId = %v,Gateconnid=%v,appID =%v",
		srcData.GetUserId(), srcData.GetGateconnid(), util.GetLUint32FromUint64(srcData.GetGateconnid()))

	msgRespond := func(errCode int32) {
		var rsp rCMD.JoinRsp
		rsp.AppId = proto.Uint32(conf.AppInfo.Id)
		rsp.ErrInfo = new(types.ErrorInfo)
		rsp.ErrInfo.Code = proto.Int32(errCode)
		rspBm := n.BaseMessage{MyMessage: &rsp, TraceId: ""}
		rspBm.Cmd = n.TCPCommand{AppType: uint16(n.AppRoom), CmdId: uint16(rCMD.CMDRoom_IDJoinRsp)}
		g.SendMessage2Client(rspBm, srcData.GetGateconnid(), 0)
	}

	userID := srcData.GetUserId()
	if _, ok := userList[userID]; ok {
		msgRespond(1)
		return
	}

	p := player.NewPlayer(userID)
	p.GateConnId = util.MakeUint64FromUint32(b.AgentInfo.AppType, b.AgentInfo.AppId)
	userList[userID] = p

	msgRespond(0)
}

func handleUserActionReq(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*rCMD.UserActionReq)
	srcData := args[n.OtherIndex].(*gateway.TransferDataReq)

	msgRespond := func(errCode int32) {
		var rsp rCMD.UserActionRsp
		rsp.Action = (*rCMD.ActionType)(proto.Int32(int32(m.GetAction())))
		rsp.ErrInfo = new(types.ErrorInfo)
		rsp.ErrInfo.Code = proto.Int32(errCode)
		rspBm := n.BaseMessage{MyMessage: &rsp, TraceId: b.TraceId}
		rspBm.Cmd = n.TCPCommand{AppType: uint16(n.AppRoom), CmdId: uint16(rCMD.CMDRoom_IDUserActionRsp)}
		g.SendMessage2Client(rspBm, srcData.GetGateconnid(), 0)
	}

	userID := srcData.GetUserId()
	if _, ok := userList[userID]; !ok {
		msgRespond(1)
		return
	}

	if m.GetAction() == rCMD.ActionType_Ready {
		setUserState(userList[userID], player.HandsUpState)
	}

	msgRespond(0)
}

func handleExitReq(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	srcData := args[n.OtherIndex].(*gateway.TransferDataReq)

	msgRespond := func(errCode int32) {
		var rsp rCMD.ExitRsp
		rsp.ErrInfo = new(types.ErrorInfo)
		rsp.ErrInfo.Code = proto.Int32(errCode)
		rspBm := n.BaseMessage{MyMessage: &rsp, TraceId: b.TraceId}
		rspBm.Cmd = n.TCPCommand{AppType: uint16(n.AppRoom), CmdId: uint16(rCMD.CMDRoom_IDUserActionRsp)}
		g.SendMessage2Client(rspBm, srcData.GetGateconnid(), 0)
	}

	userID := srcData.GetUserId()
	if _, ok := userList[userID]; !ok {
		msgRespond(1)
		return
	}

	delete(userList, userID)
	msgRespond(0)
}

func handleModifyPropertyRsp(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*property.ModifyPropertyRsp)

	log.Debug("", "修改财富返回,userId=%v, opType=%v", m.GetUserId(), m.GetOpType())
}

func checkMatchTable() {
	var matchPlayers []*player.Player
	for _, pl := range userList {
		if pl.State == player.HandsUpState {
			matchPlayers = append(matchPlayers, pl)
		}
	}

	seatCount := apollo.GetConfigAsInt64("座位数量", 3)
	for len(matchPlayers) >= int(seatCount) {
		tablePlayers := matchPlayers[:seatCount]
		matchPlayers = matchPlayers[seatCount:]

		log.Debug("", "allCount=%v,len=%v", len(userList), len(matchPlayers))
		t := table.GetAFreeTable()
		if t == nil {
			log.Warning("", "木有空闲桌子了,allCount=%v,len=%v", len(userList), len(matchPlayers))
			break
		}
		tableID := t.Id
		tableAppID := apollo.GetConfigAsInt64("桌子服务AppID", 1000)
		var req tCMD.MatchTableReq
		req.TableId = proto.Uint64(tableID)
		for i := 0; i < int(seatCount); i++ {
			log.Debug("", "设置用户,userId=%v,tableID=%v", tablePlayers[i].UserID, tableID)
			req.Players = append(req.Players, tablePlayers[i].UserID)
			tablePlayers[i].TableServiceId = uint32(tableAppID)
			tablePlayers[i].TableId = tableID
			tablePlayers[i].SeatId = uint32(i)
			t.Players[uint32(i)] = tablePlayers[i]
			setPlayerToTable(tablePlayers[i], uint32(tableAppID))
			setUserState(tablePlayers[i], player.PlayingState)
		}

		g.SendData2App(n.AppTable, uint32(tableAppID), n.AppTable, uint32(tCMD.CMDTable_IDMatchTableReq), &req)
	}
}

func setPlayerToTable(pl *player.Player, tableAppID uint32) {
	var req tCMD.SetPlayerToTableReq
	req.UserId = proto.Uint64(pl.UserID)
	req.TableId = proto.Uint64(pl.TableId)
	req.SeatId = proto.Uint32(pl.SeatId)
	req.Gateconnid = proto.Uint64(pl.GateConnId)
	g.SendData2App(n.AppTable, tableAppID, n.AppTable, uint32(tCMD.CMDTable_IDSetPlayerToTableReq), &req)
}

func setUserState(p *player.Player, s uint32) {
	if p == nil {
		return
	}
	oldState := p.State
	p.State = s
	if s != oldState {
		var state rCMD.UserStateChange
		state.UserId = proto.Uint64(p.UserID)
		state.TableServiceId = proto.Uint32(p.TableServiceId)
		state.TableId = proto.Uint64(p.TableId)
		state.SeatId = proto.Uint32(p.SeatId)

		rspBm := n.BaseMessage{MyMessage: &state, TraceId: ""}
		rspBm.Cmd = n.TCPCommand{AppType: uint16(n.AppRoom), CmdId: uint16(rCMD.CMDRoom_IDUserStateChange)}
		g.SendMessage2Client(rspBm, p.GateConnId, 0)
	}
}
