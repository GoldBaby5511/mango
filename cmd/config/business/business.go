package business

import (
	"encoding/json"
	"fmt"
	"github.com/GoldBaby5511/go-simplejson"
	"github.com/apolloconfig/agollo/v4"
	aConfig "github.com/apolloconfig/agollo/v4/env/config"
	"github.com/apolloconfig/agollo/v4/storage"
	"io/ioutil"
	"mango/api/config"
	"mango/cmd/config/conf"
	mconf "mango/pkg/conf"
	"mango/pkg/conf/apollo"
	g "mango/pkg/gate"
	"mango/pkg/log"
	n "mango/pkg/network"
	"mango/pkg/timer"
	"mango/pkg/util"
	"path"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/golang/protobuf/proto"
)

var (
	listenerList   = make(map[uint64]*configChangeListener)
	needNotifyList = make(map[uint64]*configChangeListener)
)

func init() {
	g.MsgRegister(&config.ConfigReq{}, n.AppConfig, uint16(config.CMDConfig_IDConfigReq), handleApolloCfgReq)
	g.EventRegister(g.ConnectSuccess, connectSuccess)
	g.EventRegister(g.Disconnect, disconnect)
	g.EventRegister(g.CenterRegResult, centerRegResult)
	g.EventRegister(g.CommonServerReg, commonServerReg)
	g.CallBackRegister(g.CbBeforeServiceStart, beforeServiceStart)

	//对自身配置创建监听
	watchMyself()
}

func connectSuccess(args []interface{}) {
	log.Info("连接", "来了老弟,参数数量=%d", len(args))
}

func disconnect(args []interface{}) {
	log.Info("连接", "告辞中,参数数量=%d", len(args))
}

func centerRegResult(args []interface{}) {
	//TODO 下面这段真的丑
	r := args[0].(uint32)
	centerId := args[1].(uint32)
	regAppInfo := args[2].(mconf.BaseInfo)
	if r == 0 && regAppInfo.Id == mconf.AppInfo.Id {
		l := getListener(n.AppCenter, centerId)
		if l == nil {
			log.Warning("配置", "center配置不存在,l=%v,appType=%v,centerId=%v",
				l, n.AppCenter, centerId)
			return
		}

		l.addSubscriptionItem(n.AppCenter, centerId, n.AppCenter, centerId, "")
		l.notifySubscriptionList("", n.AppCenter, centerId)
	}
}

func commonServerReg(args []interface{}) {
	a := args[g.AgentIndex].(n.AgentClient)
	aInfo := a.AgentInfo()
	log.Info("", "来注册了,name=%v,type=%v,Id=%v", aInfo.AppName, aInfo.AppType, aInfo.AppId)

	sourceKey := util.MakeUint64FromUint32(aInfo.AppType, aInfo.AppId)
	if _, ok := needNotifyList[sourceKey]; ok {
		l := needNotifyList[sourceKey]
		if _, ok := l.subscriptionList[sourceKey]; ok {
			log.Info("", "待到注册时候通知,key=%v,type=%v,Id=%v", l.subscriptionList[sourceKey].Key, aInfo.AppType, aInfo.AppId)
			l.notifySubscriptionList(l.subscriptionList[sourceKey].Key, aInfo.AppType, aInfo.AppId)
		} else {
			log.Error("", "异常,需要通知但找不到注册?,type=%v,Id=%v", aInfo.AppType, aInfo.AppId)
		}
		delete(needNotifyList, sourceKey)
	}
}

