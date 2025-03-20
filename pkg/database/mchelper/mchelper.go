package mchelper

import (
	"errors"
	"github.com/bradfitz/gomemcache/memcache"
)

type MCHelper struct {
	client *memcache.Client
}

func (mch *MCHelper) Init(server ...string) {
	mch.client = memcache.New(server...)
}

func (mch *MCHelper) Set(Key string, Value string) error {
	return mch.SetWithExp(Key, Value, 0)
}

func (mch *MCHelper) SetWithExp(Key string, Value string, Exp int32) error {
	if mch.client == nil {
		return errors.New("mch.client is null")
	}
	return mch.client.Set(&memcache.Item{Key: Key, Value: []byte(Value), Expiration: Exp})
}

func (mch *MCHelper) SetWithExpB(Key string, Value []byte, Exp int32) error {
	if mch.client == nil {
		return errors.New("mch.client is null")
	}
	return mch.client.Set(&memcache.Item{Key: Key, Value: Value, Expiration: Exp})
}

func (mch *MCHelper) Get(Key string) *memcache.Item {
	if mch.client == nil {
		return nil
	}
	item, err := mch.client.Get(Key)
	if err != nil {
		return nil
	}
	return item
}
