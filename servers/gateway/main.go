package main

import (
	"net/http"
	_ "net/http/pprof"
	"os"
	"xlddz/core"
	"xlddz/core/log"
	"xlddz/servers/gateway/business"
)

func main() {
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
