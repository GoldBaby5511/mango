package network

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"mango/pkg/util/errorhelper"
	"io"
)

const (
	AppLogger   uint32 = 1
	AppCenter   uint32 = 2
	AppConfig   uint32 = 3
	AppGate     uint32 = 4
	AppLobby    uint32 = 5
	AppProperty uint32 = 6
	AppBattle   uint32 = 7
	AppLogic    uint32 = 8
	AppRobot    uint32 = 9
	AppList     uint32 = 10
	AppTable    uint32 = 11
	AppRoom     uint32 = 12
	AppDaemon   uint32 = 100

	Send2All    uint32 = 1
	Send2AnyOne uint32 = 2

	DataIndex  = 0 //数据
	AgentIndex = 1 //网络代理
	CMDIndex   = 2 //TCPCommon
	OtherIndex = 3 //其他

	MinRouteArgsCount = 3

	// FlagOtherTraceId 消息头内other字段常量
	FlagOtherTraceId = 1
	TraceIdLen       = 16

	msgSizeLen    = 4
	headerSizeLen = 2
	msgHeaderLen  = 6
)

type (
	//网络命令
	TCPCommand struct {
		AppType uint16
		CmdId   uint16
	}

	MessageHeader struct {
		version    uint8
		encrypt    uint8
		TCPCommand        //命令
		Other      []byte // 0xFF字节以内
	}

	// BaseMessage 基础消息结构
	BaseMessage struct {
		MyMessage interface{} //消息体
		AgentInfo BaseAgentInfo
		Cmd       TCPCommand //命令
		TraceId   string     //traceId
	}

	MsgParser struct {
		minMsgLen uint32
		maxMsgLen uint32
	}

	//丢失消息
	MissingMessage struct {
		DestAppType uint32        `bson:"destAppType"`
		DestAppId   uint32        `bson:"destAppId"`
		AgentInfo   BaseAgentInfo `bson:"agentInfo"`
		Cmd         TCPCommand    `bson:"cmd"`
		TraceId     string        `bson:"traceId"`
		Data        []byte        `bson:"data"`
		Time        int64         `bson:"time"`
	}
)

func NewMsgParser() *MsgParser {
	p := new(MsgParser)
	p.minMsgLen = headerSizeLen + msgHeaderLen
	p.maxMsgLen = 16 * 1024

	return p
}

// SetMsgLen It's dangerous to call the method on reading or writing
func (p *MsgParser) SetMsgLen(minMsgLen uint32, maxMsgLen uint32) {
	if minMsgLen != 0 {
		p.minMsgLen = minMsgLen
	}
	if maxMsgLen != 0 {
		p.maxMsgLen = maxMsgLen
	}
}

// |	msgSize	 |	headSize		| 						header 												| msgData
// |4bit(msgSize)| 2bit(headSize) 	| 1bit(version) + 1bit(encrypt) + 2bit(AppType) + 2bit(CmdId) + Xbit(other) | msgData
func (p *MsgParser) Read(conn *TCPConn) (BaseMessage, []byte, error) {
	defer errorhelper.Recover()
	msgSizeBuf := make([]byte, msgSizeLen)
	if _, err := io.ReadFull(conn, msgSizeBuf); err != nil {
		return BaseMessage{}, nil, fmt.Errorf("消息头读取失败,%v", err)
	}

	var msgSize uint32 = 0
	if err := binary.Read(bytes.NewBuffer(msgSizeBuf), binary.BigEndian, &msgSize); err != nil {
		return BaseMessage{}, nil, fmt.Errorf("消息体长度读取失败,%v", err)
	}

	if msgSize < p.minMsgLen || msgSize > p.maxMsgLen {
		return BaseMessage{}, nil, fmt.Errorf("消息长度有问题,msgSize=%v,minMsgLen=%d,maxMsgLen=%d", msgSize, p.minMsgLen, p.maxMsgLen)
	}

	// data
	allData := make([]byte, msgSize)
	if _, err := io.ReadFull(conn, allData); err != nil {
		return BaseMessage{}, nil, fmt.Errorf("消息体内容读取失败,%v", err)
	}

	var headSize uint16 = 0
	_ = binary.Read(bytes.NewBuffer(allData[0:headerSizeLen]), binary.BigEndian, &headSize)
	if headSize > (1 + 1 + 2 + 2 + 0xFF) {
		return BaseMessage{}, nil, fmt.Errorf("消息头长度异常,headSize=%v", headSize)
	}

	header := &MessageHeader{}
	dataBuf := bytes.NewBuffer(allData[headerSizeLen:])
	_ = binary.Read(dataBuf, binary.BigEndian, &header.version)
	_ = binary.Read(dataBuf, binary.BigEndian, &header.encrypt)
	_ = binary.Read(dataBuf, binary.BigEndian, &header.AppType)
	_ = binary.Read(dataBuf, binary.BigEndian, &header.CmdId)

	//获取traceId，不做通用按字节去读，前8个字节是固定的，第9位如果等于1则紧跟在后边的16位就是traceId
	traceId := ""
	if len(allData) >= 8+1+TraceIdLen {
		//获取traceId == 1为具体标识
		var traceIdFlag uint8 = 0
		_ = binary.Read(bytes.NewBuffer(allData[8:8+1]), binary.BigEndian, &traceIdFlag)
		if traceIdFlag == FlagOtherTraceId {
			traceId = string(allData[8+1 : 8+1+TraceIdLen])
		}
	}

	//构造参数
	headCmd := &TCPCommand{AppType: header.AppType, CmdId: header.CmdId}
	msgData := allData[headSize+headerSizeLen:]
	bm := BaseMessage{Cmd: *headCmd, TraceId: traceId}
	return bm, msgData, nil
}

// |	msgSize	 |	headSize		| 						header 												| msgData
// |4bit(msgSize)| 2bit(headSize) 	| 1bit(version) + 1bit(encrypt) + 2bit(AppType) + 2bit(CmdId) + Xbit(other) | msgData
func (p *MsgParser) Write(appType, cmdId uint16, conn *TCPConn, msgData, otherData []byte) error {
	defer errorhelper.Recover()
	var headSize uint16 = 1 + 1 + 2 + 2 + uint16(len(otherData))
	var msgSize uint32 = headerSizeLen + uint32(headSize) + uint32(len(msgData))

	header := MessageHeader{0, 0, TCPCommand{appType, cmdId}, otherData}
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, msgSize)
	_ = binary.Write(buf, binary.BigEndian, headSize)
	_ = binary.Write(buf, binary.BigEndian, header.version)
	_ = binary.Write(buf, binary.BigEndian, header.encrypt)
	_ = binary.Write(buf, binary.BigEndian, header.AppType)
	_ = binary.Write(buf, binary.BigEndian, header.CmdId)
	if len(otherData) > 0 {
		_ = binary.Write(buf, binary.BigEndian, otherData)
	}
	_ = binary.Write(buf, binary.BigEndian, msgData)

	conn.Write(buf.Bytes())

	return nil
}
