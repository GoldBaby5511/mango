package network

import (
	"github.com/golang/protobuf/proto"
	"net"
)

const (
	NormalUser   uint32 = 0
	CommonServer uint32 = 1
)

type (
	BaseAgentInfo struct {
		AgentType    uint32
		AppName      string
		AppType      uint32
		AppId        uint32
		ListenOnAddr string
	}

	Agent interface {
		Run()
		OnClose()
		SendData(appType, cmdId uint32, m proto.Message)
		Close()
		Destroy()
	}

	AgentClient interface {
		Agent
		AgentInfo() *BaseAgentInfo
		LocalAddr() net.Addr
		RemoteAddr() net.Addr
	}

	AgentServer interface {
		Agent
		SendMessage(bm BaseMessage)
	}
)
