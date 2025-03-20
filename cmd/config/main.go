package main

import (
	_ "mango/cmd/config/business"
	_ "mango/cmd/config/conf"
	"mango/pkg/gate"
)

func main() {
	gate.Start("config")
}
