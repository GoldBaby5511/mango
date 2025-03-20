package player

import (
	"github.com/golang/protobuf/proto"
	"mango/api/gateway"
	"mango/api/list"
	"mango/api/lobby"
	"mango/api/room"
	tCMD "mango/api/table"
	"mango/api/types"
	"mango/cmd/robot/business/game"
	"mango/cmd/robot/business/game/ddz"
	"mango/pkg/conf"
	"mango/pkg/conf/apollo"
	"mango/pkg/log"
	"mango/pkg/module"
	n "mango/pkg/network"
	"mango/pkg/network/protobuf"
	"mango/pkg/timer"
	"mango/pkg/util"
	"math/rand"
	"reflect"
	"strconv"
	"time"
)

const (
	NilState       uint32 = 0
	Logging        uint32 = 1
	LoggedIn       uint32 = 2
	JoinRoom       uint32 = 3
	StandingInRoom uint32 = 4
	HandsUp        uint32 = 5
	PlayingState   uint32 = 6
)

type Player struct {
	a         *agentPlayer
	processor *protobuf.Processor
	skeleton  *module.Skeleton
	roomList  map[uint64]*types.RoomInfo
	userInfo  game.UserInfo
	gameSink  game.Sink
	gameKind  uint32
}

func NewPlayer(account, passWord string, gameKind uint32) *Player {
	p := new(Player)
	p.gameKind = gameKind
	p.userInfo.Account = account
	p.userInfo.PassWord = passWord
	p.userInfo.UserId = 0
	p.userInfo.State = NilState
	p.roomList = make(map[uint64]*types.RoomInfo)
	p.processor = protobuf.NewProcessor()
	p.gameSink = createGameSink(gameKind, p)
	p.skeleton = module.NewSkeleton(conf.GoLen, conf.TimerDispatcherLen, conf.AsynCallLen, conf.ChanRPCLen)
	go func() {
		p.skeleton.Run()
	}()

	p.msgRegister(&gateway.HelloRsp{}, n.AppGate, uint16(gateway.CMDGateway_IDHelloRsp), p.handleHelloRsp)
	p.msgRegister(&lobby.LoginRsp{}, n.AppLobby, uint16(lobby.CMDLobby_IDLoginRsp), p.handleLoginRsp)
	p.msgRegister(&list.RoomListRsp{}, n.AppList, uint16(list.CMDList_IDRoomListRsp), p.handleRoomListRsp)
	p.msgRegister(&room.JoinRsp{}, n.AppRoom, uint16(room.CMDRoom_IDJoinRsp), p.handleJoinRoomRsp)
	p.msgRegister(&room.UserActionRsp{}, n.AppRoom, uint16(room.CMDRoom_IDUserActionRsp), p.handleRoomActionRsp)
	p.msgRegister(&room.UserStateChange{}, n.AppRoom, uint16(room.CMDRoom_IDUserStateChange), p.handleUserStateChange)
	p.msgRegister(&tCMD.GameMessage{}, n.AppTable, uint16(tCMD.CMDTable_IDGameMessage), p.handleGameMessage)

	p.skeleton.AfterFunc(time.Duration(rand.Intn(9)+1)*time.Second, p.connect)

	return p
}

func createGameSink(gameKind uint32, frame game.Frame) game.Sink {
	switch gameKind {
	case 666:
		return ddz.NewDdz(frame)
	}
	return nil
}

func (p *Player) AfterFunc(d time.Duration, cb func()) {
	p.skeleton.AfterFunc(d, cb)
}

func (p *Player) SendGameMessage(bm n.BaseMessage) {
	var gameMessage tCMD.GameMessage
	gameMessage.SubCmdid = proto.Uint32(uint32(bm.Cmd.CmdId))
	gameMessage.Data, _ = proto.Marshal(bm.MyMessage.(proto.Message))
	cmd := n.TCPCommand{AppType: uint16(n.AppTable), CmdId: uint16(tCMD.CMDTable_IDGameMessage)}
	bm = n.BaseMessage{MyMessage: &gameMessage, Cmd: cmd}
	p.sendMessage2Gate(n.AppTable, p.userInfo.TableServiceId, bm)
}

