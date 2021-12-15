package conf

var (
	LenStackBuf = 4096

	// console
	ConsolePort   int    = 2021
	ConsolePrompt string = "Leaf$:"
	ProfilePath   string

	// skeleton conf
	GoLen              = 10000
	TimerDispatcherLen = 10000
	AsynCallLen        = 10000
	ChanRPCLen         = 10000

	//服务基础属性
	AppName         string
	AppID           uint32
	AppType         uint32
	ListenOnAddress string
	CenterAddr      string
)
