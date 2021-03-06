package util

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

func CurMemory() int64 {
	var rtm runtime.MemStats
	runtime.ReadMemStats(&rtm)
	return int64(rtm.Alloc / 1024)
}

func ParseArgsUint32(name string) (uint32, bool) {
	args := os.Args
	for i := 0; i < len(args); i++ {
		a := strings.Split(args[i], "=")
		if len(a) != 2 {
			continue
		}
		if a[0] == name {
			v, err := strconv.Atoi(a[1])
			if err == nil {
				return uint32(v), true
			}
		}
	}
	return 0, false
}

func ParseArgsString(name string) (string, bool) {
	args := os.Args
	for i := 0; i < len(args); i++ {
		a := strings.Split(args[i], "=")
		if len(a) != 2 {
			continue
		}
		if a[0] == name {
			return a[1], true
		}
	}
	return "", false
}

func MakeUint64FromUint32(high, low uint32) uint64 {
	return uint64(high)<<32 | uint64(low)
}

func Get2Uint32FromUint64(v uint64) (uint32, uint32) {
	return GetHUint32FromUint64(v), GetLUint32FromUint64(v)
}

func GetHUint32FromUint64(v uint64) uint32 {
	return uint32(v >> 32)
}

func GetLUint32FromUint64(v uint64) uint32 {
	return uint32(v & 0xFFFFFFFF)
}

func GetIPFromIPAddress(addr string) string {
	a := strings.Split(addr, ":")
	if len(a) != 2 {
		return ""
	}
	return a[0]
}

func GetPortFromIPAddress(addr string) int {
	a := strings.Split(addr, ":")
	if len(a) != 2 {
		return 0
	}
	p, _ := strconv.Atoi(a[1])
	return p
}

func PortInUse(portNumber int) int {
	res := -1
	resStr := ""
	if runtime.GOOS == `windows` {
		var outBytes bytes.Buffer
		cmdStr := fmt.Sprintf("netstat -ano -p tcp | findstr %d", portNumber)
		cmd := exec.Command("cmd", "/c", cmdStr)
		cmd.Stdout = &outBytes
		cmd.Run()
		resStr = outBytes.String()
	} else {
		cmdStr := fmt.Sprintf("lsof -i:%d", portNumber)
		output, _ := exec.Command("sh", "-c", cmdStr).CombinedOutput()
		if len(output) > 0 {
			resStr = string(output)
		}
	}

	if resStr != "" {
		r := regexp.MustCompile(`\s\d+\s`).FindAllString(resStr, -1)
		if len(r) > 0 {
			pid, err := strconv.Atoi(strings.TrimSpace(r[0]))
			if err != nil {
				res = -1
			} else {
				res = pid
			}
		}
	}

	return res
}
