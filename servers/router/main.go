package main

import (
	"xlddz/core"
	"xlddz/servers/router/business"
)

func main() {
	core.Run(
		new(business.Gate),
		new(business.Module),
	)
}
