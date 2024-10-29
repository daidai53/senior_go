//go:build e2e

package server

import (
	"fmt"
	"net/http"
	"testing"
)

func TestServer(t *testing.T) {
	var h Server = &HttpServer{
		&router{
			tree: map[string]*node{},
		},
	}
	//http.ListenAndServe(":8080", h)
	//http.ListenAndServeTLS(":443", "", "", h)
	h.addRoute(http.MethodGet, "/user", func(ctx *Context) {
		fmt.Println("处理1")
	})

	h.Start(":8080")
}
