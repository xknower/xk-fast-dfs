package server

import (
	"../conf"
	jsoniter "github.com/json-iterator/go"
	"github.com/sjqzhang/goutil"
)

const (
	CONST_SMALL_FILE_SIZE          = conf.CONST_SMALL_FILE_SIZE
	CONST_STAT_FILE_COUNT_KEY      = conf.CONST_STAT_FILE_COUNT_KEY
	CONST_FILE_Md5_FILE_NAME       = conf.CONST_FILE_Md5_FILE_NAME
	CONST_BIG_UPLOAD_PATH_SUFFIX   = conf.CONST_BIG_UPLOAD_PATH_SUFFIX
	CONST_STAT_FILE_TOTAL_SIZE_KEY = conf.CONST_STAT_FILE_TOTAL_SIZE_KEY
	CONST_Md5_ERROR_FILE_NAME      = conf.CONST_Md5_ERROR_FILE_NAME
	CONST_Md5_QUEUE_FILE_NAME      = conf.CONST_Md5_QUEUE_FILE_NAME
	CONST_REMOME_Md5_FILE_NAME     = conf.CONST_REMOME_Md5_FILE_NAME
	CONST_MESSAGE_CLUSTER_IP       = conf.CONST_MESSAGE_CLUSTER_IP
	GO_FASTDFS_IP                  = conf.GO_FASTDFS_IP
	Go_FastDFS                     = conf.Go_FastDFS
)

// 项目应用目录
var (
	DOCKER_DIR                  = conf.DirDocker
	DATA_DIR                    = conf.DirData
	STORE_DIR                   = conf.DirStore
	LARGE_DIR_NAME              = conf.DirLargeName
	STATIC_DIR                  = conf.DirStatic
	LARGE_DIR                   = conf.DirLarge
	LOG_DIR                     = conf.DirLog
	STORE_DIR_NAME              = conf.STORE_DIR_NAME
	CONST_CONF_FILE_NAME        = conf.CONSTConfFileName
	CONST_LEVELDB_FILE_NAME     = conf.CONSTLevelDBFileName
	CONST_LOG_LEVELDB_FILE_NAME = conf.CONSTLevelDBFileNameLog
	CONST_SEARCH_FILE_NAME      = conf.CONSTSearchFileName
	CONST_STAT_FILE_NAME        = conf.CONSTStatFileName
	CONST_QUEUE_SIZE            = conf.CONSTQueueSize
	CONST_UPLOAD_COUNTER_KEY    = conf.CONSTUploadCounterKey
)

var (
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
	mail                conf.Mail
	//
	name                 string
	peerId               string
	scenes               []string
	readOnly             bool
	renameFile           bool
	syncTimeout          int64
	downloadDomain       string
	enableCustomPath     bool
	enableDistinctFile   bool
	enableTus            bool
	enableMergeSmallFile bool
	alarmUrl             string
	alarmReceivers       []string
	fileSumArithmetic    string
	extensions           []string
	uploadQueueSize      int
	uploadWorker         int
	syncWorker           int
	retryCount           int
	enableFsnotify       bool
)

// JSON 解析
var json = jsoniter.ConfigCompatibleWithStandardLibrary

// 工具
var util = &goutil.Common{}

func init() {
	//
	peers = conf.Global().Peers
	alarmReceivers = conf.Global().AlarmReceivers
	alarmUrl = conf.Global().AlarmUrl
	syncWorker = conf.Global().SyncWorker
	uploadWorker = conf.Global().UploadWorker
	uploadQueueSize = conf.Global().UploadQueueSize
	authUrl = conf.Global().AuthUrl
	enableDistinctFile = conf.Global().EnableDistinctFile
	retryCount = conf.Global().RetryCount
	name = conf.Global().Name
	group = conf.Global().Group
	syncTimeout = conf.Global().SyncTimeout
	readOnly = conf.Global().ReadOnly
	peerId = conf.Global().PeerId
	mail = conf.Global().Mail
	enableCrossOrigin = conf.Global().EnableCrossOrigin
	enableCustomPath = conf.Global().EnableCustomPath
	enableGoogleAuth = conf.Global().EnableGoogleAuth
	downloadTokenExpire = conf.Global().DownloadTokenExpire
	defaultScene = conf.Global().DefaultScene
	fileSumArithmetic = conf.Global().FileSumArithmetic
	host = conf.Global().Host
	downloadDomain = conf.Global().DownloadDomain
	renameFile = conf.Global().RenameFile
	enableMergeSmallFile = conf.Global().EnableMergeSmallFile
	extensions = conf.Global().Extensions
	supportGroupManage = conf.Global().SupportGroupManage
	enableFsnotify = conf.Global().EnableFsnotify
	enableMigrate = conf.Global().EnableMigrate
	autoRepair = conf.Global().AutoRepair
	enableTus = conf.Global().EnableTus
	scenes = conf.Global().Scenes
	readTimeout = conf.Global().ReadTimeout
	writeTimeout = conf.Global().WriteTimeout
	refreshInterval = conf.Global().RefreshInterval
	addr = conf.Global().Addr
}
