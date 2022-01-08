package business

import (
	"github.com/golang/protobuf/proto"
	"mango/api/property"
	"mango/api/types"
	g "mango/pkg/gate"
	"mango/pkg/log"
	n "mango/pkg/network"
)

var (
	userList map[uint64]int64 = make(map[uint64]int64)
)

func init() {
	g.MsgRegister(&property.QueryPropertyReq{}, n.CMDProperty, uint16(property.CMDID_Property_IDQueryPropertyReq), handleQueryPropertyReq)
	g.MsgRegister(&property.ModifyPropertyReq{}, n.CMDProperty, uint16(property.CMDID_Property_IDModifyPropertyReq), handleModifyPropertyReq)
	g.EventRegister(g.ConnectSuccess, connectSuccess)
	g.EventRegister(g.Disconnect, disconnect)
}

func connectSuccess(args []interface{}) {
}

func disconnect(args []interface{}) {
}

func handleQueryPropertyReq(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*property.QueryPropertyReq)
	srcApp := b.AgentInfo

	if _, ok := userList[m.GetUserId()]; !ok {
		userList[m.GetUserId()] = 1000000000
	}

	log.Debug("", "收到查询,appId=%d,userId=%d", srcApp.AppID, m.GetUserId())

	var rsp property.QueryPropertyRsp
	rsp.UserId = proto.Uint64(m.GetUserId())
	p := new(types.PropItem)
	p.PropId = (*types.PropType)(proto.Int32(int32(types.PropType_Score)))
	p.PropCount = proto.Int64(userList[m.GetUserId()])
	rsp.UserProps = append(rsp.UserProps, p)
	cmd := n.TCPCommand{MainCmdID: uint16(n.CMDProperty), SubCmdID: uint16(property.CMDID_Property_IDQueryPropertyRsp)}
	bm := n.BaseMessage{MyMessage: &rsp, Cmd: cmd}
	g.SendData(srcApp, bm)
}

func handleModifyPropertyReq(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*property.ModifyPropertyReq)

	if _, ok := userList[m.GetUserId()]; !ok {
		userList[m.GetUserId()] = 1000000000
	}

	log.Debug("", "收到修改,appId=%d,userId=%d,opType=%v", b.AgentInfo.AppID, m.GetUserId(), m.GetOpType())

	if m.GetOpType() == 0 {
		userList[m.GetUserId()] += 100
	} else {
		userList[m.GetUserId()] -= 100
	}

	var rsp property.ModifyPropertyRsp
	rsp.UserId = proto.Uint64(m.GetUserId())
	rsp.OpType = proto.Int32(m.GetOpType())
	p := new(types.PropItem)
	p.PropId = (*types.PropType)(proto.Int32(int32(types.PropType_Score)))
	p.PropCount = proto.Int64(100)
	rsp.UserProps = append(rsp.UserProps, p)
	cmd := n.TCPCommand{MainCmdID: uint16(n.CMDProperty), SubCmdID: uint16(property.CMDID_Property_IDModifyPropertyRsp)}
	bm := n.BaseMessage{MyMessage: &rsp, Cmd: cmd}
	g.SendData(b.AgentInfo, bm)
}