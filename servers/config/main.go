package main

import (
	"xlddz/core"
	lconf "xlddz/core/conf"
	"xlddz/core/log"
	"xlddz/servers/config/business"
)

func main() {

	log.Info("主流程", "服务器启动,AppName=%v,AppType=%v,AppID=%v", lconf.AppName, lconf.AppType, lconf.AppID)

	core.Run(
		new(business.Gate),
		new(business.Module),
	)
}
