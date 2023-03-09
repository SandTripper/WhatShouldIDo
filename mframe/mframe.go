package mframe

import (
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

// 路由组
type RouterGroup struct {
	prefix      string        //组的路径
	middlewares []HandlerFunc // 该组需要执行的中间件
	parent      *RouterGroup  // 父组
	engine      *Engine       //所有组共享一个engine
}

// 定义处理函数类
type HandlerFunc func(*Context)

// Engine实现ServeHTTP接口，且作为最顶级的组，管理所有组
type Engine struct {
	*RouterGroup
	router *router
	groups []*RouterGroup // 所有组
}

func NewEngine() *Engine {
	engine := &Engine{router: newRouter()}
	engine.RouterGroup = &RouterGroup{engine: engine, prefix: "/"}
	engine.groups = []*RouterGroup{engine.RouterGroup}
	return engine
}

// 注册一个子组
func (group *RouterGroup) Group(prefix string) *RouterGroup {
	engine := group.engine
	newGroup := &RouterGroup{
		prefix: group.prefix + prefix,
		parent: group,
		engine: engine,
	}
	engine.groups = append(engine.groups, newGroup)
	return newGroup
}

func (group *RouterGroup) addRoute(method string, comp string, handler HandlerFunc) {
	pattern := group.prefix + comp
	log.Printf("Route %4s - %s", method, pattern)
	group.engine.router.addRoute(method, pattern, handler)
}

// 添加method为GET的对应请求路径的处理函数
func (group *RouterGroup) GET(pattern string, handler HandlerFunc) {
	group.addRoute("GET", pattern, handler)
}

// 添加method为POST的对应请求路径的处理函数
func (group *RouterGroup) POST(pattern string, handler HandlerFunc) {
	group.addRoute("POST", pattern, handler)
}

// 将中间件应用到该组
func (group *RouterGroup) Use(middlewares ...HandlerFunc) {
	group.middlewares = append(group.middlewares, middlewares...)
}

// 框架的http启动函数
func (engine *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, engine)
}

// 框架的https启动函数
func (engine *Engine) RunTLS(addr string, certFile string, keyFile string) (err error) {
	return http.ListenAndServeTLS(addr, certFile, keyFile, engine)
}

func (engine *Engine) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	var middlewares []HandlerFunc
	for _, group := range engine.groups {
		if strings.HasPrefix(req.URL.Path, group.prefix) {
			middlewares = append(middlewares, group.middlewares...)
		}
	}
	context := newContext(writer, req)
	context.handlers = middlewares //添加中间件
	engine.router.handle(context)  //执行路由匹配
}
