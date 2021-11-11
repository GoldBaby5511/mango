package console

import (
	"bufio"
	"math"
	"net"
	"strconv"
	"strings"
	"xlddz/core/conf"
	"xlddz/core/network"

	"google.golang.org/protobuf/proto"
)

var server *network.TCPServer

func Init() {
	if conf.ConsolePort == 0 {
		return
	}

	// log.Debug("Console", "Console localhost:=%v", conf.ConsolePort)

	server = new(network.TCPServer)
	server.Addr = "localhost:" + strconv.Itoa(conf.ConsolePort)
	server.MaxConnNum = int(math.MaxInt32)
	server.PendingWriteNum = 100
	server.NewAgent = newAgent

	server.Start()
}

func Destroy() {
	if server != nil {
		server.Close()
	}
}

type Agent struct {
	conn   *network.TCPConn
	reader *bufio.Reader
}

func newAgent(conn *network.TCPConn, id uint64) network.Agent {
	a := new(Agent)
	a.conn = conn
	a.reader = bufio.NewReader(conn)
	return a
}

func (a *Agent) Run() {
	for {
		if conf.ConsolePrompt != "" {
			a.conn.Write([]byte(conf.ConsolePrompt))
		}

		line, err := a.reader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimSuffix(line[:len(line)-1], "\r")

		args := strings.Fields(line)
		if len(args) == 0 {
			continue
		}
		if args[0] == "quit" {
			break
		}

		var c Command
		for _, _c := range commands {
			if _c.name() == args[0] {
				c = _c
				break
			}
		}
		if c == nil {
			a.conn.Write([]byte("command not found, try `help` for help\r\n"))
			continue
		}
		output := c.run(args[1:])
		if output != "" {
			a.conn.Write([]byte(output + "\r\n"))
		}
	}
}

func (a *Agent) OnClose()                                                                        {}
func (a *Agent) SendMessage(bm network.BaseMessage)                                              {}
func (a *Agent) SendMessage2App(destAppType, destAppid uint32, bm network.BaseMessage)           {}
func (a *Agent) SendMessage2Client(bm network.BaseMessage, userID, gateConnID, sessionID uint64) {}
func (a *Agent) SendData(mainCmdID, subCmdID uint32, m proto.Message)                            {}
func (a *Agent) SendData2App(DestApptype uint32, DestAppid uint32, mainCmdID uint32, subCmdID uint32, m proto.Message) {
}
func (a *Agent) LocalAddr() net.Addr {
	return a.conn.LocalAddr()
}
func (a *Agent) RemoteAddr() net.Addr {
	return a.conn.RemoteAddr()
}
func (a *Agent) Close()   {}
func (a *Agent) Destroy() {}
