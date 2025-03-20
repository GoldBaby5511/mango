package redishelper

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	"mango/pkg/log"
	"mango/pkg/util/errorhelper"
	"sync"
	"time"
)

type RedisHelper struct {
	//client redis.Conn
	server string
	pwd    string
	pool   *redis.Pool
	sync.RWMutex
}

func (rh *RedisHelper) Init(server, pwd string) {
	rh.server = server
	rh.pwd = pwd
	rh.createPool()
	log.Info("RedisHelper", "初始化完成,server=%v,pwd=%v", server, pwd)
}

func (rh *RedisHelper) createPool() {
	rh.pool = &redis.Pool{
		MaxIdle:     1,
		IdleTimeout: 240 * time.Second,
		// Dial or DialContext must be set. When both are set, DialContext takes precedence over Dial.
		Dial: func() (redis.Conn, error) {
			var options []redis.DialOption
			if len(rh.pwd) > 0 {
				options = append(options, redis.DialPassword(rh.pwd))
			}
			return redis.Dial("tcp", rh.server, options...)
		},
	}
}

func (rh *RedisHelper) checkConnect(conn redis.Conn) error {
	defer rh.Unlock()
	rh.Lock()
	var err error
	if conn != nil {
		err = conn.Err()
		if err == nil {
			return nil
		}
	}

	var options []redis.DialOption
	if len(rh.pwd) > 0 {
		options = append(options, redis.DialPassword(rh.pwd))
	}
	conn, err = redis.Dial("tcp", rh.server, options...)
	if err != nil {
		log.Error("", "Connect to redis error,err=%v", err)
		return err
	}

	return nil
}

func (rh *RedisHelper) Set(Key string, Value interface{}) error {
	return rh.HSetWithExp(Key, "", Value, 0)
}

func (rh *RedisHelper) HSet(Key, Field string, Value interface{}) error {
	return rh.HSetWithExp(Key, Field, Value, 0)
}

func (rh *RedisHelper) HMSet(Key string, Values ...interface{}) error {
	defer errorhelper.Recover()
	client := rh.pool.Get()
	err := rh.checkConnect(client)
	if err != nil {
		return err
	}

	args := make([]interface{}, 1)
	args[0] = Key
	Values = append(args, Values...)
	_, err = client.Do("HMSET", Values...)

	if err != nil {
		fmt.Println("redis set failed:", err)
	}

	return err
}

func (rh *RedisHelper) HSetWithExp(Key, Field string, Value interface{}, Exp int32) error {
	defer errorhelper.Recover()
	client := rh.pool.Get()
	err := rh.checkConnect(client)
	if err != nil {
		return err
	}

	if len(Field) > 0 {
		if Exp > 0 {
			_, err = client.Do("HSET", Key, Field, Value, "EX", Exp)
		} else {
			_, err = client.Do("HSET", Key, Field, Value)
		}
	} else {
		if Exp > 0 {
			_, err = client.Do("SET", Key, Value, "EX", Exp)
		} else {
			_, err = client.Do("SET", Key, Value)
		}
	}

	if err != nil {
		fmt.Println("redis set failed:", err)
	}

	return err
}

func (rh *RedisHelper) ExpireAt(Key string, time int64) error {
	defer errorhelper.Recover()
	client := rh.pool.Get()
	err := rh.checkConnect(client)
	if err != nil {
		return err
	}
	_, err = client.Do("EXPIREAT", Key, time)
	if err != nil {
		fmt.Println("expireAt set failed:", err)
	}

	return err
}

func (rh *RedisHelper) SetWithExp(Key string, Value interface{}, Exp int32) error {
	defer errorhelper.Recover()
	client := rh.pool.Get()
	err := rh.checkConnect(client)
	if err != nil {
		return err
	}

	if Exp > 0 {
		_, err = client.Do("SET", Key, Value, "EX", Exp)
	} else {
		_, err = client.Do("SET", Key, Value)
	}

	if err != nil {
		fmt.Println("redis set failed:", err)
	}

	return err
}

func (rh *RedisHelper) Get(Key string) interface{} {
	defer errorhelper.Recover()
	client := rh.pool.Get()
	err := rh.checkConnect(client)
	if err != nil {
		return nil
	}

	value, err := client.Do("GET", Key)
	if err != nil {
		return nil
	}
	return value
}

func (rh *RedisHelper) HGet(Key, Field string) interface{} {
	defer errorhelper.Recover()
	client := rh.pool.Get()
	err := rh.checkConnect(client)
	if err != nil {
		return nil
	}

	value, err := client.Do("HGET", Key, Field)
	if err != nil {
		return nil
	}
	return value
}

func (rh *RedisHelper) HGetALL(Key string) interface{} {
	defer errorhelper.Recover()
	client := rh.pool.Get()
	err := rh.checkConnect(client)
	if err != nil {
		return nil
	}

	value, err := client.Do("HGETALL", Key)
	if err != nil {
		return nil
	}
	return value
}

func (rh *RedisHelper) GetString(Key string) string {
	defer errorhelper.Recover()
	str, _ := redis.String(rh.Get(Key), nil)
	return str
}

func (rh *RedisHelper) GetBytes(Key string) []byte {
	defer errorhelper.Recover()
	bytes, _ := redis.Bytes(rh.Get(Key), nil)
	return bytes
}

func (rh *RedisHelper) HGetString(Key, Field string) string {
	defer errorhelper.Recover()
	str, _ := redis.String(rh.HGet(Key, Field), nil)
	return str
}

func (rh *RedisHelper) HGetBytes(Key, Field string) []byte {
	defer errorhelper.Recover()
	bytes, _ := redis.Bytes(rh.HGet(Key, Field), nil)
	return bytes
}

func (rh *RedisHelper) HGetALLString(Key string) string {
	defer errorhelper.Recover()
	str, _ := redis.String(rh.HGetALL(Key), nil)
	return str
}

func (rh *RedisHelper) HGetALLBytes(Key string) []byte {
	defer errorhelper.Recover()
	bytes, _ := redis.Bytes(rh.HGetALL(Key), nil)
	return bytes
}

func (rh *RedisHelper) HGetALLStringMap(Key string) map[string]string {
	defer errorhelper.Recover()
	maps, _ := redis.StringMap(rh.HGetALL(Key), nil)
	return maps
}