func (p *Player) GetMyInfo() *game.UserInfo {
	return &p.userInfo
}

func (p *Player) GameOver() {
	log.Debug("", "游戏结束消息,UserId=%v,a=%v", p.userInfo.UserId, p.userInfo.Account)

	p.userInfo.State = StandingInRoom
	p.skeleton.AfterFunc(time.Duration(rand.Intn(3)+3)*time.Second, p.actionRoom)
}

func (p *Player) msgRegister(m proto.Message, appType uint32, cmdId uint16, f interface{}) {
	chanRPC := p.skeleton.ChanRPCServer
	p.processor.Register(m, appType, cmdId, chanRPC)
	chanRPC.Register(reflect.TypeOf(m), f)
}

func (p *Player) heartbeat() {
	var req gateway.PulseReq
	p.a.SendData(n.AppGate, uint32(gateway.CMDGateway_IDPulseReq), &req)
}

func (p *Player) connect() {
	tcpClient := new(n.TCPClient)
	tcpClient.Addr = apollo.GetConfig("网关地址", "127.0.0.1:10100")
	if conf.RunInLocalDocker() {
		tcpClient.Addr = "gateway:" + strconv.Itoa(util.GetPortFromIPAddress(tcpClient.Addr))
	}
	tcpClient.PendingWriteNum = 0
	tcpClient.AutoReconnect = false
	tcpClient.NewAgent = func(conn *n.TCPConn) n.AgentServer {
		a := &agentPlayer{tcpClient: tcpClient, conn: conn, p: p}
		p.a = a

		var req gateway.HelloReq
		a.SendData(n.AppGate, uint32(gateway.CMDGateway_IDHelloReq), &req)
		return a
	}

	log.Debug("", "开始连接,UserId=%v,a=%v", p.userInfo.UserId, p.userInfo.Account)

	if tcpClient != nil {
		tcpClient.Start()
	}
}

func (p *Player) checkRoomList() {
	var req list.RoomListReq
	req.ListId = proto.Uint32(0)
	cmd := n.TCPCommand{AppType: uint16(n.AppList), CmdId: uint16(list.CMDList_IDRoomListReq)}
	bm := n.BaseMessage{MyMessage: &req, Cmd: cmd}
	p.sendMessage2Gate(n.AppList, n.Send2AnyOne, bm)
}

func (p *Player) joinRoom() {
	if len(p.roomList) == 0 {
		return
	}
	r := &types.RoomInfo{}
	for _, v := range p.roomList {
		r = v
		break
	}

	log.Debug("", "进入房间,UserId=%v,a=%v,p=%v", p.userInfo.UserId, p.userInfo.Account, p.userInfo.PassWord)

	p.userInfo.State = JoinRoom
	var req room.JoinReq
	cmd := n.TCPCommand{AppType: uint16(n.AppRoom), CmdId: uint16(room.CMDRoom_IDJoinReq)}
	bm := n.BaseMessage{MyMessage: &req, Cmd: cmd}
	p.sendMessage2Gate(r.AppInfo.GetType(), r.AppInfo.GetId(), bm)
}

func (p *Player) actionRoom() {
	if p.userInfo.State != StandingInRoom {
		return
	}

	log.Debug("", "房间动作,UserId=%v,a=%v,p=%v", p.userInfo.UserId, p.userInfo.Account, p.userInfo.PassWord)

	var req room.UserActionReq
	req.Action = (*room.ActionType)(proto.Int32(int32(room.ActionType_Ready)))
	cmd := n.TCPCommand{AppType: uint16(n.AppRoom), CmdId: uint16(room.CMDRoom_IDUserActionReq)}
	bm := n.BaseMessage{MyMessage: &req, Cmd: cmd}
	p.sendMessage2Gate(n.AppRoom, p.userInfo.RoomID, bm)
}

