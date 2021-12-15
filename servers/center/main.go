package main

import (
	"xlddz/core"
	_ "xlddz/servers/center/business"
	_ "xlddz/servers/center/conf"
)

func main() {
	core.Start()
}
