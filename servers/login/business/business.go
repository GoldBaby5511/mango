package business

import (
	"google.golang.org/protobuf/proto"
	"reflect"
	g "xlddz/core/gate"
	"xlddz/core/log"
	"xlddz/core/module"
	n "xlddz/core/network"
	"xlddz/core/network/protobuf"
	"xlddz/protocol/client"
	"xlddz/protocol/gate"
	"xlddz/protocol/types"
	"xlddz/servers/login/conf"
)

var (
	skeleton        = module.NewSkeleton(conf.GoLen, conf.TimerDispatcherLen, conf.AsynCallLen, conf.ChanRPCLen)
	processor       = protobuf.NewProcessor()
	userCount int64 = 0
)

func init() {
	//消息注册
	chanRPC := skeleton.ChanRPCServer
	processor.Register(&client.LoginReq{}, n.CMDClient, uint16(client.CMDID_Client_IDLoginReq), chanRPC)
	processor.Register(&client.LogoutReq{}, n.CMDClient, uint16(client.CMDID_Client_IDLogoutReq), chanRPC)

	chanRPC.Register(reflect.TypeOf(&client.LoginReq{}), handleLoginReq)
	chanRPC.Register(reflect.TypeOf(&client.LogoutReq{}), handleLogoutReq)

	chanRPC.Register(g.ConnectSuccess, connectSuccess)
	chanRPC.Register(g.Disconnect, disconnect)
}

type Gate struct {
	*g.Gate
}

func (m *Gate) OnInit() {
	g.AgentChanRPC = skeleton.ChanRPCServer
	g.Processor = processor
	m.Gate = &g.Gate{
		TCPAddr:       conf.Server.TCPAddr,
		TCPClientAddr: conf.Server.TCPClientAddr,
	}
}

func (m *Gate) OnDestroy() {}

type Module struct {
	*module.Skeleton
}

func (m *Module) OnInit() {

	log.Debug("Module", "登录服务器初始化")

	m.Skeleton = skeleton
}

func (m *Module) OnDestroy() {}

func connectSuccess(args []interface{}) {
	log.Info("连接", "来了老弟,参数数量=%d", len(args))
}

func disconnect(args []interface{}) {
	log.Info("连接", "告辞中,参数数量=%d", len(args))
}

func handleLoginReq(args []interface{}) {
	a := args[n.AGENT_INDEX].(n.AgentClient)
	b := args[n.DATA_INDEX].(n.BaseMessage)
	m := (b.MyMessage).(*client.LoginReq)
	srcData := args[n.OTHER_INDEX].(*gate.TransferDataReq)

	userCount++

	log.Debug("登录", "收到登录,主渠道=%d,子渠道=%d,userCount=%v", m.GetChannelId(), m.GetSiteId(), userCount)

	sendLoginRsp(a, srcData.GetAttGateconnid(), "成功", int32(client.LoginRsp_SUCCESS))
}

func handleLogoutReq(args []interface{}) {
	b := args[n.DATA_INDEX].(n.BaseMessage)
	m := (b.MyMessage).(*client.LogoutReq)
	log.Debug("注销", "注销请求,userId=%v", m.GetUserId())
}

// 发送登录响应
func sendLoginRsp(a n.AgentClient, gateConnId uint64, info string, code int32) {
	log.Info("登录", "发送登录响应,gateConnId=%v,info=%v,code=%v", gateConnId, info, code)

	var rsp client.LoginRsp
	rsp.LoginInfo = proto.String(info)
	rsp.LoginResult = (*client.LoginRsp_Result)(proto.Int32(code))
	rsp.BaseInfo = new(types.BaseUserInfo)
	rsp.BaseInfo.UserId = proto.Uint64(10001)
	rsp.BaseInfo.GameId = proto.Uint64(10001)

	rspBm := n.BaseMessage{MyMessage: &rsp, TraceId: ""}
	rspBm.Cmd = n.TCPCommand{MainCmdID: uint16(n.CMDClient), SubCmdID: uint16(client.CMDID_Client_IDLoginRsp)}
	g.SendMessage2Client(rspBm, 0, gateConnId, 0)
}
