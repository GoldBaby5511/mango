/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package agollo

import (
	"container/list"
	"strconv"

	"mango/third_party/agollo/agcache"
	"mango/third_party/agollo/agcache/memory"
	"mango/third_party/agollo/cluster/roundrobin"
	"mango/third_party/agollo/component"
	"mango/third_party/agollo/component/log"
	"mango/third_party/agollo/component/notify"
	"mango/third_party/agollo/component/remote"
	"mango/third_party/agollo/component/serverlist"
	"mango/third_party/agollo/constant"
	"mango/third_party/agollo/env"
	"mango/third_party/agollo/env/config"
	jsonFile "mango/third_party/agollo/env/file/json"
	"mango/third_party/agollo/extension"
	"mango/third_party/agollo/protocol/auth/sign"
	"mango/third_party/agollo/storage"
	"mango/third_party/agollo/utils"
	"mango/third_party/agollo/utils/parse/normal"
	"mango/third_party/agollo/utils/parse/properties"
	"mango/third_party/agollo/utils/parse/yaml"
	"mango/third_party/agollo/utils/parse/yml"
)

var (
	//next try connect period - 60 second
	nextTryConnectPeriod int64 = 60
)

func init() {
	extension.SetCacheFactory(&memory.DefaultCacheFactory{})
	extension.SetLoadBalance(&roundrobin.RoundRobin{})
	extension.SetFileHandler(&jsonFile.FileHandler{})
	extension.SetHTTPAuth(&sign.AuthSignature{})

	// file parser
	extension.AddFormatParser(constant.DEFAULT, &normal.Parser{})
	extension.AddFormatParser(constant.Properties, &properties.Parser{})
	extension.AddFormatParser(constant.YML, &yml.Parser{})
	extension.AddFormatParser(constant.YAML, &yaml.Parser{})
}

var syncApolloConfig = remote.CreateSyncApolloConfig()

// Client apollo ???????????????
type Client struct {
	initAppConfigFunc func() (*config.AppConfig, error)
	appConfig         *config.AppConfig
	cache             *storage.Cache
}

func (c *Client) getAppConfig() config.AppConfig {
	return *c.appConfig
}

func create() *Client {

	appConfig := env.InitFileConfig()
	return &Client{
		appConfig: appConfig,
	}
}

// Start ????????????????????????
func Start() (*Client, error) {
	return StartWithConfig(nil)
}

// StartWithConfig ??????????????????
func StartWithConfig(loadAppConfig func() (*config.AppConfig, error)) (*Client, error) {
	// ???????????????????????????????????????
	appConfig, err := env.InitConfig(loadAppConfig)
	if err != nil {
		return nil, err
	}

	c := create()
	if appConfig != nil {
		c.appConfig = appConfig
	}

	c.cache = storage.CreateNamespaceConfig(appConfig.NamespaceName)
	appConfig.Init()

	serverlist.InitSyncServerIPList(c.getAppConfig)

	//first sync
	configs := syncApolloConfig.Sync(c.getAppConfig)
	if len(configs) > 0 {
		for _, apolloConfig := range configs {
			c.cache.UpdateApolloConfig(apolloConfig, c.getAppConfig)
		}
	}

	log.Debug("init notifySyncConfigServices finished")

	//start long poll sync config
	configComponent := &notify.ConfigComponent{}
	configComponent.SetAppConfig(c.getAppConfig)
	configComponent.SetCache(c.cache)
	go component.StartRefreshConfig(configComponent)

	log.Info("agollo start finished ! ")

	return c, nil
}

//GetConfig ??????namespace??????apollo??????
func (c *Client) GetConfig(namespace string) *storage.Config {
	return c.GetConfigAndInit(namespace)
}

