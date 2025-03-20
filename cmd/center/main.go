package main

import (
	_ "mango/cmd/center/business"
	"mango/pkg/gate"
)

func main() {
	gate.Start("center")
}
