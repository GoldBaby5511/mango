package business

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"mango/api/center"
	"mango/pkg/conf"
	"mango/pkg/console"
	g "mango/pkg/gate"
	"mango/pkg/log"
	n "mango/pkg/network"
	"mango/pkg/util"
	"strconv"
	"time"
)

type controlApp struct {
}

var (
	ctl        = new(controlApp)
	ctlIdValue = map[string]int32{
		"m":     int32(center.CtlId_Maintenance),
		"mf":    int32(center.CtlId_MaintenanceFinish),
		"s":     int32(center.CtlId_ShowServerList),
		"start": int32(center.CtlId_StartService),
		"stop":  int32(center.CtlId_StopService),
	}
)

func init() {
	g.MsgRegister(&center.AppControlRsp{}, n.AppCenter, uint16(center.CMDCenter_IDAppControlRsp), handleAppControlRsp)
	g.CallBackRegister(g.CbAfterServiceStart, afterServiceStart)
}

func afterServiceStart([]interface{}) {
	console.Register(ctl.name(), ctl.help(), ctl.run, g.Skeleton.ChanRPCServer)

	console.Init()
}

func (c *controlApp) name() string {
	return "ctl"
}

func (c *controlApp) help() string {
	return "control server"
}

func (c *controlApp) usage() string {
	h := "Maintenance			: m appType appId\r\n"
	h += "Maintenance Finish	: mf appType appId\r\n"
	h += "show app list			: s\r\n"
	return h
}

func (c *controlApp) run(args []interface{}) interface{} {
	if len(args) == 0 {
		return c.usage()
	}

	if _, ok := ctlIdValue[args[0].(string)]; !ok {
		log.Warning("", "未定义命令,c=%v,len=%v", args[0].(string), len(args))
		return c.usage()
	}

	result := "success"
	switch args[0] {
	case "m", "mf":
		if len(args) != 3 {
			return c.usage()
		}
		appType, _ := strconv.Atoi(args[1].(string))
		appId, _ := strconv.Atoi(args[2].(string))
		ctlId := ctlIdValue[args[0].(string)]

		err := controlServer(uint32(appType), uint32(appId), ctlId)

		result = err.Error()
	case "s":
		for _, v := range appRegData {
			i := fmt.Sprintf("App name=%v,type=%v,id=%v,state=%v", v.appInfo.Name, v.appInfo.Type, v.appInfo.Id, v.stateDescription)
			log.Debug("", "%v", i)
		}
	default:
		result = c.usage()
	}

	log.Debug("", "收到控制,len=%v", len(args))
	for _, a := range args {
		log.Debug("", "执行,a=%v", a)
	}

	return result
}

func handleAppControlRsp(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*center.AppControlRsp)

	key := util.MakeUint64FromUint32(m.GetAppType(), m.GetAppId())
	if _, ok := appRegData[key]; !ok {
		log.Warning("", "can not found app type=%v,id=%v", m.GetAppType(), m.GetAppId())
		return
	}

	log.Debug("", "控制回复,code=%v,ctlId=%v,type=%v,Id=%v", m.GetCode(), m.GetCtlId(), m.GetAppType(), m.GetAppId())
}

