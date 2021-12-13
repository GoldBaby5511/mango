package core

import (
	"os"
	"os/signal"
	"strings"
	"xlddz/core/conf"
	"xlddz/core/conf/apollo"
	"xlddz/core/log"
	"xlddz/core/module"
	"xlddz/core/network"
	"xlddz/core/util"
)

func Run(mods ...module.Module) {
	// logger
	logger, err := log.New(conf.AppName)
	if err != nil {
		panic(err)
	}
	log.Export(logger)
	defer logger.Close()

	if v, ok := util.ParseArgs("/AppID"); ok {
		conf.AppID = v
	}
	if v, ok := util.ParseArgs("/AppType"); ok {
		conf.AppType = v
	}
	if v, ok := util.ParseArgs("/DockerRun"); ok {
		if v == 1 && conf.AppType != network.AppCenter {
			addr := strings.Split(conf.CenterAddr, ":")
			if len(addr) == 2 {
				conf.CenterAddr = "center:" + addr[1]
			}
		}
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
