package player

const (
	NilState     uint32 = 0
	HandsUpState uint32 = 1
	PlayingState uint32 = 2
	SitdownState uint32 = 3
)

type Player struct {
	UserId     uint64
	State      uint32
	TableId    uint64
	SeatId     uint32
	GateConnId uint64
	SrcAppId   uint32
}

func NewPlayer() *Player {
	p := new(Player)
	p.UserId = 0
	p.State = NilState
	return p
}
