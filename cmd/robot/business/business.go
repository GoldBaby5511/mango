package business

import (
	"fmt"
	"mango/pkg/conf/apollo"
	g "mango/pkg/gate"
	"mango/pkg/log"
	"mango/cmd/robot/business/player"
	"strconv"
)

var (
	userList = make([]*player.Player, 0)
)

func init() {
	g.CallBackRegister(g.CbConfigChangeNotify, configChangeNotify)
}

func configChangeNotify(args []interface{}) {
	key := args[apollo.KeyIndex].(apollo.ConfKey)
	value := args[apollo.ValueIndex].(apollo.ConfValue)

	switch key.Key {
	case "机器人数量":
		robotCount, _ := strconv.Atoi(value.Value)
		log.Debug("", "开始创建,robotCount=%v", robotCount)
		for i := 0; i < int(robotCount); i++ {
			pl := player.NewPlayer(fmt.Sprintf("robot%05d", i), "", 666)
			if pl != nil {
				userList = append(userList, pl)
			}
		}
	default:
		break
	}
}
