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
	"time"
)

const GO_FASTDFS_IP = "GO_FASTDFS_IP"

// JSON 解析
var json = jsoniter.ConfigCompatibleWithStandardLibrary

// 工具
var util = &goutil.Common{}

// 定义服务结构
type Service struct {
	ldb            *leveldb.DB         // 数据日志, 文件信息数据
	logDB          *leveldb.DB         // 操作日志
	statMap        *goutil.CommonMap   // 状态信息 state.json
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
