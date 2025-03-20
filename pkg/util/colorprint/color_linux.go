// +build linux

package colorprint

import "fmt"

var (
	FontColor Color = Color{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
)

type Color struct {
	Black       int // 黑色
	Blue        int // 蓝色
	Green       int // 绿色
	Cyan        int // 青色
	Red         int // 红色
	Purple      int // 紫色
	Yellow      int // 黄色
	LightGray   int // 淡灰色
	Gray        int // 灰色
	LightBlue   int // 亮蓝色
	LightGreen  int // 亮绿色
	LightCyan   int // 亮青色
	LightRed    int // 亮红色
	LightPurple int // 亮紫色
	LightYellow int // 亮黄色
	White       int // 白色
}

func ColorPrint(s string, i int) {
	switch i {
	case FontColor.Yellow:
		fmt.Printf("%c[0;40;33m%s%c[0m", 0x1B, s, 0x1B)
	case FontColor.Red:
		fmt.Printf("%c[0;40;31m%s%c[0m", 0x1B, s, 0x1B)
	case FontColor.LightRed:
		fmt.Printf("%c[1;40;31m%s%c[0m", 0x1B, s, 0x1B)
	default:
		fmt.Print(s)
	}
}
