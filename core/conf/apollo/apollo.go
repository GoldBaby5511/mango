package apollo

import (
	"errors"
	"github.com/golang/protobuf/proto"
	"reflect"
	"strconv"
	"sync"
	"time"
	"xlddz/core/conf"
	"xlddz/core/log"
	"xlddz/core/network"
	"xlddz/protocol/config"
)

var (
	configValues map[ConfKey]*ConfValue = make(map[ConfKey]*ConfValue)
	regSubList   map[ConfKey]*ConfValue = make(map[ConfKey]*ConfValue)
	netAgent     network.AgentServer    = nil
	apolloNotify []cbNotify
	mutexConfig  sync.Mutex
	mxRegSub     sync.Mutex
)

type cbNotify func(key ConfKey, value ConfValue)

type ConfKey struct {
	AppType uint32
	AppId   uint32
	Key     string
}

type ConfValue struct {
	Value    string
	RspCount uint64
	Cb       cbNotify
}

func init() {
	log.GetMinLevelConfig = GetConfigAsInt64
}

//设置代理
func SetNetAgent(a network.AgentServer) {
	netAgent = a

	//连接成功后开启定时订阅
	timeInterval := 30 * time.Second
	timer := time.NewTimer(timeInterval)
	go func(t *time.Timer) {
		for {
			<-t.C

			//保持订阅
			for key, _ := range regSubList {
				SendSubscribeReq(key, false)
			}

			t.Reset(timeInterval)
		}
	}(timer)
}

func CenterDisconnect() {
	regSubList = make(map[ConfKey]*ConfValue)
}

//消息处理
func ProcessReq(cmd *network.TCPCommand, data []byte) error {
	switch cmd.SubCmdID {
	case uint16(config.CMDID_Config_IDApolloCfgRsp): //配置响应
		var m config.ApolloCfgRsp
		proto.Unmarshal(data, &m)

		//不接受空内容，毫无意义
		if len(m.GetKey()) == 0 || (len(m.GetKey()) != len(m.GetValue())) {
			log.Error("apollo", "异常,收到空的Apollo配置,PacketId=%v,ns=%v,key=%v,type=%v,appid=%v",
				m.GetPacketId(), m.GetNameSpace(), m.GetKeyName(), m.GetSubAppType(), m.GetSubAppId())
			return errors.New("异常，异常,k,v不相等")
		}

		nsKey := ConfKey{Key: m.GetKeyName(), AppType: m.GetSubAppType(), AppId: m.GetSubAppId()}
		mxRegSub.Lock()
		if _, ok := regSubList[nsKey]; !ok {
			mxRegSub.Unlock()
			log.Error("apollo", "异常，返回的竟然是自己没订阅的")
			return errors.New("异常，返回的竟然是自己没订阅的")
		}
		regSubList[nsKey].RspCount += 1
		mxRegSub.Unlock()

		for i := 0; i < len(m.GetKey()); i++ {
			key := ConfKey{Key: m.GetKey()[i], AppType: m.GetSubAppType(), AppId: m.GetSubAppId()}
			if m.GetKey()[i] == "LogScreenPrint" {
				p, _ := strconv.Atoi(m.GetValue()[i])
				log.SetScreenPrint(p)
			}
			mutexConfig.Lock()
			if _, ok := configValues[key]; ok {
				configValues[key].Value = m.GetValue()[i]
				configValues[key].RspCount += 1
			} else {
				configValues[key] = &ConfValue{Value: m.GetValue()[i], RspCount: 1}
			}
			cbValue := *configValues[key]
			mutexConfig.Unlock()
			//获取回调
			for _, cb := range apolloNotify {
				cb(key, cbValue)
			}
		}
	default:
		log.Warning("apollo", "apollo,还未处理消息,%v", cmd)
	}

	return nil
}

func GetConfig(key, defaultValue string) string {
	nsKey := ConfKey{Key: key, AppType: conf.AppInfo.AppType, AppId: conf.AppInfo.AppID}
	mutexConfig.Lock()
	defer mutexConfig.Unlock()
	if item, ok := configValues[nsKey]; ok {
		return item.Value
	}
	return defaultValue
}

func GetConfigAsInt64(key string, defaultValue int64) int64 {
	v, _ := strconv.ParseInt(GetConfig(key, strconv.FormatInt(defaultValue, 10)), 10, 64)
	return v
}

func RegisterConfig(key string, reqAppType, reqAppId uint32, cb cbNotify) {
	mxRegSub.Lock()
	nsKey := ConfKey{Key: key, AppType: reqAppType, AppId: reqAppId}
	if _, ok := regSubList[nsKey]; ok {
		log.Info("Apollo", "这个key已经注册过了")
		return
	}

	regSubList[nsKey] = &ConfValue{Cb: cb}
	mxRegSub.Unlock()
	log.Info("Apollo", "注册Apollo订阅，%v", nsKey)

	SendSubscribeReq(nsKey, false)
}

func SendSubscribeReq(k ConfKey, cancel bool) {
	if netAgent == nil {
		return
	}
	//没注册过的走你
	mxRegSub.Lock()
	defer mxRegSub.Unlock()
	if _, ok := regSubList[k]; !ok {
		return
	}

	var req config.ApolloCfgReq
	req.AppType = proto.Uint32(conf.AppInfo.AppType)
	req.AppId = proto.Uint32(conf.AppInfo.AppID)
	req.SubAppType = proto.Uint32(k.AppType)
	req.SubAppId = proto.Uint32(k.AppId)
	req.KeyName = proto.String(k.Key)
	Subscribe := config.ApolloCfgReq_SUBSCRIBE
	if regSubList[k].RspCount == 0 {
		Subscribe = Subscribe | config.ApolloCfgReq_NEED_RSP
	}
	if cancel {
		Subscribe = config.ApolloCfgReq_UNSUBSCRIBE
	}
	req.Subscribe = proto.Uint32(uint32(Subscribe))

	cmd := network.TCPCommand{MainCmdID: uint16(network.AppConfig), SubCmdID: uint16(config.CMDID_Config_IDApolloCfgReq)}
	bm := network.BaseMessage{MyMessage: &req, Cmd: cmd}
	netAgent.SendMessage(bm)
}

func RegPublicCB(cb cbNotify) {
	if cb == nil {
		return
	}
	//重复判断
	regPointer := reflect.ValueOf(cb).Pointer()
	for i := 0; i < len(apolloNotify); i++ {
		if reflect.ValueOf(apolloNotify[i]).Pointer() == regPointer {
			return
		}
	}
	apolloNotify = append(apolloNotify, cb)
}
