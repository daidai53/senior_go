package server

import (
	"net"
	"net/http"
)

var _ Server = &HttpServer{}

type HandleFunc func(ctx *Context)

type Server interface {
	http.Handler
	Start(addr string) error

	// AddRoute 注册路由
	// method是HTTP方法
	// path是请求路径
	// handleFunc是业务逻辑
	addRoute(method string, path string, handleFunc HandleFunc)
	findRoute(method string, path string) (*matchInfo, bool)
}

type HttpServer struct {
	*router
}

// ServeHTTP 处理请求的入口
// 框架代码就是写在这里，包括：Context构建；路由匹配；执行业务逻辑。
func (s *HttpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	context := &Context{
		request:  r,
		response: w,
	}

	s.serve(context)
}

func (s *HttpServer) serve(ctx *Context) {
	r, ok := s.findRoute(ctx.request.Method, ctx.request.URL.Path)
	if !ok || r.n.handler == nil {
		ctx.response.WriteHeader(http.StatusNotFound)
		_, _ = ctx.response.Write([]byte("你要通往何方。。。"))
		return
	}
	ctx.pathParams = r.pathParams
	r.n.handler(ctx)
}

func (s *HttpServer) Start(addr string) error {
	h, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return http.Serve(h, s)
}

func (s *HttpServer) NewHTTPServer() *HttpServer {
	return &HttpServer{
		router: newRouter(),
	}
}
