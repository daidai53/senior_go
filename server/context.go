package server

import "net/http"

type Context struct {
	request    *http.Request
	response   http.ResponseWriter
	pathParams map[string]string
}
