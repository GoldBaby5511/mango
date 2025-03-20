package util

import (
	"fmt"
	"os"
	"path"
	"runtime/debug"
	"time"
)

func TryE(pathName string) {
	errs := recover()
	if errs == nil {
		return
	}

	filename := fmt.Sprintf("%s_pid%d_dump.log",
		time.Now().Format("2006-01-02_15-04-05"),
		os.Getpid())
	f, err := os.Create(path.Join(pathName, filename))
	if err != nil {
		return
	}

	defer f.Close()

	f.WriteString(fmt.Sprintf("%v\r\n", errs)) //输出panic信息
	f.WriteString("========\r\n")
	f.WriteString(string(debug.Stack())) //输出堆栈信息
}
