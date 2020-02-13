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

func init() {
	DATA_DIR = conf.DirData
	CONST_LEVELDB_FILE_NAME = conf.CONSTLevelDBFileName
	CONST_LOG_LEVELDB_FILE_NAME = conf.CONSTLevelDBFileNameLog
	CONST_STAT_FILE_NAME = conf.CONSTStatFileName
	CONST_QUEUE_SIZE = conf.CONSTQueueSize
	STORE_DIR = conf.DirStore
	DOCKER_DIR = conf.DirDocker
	LARGE_DIR = conf.DirLarge
	CONST_UPLOAD_COUNTER_KEY = conf.CONSTUploadCounterKey
	LARGE_DIR_NAME = conf.DirLargeName
	STORE_DIR_NAME = conf.STORE_DIR_NAME
	LOG_DIR = conf.DirLog
	//
}
