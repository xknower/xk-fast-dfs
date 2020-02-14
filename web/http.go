package web

import (
	"fmt"
	_ "github.com/eventials/go-tus"
	slog "github.com/sjqzhang/seelog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime/debug"
	"strings"
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
			slog.Error(err)
			slog.Error(string(buff))
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
		http.HandleFunc(fmt.Sprintf("%s", "/"), hs.Download)
		http.HandleFunc(fmt.Sprintf("/%s", uploadPage), hs.Index)
	} else {
		http.HandleFunc(fmt.Sprintf("%s", "/"), hs.Download)
		http.HandleFunc(fmt.Sprintf("%s", route), hs.Download)
		http.HandleFunc(fmt.Sprintf("%s/%s", route, uploadPage), hs.Index)
	}

	http.HandleFunc(fmt.Sprintf("%s/check_files_exist", route), hs.CheckFilesExist)
	http.HandleFunc(fmt.Sprintf("%s/check_file_exist", route), hs.CheckFileExist)
	http.HandleFunc(fmt.Sprintf("%s/upload", route), hs.Upload)
	http.HandleFunc(fmt.Sprintf("%s/delete", route), hs.RemoveFile)
	http.HandleFunc(fmt.Sprintf("%s/get_file_info", route), hs.GetFileInfo)
	http.HandleFunc(fmt.Sprintf("%s/sync", route), hs.Sync)
	http.HandleFunc(fmt.Sprintf("%s/stat", route), hs.Stat)
	http.HandleFunc(fmt.Sprintf("%s/repair_stat", route), hs.RepairStatWeb)
	http.HandleFunc(fmt.Sprintf("%s/status", route), hs.Status)
	http.HandleFunc(fmt.Sprintf("%s/repair", route), hs.Repair)
	http.HandleFunc(fmt.Sprintf("%s/report", route), hs.Report)
	http.HandleFunc(fmt.Sprintf("%s/backup", route), hs.BackUp)
	http.HandleFunc(fmt.Sprintf("%s/search", route), hs.Search)
	http.HandleFunc(fmt.Sprintf("%s/list_dir", route), hs.ListDir)
	http.HandleFunc(fmt.Sprintf("%s/remove_empty_dir", route), hs.RemoveEmptyDir)
	http.HandleFunc(fmt.Sprintf("%s/repair_fileinfo", route), hs.RepairFileInfo)
	http.HandleFunc(fmt.Sprintf("%s/reload", route), hs.s.Reload)
	http.HandleFunc(fmt.Sprintf("%s/syncfile_info", route), hs.SyncFileInfo)
	http.HandleFunc(fmt.Sprintf("%s/get_md5s_by_date", route), hs.GetMd5sForWeb)
	http.HandleFunc(fmt.Sprintf("%s/receive_md5s", route), hs.ReceiveMd5s)
	http.HandleFunc(fmt.Sprintf("%s/gen_google_secret", route), hs.GenGoogleSecret)
	http.HandleFunc(fmt.Sprintf("%s/gen_google_code", route), hs.GenGoogleCode)
	http.HandleFunc("/"+Group+"/", hs.Download)
}

// 跨域访问配置 [https://blog.csdn.net/yanzisu_congcong/article/details/80552155]
func CrossOrigin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Depth, User-Agent, X-File-Size, X-Requested-With, X-Requested-By, If-Modified-Since, X-File-Name, X-File-Type, Cache-Control, Origin")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Expose-Headers", "Authorization")
}

func GetClusterNotPermitMessage(r *http.Request) string {
	var (
		message string
	)
	message = fmt.Sprintf(CONST_MESSAGE_CLUSTER_IP, util.GetClientIp(r))
	return message
}

func IsPeer(r *http.Request) bool {
	var (
		ip    string
		peer  string
		bflag bool
	)
	//return true
	ip = util.GetClientIp(r)
	realIp := os.Getenv(GO_FASTDFS_IP)
	if realIp == "" {
		realIp = util.GetPulicIP()
	}
	if ip == "127.0.0.1" || ip == realIp {
		return true
	}
	if util.Contains(ip, AdminIps) {
		return true
	}
	ip = "http://" + ip
	bflag = false
	for _, peer = range Peers {
		if strings.HasPrefix(peer, ip) {
			bflag = true
			break
		}
	}
	return bflag
}
