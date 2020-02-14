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
	AdminIps                    []string
	CONST_CONF_FILE_NAME        = ""
)

var (
	// 集群": "集群列表,注意为了高可用，IP必须不能是同一个,同一不会自动备份，且不能为127.0.0.1,且必须为内网IP，默认自动生成"
	peers []string
	// "告警接收邮件列表": "接收人数组",
	alarmReceivers []string
	// "告警接收URL": "方法post,参数:subject,message",
	alarmUrl        string
	syncWorker      int // 文件下载队列处理线程数
	uploadWorker    int // 处理上传队列处理线程数
	uploadQueueSize int
	// "认证url": "当url不为空时生效,注意:普通上传中使用http参数 auth_token 作为认证参数, 在断点续传中通过HTTP头Upload-Metadata中的auth_token作为认证参数,认证流程参考认证架构图",
	authUrl string
	// "文件是否去重": "默认去重",
	enableDistinctFile bool
	retryCount         int
	// "服务名": "用于区别不同的主机",
	name string
	// "组号": "用于区别不同的集群(上传或下载)与support_group_manage配合使用,带在下载路径中",
	group string
	// "同步单一文件超时时间（单位秒）": "默认为0,程序自动计算，在特殊情况下，自已设定",
	syncTimeout int64
	// "本机是否只读": "默认可读可写",
	readOnly bool
	// "PeerID": "集群内唯一,请使用0-9的单字符，默认自动生成",
	peerId string
	// "邮件配置": "",
	mail conf.Mail
	// "是否开启跨站访问": "默认开启",
	enableCrossOrigin bool
	// "是否支持非日期路径": "默认支持非日期路径,也即支持自定义路径,需要上传文件时指定path",
	enableCustomPath bool
	// "是否开启Google认证，实现安全的上传、下载": "默认不开启",
	enableGoogleAuth bool
	// "下载token过期时间": "单位秒",
	downloadTokenExpire int
	// "默认场景": "默认default",
	defaultScene string
	// "文件去重算法md5可能存在冲突，默认md5": "sha1|md5",
	fileSumArithmetic string
	// "本主机地址": "本机http地址,默认自动生成(注意端口必须与addr中的端口一致），必段为内网，自动生成不为内网请自行修改，下同",
	host string
	// "下载域名": "用于外网下载文件的域名,不包含http://",
	downloadDomain string
	// "是否自动重命名": "默认不自动重命名,使用原文件名",
	renameFile bool
	// "是否合并小文件": "默认不合并,合并可以解决inode不够用的情况（当前对于小于1M文件）进行合并",
	enableMergeSmallFile bool
	// "允许后缀名": "允许可以上传的文件后缀名，如jpg,jpeg,png等。留空允许所有。",
	extensions []string
	// "是否支持按组(集群)管理, 主要用途是Nginx支持多集群": "默认支持,不支持时路径为http://10.1.5.4:8080/action,支持时为http://10.1.5.4:8080/group(配置中的group参数)/action,action为动作名，如status,delete,sync等",
	supportGroupManage bool
	enableFsnotify     bool
	// "是否启用迁移": "默认不启用",
	enableMigrate bool
	// "是否自动修复": "在超过1亿文件时出现性能问题，取消此选项，请手动按天同步，请查看FAQ",
	autoRepair bool
	// "是否开启断点续传": "默认开启",
	enableTus bool
	// "场景列表": "当设定后，用户指的场景必项在列表中，默认不做限制(注意: 如果想开启场景认功能, 格式如下: '场景名:googleauth_secret' 如 default:N7IET373HB2C5M6D ",
	scenes       []string
	readTimeout  int
	writeTimeout int
	// "重试同步失败文件的时间": "单位秒",
	refreshInterval int
	// "绑定端号": "端口",
	addr string
)

func init() {
	DATA_DIR = conf.DirData
	CONST_LEVELDB_FILE_NAME = conf.CONSTLevelDBFileName
	CONST_LOG_LEVELDB_FILE_NAME = conf.CONSTLevelDBFileNameLog
	CONST_STAT_FILE_NAME = conf.CONSTStatFileName
	CONST_SEARCH_FILE_NAME = conf.CONSTSearchFileName
	CONST_QUEUE_SIZE = conf.CONSTQueueSize
	CONST_CONF_FILE_NAME = conf.CONSTConfFileName
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
	AdminIps = conf.Global().AdminIps
}