func beforeServiceStart(args []interface{}) {
	//是否使用Apollo
	if conf.Server.UseApollo {
		c := &conf.Server.Config
		listenerConf := conf.ApolloConfig{ServerType: mconf.AppInfo.Type, ServerId: mconf.AppInfo.Id}
		listener := newListener(c, listenerConf)
		agollo.SetLogger(&DefaultLogger{})
		client, _ := agollo.StartWithConfig(func() (*aConfig.AppConfig, error) {
			return c, nil
		})
		client.AddChangeListener(listener)

		//Use your apollo key to test
		cache := client.GetConfigCache(c.NamespaceName)
		cache.Range(func(key, value interface{}) bool {
			listener.configs[key.(string)] = value.(string)

			switch key.(string) {
			case "common-server", "room", "table":
				var s []conf.ApolloConfig
				err := json.Unmarshal([]byte(value.(string)), &s)
				if err != nil {
					log.Fatal("初始化", "配置解析失败,err=%v", err)
				}
				for _, v := range s {
					cs := &aConfig.AppConfig{
						AppID:          v.Appid,
						Cluster:        v.Cluster,
						IP:             v.Ip,
						NamespaceName:  v.Ns,
						IsBackupConfig: true,
					}

					serverClient, _ := agollo.StartWithConfig(func() (*aConfig.AppConfig, error) {
						return cs, nil
					})

					l := newListener(c, v)
					serverClient.AddChangeListener(l)
					serverCache := serverClient.GetConfigCache(cs.NamespaceName)
					if serverCache != nil {
						serverCache.Range(func(key, value interface{}) bool {
							l.configs[key.(string)] = value.(string)
							return true
						})
						key := util.MakeUint64FromUint32(v.ServerType, v.ServerId)
						listenerList[key] = l
						log.Debug("创建", "apollo模式创建,appType=%v,appId=%v,len=%v",
							l.appType, l.appId, len(listenerList))
					} else {
						serverClient.RemoveChangeListener(l)
						log.Fatal("创建", "必须处理,服务监听创建失败,Appid=%v,Cluster=%v,Ns=%v,Ip=%v,ServerType=%v,ServerId=%v,",
							v.Appid, v.Cluster, v.Ns, v.Ip, v.ServerType, v.ServerId)
					}
				}
			case "日志服务器地址":
				g.ConnectLogServer(value.(string))
			default:
				log.Warning("", "没有处理的,k=%v,v=%v", key, value)
			}
			return true
		})
	} else {
		if conf.Server.LoggerAddr != "" {
			g.ConnectLogServer(conf.Server.LoggerAddr)
		}
		createChangeListener()
	}
}

func watchMyself() {
	changeChan := make(chan struct{})
	_, err := watchFile(conf.DefaultConfigFile, changeChan)
	if err != nil {
		log.Fatal("", "异常,创建自身监听失败!!!")
		return
	}

	go func() {
		for {
			<-changeChan

			loadDefaultConfig := func() error {
				fileData, err := ioutil.ReadFile(conf.DefaultConfigFile)
				if err != nil {
					log.Warning("Listener", "自身变化文件失败,fileName=%v,err=%v", conf.DefaultConfigFile, err)
					return err
				}

				cfg, err := simplejson.NewJson(fileData)
				if err != nil {
					log.Warning("", "自身变化创建JSON失败,err=%v", err)
					return err
				}

				data, err := cfg.Get("CommonServers").MarshalJSON()
				if err != nil {
					log.Warning("", "序列化CommonServers失败,err=%v", err)
					return err
				}

				commonServers := make([]conf.ApolloConfig, 0)
				err = json.Unmarshal(data, &commonServers)
				if err != nil {
					log.Warning("", "反序列化CommonServers失败,err=%v", err)
					return err
				}
				conf.Server.CommonServers = commonServers
				log.Debug("监听", "自身发生变化,len=%v", len(commonServers))
				//暂时只处理新增,可以发生变化即视为新增
				createChangeListener()
				return nil
			}

			if loadDefaultConfig() != nil {
				//重新加载一次,再一次应该就够了,不够再说
				g.Skeleton.AfterFunc(2*time.Second, func() {
					loadDefaultConfig()
				})
			}
		}
	}()
}

