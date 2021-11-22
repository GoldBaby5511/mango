package business

import (
	"encoding/json"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"google.golang.org/protobuf/proto"
	_ "google.golang.org/protobuf/proto"
	"io/ioutil"
	"xlddz/core/conf/apollo"
	"xlddz/core/gate"
	"xlddz/core/log"
	n "xlddz/core/network"
	"xlddz/core/network/protobuf"
	"xlddz/protocol/config"
	"xlddz/servers/config/agollo"
	aConfig "xlddz/servers/config/agollo/env/config"
	"xlddz/servers/config/agollo/storage"
	"xlddz/servers/config/conf"
)

var (
	listenerList []*configChangeListener
)

type Module struct {
	*gate.Gate
}

func (m *Module) OnInit() {

	log.Debug("Module", "配置中心初始化")

	//消息注册
	p := protobuf.NewProcessor()
	p.RegHandle(&config.ApolloCfgReq{}, n.CMDConfig, uint16(config.CMDID_Config_IDApolloCfgReq), handleApolloCfgReq)

	m.Gate = &gate.Gate{
		Processor:     p,
		TCPClientAddr: conf.Server.TCPClientAddr,
		LogAddr:       conf.Server.LogServerAddr,
	}

	//是否使用Apollo
	if conf.Server.UseApollo {
		c := &conf.Server.Config
		listenerConf := conf.ApolloConfig{ServerType: conf.Server.AppType, ServerId: conf.Server.AppID}
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
			case "common-server":
				//var s configs
				var s []conf.ApolloConfig
				err := json.Unmarshal([]byte(value.(string)), &s)
				if err != nil {
					log.Fatal("初始化", "配置解析失败,err=%v", err)
					return false
				}
				for i, v := range s {
					log.Debug("key", "读取普通服务,i = %v,len=%v,info=%v", i, len(s), v)

					c := &aConfig.AppConfig{
						AppID:          v.Appid,
						Cluster:        v.Cluster,
						IP:             v.Ip,
						NamespaceName:  v.Ns,
						IsBackupConfig: true,
					}

					client, _ := agollo.StartWithConfig(func() (*aConfig.AppConfig, error) {
						log.Debug("StartWithConfig", "StartWithConfig,c=%v", c)
						return c, nil
					})

					l := newListener(c, v)
					client.AddChangeListener(l)
					cache := client.GetConfigCache(c.NamespaceName)
					cache.Range(func(key, value interface{}) bool {
						//fmt.Println("key : ", key, ", value :", value)

						l.configs[key.(string)] = value.(string)
						return true
					})
					listenerList = append(listenerList, l)

					log.Debug("创建", "创建,appType=%v,appId=%v,len=%v",
						l.appType, l.appId, len(listenerList))
				}
			case "日志服务器地址":
				k := apollo.ConfKey{Key: "日志服务器地址"}
				v := apollo.ConfValue{}
				gate.ApolloNotify(k, v)
			default:
				log.Warning("", "没有处理的,k=%v,v=%v", key, value)
			}

			return true
		})

		log.Debug("测试", "configs=%v", listener.configs)
	} else {

		for _, c := range conf.Server.CommonServers {
			l := newListener(nil, c)
			err := l.loadConfigFile()
			if err != nil {
				continue
			}
			l.watchConfigFile()
			l.listenChange()
			listenerList = append(listenerList, l)

			log.Debug("创建", "创建,appType=%v,appId=%v,len=%v",
				l.appType, l.appId, len(listenerList))
		}
	}

}

//OnDestroy module实现
func (m *Module) OnDestroy() {

}

type configChangeListener struct {
	info             *aConfig.AppConfig
	appType          uint32
	appId            uint32
	configs          map[string]string
	changeChan       chan struct{}
	fileName         string
	subscriptionList map[uint64]*apollo.ConfKey
	fileWatch        *fsnotify.Watcher
	routeAgent       n.Agent
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
	l.routeAgent = nil
	return l
}

func (c *configChangeListener) OnChange(changeEvent *storage.ChangeEvent) {
	//write your code here
	fmt.Println(changeEvent.Changes)
	for key, value := range changeEvent.Changes {
		fmt.Println("change key : ", key, ", value :", value)
		c.configs[key] = value.NewValue.(string)
		c.notifySubscriptionList(key)
	}
	fmt.Println(changeEvent.Namespace)
}

func (c *configChangeListener) OnNewestChange(event *storage.FullChangeEvent) {
	//write your code here
}

func (c *configChangeListener) listenChange() {
	go func() {
		for {
			<-c.changeChan
			log.Debug("监听", "文件发生变化,len=%v,type=%v,id=%v",
				len(c.subscriptionList), c.appType, c.appId)

			err := c.loadConfigFile()
			if err == nil {
				c.notifySubscriptionList("")
			}
		}
	}()
}

func (c *configChangeListener) addSubscriptionItem(appType, appId uint32, subAppType, subAppId uint32, subKey string) {
	sourceKey := uint64(appType)<<32 + uint64(appId)
	if _, ok := c.subscriptionList[sourceKey]; ok {
		return
	}
	c.subscriptionList[sourceKey] = &apollo.ConfKey{AppType: subAppType, AppId: subAppId, Key: subKey}

	log.Debug("插入", "插入列表,key=%v,%v, %v,subKey=%v", sourceKey, appType, appId, subKey)

}

