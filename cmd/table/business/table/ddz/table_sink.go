package ddz

import (
	"github.com/golang/protobuf/proto"
	"mango/api/gameddz"
	"mango/cmd/table/business/table"
	"mango/pkg/log"
	n "mango/pkg/network"
	"math/rand"
	"time"
)

const (
	playerCount = 3
)

type TableSink struct {
	frame         table.Frame
	userHandCards [playerCount][]uint8
	bottomCards   []uint8
}

func (s *TableSink) StartGame(f table.Frame) {
	s.frame = f

	c := cardData
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(c), func(i, j int) {
		c[i], c[j] = c[j], c[i]
	})

	s.userHandCards[0] = c[17*0 : 17*1]
	s.userHandCards[1] = c[17*1 : 17*2]
	s.userHandCards[2] = c[17*2 : 17*3]
	s.bottomCards = c[17*3:]

	log.Debug("", "游戏开始")

	var start gameddz.GameStart
	start.CurrentSeat = proto.Uint32(0)
	for i := 0; i < playerCount; i++ {
		start.HandCard = make([][]byte, playerCount)
		start.HandCard[i] = s.userHandCards[0]
		bm := n.BaseMessage{MyMessage: &start, TraceId: ""}
		bm.Cmd = n.TCPCommand{AppType: uint16(n.AppTable), CmdId: uint16(gameddz.CMDGameddz_IDGameStart)}
		s.frame.SendTableData(uint32(i), bm)
	}
}

func (s *TableSink) EndGame() {

}

func (s *TableSink) GameMessage(seatId, cmdId uint32, data []byte) {
	switch cmdId {
	case uint32(gameddz.CMDGameddz_IDCallLandReq):
		s.CallLandReq(seatId, data)
	case uint32(gameddz.CMDGameddz_IDOutCardReq):
		s.OutCardReq(seatId, data)
	case uint32(gameddz.CMDGameddz_IDGameDataReq):
		s.GameDataReq(seatId, data)
	default:
		log.Warning("", "未定义消息,seatId=%d,cmdId=%d", seatId, cmdId)
	}
}

func (s *TableSink) CallLandReq(seatId uint32, data []byte) {
	var m gameddz.CallLandReq
	_ = proto.Unmarshal(data, &m)

	var rsp gameddz.CallLandRsp
	bm := n.BaseMessage{MyMessage: &rsp, TraceId: ""}
	bm.Cmd = n.TCPCommand{AppType: uint16(n.AppTable), CmdId: uint16(gameddz.CMDGameddz_IDCallLandRsp)}
	s.frame.SendTableData(table.InvalidSeadID, bm)

	log.Debug("", "叫地主消息,seatId=%d", seatId)
}

func (s *TableSink) OutCardReq(seatId uint32, data []byte) {
	var m gameddz.OutCardReq
	_ = proto.Unmarshal(data, &m)

	if len(m.GetOutCard()) >= len(s.userHandCards[seatId]) {
		s.userHandCards[seatId] = append([]uint8{})
	} else {
		s.userHandCards[seatId] = s.userHandCards[seatId][:len(s.userHandCards[seatId])-len(m.GetOutCard())]
	}

	var rsp gameddz.OutCardRsp
	bm := n.BaseMessage{MyMessage: &rsp, TraceId: ""}
	bm.Cmd = n.TCPCommand{AppType: uint16(n.AppTable), CmdId: uint16(gameddz.CMDGameddz_IDOutCardRsp)}
	s.frame.SendTableData(seatId, bm)

	log.Debug("", "出牌消息,seatId=%d,len=%v", seatId, len(s.userHandCards[seatId]))

	if len(s.userHandCards[seatId]) == 0 {
		log.Debug("", "本局结束")

		var over gameddz.GameOver
		bm := n.BaseMessage{MyMessage: &over, TraceId: ""}
		bm.Cmd = n.TCPCommand{AppType: uint16(n.AppTable), CmdId: uint16(gameddz.CMDGameddz_IDGameOver)}
		s.frame.SendTableData(table.InvalidSeadID, bm)

		s.frame.WriteGameScore()
		s.frame.GameOver()
	}
}

func (s *TableSink) GameDataReq(seatId uint32, data []byte) {
	var m gameddz.GameDataReq
	_ = proto.Unmarshal(data, &m)

	var rsp gameddz.GameDataRsp
	bm := n.BaseMessage{MyMessage: &rsp, TraceId: ""}
	bm.Cmd = n.TCPCommand{AppType: uint16(n.AppTable), CmdId: uint16(gameddz.CMDGameddz_IDGameDataRsp)}
	s.frame.SendTableData(table.InvalidSeadID, bm)

	log.Debug("", "数据消息,seatId=%d", seatId)
}
