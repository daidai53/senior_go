package web

import (
	"fmt"
	"regexp"
	"strings"
)

type router struct {
	// trees 是按照 HTTP 方法来组织的
	// 如 GET => *node
	trees map[string]*node
}

func newRouter() router {
	return router{
		trees: map[string]*node{},
	}
}

// addRoute 注册路由。
// method 是 HTTP 方法
// - 已经注册了的路由，无法被覆盖。例如 /user/home 注册两次，会冲突
// - path 必须以 / 开始并且结尾不能有 /，中间也不允许有连续的 /
// - 不能在同一个位置注册不同的参数路由，例如 /user/:id 和 /user/:name 冲突
// - 不能在同一个位置同时注册通配符路由和参数路由，例如 /user/:id 和 /user/* 冲突
// - 同名路径参数，在路由匹配的时候，值会被覆盖。例如 /user/:id/abc/:id，那么 /user/123/abc/456 最终 id = 456
func (r *router) addRoute(method string, path string, handler HandleFunc) {
	root, ok := r.trees[method]
	if !ok {
		root = &node{
			typ:  nodeTypeStatic,
			path: "/",
		}
		r.trees[method] = root
	}

	if path == "" {
		panic("web: 路由是空字符串")
	}

	if path == "/" {
		if root.handler != nil {
			panic("web: 路由冲突[/]")
		} else {
			root.handler = handler
		}
		return
	}

	if strings.Contains(path, "//") {
		panic(fmt.Sprintf("web: 非法路由。不允许使用 //a/b, /a//b 之类的路由, [%s]", path))
	}
	pathSegments := strings.Split(path, "/")
	if len(pathSegments) < 2 {
		panic("Invalid path: " + path)
	}
	if pathSegments[0] != "" {
		panic("web: 路由必须以 / 开头")
	}
	if pathSegments[len(pathSegments)-1] == "" {
		panic("web: 路由不能以 / 结尾")
	}
	for _, segment := range pathSegments[1:] {
		root = root.childOrCreate(segment)
		if root == nil {
			panic("Something went wrong, got a nil node")
		}
	}
	if root.handler != nil {
		panic(fmt.Sprintf("web: 路由冲突[%s]", path))
	}
	root.handler = handler
}

// findRoute 查找对应的节点
// 注意，返回的 node 内部 HandleFunc 不为 nil 才算是注册了路由
func (r *router) findRoute(method string, path string) (*matchInfo, bool) {
	if r == nil {
		return nil, false
	}
	tree, ok := r.trees[method]
	if !ok {
		return nil, false
	}

	if path == "/" {
		if tree.path == "/" {
			return &matchInfo{n: tree}, true
		} else {
			return nil, false
		}
	}

	pathSegments := strings.Split(path, "/")
	if len(pathSegments) == 0 {
		return nil, false
	}
	if pathSegments[0] == "" {
		pathSegments = pathSegments[1:]
	}
	child := tree
	var param string
	ret := &matchInfo{}
	for _, segment := range pathSegments {
		param = segment
		newchild, ok := child.childOf(segment)
		if !ok {
			if child.typ == nodeTypeAny {
				return &matchInfo{n: child}, true
			}
			return nil, false
		}
		if newchild.typ == nodeTypeParam || newchild.typ == nodeTypeReg {
			if ret.pathParams == nil {
				ret.pathParams = make(map[string]string)
			}
			ret.pathParams[newchild.paramName] = param
		}
		child = newchild
	}
	ret.n = child
	return ret, true
}

type nodeType int

const (
	// 静态路由
	nodeTypeStatic = iota
	// 正则路由
	nodeTypeReg
	// 路径参数路由
	nodeTypeParam
	// 通配符路由
	nodeTypeAny
)

// node 代表路由树的节点
// 路由树的匹配顺序是：
// 1. 静态完全匹配
// 2. 正则匹配，形式 :param_name(reg_expr)
// 3. 路径参数匹配：形式 :param_name
// 4. 通配符匹配：*
// 这是不回溯匹配
type node struct {
	typ nodeType

	path string
	// children 子节点
	// 子节点的 path => node
	children map[string]*node
	// handler 命中路由之后执行的逻辑
	handler HandleFunc

	// 通配符 * 表达的节点，任意匹配
	starChild *node

	paramChild *node
	// 正则路由和参数路由都会使用这个字段
	paramName string

	// 正则表达式
	regChild *node
	regExpr  *regexp.Regexp
}

