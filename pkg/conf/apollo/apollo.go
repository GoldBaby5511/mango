package apollo

import (
	"mango/api/config"
	"mango/pkg/chanrpc"
	"mango/pkg/conf"
	"mango/pkg/log"
	n "mango/pkg/network"
	"strconv"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
)

const (
	StateComplete  uint32 = 0
	StateBeginSend uint32 = 1
	StateSending   uint32 = 2

	DefaultAppTypeEx uint32 = 0 //默认扩展

	KeyIndex   = 0
	ValueIndex = 1

	ConfigChangeNotifyId string = "ConfigChangeNotifyId"
)

var (
	configValues               = make(map[ConfKey]*ConfValue)
	regSubList                 = make(map[ConfKey]*ConfValue)
	netAgent     n.AgentServer = nil
	mutexConfig  sync.Mutex
	mutexRegSub  sync.Mutex
	MsgRouter    *chanrpc.Server = nil
)

type (
	ConfKey struct {
		AppType, AppId uint32
		Key            string
	}

	ConfValue struct {
		Value    string
		RspCount uint64
		state    uint32 //接收状态 0完成、1开始发送、2发送中
	}
)

func SetNetAgent(a n.AgentServer) {
	if a == nil {
		return
	}
	netAgent = a
	for key, _ := range regSubList {
		sendSubscribeReq(key, false)
	}

	time.AfterFunc(1*time.Second, checkConfigRsp)
}

func HandleConfigRsp(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*config.ConfigRsp)
	if len(m.GetItem()) == 0 {
		log.Error("apollo", "异常,收到空的Apollo配置,len=%v,regKey=%v,type=%v,appid=%v",
			len(m.GetItem()), m.GetRegKey(), m.GetSubAppType(), m.GetSubAppId())
		return
	}

	key := ConfKey{Key: m.GetRegKey(), AppType: m.GetSubAppType(), AppId: m.GetSubAppId()}
	mutexRegSub.Lock()
	if _, ok := regSubList[key]; !ok {
		mutexRegSub.Unlock()
		log.Warning("apollo", "异常，返回的竟然是自己没订阅的,key=%v", key)
		return
	}
	regSubList[key].RspCount += 1
	mutexRegSub.Unlock()

	for _, v := range m.GetItem() {
		key.Key = v.GetKey()
		//
		switch key.Key {
		case "LogScreenPrint":
			p, _ := strconv.Atoi(v.GetValue())
			log.SetScreenPrint(p)
		case "LogMinLevel":
			log.MinLevel, _ = strconv.Atoi(v.GetValue())
		default:
			break
		}

		log.Trace("apollo", "收到配置,regKey=%v,subType=%v,subId=%v,k=%v,len=%v",
			m.GetRegKey(), m.GetSubAppType(), m.GetSubAppId(), v.GetKey(), len(v.GetValue()))

		mutexConfig.Lock()
		if _, ok := configValues[key]; ok {
			if configValues[key].state != StateComplete {
				configValues[key].Value += v.GetValue()
			} else {
				configValues[key].Value = v.GetValue()
				configValues[key].RspCount += 1
			}
		} else {
			configValues[key] = &ConfValue{Value: v.GetValue(), RspCount: 1, state: StateComplete}
		}
		mutexConfig.Unlock()

		if _, ok := configValues[key]; ok && configValues[key].state == StateComplete {
			changeNotify(key)
		}
	}
}

func HandleItemRspState(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*config.ItemRspState)
	key := ConfKey{Key: m.GetKey(), AppType: m.GetSubAppType(), AppId: m.GetSubAppId()}
	mutexConfig.Lock()
	if _, ok := configValues[key]; ok {
		configValues[key].state = m.GetState()
		configValues[key].RspCount += 1
		if configValues[key].state == StateBeginSend {
			configValues[key].Value = ""
		}
	} else {
		//大字段数据先增加映射
		if m.GetState() == StateBeginSend {
			configValues[key] = &ConfValue{Value: "", RspCount: 1, state: m.GetState()}
		}
	}
	mutexConfig.Unlock()
	log.Trace("apollo", "接收状态,regKey=%v,state=%v", key, m.GetState())
	if _, ok := configValues[key]; ok && configValues[key].state == StateComplete {
		changeNotify(key)
	}
}

