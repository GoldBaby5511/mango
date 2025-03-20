package main

import (
	_ "mango/cmd/room/business"
	"mango/pkg/gate"
)

func main() {
	gate.Start("room")
}
