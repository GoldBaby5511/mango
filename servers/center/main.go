package main

import (
	"xlddz/core"
	_ "xlddz/servers/center/business"
)

func main() {
	core.Start("center")
}
