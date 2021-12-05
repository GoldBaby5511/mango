package core

import (
	"os"
	"os/signal"
	"strconv"
	"xlddz/core/conf"
	"xlddz/core/conf/apollo"
	"xlddz/core/log"
	"xlddz/core/module"
	"xlddz/core/network"
)

func Run(mods ...module.Module) {
	// logger
	logger, err := log.New(conf.AppName)
	if err != nil {
		panic(err)
	}
	log.Export(logger)
	defer logger.Close()

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
		conf.AppID = v
	}
	if v, ok := parseArgs("/AppType"); ok {
		conf.AppType = v
	}
	if conf.AppType == network.AppCenter {
		apollo.RegisterConfig("", conf.AppType, conf.AppID, nil)
	}

	for i := 0; i < len(mods); i++ {
		module.Register(mods[i])
	}

	module.Init()

	// close
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	sig := <-c
	log.Info("主流程", "服务器关闭 (signal: %v)", sig)
	module.Destroy()
}
