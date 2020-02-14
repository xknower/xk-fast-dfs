package web

import (
	"../conf"
	jsoniter "github.com/json-iterator/go"
	"github.com/sjqzhang/goutil"
	"net/http"
)

// JSON 解析
var json = jsoniter.ConfigCompatibleWithStandardLibrary

//
var util = &goutil.Common{}

var (
	STORE_DIR      = ""
	DATA_DIR       = ""
	DOCKER_DIR     = ""
	STATIC_DIR     = ""
	LARGE_DIR_NAME = ""
)

const (
	CONST_SMALL_FILE_SIZE          = int(1024 * 1024)
	CONST_STAT_FILE_COUNT_KEY      = "fileCount"
	CONST_FILE_Md5_FILE_NAME       = "files.md5"
	CONST_BIG_UPLOAD_PATH_SUFFIX   = "/big/upload/"
	CONST_STAT_FILE_TOTAL_SIZE_KEY = "totalSize"
	CONST_Md5_ERROR_FILE_NAME      = "errors.md5"
	CONST_Md5_QUEUE_FILE_NAME      = "queue.md5"
	CONST_REMOME_Md5_FILE_NAME     = "removes.md5"
	CONST_MESSAGE_CLUSTER_IP       = "Can only be called by the cluster ip or 127.0.0.1 or admin_ips(cfg.json),current ip:%s"
	GO_FASTDFS_IP                  = "GO_FASTDFS_IP"
)

var (
	STORE_DIR_NAME      = ""
	Group               string
	EnableCrossOrigin   bool
	ShowDir             bool
	EnableDownloadAuth  bool
	AuthUrl             string
	DownloadUseToken    bool
	DownloadTokenExpire int
	DefaultDownload     bool
	EnableGoogleAuth    bool
	Peers               []string
	EnableMigrate       bool
	EnableWebUpload     bool
	SupportGroupManage  bool
	DefaultScene        string
	Host                string
	AutoRepair          bool
	RefreshInterval     int
	AdminIps            []string
	Addr                string
	ReadTimeout         int
	ReadHeaderTimeout   int
	WriteTimeout        int
	IdleTimeout         int
)

//
var staticHandler http.Handler

func init() {
	STORE_DIR_NAME = conf.STORE_DIR_NAME
	Group = conf.Global().Group
	EnableCrossOrigin = conf.Global().EnableCrossOrigin
	ShowDir = conf.Global().ShowDir
	EnableDownloadAuth = conf.Global().EnableDownloadAuth
	AuthUrl = conf.Global().AuthUrl
	DownloadUseToken = conf.Global().DownloadUseToken
	DownloadTokenExpire = conf.Global().DownloadTokenExpire
	DefaultDownload = conf.Global().DefaultDownload
	EnableGoogleAuth = conf.Global().EnableGoogleAuth
	Peers = conf.Global().Peers
	EnableMigrate = conf.Global().EnableMigrate
	EnableWebUpload = conf.Global().EnableWebUpload
	SupportGroupManage = conf.Global().SupportGroupManage
	DefaultScene = conf.Global().DefaultScene
	Host = conf.Global().Host
	AutoRepair = conf.Global().AutoRepair
	RefreshInterval = conf.Global().RefreshInterval
	AdminIps = conf.Global().AdminIps
	Addr = conf.Global().Addr
	ReadTimeout = conf.Global().ReadTimeout
	ReadHeaderTimeout = conf.Global().ReadHeaderTimeout
	WriteTimeout = conf.Global().WriteTimeout
	IdleTimeout = conf.Global().IdleTimeout

	if SupportGroupManage {
		staticHandler = http.StripPrefix("/"+Group+"/", http.FileServer(http.Dir(STORE_DIR)))
	} else {
		staticHandler = http.StripPrefix("/", http.FileServer(http.Dir(STORE_DIR)))
	}
}
