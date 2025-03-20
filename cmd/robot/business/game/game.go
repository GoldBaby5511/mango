package game

import (
	n "mango/pkg/network"
	"time"
)

const (
	Begin uint32 = 1
	Over  uint32 = 2
)

type (
	UserInfo struct {
		Account        string
		PassWord       string
		UserId         uint64
		State          uint32
		RoomID         uint32
		TableServiceId uint32
		TableId        uint64
		SeatId         uint32
		Scene          uint32
	}

	Sink interface {
		GameMessage(seatId, cmdId uint32, data []byte)
	}

	Frame interface {
		AfterFunc(d time.Duration, cb func())
		SendGameMessage(bm n.BaseMessage)
		GetMyInfo() *UserInfo
		GameOver()
	}
)
