// 配置参数结构定义
package conf

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync/atomic"
	"unsafe"
)

// 全局配置
type GlobalConfig struct {
	Host                string   `json:"host"`                  // 本地主机(HTTP)地址, http://ip:port
	Addr                string   `json:"addr"`                  // 绑定监听端口号
	Group               string   `json:"group"`                 // 集群分组名, 根据分组名区分集群
	Peers               []string `json:"peers"`                 // 集群列表
	DefaultScene        string   `json:"default_scene"`         // 默认场景, 默认为 default
	EnableCrossOrigin   bool     `json:"enable_cross_origin"`   // 是否开启跨域访问, 默认开启
	EnableGoogleAuth    bool     `json:"enable_google_auth"`    // 是否开启Google认证(安全上传下载), 默认关闭
	EnableMigrate       bool     `json:"enable_migrate"`        // 是否启用迁移, 默认关闭
	SupportGroupManage  bool     `json:"support_group_manage"`  // 是否支持集群资源按组管理, 默认支持, 路由需要带分组名
	AuthUrl             string   `json:"auth_url"`              // 认证URL, 为空不认证 (一般上传认证通过 参数auth_token验证, 断点续传 HTTP头 Upload-Metadata中的 auth_token 认证)
	DownloadTokenExpire int      `json:"download_token_expire"` // 下载Token过期时间, 单位 秒
	RefreshInterval     int      `json:"refresh_interval"`      // 重试同步失败文件的时间, 单位 秒
	AutoRepair          bool     `json:"auto_repair"`           // 是否开启自动修复, 默认打开
	ReadTimeout         int      `json:"read_timeout"`
	WriteTimeout        int      `json:"write_timeout"`
	Mail                Mail     `json:"mail"`
	QueueSize           int      `json:"queue_size"`
	EnableDiskCache     bool     `json:"enable_disk_cache"`
	ConnectTimeout      bool     `json:"connect_timeout"`
	*ServerConfig       `json:"s"`
	*WebConfig          `json:"w"`
}

// 服务端配置
type ServerConfig struct {
	Name                 string   `json:"name"`                    // 服务名, 用于区别不同的主机
	PeerId               string   `json:"peer_id"`                 // (集群)节点标识符号, 集群中唯一
	Scenes               []string `json:"scenes"`                  // 场景列表
	ReadOnly             bool     `json:"read_only"`               // 本机是否只读, 默认可读可写
	RenameFile           bool     `json:"rename_file"`             // 是否自动重命名, 默认关闭, 使用原文件名
	SyncTimeout          int64    `json:"sync_timeout"`            // 同步单一文件超时时间, 单位 秒, 默认零程序自动计算
	DownloadDomain       string   `json:"download_domain"`         // 下载域名, 用于外网下载 (不包含 http://)
	EnableCustomPath     bool     `json:"enable_custom_path"`      // 是否支持非日期路径, 默认支持, 上传文件时指定路径(path)
	EnableDistinctFile   bool     `json:"enable_distinct_file"`    // 是否开启去重, 默认开始
	EnableTus            bool     `json:"enable_tus"`              // 是否开启断点续传, 默认开始
	EnableMergeSmallFile bool     `json:"enable_merge_small_file"` // 是否合并小文件, 默认不合并
	AlarmUrl             string   `json:"alarm_url"`               // 告警接收URL
	AlarmReceivers       []string `json:"alarm_receivers"`         // 告警接收邮件列表
	FileSumArithmetic    string   `json:"file_sum_arithmetic"`     // 文件去重算法md5可能存在冲突, 默认md5 (sha1|md5)
	Extensions           []string `json:"extensions"`              // 允许后缀名, 允许可以上传的文件后缀名, 空白则不限制
	UploadQueueSize      int      `json:"upload_queue_size"`
	UploadWorker         int      `json:"upload_worker"`
	SyncWorker           int      `json:"sync_worker"`
	RetryCount           int      `json:"retry_count"`
	EnableFsnotify       bool     `json:"enable_fsnotify"`
}

// HTTP WEB 配置
type WebConfig struct {
	AdminIps           []string `json:"admin_ips"`            // 管理IP列表, 用于管理集的IP白名单
	EnableWebUpload    bool     `json:"enable_web_upload"`    // 是否支持web上传, 默认支持
	EnableDownloadAuth bool     `json:"enable_download_auth"` // 下载是否认证, 默认不认证 (auth_url 不为空是才生效)
	ShowDir            bool     `json:"show_dir"`             // 是否显示目录, 默认显示, 上线时请关闭
	DefaultDownload    bool     `json:"default_download"`     // 默认是否下载, 默认下载
	DownloadUseToken   bool     `json:"download_use_token"`   // 下载是否需带token, 默认不需要
	ReadHeaderTimeout  int      `json:"read_header_timeout"`
	IdleTimeout        int      `json:"idle_timeout"`
}

//
func Global() *GlobalConfig {
	return (*GlobalConfig)(atomic.LoadPointer(&ptr))
}

//
func Server() *ServerConfig {
	return Global().ServerConfig
}

//
func Web() *WebConfig {
	return Global().WebConfig
}

type Mail struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Host     string `json:"host"`
}

// 解析配置文件
func ParseConfig(filePath string) {
	var data []byte
	if filePath == "" {
		// 使用默认配置项目
		data = []byte(strings.TrimSpace(CONFIG_JSON))
	} else {
		// 加载配置文件

		file, err := os.Open(filePath)
		if err != nil {
			panic(fmt.Sprintln("open file path:", filePath, "error:", err))
		}
		defer file.Close()
		//
		FileName = filePath
		data, err = ioutil.ReadAll(file)
		if err != nil {
			panic(fmt.Sprintln("file path:", filePath, " read all error:", err))
		}
	}

	// 加载全局配置
	var c GlobalConfig
	if err := json.Unmarshal(data, &c); err != nil {
		panic(fmt.Sprintln("file path:", filePath, "json unmarshal error:", err))
	}

	//
	Log.Info(c)
	atomic.StorePointer(&ptr, unsafe.Pointer(&c))
	Log.Info("config parse success")
}
