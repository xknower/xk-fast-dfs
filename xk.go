// 应用启动入口
package main

import (
	"../xk-fastdfs/server"
	"../xk-fastdfs/web"
	"fmt"
)

//
func main() {

	// 初始化 HTTP WEB
	hs := web.NewHttpServer()

	// 初始化服务端
	s, _ := server.NewService()
	s.Start()

	// 开始服务
	hs.Start(s)

	//
	fmt.Print(hs)
}
