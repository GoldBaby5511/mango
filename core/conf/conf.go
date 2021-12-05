package conf

var (
	LenStackBuf = 4096

	// console
	ConsolePort   int    = 2021
	ConsolePrompt string = "Leaf$:"
	ProfilePath   string

	//服务基础属性
	AppName         string
	AppID           uint32
	AppType         uint32
	ListenOnAddress string
)
