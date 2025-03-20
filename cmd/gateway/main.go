package main

import (
	_ "mango/cmd/gateway/business"
	"mango/pkg/gate"
)

func main() {
	gate.Start("gateway")
}
