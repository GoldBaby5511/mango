package network

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"xlddz/core/log"
)

//appid类型
const (
	Send2All    uint32 = 1 //发送到所有
	Send2AnyOne uint32 = 2 //发送到随意任意一个
)

//消息类型
const (
	NullType          uint32 = 0  //空类型l
	CMDServiceManager uint32 = 3  //服务管理
	CMDLogger         uint32 = 8  //日志
	CMDRouter         uint32 = 10 //router
	CMDAppFrame       uint32 = 11
	CMDGate           uint32 = 12 //gate
	CMDClient         uint32 = 13
)

//apptype类型
const (
	AppGate            uint32 = 5
	AppRouter          uint32 = 6
	AppLogin           uint32 = 7
	AppOnline          uint32 = 8
	AppApollo          uint32 = 56
	AppMatchSVCManager uint32 = 58
	AppActionData      uint32 = 73
)

//回调参数列表下标
const (
	DATA_INDEX  = 0 //数据
	AGENT_INDEX = 1 //网络代理
	CMD_INDEX   = 2 //TCPCommon
	OTHER_INDEX = 3 //其他
)

//默认网络消息rpc处理Id
const DefaultNetMsgFuncId string = "DefaultNetMsgFuncId"

//网络命令
type TCPCommand struct {
	MainCmdID uint16
	SubCmdID  uint16
}

//消息头内other字段常量
const (
	FlagOtherTraceId = 1
	TraceIdLen       = 16
)

// PackageHeader 比赛架构下的消息头
type PackageHeader struct {
	version uint8
	encrypt uint8
	CmdKind uint16
	CmdId   uint16
	Other   string // 0xFF字节以内
}

// BaseMessage 基础消息结构
type BaseMessage struct {
	MyMessage interface{} //消息体
	Cmd       TCPCommand  //命令
	TraceId   string      //traceId
}

// MsgParser --------------
// | msgSize | headSize | header | msgData |
// | 4bit(msgSize) | 2bit(headSize) | 1bit(version) + 1bit(encrypt) + 2bit(CmdKind) + 2bit(CmdId) + Xbit(other) | msgData
// --------------
type MsgParser struct {
	lenMsgLen    int
	minMsgLen    uint32
	maxMsgLen    uint32
	littleEndian bool
}

func NewMsgParser() *MsgParser {
	p := new(MsgParser)
	p.lenMsgLen = 2
	p.minMsgLen = 8
	p.maxMsgLen = 4096
	p.littleEndian = false

	return p
}

// It's dangerous to call the method on reading or writing
func (p *MsgParser) SetMsgLen(lenMsgLen int, minMsgLen uint32, maxMsgLen uint32) {
	if lenMsgLen == 1 || lenMsgLen == 2 || lenMsgLen == 4 {
		p.lenMsgLen = lenMsgLen
	}
	if minMsgLen != 0 {
		p.minMsgLen = minMsgLen
	}
	if maxMsgLen != 0 {
		p.maxMsgLen = maxMsgLen
	}

	var max uint32
	switch p.lenMsgLen {
	case 1:
		max = math.MaxUint8
	case 2:
		max = math.MaxUint16
	case 4:
		max = math.MaxUint32
	}
	if p.minMsgLen > max {
		p.minMsgLen = max
	}
	if p.maxMsgLen > max {
		p.maxMsgLen = max
	}
}

// It's dangerous to call the method on reading or writing
func (p *MsgParser) SetByteOrder(littleEndian bool) {
	p.littleEndian = littleEndian
}

// |	msgSize	 |	headSize		| 						header 												| msgData
// |4bit(msgSize)| 2bit(headSize) 	| 1bit(version) + 1bit(encrypt) + 2bit(CmdKind) + 2bit(CmdId) + Xbit(other) | msgData
func (p *MsgParser) Read(conn *TCPConn) (BaseMessage, []byte, error) {
	//读取消息头
	msgSizeBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, msgSizeBuf); err != nil {
		log.Warning("MsgParser", "消息头读取失败,%v", err)
		return BaseMessage{}, nil, err
	}

	var msgSize uint32 = 0
	if err := binary.Read(bytes.NewBuffer(msgSizeBuf), binary.BigEndian, &msgSize); err != nil {
		log.Error("MsgParser", "消息体长度读取失败,%v", err)
		return BaseMessage{}, nil, err
	}

	// data
	allData := make([]byte, msgSize)
	if _, err := io.ReadFull(conn, allData); err != nil {
		log.Error("MsgParser", "消息体内容读取失败,%v", err)
		return BaseMessage{}, nil, err
	}

	var headSize uint16 = 0
	_ = binary.Read(bytes.NewBuffer(allData[0:2]), binary.BigEndian, &headSize)

	header := &PackageHeader{}
	dataBuf := bytes.NewBuffer(allData[2:])
	_ = binary.Read(dataBuf, binary.BigEndian, &header.version)
	_ = binary.Read(dataBuf, binary.BigEndian, &header.encrypt)
	_ = binary.Read(dataBuf, binary.BigEndian, &header.CmdKind)
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
	headCmd := &TCPCommand{MainCmdID: header.CmdKind, SubCmdID: header.CmdId}
	msgData := allData[headSize+2:]
	bm := BaseMessage{Cmd: *headCmd, TraceId: traceId}
	return bm, msgData, nil
}

// |	msgSize	 |	headSize		| 						header 												| msgData
// |4bit(msgSize)| 2bit(headSize) 	| 1bit(version) + 1bit(encrypt) + 2bit(CmdKind) + 2bit(CmdId) + Xbit(other) | msgData
func (p *MsgParser) Write(mainCmdID, subCmdID uint16, conn *TCPConn, msgData, otherData []byte) error {
	var headSize uint16 = 1 + 1 + 2 + 2 + uint16(len(otherData))
	var msgSize uint32 = 2 + uint32(headSize) + uint32(len(msgData))

	header := PackageHeader{uint8(99), uint8(104), mainCmdID, subCmdID, ""}
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, msgSize)
	_ = binary.Write(buf, binary.BigEndian, headSize)
	_ = binary.Write(buf, binary.BigEndian, header.version)
	_ = binary.Write(buf, binary.BigEndian, header.encrypt)
	_ = binary.Write(buf, binary.BigEndian, header.CmdKind)
	_ = binary.Write(buf, binary.BigEndian, header.CmdId)
	if len(otherData) > 0 {
		_ = binary.Write(buf, binary.BigEndian, otherData)
	}
	_ = binary.Write(buf, binary.BigEndian, msgData)

	conn.Write(buf.Bytes())

	return nil
}
