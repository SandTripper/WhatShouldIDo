package handler

import (
	"WhatShouldIDo/config"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

var requestLimitPerSecond = config.RequestLimitPerSecond //同一ip每秒最多请求次数

var requestLimitPerDay = config.RequestLimitPerDay //同一ip每天最多请求次数

var ip_datas map[string]map[string]uint
var ip_datas_rwlock sync.Mutex

func init() {
	ip_datas = make(map[string]map[string]uint)
}

// 中间件，记录访问日志
func RecordAccessLog(c *Context) {
	log.Infof("(ip:%s) %s URI:(%s)", c.IpAddress, c.Req.Method, c.Req.RequestURI)
}

// 检查并限制访问次数
func RequestRateLimit(c *Context) {
	ip_datas_rwlock.Lock()
	defer ip_datas_rwlock.Unlock()
	currentTime := uint(time.Now().Unix())
	if _, ok := ip_datas[c.IpAddress]; !ok {
		ip_datas[c.IpAddress] = make(map[string]uint)
		ip_datas[c.IpAddress]["last_second"] = 0
		ip_datas[c.IpAddress]["times_in_second"] = 0
		ip_datas[c.IpAddress]["last_day"] = 0
		ip_datas[c.IpAddress]["times_in_day"] = 0
	}
	if ip_datas[c.IpAddress]["last_second"] == currentTime {
		if ip_datas[c.IpAddress]["times_in_second"] >= requestLimitPerSecond {
			c.IsContinue = false
		} else {
			ip_datas[c.IpAddress]["times_in_second"] += 1
		}

	} else {
		ip_datas[c.IpAddress]["last_second"] = currentTime
		ip_datas[c.IpAddress]["times_in_second"] = 0
	}

	if ip_datas[c.IpAddress]["last_day"] == currentTime/3600/24 {
		if ip_datas[c.IpAddress]["times_in_day"] >= requestLimitPerDay {
			c.IsContinue = false
		} else {
			ip_datas[c.IpAddress]["times_in_day"] += 1
		}
	} else {
		ip_datas[c.IpAddress]["last_day"] = currentTime / 3600 / 24
		ip_datas[c.IpAddress]["times_in_day"] = 0
	}
}

// 解析来源IP地址
func ParseIpAddress(c *Context) {
	c.IpAddress = c.Req.Header.Get("X-Real-IP")
}
