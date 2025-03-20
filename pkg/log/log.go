package log

import (
	"container/list"
	"fmt"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"mango/pkg/util"
	"mango/pkg/util/colorprint"
	"os"
	"path"
	"runtime"
	"sync"
	"time"
)

// levels
const (
	TraceLevel   = 0
	DebugLevel   = 1
	InfoLevel    = 2
	WarningLevel = 3
	ErrorLevel   = 4
	FatalLevel   = 5
)

type (
	LogInfo struct {
		File      string
		Line      int
		Classname string
		Level     int
		LogStr    string
		TimeNs    int64
	}
)

var (
	contextLogger  *logrus.Entry   = nil
	logDir                         = ""
	screenPrint                    = 1
	MinLevel                       = 0 //低于这个等级的日志不输出
	chanPrint                      = make(chan LogInfo, 100)
	cb             func(i LogInfo) = nil
	tmpLogList                     = list.New()
	maxTmpLogCount                 = 100000
	mxTemLogList   sync.Mutex

	levelName = map[int]string{
		TraceLevel:   "[ trace ] ",
		DebugLevel:   "[ debug ] ",
		InfoLevel:    "[ info ] ",
		WarningLevel: "[ warning ] ",
		ErrorLevel:   "[ error  ] ",
		FatalLevel:   "[ fatal ] ",
	}
)

func init() {
	go func() {
		for {
			i := <-chanPrint
			logStr := i.LogStr
			if i.Level >= WarningLevel {
				c := colorprint.FontColor.LightGray
				if i.Level == WarningLevel {
					c = colorprint.FontColor.Yellow
				} else if i.Level == ErrorLevel {
					c = colorprint.FontColor.Red
				} else {
					c = colorprint.FontColor.LightRed
				}
				colorprint.ColorPrint(logStr, c)
				colorprint.ColorPrint("\n", colorprint.FontColor.LightGray)
			} else {
				fmt.Println(logStr)
			}
		}
	}()
}

func New(dir string) error {
	contextLogger = logrus.WithFields(logrus.Fields{})
	logDir = dir
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	logrus.SetLevel(logrus.TraceLevel)

	filename := time.Now().Format("2006-01-02_15-04-05") + ".log"
	logrus.SetOutput(&lumberjack.Logger{
		Filename:   path.Join(logDir, filename),
		MaxSize:    50, // M
		MaxBackups: 100,
		MaxAge:     90,   //days
		Compress:   true, // disabled by default
		LocalTime:  true,
	})

	Info("", "Logger is successfully initialized!")
	return nil
}

func SetCallback(c func(i LogInfo)) {
	if cb != nil && c != nil {
		return
	}
	cb = c
	if cb != nil {
		mxTemLogList.Lock()
		for e := tmpLogList.Front(); e != nil; {
			c(e.Value.(LogInfo))
			e = e.Next()
		}
		tmpLogList.Init()
		mxTemLogList.Unlock()
	}
}

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
	dir, _ := util.GetCurrentPath()
	dir = path.Join(dir, "log")
	if contextLogger != nil {
		dir = logDir
	}
	defer util.TryE(dir)
	if level < MinLevel {
		return
	}

	//组装格式
	if screenPrint != 0 || level >= ErrorLevel || cb == nil {
		logStr := fmt.Sprintf(nowTimeString()+GetLogLevelStr(level)+format, a...)
		chanPrint <- LogInfo{
			LogStr: logStr,
			Level:  level,
		}
	}

	//保存全部
	if contextLogger != nil {
		logStr := fmt.Sprintf(nowTimeString()+" "+format, a...) + fmt.Sprintf(" << %s, line #%d ", file, line)
		switch level {
		case TraceLevel:
			logrus.Trace(logStr)
		case DebugLevel:
			logrus.Debug(logStr)
		case InfoLevel:
			logrus.Info(logStr)
		case WarningLevel:
			logrus.Warning(logStr)
		case ErrorLevel, FatalLevel:
			logrus.Error(logStr)
		default:
			logrus.Info(logStr)
		}
	}

	//日志变量
	logInfo := LogInfo{
		File:      file,
		Line:      line,
		Classname: classname,
		Level:     level,
		LogStr:    fmt.Sprintf(format, a...),
		TimeNs:    time.Now().UnixNano(),
	}
	if cb != nil {
		cb(logInfo)
	} else {
		mxTemLogList.Lock()
		if tmpLogList.Len() > maxTmpLogCount {
			tmpLogList.Remove(tmpLogList.Front())
		}
		tmpLogList.PushBack(logInfo)
		mxTemLogList.Unlock()
	}
}

func Trace(classname, format string, a ...interface{}) {
	_, file, line, _ := runtime.Caller(2)
	printLog(classname, file, format, line, TraceLevel, a...)
}

func Debug(classname, format string, a ...interface{}) {
	_, file, line, _ := runtime.Caller(2)
	printLog(classname, file, format, line, DebugLevel, a...)
}

func Info(classname, format string, a ...interface{}) {
	_, file, line, _ := runtime.Caller(2)
	printLog(classname, file, format, line, InfoLevel, a...)
}

func Warning(classname, format string, a ...interface{}) {
	_, file, line, _ := runtime.Caller(2)
	printLog(classname, file, format, line, WarningLevel, a...)
}

func Error(classname, format string, a ...interface{}) {
	_, file, line, _ := runtime.Caller(2)
	printLog(classname, file, format, line, ErrorLevel, a...)
}

func Fatal(classname, format string, a ...interface{}) {
	_, file, line, _ := runtime.Caller(2)
	printLog(classname, file, format, line, FatalLevel, a...)
	fatalExit()
}

func fatalExit() {
	time.Sleep(time.Second / 2)
	os.Exit(1)
}

func GetLogLevelStr(level int) string {
	if _, ok := levelName[level]; ok {
		return levelName[level]
	}
	return ""
}
