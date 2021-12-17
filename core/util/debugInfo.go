package util

import (
	"fmt"
	"os"
	"path"
	"runtime/debug"
	"time"
)

func TryE(appName string) {
	errs := recover()
	if errs == nil {
		return
	}

	pid := os.Getpid() //获取进程ID

	pathname := ""
	curPath, err := GetCurrentPath()
	if err == nil {
		pathname = curPath + "log/" + appName + "/"
		_, err := os.Stat(pathname)
		if err != nil && os.IsNotExist(err) {
			err = os.MkdirAll(pathname, os.ModePerm)
			if err != nil {
				pathname = ""
			}
		}
	}
	if pathname == "" {
		return
	}

	now := time.Now()
	filename := fmt.Sprintf("%d%02d%02d_%02d_%02d_%02d_pid%d_dump.log",
		now.Year(),
		now.Month(),
		now.Day(),
		now.Hour(),
		now.Minute(),
		now.Second(),
		pid)
	f, err := os.Create(path.Join(pathname, filename))
	if err != nil {
		return
	}

	defer f.Close()

	f.WriteString(fmt.Sprintf("%v\r\n", errs)) //输出panic信息
	f.WriteString("========\r\n")

	f.WriteString(string(debug.Stack())) //输出堆栈信息
}
