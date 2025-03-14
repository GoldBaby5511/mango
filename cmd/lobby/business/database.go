package business

import (
	"errors"
	"fmt"
	"mango/api/lobby"
	"mango/api/types"
	"mango/pkg/database"
	"mango/pkg/log"
)

func dbUserLogin(connId uint64, m *lobby.LoginReq) (uint64, error) {
	var userId uint64 = 0
	db := database.GetMasterSqlDB()
	if db == nil {
		err := fmt.Sprintf("服务端异常,数据库连接失败,uId=%v", userId)
		respondUserLogin(userId, connId, int32(lobby.LoginRsp_SERVERERROR), err)
		log.Error("", "异常,登录的时候数据连接没了,err=%v", err)
		return 0, errors.New("异常,登录的时候数据连接没了")
	}

	//创建用户
	u := &types.BaseUserInfo{
		Account:    m.GetAccount(),
		UserId:     userId,
		GateConnId: connId,
	}

	//根据登录方式
	switch m.GetLoginType() {
	case lobby.LoginReq_acc:
		sql := fmt.Sprintf("SELECT user_id,game_id,nick_name FROM account WHERE account = \"%v\" ORDER BY user_id DESC LIMIT 1;", m.GetAccount())
		r, err := db.ExecGetResult(sql)
		if err != nil {
			errInfo := fmt.Sprintf("数据库执行错误,sql=%v,err=%v", sql, err)
			respondUserLogin(userId, connId, int32(lobby.LoginRsp_SERVERERROR), errInfo)
			return 0, err
		}

		if r.RowCount == 0 {
			respondUserLogin(userId, connId, int32(lobby.LoginRsp_SERVERERROR), "账号不存在")
			return 0, err
		}

		//获取账号信息
		userId = r.GetUInt64Value(0, 0)
		u.UserId = userId
		u.GameId = r.GetUInt64Value(0, 1)
		u.NickName = r.GetStringValue(0, 2)
	case lobby.LoginReq_unionId: //游客登录
		sql := fmt.Sprintf("SELECT user_id,game_id,nick_name FROM account WHERE account_machine = \"%v\" ORDER BY user_id DESC LIMIT 1;", m.GetMachineId())
		r, err := db.ExecGetResult(sql)
		if err != nil {
			errInfo := fmt.Sprintf("数据库执行错误,sql=%v,err=%v", sql, err)
			respondUserLogin(userId, connId, int32(lobby.LoginRsp_SERVERERROR), errInfo)
			return 0, err
		}

		//不存在就创建一个
		if r.RowCount == 0 {
			sql = fmt.Sprintf("INSERT INTO account (account_machine,register_machine) VALUES (\"%v\",\"%v\");", m.GetMachineId(), m.GetMachineId())
			_, err = db.Exec(sql)
			if err != nil {
				log.Error("", "数据库执行错误,sql=%v,err=%v", sql, err)
				respondUserLogin(userId, connId, int32(lobby.LoginRsp_SERVERERROR), "账号不存在")
				return 0, err
			}

			//更新game_id
			sql = fmt.Sprintf("UPDATE account SET game_id=user_id+100000 WHERE account_machine = \"%v\";", m.GetMachineId())
			_, err = db.Exec(sql)
			if err != nil {
				log.Error("", "数据库执行错误,sql=%v,err=%v", sql, err)
				respondUserLogin(userId, connId, int32(lobby.LoginRsp_SERVERERROR), "账号不存在")
				return 0, err
			}
			//重新查询
			sql = fmt.Sprintf("SELECT user_id,game_id,nick_name FROM account WHERE account_machine = \"%v\" ORDER BY user_id DESC LIMIT 1;", m.GetMachineId())
			r, err = db.ExecGetResult(sql)
			if err != nil {
				errInfo := fmt.Sprintf("数据库执行错误,sql=%v,err=%v", sql, err)
				respondUserLogin(userId, connId, int32(lobby.LoginRsp_SERVERERROR), errInfo)
				return 0, err
			}
		}

		//获取账号信息
		userId = r.GetUInt64Value(0, 0)
		u.UserId = userId
		u.GameId = r.GetUInt64Value(0, 1)
		u.NickName = r.GetStringValue(0, 2)
	default:
		log.Warning("", "暂时没有处理其他登录情况,uId=%v,t=%v", userId, m.GetLoginType())
		respondUserLogin(userId, connId, int32(lobby.LoginRsp_SERVERERROR), "登录方式有误")
		return 0, errors.New("暂时没有处理其他登录情况")
	}

	//登录成功
	userList[userId] = u

	return userId, nil
}