func GetConfig(key, defaultValue string) string {
	return configValue(ConfKey{Key: key, AppType: conf.AppInfo.Type, AppId: conf.AppInfo.Id}, defaultValue)
}

// GetConfigEx 获取扩展配置
func GetConfigEx(appType, appId uint32, key, defaultValue string) string {
	return configValue(ConfKey{Key: key, AppType: appType, AppId: appId}, defaultValue)
}

func configValue(k ConfKey, defaultValue string) string {
	mutexConfig.Lock()
	defer mutexConfig.Unlock()
	if item, ok := configValues[k]; ok {
		if item.state == StateComplete {
			return item.Value
		} else {
			return defaultValue
		}
	}
	return defaultValue
}

func GetConfigAsInt64(key string, defaultValue int64) int64 {
	v, _ := strconv.ParseInt(GetConfig(key, strconv.FormatInt(defaultValue, 10)), 10, 64)
	return v
}

func GetConfigAsFloat64(key string, defaultValue int64) float64 {
	v, _ := strconv.ParseFloat(GetConfig(key, strconv.FormatInt(defaultValue, 10)), 64)
	return v
}

func GetConfigAsFloat32(key string, defaultValue int64) float32 {
	return float32(GetConfigAsFloat64(key, defaultValue))
}

func changeNotify(k ConfKey) {
	if _, ok := configValues[k]; !ok {
		return
	}

	if MsgRouter == nil {
		return
	}

	MsgRouter.Go(ConfigChangeNotifyId, k, *configValues[k])
}

func checkConfigRsp() {
	mutexRegSub.Lock()
	if len(regSubList) == 0 {
		mutexRegSub.Unlock()
		return
	}
	totalRspCount := uint64(0)
	for _, v := range regSubList {
		totalRspCount += v.RspCount
	}
	mutexRegSub.Unlock()

	log.Info("", "配置回复检查,c=%v", totalRspCount)

	if totalRspCount == 0 {
		time.AfterFunc(1*time.Second, checkConfigRsp)
	}
}

func RegisterConfig(key string, reqAppType, reqAppId uint32) {
	regKey := ConfKey{Key: key, AppType: reqAppType, AppId: reqAppId}
	mutexRegSub.Lock()
	if _, ok := regSubList[regKey]; ok {
		mutexRegSub.Unlock()
		log.Debug("Apollo", "这个key已经注册过了,key=%v", regKey)
		return
	}
	regSubList[regKey] = &ConfValue{}
	mutexRegSub.Unlock()

	log.Info("Apollo", "注册Apollo订阅,regKey=%v", regKey)

	sendSubscribeReq(regKey, false)
}

func sendSubscribeReq(k ConfKey, cancel bool) {
	if netAgent == nil {
		return
	}
	mutexRegSub.Lock()
	defer mutexRegSub.Unlock()
	if _, ok := regSubList[k]; !ok {
		return
	}

	var req config.ConfigReq
	req.AppType = proto.Uint32(conf.AppInfo.Type)
	req.AppId = proto.Uint32(conf.AppInfo.Id)
	req.SubAppType = proto.Uint32(k.AppType)
	req.SubAppId = proto.Uint32(k.AppId)
	req.Key = proto.String(k.Key)
	subscribe := config.ConfigReq_SUBSCRIBE
	if regSubList[k].RspCount == 0 {
		subscribe = subscribe | config.ConfigReq_NEED_RSP
	}
	if cancel {
		subscribe = config.ConfigReq_UNSUBSCRIBE
	}
	req.Subscribe = proto.Uint32(uint32(subscribe))

	log.Info("Apollo", "发送订阅,k=%v,subscribe=%v", k, subscribe)

	cmd := n.TCPCommand{AppType: uint16(n.AppConfig), CmdId: uint16(config.CMDConfig_IDConfigReq)}
	bm := n.BaseMessage{MyMessage: &req, Cmd: cmd}
	netAgent.SendMessage(bm)
}