func controlServer(appType, appId uint32, ctlId int32) error {
	//合法判断
	if _, ok := center.CtlId_name[ctlId]; !ok {
		return fmt.Errorf("bad command type=%v,id=%v,ctlId=%v", appType, appId, ctlId)
	}

	//命令判断
	switch ctlId {
	case int32(center.CtlId_Maintenance), int32(center.CtlId_MaintenanceFinish), int32(center.CtlId_StopService):
		key := util.MakeUint64FromUint32(appType, appId)
		if _, ok := appRegData[key]; !ok {
			return fmt.Errorf("can not found app type=%v,id=%v", appType, appId)
		}

		appClient := appRegData[key].a
		if appClient == nil {
			return fmt.Errorf("appClient == nil type=%v,id=%v", appType, appId)
		}
		regInfo := appRegData[key].appInfo
		if ctlId == int32(center.CtlId_Maintenance) {
			appRegData[key].appState = conf.AppStateMaintenance
		} else if ctlId == int32(center.CtlId_MaintenanceFinish) {
			appRegData[key].appState = conf.AppStateMaintenanceFinish
		}

		var req center.AppControlReq
		req.CtlId = proto.Int32(ctlId)
		req.AppType = proto.Uint32(regInfo.Type)
		req.AppId = proto.Uint32(regInfo.Id)
		//是否配置中
		s := getBaseInfoFromConfigList(appType, appId)
		if s != nil {
			//寻找守护进程
			key := util.MakeUint64FromUint32(n.AppDaemon, s.DaemonId)
			if _, ok := appRegData[key]; ok {
				appRegData[key].a.SendData(n.AppCenter, uint32(center.CMDCenter_IDAppControlReq), &req)
				time.Sleep(time.Millisecond * 100)
			}
		}

		appClient.SendData(n.AppCenter, uint32(center.CMDCenter_IDAppControlReq), &req)
	case int32(center.CtlId_StartService), int32(center.CtlId_UpdateService):
		//是否配置中
		s := getBaseInfoFromConfigList(appType, appId)
		if s == nil {
			return fmt.Errorf("start fail not found app type=%v,id=%v", appType, appId)
		}

		//寻找守护进程
		key := util.MakeUint64FromUint32(n.AppDaemon, s.DaemonId)
		if _, ok := appRegData[key]; !ok {
			return fmt.Errorf("can not found app key=%v,type=%v,id=%v", key, n.AppDaemon, s.DaemonId)
		}

		var req center.AppControlReq
		req.CtlId = proto.Int32(ctlId)
		req.AppType = proto.Uint32(appRegData[key].appInfo.Type)
		req.AppId = proto.Uint32(appRegData[key].appInfo.Id)
		ss := &center.ControlItem{}
		ss.Name = proto.String(s.Name)
		ss.Type = proto.Uint32(appType)
		ss.Id = proto.Uint32(appId)
		if ctlId == int32(center.CtlId_StartService) {
			//是否已经启动
			key := util.MakeUint64FromUint32(appType, appId)
			if _, ok := appRegData[key]; ok {
				return fmt.Errorf("already running,type=%v,id=%v", appType, appId)
			}

			ss.Args = append(ss.Args, fmt.Sprintf(s.Name))
			ss.Args = append(ss.Args, fmt.Sprintf("-Type=%v", s.Type))
			ss.Args = append(ss.Args, fmt.Sprintf("-Id=%v", s.Id))
			ss.Args = append(ss.Args, fmt.Sprintf("-ListenOnAddr=%v", s.ListenOnAddr))
			ss.Args = append(ss.Args, fmt.Sprintf("-CenterAddr=%v", s.CenterAddr))
			req.CtlServers = append(req.CtlServers, ss)
		} else {
			req.Args = append(req.Args, fmt.Sprintf("/usr/bin/svn"))
			req.Args = append(req.Args, fmt.Sprintf("update"))
			req.Args = append(req.Args, fmt.Sprintf(s.Name))
		}

		//TODO 暂时保留,后续可能会用到
		//conn, err := grpc.Dial(appRegData[key].rpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		//if err != nil {
		//	return fmt.Errorf("did not connect: err=%v", err)
		//}
		//defer conn.Close()
		//c := center.NewAppControlClient(conn)
		//
		//// Contact the server and print out its response.
		//ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		//defer cancel()
		//r, err := c.ControlReq(ctx, &req)
		//if err != nil {
		//	return fmt.Errorf("could not greet: %v", err)
		//}
		//
		//log.Debug("", "r=%v", r)

		appRegData[key].a.SendData(n.AppCenter, uint32(center.CMDCenter_IDAppControlReq), &req)
	default:
		log.Warning("", "bad command type=%v,id=%v,ctlId=%v", appType, appId, ctlId)
		return fmt.Errorf("bad command type=%v,id=%v,ctlId=%v", appType, appId, ctlId)
	}

	return nil
}
