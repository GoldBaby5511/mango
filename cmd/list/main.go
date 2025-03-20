package main

import (
	_ "mango/cmd/list/business"
	"mango/pkg/gate"
)

func main() {
	gate.Start("list")
}
