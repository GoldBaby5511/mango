package main

import (
	_ "mango/cmd/table/business"
	"mango/pkg/gate"
)

func main() {
	gate.Start("table")
}
