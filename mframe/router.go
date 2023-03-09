package mframe

import (
	"net/http"
	"strings"
)

type router struct {
	roots    map[string]*node //string为请求方式，如 GET,POST
	handlers map[string]HandlerFunc
}

// 创建新路由
func newRouter() *router {
	return &router{
		roots:    make(map[string]*node),
		handlers: make(map[string]HandlerFunc),
	}
}

// 解析模式串
func parsePattern(pattern string) []string {
	vs := strings.Split(pattern, "/")

	parts := make([]string, 0)
	for _, item := range vs {
		if item != "" {
			parts = append(parts, item)
			if item[0] == '*' {
				break
			}
		}
	}
	return parts
}

// 添加路由规则
func (r *router) addRoute(method string, pattern string, handler HandlerFunc) {
	parts := parsePattern(pattern) //把模式串解析成字符串切片
	key := method + "-" + pattern
	_, ok := r.roots[method]
	if !ok { //不存在根节点
		r.roots[method] = &node{}
	}
	r.roots[method].insert(pattern, parts, 0) //插入前缀树
	r.handlers[key] = handler
}

// 执行路由规则对应的处理函数
func (r *router) handle(c *Context) {
	node, params := r.getRoute(c.Method, c.Path)
	if node != nil {
		c.Params = params
		key := c.Method + "-" + node.pattern
		c.handlers = append(c.handlers, r.handlers[key]) //将处理函数添加到中间件的最后，以做到中间件可以在处理函数的开始和结束执行操作
	} else {
		c.handlers = append(c.handlers, func(c *Context) { //将处理函数添加到中间件的最后，以做到中间件可以在处理函数的开始和结束执行操作
			err := c.HTMLF(http.StatusOK, "root/404.html")
			if err != nil { //读取页面时出错
				c.String(http.StatusNotFound, "404 NOT FOUND: %s\n", c.Path) //返回默认404界面
			}
		})
	}
	c.Next()
}

// 将请求路径与前缀树进行匹配，返回匹配到的节点和匹配的表单
func (r *router) getRoute(method string, path string) (*node, map[string]string) {
	searchParts := parsePattern(path)
	params := make(map[string]string)
	root, ok := r.roots[method]

	if !ok { //不存在根节点
		return nil, nil
	}

	n := root.search(searchParts, 0) //匹配节点

	if n != nil { //节点匹配成功
		parts := parsePattern(n.pattern)
		for index, part := range parts { //遍历part，将动态匹配的内容放入匹配表单中
			if part[0] == ':' {
				params[part[1:]] = searchParts[index]
			}
			if part[0] == '*' && len(part) > 1 {
				params[part[1:]] = strings.Join(searchParts[index:], "/")
				break
			}
		}
		return n, params
	}

	return nil, nil
}
