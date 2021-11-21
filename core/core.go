package core

import (
	"os"
	"os/signal"
	"xlddz/core/conf"
	"xlddz/core/log"
	"xlddz/core/module"
)

func Run(mods ...module.Module) {
	// logger
	logger, err := log.New(conf.AppName)
	if err != nil {
		panic(err)
	}
	log.Export(logger)
	defer logger.Close()

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
