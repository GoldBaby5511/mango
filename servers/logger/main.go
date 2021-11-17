package main

import (
	"xlddz/core"
	"xlddz/servers/logger/business"
)

func main() {
	core.Run(
		new(business.Gate),
		new(business.Module),
	)
}
