package server

import "../conf"

const (
	//
	CONST_STAT_FILE_COUNT_KEY      = "fileCount"
	CONST_FILE_Md5_FILE_NAME       = "files.md5"
	CONST_BIG_UPLOAD_PATH_SUFFIX   = "/big/upload/"
	CONST_STAT_FILE_TOTAL_SIZE_KEY = "totalSize"
	CONST_Md5_ERROR_FILE_NAME      = "errors.md5"
	CONST_Md5_QUEUE_FILE_NAME      = "queue.md5"
	CONST_REMOME_Md5_FILE_NAME     = "removes.md5"
	CONST_SMALL_FILE_SIZE          = int64(1024 * 1024)
	CONST_MESSAGE_CLUSTER_IP       = "Can only be called by the cluster ip or 127.0.0.1 or admin_ips(cfg.json),current ip:%s"
)

var (
	DATA_DIR                    = ""
	CONST_LEVELDB_FILE_NAME     = ""
	CONST_LOG_LEVELDB_FILE_NAME = ""
	CONST_SEARCH_FILE_NAME      = ""
	CONST_STAT_FILE_NAME        = ""
	CONST_QUEUE_SIZE            = 1
	STORE_DIR                   = ""
	DOCKER_DIR                  = ""
	LARGE_DIR                   = ""
	CONST_UPLOAD_COUNTER_KEY    = ""
	LARGE_DIR_NAME              = ""
	STORE_DIR_NAME              = ""
	LOG_DIR                     = ""
)

var (
	peers                []string
	alarmReceivers       []string
	alarmUrl             string
	syncWorker           int
	uploadWorker         int
	uploadQueueSize      int
	authUrl              string
	enableDistinctFile   bool
	retryCount           int
	name                 string
	group                string
	syncTimeout          int64
	readOnly             bool
	peerId               string
	mail                 conf.Mail
	enableCrossOrigin    bool
	enableCustomPath     bool
	enableGoogleAuth     bool
	downloadTokenExpire  int
	defaultScene         string
	fileSumArithmetic    string
	host                 string
	downloadDomain       string
	renameFile           bool
	enableMergeSmallFile bool
	extensions           []string
	supportGroupManage   bool
	enableFsnotify       bool
	enableMigrate        bool
	autoRepair           bool
	enableTus            bool
	scenes               []string
	readTimeout          int
	writeTimeout         int
	refreshInterval      int
	addr                 string
)

func init() {
	DATA_DIR = conf.DirData
	CONST_LEVELDB_FILE_NAME = conf.CONSTLevelDBFileName
	CONST_LOG_LEVELDB_FILE_NAME = conf.CONSTLevelDBFileNameLog
	CONST_STAT_FILE_NAME = conf.CONSTStatFileName
	CONST_SEARCH_FILE_NAME = conf.CONSTSearchFileName
	CONST_QUEUE_SIZE = conf.CONSTQueueSize
	STORE_DIR = conf.DirStore
	DOCKER_DIR = conf.DirDocker
	LARGE_DIR = conf.DirLarge
	CONST_UPLOAD_COUNTER_KEY = conf.CONSTUploadCounterKey
	LARGE_DIR_NAME = conf.DirLargeName
	STORE_DIR_NAME = conf.STORE_DIR_NAME
	LOG_DIR = conf.DirLog
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
