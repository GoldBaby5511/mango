package main

import (
	_ "mango/cmd/robot/business"
	"mango/pkg/gate"
)

func main() {
	gate.Start("robot")
}
