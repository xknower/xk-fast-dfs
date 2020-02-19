// Web 参数初始化
package web

import (
	"../conf"
	jsoniter "github.com/json-iterator/go"
	"github.com/sjqzhang/goutil"
	slog "github.com/sjqzhang/seelog"
	"net/http"
)

// JSON 解析
var json = jsoniter.ConfigCompatibleWithStandardLibrary

//
var Log slog.LoggerInterface

//
var util = &goutil.Common{}

const (
	CONST_SMALL_FILE_SIZE = int(conf.CONST_SMALL_FILE_SIZE)
	// 操作标识
	CONST_FILE_Md5_FILE_NAME   = conf.CONST_FILE_Md5_FILE_NAME
	CONST_Md5_ERROR_FILE_NAME  = conf.CONST_Md5_ERROR_FILE_NAME
	CONST_Md5_QUEUE_FILE_NAME  = conf.CONST_Md5_QUEUE_FILE_NAME
	CONST_REMOME_Md5_FILE_NAME = conf.CONST_REMOME_Md5_FILE_NAME
	//
	CONST_STAT_FILE_COUNT_KEY      = conf.CONST_STAT_FILE_COUNT_KEY
	CONST_STAT_FILE_TOTAL_SIZE_KEY = conf.CONST_STAT_FILE_TOTAL_SIZE_KEY

	CONST_BIG_UPLOAD_PATH_SUFFIX = conf.CONST_BIG_UPLOAD_PATH_SUFFIX

	GO_FASTDFS_IP = conf.GO_FASTDFS_IP
	UPPY_HTML     = conf.UPLOAD_UPPY_HTML
)

// 项目应用目录
var (
	DOCKER_DIR     = conf.DirDocker
	DATA_DIR       = conf.DirData
	STORE_DIR      = conf.DirStore
	LARGE_DIR_NAME = conf.DirLargeName
	STATIC_DIR     = conf.DirStatic
	STORE_DIR_NAME = conf.STORE_DIR_NAME
)

var (
	//
	host                string
	addr                string
	group               string
	peers               []string
	defaultScene        string
	enableCrossOrigin   bool
	enableGoogleAuth    bool
	enableMigrate       bool
	supportGroupManage  bool
	authUrl             string
	downloadTokenExpire int
	refreshInterval     int
	autoRepair          bool
	readTimeout         int
	writeTimeout        int

	//
	adminIps           []string
	enableWebUpload    bool
	enableDownloadAuth bool
	showDir            bool
	defaultDownload    bool
	downloadUseToken   bool
	readHeaderTimeout  int
	idleTimeout        int
)

// 文件下载配置
var staticHandler http.Handler

func init() {
	group = conf.Global().Group
	enableCrossOrigin = conf.Global().EnableCrossOrigin
	showDir = conf.Global().ShowDir
	enableDownloadAuth = conf.Global().EnableDownloadAuth
	authUrl = conf.Global().AuthUrl
	downloadUseToken = conf.Global().DownloadUseToken
	downloadTokenExpire = conf.Global().DownloadTokenExpire
	defaultDownload = conf.Global().DefaultDownload
	enableGoogleAuth = conf.Global().EnableGoogleAuth
	peers = conf.Global().Peers
	enableMigrate = conf.Global().EnableMigrate
	enableWebUpload = conf.Global().EnableWebUpload
	supportGroupManage = conf.Global().SupportGroupManage
	defaultScene = conf.Global().DefaultScene
	host = conf.Global().Host
	autoRepair = conf.Global().AutoRepair
	refreshInterval = conf.Global().RefreshInterval
	adminIps = conf.Global().AdminIps
	addr = conf.Global().Addr
	readTimeout = conf.Global().ReadTimeout
	readHeaderTimeout = conf.Global().ReadHeaderTimeout
	writeTimeout = conf.Global().WriteTimeout
	idleTimeout = conf.Global().IdleTimeout

	Log = conf.Log

	// 下载文件配置, 文件服务器
	if supportGroupManage {
		staticHandler = http.StripPrefix("/"+group+"/", http.FileServer(http.Dir(STORE_DIR)))
	} else {
		staticHandler = http.StripPrefix("/", http.FileServer(http.Dir(STORE_DIR)))
	}
}
