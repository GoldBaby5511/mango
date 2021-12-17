package core

import (
	"os"
	"os/signal"
	"xlddz/core/conf"
	"xlddz/core/gate"
	"xlddz/core/log"
)

func Start(appName string) {
	conf.AppInfo.AppName = appName
	// logger
	logger, err := log.New(conf.AppInfo.AppName)
	if err != nil {
		panic(err)
	}
	log.Export(logger)
	defer logger.Close()

	//args
	conf.ParseCmdArgs()

	gate.Start()

	// close
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	sig := <-c
	log.Info("主流程", "服务器关闭 (signal: %v)", sig)
	gate.Stop()
}
