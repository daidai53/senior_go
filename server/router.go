package server

import (
	"fmt"
	"strings"
)

type router struct {
	tree map[string]*node
}

type node struct {
	path       string
	children   map[string]*node
	starChild  *node
	paramChild *node
	handler    HandleFunc
}

type matchInfo struct {
	n          *node
	pathParams map[string]string
}

func newRouter() *router {
	return &router{tree: make(map[string]*node)}
}

func (r *router) addRoute(method string, path string, handleFunc HandleFunc) {
	if path == "" {
		panic("web: ；路由不能为空字符串")
	}
	root, ok := r.tree[method]
	if !ok {
		r.tree[method] = &node{
			path: "/",
		}
		root = r.tree[method]
	}

	if path[0] != '/' {
		panic("web: 路由必须以 / 开头")
	}

	if path != "/" && path[len(path)-1] == '/' {
		panic("web: 路由不能以 / 结尾")
	}

	if path == "/" {
		if root.handler != nil {
			panic("web: 重复注册路由[/]")
		}
		root.handler = handleFunc
		return
	}

	segs := strings.Split(strings.TrimPrefix(path, "/"), "/")
	for _, seg := range segs {
		if seg == "" {
			panic("web: 不能出现连续的 /")
		}
		root = root.getOrCreateChild(seg)
	}
	if root.handler != nil {
		panic(fmt.Sprintf("web: 重复注册路由[%s]", path))
	}
	root.handler = handleFunc
}

func (r *router) findRoute(method string, path string) (*matchInfo, bool) {
	root, ok := r.tree[method]
	if !ok {
		return nil, false
	}

	if path == "/" {
		return &matchInfo{n: root}, true
	}

	path = strings.Trim(path, "/")
	segs := strings.Split(path, "/")
	var pathParams map[string]string
	for _, seg := range segs {
		child, paramChild, found := root.childOf(seg)
		if !found {
			return nil, false
		}
		if paramChild {
			if pathParams == nil {
				pathParams = map[string]string{}
			}
			pathParams[child.path[1:]] = seg
		}
		root = child
	}
	// 只返回有处理方法的节点
	return &matchInfo{
		n:          root,
		pathParams: pathParams,
	}, root.handler != nil
}

func (n *node) getOrCreateChild(seg string) *node {
	if seg[0] == ':' {
		if n.starChild != nil {
			panic("web：不允许同时注册路径参数和通配符匹配，已有通配符匹配")
		}
		if n.paramChild != nil {
			return n.paramChild
		}
		n.paramChild = &node{
			path: seg,
		}
		return n.paramChild
	}
	if seg == "*" {
		if n.paramChild != nil {
			panic("web：不允许同时注册路径参数和通配符匹配，已有路径参数匹配")
		}
		if n.starChild != nil {
			return n.starChild
		}
		n.starChild = &node{
			path: seg,
		}
		return n.starChild
	}
	if n.children == nil {
		n.children = make(map[string]*node)
		res := &node{
			path: seg,
		}
		n.children[seg] = res
		return res
	}
	child, ok := n.children[seg]
	if !ok {
		child = &node{
			path: seg,
		}
		n.children[seg] = child
	}
	return child
}

func (n *node) childOf(seg string) (*node, bool, bool) {
	if n == nil || n.children == nil {
		if n.paramChild != nil {
			return n.paramChild, true, true
		}
		return n.starChild, false, n.starChild != nil
	}
	child, ok := n.children[seg]
	if !ok {
		if n.paramChild != nil {
			return n.paramChild, true, true
		}
		return n.starChild, false, n.starChild != nil
	}
	return child, false, ok
}
