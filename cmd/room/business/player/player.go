package player

const (
	NilState       uint32 = 0
	StandingInRoom uint32 = 1
	HandsUpState   uint32 = 2
	PlayingState   uint32 = 3
)

type Player struct {
	UserID         uint64
	State          uint32
	TableServiceId uint32
	TableId        uint64
	SeatId         uint32
	GateConnId     uint64
}

func NewPlayer(userID uint64) *Player {
	p := new(Player)
	p.UserID = userID
	p.State = NilState
	return p
}
