package config

import (
	"encoding/json"
	"os"

	log "github.com/sirupsen/logrus"
)

var RequestLimitPerSecond int //同一ip每秒最多请求次数

var RequestLimitPerDay int //同一ip每天最多请求次数

var DataLoginUsername string //数据库登录用户名

var DataLoginPassword string //数据库登录密码

var KeyCacheMaxSize int //查询缓存条数

var SessionCacheMaxSize int //用户缓存条数

type configData struct {
	RequestLimitPerSecond int

	RequestLimitPerDay int

	DataLoginUsername string

	DataLoginPassword string

	KeyCacheMaxSize int

	SessionCacheMaxSize int
}

func init() {
	var cd configData
	byteValue, err := os.ReadFile("config.json")
	if err != nil {
		log.Panic(err)
	}
	json.Unmarshal([]byte(byteValue), &cd)

	RequestLimitPerSecond = cd.RequestLimitPerSecond
	RequestLimitPerDay = cd.RequestLimitPerDay
	DataLoginUsername = cd.DataLoginUsername
	DataLoginPassword = cd.DataLoginPassword
	KeyCacheMaxSize = cd.KeyCacheMaxSize
	SessionCacheMaxSize = cd.SessionCacheMaxSize
}