// child 返回子节点
// 第一个返回值 *node 是命中的节点
// 第二个返回值 bool 代表是否命中
func (n *node) childOf(path string) (*node, bool) {
	staticChild, ok := n.children[path]
	if ok && staticChild.typ == nodeTypeStatic && staticChild.path == path {
		return staticChild, ok
	}
	if n.regChild != nil && n.regChild.typ == nodeTypeReg && n.regChild.regExpr.MatchString(path) {
		return n.regChild, true
	}
	if n.paramChild != nil && n.paramChild.typ == nodeTypeParam {
		return n.paramChild, true
	}
	if n.starChild != nil && n.starChild.typ == nodeTypeAny {
		return n.starChild, true
	}
	return nil, false
}

// childOrCreate 查找子节点，
// 首先会判断 path 是不是通配符路径
// 其次判断 path 是不是参数路径，即以 : 开头的路径
// 最后会从 children 里面查找，
// 如果没有找到，那么会创建一个新的节点，并且保存在 node 里面
func (n *node) childOrCreate(path string) *node {
	// 通配
	if path == "*" {
		if n.starChild != nil {
			return n.starChild
		}
		if n.paramChild != nil {
			panic(fmt.Sprintf("web: 非法路由，已有路径参数路由。不允许同时注册通配符路由和参数路由 [*]"))
		}
		if n.regChild != nil {
			panic(fmt.Sprintf("web: 非法路由，已有正则路由。不允许同时注册通配符路由和正则路由 [*]"))
		}
		child := &node{
			typ:  nodeTypeAny,
			path: "*",
		}
		n.starChild = child
		return child
	}
	// :开头，路径参数或者正则匹配
	if path[0] == ':' {
		tmps := strings.Split(path[1:], "(")
		if len(tmps) == 2 {
			tmp1 := tmps[1]
			if tmp1[len(tmp1)-1] == ')' {
				tmp1 = tmp1[:len(tmp1)-1]
				regExpr, err := regexp.Compile(tmp1)
				if err != nil {
					panic(err)
				}
				if n.regChild != nil {
					if n.regChild.paramName == tmps[0] && n.regChild.regExpr.String() == regExpr.String() {
						return n.regChild
					} else {
						panic("1")
					}
				}
				if n.starChild != nil {
					panic(fmt.Sprintf("web: 非法路由，已有通配符路由。不允许同时注册通配符路由和正则路由 [%s]", path))
				}
				if n.paramChild != nil {
					panic(fmt.Sprintf("web: 非法路由，已有路径参数路由。不允许同时注册正则路由和参数路由 [%s]", path))
				}
				child := &node{
					typ:       nodeTypeReg,
					path:      path,
					regExpr:   regExpr,
					paramName: tmps[0],
				}
				n.regChild = child
				return child
			} else {
				panic("Invalid path: " + path)
			}
		}
		if n.paramChild != nil {
			if n.paramChild.paramName == path[1:] {
				return n.paramChild
			} else {
				panic(fmt.Sprintf("web: 路由冲突，参数路由冲突，已有 :%s，新注册 %s", n.paramChild.paramName, path))
			}
		}
		if n.starChild != nil {
			panic(fmt.Sprintf("web: 非法路由，已有通配符路由。不允许同时注册通配符路由和参数路由 [%s]", path))
		}
		if n.regChild != nil {
			panic(fmt.Sprintf("web: 非法路由，已有正则路由。不允许同时注册正则路由和参数路由 [%s]", path))
		}
		child := &node{
			typ:       nodeTypeParam,
			path:      path,
			paramName: tmps[0],
		}
		n.paramChild = child
		return child
	}
	child, ok := n.children[path]
	if ok {
		return child
	}
	child = &node{
		typ:  nodeTypeStatic,
		path: path,
	}
	if n.children == nil {
		n.children = make(map[string]*node)
	}
	n.children[path] = child
	return child
}

type matchInfo struct {
	n          *node
	pathParams map[string]string
}

func (m *matchInfo) addValue(key string, value string) {
	if m.pathParams == nil {
		// 大多数情况，参数路径只会有一段
		m.pathParams = map[string]string{key: value}
	}
	m.pathParams[key] = value
}
