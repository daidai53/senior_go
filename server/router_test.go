package server

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"reflect"
	"testing"
)

func TestRouter_AddRoute(t *testing.T) {
	// 注册路由树
	// 验证路由树
	testRoutes := []struct {
		method string
		path   string
	}{
		{
			method: http.MethodGet,
			path:   "/user/home",
		},
		{
			method: http.MethodGet,
			path:   "/",
		},
		{
			method: http.MethodGet,
			path:   "/user",
		},
		{
			method: http.MethodGet,
			path:   "/order/detail",
		},
		{
			method: http.MethodGet,
			path:   "/order/*",
		},
		{
			method: http.MethodGet,
			path:   "/*",
		},
		{
			method: http.MethodGet,
			path:   "/*/*",
		},
		{
			method: http.MethodGet,
			path:   "/*/aaa",
		},
		{
			method: http.MethodGet,
			path:   "/*/aaa/*",
		},
		{
			method: http.MethodPost,
			path:   "/order/create",
		},
		{
			method: http.MethodPost,
			path:   "/login",
		},
		{
			method: http.MethodGet,
			path:   "/order/detail/:id",
		},
	}

	var mockHandler HandleFunc = func(ctx *Context) {}
	r := newRouter()

	for _, route := range testRoutes {
		r.addRoute(route.method, route.path, mockHandler)
	}

	wantRouter := &router{
		tree: map[string]*node{
			http.MethodGet: &node{
				path: "/",
				children: map[string]*node{
					"user": &node{
						path: "user",
						children: map[string]*node{
							"home": &node{
								path:    "home",
								handler: mockHandler,
							},
						},
						handler: mockHandler,
					},
					"order": &node{
						path: "order",
						children: map[string]*node{
							"detail": &node{
								path:    "detail",
								handler: mockHandler,
								paramChild: &node{
									path:    ":id",
									handler: mockHandler,
								},
							},
						},
						starChild: &node{
							path:    "*",
							handler: mockHandler,
						},
					},
				},
				starChild: &node{
					path:    "*",
					handler: mockHandler,
					children: map[string]*node{
						"aaa": &node{
							path: "aaa",
							starChild: &node{
								path:    "*",
								handler: mockHandler,
							},
							handler: mockHandler,
						},
					},
					starChild: &node{
						path:    "*",
						handler: mockHandler,
					},
				},
				handler: mockHandler,
			},
			http.MethodPost: &node{
				path: "/",
				children: map[string]*node{
					"order": &node{
						path: "order",
						children: map[string]*node{
							"create": &node{
								path:    "create",
								handler: mockHandler,
							},
						},
					},
					"login": &node{
						path:    "login",
						handler: mockHandler,
					},
				},
			},
		},
	}
	msg, ok := wantRouter.equal(r)
	assert.True(t, ok, msg)
	msg, ok = r.equal(wantRouter)
	assert.True(t, ok, msg)

	assert.Panicsf(t, func() {
		r.addRoute(http.MethodGet, "", mockHandler)
	}, "web: ；路由不能为空字符串")

	assert.Panicsf(t, func() {
		r.addRoute(http.MethodGet, "aaa", mockHandler)
	}, "web: 路由必须以 / 开头")

	assert.Panicsf(t, func() {
		r.addRoute(http.MethodGet, "/aaa/", mockHandler)
	}, "web: 路由不能以 / 结尾")

	assert.Panicsf(t, func() {
		r.addRoute(http.MethodGet, "/aa//a/", mockHandler)
	}, "web: 不能出现连续的 /")

	r = newRouter()
	r.addRoute(http.MethodGet, "/", mockHandler)
	assert.Panicsf(t, func() {
		r.addRoute(http.MethodGet, "/", mockHandler)
	}, "web: 重复注册路由[/]")

	r = newRouter()
	r.addRoute(http.MethodGet, "/login/*", mockHandler)
	assert.Panicsf(t, func() {
		r.addRoute(http.MethodGet, "/login/:id", mockHandler)
	}, "web：不允许同时注册路径参数和通配符匹配，已有通配符匹配")

	r = newRouter()
	r.addRoute(http.MethodGet, "/login/:id", mockHandler)
	assert.Panicsf(t, func() {
		r.addRoute(http.MethodGet, "/login/*", mockHandler)
	}, "web：不允许同时注册路径参数和通配符匹配，已有路径参数匹配")
}

