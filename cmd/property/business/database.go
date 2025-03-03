package business

import (
	"errors"
	"fmt"
	"mango/api/property"
	"mango/api/types"
	"mango/pkg/database"
	"mango/pkg/log"
)

var (
	newUserSendScore int64 = 100000000 //一个小目标
)

// 查询财富
func dbQueryProperty(userId uint64) ([]*types.PropItem, error) {
	db := database.GetMasterSqlDB()
	if db == nil {
		err := fmt.Sprintf("服务端异常,数据库连接失败,uId=%v", userId)
		log.Error("", "异常,登录的时候数据连接没了,err=%v", err)
		return nil, errors.New("异常,登录的时候数据连接没了")
	}

	sql := fmt.Sprintf("SELECT user_id,score,ingot,red_packet FROM game_score_info WHERE user_id = %v ORDER BY user_id DESC LIMIT 1;", userId)
	r, err := db.ExecGetResult(sql)
	if err != nil {
		log.Error("", "数据库执行错误,sql=%v,err=%v", sql, err)
		return nil, err
	}

	ps := make([]*types.PropItem, 0)
	if r.RowCount > 0 {
		//获取账号信息
		userId = r.GetUInt64Value(0, 0)
		ps = append(ps, &types.PropItem{
			Id:    types.PropItem_coin,
			Count: r.GetInt64Value(0, 1),
		})
		ps = append(ps, &types.PropItem{
			Id:    types.PropItem_ingot,
			Count: r.GetInt64Value(0, 2),
		})
		ps = append(ps, &types.PropItem{
			Id:    types.PropItem_red_packet,
			Count: r.GetInt64Value(0, 3),
		})
	} else {
		//不存在则当新用户处理
		sql = fmt.Sprintf("INSERT INTO game_score_info (user_id,score,ingot,red_packet) VALUES (%v,%v,%v,%v);", userId, newUserSendScore, 0, 0)
		r, err := db.Exec(sql)
		if err != nil {
			log.Error("", "数据库执行错误,sql=%v,err=%v", sql, err)
			return nil, err
		}
		log.Info("", "新用户,sql=%v,r=%v", sql, r)

		ps = append(ps, &types.PropItem{
			Id:    types.PropItem_coin,
			Count: newUserSendScore,
		})
	}

	log.Info("", "查询财富,userId=%v,ps=%v", userId, ps)

	return ps, nil
}

// 修改积分
func dbWriteGameScore(m *property.WriteGameScoreReq) error {
	userId := m.UserId
	db := database.GetMasterSqlDB()
	if db == nil {
		err := fmt.Sprintf("服务端异常,数据库连接失败,uId=%v", userId)
		log.Error("", "异常,登录的时候数据连接没了,err=%v", err)
		return errors.New("异常,数据连接没了")
	}

	sql := fmt.Sprintf("SELECT user_id,score,ingot,red_packet FROM game_score_info WHERE user_id = %v ORDER BY user_id DESC LIMIT 1;", userId)
	dr, err := db.ExecGetResult(sql)
	if err != nil {
		log.Error("", "数据库执行错误,sql=%v,err=%v", sql, err)
		return err
	}
	if dr.RowCount == 0 {
		log.Error("", "数据库执行错误,sql=%v,err=%v", sql, err)
		return errors.New("数据库执行错误")
	}

	//更新积分
	sql = fmt.Sprintf("UPDATE game_score_info SET score = score + %v,ingot = ingot+ %v,Revenue = Revenue+%v WHERE user_id = %v;",
		m.VariationInfo.Score, m.VariationInfo.Ingot, m.VariationInfo.Revenue, userId)
	r, err := db.Exec(sql)
	if err != nil {
		log.Error("", "数据库执行错误,sql=%v,err=%v", sql, err)
		return err
	}

	log.Info("", "更新积分,sql=%v,r=%v", sql, r)

	return nil
}