func (c *configChangeListener) notifySubscriptionList(changeKey string) {
	if c.routeAgent == nil {
		log.Error("通知", "异常,准备通知时router为空,len=%v,changeKey=%v,type=%v,id=%v",
			len(c.subscriptionList), changeKey, c.appType, c.appId)
		return
	}

	for k, v := range c.subscriptionList {
		if v.Key != changeKey {
			continue
		}
		appType := uint32(k >> 32)
		appId := uint32(k & 0xFFFFFFFF)

		var rsp config.ApolloCfgRsp
		rsp.SubAppType = proto.Uint32(v.AppType)
		rsp.SubAppId = proto.Uint32(v.AppId)
		if changeKey == "" {
			for key, value := range c.configs {
				rsp.Key = append(rsp.Key, key)
				rsp.Value = append(rsp.Value, value)
			}
		} else {
			rsp.Key = append(rsp.Key, changeKey)
			for key, value := range c.configs {
				if key != changeKey {
					continue
				}
				rsp.Value = append(rsp.Value, value)
				break
			}
		}
		log.Debug("通知", "通知,key=%v,%v, %v", v, appType, appId)
		c.routeAgent.SendData2App(appType, appId, n.CMDConfig, uint32(config.CMDID_Config_IDApolloCfgRsp), &rsp)
	}
}

func (c *configChangeListener) loadConfigFile() error {
	jsonData, err := ioutil.ReadFile(c.fileName)
	if err != nil {
		log.Error("Listener", "加载配置文件失败,fileName=%v,err=%v", c.fileName, err)
		return err
	}
	var v interface{}
	err = json.Unmarshal(jsonData, &v)
	if err != nil {
		log.Error("jsonconf", "Unmarshal err=%v", err)
		return err
	}

	c.configs = make(map[string]string)
	for k, v := range v.(map[string]interface{}) {
		switch v := v.(type) {
		case map[string]interface{}:
			if k == "configurations" {
				configData, _ := json.Marshal(v)
				err := json.Unmarshal(configData, &c.configs)
				if err != nil {
					panic(1)
				}
			}
		default:
		}
	}

	log.Debug("测试", "文件加载完成,appType=%v,id=%v,=%v", c.appType, c.appId, c.configs)
	return nil
}

func (c *configChangeListener) watchConfigFile() {
	//创建一个监控对象
	var err error
	c.fileWatch, err = fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("", "%v", err)
	}

	//添加要监控的对象，文件或文件夹
	err = c.fileWatch.Add(c.fileName)
	if err != nil {
		log.Fatal("", "%v", err)
	}
	//我们另启一个goroutine来处理监控对象的事件
	go func() {
		for {
			select {
			case ev := <-c.fileWatch.Events:
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
						log.Debug("", "写入文件 : %v,%v", ev.Name, &c)
						c.changeChan <- struct{}{}
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
			case err := <-c.fileWatch.Errors:
				{
					if err != nil {
						log.Error("", "fileWatch.Errors,err=%v : ", err)
					}
					return
				}
			}
		}
	}()
}

type DefaultLogger struct {
}

func (d *DefaultLogger) Debugf(format string, params ...interface{}) {
	log.Debug("agollo", format, params)
}

func (d *DefaultLogger) Infof(format string, params ...interface{}) {
	log.Info("agollo", format, params)
}

func (d *DefaultLogger) Warnf(format string, params ...interface{}) {
	log.Warning("agollo", format, params)
}

func (d *DefaultLogger) Errorf(format string, params ...interface{}) {
	log.Error("agollo", format, params)
}

func (d *DefaultLogger) Debug(v ...interface{}) {
	log.Debug("agollo", "%v", v)
}
func (d *DefaultLogger) Info(v ...interface{}) {
	log.Info("agollo", "%v", v)
}

func (d *DefaultLogger) Warn(v ...interface{}) {
	log.Warning("agollo", "%v", v)
}

func (d *DefaultLogger) Error(v ...interface{}) {
	log.Error("agollo", "%v", v)
}

func handleApolloCfgReq(args []interface{}) {
	a := args[n.AGENT_INDEX].(n.Agent)
	b := args[n.DATA_INDEX].(n.BaseMessage)
	m := (b.MyMessage).(*config.ApolloCfgReq)

	log.Debug("配置", "收到配置请求,AppType=%v,AppId=%v,KeyName=%v,SubAppType=%v,SubAppId=%v,Subscribe=%v",
		m.GetAppType(), m.GetAppId(), m.GetKeyName(), m.GetSubAppType(), m.GetSubAppId(), m.GetSubscribe())

	findListerIndex := func(appType, appId uint32) int {
		for i, v := range listenerList {
			if v.appType == appType && v.appId == appId {
				return i
			}
		}
		return -1
	}

	listerIndex := findListerIndex(m.GetSubAppType(), m.GetSubAppId())
	if listerIndex < 0 {
		listerIndex = findListerIndex(m.GetSubAppType(), 0)
	}
	if listerIndex < 0 {
		log.Warning("配置", "配置不存在,NameSpace=%v,KeyName=%v", m.GetNameSpace(), m.GetKeyName())
		return
	}

	listenerList[listerIndex].routeAgent = a
	listenerList[listerIndex].addSubscriptionItem(m.GetAppType(), m.GetAppId(), m.GetSubAppType(), m.GetSubAppId(), m.GetKeyName())
	//是否立即下发
	if m.GetSubscribe()&uint32(config.ApolloCfgReq_NEED_RSP) != 0 {
		listenerList[listerIndex].notifySubscriptionList(m.GetKeyName())
	}

}
