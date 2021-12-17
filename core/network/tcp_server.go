package network

import (
	"net"
	"runtime"
	"sync"
	"time"
	"xlddz/core/log"
	"xlddz/core/util"
)

type TCPServer struct {
	Addr            string
	MaxConnNum      int
	PendingWriteNum int
	NewAgent        func(*TCPConn, uint64) AgentClient
	GetConfig       func(key string, defaultValue int64) int64
	ln              net.Listener
	conns           ConnSet
	mutexConns      sync.Mutex
	wgLn            sync.WaitGroup
	wgConns         sync.WaitGroup
	memOverLimit    bool
	rwMemLimit      sync.RWMutex
	agentId         uint64

	// msg parser
	LenMsgLen int
	MinMsgLen uint32
	MaxMsgLen uint32
	msgParser *MsgParser
}

func (server *TCPServer) Start() {
	server.init()
	go server.run()
}

func (server *TCPServer) init() {
	ln, err := net.Listen("tcp", server.Addr)
	if err != nil {
		log.Fatal("tcpserver", "%v", err)
	}

	if server.MaxConnNum <= 0 {
		server.MaxConnNum = 10000
	}
	if server.PendingWriteNum <= 0 {
		server.PendingWriteNum = 10000
	}
	if server.NewAgent == nil || server.GetConfig == nil {
		log.Fatal("tcpserver", "NewAgent or GetConfig must not be nil")
	}

	server.ln = ln
	server.conns = make(ConnSet)
	server.memOverLimit = false
	server.agentId = 0

	// msg parser
	msgParser := NewMsgParser()
	msgParser.SetMsgLen(server.LenMsgLen, server.MinMsgLen, server.MaxMsgLen)
	server.msgParser = msgParser
}

func (server *TCPServer) run() {
	server.wgLn.Add(1)
	defer server.wgLn.Done()

	var tempDelay time.Duration
	for {
		conn, err := server.ln.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				log.Error("TCPServer", "accept error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return
		}
		tempDelay = 0

		server.mutexConns.Lock()
		alloc, curMemory := server.checkMemory()
		if len(server.conns) >= server.MaxConnNum || !alloc {
			server.mutexConns.Unlock()
			conn.Close()
			log.Warning("TCPServer", "超出连接上限,MaxConnNum=%d,alloc=%v,curMemory=%d", server.MaxConnNum, alloc, curMemory)
			continue
		}

		server.conns[conn] = struct{}{}
		server.agentId++
		server.mutexConns.Unlock()

		server.wgConns.Add(1)

		tcpConn := newTCPConn(conn, server.PendingWriteNum, server.msgParser)
		agent := server.NewAgent(tcpConn, server.agentId)
		go func(id uint64) {
			agent.Run()
			// cleanup
			tcpConn.Close()
			server.mutexConns.Lock()
			delete(server.conns, conn)
			server.mutexConns.Unlock()
			agent.OnClose()

			server.wgConns.Done()
		}(server.agentId)
	}
}

func (server *TCPServer) checkMemory() (bool, int64) {
	maxMemory := server.GetConfig("内存限制", 0)
	checkCount := server.GetConfig("开始监控连接数量", 5000)
	if maxMemory <= 0 || len(server.conns) < int(checkCount) {
		return true, 0
	}

	server.rwMemLimit.RLock()
	if server.memOverLimit {
		server.rwMemLimit.RUnlock()
		return false, util.CurMemory()
	}
	server.rwMemLimit.RUnlock()

	watchInterval := server.GetConfig("监控间隔", 1000)
	if (len(server.conns) % int(watchInterval)) == 0 {
		if util.CurMemory() > maxMemory {
			server.rwMemLimit.Lock()
			server.memOverLimit = true
			server.rwMemLimit.Unlock()

			timeInterval := 1 * time.Second
			timer := time.NewTimer(timeInterval)
			go func(t *time.Timer) {
				for {
					<-t.C
					log.Warning("TCPServer", "超标,开始GC,mem=%v", util.CurMemory())
					runtime.GC()
					if util.CurMemory() < (maxMemory * 9 / 10) {
						server.rwMemLimit.Lock()
						server.memOverLimit = false
						server.rwMemLimit.Unlock()
						timer.Stop()
						log.Warning("TCPServer", "恢复,当前,mem=%v", util.CurMemory())
						break
					}
					t.Reset(timeInterval)
				}
			}(timer)

			return false, util.CurMemory()
		}
	}
	return true, 0
}

func (server *TCPServer) Close() {
	server.ln.Close()
	server.wgLn.Wait()

	server.mutexConns.Lock()
	for conn := range server.conns {
		conn.Close()
	}
	server.conns = nil
	server.mutexConns.Unlock()
	server.wgConns.Wait()
}
