package handler

import (
	"WhatShouldIDo/config"
	"WhatShouldIDo/mframe"
	"database/sql"
	"net/http"
	"os"
	"sync"

	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
)

type Context = mframe.Context

var GlobalDb *sql.DB
var GlobalDbLock sync.Mutex

func init() {
	var err error
	GlobalDb, err = sql.Open("mysql", config.DataLoginUsername+":"+config.DataLoginUsername+"@/what_should_i_do?charset=utf8")
	checkError(err)
}

// 显示首页
func ShowIndexPage(c *Context) {
	err := c.HTMLF(http.StatusOK, "root/index.html")
	checkServerUnavailableErr(c, err)
}

// 显示查询结果页
func ShowResultPage(c *Context) {
	err := c.HTMLFT(http.StatusOK, "root/result.html", c.PostForm("key"))
	checkServerUnavailableErr(c, err)
}

// 显示查询结果页
func Next(c *Context) {
	GlobalDbLock.Lock()
	defer GlobalDbLock.Unlock()
	key := "%" + c.PostForm("key") + "%"
	var id, company, require_text, post_name string
	err := GlobalDb.QueryRow(`SELECT id,recruitment_unit,post_name,require_text FROM job_information_tb WHERE post_name like ?  and id >= (SELECT FLOOR( MAX(id) * RAND()) FROM job_information_tb WHERE post_name like ? ) ORDER BY id LIMIT 1;`, key, key).Scan(&id, &company, &post_name, &require_text)

	data := make(map[string]interface{})
	switch {
	case err == sql.ErrNoRows: //不存在结果
		data["status"] = "error"
		c.JSON(http.StatusOK, data)
		return
	case err != nil: //查询出现错误
		serviceUnavailable(c)
		log.Error(err)
		return
	default:
		data["status"] = "ok"
		data["id"] = id
		data["company"] = company
		data["post_name"] = post_name
		data["require_text"] = require_text
		c.JSON(http.StatusOK, data)
	}
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
