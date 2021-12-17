package main

import (
	"xlddz/core"
	_ "xlddz/servers/config/business"
	_ "xlddz/servers/config/conf"
)

func main() {
	core.Start("config")
}
