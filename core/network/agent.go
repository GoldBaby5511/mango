package network

import (
	"github.com/golang/protobuf/proto"
	"net"
)

const (
	NormalUser   uint32 = 0
	CommonServer uint32 = 1
)

type BaseAgentInfo struct {
	AgentType       uint32
	AppName         string
	AppID           uint32
	AppType         uint32
	ListenOnAddress string
}

type AgentClient interface {
	Run()
	OnClose()
	SendData(mainCmdID, subCmdID uint32, m proto.Message)
	AgentInfo() BaseAgentInfo
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	Close()
	Destroy()
}

type AgentServer interface {
	Run()
	OnClose()
	SendMessage(bm BaseMessage)
	SendData(mainCmdID, subCmdID uint32, m proto.Message)
	Close()
	Destroy()
}
