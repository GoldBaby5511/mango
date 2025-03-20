package table

import (
	"github.com/golang/protobuf/proto"
	tCMD "mango/api/table"
	"mango/cmd/table/business/player"
	g "mango/pkg/gate"
	"mango/pkg/log"
	n "mango/pkg/network"
)

const (
	InvalidSeadID = 0xFFFF

	DdzKind int64 = 666
)

type (
	Frame interface {
		SendTableData(seatId uint32, bm n.BaseMessage)
		WriteGameScore()
		GameOver()
	}

	FrameSink interface {
		StartGame(f Frame)
		GameMessage(seatId, cmdId uint32, data []byte)
	}
)

type Table struct {
	Id        uint64
	HostAppId uint32
	gameSink  FrameSink
	Players   map[uint32]*player.Player
}

func NewTable(id uint64, sink FrameSink) *Table {
	t := new(Table)
	t.Id = id
	t.HostAppId = 0
	t.gameSink = sink
	t.Players = make(map[uint32]*player.Player)
	return t
}

func (t *Table) SendTableData(seatId uint32, bm n.BaseMessage) {
	var gameMessage tCMD.GameMessage
	gameMessage.SubCmdid = proto.Uint32(uint32(bm.Cmd.CmdId))
	gameMessage.Data, _ = proto.Marshal(bm.MyMessage.(proto.Message))
	bm.Cmd.CmdId = uint16(tCMD.CMDTable_IDGameMessage)
	bm.MyMessage = &gameMessage

	if seatId == InvalidSeadID {
		for _, pl := range t.Players {
			g.SendMessage2Client(bm, pl.GateConnId, 0)
		}
	} else {
		pl, ok := t.Players[seatId]
		if !ok {
			log.Warning("", "没找到,seatId=%d,id=%v,hostId=%v", seatId, t.Id, t.HostAppId)
			return
		}
		g.SendMessage2Client(bm, pl.GateConnId, 0)
	}
}

func (t *Table) WriteGameScore() {
	var writeScore tCMD.WriteGameScore
	g.SendData2App(n.AppRoom, t.HostAppId, n.AppTable, uint32(tCMD.CMDTable_IDWriteGameScore), &writeScore)
}

func (t *Table) GameOver() {
	t.Players = make(map[uint32]*player.Player)
	var over tCMD.GameOver
	over.TableId = proto.Uint64(t.Id)
	g.SendData2App(n.AppRoom, t.HostAppId, n.AppTable, uint32(tCMD.CMDTable_IDGameOver), &over)
}

func (t *Table) Reset() {
	t.HostAppId = 0
	t.Players = make(map[uint32]*player.Player)
}

func (t *Table) SetPlayer(pl *player.Player) {
	if _, ok := t.Players[pl.SeatId]; ok {
		log.Warning("", "有人了,id=%v,userId=%v,seatId=%v", t.Id, pl.UserId, pl.SeatId)
		return
	}
	t.Players[pl.SeatId] = pl
}

func (t *Table) Start() {
	t.gameSink.StartGame(t)
}

func (t *Table) GameMessage(seatId, cmdId uint32, data []byte) {
	t.gameSink.GameMessage(seatId, cmdId, data)
}
