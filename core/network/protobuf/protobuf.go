package protobuf

import (
	"errors"
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
	littleEndian     bool
	msgInfo          map[network.TCPCommand]*MsgInfo
	msgID            map[reflect.Type]network.TCPCommand
	defaultMsgRouter *chanrpc.Server
}

type MsgInfo struct {
	msgType     reflect.Type
	msgRouter   *chanrpc.Server
	msgCallBack func(args []interface{})
}

func NewProcessor() *Processor {
	p := new(Processor)
	p.littleEndian = false
	p.msgInfo = make(map[network.TCPCommand]*MsgInfo)
	p.msgID = make(map[reflect.Type]network.TCPCommand)
	p.defaultMsgRouter = nil

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

// 同步回调
func (p *Processor) RegHandle(msg proto.Message, mainCmdID uint32, subCmdID uint16, msgCallBack func(args []interface{})) {
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
	i.msgCallBack = msgCallBack
	p.msgInfo[command] = i
}

//设置默认路由
func (p *Processor) SetDefaultRouter(msgRouter *chanrpc.Server) {
	p.defaultMsgRouter = msgRouter
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
	//最少三个参数
	if len(args) < 3 {
		return fmt.Errorf("路由消息参数过少,%v", len(args))
	}

	//注册处理
	id := *args[network.CMD_INDEX].(*network.TCPCommand)
	i, ok := p.msgInfo[id]
	if ok {
		//使用rpc异步投递
		if i.msgRouter != nil {
			i.msgRouter.Go(i.msgType, args...)
			return nil
		}

		//使用回调同步回调
		if i.msgCallBack != nil {
			i.msgCallBack(args)
			return nil
		}
	}

	//默认处理
	if p.defaultMsgRouter != nil {
		p.defaultMsgRouter.Go(network.DefaultNetMsgFuncId, args...)
		return nil
	}

	log.Error("proto", "protobuf.go Route nil,id=%v ", id)
	return errors.New("异常,protobuf.go Route nil")
}

// goroutine safe
func (p *Processor) Unmarshal(mainCmdID uint16, subCmdID uint16, data []byte) (interface{}, interface{}, error) {

	id := network.TCPCommand{MainCmdID: mainCmdID, SubCmdID: subCmdID}

	//是否为注册消息
	if _, ok := p.msgInfo[id]; !ok {
		log.Error("proto", "protobuf Unmarshal木有找到ID=%v", id)
		errInfo := fmt.Sprintf("解析时木找到注册id=%v", id)
		return &id, nil, errors.New(errInfo)
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
