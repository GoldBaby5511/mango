package main

import (
	_ "mango/cmd/logger/business"
	"mango/pkg/gate"
)

func main() {
	gate.Start("logger")
}
