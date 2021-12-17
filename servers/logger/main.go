package main

import (
	"xlddz/core"
	_ "xlddz/servers/logger/business"
)

func main() {
	core.Start("logger")
}
