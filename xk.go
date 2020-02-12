package main

import (
	"../xk-fastdfs/server"
	"../xk-fastdfs/web"
	"fmt"
)

//
func main() {

	// 初始化服务端
	s := server.NewServer("s", "x")

	// 初始化 HTTP WEB
	hs := web.NewHttpServer(s)

	//
	fmt.Print(hs)
}
