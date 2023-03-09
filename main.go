package main

import (
	"WhatShouldIDo/handler"
	"WhatShouldIDo/mframe"
	"fmt"
	"path"
	"path/filepath"
	"runtime"
	"time"

	nested "github.com/antonfisher/nested-logrus-formatter"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// 配置日志文件切割
func configLocalFileSystemLogger(logPath string, logFileName string, rotationTime time.Duration, leastFile uint) {
	baseLogPath := path.Join(logPath, logFileName)
	writer, err := rotatelogs.New(
		baseLogPath+"-%Y-%m-%d.log",
		rotatelogs.WithRotationTime(rotationTime), // 日志切割时间间隔
		rotatelogs.WithRotationCount(leastFile),   // 保留的日志数量
	)
	if err != nil {
		log.Errorf("config local file system logger error. %+v", errors.WithStack(err))
	}
	log.SetOutput(writer)
}

// 配置日志系统
func configLogger() {
	configLocalFileSystemLogger("./logs", "ServerLog", time.Hour*24, 3)

	//配置格式
	log.SetFormatter(&nested.Formatter{
		TimestampFormat: "2006-01-02 15:04:05",
		CustomCallerFormatter: func(frame *runtime.Frame) string {
			funcInfo := runtime.FuncForPC(frame.PC)
			if funcInfo == nil {
				return "error during runtime.FuncForPC"
			}
			fullPath, line := funcInfo.FileLine(frame.PC)
			return fmt.Sprintf(" [%v:%v]", filepath.Base(fullPath), line)
		},
		NoColors: true,
	})
	//日志中显示文件名和行数
	log.SetReportCaller(true)
}

func main() {
	//配置日志系统
	configLogger()

	engine := mframe.NewEngine()
	engine.Use(handler.ParseIpAddress)
	engine.Use(handler.RecordAccessLog)
	engine.Use(handler.RequestRateLimit)

	wsid := engine.Group("WSID")
	wsid.GET("/", handler.ShowIndexPage)
	wsid.GET("/next", handler.Next)
	wsid.GET("/search", handler.ShowResultPage)
	wsid.GET("/image/*imagename", handler.ShowImage)

	fmt.Print("server start at port 8888\n")
	engine.Run(":8888")
	// engine.RunTLS() //https
}
