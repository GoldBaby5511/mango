package sqlhelper

import (
	"mango/pkg/log"
	"mango/pkg/util/errorhelper"
	"database/sql"
	"errors"
	"github.com/GoldBaby5511/go-simplejson"
	_ "github.com/go-sql-driver/mysql"
	"sync"
	"time"
)

type SqlHelper struct {
	db    *sql.DB
	stmts map[string]*sql.Stmt
	sync.Mutex
}

type SqlProc struct {
	Name   string
	Params []interface{}
}

func (sh *SqlHelper) Init(DataBase *simplejson.Json) {
	server := DataBase.Get("server").MustString()
	port := DataBase.Get("port").MustString()
	database := DataBase.Get("database").MustString()
	userid := DataBase.Get("userid").MustString()
	password := DataBase.Get("password").MustString()
	maxOpenConnects := DataBase.Get("maxOpenConnects").MustInt()
	maxIdleConnects := DataBase.Get("maxIdleConnects").MustInt()
	connMaxLifeTime := DataBase.Get("connMaxLifeTime").MustInt()
	connMaxIdleTime := DataBase.Get("connMaxIdleTime").MustInt()
	driver, _ := DataBase.Get("driver").String()
	if len(driver) == 0 {
		driver = "mysql"
	}

	var dsn string
	switch driver {
	case "mssql":
		//dsn:= fmt.Sprintf("server=%s;user id=%s;password=%s;database=%s;port=%d;encrypt=disable", *server, *userid, *password,*database, *port)
		dsn = "server=" + server + ";user id=" + userid + ";password=" + password + ";database=" + database + ";port=" + port + ";encrypt=disable"
	case "mysql":
		//user:password@tcp(localhost:5555)/dbname?charset=utf8
		dsn = userid + ":" + password + "@tcp(" + server + ":" + port + ")/" + database + "?charset=utf8mb4,utf8"
	}

	var err error
	sh.db, err = sql.Open(driver, dsn)
	if err != nil {
		log.Fatal("SqlHelper", "sqlHelper.sql.Open,err=%v", err)
		return
	}
	sh.db.SetMaxOpenConns(maxOpenConnects)
	sh.db.SetMaxIdleConns(maxIdleConnects)
	sh.db.SetConnMaxLifetime(time.Duration(connMaxLifeTime) * time.Millisecond)
	sh.db.SetConnMaxIdleTime(time.Duration(connMaxIdleTime) * time.Millisecond)

	err = sh.db.Ping()
	if err != nil {
		log.Fatal("SqlHelper", "sqlHelper.sh.db.Ping,err=%v", err)
		return
	}
	sh.stmts = make(map[string]*sql.Stmt)
	log.Info("SqlHelper", "初始化完成,driver=%v,dsn=%v", driver, dsn)
}

func (sh *SqlHelper) ExecGetResult(sqlStatement string, params ...interface{}) (*DataResult, error) {
	stmt, err := sh.GetStmt(sqlStatement, nil)
	if err != nil {
		log.Error("", "ExecGetResult error:%s %s %s", sqlStatement, params, err)
		return nil, err
	}
	return sh.TransactionExecGetResult(nil, stmt, sqlStatement, params...)
}

