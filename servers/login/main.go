package main

import (
	"xlddz/core"
	_ "xlddz/servers/login/business"
)

func main() {
	core.Start("login")
}
