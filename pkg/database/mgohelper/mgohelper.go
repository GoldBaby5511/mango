package mgohelper

import (
	"mango/pkg/log"
	"gopkg.in/mgo.v2"
)

type MgoHelper struct {
	server *mgo.Session
	db     *mgo.Database
}

func (mh *MgoHelper) Init(server, database, username, pwd string) error {
	var err error
	mh.server, err = mgo.Dial(server)
	if err != nil {
		log.Error("", "MgoHelper Init:err=%v", err)
		return err
	}
	mh.server.SetMode(mgo.Monotonic, true)
	mh.db = mh.server.DB(database)
	if len(username) > 0 && len(pwd) > 0 {
		err := mh.db.Login(username, pwd)
		if err != nil {
			log.Error("", "MgoHelper InitDB:err=%v", err)
			return err
		}
	}

	log.Info("MgoHelper", "mongodb连接完成,server=%v,database=%v, username=%v, pwd=%v", server, database, username, pwd)

	return nil
}

func (mh *MgoHelper) GetDB() *mgo.Database {
	return mh.db
}

func (mh *MgoHelper) Ping() error {
	return mh.server.Ping()
}

func (mh *MgoHelper) Close(server string) {
	if mh.server == nil {
		return
	}
	mh.server.Close()
}
