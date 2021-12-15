package main

import (
	"xlddz/core"
	_ "xlddz/servers/login/business"
	_ "xlddz/servers/login/conf"
)

func main() {
	core.Start()
	//core.Run(
	//	new(business.Gate),
	//	new(business.Module),
	//)
}
