package apollo

import (
	"errors"
	"google.golang.org/protobuf/proto"
	"reflect"
	"strconv"
	"sync"
	"xlddz/core/conf"
	"xlddz/core/log"
	"xlddz/core/network"
	"xlddz/protocol/config"
	"xlddz/protocol/router"
)

var (
	configValues map[ConfKey]*ConfValue = make(map[ConfKey]*ConfValue)
	regSubList   map[ConfKey]*ConfValue = make(map[ConfKey]*ConfValue)
	routeAgent   network.Agent          = nil
	apolloNotify []PublicCb
	mutexConfig  sync.Mutex
	mxRegSub     sync.Mutex
)

type PublicCb func(key ConfKey, value ConfValue)

type ConfKey struct {
	AppType uint32
	AppId   uint32
	Key     string
}

type ConfValue struct {
	Value    string
	RspCount uint64
	Cb       PublicCb
}

//设置代理
func SetRouterAgent(a network.Agent) {
	routeAgent = a

	//连接成功后开启定时订阅
	//timeInterval := 30 * time.Second
	//timerHeartbeat := time.NewTimer(timeInterval)
	//go func(t *time.Timer) {
	//	for {
	//		<-t.C
	//
	//		//保持订阅
	//		for key, _ := range regSubList {
	//			SendSubscribeReq(key, false)
	//		}
	//
	//		t.Reset(timeInterval)
	//	}
	//}(timerHeartbeat)
}

//router断开
func RouterDisconnect() {
	//允许重新注册,原配置保留
	regSubList = make(map[ConfKey]*ConfValue)
}

//消息处理
func ProcessReq(dataTransferReq *router.DataTransferReq) error {
	//消息处理
	data := dataTransferReq.GetDataBuff()
	switch dataTransferReq.GetDataCmdsubid() {
	case uint32(config.CMDID_Config_IDApolloCfgRsp): //配置响应
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
		log.Info("apollo", "apollo,还未处理消息,%d", dataTransferReq.GetDataCmdsubid())
	}

	return nil
}

// 读取配置中心的配置，找不到时，返回空字符串
func GetConfig(key, defaultValue string) string {
	nsKey := ConfKey{Key: key, AppType: conf.AppType, AppId: conf.AppID}
	mutexConfig.Lock()
	defer mutexConfig.Unlock()
	if item, ok := configValues[nsKey]; ok {
		return item.Value
	}
	return defaultValue
}

// 读取配置中心的配置，找不到或出错时，返回0
func GetConfigAsInt64(key string, defaultValue int64) int64 {
	v, _ := strconv.ParseInt(GetConfig(key, strconv.FormatInt(defaultValue, 10)), 10, 64)
	return v
}

func RegisterConfig(key string, reqAppType, reqAppId uint32, cb PublicCb) {
	mxRegSub.Lock()
	nsKey := ConfKey{Key: key, AppType: reqAppType, AppId: reqAppId}
	if _, ok := regSubList[nsKey]; ok {
		log.Info("Apollo", "这个key已经注册过了")
		return
	}

	regSubList[nsKey] = &ConfValue{}
	mxRegSub.Unlock()
	log.Info("Apollo", "注册Apollo订阅，%v", nsKey)

	//发起一次订阅
	SendSubscribeReq(nsKey, false)
}

//发送订阅
func SendSubscribeReq(k ConfKey, cancel bool) {
	if routeAgent == nil {
		return
	}
	//没注册过的走你
	mxRegSub.Lock()
	defer mxRegSub.Unlock()
	if _, ok := regSubList[k]; !ok {
		return
	}

	var cfgReq config.ApolloCfgReq
	cfgReq.AppType = proto.Uint32(conf.AppType)
	cfgReq.AppId = proto.Uint32(conf.AppID)
	cfgReq.SubAppType = proto.Uint32(k.AppType)
	cfgReq.SubAppId = proto.Uint32(k.AppId)
	cfgReq.KeyName = proto.String(k.Key)
	Subscribe := config.ApolloCfgReq_SUBSCRIBE
	if regSubList[k].RspCount == 0 {
		Subscribe = Subscribe | config.ApolloCfgReq_NEED_RSP
	}
	if cancel {
		Subscribe = config.ApolloCfgReq_UNSUBSCRIBE
	}
	cfgReq.Subscribe = proto.Uint32(uint32(Subscribe))
	routeAgent.SendData2App(network.AppConfig, network.Send2AnyOne, network.CMDConfig, uint32(config.CMDID_Config_IDApolloCfgReq), &cfgReq)
}

//注册回调
func RegPublicCB(cb PublicCb) {
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
