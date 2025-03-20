package table

import (
	"mango/pkg/conf/apollo"
	g "mango/pkg/gate"
	"mango/pkg/log"
	n "mango/pkg/network"
	"github.com/golang/protobuf/proto"
	tCMD "mango/api/table"
	"mango/cmd/room/business/player"
)

const (
	All   uint32 = 0
	Free  uint32 = 1
	InUse uint32 = 2
)

var (
	gameTables = make(map[uint64]*Table)
)

type Table struct {
	Id      uint64
	Players map[uint32]*player.Player
}

func NewTable(id uint64) {
	if _, ok := gameTables[id]; ok {
		log.Warning("", "已存在,id=%v", id)
		return
	}
	t := new(Table)
	t.Id = id
	t.Players = make(map[uint32]*player.Player)
	gameTables[id] = t
}

func CheckApplyTable() {
	maxCount := int(apollo.GetConfigAsInt64("最大桌子数量", 3000))
	tableAppID := apollo.GetConfigAsInt64("桌子服务AppID", 0)
	if tableAppID == 0 || GetTableCount(All) > maxCount {
		return
	}

	if GetTableCount(Free) == 0 && GetTableCount(All) <= maxCount {
		var req tCMD.ApplyReq
		req.ApplyCount = proto.Uint32(uint32(apollo.GetConfigAsInt64("申请桌子数", 1000)))
		g.SendData2App(n.AppTable, uint32(tableAppID), n.AppTable, uint32(tCMD.CMDTable_IDApplyReq), &req)
	}
}

func GetAFreeTable() *Table {
	if GetTableCount(Free) == 0 {
		return nil
	}
	for _, v := range gameTables {
		if len(v.Players) == 0 {
			return v
		}
	}
	return nil
}

func GetTableCount(tableType uint32) int {
	switch tableType {
	case All:
		return len(gameTables)
	case Free:
		count := 0
		for _, v := range gameTables {
			if len(v.Players) == 0 {
				count++
			}
		}
		return count
	case InUse:
		count := 0
		for _, v := range gameTables {
			if len(v.Players) != 0 {
				count++
			}
		}
		return count
	default:
		break
	}
	return 0
}
func GameOver(id uint64) {
	if _, ok := gameTables[id]; !ok {
		log.Warning("", "结束,不存在,id=%v", id)
		return
	}

	gameTables[id].Players = make(map[uint32]*player.Player)

	log.Debug("", "桌子信息,all=%v,free=%v,inuse=%v",
		GetTableCount(All), GetTableCount(Free), GetTableCount(InUse))
}
