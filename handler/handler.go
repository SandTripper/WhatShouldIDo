package handler

import (
	"WhatShouldIDo/config"
	"WhatShouldIDo/mcache"
	"WhatShouldIDo/mframe"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
)

type Context = mframe.Context

var GlobalDb *sql.DB
var GlobalDbLock sync.Mutex
var KeyCache *mcache.LRUCache
var SessionCache *mcache.LRUCache

type Statistical struct {
	QueryTimes int64 `json:"QueryTimes"`
}

var StatisticalData Statistical

func init() {
	// 实例化键缓存
	KeyCache = mcache.NewCache(config.KeyCacheMaxSize)

	// 实例化查询缓存
	SessionCache = mcache.NewCache(config.SessionCacheMaxSize)

	//连接数据库
	var err error
	GlobalDb, err = sql.Open("mysql", config.DataLoginUsername+":"+config.DataLoginPassword+"@/what_should_i_do?charset=utf8")
	checkError(err)

	// 打开统计json文件
	statistical_data_file, err := os.Open("statistical_data.json")
	if err != nil {
		log.Panic(err)
	}
	defer statistical_data_file.Close()
	byteValue, _ := ioutil.ReadAll(statistical_data_file)
	json.Unmarshal([]byte(byteValue), &StatisticalData) //解析json文件
	// 每1分钟执行一次统计数据持久化
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			b, _ := json.Marshal(StatisticalData)
			file, err := os.OpenFile("statistical_data.json", os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
			if err != nil {
				log.Panic(err)
			}
			file.Write(b)
			file.Close()
		}
	}()
}

// 显示首页
func ShowIndexPage(c *Context) {
	err := c.HTMLF(http.StatusOK, "root/index.html")
	checkServerUnavailableErr(c, err)
}

// 显示查询结果页
func ShowResultPage(c *Context) {
	key := url.QueryEscape(c.PostForm("key"))
	var queryid uint64
	for {
		queryid = rand.Uint64()
		if SessionCache.Get(fmt.Sprint(queryid)) == nil {
			break
		}
	}
	err := c.HTMLFT(http.StatusOK, "root/result.html", struct {
		Key     string
		Queryid string
	}{key, fmt.Sprint(queryid)})

	checkServerUnavailableErr(c, err)
}

// 显示下一条信息
func Next(c *Context) {
	atomic.AddInt64(&StatisticalData.QueryTimes, 1)

	data, err := doNext(c)

	if err != nil { //出现错误
		data = make(map[string]interface{})
		data["status"] = "none"
		data["queryTimes"] = StatisticalData.QueryTimes
		c.JSON(http.StatusOK, data)
		log.Error(err)
		return
	}
	if data == nil { //没有数据
		data = make(map[string]interface{})
		data["status"] = "none"
		data["queryTimes"] = StatisticalData.QueryTimes
		c.JSON(http.StatusOK, data)
		return
	}

	data["queryTimes"] = StatisticalData.QueryTimes
	c.JSON(http.StatusOK, data) //返回正常数据
}

// 返回图片
func ShowImage(c *Context) {
	filepath := "root/images/" + c.Param("imagename")
	if fileExists(filepath) {
		data, err := os.ReadFile(filepath)
		checkError(err)
		c.Data(http.StatusOK, data)
	} else {
		c.String(http.StatusNotFound, "404 not found")
	}
}

// 进行数据的查询工作，返回查询到的值，无结果返回nil
func doNext(c *Context) (map[string]interface{}, error) {
	key := c.PostForm("key")
	key = strings.ToLower(key)

	var lst []interface{} //存储查询到的结果集

	GlobalDbLock.Lock()
	defer GlobalDbLock.Unlock()

	res := KeyCache.Get(key)

	if res != nil { //缓存中存在
		lst = res.([]interface{})
	} else { //缓存中不存在

		rows, err := GlobalDb.Query(`SELECT id FROM job_information_tb WHERE post_name like ? `, "%"+key+"%") //从数据库中查询所有匹配数据

		if err != nil {
			return nil, err
		}

		defer rows.Close()

		lst = make([]interface{}, 0)
		for rows.Next() { //提取所有匹配的id存入lst
			var id int
			err = rows.Scan(&id)

			if err != nil {
				return nil, err
			}
			lst = append(lst, id)
		}

		KeyCache.Replace(key, lst) //存入缓存
	}

	if len(lst) <= 0 { //无匹配结果
		return nil, nil
	}

	queryid := c.PostForm("queryid")
	if queryid == "" { //不带queryid，非法查询
		return nil, errors.New("invalid query")
	}

	var nowIndex, step, totQuery int
	var sessionData map[string]int

	res1 := SessionCache.Get(queryid)
	if res1 == nil { //不存在session，创建

		sessionData = make(map[string]int)

		//随机生成nowIndex和step
		nowIndex = rand.Int() % len(lst)
		step = findCoprimeNumber(len(lst))
		totQuery = 0

		sessionData["nowIndex"] = nowIndex
		sessionData["step"] = step
		sessionData["totQuery"] = totQuery

		//存入缓存
		SessionCache.Replace(queryid, sessionData)
	} else { //存在session，读取数据
		sessionData = res1.(map[string]int)

		nowIndex = sessionData["nowIndex"]
		step = sessionData["step"]
		totQuery = sessionData["totQuery"]
	}

	if totQuery >= len(lst) { //已全部读完
		return nil, nil
	}

	nowIndex = (nowIndex + step) % len(lst) //更新步数
	totQuery += 1                           //更新获取次数

	//将更新写入缓存
	sessionData["nowIndex"] = nowIndex
	sessionData["totQuery"] = totQuery

	var recruitment_unit, post_name, require_text string

	err := GlobalDb.QueryRow(`SELECT recruitment_unit,post_name,require_text FROM job_information_tb WHERE id = ?`, lst[nowIndex]).Scan(&recruitment_unit, &post_name, &require_text)

	if err != nil { //查询出现错误
		return nil, err
	}
	data := make(map[string]interface{})
	data["status"] = "ok"
	data["company"] = recruitment_unit
	data["post_name"] = post_name
	data["require_text"] = require_text
	return data, nil
}

// 求gcd
func gcd(a, b uint) uint {
	if b > 0 {
		return gcd(b, a%b)
	}
	return a
}

func findCoprimeNumber(x int) int {
	for {
		res := rand.Int()%x + 1
		if gcd(uint(x), uint(res)) == 1 {
			return res
		}
	}
}

// 判断所给路径文件/文件夹是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	return !os.IsNotExist(err)
}

// 检查是否出现错误，若出错，向浏览器返回503，并返回true
func checkServerUnavailableErr(c *Context, err error) bool {
	if err != nil {
		log.Error(err)
		serviceUnavailable(c) //向客户端返回服务器错误
		return true
	}
	return false
}

func serviceUnavailable(c *Context) {
	c.String(http.StatusServiceUnavailable, "Service Unavailable")
}

// 如果出现错误，panic
func checkError(err error) {
	if err != nil {
		log.Panic(err)
	}
}
