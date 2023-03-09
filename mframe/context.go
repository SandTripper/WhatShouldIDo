package mframe

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"text/template"
)

type Context struct {
	Writer     http.ResponseWriter
	Req        *http.Request
	Path       string            //请求的路径
	Method     string            //请求模式
	StatusCode int               //回复的状态码
	Params     map[string]string //存储动态路由匹配的表单
	handlers   []HandlerFunc     //需要执行的中间件
	index      int               //当前执行的中间件的序号
	IsContinue bool              //是否继续执行剩余handler
	IpAddress  string            //源IP地址
}

// 新建一个Context
func newContext(writer http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Writer:     writer,
		Req:        req,
		Path:       req.URL.Path,
		Method:     req.Method,
		index:      -1,
		IsContinue: true,
	}
}

// 中间件执行过程中调用以先执行其余中间件和逻辑函数
func (c *Context) Next() {
	c.index++
	s := len(c.handlers)
	for ; c.index < s && c.IsContinue; c.index++ {
		c.handlers[c.index](c)
	}
}

// 获取动态匹配后某个键对应的值
func (c *Context) Param(key string) string {
	value := c.Params[key]
	return value
}

// 返回请求的表单中key对应的值
func (c *Context) PostForm(key string) string {
	return c.Req.FormValue(key)
}

// 返回在URL中的表单中key的值
func (c *Context) Query(key string) string {
	return c.Req.URL.Query().Get(key)
}

// 设置返回状态
func (c *Context) Status(code int) {
	c.StatusCode = code
	c.Writer.WriteHeader(code)
}

// 往返回头添加键值对
func (c *Context) SetHeader(key string, value string) {
	c.Writer.Header().Set(key, value)
}

// 返回字符串
func (c *Context) String(code int, format string, values ...interface{}) {
	c.SetHeader("Content-Type", "text/plain")
	c.Status(code)
	c.Writer.Write([]byte(fmt.Sprintf(format, values...)))
}

// 返回json文件
func (c *Context) JSON(code int, obj interface{}) {
	c.SetHeader("Content-Type", "application/json")
	c.Status(code)
	encoder := json.NewEncoder(c.Writer)
	if err := encoder.Encode(obj); err != nil {
		http.Error(c.Writer, err.Error(), 500)
	}
}

// 返回文件
func (c *Context) File(code int, path string) {
	c.SetHeader("Content-Type", "application/octet-stream")
	c.Status(code)
	http.ServeFile(c.Writer, c.Req, path)
}

// 返回二进制数组
func (c *Context) Data(code int, data []byte) {
	c.Status(code)
	c.Writer.Write(data)
}

// 返回html字符串
func (c *Context) HTML(code int, html string) {
	c.SetHeader("Content-Type", "text/html")
	c.Status(code)
	c.Writer.Write([]byte(html))
}

// 返回html文件
func (c *Context) HTMLF(code int, filepath string) error {
	text, err := os.ReadFile(filepath)
	if err != nil {
		return errors.New("failed to read file")
	}
	c.SetHeader("Content-Type", "text/html")
	c.Status(code)
	c.Writer.Write(text)
	return nil
}

// 返回进行模板匹配后的html文件
func (c *Context) HTMLFT(code int, filepath string, data interface{}) error {
	t1, err := template.ParseFiles(filepath)
	if err != nil {
		return errors.New("failed to read file")
	}
	c.SetHeader("Content-Type", "text/html")
	c.Status(code)
	t1.Execute(c.Writer, data)
	return nil
}
