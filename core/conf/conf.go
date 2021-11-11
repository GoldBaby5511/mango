package conf

var (
	LenStackBuf = 4096

	// console
	ConsolePort   int    = 2021
	ConsolePrompt string = "Leaf$:"
	ProfilePath   string

	RouterGoroutineNum int = 1000 //普通服务协程默认数量

	//服务基础属性
	ApolloDefaultNamespace string
	AppName                string
	AppID                  uint32
	AppType                uint32
)
