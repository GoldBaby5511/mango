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
	logger := new(Logger)
	logger.baseFile = file
	logger.pathname = pathname

	return logger, nil
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
func (logger *Logger) Close() {
	if logger.baseFile != nil {
		logger.baseFile.Close()
	}

	logger.baseFile = nil
}

func (logger *Logger) doPrintf(level int, logInfo string) {
	if level < logger.level {
		return
	}
	if logger.baseFile == nil {
		panic("logger closed")
	}

	logger.baseFile.WriteString(logInfo + "\n")

	//60M分割文件 1024*1024*60
	fi, err := logger.baseFile.Stat()
	if err == nil && fi.Size() >= 1024*1024*60 {
		file, err := createNewLogFile(logger.pathname)
		if err == nil {
			logger.baseFile.Close()
			logger.baseFile = file
		}
	}

	if level == fatalLevel {
		os.Exit(1)
	}
}

//临时存储服务器未连接时的信息
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
	gLogger                *Logger                                               = nil
	gChanCall              chan LogInfo                                          = nil
	gScreenPrint           bool                                                  = true
	ApolloGetConfigAsInt64 func(nameSpace, key string, defaultValue int64) int64 = nil
	gtempLogInfo           []LogInfo
)

// It's dangerous to call the method on logging
func Export(logger *Logger) {
	if logger != nil {
		gLogger = logger
	}
}

func SetLogCallBack(c chan LogInfo) {
	//不接受重复赋值
	if gChanCall != nil && c != nil {
		return
	}

	gChanCall = c

	//调用时不为空被视为服务连接成功,优先将缓存日志写入日志服务器，为空视为与日志断开
	if gChanCall != nil {
		for _, logInfo := range gtempLogInfo {
			gChanCall <- logInfo
		}

		//清空缓存
		gtempLogInfo = append([]LogInfo{})
	}

}

//显示日志开关
func SetScreenPrint(print bool) {
	gScreenPrint = print
}

func nowTimeString() string {
	now := time.Now()
	timeStr := fmt.Sprintf("%v-%02d-%02d %02d:%02d:%02d.%09d",
		now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond())
	return timeStr
}

func printLog(classname, file, format string, line, level int, a ...interface{}) {
	//配置等级判断
	if ApolloGetConfigAsInt64 != nil && level < int(ApolloGetConfigAsInt64("", "最低日志级别", 0)) {
		return
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
	//丢到日志服务器
	if gChanCall != nil {
		gChanCall <- logInfo
	} else {
		gtempLogInfo = append(gtempLogInfo, logInfo)
	}

	//组装格式
	if gScreenPrint || level >= errorLevel || gChanCall == nil {
		//屏幕打印时调用位置不打印了 + fmt.Sprintf(" << %s, line #%d, func: %v ", file, line, runtime.FuncForPC(pc).Name())
		logLevel := GetLogLevelStr(level)

		format = nowTimeString() + logLevel + format
		logStr := fmt.Sprintf(format, a...)
		fmt.Println(logStr)

		//失去连接时写入文件保存
		if gChanCall == nil && gLogger != nil {
			logStr = logStr + fmt.Sprintf(" << %s, line #%d ", file, line)
			gLogger.doPrintf(level, logStr)
		}
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
	if gLogger != nil {
		gLogger.Close()
	}
}
