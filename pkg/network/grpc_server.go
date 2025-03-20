package network

import (
	"mango/pkg/log"
	"mango/pkg/util/errorhelper"
	"fmt"
	"google.golang.org/grpc"
	"net"
)

type RpcServer struct {
	Port     int
	isStart  bool
	services []func(gs *grpc.Server)
}

func NewRpcServer() *RpcServer {
	rs := new(RpcServer)
	rs.isStart = false
	return rs
}

func (rs *RpcServer) AddService(service func(gs *grpc.Server)) {
	rs.services = append(rs.services, service)
}

func (rs *RpcServer) IsStart() bool {
	return rs.isStart
}

func (rs *RpcServer) Start() {
	if rs.Port == 0 || rs.IsStart() {
		return
	}
	go rs.run()
}

func (rs *RpcServer) run() {
	defer errorhelper.Recover()

	//建立连接
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", rs.Port))
	if err != nil {
		log.Error("", "异常,rpc服务启动失败,port=%v,err=%v", rs.Port, err)
		return
	}

	//创建服务
	s := grpc.NewServer()
	for _, f := range rs.services {
		f(s)
	}

	rs.isStart = true

	log.Info("", fmt.Sprintf("listening rpc on %d,len=%v", rs.Port, len(rs.services)))

	// 运行服务
	if err := s.Serve(ln); err != nil {
		log.Error("", "failed to serve: err=%v", err)
		return
	}
}