func TestRouter_FindRoute(t *testing.T) {
	testRoutes := []struct {
		method string
		path   string
	}{
		{
			method: http.MethodGet,
			path:   "/user/home",
		},
		{
			method: http.MethodGet,
			path:   "/",
		},
		{
			method: http.MethodGet,
			path:   "/user",
		},
		{
			method: http.MethodGet,
			path:   "/order/detail",
		},
		{
			method: http.MethodPost,
			path:   "/order/create",
		},
		{
			method: http.MethodPost,
			path:   "/login",
		},
		{
			method: http.MethodPost,
			path:   "/login/:username",
		},
		{
			method: http.MethodDelete,
			path:   "/",
		},
		{
			method: http.MethodGet,
			path:   "/order/*",
		},
	}

	var mockHandler HandleFunc = func(ctx *Context) {}
	r := newRouter()
	for _, route := range testRoutes {
		r.addRoute(route.method, route.path, mockHandler)
	}

	testCases := []struct {
		name string

		method string
		path   string

		wantFound bool
		info      *matchInfo
	}{
		{
			name:      "method not found",
			method:    http.MethodOptions,
			wantFound: false,
		},
		{
			// 完全命中
			name:      "order detail",
			method:    http.MethodGet,
			path:      "/order/detail",
			wantFound: true,
			info: &matchInfo{
				n: &node{
					path:    "detail",
					handler: mockHandler,
				},
			},
		},
		{
			name:      "order",
			method:    http.MethodGet,
			path:      "/order",
			wantFound: false,
		},
		{
			name:      "root",
			method:    http.MethodDelete,
			path:      "/",
			wantFound: true,
			info: &matchInfo{
				n: &node{
					path:    "/",
					handler: mockHandler,
				},
			},
		},
		{
			name:      "order star",
			method:    http.MethodGet,
			path:      "/order/*",
			wantFound: true,
			info: &matchInfo{
				n: &node{
					path:    "*",
					handler: mockHandler,
				},
			},
		},
		{
			name:      "login username",
			method:    http.MethodPost,
			path:      "/login/daidai53",
			wantFound: true,
			info: &matchInfo{
				n: &node{
					path:    ":username",
					handler: mockHandler,
				},
				pathParams: map[string]string{
					"username": "daidai53",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info, found := r.findRoute(tc.method, tc.path)
			assert.Equal(t, found, tc.wantFound)
			if !found {
				return
			}

			assert.Equal(t, tc.info.pathParams, info.pathParams)
			msg, ok := tc.info.n.equal(info.n)
			assert.True(t, ok, msg)
		})
	}
}

func (r *router) equal(s *router) (string, bool) {
	for k, v := range r.tree {
		dst, ok := s.tree[k]
		if !ok {
			return fmt.Sprintf("cannot find this http method: %v", k), false
		}
		msg, equal := v.equal(dst)
		if !equal {
			return msg, false
		}
	}
	return "", true
}

func (n *node) equal(o *node) (string, bool) {
	if n == nil && o == nil {
		return "", true
	}
	if n == nil || o == nil {
		return "其中一个为nil", false
	}
	if n.path != o.path {
		return fmt.Sprintf("path不同"), false
	}
	if len(n.children) != len(o.children) {
		return fmt.Sprintf("子节点数量不同"), false
	}
	if n.starChild != nil {
		msg, ok := n.starChild.equal(o.starChild)
		if !ok {
			return msg, ok
		}
	}
	if n.paramChild != nil {
		msg, ok := o.paramChild.equal(n.paramChild)
		if !ok {
			return msg, ok
		}
	}
	nHandler := reflect.ValueOf(n.handler)
	oHandler := reflect.ValueOf(o.handler)
	if nHandler != oHandler {
		return fmt.Sprintf("handler不同"), false
	}

	for path, child := range n.children {
		dst, ok := o.children[path]
		if !ok {
			return fmt.Sprintf("该method下的子节点不存在: %v", path), false
		}
		msg, equal := child.equal(dst)
		if !equal {
			return msg, false
		}
	}
	return "", true
}