//GetConfigAndInit ??????namespace??????apollo??????
func (c *Client) GetConfigAndInit(namespace string) *storage.Config {
	if namespace == "" {
		return nil
	}

	config := c.cache.GetConfig(namespace)

	if config == nil {
		//init cache
		storage.CreateNamespaceConfig(namespace)

		//sync config
		syncApolloConfig.SyncWithNamespace(namespace, c.getAppConfig)
	}

	config = c.cache.GetConfig(namespace)

	return config
}

//GetConfigCache ??????namespace??????apollo???????????????
func (c *Client) GetConfigCache(namespace string) agcache.CacheInterface {
	config := c.GetConfigAndInit(namespace)
	if config == nil {
		return nil
	}

	return config.GetCache()
}

//GetDefaultConfigCache ??????????????????
func (c *Client) GetDefaultConfigCache() agcache.CacheInterface {
	config := c.GetConfigAndInit(storage.GetDefaultNamespace())
	if config != nil {
		return config.GetCache()
	}
	return nil
}

//GetApolloConfigCache ????????????namespace???apollo??????
func (c *Client) GetApolloConfigCache() agcache.CacheInterface {
	return c.GetDefaultConfigCache()
}

//GetValue ????????????
func (c *Client) GetValue(key string) string {
	value := c.getConfigValue(key)
	if value == nil {
		return utils.Empty
	}

	return value.(string)
}

//GetStringValue ??????string?????????
func (c *Client) GetStringValue(key string, defaultValue string) string {
	value := c.GetValue(key)
	if value == utils.Empty {
		return defaultValue
	}

	return value
}

//GetIntValue ??????int?????????
func (c *Client) GetIntValue(key string, defaultValue int) int {
	value := c.GetValue(key)

	i, err := strconv.Atoi(value)
	if err != nil {
		log.Debug("convert to int fail!error:", err)
		return defaultValue
	}

	return i
}

//GetFloatValue ??????float?????????
func (c *Client) GetFloatValue(key string, defaultValue float64) float64 {
	value := c.GetValue(key)

	i, err := strconv.ParseFloat(value, 64)
	if err != nil {
		log.Debug("convert to float fail!error:", err)
		return defaultValue
	}

	return i
}

//GetBoolValue ??????bool ?????????
func (c *Client) GetBoolValue(key string, defaultValue bool) bool {
	value := c.GetValue(key)

	b, err := strconv.ParseBool(value)
	if err != nil {
		log.Debug("convert to bool fail!error:", err)
		return defaultValue
	}

	return b
}

//GetStringSliceValue ??????[]string ?????????
func (c *Client) GetStringSliceValue(key string, defaultValue []string) []string {
	value := c.getConfigValue(key)

	if value == nil {
		return defaultValue
	}
	s, ok := value.([]string)
	if !ok {
		return defaultValue
	}
	return s
}

//GetIntSliceValue ??????[]int ?????????
func (c *Client) GetIntSliceValue(key string, defaultValue []int) []int {
	value := c.getConfigValue(key)

	if value == nil {
		return defaultValue
	}
	s, ok := value.([]int)
	if !ok {
		return defaultValue
	}
	return s
}

func (c *Client) getConfigValue(key string) interface{} {
	cache := c.GetDefaultConfigCache()
	if cache == nil {
		return utils.Empty
	}

	value, err := cache.Get(key)
	if err != nil {
		log.Errorf("get config value fail!key:%s,err:%s", key, err)
		return utils.Empty
	}

	return value
}

// AddChangeListener ??????????????????
func (c *Client) AddChangeListener(listener storage.ChangeListener) {
	c.cache.AddChangeListener(listener)
}

// RemoveChangeListener ??????????????????
func (c *Client) RemoveChangeListener(listener storage.ChangeListener) {
	c.cache.RemoveChangeListener(listener)
}

// GetChangeListeners ?????????????????????????????????
func (c *Client) GetChangeListeners() *list.List {
	return c.cache.GetChangeListeners()
}

// UseEventDispatch  ???????????????key??????event??????
func (c *Client) UseEventDispatch() {
	c.AddChangeListener(storage.UseEventDispatch())
}
