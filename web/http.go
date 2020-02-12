package web

import (
	"../conf"
	"fmt"
	_ "github.com/eventials/go-tus"
	"github.com/sjqzhang/goutil"
	log "github.com/sjqzhang/seelog"
	"net/http"
	_ "net/http/pprof"
	"runtime/debug"
	"time"
)

//
var util = &goutil.Common{}

// 定义 HTTP Handler
type HttpHandler struct {
	hs *HttpServer
}

// 定义 Handler ServeHTTP
func (hh *HttpHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	code := "200"
	defer func(t time.Time) {
		// 定义日志, 输出日志
		logStr := fmt.Sprintf("[Access] %s | %s | %s | %s | %s |%s",
			time.Now().Format("2006/01/02 - 15:04:05"),
			//res.Header(),
			time.Since(t).String(),
			util.GetClientIp(req),
			req.Method,
			code,
			req.RequestURI,
		)
		conf.Log.Info(logStr)
	}(time.Now())
	defer func() {
		if err := recover(); err != nil {
			code = "500"
			res.WriteHeader(500)
			print(err)
			buff := debug.Stack()
			log.Error(err)
			log.Error(string(buff))
		}
	}()
	// 是否允许跨域访问
	if conf.Global().EnableCrossOrigin {
		CrossOrigin(res, req)
	}
	//
	http.DefaultServeMux.ServeHTTP(res, req)
}

// 开启Web服务, 提供对外接口
// ---------- ---------- ----------
// route 路由
// ---------- ---------- ----------
func (hs *HttpServer) initHttpServer(route string) {

	//
	hs.initHandler(route)

	//
	fmt.Println("Listen on " + conf.Global().Addr)
	srv := &http.Server{
		Addr:              conf.Global().Addr,
		Handler:           &HttpHandler{hs},
		ReadTimeout:       time.Duration(conf.Global().ReadTimeout) * time.Second,
		ReadHeaderTimeout: time.Duration(conf.Global().ReadHeaderTimeout) * time.Second,
		WriteTimeout:      time.Duration(conf.Global().WriteTimeout) * time.Second,
		IdleTimeout:       time.Duration(conf.Global().IdleTimeout) * time.Second,
	}

	// 开启HTTP服务, (阻塞主线程)
	err := srv.ListenAndServe()

	//
	_ = log.Error(err)
	fmt.Println(err)
}

// 定义HTTP接口
func (hs *HttpServer) initHandler(route string) {
	//
	if route == "" {
		http.HandleFunc(fmt.Sprintf("%s", "/"), hs.Home)
	} else {
		//
		http.HandleFunc(fmt.Sprintf("%s", "/"), hs.Home)
		http.HandleFunc(fmt.Sprintf("%s", route), hs.Home)
	}
}

// 跨域访问配置 [https://blog.csdn.net/yanzisu_congcong/article/details/80552155]
func CrossOrigin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Depth, User-Agent, X-File-Size, X-Requested-With, X-Requested-By, If-Modified-Since, X-File-Name, X-File-Type, Cache-Control, Origin")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Expose-Headers", "Authorization")
}
