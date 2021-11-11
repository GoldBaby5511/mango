package main

import (
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"xlddz/core"
	lconf "xlddz/core/conf"
	"xlddz/core/log"
	"xlddz/servers/gateway/business"
	"xlddz/servers/gateway/conf"
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

	lconf.ConsolePort = conf.Server.ConsolePort
	lconf.AppName = conf.Server.AppName
	lconf.AppID = conf.Server.AppID
	lconf.AppType = conf.Server.AppType
	lconf.ApolloDefaultNamespace = "hall.bs.app.gate"

	log.Info("主流程", "服务器启动,AppName=%v,AppType=%v,AppID=%v",
		lconf.AppName, lconf.AppType, lconf.AppID)

	go func() {
		// 启动一个 http server，注意 pprof 相关的 handler 已经自动注册过了
		if err := http.ListenAndServe("0.0.0.0:6060", nil); err != nil {
			log.Fatal("监控", "创建监控失败,%v", err)
		}
		os.Exit(0)
	}()

	core.Run(
		new(business.Module),
		new(business.Gate),
	)
}
