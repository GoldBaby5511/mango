package main

import (
	"xlddz/core"
	"xlddz/servers/center/business"
)

func main() {
	core.Run(
		new(business.Gate),
		new(business.Module),
	)
}
