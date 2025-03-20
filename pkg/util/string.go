package util

import (
	"strconv"
	"strings"
)

// 字符串转数组 - 默认|分隔
func SplitToInt(val string, psep ...string) []int {
	sep := "|"
	if len(psep) == 1 {
		sep = psep[0]
	}
	if len(val) == 0 {
		return []int{}
	}
	vals := strings.Split(val, sep)
	ret := make([]int, len(vals))
	for i, v := range vals {
		vint, _ := strconv.Atoi(v)
		ret[i] = vint
	}
	return ret
}

// 字符串转数组 - 默认|分隔
func SplitToInt32(val string, psep ...string) []int32 {
	sep := "|"
	if len(psep) == 1 {
		sep = psep[0]
	}
	if len(val) == 0 {
		return []int32{}
	}
	vals := strings.Split(val, sep)
	ret := make([]int32, len(vals))
	for i, v := range vals {
		vint, _ := strconv.Atoi(v)
		ret[i] = int32(vint)
	}
	return ret
}

func Uint64Slice2String(val []uint64, psep string) string {
	if len(val) == 1 {
		return strconv.Itoa(int(val[0]))
	}
	str := make([]string, 0)
	for _, v := range val {
		str = append(str, strconv.Itoa(int(v)))
	}
	return strings.Join(str, psep)
}
