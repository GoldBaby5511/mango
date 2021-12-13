package util

import (
	"os"
	"runtime"
	"strconv"
)

func CurMemory() int64 {
	var rtm runtime.MemStats
	runtime.ReadMemStats(&rtm)
	return int64(rtm.Alloc / 1024)
}

//解析参数
func ParseArgs(argType string) (uint32, bool) {
	args := os.Args
	for i := 0; i < len(args); i++ {
		if args[i] == argType && i+1 < len(args) {
			v, err := strconv.Atoi(args[i+1])
			if err == nil {
				return uint32(v), true
			}
		}
	}
	return 0, false
}
