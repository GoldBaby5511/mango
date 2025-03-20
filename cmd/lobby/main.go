package main

import (
	_ "mango/cmd/lobby/business"
	"mango/pkg/gate"
)

func main() {
	gate.Start("login")
}
