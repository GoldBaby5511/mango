package main

import (
	"os"
	"strconv"
	"xlddz/core"
	lconf "xlddz/core/conf"
	"xlddz/core/log"
	"xlddz/servers/login/business"
	"xlddz/servers/login/conf"
)

func main() {
	//解析参数
	parseArgs := func(argType string) (uint32, bool) {
		args := os.Args
		for i := 0; i < len(args); i++ {
			if args[i] == argType && i+1 < len(args) {
				appID, err := strconv.Atoi(args[i+1])
				if err == nil {
					return uint32(appID), true
				}
			}
		}
		return 0, false
	}

	if v, ok := parseArgs("/AppID"); ok {
		conf.Server.AppID = v
	}
	if v, ok := parseArgs("/AppType"); ok {
		conf.Server.AppType = v
	}

	log.Info("主流程", "服务器启动,AppName=%v,AppType=%v,AppID=%v", lconf.AppName, lconf.AppType, lconf.AppID)

	core.Run(
		new(business.Module),
	)
}
