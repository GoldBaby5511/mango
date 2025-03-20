package json

import (
	"encoding/json"
	"errors"
	"fmt"
	"mango/pkg/chanrpc"
	"mango/pkg/log"
	"reflect"
)

type Processor struct {
	msgInfo map[string]*MsgInfo
}

type MsgInfo struct {
	msgType       reflect.Type
	msgRouter     *chanrpc.Server
	msgHandler    MsgHandler
	msgRawHandler MsgHandler
}

type MsgHandler func([]interface{})

type MsgRaw struct {
	msgID      string
	msgRawData json.RawMessage
}

func NewProcessor() *Processor {
	p := new(Processor)
	p.msgInfo = make(map[string]*MsgInfo)
	return p
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) Register(msg interface{}) string {
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		log.Fatal("json", "json message pointer required")
	}
	msgID := msgType.Elem().Name()
	if msgID == "" {
		log.Fatal("json", "unnamed json message")
	}
	if _, ok := p.msgInfo[msgID]; ok {
		log.Fatal("json", "message %v is already registered", msgID)
	}

	i := new(MsgInfo)
	i.msgType = msgType
	p.msgInfo[msgID] = i
	return msgID
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) SetRouter(msg interface{}, msgRouter *chanrpc.Server) {
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		log.Fatal("json", "json message pointer required")
	}
	msgID := msgType.Elem().Name()
	i, ok := p.msgInfo[msgID]
	if !ok {
		log.Fatal("json", "message %v not registered", msgID)
	}

	i.msgRouter = msgRouter
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) SetHandler(msg interface{}, msgHandler MsgHandler) {
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		log.Fatal("json", "json message pointer required")
	}
	msgID := msgType.Elem().Name()
	i, ok := p.msgInfo[msgID]
	if !ok {
		log.Fatal("json", "message %v not registered", msgID)
	}

	i.msgHandler = msgHandler
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) SetRawHandler(msgID string, msgRawHandler MsgHandler) {
	i, ok := p.msgInfo[msgID]
	if !ok {
		log.Fatal("json", "message %v not registered", msgID)
	}

	i.msgRawHandler = msgRawHandler
}

// goroutine safe
func (p *Processor) Route(userData interface{}, cmd interface{}, msg interface{}) error {
	// raw
	if msgRaw, ok := msg.(MsgRaw); ok {
		i, ok := p.msgInfo[msgRaw.msgID]
		if !ok {
			return fmt.Errorf("message %v not registered", msgRaw.msgID)
		}
		if i.msgRawHandler != nil {
			i.msgRawHandler([]interface{}{msgRaw.msgID, msgRaw.msgRawData, userData})
		}
		return nil
	}

	// json
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		return errors.New("json message pointer required")
	}
	msgID := msgType.Elem().Name()
	i, ok := p.msgInfo[msgID]
	if !ok {
		return fmt.Errorf("message %v not registered", msgID)
	}
	if i.msgHandler != nil {
		i.msgHandler([]interface{}{msg, userData})
	}
	if i.msgRouter != nil {
		i.msgRouter.Go(msgType, msg, userData)
	}
	return nil
}

// goroutine safe
func (p *Processor) Unmarshal(data []byte) (interface{}, interface{}, error) {
	//去除末尾0
	//data = data[0 : len(data)-1]
	//手动解析
	head := make([]byte, 8)
	copy(head, data[0:8])

	//buf := bytes.NewBuffer(head)

	//tcpHead := &network.MessageHeader{}
	//binary.Read(buf, binary.LittleEndian, &tcpHead.DataKind)
	//binary.Read(buf, binary.LittleEndian, &tcpHead.CheckCode)
	//binary.Read(buf, binary.LittleEndian, &tcpHead.PacketSize)
	//binary.Read(buf, binary.LittleEndian, &tcpHead.AppType)
	//binary.Read(buf, binary.LittleEndian, &tcpHead.CmdId)
	//
	////log.Debug("JSON 解析数据% d\n", data)
	//
	//if tcpHead.PacketSize == 8 {
	//	tcpData, err := p.Marshal(&tcpHead)
	//	if err != nil {
	//		log.Error("json", "marshal message %v error: %v", reflect.TypeOf(&network.TCPHead{}), err)
	//		return nil, nil, errors.New("数据包头序列化失败")
	//	}
	//	l := 0
	//	for i := 0; i < len(tcpData); i++ {
	//		copy(data[l:], tcpData[i])
	//		l += len(tcpData[i])
	//	}
	//} else {
	//	data = data[8:]
	//}

	//log.Debug("JSON 解析数据% d\n", data)

	//JSON反序列化
	var m map[string]json.RawMessage
	err := json.Unmarshal(data, &m)
	if err != nil {
		log.Error("json", "JSON反序列化失败")
		return nil, nil, err
	}
	if len(m) != 1 {
		return nil, nil, errors.New("invalid json data")
	}

	for msgID, data := range m {
		i, ok := p.msgInfo[msgID]
		if !ok {
			return nil, nil, fmt.Errorf("message %v not registered", msgID)
		}

		// msg
		if i.msgRawHandler != nil {
			return nil, MsgRaw{msgID, data}, nil
		} else {
			msg := reflect.New(i.msgType.Elem()).Interface()
			return nil, msg, json.Unmarshal(data, msg)
		}
	}

	panic("bug")
}

// goroutine safe
func (p *Processor) Marshal(msg interface{}) ([][]byte, error) {
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		return nil, errors.New("json message pointer required")
	}
	msgID := msgType.Elem().Name()
	if _, ok := p.msgInfo[msgID]; !ok {
		return nil, fmt.Errorf("message %v not registered", msgID)
	}

	// data
	m := map[string]interface{}{msgID: msg}
	data, err := json.Marshal(m)
	return [][]byte{data}, err
}
