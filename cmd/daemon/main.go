package main

import (
	_ "mango/cmd/daemon/business"
	"mango/pkg/gate"
)

func main() {
	gate.Start("daemon")
}
