package business

import (
	"bytes"
	"fmt"
	"github.com/golang/protobuf/proto"
	"mango/api/center"
	"mango/pkg/conf"
	"mango/pkg/conf/apollo"
	g "mango/pkg/gate"
	"mango/pkg/log"
	n "mango/pkg/network"
	"mango/pkg/util"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

const (
	ParamError = 1
)

var (
	daemonServiceList   = make(map[uint64]*conf.BaseInfo)
	mxDaemonServiceList sync.Mutex
)

func init() {
	g.EventRegister(g.ConnectSuccess, connectSuccess)
	g.EventRegister(g.Disconnect, disconnect)

	//注册回调
	g.CallBackRegister(g.CbConfigChangeNotify, configChangeNotify)
	g.CallBackRegister(g.CbAppControlNotify, appControlNotify)
}

func connectSuccess(args []interface{}) {
}

func disconnect(args []interface{}) {
}

func configChangeNotify(args []interface{}) {
	key := args[apollo.KeyIndex].(apollo.ConfKey)
	value := args[apollo.ValueIndex].(apollo.ConfValue)

	switch key.Key {
	case "服务路径":
		pathName := value.Value
		if err := os.MkdirAll(pathName, os.ModePerm); err != nil {
			log.Error("", "异常,服务路径创建失败,appPath=%v", pathName)
		} else {
			log.Debug("", "服务路径,appPath=%v", pathName)
		}

		//serverPath := apollo.GetConfig("服务路径", "")
		//go func() {
		//	var ss center.ControlItem
		//	ss.Command = proto.String("cmd")
		//	ss.Args = append(ss.Args, fmt.Sprintf("/c"))
		//	ss.Args = append(ss.Args, fmt.Sprintf("login"))
		//	ss.Args = append(ss.Args, fmt.Sprintf("-Type=%v", 5))
		//	ss.Args = append(ss.Args, fmt.Sprintf("-Id=%v", 70))
		//	ss.Args = append(ss.Args, fmt.Sprintf("-ListenOnAddr=127.0.0.1:10070"))
		//	ss.Args = append(ss.Args, fmt.Sprintf("-CenterAddr=127.0.0.1:10050"))
		//	ss.Args = append(ss.Args, fmt.Sprintf("1>./log/log_login_%v.log", time.Now().Unix()))
		//	ss.Args = append(ss.Args, fmt.Sprintf("2>./log/err_login_%v.log", time.Now().Unix()))
		//
		//	cmd := exec.Command(ss.GetCommand(), ss.GetArgs()...)
		//	cmd.Dir = serverPath
		//	err := cmd.Run()
		//	if err != nil {
		//		log.Warning("", "Run失败了,dir=%v,c=%v,args=%v,err=%v", cmd.Path, ss.GetCommand(), ss.GetArgs(), err)
		//	}
		//}()

	default:
		break
	}
}

//控制消息
func appControlNotify(args []interface{}) {
	b := args[n.DataIndex].(n.BaseMessage)
	m := (b.MyMessage).(*center.AppControlReq)
	srcAgent := b.AgentInfo

	log.Debug("控制", "控制消息,AppType=%v,AppId=%v,CtlId=%v,type=%v,Id=%v,len=%v",
		srcAgent.AppType, srcAgent.AppId, m.GetCtlId(), m.GetAppType(), m.GetAppId(), len(m.GetCtlServers()))

	msgRespond := func(errCode int32, errInfo string) {
		log.Debug("", "控制回复,appId=%v,code=%v,errInfo=%v", srcAgent.AppId, errCode, errInfo)

		var rsp center.AppControlRsp
		rsp.CtlId = proto.Int32(m.GetCtlId())
		rsp.AppType = proto.Uint32(m.GetAppType())
		rsp.AppId = proto.Uint32(m.GetAppId())
		rsp.Code = proto.Int32(errCode)
		rsp.Info = proto.String(errInfo)
		g.SendData2App(n.AppCenter, srcAgent.AppId, n.AppCenter, uint32(center.CMDCenter_IDAppControlRsp), &rsp)
	}

	//来源判断
	if srcAgent.AppType != n.AppCenter {
		msgRespond(int32(ParamError), fmt.Sprintf("不支持类型,type=%v", srcAgent.AppType))
		return
	}

	errInfo := ""
	var errCode int32 = 0
	workDir := getWorkDir()
	if workDir == "" {
		log.Error("", "异常,服务路径为空")
		return
	}
	osType := runtime.GOOS
	switch m.GetCtlId() {
	case int32(center.CtlId_StartService):
		for _, item := range m.GetCtlServers() {
			bInfo := item
			key := util.MakeUint64FromUint32(bInfo.GetType(), bInfo.GetId())
			mxDaemonServiceList.Lock()
			daemonServiceList[key] = &conf.BaseInfo{
				Name: bInfo.GetName(),
				Type: bInfo.GetType(),
				Id:   bInfo.GetId(),
			}
			mxDaemonServiceList.Unlock()

			go func(args []string) {
				if len(args) == 0 {
					log.Error("", "异常,args为空")
					return
				}

				lInfo := ""
				if osType == "linux" {
					if err := os.Chmod(workDir+args[0], 0777); err != nil {
						log.Warning("", "修改文件权限失败,file=%v,err=%v", workDir+args[0], err)
					}
					cmd := exec.Command("./"+args[0], args[1:]...)
					fName := fmt.Sprintf("%v_%v_%v_%v.log", bInfo.GetName(), bInfo.GetType(), bInfo.GetId(), time.Now().Format("2006-01-02_15-04-05"))
					var err error
					cmd.Stdout, err = os.Create(conf.ApplogDir + "stdout_" + fName)
					if err != nil {
						log.Error("", "创建stdout文件失败,err=%v", err)
						return
					}

					cmd.Stderr, err = os.Create(conf.ApplogDir + "stderr_" + fName)
					if err != nil {
						log.Error("", "创建stderr文件失败,err=%v", err)
						return
					}

					cmd.Dir = workDir
					log.Debug("", "cmd=%v", cmd.String())
					err = cmd.Start()
					if err != nil {
						errInfo += err.Error()
						errCode = 1
						log.Warning("", "执行Start失败,err=%v,dir=%v,args=%v", err, workDir, cmd.Args)
						return
					}
					err = cmd.Wait()
					lInfo = fmt.Sprintf("结束执行,name=%v,type=%v,Id=%v,err=%v,dir=%v,args=%v",
						bInfo.GetName(), bInfo.GetType(), bInfo.GetId(), err, workDir, cmd.Args)
				} else if osType == "windows" {
					//TODO Windows下的命令
					//a := make([]string, 0, len(args)+1)
					//a = append(a, "/C")
					//a = append(a, args...)
					//cmd = exec.Command("cmd", a...)
				}

				mxDaemonServiceList.Lock()
				_, ok := daemonServiceList[util.MakeUint64FromUint32(bInfo.GetType(), bInfo.GetId())]
				mxDaemonServiceList.Unlock()
				if ok {
					log.Error("", lInfo)
				} else {
					log.Info("", lInfo)
				}

			}(item.GetArgs())
			if len(m.GetCtlServers()) > 1 {
				time.Sleep(time.Second)
			}
		}

		msgRespond(errCode, errInfo)
	case int32(center.CtlId_StopService):
		key := util.MakeUint64FromUint32(m.GetAppType(), m.GetAppId())
		mxDaemonServiceList.Lock()
		if _, ok := daemonServiceList[key]; ok {
			log.Debug("", "停止服务,key=%v,type=%v,Id=%v", key, m.GetAppType(), m.GetAppId())
			delete(daemonServiceList, key)
		} else {
			log.Warning("", "服务不存在,key=%v,type=%v,Id=%v", key, m.GetAppType(), m.GetAppId())
		}
		mxDaemonServiceList.Unlock()
	case int32(center.CtlId_UpdateService):
		//TODO 当前只支持linux
		if osType != "linux" {
			msgRespond(int32(ParamError), fmt.Sprintf("不支持的操作系统,osType=%v", osType))
			return
		}
		if len(m.GetArgs()) != 3 {
			log.Warning("", "参数错误,args=%v", m.GetArgs())
			return
		}
		args := m.GetArgs()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = workDir
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		log.Debug("", "cmd=%v", cmd.String())
		log.Debug("", "开始执行,stdout=%v,stderr=%v,dir=%v,args=%v", stdout.String(), stderr.String(), workDir, cmd.Args)
		err := cmd.Run()
		if err != nil {
			errInfo += err.Error()
			errCode = 1
		}
		if err := os.Chmod(workDir+args[2], 0777); err != nil {
			log.Warning("", "修改文件权限失败,file=%v,err=%v", workDir+args[2], err)
		}
		log.Debug("", "执行完成,err=%v,stdout=%v,stderr=%v,dir=%v,args=%v", err, stdout.String(), stderr.String(), workDir, cmd.Args)
	default:
		log.Warning("", "不支持的操作类型,ctlId=%v", m.GetCtlId())
	}
	msgRespond(errCode, errInfo)
}

func getWorkDir() string {
	workDir := ""
	if v, ok := util.ParseArgsString("-Path", os.Args); ok {
		workDir = v
	} else {
		workDir = apollo.GetConfig("服务路径", "")
	}

	return workDir
}
