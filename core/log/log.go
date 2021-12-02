package log

import (
	"errors"
	"fmt"
	"os"
	"path"
	"runtime"
	"time"
	"xlddz/core/util"
)

type LogInfo struct {
	File      string
	Line      int
	Classname string
	Level     int
	LogStr    string
	TimeMs    uint64
}

//全局变量
var (
	logger            *Logger                                    = nil
	screenPrint       int                                        = 1
	GetMinLevelConfig func(key string, defaultValue int64) int64 = nil
	cb                func(i LogInfo)                            = nil
	tempLogInfo       []LogInfo
)

// levels
const (
	trace      = 0
	debugLevel = 1
	info       = 2
	warning    = 3
	errorLevel = 4
	fatalLevel = 5
)

const (
	printDebugLevel   = "[ debug ] "
	printInfoLevel    = "[ info  ] "
	printWarningLevel = "[warning] "
	printErrorLevel   = "[ error ] "
	printFatalLevel   = "[ fatal ] "
)

type Logger struct {
	level    int
	baseFile *os.File
	pathname string
}

func New(appName string) (*Logger, error) {

	pathname := ""
	curPath, err := util.GetCurrentPath()
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
		return nil, errors.New("文件路径创建失败")
	}

	file, err := createNewLogFile(pathname)
	if err != nil {
		return nil, err
	}

	// new
	l := new(Logger)
	l.baseFile = file
	l.pathname = pathname

	return l, nil
}

func createNewLogFile(pathName string) (*os.File, error) {
	now := time.Now()
	filename := fmt.Sprintf("%d%02d%02d_%02d_%02d_%02d.log",
		now.Year(),
		now.Month(),
		now.Day(),
		now.Hour(),
		now.Minute(),
		now.Second())
	file, err := os.Create(path.Join(pathName, filename))
	if err != nil {
		return nil, err
	}
	return file, nil
}

// It's dangerous to call the method on logging
func (l *Logger) Close() {
	if l.baseFile != nil {
		l.baseFile.Close()
	}

	l.baseFile = nil
}

func (l *Logger) doPrintf(level int, logInfo string) {
	if level < l.level {
		return
	}
	if l.baseFile == nil {
		panic("logger closed")
	}

	l.baseFile.WriteString(logInfo + "\n")

	//60M分割文件 1024*1024*60
	fi, err := l.baseFile.Stat()
	if err == nil && fi.Size() >= 1024*1024*60 {
		file, err := createNewLogFile(l.pathname)
		if err == nil {
			l.baseFile.Close()
			l.baseFile = file
		}
	}

	if level == fatalLevel {
		os.Exit(1)
	}
}

// It's dangerous to call the method on logging
func Export(l *Logger) {
	if l != nil {
		logger = l
	}
}

func SetCallback(c func(i LogInfo)) {
	if cb != nil && c != nil {
		return
	}
	cb = c
	if cb != nil {
		for _, logInfo := range tempLogInfo {
			c(logInfo)
		}

		//清空缓存
		tempLogInfo = append([]LogInfo{})
	}

}

//显示日志开关
func SetScreenPrint(print int) {
	screenPrint = print
}

func nowTimeString() string {
	now := time.Now()
	timeStr := fmt.Sprintf("%v-%02d-%02d %02d:%02d:%02d.%09d",
		now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond())
	return timeStr
}

func printLog(classname, file, format string, line, level int, a ...interface{}) {

	//配置等级判断
	if GetMinLevelConfig != nil && level < int(GetMinLevelConfig("最低日志级别", 0)) {
		return
	}

	//组装格式
	if screenPrint != 0 || level >= errorLevel || cb == nil {
		//屏幕打印时调用位置不打印了 + fmt.Sprintf(" << %s, line #%d, func: %v ", file, line, runtime.FuncForPC(pc).Name())
		format = nowTimeString() + GetLogLevelStr(level) + format
		logStr := fmt.Sprintf(format, a...)
		fmt.Println(logStr)

		//失去连接时写入文件保存
		if cb == nil && logger != nil {
			logStr = logStr + fmt.Sprintf(" << %s, line #%d ", file, line)
			logger.doPrintf(level, logStr)
		}
	}

	//日志变量
	logInfo := LogInfo{
		File:      file,
		Line:      line,
		Classname: classname,
		Level:     level,
		LogStr:    fmt.Sprintf(format, a...),
		TimeMs:    uint64(time.Now().UnixNano() / 1000000),
	}
	if cb != nil {
		cb(logInfo)
	} else {
		tempLogInfo = append(tempLogInfo, logInfo)
	}
}

func Debug(classname, format string, a ...interface{}) {
	//获取调用位置信息
	_, file, line, _ := runtime.Caller(2)
	printLog(classname, file, format, line, debugLevel, a...)
}

func Info(classname, format string, a ...interface{}) {
	//获取调用位置信息
	_, file, line, _ := runtime.Caller(2)
	printLog(classname, file, format, line, info, a...)
}

func Warning(classname, format string, a ...interface{}) {
	_, file, line, _ := runtime.Caller(2)
	printLog(classname, file, format, line, warning, a...)
}

func Error(classname, format string, a ...interface{}) {
	_, file, line, _ := runtime.Caller(2)
	printLog(classname, file, format, line, errorLevel, a...)
}

func Fatal(classname, format string, a ...interface{}) {
	_, file, line, _ := runtime.Caller(2)
	printLog(classname, file, format, line, errorLevel, a...)
	os.Exit(1)
}

func GetLogLevelStr(level int) string {
	logLevel := ""
	if level == debugLevel {
		logLevel = printDebugLevel
	} else if level == info {
		logLevel = printInfoLevel
	} else if level == warning {
		logLevel = printWarningLevel
	} else if level == errorLevel {
		logLevel = printErrorLevel
	}
	return logLevel
}

func Close() {
	if logger != nil {
		logger.Close()
	}
}
