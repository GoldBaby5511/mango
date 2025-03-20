// +build windows

package colorprint

import "syscall"

var (
	kernel32    *syscall.LazyDLL  = syscall.NewLazyDLL(`kernel32.dll`)
	proc        *syscall.LazyProc = kernel32.NewProc(`SetConsoleTextAttribute`)
	CloseHandle *syscall.LazyProc = kernel32.NewProc(`CloseHandle`)

	// 给字体颜色对象赋值
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
	LightGray   int // 淡灰色（系统默认值）
	Gray        int // 灰色
	LightBlue   int // 亮蓝色
	LightGreen  int // 亮绿色
	LightCyan   int // 亮青色
	LightRed    int // 亮红色
	LightPurple int // 亮紫色
	LightYellow int // 亮黄色
	White       int // 白色
}

// 输出有颜色的字体
func ColorPrint(s string, i int) {
	handle, _, _ := proc.Call(uintptr(syscall.Stdout), uintptr(i))
	print(s)
	CloseHandle.Call(handle)
}
