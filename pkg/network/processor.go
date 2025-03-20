package network

type Processor interface {
	// must goroutine safe
	Route(args ...interface{}) error
	// must goroutine safe
	Unmarshal(appType uint16, cmdId uint16, data []byte) (interface{}, interface{}, error)
	// must goroutine safe
	Marshal(msg interface{}) ([][]byte, error)
}
