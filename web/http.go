package web

import (
	"fmt"
	_ "github.com/eventials/go-tus"
	slog "github.com/sjqzhang/seelog"
	"net/http"
	_ "net/http/pprof"
	"runtime/debug"
	"time"
)

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
		slog.Info(logStr)
	}(time.Now())
	defer func() {
		if err := recover(); err != nil {
			code = "500"
			res.WriteHeader(500)
			print(err)
			buff := debug.Stack()
			_ = slog.Error(err)
			_ = slog.Error(string(buff))
		}
	}()
	// 是否允许跨域访问
	if EnableCrossOrigin {
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
	fmt.Println("Listen on " + Addr)
	srv := &http.Server{
		Addr:              Addr,
		Handler:           &HttpHandler{hs},
		ReadTimeout:       time.Duration(ReadTimeout) * time.Second,
		ReadHeaderTimeout: time.Duration(ReadHeaderTimeout) * time.Second,
		WriteTimeout:      time.Duration(WriteTimeout) * time.Second,
		IdleTimeout:       time.Duration(IdleTimeout) * time.Second,
	}

	// 开启HTTP服务, (阻塞主线程)
	err := srv.ListenAndServe()

	//
	_ = slog.Error(err)
	fmt.Println(err)
}

// 定义HTTP接口
func (hs *HttpServer) initHandler(route string) {
	////
	//if route == "" {
	//	http.HandleFunc(fmt.Sprintf("%s", "/"), hs.Home)
	//} else {
	//	//
	//	http.HandleFunc(fmt.Sprintf("%s", "/"), hs.Home)
	//	http.HandleFunc(fmt.Sprintf("%s", route), hs.Home)
	//}

	uploadPage := "upload.html"
	if route == "" {
		// 上传页面
		http.HandleFunc(fmt.Sprintf("/%s", uploadPage), hs.IndexHTML)
	} else {
		http.HandleFunc(fmt.Sprintf("%s", route), hs.Home)
		http.HandleFunc(fmt.Sprintf("%s/%s", route, uploadPage), hs.IndexHTML)

	}

	// 主页
	http.HandleFunc(fmt.Sprintf("%s", "/"), hs.Home)
	http.HandleFunc(fmt.Sprintf("%s/report", route), hs.ReportHTML)
	// 上传文件
	http.HandleFunc(fmt.Sprintf("%s/upload", route), hs.Upload)
	// 下载文件
	http.HandleFunc(fmt.Sprintf("/%s/", Group), hs.Download)

	// 检测并查询文件信息
	http.HandleFunc(fmt.Sprintf("%s/check_files_exist", route), hs.CheckFilesExist)
	http.HandleFunc(fmt.Sprintf("%s/check_file_exist", route), hs.CheckFileExist)
	// 获取文件信息 [md5, path]
	http.HandleFunc(fmt.Sprintf("%s/get_file_info", route), hs.GetFileInfo)
	http.HandleFunc(fmt.Sprintf("%s/list_dir", route), hs.ListDir)
	http.HandleFunc(fmt.Sprintf("%s/search", route), hs.Search)
	http.HandleFunc(fmt.Sprintf("%s/delete", route), hs.RemoveFile)
	http.HandleFunc(fmt.Sprintf("%s/remove_empty_dir", route), hs.RemoveEmptyDir)
	// 上传文件信息, 异步从集权下载文件, 获取下载地址
	http.HandleFunc(fmt.Sprintf("%s/syncfile_info", route), hs.SyncFileInfo)
	http.HandleFunc(fmt.Sprintf("%s/sync", route), hs.Sync)
	//
	http.HandleFunc(fmt.Sprintf("%s/get_md5s_by_date", route), hs.GetMd5sForWeb)
	http.HandleFunc(fmt.Sprintf("%s/receive_md5s", route), hs.ReceiveMd5s)
	// 获取状态信息
	http.HandleFunc(fmt.Sprintf("%s/stat", route), hs.Stat)
	http.HandleFunc(fmt.Sprintf("%s/status", route), hs.Status)
	//
	http.HandleFunc(fmt.Sprintf("%s/repair", route), hs.Repair)
	http.HandleFunc(fmt.Sprintf("%s/repair_stat", route), hs.RepairStatWeb)
	http.HandleFunc(fmt.Sprintf("%s/repair_fileinfo", route), hs.RepairFileInfo)
	//
	http.HandleFunc(fmt.Sprintf("%s/backup", route), hs.BackUp)
	http.HandleFunc(fmt.Sprintf("%s/gen_google_code", route), hs.GenGoogleCode)
	http.HandleFunc(fmt.Sprintf("%s/gen_google_secret", route), hs.GenGoogleSecret)
	// 重启后台服务
	http.HandleFunc(fmt.Sprintf("%s/reload", route), hs.s.Reload)

}
