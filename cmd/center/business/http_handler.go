package business

import (
	"encoding/json"
	"io/ioutil"
	g "mango/pkg/gate"
	"mango/pkg/log"
	n "mango/pkg/network"
	"mango/pkg/util"
	"mango/pkg/util/errorhelper"
	"net/http"
	"sort"
)

const (
	//code成功
	CODE_SUCCESS = 200
	//未知错误
	CODE_UNKNOWN = 10001
	//参数错误
	CODE_PARAMERROR = 10002
	//token验证失败
	CODE_TOKENERROR = 10003
)

func init() {
	if g.HttpServer == nil {
		log.Warning("", "警告,HttpServer == nil")
		return
	}

	if g.HttpServer.IsStart() {
		log.Warning("", "警告,http服务已启动")
		return
	}

	//注册路由
	g.HttpRouteRegister("/server_list", httpHandleServerList)
	g.HttpRouteRegister("/server_control", httpHandleServerControl)
	g.HttpRouteRegister("/server_start", httpHandleServerStart)
}

func verify(rsp http.ResponseWriter, req *http.Request) bool {
	n.SetupCORS(&rsp)
	if req.Method == "OPTIONS" {
		return false
	}

	req.ParseForm()
	return true
}

//获取服务器列表
func httpHandleServerList(rsp http.ResponseWriter, req *http.Request) {
	defer errorhelper.Recover()

	if !verify(rsp, req) {
		return
	}
	log.Debug("", "收到列表请求")
	responseServerList(rsp)
}

//
func httpHandleServerControl(rsp http.ResponseWriter, req *http.Request) {
	defer errorhelper.Recover()

	if !verify(rsp, req) {
		return
	}

	type serverItem struct {
		AppType uint32 `json:"appType"`
		AppId   uint32 `json:"appId"`
	}

	params := &struct {
		CtlId      int32        `json:"ctlId"`
		Token      string       `json:"token"`
		ServerList []serverItem `json:"serverList"`
	}{}

	if err := json.NewDecoder(req.Body).Decode(params); err != nil {
		n.HttpFail(rsp, CODE_PARAMERROR, err, "")
		return
	}

	//TODO:验证token

	log.Debug("", "控制命令,params=%v", params)
	errInfo := ""
	for _, item := range params.ServerList {
		if err := controlServer(item.AppType, item.AppId, params.CtlId); err != nil {
			errInfo += err.Error()
		}
	}

	if errInfo != "" {
		n.HttpFail(rsp, CODE_PARAMERROR, nil, errInfo)
	} else {
		n.HttpSuccess(rsp, nil, "执行成功")
	}
}

//开启服务
func httpHandleServerStart(rsp http.ResponseWriter, req *http.Request) {
	defer errorhelper.Recover()

	if !verify(rsp, req) {
		return
	}

	buff, _ := ioutil.ReadAll(req.Body)
	log.Debug("", "收到启动请求,buff=%v", string(buff))
}

//获取服务器列表
func responseServerList(rsp http.ResponseWriter) {
	type ServerList struct {
		Id          int    `json:"id" `
		AppName     string `json:"appName"`
		AppType     uint32 `json:"appType"`
		AppId       uint32 `json:"appId"`
		AppState    int    `json:"appState" `
		Address     string `json:"address" `
		Description string `json:"description"`
	}

	idKey := make([]float64, 0)
	for k, _ := range appRegData {
		idKey = append(idKey, float64(k))
	}
	sort.Float64s(idKey)
	sList := make([]ServerList, 0)
	for _, k := range idKey {
		v := appRegData[uint64(k)]
		if v.appInfo.Type == n.AppDaemon {
			continue
		}
		l := ServerList{
			Id:      len(sList) + 1,
			AppName: v.appInfo.Name,
			AppType: v.appInfo.Type,
			AppId:   v.appInfo.Id,
			Address: v.appInfo.ListenOnAddr,
		}
		l.AppState = v.appState
		l.Description = v.stateDescription
		if s := getBaseInfoFromConfigList(v.appInfo.Type, v.appInfo.Id); s != nil {
			l.AppName = s.Alias
		}
		sList = append(sList, l)
	}

	//添加配置但未启动注册服务
	cs := getConfigServerList()
	for _, s := range cs {
		if s.Type == n.AppDaemon {
			continue
		}
		key := util.MakeUint64FromUint32(s.Type, s.Id)
		if _, ok := appRegData[key]; !ok {
			l := ServerList{
				Id:      len(sList) + 1,
				AppName: s.Alias,
				AppType: s.Type,
				AppId:   s.Id,
				Address: s.ListenOnAddr,
			}
			sList = append(sList, l)
		}
	}

	r := &struct {
		List  []ServerList `json:"list"`
		Count int          `json:"count"`
	}{
		List: sList, Count: len(sList),
	}

	n.HttpSuccess(rsp, r, "成功了啊")
}
