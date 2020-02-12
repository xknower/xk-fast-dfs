package web

import (
	"../server"
	_ "github.com/eventials/go-tus"
	_ "net/http/pprof"
)

// 定义HTTP接口
type HttpServer struct {
	// 服务端, 访问服务端功能
	s server.Server
}

// 初始化HTTP服务
func NewHttpServer(s server.Server) *HttpServer {
	// 初始化
	hs := &HttpServer{
		s,
	}

	// 初始化 HTTP Handler
	hs.initHttpServer(s.GetGroupRouteName())
	return hs
}
