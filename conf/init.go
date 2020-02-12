package conf

import (
	"flag"
	"fmt"
	"github.com/sjqzhang/goutil"
	logger "github.com/sjqzhang/seelog"
	"os"
	"path/filepath"
	"strings"
)

var (
	//
	util = &goutil.Common{}
	// 日志
	Log logger.LoggerInterface
	//
	v = flag.Bool("v", false, "display version")
)

// 项目版本信息
var (
	VERSION     string
	BUILD_TIME  string
	GO_VERSION  string
	GIT_VERSION string
)

// 默认启动队列大小
var CONSTQueueSize = 10000

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
		Log.Warn(msg)
		fmt.Println(msg)
		os.Exit(1)
	}

	// 获取全局变量, 环境变量指定运行目录
	DirDocker = os.Getenv("GO_FASTDFS_DIR")
	if DirDocker != "" {
		if !strings.HasSuffix(DirDocker, "/") {
			DirDocker = DirDocker + "/"
		}
	}

	// 初始化 peer 并输出默认配置文件
	peerId := fmt.Sprintf("%d", util.RandInt(0, 9))
	if !util.FileExists(CONSTConfFileName) {
		var ip string
		if ip = os.Getenv("GO_FASTDFS_IP"); ip == "" {
			ip = util.GetPulicIP()
		}
		peer := "http://" + ip + ":8080"
		cfg := fmt.Sprintf(CONFIG_JSON, peerId, peer, peer)
		util.WriteFile(CONSTConfFileName, cfg)
	}

	// 初始化全局配置变量
	DirStore = DirDocker + STORE_DIR_NAME
	DirConf = DirDocker + CONF_DIR_NAME
	DirData = DirDocker + DATA_DIR_NAME
	DirLog = DirDocker + LOG_DIR_NAME
	DirStatic = DirDocker + STATIC_DIR_NAME
	DirLargeName = "haystack"
	DirLarge = DirStore + "/haystack"
	CONSTLevelDBFileName = DirData + "/fileserver.db"
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
		os.MkdirAll(folder, 0775)
	}

	// 初始化日志对象
	if _logger, err := logger.LoggerFromConfigAsBytes([]byte(LogConfigStr)); err != nil {
		panic(err)
	} else {
		logger.ReplaceLogger(_logger)
	}
	//
	if _logger, err := logger.LoggerFromConfigAsBytes([]byte(LogAccessConfigStr)); err == nil {
		Log = _logger
		Log.Info("succes init log access")
	} else {
		Log.Error(err.Error())
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
