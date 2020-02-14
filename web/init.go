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

const (
	CONST_SMALL_FILE_SIZE          = int(conf.CONST_SMALL_FILE_SIZE)
	CONST_STAT_FILE_COUNT_KEY      = conf.CONST_STAT_FILE_COUNT_KEY
	CONST_FILE_Md5_FILE_NAME       = conf.CONST_FILE_Md5_FILE_NAME
	CONST_BIG_UPLOAD_PATH_SUFFIX   = conf.CONST_BIG_UPLOAD_PATH_SUFFIX
	CONST_STAT_FILE_TOTAL_SIZE_KEY = conf.CONST_STAT_FILE_TOTAL_SIZE_KEY
	CONST_Md5_ERROR_FILE_NAME      = conf.CONST_Md5_ERROR_FILE_NAME
	CONST_Md5_QUEUE_FILE_NAME      = conf.CONST_Md5_QUEUE_FILE_NAME
	CONST_REMOME_Md5_FILE_NAME     = conf.CONST_REMOME_Md5_FILE_NAME
	CONST_MESSAGE_CLUSTER_IP       = conf.CONST_MESSAGE_CLUSTER_IP
	GO_FASTDFS_IP                  = conf.GO_FASTDFS_IP
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
