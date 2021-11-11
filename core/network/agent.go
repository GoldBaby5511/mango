package network

import (
	"google.golang.org/protobuf/proto"
	"net"
)

type Agent interface {
	Run()
	OnClose()
	SendMessage(bm BaseMessage)
	SendMessage2App(destAppType, destAppid uint32, bm BaseMessage)
	SendMessage2Client(bm BaseMessage, userID, gateConnID, sessionID uint64)
	SendData(mainCmdID, subCmdID uint32, m proto.Message)
	SendData2App(destAppType, destAppid, mainCmdID, subCmdID uint32, m proto.Message)
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	Close()
	Destroy()
}
