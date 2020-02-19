// 配置参数初始化
package conf

import (
	"flag"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/sjqzhang/goutil"
	slog "github.com/sjqzhang/seelog"
	"os"
	"path/filepath"
	"strings"
	"unsafe"
)

var (
	//
	util = &goutil.Common{}
	// 日志
	Log slog.LoggerInterface
	//
	v = flag.Bool("v", false, "display version")
	// 配置文件名
	FileName string
	//
	ptr unsafe.Pointer
	// JSON 解析
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

// 项目版本信息
var (
	VERSION     string
	BUILD_TIME  string
	GO_VERSION  string
	GIT_VERSION string
)

const (
	CONST_SMALL_FILE_SIZE = int64(1024 * 1024) // 小文件定义 1M

	CONST_FILE_Md5_FILE_NAME   = "files.md5"   // 文件信息操作标识 保存文件处理
	CONST_Md5_ERROR_FILE_NAME  = "errors.md5"  // 文件信息操作标识 错误文件处理
	CONST_Md5_QUEUE_FILE_NAME  = "queue.md5"   // 文件信息操作标识 队列文件处理 , 下载队列
	CONST_REMOME_Md5_FILE_NAME = "removes.md5" // 文件信息操作标识 删除文件处理
	//
	CONST_STAT_FILE_TOTAL_SIZE_KEY = "totalSize"
	CONST_STAT_FILE_COUNT_KEY      = "fileCount"

	CONST_BIG_UPLOAD_PATH_SUFFIX = "/big/upload/"

	GO_FASTDFS_DIR = "GO_FASTDFS_DIR"
	GO_FASTDFS_IP  = "GO_FASTDFS_IP"
	Go_FastDFS     = "Go-FastDFS"
)

const (
	STORE_DIR_NAME  = "files"  // 文件存储目录
	LOG_DIR_NAME    = "log"    // 日志目录
	DATA_DIR_NAME   = "data"   // 数据目录
	CONF_DIR_NAME   = "conf"   // 配置文件目录
	STATIC_DIR_NAME = "static" // 静态资源目录
)

// 项目运行目录定义
var (
	FOLDERS = []string{DirData, DirStore, DirConf, DirStatic}
	// 项目运行目录
	DirDocker = ""
	// 数据目录
	DirData = DATA_DIR_NAME
	// 文件存储目录
	DirStore = STORE_DIR_NAME
	// 配置文件目录
	DirConf = CONF_DIR_NAME
	// 静态资源目录
	DirStatic = STATIC_DIR_NAME
	//
	DirLargeName = "haystack"
	// 日志目录
	DirLog = LOG_DIR_NAME
	//
	DirLarge                = DirStore + "/haystack"
	CONSTLevelDBFileName    = DirData + "/data.db"   // 文件信息数据库
	CONSTLevelDBFileNameLog = DirData + "/log.db"    // 文件日志数据库
	CONSTStatFileName       = DirData + "/stat.json" // 应用状态文件
	CONSTConfFileName       = DirConf + "/cfg.json"  // 应用配置文件
	CONSTSearchFileName     = DirData + "/search.txt"
	CONSTUploadCounterKey   = "__CONST_UPLOAD_COUNTER_KEY__"
)

// 默认启动队列大小
var CONSTQueueSize = 10000

// 全局变量配置
var (
	LogConfigStr = `
<seelog type="asynctimer" asyncinterval="1000" minlevel="trace" maxlevel="error">  
	<outputs formatid="common">  
		<buffered formatid="common" size="1048576" flushperiod="1000">  
			<rollingfile type="size" filename="{DirDocker}log/xk.log" maxsize="104857600" maxrolls="10"/>  
		</buffered>
	</outputs>  	  
	 <formats>
		 <format id="common" format="%Date %Time [%LEV] [%File:%Line] [%Func] %Msg%n" />  
	 </formats>  
</seelog>
`
	LogAccessConfigStr = `
<seelog type="asynctimer" asyncinterval="1000" minlevel="trace" maxlevel="error">  
	<outputs formatid="common">  
		<buffered formatid="common" size="1048576" flushperiod="1000">  
			<rollingfile type="size" filename="{DirDocker}log/access.log" maxsize="104857600" maxrolls="10"/>  
		</buffered>
	</outputs>  	  
	 <formats>
		 <format id="common" format="%Date %Time [%LEV] [%File:%Line] [%Func] %Msg%n" />  
	 </formats>  
</seelog>
`
)

//
func init() {
	//
	flag.Parse()
	if *v {
		fmt.Printf("%s\n%s\n%s\n%s\n", VERSION, BUILD_TIME, GO_VERSION, GIT_VERSION)
		os.Exit(0)
	}

	// 加载配置文件
	appDir, e1 := filepath.Abs(filepath.Dir(os.Args[0]))
	curDir, e2 := filepath.Abs(".")
	if e1 == nil && e2 == nil && appDir != curDir {
		msg := fmt.Sprintf("please change directory to '%s' start fileserver\n", appDir)
		msg = msg + fmt.Sprintf("请切换到 '%s' 目录启动 fileserver ", appDir)
		_ = Log.Warn(msg)
		fmt.Println(msg)
		os.Exit(1)
	}

	// 获取全局变量, 环境变量指定运行目录
	DirDocker = os.Getenv(GO_FASTDFS_DIR)
	if DirDocker != "" {
		if !strings.HasSuffix(DirDocker, "/") {
			DirDocker = DirDocker + "/"
		}
	}

	// 初始化全局配置变量
	DirStore = DirDocker + STORE_DIR_NAME
	DirConf = DirDocker + CONF_DIR_NAME
	DirData = DirDocker + DATA_DIR_NAME
	DirLog = DirDocker + LOG_DIR_NAME
	DirStatic = DirDocker + STATIC_DIR_NAME
	DirLargeName = "haystack"
	DirLarge = DirStore + "/haystack"
	CONSTLevelDBFileName = DirData + "/data.db"
	CONSTLevelDBFileNameLog = DirData + "/log.db"
	CONSTStatFileName = DirData + "/stat.json"
	CONSTConfFileName = DirConf + "/cfg.json"
	CONSTSearchFileName = DirData + "/search.txt"
	//
	FOLDERS = []string{DirData, DirStore, DirConf, DirStatic}
	//
	LogAccessConfigStr = strings.Replace(LogAccessConfigStr, "{DirDocker}", DirDocker, -1)
	//
	LogConfigStr = strings.Replace(LogConfigStr, "{DirDocker}", DirDocker, -1)
	// 创建目录
	for _, folder := range FOLDERS {
		_ = os.MkdirAll(folder, 0775)
	}

	// 初始化日志对象
	if _logger, err := slog.LoggerFromConfigAsBytes([]byte(LogConfigStr)); err != nil {
		panic(err)
	} else {
		_ = slog.ReplaceLogger(_logger)
	}
	//
	if _logger, err := slog.LoggerFromConfigAsBytes([]byte(LogAccessConfigStr)); err == nil {
		Log = _logger
		Log.Info("succes init log access")
	} else {
		_ = Log.Error(err.Error())
	}

	// 初始化 peer 并输出默认配置文件
	peerId := fmt.Sprintf("%d", util.RandInt(0, 9))
	if !util.FileExists(CONSTConfFileName) {
		var ip string
		if ip = os.Getenv(GO_FASTDFS_IP); ip == "" {
			ip = util.GetPulicIP()
		}
		peer := "http://" + ip + ":8080"
		cfg := fmt.Sprintf(CONFIG_JSON, peerId, peer, peer)
		util.WriteFile(CONSTConfFileName, cfg)
	}

	// 加载配置文件初始化配置
	ParseConfig(CONSTConfFileName)
	//
	if Global().QueueSize == 0 {
		//
		Global().QueueSize = CONSTQueueSize
	}
	if Global().PeerId == "" {
		Global().PeerId = peerId
	}
	Log.Info(Global())
}
