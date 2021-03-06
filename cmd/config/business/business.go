package business

import (
	"encoding/json"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/golang/protobuf/proto"
	"io/ioutil"
	"mango/api/config"
	"mango/cmd/config/conf"
	lconf "mango/pkg/conf"
	"mango/pkg/conf/apollo"
	g "mango/pkg/gate"
	"mango/pkg/log"
	n "mango/pkg/network"
	"mango/pkg/util"
	"mango/third_party/agollo"
	aConfig "mango/third_party/agollo/env/config"
	"mango/third_party/agollo/storage"
	"mango/third_party/go-simplejson"
)

var (
	listenerList []*configChangeListener
)

func init() {
	g.MsgRegister(&config.ApolloCfgReq{}, n.AppConfig, uint16(config.CMDConfig_IDApolloCfgReq), handleApolloCfgReq)
	g.EventRegister(g.ConnectSuccess, connectSuccess)
	g.EventRegister(g.Disconnect, disconnect)
	g.EventRegister(g.CenterConnected, centerConnected)
	g.EventRegister(g.CenterRegResult, centerRegResult)

	loadConfigs()
}

func connectSuccess(args []interface{}) {
	log.Info("连接", "来了老弟,参数数量=%d", len(args))
}

func disconnect(args []interface{}) {
	log.Info("连接", "告辞中,参数数量=%d", len(args))
}

func loadConfigs() {
	//是否使用Apollo
	if conf.Server.UseApollo {
		c := &conf.Server.Config
		listenerConf := conf.ApolloConfig{ServerType: lconf.AppInfo.Type, ServerId: conf.Server.AppID}
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
						listenerList = append(listenerList, l)
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
		for _, c := range conf.Server.CommonServers {
			l := newListener(nil, c)
			err := l.loadConfigFile()
			if err != nil {
				continue
			}
			l.watchConfigFile()
			l.listenChange()
			listenerList = append(listenerList, l)

			log.Debug("创建", "文件模式创建,file=%v,appType=%v,appId=%v",
				l.fileName, l.appType, l.appId)
		}
	}
}

func centerConnected(args []interface{}) {
}

func centerRegResult(args []interface{}) {
	r := args[0].(uint32)
	routerId := args[1].(uint32)
	if r == 0 {
		listerIndex := getListenerIndex(n.AppCenter, routerId)
		if listerIndex < 0 {
			log.Warning("配置", "center配置不存在,listerIndex=%v,appType=%v,routerId=%v",
				listerIndex, n.AppCenter, routerId)
			return
		}

		listenerList[listerIndex].addSubscriptionItem(n.AppCenter, routerId, n.AppCenter, routerId, "")
		listenerList[listerIndex].notifySubscriptionList("")
	}
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

		c.notifySubscriptionList(key)
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
	for k, v := range c.subscriptionList {
		if v.Key != "" && v.Key != changeKey {
			continue
		}

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
		appType := util.GetHUint32FromUint64(k)
		appId := util.GetLUint32FromUint64(k)
		log.Debug("通知", "下发通知,appType=%v, appId=%v,SubAppType=%v, SubAppId=%v,changeKey=%v",
			appType, appId, v.AppType, v.AppId, changeKey)
		g.SendData2App(appType, appId, n.AppConfig, uint32(config.CMDConfig_IDApolloCfgRsp), &rsp)
	}
}

func (c *configChangeListener) loadConfigFile() error {
	jsonData, err := ioutil.ReadFile(c.fileName)
	if err != nil {
		log.Fatal("Listener", "加载配置文件失败,fileName=%v,err=%v", c.fileName, err)
		return err
	}

	config, err := simplejson.NewJson(jsonData)
	if err != nil {
		log.Warning("", "Error!err=%v", err)
		return err
	}

	configurations := config.Get("configurations")
	c.configs = make(map[string]string)
	for k, v := range configurations.MustMap() {
		var value string
		switch t := v.(type) {
		case string:
			value = t
		case map[string]interface{}:
			if b, err := configurations.Get(k).MarshalJSON(); err == nil {
				value = string(b)
			} else {
				log.Warning("", "出错,file=%v,k=%v,err=%v", c.fileName, k, err)
			}
		default:
			log.Warning("", "未处理类型配置,file=%v,k=%v,t=%v", c.fileName, k, t)
		}
		c.configs[k] = value
	}
	return nil
}

func (c *configChangeListener) watchConfigFile() {
	//创建一个监控对象
	var err error
	c.fileWatch, err = fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("", "err=%v", err)
		return
	}

	//添加要监控的对象，文件或文件夹
	err = c.fileWatch.Add(c.fileName)
	if err != nil {
		log.Fatal("", "err=%v", err)
		return
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
						log.Debug("", "写入文件 : %v", ev.Name)
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
						log.Error("", "fileWatch.Errors,err=%v", err)
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
	//log.Debug("agollo", format, params...)
}

func (d *DefaultLogger) Infof(format string, params ...interface{}) {
	log.Info("agollo", format, params...)
}

func (d *DefaultLogger) Warnf(format string, params ...interface{}) {
	log.Warning("agollo", format, params...)
}

func (d *DefaultLogger) Errorf(format string, params ...interface{}) {
	log.Error("agollo", format, params...)
}

func (d *DefaultLogger) Debug(v ...interface{}) {
	//log.Debug("agollo", "%v", fmt.Sprint(v...))
}
func (d *DefaultLogger) Info(v ...interface{}) {
	log.Info("agollo", "%v", fmt.Sprint(v...))
}

func (d *DefaultLogger) Warn(v ...interface{}) {
	log.Warning("agollo", "%v", fmt.Sprint(v...))
}

func (d *DefaultLogger) Error(v ...interface{}) {
	log.Error("agollo", "%v", fmt.Sprint(v...))
}

func handleApolloCfgReq(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*config.ApolloCfgReq)

	log.Debug("配置", "收到配置请求,AppType=%v,AppId=%v,KeyName=%v,SubAppType=%v,SubAppId=%v,Subscribe=%v",
		m.GetAppType(), m.GetAppId(), m.GetKeyName(), m.GetSubAppType(), m.GetSubAppId(), m.GetSubscribe())

	listerIndex := getListenerIndex(m.GetSubAppType(), m.GetSubAppId())
	if listerIndex < 0 {
		log.Warning("配置", "配置不存在,NameSpace=%v,KeyName=%v", m.GetNameSpace(), m.GetKeyName())
		return
	}

	listenerList[listerIndex].addSubscriptionItem(m.GetAppType(), m.GetAppId(), m.GetSubAppType(), m.GetSubAppId(), m.GetKeyName())
	if m.GetSubscribe()&uint32(config.ApolloCfgReq_NEED_RSP) != 0 {
		listenerList[listerIndex].notifySubscriptionList(m.GetKeyName())
	}
}

func getListenerIndex(appType, appId uint32) int {
	findListerIndex := func(appType, appId uint32) int {
		for i, v := range listenerList {
			if v.appType == appType && v.appId == appId {
				return i
			}
		}
		return -1
	}

	listerIndex := findListerIndex(appType, appId)
	if listerIndex < 0 {
		listerIndex = findListerIndex(appType, 0)
	}

	return listerIndex
}
