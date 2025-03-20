package main

import (
	_ "mango/cmd/property/business"
	"mango/pkg/gate"
)

func main() {
	gate.Start("property")
}
