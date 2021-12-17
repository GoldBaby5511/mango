package protobuf

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"math"
	"reflect"
	"xlddz/core/chanrpc"
	"xlddz/core/log"
	"xlddz/core/network"
)

// -------------------------
// | id | protobuf message |
// -------------------------
type Processor struct {
	littleEndian bool
	msgInfo      map[network.TCPCommand]*MsgInfo
	msgID        map[reflect.Type]network.TCPCommand
}

type MsgInfo struct {
	msgType   reflect.Type
	msgRouter *chanrpc.Server
}

func NewProcessor() *Processor {
	p := new(Processor)
	p.littleEndian = false
	p.msgInfo = make(map[network.TCPCommand]*MsgInfo)
	p.msgID = make(map[reflect.Type]network.TCPCommand)

	return p
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) SetByteOrder(littleEndian bool) {
	p.littleEndian = littleEndian
}

//异步rpc
func (p *Processor) Register(msg proto.Message, mainCmdID uint32, subCmdID uint16, msgRouter *chanrpc.Server) {
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		log.Fatal("proto", "protobuf message pointer required")
	}
	if len(p.msgInfo) >= math.MaxUint16 {
		log.Fatal("proto", "too many protobuf messages (max = %v)", math.MaxUint16)
	}

	//协议命令
	command := network.TCPCommand{MainCmdID: uint16(mainCmdID), SubCmdID: subCmdID}
	if _, ok := p.msgInfo[command]; ok {
		log.Fatal("proto", "message %s,cmd=%v is already registered", msgType, command)
	}

	i := new(MsgInfo)
	i.msgType = msgType
	i.msgRouter = msgRouter
	p.msgInfo[command] = i
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) SetRouter(id network.TCPCommand, msgRouter *chanrpc.Server) {
	_, ok := p.msgInfo[id]
	if !ok {
		log.Fatal("proto", "message %v not registered", id)
	}

	p.msgInfo[id].msgRouter = msgRouter
}

// goroutine safe
func (p *Processor) Route(args ...interface{}) error {
	if len(args) < network.MinRouteArgsCount {
		return fmt.Errorf("路由消息参数过少,MinRouteArgsCount=%v,len(args)=%v", len(args), network.MinRouteArgsCount)
	}
	//注册处理
	id := *args[network.CMDIndex].(*network.TCPCommand)
	i, ok := p.msgInfo[id]
	if ok && i.msgRouter != nil {
		i.msgRouter.Go(i.msgType, args...)
		return nil
	}
	return fmt.Errorf("异常,protobuf.go Route nil,ok=%v,id=%v", ok, id)
}

// goroutine safe
func (p *Processor) Unmarshal(mainCmdID, subCmdID uint16, data []byte) (interface{}, interface{}, error) {
	id := network.TCPCommand{MainCmdID: mainCmdID, SubCmdID: subCmdID}
	if _, ok := p.msgInfo[id]; !ok {
		return &id, nil, fmt.Errorf("protobuf Unmarshal木有找到ID=%v", id)
	}

	// msg
	i := p.msgInfo[id]
	msg := reflect.New(i.msgType.Elem()).Interface()
	return &id, msg, proto.UnmarshalMerge(data, msg.(proto.Message))
}

// goroutine safe
func (p *Processor) Marshal(msg interface{}) ([][]byte, error) {
	msgType := reflect.TypeOf(msg)

	// id
	_, ok := p.msgID[msgType]
	if !ok {
		err := fmt.Errorf("message %s not registered", msgType)
		return nil, err
	}

	// data
	data, err := proto.Marshal(msg.(proto.Message))
	return [][]byte{data}, err
}

// goroutine safe
func (p *Processor) Range(f func(id network.TCPCommand, t reflect.Type)) {
	for id, i := range p.msgInfo {
		f(network.TCPCommand(id), i.msgType)
	}
}
