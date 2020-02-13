package server

import (
	"../en"
	"fmt"
	"github.com/astaxie/beego/httplib"
	_ "github.com/eventials/go-tus"
	jsoniter "github.com/json-iterator/go"
	"github.com/sjqzhang/goutil"
	slog "github.com/sjqzhang/seelog"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

const GO_FASTDFS_IP = "GO_FASTDFS_IP"

// JSON 解析
var json = jsoniter.ConfigCompatibleWithStandardLibrary

//
var util = &goutil.Common{}

// 定义服务结构
type Service struct {
	ldb            *leveldb.DB         // 数据日志, 文件信息数据
	logDB          *leveldb.DB         // 操作日志
	statMap        *goutil.CommonMap   // 状态信息
	sumMap         *goutil.CommonMap   //
	rtMap          *goutil.CommonMap   // 上传文件 (读写锁)
	queueToPeers   chan en.FileInfo    // 文件信息处理队列, 文件信息保存
	queueFromPeers chan en.FileInfo    // 文件信息处理队列, 文件信息获取
	queueFileLog   chan *en.FileLog    // 文件日志处理队列
	queueUpload    chan en.WrapReqResp // HTTP文件上传处理队列
	lockMap        *goutil.CommonMap   // 读写锁
	sceneMap       *goutil.CommonMap   //
	searchMap      *goutil.CommonMap   // 加载搜索字典 (读写锁)
	curDate        string
	host           string // http://IP:PORT
	name           string // 服务名称
	group          string // 分组(路由)名称
}

// 获取服务名称
func (server Service) GetServerName() string {
	return server.name
}

// 获取访问路由名称, 未配置使用服务名称
func (server Service) GetGroupRouteName() string {
	if server.group == "" {
		return server.name
	}
	return server.group
}

//
func NewService() (server *Service, err error) {
	server = &Service{
		statMap:        goutil.NewCommonMap(0),
		lockMap:        goutil.NewCommonMap(0),
		rtMap:          goutil.NewCommonMap(0),
		sceneMap:       goutil.NewCommonMap(0),
		searchMap:      goutil.NewCommonMap(0),
		queueToPeers:   make(chan en.FileInfo, CONST_QUEUE_SIZE),
		queueFromPeers: make(chan en.FileInfo, CONST_QUEUE_SIZE),
		queueFileLog:   make(chan *en.FileLog, CONST_QUEUE_SIZE),
		queueUpload:    make(chan en.WrapReqResp, 100),
		sumMap:         goutil.NewCommonMap(365 * 3),
	}

	defaultTransport := &http.Transport{
		DisableKeepAlives:   true,
		Dial:                httplib.TimeoutDialer(time.Second*15, time.Second*300),
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
	}
	settings := httplib.BeegoHTTPSettings{
		UserAgent:        "Go-FastDFS",
		ConnectTimeout:   15 * time.Second,
		ReadWriteTimeout: 15 * time.Second,
		Gzip:             true,
		DumpBody:         true,
		Transport:        defaultTransport,
	}

	httplib.SetDefaultSetting(settings)
	server.statMap.Put(CONST_STAT_FILE_COUNT_KEY, int64(0))
	server.statMap.Put(CONST_STAT_FILE_TOTAL_SIZE_KEY, int64(0))
	server.statMap.Put(util.GetToDay()+"_"+CONST_STAT_FILE_COUNT_KEY, int64(0))
	server.statMap.Put(util.GetToDay()+"_"+CONST_STAT_FILE_TOTAL_SIZE_KEY, int64(0))
	server.curDate = util.GetToDay()
	opts := &opt.Options{
		CompactionTableSize: 1024 * 1024 * 20,
		WriteBuffer:         1024 * 1024 * 20,
	}

	//
	if server.ldb, err = leveldb.OpenFile(CONST_LEVELDB_FILE_NAME, opts); err != nil {
		fmt.Println(fmt.Sprintf("open db file %s fail,maybe has opening", CONST_LEVELDB_FILE_NAME))
		slog.Error(err)
		panic(err)
	}

	//
	server.logDB, err = leveldb.OpenFile(CONST_LOG_LEVELDB_FILE_NAME, opts)
	if err != nil {
		fmt.Println(fmt.Sprintf("open db file %s fail,maybe has opening", CONST_LOG_LEVELDB_FILE_NAME))
		slog.Error(err)
		panic(err)
	}

	return server, nil
}

func (server *Service) Start() {
	// 初始化相关参数
	server.initComponent(false)

	// 启动
	server.startComponent()
}

// 相关参数配置初始化
// ---------- ---------- ----------
// isReload 是否为重新加载
// ---------- ---------- ----------
func (server *Service) initComponent(isReload bool) {
	// -> IP
	var ip string
	if ip := os.Getenv(GO_FASTDFS_IP); ip == "" {
		ip = util.GetPulicIP()
	}
	// -> HOST
	if host == "" {
		if len(strings.Split(addr, ":")) == 2 {
			server.host = fmt.Sprintf("http://%s:%s", ip, strings.Split(addr, ":")[1])
			host = server.host
		}
	} else {
		if strings.HasPrefix(host, "http") {
			server.host = host
		} else {
			server.host = "http://" + host
		}
	}

	// -> NAME GROUP
	if supportGroupManage {
		server.group = "/" + group
	}
	server.name = name

	// -> 节点名
	rex, _ := regexp.Compile("\\d+\\.\\d+\\.\\d+\\.\\d+")
	var prs []string
	for _, peer := range prs {
		if util.Contains(ip, rex.FindAllString(peer, -1)) ||
			util.Contains("127.0.0.1", rex.FindAllString(peer, -1)) {
			continue
		}
		if strings.HasPrefix(peer, "http") {
			peers = append(peers, peer)
		} else {
			peers = append(peers, "http://"+peer)
		}
	}
	peers = prs

	if !isReload {
		// -> 第一次加载时, 检测并格式化, stat.json 状态文件
		server.formatStatInfo()
		if enableTus {
			// -> 第一次加载时, 初始化 Tus
			server.initTus()
		}
	}

	// -> 场景解析
	for _, s := range scenes {
		kv := strings.Split(s, ":")
		if len(kv) == 2 {
			server.sceneMap.Put(kv[0], kv[1])
		}
	}

	// -> 检验并初始化相关参数
	if readTimeout == 0 {
		readTimeout = 60 * 10
	}
	if writeTimeout == 0 {
		writeTimeout = 60 * 10
	}
	if syncWorker == 0 {
		syncWorker = 200
	}
	if uploadWorker == 0 {
		uploadWorker = runtime.NumCPU() + 4
		if runtime.NumCPU() < 4 {
			uploadWorker = 8
		}
	}
	if uploadQueueSize == 0 {
		uploadQueueSize = 200
	}
	if retryCount == 0 {
		retryCount = 3
	}
}

// 启动服务组件
// ---------- ---------- ----------
func (server *Service) startComponent() {
	go func() {
		// 00 开始服务 -> 检测文件并加载到处理队列 (定时, 自动检测系统状态 (文件和结点状态))
		for {
			server.checkFileAndSendToPeer(util.GetToDay(), CONST_Md5_ERROR_FILE_NAME, false)
			//fmt.Println("CheckFileAndSendToPeer")
			time.Sleep(time.Second * time.Duration(refreshInterval))
			//util.RemoveEmptyDir(STORE_DIR)
		}
	}()

	// 01 开启服务 -> 定期清理及备份数据服务
	go server.cleanAndBackUp()
	// 02 开启服务 -> 检测集群状态服务
	go server.checkClusterStatus()
	// 03 开启服务
	go server.loadQueueSendToPeer()
	// 04 开启服务
	go server.consumerPostToPeer()
	// 05 开启服务 -> 处理日志队列服务
	go server.consumerLog()
	// 06 开启服务 -> 处理文件下载队列服务
	go server.consumerDownLoad()
	// 07 开启服务 -> 处理文件上传队列服务
	go server.consumerUpload()
	// 08 开启服务 -> 清除过期(下载)文件服务
	go server.removeDownloading()

	// 支持按组(集群)管理
	if enableFsnotify {
		// 09 开启服务 -> 监控文件变更服务
		go server.watchFilesChange()
	}
	// 10 开启服务 ->
	go server.loadSearchDict()

	if enableMigrate {
		// 11 开启服务 -> 数据迁移服务
		go server.repairFileInfoFromFile()
	}

	if autoRepair {
		go func() {
			for {
				time.Sleep(time.Minute * 3)
				// 12 开启服务 -> 数据修复更新服务
				server.autoRepair(false)
				time.Sleep(time.Minute * 60)
			}
		}()
	}

	go func() {
		for {
			// 13 开启服务 -> 定时强制释放内存, force free memory
			time.Sleep(time.Minute * 1)
			debug.FreeOSMemory()
		}
	}()
}