func watchFile(fileName string, changeChan chan struct{}) (*fsnotify.Watcher, error) {
	//创建一个监控对象
	fileWatch, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("", "err=%v", err)
		return nil, err
	}

	//添加要监控的对象，文件或文件夹
	err = fileWatch.Add(fileName)
	if err != nil {
		log.Error("", "err=%v", err)
		return nil, err
	}
	//我们另启一个goroutine来处理监控对象的事件
	go func() {
		for {
			select {
			case ev := <-fileWatch.Events:
				{
					//判断事件发生的类型，如下5种
					// Create 创建
					// Write 写入
					// Remove 删除
					// Rename 重命名
					// Chmod 修改权限
					if ev.Op&fsnotify.Create == fsnotify.Create {
						log.Debug("", "创建文件 : %v", ev.Name)
					}
					if ev.Op&fsnotify.Write == fsnotify.Write {
						log.Debug("", "写入文件,ev.Name=%v,file=%v", ev.Name, fileName)
						changeChan <- struct{}{}
					}
					if ev.Op&fsnotify.Remove == fsnotify.Remove {
						log.Debug("", "删除文件 : %v", ev.Name)
					}
					if ev.Op&fsnotify.Rename == fsnotify.Rename {
						log.Debug("", "重命名文件 : %v", ev.Name)
					}
					if ev.Op&fsnotify.Chmod == fsnotify.Chmod {
						log.Debug("", "修改权限 : %v", ev.Name)
					}
				}
			case err := <-fileWatch.Errors:
				{
					if err != nil {
						log.Error("", "fileWatch.Errors,err=%v", err)
					}
				}
			}
		}
	}()
	return fileWatch, nil
}

func createChangeListener() {
	for _, c := range conf.Server.CommonServers {
		key := util.MakeUint64FromUint32(c.ServerType, c.ServerId)
		if _, ok := listenerList[key]; ok {
			continue
		}

		l := newListener(nil, c)
		err := l.loadConfigFile()
		if err != nil {
			continue
		}
		l.fileWatch, err = watchFile(l.fileName, l.changeChan)
		if err != nil {
			continue
		}
		l.listenChange()
		listenerList[key] = l

		log.Debug("创建", "文件模式创建,file=%v,appType=%v,appId=%v,key=%v",
			l.fileName, l.appType, l.appId, key)
	}

	//定时检测
	g.Skeleton.LoopFunc(1*time.Second, func() {
		for _, l := range listenerList {
			if !l.needReadLoad {
				continue
			}

			log.Debug("", "不断重新载入:file=%v,type=%v,id=%v", l.fileName, l.appType, l.appId)
			l.changeChan <- struct{}{}
		}
	}, timer.LoopForever)
}

type configChangeListener struct {
	info             *aConfig.AppConfig
	appType, appId   uint32
	configs          map[string]string
	changeChan       chan struct{}
	fileName         string
	subscriptionList map[uint64]*apollo.ConfKey
	fileWatch        *fsnotify.Watcher
	needReadLoad     bool
}

func newListener(info *aConfig.AppConfig, apolloConfig conf.ApolloConfig) *configChangeListener {
	l := new(configChangeListener)
	l.info = info
	l.appType = apolloConfig.ServerType
	l.appId = apolloConfig.ServerId
	l.fileName = apolloConfig.Ns
	l.configs = make(map[string]string)
	l.subscriptionList = make(map[uint64]*apollo.ConfKey)
	l.changeChan = make(chan struct{})
	l.fileWatch = nil
	l.needReadLoad = false
	return l
}

func (c *configChangeListener) OnChange(changeEvent *storage.ChangeEvent) {
	//write your code here
	for key, value := range changeEvent.Changes {
		log.Debug("OnChange", "OnChange,key=%v,v=%v", key, value)
		switch value.ChangeType {
		case storage.ADDED, storage.MODIFIED:
			c.configs[key] = value.NewValue.(string)
		case storage.DELETED:
			delete(c.configs, key)
		default:
			log.Warning("变化", "出现未知变化类型,k=%v,v=%v", key, value)
		}

		c.notifySubscriptionList(key, 0, 0)
	}
}

func (c *configChangeListener) OnNewestChange(event *storage.FullChangeEvent) {
	//write your code here
	//for key, value := range event.Changes {
	//	log.Debug("OnNewestChange", "OnNewestChange,key=%v,v=%v", key, value)
	//}
}

func (c *configChangeListener) listenChange() {
	go func() {
		for {
			<-c.changeChan

			err := c.loadConfigFile()

			log.Debug("监听", "文件发生变化,fileName=%v,len=%v,type=%v,id=%v,needReadLoad=%v,err=%v",
				c.fileName, len(c.subscriptionList), c.appType, c.appId, c.needReadLoad, err)

			if err == nil {
				c.needReadLoad = false
				c.notifySubscriptionList("", 0, 0)
			} else {
				c.needReadLoad = true
			}
		}
	}()
}

