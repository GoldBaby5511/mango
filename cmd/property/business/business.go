package business

import (
	"mango/api/property"
	"mango/api/types"
	g "mango/pkg/gate"
	"mango/pkg/log"
	n "mango/pkg/network"
)

var (
	userList = make(map[uint64]int64)
)

func init() {
	g.MsgRegister(&property.QueryPropertyReq{}, n.AppProperty, uint16(property.CMDProperty_IDQueryPropertyReq), handleQueryPropertyReq)
	g.MsgRegister(&property.ModifyPropertyReq{}, n.AppProperty, uint16(property.CMDProperty_IDModifyPropertyReq), handleModifyPropertyReq)
	g.MsgRegister(&property.WriteGameScoreReq{}, n.AppProperty, uint16(property.CMDProperty_IDWriteGameScoreReq), handleWriteGameScoreReq)
	g.EventRegister(g.ConnectSuccess, connectSuccess)
	g.EventRegister(g.Disconnect, disconnect)
}

func connectSuccess(args []interface{}) {
}

func disconnect(args []interface{}) {
}

// 查询财富
func handleQueryPropertyReq(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*property.QueryPropertyReq)
	srcApp := b.AgentInfo

	//if _, ok := userList[m.GetUserId()]; !ok {
	//	userList[m.GetUserId()] = 1000000000
	//}

	log.Debug("", "收到查询,appId=%d,userId=%d", srcApp.AppId, m.GetUserId())

	rsp := property.QueryPropertyRsp{
		UserId: m.GetUserId(),
	}

	ps, err := dbQueryProperty(m.GetUserId())
	if err != nil {
		rsp.ErrInfo = &types.ErrorInfo{
			Code: types.ErrorInfo_failed,
			Info: err.Error(),
		}
	} else {
		rsp.UserProps = ps
	}

	//p := &types.PropItem{
	//	Id:    types.PropItem_coin,
	//	Count: userList[m.GetUserId()],
	//}
	//
	//rsp.UserProps = append(rsp.UserProps, p)

	cmd := n.TCPCommand{MainCmdID: uint16(n.AppProperty), SubCmdID: uint16(property.CMDProperty_IDQueryPropertyRsp)}
	bm := n.BaseMessage{MyMessage: &rsp, Cmd: cmd}
	g.SendData(srcApp, bm)
}

func handleModifyPropertyReq(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*property.ModifyPropertyReq)

	if _, ok := userList[m.GetUserId()]; !ok {
		userList[m.GetUserId()] = 1000000000
	}

	log.Debug("", "收到修改,appId=%d,userId=%d,opType=%v", b.AgentInfo.AppId, m.GetUserId(), m.GetOpType())

	if m.GetOpType() == 0 {
		userList[m.GetUserId()] += 100
	} else {
		userList[m.GetUserId()] -= 100
	}

	rsp := property.ModifyPropertyRsp{
		UserId: m.GetUserId(),
		OpType: m.GetOpType(),
	}
	p := &types.PropItem{
		Id:    types.PropItem_coin,
		Count: 100,
	}
	rsp.UserProps = append(rsp.UserProps, p)
	cmd := n.TCPCommand{MainCmdID: uint16(n.AppProperty), SubCmdID: uint16(property.CMDProperty_IDModifyPropertyRsp)}
	bm := n.BaseMessage{MyMessage: &rsp, Cmd: cmd}
	g.SendData(b.AgentInfo, bm)
}

// 修改积分
func handleWriteGameScoreReq(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*property.WriteGameScoreReq)
	srcApp := b.AgentInfo

	rsp := property.WriteGameScoreRsp{
		UserId: m.UserId,
		ErrInfo: &types.ErrorInfo{
			Code: types.ErrorInfo_success,
			Info: "",
		},
	}

	log.Debug("", "收到修改积分,appId=%d,userId=%d,matchId=%d,matchNo=%d,taskForward=%v", srcApp.AppId, m.GetUserId(), m.GetMatchId(), m.GetMatchNo(), m.GetTaskForward())

	err := dbWriteGameScore(m)
	if err != nil {
		rsp.ErrInfo.Code = types.ErrorInfo_failed
		rsp.ErrInfo.Info = err.Error()
		log.Error("", "dbWriteGameScore err=%v", err)
	}

	//消息回复
	g.SendData2App(srcApp.AppType, srcApp.AppId, n.AppProperty, uint32(property.CMDProperty_IDWriteGameScoreRsp), &rsp)
}
