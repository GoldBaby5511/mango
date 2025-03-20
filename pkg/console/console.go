package console

import (
	"bufio"
	"mango/pkg/conf"
	"mango/pkg/log"
	"mango/pkg/network"
	"github.com/golang/protobuf/proto"
	"math"
	"net"
	"strconv"
	"strings"
)

var server *network.TCPServer

func Init() {
	if conf.AppInfo.Type != network.AppCenter {
		return
	}

	server = new(network.TCPServer)
	server.Addr = "localhost:" + strconv.Itoa(int(conf.DefaultBasePort+conf.AppInfo.Id+1))
	server.MaxConnNum = math.MaxInt32
	server.PendingWriteNum = 100
	server.NewAgent = newAgent

	log.Debug("", "start,addr=%v", server.Addr)

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

func newAgent(conn *network.TCPConn) network.AgentClient {
	a := new(Agent)
	a.conn = conn
	a.reader = bufio.NewReader(conn)
	return a
}

func (a *Agent) Run() {
	for {

		a.conn.Write([]byte("\r\nankots$"))

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
			a.conn.Write([]byte("\r\n" + output + "\r\n"))
		}
	}
}

func (a *Agent) OnClose() {}

func (a *Agent) SendData(appType, cmdId uint32, m proto.Message) {}
func (a *Agent) Close()                                          {}
func (a *Agent) Destroy()                                        {}
func (a *Agent) AgentInfo() *network.BaseAgentInfo {
	return nil
}
func (a *Agent) LocalAddr() net.Addr {
	return nil
}
func (a *Agent) RemoteAddr() net.Addr {
	return nil
}
