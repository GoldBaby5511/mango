package network

import (
	"encoding/json"
	"fmt"
	"mango/pkg/log"
	"mango/pkg/util"
	"mango/pkg/util/errorhelper"
	"net/http"
	"net/http/pprof"
	"sync"
)

type HttpServer struct {
	Port int
	mux  *http.ServeMux
	sync.Mutex
	isStart bool
	route   map[string]func(http.ResponseWriter, *http.Request)
}

func NewHttpServer() *HttpServer {
	hs := new(HttpServer)
	hs.isStart = false
	hs.route = make(map[string]func(http.ResponseWriter, *http.Request))
	return hs
}

func (hs *HttpServer) IsStart() bool {
	return hs.isStart
}

func (hs *HttpServer) Start() {
	if hs.Port == 0 || hs.IsStart() {
		return
	}
	go hs.run()
}

func (hs *HttpServer) run() {
	defer errorhelper.Recover()

	hs.Lock()
	if hs.mux == nil {
		hs.mux = http.NewServeMux()
	}
	hs.mux.HandleFunc("/debug/pprof/", pprof.Index)
	for r, h := range hs.route {
		hs.mux.HandleFunc(r, h)
	}
	hs.Unlock()

	hs.isStart = true

	log.Info("", fmt.Sprintf("http服务器启动,Port=%d,route=%v", hs.Port, len(hs.route)))

	if err := http.ListenAndServe(":"+fmt.Sprintf("%d", hs.Port), hs.mux); err != nil {
		log.Fatal("", "异常,http服务启动失败,port=%v,err=%v", hs.Port, err)
	}
}

func (hs *HttpServer) AddRoute(route string, handler func(http.ResponseWriter, *http.Request)) {
	if hs.IsStart() {
		return
	}
	if _, exist := hs.route[route]; exist {
		log.Warning("", "route已存在，r=%v", route)
		return
	}

	hs.route[route] = handler
}

type Responses interface {
	SetCode(int32)
	SetTraceID(string)
	SetMsg(string)
	SetData(interface{})
	SetSuccess(bool)
	Clone() Responses
}

type Response struct {
	// 数据集
	RequestId string `json:"requestId,omitempty"`
	Code      int32  `json:"code,omitempty"`
	Msg       string `json:"msg,omitempty"`
	Status    string `json:"status,omitempty"`
}

type response struct {
	Response
	Data interface{} `json:"data"`
}

type Page struct {
	Count     int `json:"count"`
	PageIndex int `json:"pageIndex"`
	PageSize  int `json:"pageSize"`
}

type page struct {
	Page
	List interface{} `json:"list"`
}

func (e *response) SetData(data interface{}) {
	e.Data = data
}

func (e response) Clone() Responses {
	return &e
}

func (e *response) SetTraceID(id string) {
	e.RequestId = id
}

func (e *response) SetMsg(s string) {
	e.Msg = s
}

func (e *response) SetCode(code int32) {
	e.Code = code
}

func (e *response) SetSuccess(success bool) {
	if !success {
		e.Status = "error"
	}
}

// SetupCORS 忽略跨域问题
func SetupCORS(rsp *http.ResponseWriter) {
	(*rsp).Header().Set("Access-Control-Allow-Origin", "*")
	(*rsp).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*rsp).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

func HttpSuccess(rsp http.ResponseWriter, data interface{}, msg string) {
	r := &response{}
	r.SetData(data)
	r.SetSuccess(true)
	if msg != "" {
		r.SetMsg(msg)
	}
	r.SetTraceID(util.GetUUID())
	r.SetCode(http.StatusOK)
	if buff, err := json.Marshal(r); err == nil {
		log.Debug("", "成功了,buff=%v", string(buff))
		rsp.Write(buff)
	} else {
		log.Warning("", "失败了")
	}
}

func HttpFail(rsp http.ResponseWriter, code uint32, err error, msg string) {
	r := &response{}

	if err != nil {
		r.SetMsg(err.Error())
	}
	if msg != "" {
		r.SetMsg(msg)
	}
	r.SetTraceID(util.GetUUID())
	r.SetCode(int32(code))
	r.SetSuccess(false)
	if buff, err := json.Marshal(r); err == nil {
		log.Debug("", "失败了,buff=%v", string(buff))
		rsp.Write(buff)
	} else {
		log.Warning("", "失败了")
	}
}