func (p *Player) handleHelloRsp(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*gateway.HelloRsp)

	log.Debug("", "收到hello消息,UserId=%v,a=%v,RspFlag=%v", p.userInfo.UserId, p.userInfo.Account, m.GetRspFlag())

	var req lobby.LoginReq
	req.Account = proto.String(p.userInfo.Account)
	req.Password = proto.String(p.userInfo.PassWord)
	cmd := n.TCPCommand{AppType: uint16(n.AppLobby), CmdId: uint16(lobby.CMDLobby_IDLoginReq)}
	bm := n.BaseMessage{MyMessage: &req, Cmd: cmd, TraceId: b.TraceId}
	p.sendMessage2Gate(n.AppLobby, n.Send2AnyOne, bm)

	p.skeleton.LoopFunc(30*time.Second, p.heartbeat, timer.LoopForever)
}

func (p *Player) handleLoginRsp(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*lobby.LoginRsp)

	p.userInfo.UserId = m.GetBaseInfo().GetUserId()

	log.Debug("", "收到登录回复,UserId=%v,a=%v,Result=%v", p.userInfo.UserId, p.userInfo.Account, m.GetResult())
	if m.GetResult() == lobby.LoginRsp_SUCCESS {
		p.userInfo.State = LoggedIn
		p.skeleton.AfterFunc(time.Duration(rand.Intn(3)+1)*time.Second, p.checkRoomList)
	}
}

func (p *Player) handleRoomListRsp(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*list.RoomListRsp)

	log.Debug("", "收到列表,UserId=%v,a=%v,len=%v", p.userInfo.UserId, p.userInfo.Account, len(m.GetRooms()))
	for _, r := range m.GetRooms() {
		regKey := util.MakeUint64FromUint32(r.AppInfo.GetType(), r.AppInfo.GetId())
		p.roomList[regKey] = r
		p.skeleton.AfterFunc(time.Duration(rand.Intn(3)+1)*time.Second, p.joinRoom)
	}
}

func (p *Player) handleJoinRoomRsp(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*room.JoinRsp)

	log.Debug("", "收到加入,UserId=%v,a=%v,Code=%v", p.userInfo.UserId, p.userInfo.Account, m.GetErrInfo().GetCode())
	if m.GetErrInfo().GetCode() == 0 {
		p.userInfo.State = StandingInRoom
		p.userInfo.RoomID = m.GetAppId()
		p.skeleton.AfterFunc(time.Duration(rand.Intn(3)+1)*time.Second, p.actionRoom)
	}
}

func (p *Player) handleRoomActionRsp(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*room.UserActionRsp)

	log.Debug("hello", "收到动作,UserId=%v,a=%v,Code=%v", p.userInfo.UserId, p.userInfo.Account, m.GetErrInfo().GetCode())
	if m.GetErrInfo().GetCode() == 0 {
		p.userInfo.State = HandsUp
	}
}

func (p *Player) handleUserStateChange(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*room.UserStateChange)

	log.Debug("hello", "状态变化,UserId=%v,a=%v,State=%v,tableId=%v,seatId=%v",
		p.userInfo.UserId, p.userInfo.Account, m.GetUserState(), m.GetTableId(), m.GetSeatId())

	p.userInfo.TableServiceId = m.GetTableServiceId()
	p.userInfo.TableId = m.GetTableId()
	p.userInfo.SeatId = m.GetSeatId()
}

func (p *Player) handleGameMessage(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*tCMD.GameMessage)

	p.gameSink.GameMessage(p.userInfo.SeatId, m.GetSubCmdid(), m.GetData())
}

func (p *Player) sendMessage2Gate(destAppType, destAppid uint32, bm n.BaseMessage) {
	var dataReq gateway.TransferDataReq
	dataReq.DestApptype = proto.Uint32(destAppType)
	dataReq.DestAppid = proto.Uint32(destAppid)
	dataReq.DataApptype = proto.Uint32(uint32(bm.Cmd.AppType))
	dataReq.DataCmdid = proto.Uint32(uint32(bm.Cmd.CmdId))
	dataReq.Data, _ = proto.Marshal(bm.MyMessage.(proto.Message))
	dataReq.Gateconnid = proto.Uint64(0)
	cmd := n.TCPCommand{AppType: uint16(n.AppGate), CmdId: uint16(gateway.CMDGateway_IDTransferDataReq)}
	transBM := n.BaseMessage{MyMessage: &dataReq, Cmd: cmd, TraceId: bm.TraceId}
	p.a.SendMessage(transBM)
}
