package main

import (
	"xlddz/core"
	_ "xlddz/servers/logger/business"
	_ "xlddz/servers/logger/conf"
)

func main() {
	core.Start()
}
