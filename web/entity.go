// Web 实体定义
package web

import (
	"../en"
	_ "github.com/eventials/go-tus"
	_ "net/http/pprof"
)

// 定义HTTP接口
type HttpServer struct {
	// 服务端, 访问服务端功能
	s en.Server
}

// 初始化HTTP服务
func NewHttpServer() *HttpServer {
	// 初始化
	hs := &HttpServer{}
	return hs
}

func (hs *HttpServer) Start(s en.Server) {
	hs.s = s

	// 初始化 HTTP Handler & 并启动服务
	hs.initHttpServer(s.GetGroupRouteName())
}