func (sh *SqlHelper) TransactionExecGetResult(tx *sql.Tx, stmt *sql.Stmt, sqlStatement string, params ...interface{}) (*DataResult, error) {
	defer errorhelper.Recover()
	if sh.db == nil {
		return nil, errors.New("db is null")
	}

	beginTime := time.Now().Unix()
	var err error
	var rows *sql.Rows
	if stmt == nil && tx != nil {
		stmt, err = sh.GetStmt(sqlStatement, tx)
		if err != nil {
			log.Warning("", "TransactionExecGetResult转换失败,sqlStatement=%s,params=%v,err=%s", sqlStatement, params, err)
			return nil, err
		}
		if tx != nil {
			defer stmt.Close()
		}
	}

	if stmt != nil {
		rows, err = stmt.Query(params...)
	} else {
		rows, err = sh.db.Query(sqlStatement, params...)
	}

	if err != nil {
		log.Warning("", "err=%v,sqlStatement=%v,params=%v", err, sqlStatement, params)
		return nil, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		log.Warning("", "rows.Columns err=%v,sqlStatement=%v,params=%v", err, sqlStatement, params)
		return nil, err
	}
	if cols == nil {
		log.Warning("", "cols is null err=%v,sqlStatement=%v,params=%v", err, sqlStatement, params)
		return nil, errors.New("cols is null")
	}
	lenCols := len(cols)
	result := &DataResult{
		Rows: make([]interface{}, 0),
	}
	for rows.Next() {
		rowValues := make([]interface{}, lenCols)
		for i := 0; i < lenCols; i++ {
			rowValues[i] = new(interface{})
		}
		err = rows.Scan(rowValues...)
		if err != nil {
			log.Warning("", "err=%v,sqlStatement=%v,params=%v", err, sqlStatement, params)
			continue
		}

		result.Rows = append(result.Rows, rowValues)
	}
	result.RowCount = len(result.Rows)
	if rows.Err() != nil {
		log.Warning("", "rows.Err=%v,sqlStatement=%v,params=%v", rows.Err(), sqlStatement, params)
		return nil, rows.Err()
	}
	log.Debug("", "sql执行时间,time=%v,sqlStatement=%v,params=%v", time.Now().Unix()-beginTime, sqlStatement, params)
	return result, nil
}

func (sh *SqlHelper) Exec(sqlStatement string, params ...interface{}) (sql.Result, error) {
	stmt, err := sh.GetStmt(sqlStatement, nil)
	if err != nil {
		log.Error("", "Exec失败,sqlStatement=%s,params=%v,err=%s", sqlStatement, params, err)
		return nil, err
	}
	return sh.TransactionExec(nil, stmt, sqlStatement, params...)
}

func (sh *SqlHelper) TransactionExec(tx *sql.Tx, stmt *sql.Stmt, sqlStatement string, params ...interface{}) (sql.Result, error) {
	defer errorhelper.Recover()
	if sh.db == nil {
		return nil, errors.New("db is null")
	}
	log.Debug("", "执行sql,sqlStatement=%v,params=%v", sqlStatement, params)
	var err error
	var result sql.Result
	if stmt == nil && tx != nil {
		stmt, err = sh.GetStmt(sqlStatement, tx)
		if err != nil {
			log.Warning("", "TransactionExec转换失败, sqlStatement=%s,params=%v,err=%s", sqlStatement, params, err)
			return nil, err
		}
		defer stmt.Close()
	}

	if stmt != nil {
		result, err = stmt.Exec(params...)
	} else {
		result, err = sh.db.Exec(sqlStatement, params...)
	}

	if err != nil {
		log.Error("", "TransactionExec执行失败,err=%v,sqlStatement=%v,params=%v", err, sqlStatement, params)
		return nil, err
	}
	return result, nil
}

func (sh *SqlHelper) GetBegin() *sql.Tx {
	tx, err := sh.db.Begin()
	if err != nil {
		log.Error("", "GetBegin err=%s", err)
		return nil
	}
	return tx
}

func (sh *SqlHelper) AddStmts(sqlStatement string) {
	defer sh.Unlock()
	sh.Lock()

	defer errorhelper.Recover()
	if sh.db == nil {
		return
	}
	var stmt *sql.Stmt
	stmt = sh.stmts[sqlStatement]
	if stmt == nil {
		var err error
		stmt, err = sh.db.Prepare(sqlStatement)
		if err != nil {
			return
		}
		sh.stmts[sqlStatement] = stmt
	}
}

func (sh *SqlHelper) GetStmt(sqlStatement string, tx *sql.Tx) (*sql.Stmt, error) {
	defer errorhelper.Recover()
	defer sh.Unlock()
	sh.Lock()

	if sh.db == nil {
		return nil, errors.New("db is null")
	}
	if tx != nil {
		return tx.Prepare(sqlStatement)
	}
	stmt := sh.stmts[sqlStatement]
	if stmt == nil {
		var err error
		stmt, err = sh.db.Prepare(sqlStatement)
		if err != nil {
			return nil, err
		}
		sh.stmts[sqlStatement] = stmt
	}
	return stmt, nil
}

func (sh *SqlHelper) Close() {
	defer errorhelper.Recover()
	defer sh.Unlock()
	sh.Lock()

	if sh.stmts != nil {
		for key, stmt := range sh.stmts {
			delete(sh.stmts, key)
			stmt.Close()
		}
	}

	if sh.db != nil {
		sh.db.Close()
	}
}