func (c *configChangeListener) addSubscriptionItem(appType, appId uint32, subAppType, subAppId uint32, subKey string) {
	sourceKey := util.MakeUint64FromUint32(appType, appId)
	if _, ok := c.subscriptionList[sourceKey]; ok {
		return
	}
	c.subscriptionList[sourceKey] = &apollo.ConfKey{AppType: subAppType, AppId: subAppId, Key: subKey}

	log.Debug("插入", "插入列表,key=%v,%v, %v,subKey=%v", sourceKey, appType, appId, subKey)
}

func (c *configChangeListener) notifySubscriptionList(changeKey string, reqType, reqId uint32) {
	for k, sub := range c.subscriptionList {
		if sub.Key != "" && sub.Key != changeKey {
			continue
		}

		appType := util.GetHUint32FromUint64(k)
		appId := util.GetLUint32FromUint64(k)
		//请求判断
		if reqType != 0 && reqId != 0 {
			if reqType != appType || reqId != appId {
				continue
			}
		}

		var rsp config.ConfigRsp
		rsp.RegKey = proto.String(sub.Key)
		rsp.SubAppType = proto.Uint32(sub.AppType)
		rsp.SubAppId = proto.Uint32(sub.AppId)

		makeItem := func(k, v string) *config.ConfigItem {
			i := new(config.ConfigItem)
			i.Key = proto.String(k)
			i.Value = proto.String(v)
			return i
		}
		msgRespond := func() {
			log.Debug("通知", "下发通知,appType=%v,appId=%v,subKey=%v,SubAppType=%v,SubAppId=%v,changeKey=%v,len=%v,sCount=%v,appCount=%v",
				appType, appId, sub.Key, sub.AppType, sub.AppId, changeKey, len(rsp.GetItem()), len(c.subscriptionList), len(g.GetDestApp(appType, appId)))

			g.SendData2App(appType, appId, n.AppConfig, uint32(config.CMDConfig_IDConfigRsp), &rsp)
			rsp.Item = make([]*config.ConfigItem, 0)
		}
		sendLargeState := func(subAppType, subAppId uint32, k string, state uint32) {
			log.Debug("通知", "下发大数据通知,appType=%v,appId=%v,subKey=%v,SubAppType=%v,SubAppId=%v,changeKey=%v,len=%v,state=%v",
				appType, appId, sub.Key, subAppType, subAppId, changeKey, len(rsp.GetItem()), state)
			var rsp config.ItemRspState
			rsp.Key = proto.String(k)
			rsp.SubAppType = proto.Uint32(subAppType)
			rsp.SubAppId = proto.Uint32(subAppId)
			rsp.State = proto.Uint32(state)
			g.SendData2App(appType, appId, n.AppConfig, uint32(config.CMDConfig_IDItemRspState), &rsp)
		}

		sendLargeValue := func(k, v string, maxLen int) {
			//开始通知
			sendLargeState(sub.AppType, sub.AppId, k, apollo.StateBeginSend)
			tValue := v
			for {
				if len(tValue) <= maxLen {
					//剩余的发完
					if len(tValue) > 0 {
						rsp.Item = append(rsp.Item, makeItem(k, tValue))
						msgRespond()
					}
					break
				}

				rsp.Item = append(rsp.Item, makeItem(k, tValue[0:maxLen]))
				msgRespond()
				tValue = tValue[maxLen:]
			}
			//结束通知
			sendLargeState(sub.AppType, sub.AppId, k, apollo.StateComplete)
		}

		sendMaxLen := (int)(g.MaxMsgLen * 8 / 10)
		//是否全部发送
		if changeKey == "" {
			msgSize := 0
			for key, value := range c.configs {
				//包体太大了
				vLen := len(value)
				if msgSize+vLen > sendMaxLen {
					//先把当前的发出去
					if len(rsp.Item) > 0 {
						msgRespond()
						msgSize = 0
					}

					//如果单个配置就已经超出上限了
					if len(value) > sendMaxLen {
						sendLargeValue(key, value, sendMaxLen)
					} else {
						rsp.Item = append(rsp.Item, makeItem(key, value))
						msgSize += len(value)
					}
				} else {
					rsp.Item = append(rsp.Item, makeItem(key, value))
					msgSize += len(value)
				}
			}
		} else {
			if value, ok := c.configs[changeKey]; ok {
				if len(value) > sendMaxLen {
					sendLargeValue(changeKey, value, sendMaxLen)
				} else {
					rsp.Item = append(rsp.Item, makeItem(changeKey, value))
				}
			}
		}

		//发送剩余
		if len(rsp.Item) > 0 {
			msgRespond()
		}
	}
}

