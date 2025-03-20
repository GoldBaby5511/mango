package ddz

import (
	"github.com/golang/protobuf/proto"
	"mango/api/gameddz"
	"mango/cmd/robot/business/game"
	"mango/pkg/log"
	n "mango/pkg/network"
	"math/rand"
	"time"
)

type Ddz struct {
	f game.Frame
}

func NewDdz(f game.Frame) game.Sink {
	d := new(Ddz)
	d.f = f
	return d
}

func (d *Ddz) GameMessage(seatId, cmdId uint32, data []byte) {
	switch cmdId {
	case uint32(gameddz.CMDGameddz_IDGameStart):
		d.GameStart(seatId, data)
	case uint32(gameddz.CMDGameddz_IDOutCardRsp):
		d.OutCardRsp(seatId, data)
	case uint32(gameddz.CMDGameddz_IDGameOver):
		d.GameOver(seatId, data)
	default:
		log.Warning("", "未定义消息,seatId=%d,cmdId=%d", seatId, cmdId)
	}
}

func (d *Ddz) GameStart(seatId uint32, data []byte) {
	var m gameddz.GameStart
	_ = proto.Unmarshal(data, &m)

	d.f.GetMyInfo().Scene = game.Begin
	log.Debug("", "游戏开始,UserId=%v,a=%v,TableId=%v,CurrentSeat=%v,p.SeatId=%v",
		d.f.GetMyInfo().UserId, d.f.GetMyInfo().Account, d.f.GetMyInfo().TableId, m.GetCurrentSeat(), d.f.GetMyInfo().SeatId)

	if d.f.GetMyInfo().SeatId == m.GetCurrentSeat() {
		d.f.AfterFunc(time.Duration(rand.Intn(3)+1)*time.Second, d.outCards)
	}
}

func (d *Ddz) OutCardRsp(seatId uint32, data []byte) {
	var m gameddz.OutCardRsp
	_ = proto.Unmarshal(data, &m)

	log.Debug("", "收到出牌消息,UserId=%v,a=%v,CurrentSeat=%v,SeatId=%v",
		d.f.GetMyInfo().UserId, d.f.GetMyInfo().Account, m.GetCurrentSeat(), d.f.GetMyInfo().SeatId)

	if d.f.GetMyInfo().SeatId == m.GetCurrentSeat() {
		d.f.AfterFunc(time.Duration(rand.Intn(3)+1)*time.Second, d.outCards)
	}
}

func (d *Ddz) GameOver(seatId uint32, data []byte) {
	var m gameddz.GameOver
	_ = proto.Unmarshal(data, &m)

	log.Debug("", "游戏结束消息,UserId=%v,a=%v", d.f.GetMyInfo().UserId, d.f.GetMyInfo().Account)

	d.f.GetMyInfo().Scene = game.Over
	d.f.GameOver()
}

func (d *Ddz) outCards() {
	if d.f.GetMyInfo().Scene == game.Over {
		return
	}

	log.Debug("", "出牌,UserId=%v,a=%v,SeatId=%v,Scene=%v",
		d.f.GetMyInfo().UserId, d.f.GetMyInfo().Account, d.f.GetMyInfo().SeatId, d.f.GetMyInfo().Scene)

	var req gameddz.OutCardReq
	for i := 0; i < rand.Intn(3)+1; i++ {
		req.OutCard = append(req.OutCard, byte(rand.Intn(3)+1))
	}
	cmd := n.TCPCommand{AppType: uint16(n.AppTable), CmdId: uint16(gameddz.CMDGameddz_IDOutCardReq)}
	bm := n.BaseMessage{MyMessage: &req, Cmd: cmd}
	d.f.SendGameMessage(bm)
}
