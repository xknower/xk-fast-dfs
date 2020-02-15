package web

import (
	"../en"
	"fmt"
	_ "github.com/eventials/go-tus"
	slog "github.com/sjqzhang/seelog"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// 获取服务状态信息 [处理状态信息 state.json]
func (hs *HttpServer) getStat() []en.StatDateFileInfo {
	var (
		err   error
		min   int64
		max   int64
		rows  []en.StatDateFileInfo
		total en.StatDateFileInfo
	)
	min = 20190101
	max = 20190101
	var i int64
	for k := range hs.s.GetStatMap().Get() {
		ks := strings.Split(k, "_")
		if len(ks) == 2 {
			if i, err = strconv.ParseInt(ks[0], 10, 64); err != nil {
				continue
			}
			if i >= max {
				max = i
			}
			if i < min {
				min = i
			}
		}
	}
	for i := min; i <= max; i++ {
		s := fmt.Sprintf("%d", i)
		if v, ok := hs.s.GetStatMap().GetValue(s + "_" + CONST_STAT_FILE_TOTAL_SIZE_KEY); ok {
			var info en.StatDateFileInfo
			info.Date = s
			switch v.(type) {
			case int64:
				info.TotalSize = v.(int64)
				total.TotalSize = total.TotalSize + v.(int64)
			}
			if v, ok := hs.s.GetStatMap().GetValue(s + "_" + CONST_STAT_FILE_COUNT_KEY); ok {
				switch v.(type) {
				case int64:
					info.FileCount = v.(int64)
					total.FileCount = total.FileCount + v.(int64)
				}
			}
			rows = append(rows, info)
		}
	}
	total.Date = "all"
	rows = append(rows, total)
	return rows
}

// ---------- ---------- ---------- ---------- ---------- ---------- ---------- ----------

// 跨域访问配置 [https://blog.csdn.net/yanzisu_congcong/article/details/80552155]
func CrossOrigin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Depth, User-Agent, X-File-Size, X-Requested-With, X-Requested-By, If-Modified-Since, X-File-Name, X-File-Type, Cache-Control, Origin")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Expose-Headers", "Authorization")
}

// 下载文件, 设置下载响应 Header
func SetDownloadHeader(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment")
}

// HTTP响应错误信息, 必须使用正确的节点IP地址访问资源
func GetClusterNotPermitMessage(r *http.Request) string {
	var (
		message string
	)
	message = fmt.Sprintf(CONST_MESSAGE_CLUSTER_IP, util.GetClientIp(r))
	return message
}

// 根据请求IP, 判断访问是否是 集权节点 (发起请求客户端, 必须是集权中的节点)
func IsPeer(r *http.Request) bool {
	// 判断是否是本地节点
	ip := util.GetClientIp(r)
	realIp := os.Getenv(GO_FASTDFS_IP)
	if realIp == "" {
		realIp = util.GetPulicIP()
	}
	if ip == "127.0.0.1" || ip == realIp {
		return true
	}
	if util.Contains(ip, adminIps) {
		return true
	}
	// 判断是否是非本地集群节点 (Peers 手动配置的及集群节点列表)
	ip = "http://" + ip
	flag := false
	for _, peer := range peers {
		if strings.HasPrefix(peer, ip) {
			flag = true
			break
		}
	}
	return flag
}

// 请求数据中, 获取文件路径
func analyseFilePathFromRequest(w http.ResponseWriter, r *http.Request) (string, string) {
	var (
		err       error
		smallPath string
		prefix    string
	)
	fullPath := r.RequestURI[1:]
	if strings.HasPrefix(r.RequestURI, "/"+group+"/") {
		fullPath = r.RequestURI[len(group)+2 : len(r.RequestURI)]
	}
	fullPath = strings.Split(fullPath, "?")[0] // just path
	fullPath = DOCKER_DIR + STORE_DIR_NAME + "/" + fullPath
	prefix = "/" + LARGE_DIR_NAME + "/"
	if supportGroupManage {
		prefix = "/" + group + "/" + LARGE_DIR_NAME + "/"
	}
	if strings.HasPrefix(r.RequestURI, prefix) {
		smallPath = fullPath //notice order
		fullPath = strings.Split(fullPath, ",")[0]
	}
	if fullPath, err = url.PathUnescape(fullPath); err != nil {
		_ = slog.Error(err)
	}
	return fullPath, smallPath
}
