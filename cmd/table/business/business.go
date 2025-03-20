package business

import (
	"github.com/golang/protobuf/proto"
	"mango/api/gateway"
	tCMD "mango/api/table"
	"mango/cmd/table/business/player"
	"mango/cmd/table/business/table"
	"mango/cmd/table/business/table/ddz"
	"mango/pkg/conf/apollo"
	g "mango/pkg/gate"
	"mango/pkg/log"
	n "mango/pkg/network"
)

var (
	freeTables = make(map[uint64]*table.Table)
	usedTables = make(map[uint64]*table.Table)
)

func init() {
	g.MsgRegister(&tCMD.ApplyReq{}, n.AppTable, uint16(tCMD.CMDTable_IDApplyReq), handleApplyReq)
	g.MsgRegister(&tCMD.ReleaseReq{}, n.AppTable, uint16(tCMD.CMDTable_IDReleaseReq), handleReleaseReq)
	g.MsgRegister(&tCMD.SetPlayerToTableReq{}, n.AppTable, uint16(tCMD.CMDTable_IDSetPlayerToTableReq), handleSetPlayerToTableReq)
	g.MsgRegister(&tCMD.MatchTableReq{}, n.AppTable, uint16(tCMD.CMDTable_IDMatchTableReq), handleMatchTableReq)
	g.MsgRegister(&tCMD.GameMessage{}, n.AppTable, uint16(tCMD.CMDTable_IDGameMessage), handleGameMessage)
	g.CallBackRegister(g.CbConfigChangeNotify, configChangeNotify)
}

func configChangeNotify(args []interface{}) {
	tableCount := apollo.GetConfigAsInt64("桌子数量", 5000)
	gameKind := apollo.GetConfigAsInt64("游戏类型", 0)
	if len(freeTables) == 0 && len(usedTables) == 0 && tableCount != 0 && gameKind != 0 {
		log.Info("配置", "初始化桌子,tableCount=%d,gameKind=%v", tableCount, gameKind)
		for i := 0; i < int(tableCount); i++ {

			switch gameKind {
			case table.DdzKind:
				freeTables[uint64(i)] = table.NewTable(uint64(i), new(ddz.TableSink))
			default:
				log.Warning("", "异常,游戏类型不存在,gameKind=%v", gameKind)
			}
		}
	}
}

func handleApplyReq(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*tCMD.ApplyReq)
	srcApp := args[n.OtherIndex].(n.BaseAgentInfo)

	if len(freeTables) < int(m.GetApplyCount()) {
		log.Warning("", "空闲桌子不够了,ApplyCount=%d,len=%d", m.GetApplyCount(), len(freeTables))
		return
	}
	log.Debug("", "收到申请,ApplyCount=%d,len=%d", m.GetApplyCount(), len(freeTables))

	var rsp tCMD.ApplyRsp
	rsp.ApplyCount = proto.Uint32(m.GetApplyCount())
	for k, v := range freeTables {
		rsp.TableIds = append(rsp.TableIds, v.Id)
		v.HostAppId = srcApp.AppId
		delete(freeTables, k)
		usedTables[k] = v
		if len(rsp.GetTableIds()) == int(m.GetApplyCount()) {
			break
		}
	}

	g.SendData2App(srcApp.AppType, srcApp.AppId, n.AppTable, uint32(tCMD.CMDTable_IDApplyRsp), &rsp)
}

func handleReleaseReq(args []interface{}) {
	m := args[n.DataIndex].(n.BaseMessage).MyMessage.(*tCMD.ReleaseReq)
	srcApp := args[n.OtherIndex].(n.BaseAgentInfo)

	log.Debug("", "收到释放,ApplyCount=%d,len=%d,srcID=%d", m.GetReleaseCount(), len(freeTables), srcApp.AppId)
	for _, tId := range m.GetTableIds() {
		t := getTable(srcApp.AppId, tId)
		if t == nil {
			continue
		}
		t.Reset()
		delete(usedTables, tId)
		freeTables[tId] = t
	}
}

func handleSetPlayerToTableReq(args []interface{}) {
	m := args[n.DataIndex].(n.BaseMessage).MyMessage.(*tCMD.SetPlayerToTableReq)
	srcApp := args[n.OtherIndex].(n.BaseAgentInfo)
	if _, ok := usedTables[m.GetTableId()]; !ok {
		log.Warning("", "没找到桌子啊,tableId=%v", m.GetTableId())
		return
	}
	t := getTable(srcApp.AppId, m.GetTableId())
	if t == nil {
		log.Warning("", "这桌子不是你的啊,tableId=%v,host=%v,srcId=%v", m.GetTableId(), usedTables[m.GetTableId()].HostAppId, srcApp.AppId)
		return
	}

	pl := getPlayer(m.GetUserId())
	if pl != nil {
		log.Warning("", "已经存在了啊,userId=%v,tableId=%v,host=%v,srcId=%v", pl.UserId, m.GetTableId(), usedTables[m.GetTableId()].HostAppId, srcApp.AppId)
		return
	}

	log.Debug("", "收到入座,UserId=%v,SeatId=%v,TableId=%d,len=%d,srcID=%d", m.GetUserId(), m.GetSeatId(), m.GetTableId(), len(freeTables), srcApp.AppId)

	pl = player.NewPlayer()
	pl.UserId = m.GetUserId()
	pl.TableId = t.Id
	pl.SrcAppId = srcApp.AppId
	pl.SeatId = m.GetSeatId()
	pl.GateConnId = m.GetGateconnid()
	pl.State = player.SitdownState
	t.SetPlayer(pl)
}

func handleMatchTableReq(args []interface{}) {
	m := args[n.DataIndex].(n.BaseMessage).MyMessage.(*tCMD.MatchTableReq)
	srcApp := args[n.OtherIndex].(n.BaseAgentInfo)

	t := getTable(srcApp.AppId, m.GetTableId())
	if t == nil {
		return
	}

	log.Debug("", "收到配桌,TableId=%d,len=%d,srcID=%d", m.GetTableId(), len(freeTables), srcApp.AppId)
	t.Start()
}

func handleGameMessage(args []interface{}) {
	m := args[n.DataIndex].(n.BaseMessage).MyMessage.(*tCMD.GameMessage)
	srcData := args[n.OtherIndex].(*gateway.TransferDataReq)

	userID := srcData.GetUserId()
	pl := getPlayer(userID)
	if pl == nil {
		log.Warning("", "游戏消息,没找到用户啊,userID=%v", userID)
		return
	}

	t := getTable(pl.SrcAppId, pl.TableId)
	if t == nil {
		log.Warning("", "游戏消息,没找到桌子啊,userID=%v,SrcAppId=%v,TableId=%v", userID, pl.SrcAppId, pl.TableId)
		return
	}
	t.GameMessage(pl.SeatId, m.GetSubCmdid(), m.GetData())
}

func getTable(srcAppId uint32, tableID uint64) *table.Table {
	for _, t := range usedTables {
		if t.Id == tableID && t.HostAppId == srcAppId {
			return t
		}
	}
	return nil
}

func getPlayer(userID uint64) *player.Player {
	for _, t := range usedTables {
		for _, pl := range t.Players {
			if pl.UserId == userID {
				return pl
			}
		}
	}
	return nil
}