func (c *configChangeListener) loadConfigFile() error {
	fileData, err := ioutil.ReadFile(c.fileName)
	if err != nil {
		log.Warning("Listener", "加载配置文件失败,fileName=%v,err=%v", c.fileName, err)
		return err
	}

	ext := path.Ext(c.fileName)
	log.Debug("", "文件后缀,ext=%v,len=%v", ext, len(fileData))
	switch ext {
	case ".csv", ".xlsx":
		filename := path.Base(c.fileName)
		key := filename[0 : len(filename)-len(path.Ext(c.fileName))]
		c.configs[key] = string(fileData)
	case ".json":
		cfg, err := simplejson.NewJson(fileData)
		if err != nil {
			log.Warning("", "Error!err=%v", err)
			return err
		}

		configurations := cfg.Get("configurations")
		c.configs = make(map[string]string)
		for k, v := range configurations.MustMap() {
			var value string
			switch t := v.(type) {
			case string:
				value = t
			default:
				if b, err := configurations.Get(k).MarshalJSON(); err == nil {
					value = string(b)
				} else {
					log.Warning("", "未处理类型配置,file=%v,k=%v,t=%v", c.fileName, k, t)
				}
			}
			c.configs[k] = value
		}
	default:
		return fmt.Errorf("文件加载失败,file=%v,ext=%v", c.fileName, ext)
	}

	return nil
}

func handleApolloCfgReq(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*config.ConfigReq)

	l := getListener(m.GetSubAppType(), m.GetSubAppId())
	if l == nil {
		key := util.MakeUint64FromUint32(m.GetSubAppType(), m.GetSubAppId())
		log.Warning("配置", "配置不存在,AppType=%v,AppId=%v,KeyName=%v,SubAppType=%v,SubAppId=%v,Subscribe=%v,key=%v,len=%v",
			m.GetAppType(), m.GetAppId(), m.GetKey(), m.GetSubAppType(), m.GetSubAppId(), m.GetSubscribe(), key, len(listenerList))

		return
	}

	log.Debug("配置", "收到配置请求,AppType=%v,AppId=%v,KeyName=%v,SubAppType=%v,SubAppId=%v,Subscribe=%v",
		m.GetAppType(), m.GetAppId(), m.GetKey(), m.GetSubAppType(), m.GetSubAppId(), m.GetSubscribe())

	l.addSubscriptionItem(m.GetAppType(), m.GetAppId(), m.GetSubAppType(), m.GetSubAppId(), m.GetKey())
	if m.GetSubscribe()&uint32(config.ConfigReq_NEED_RSP) != 0 {
		if len(g.GetDestApp(m.GetAppType(), m.GetAppId())) > 0 {
			l.notifySubscriptionList(m.GetKey(), m.GetAppType(), m.GetAppId())
		} else {
			log.Info("", "可能还没注册好,稍后注册后再通知,key=%v,type=%v,Id=%v", m.GetKey(), m.GetAppType(), m.GetAppId())
			needNotifyList[util.MakeUint64FromUint32(m.GetAppType(), m.GetAppId())] = l
		}
	}
}

func getListener(appType, appId uint32) *configChangeListener {
	key := util.MakeUint64FromUint32(appType, appId)
	if _, ok := listenerList[key]; ok {
		return listenerList[key]
	}

	key = util.MakeUint64FromUint32(appType, 0)
	if _, ok := listenerList[key]; ok {
		return listenerList[key]
	}

	return nil
}
