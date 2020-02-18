// 服务端, 实体结构定义
package server

import (
	"../en"
	"fmt"
	_ "github.com/eventials/go-tus"
	"github.com/sjqzhang/goutil"
	slog "github.com/sjqzhang/seelog"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	_ "net/http/pprof"
)

// 定义服务结构
type Service struct {
	ldb            *leveldb.DB         // 文件信息数据库, 数据日志, 文件信息数据 [data.db]
	logDB          *leveldb.DB         // 文件日志数据库, 操作日志 [log.db]
	queueToPeers   chan en.FileInfo    // 文件上传处理队列
	queueFromPeers chan en.FileInfo    // 文件下载处理队列
	queueFileLog   chan *en.FileLog    // 文件日志处理队列
	queueUpload    chan en.WrapReqResp // HTTP文件上传处理队列
	statMap        *goutil.CommonMap   // 处理状态信息 state.json
	sumMap         *goutil.CommonMap   //
	rtMap          *goutil.CommonMap   // 处理[HTTP文件上传处理队列]上传文件 (读写锁)
	lockMap        *goutil.CommonMap   // 读写锁
	sceneMap       *goutil.CommonMap   //
	searchMap      *goutil.CommonMap   // 加载搜索字典 (读写锁)
	curDate        string              // 处理时间(定时清理备份数据)
	host           string              // http://IP:PORT
	name           string              // 服务名称
	group          string              // 分组(路由)名称
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

	server.statMap.Put(CONST_STAT_FILE_COUNT_KEY, int64(0))
	server.statMap.Put(CONST_STAT_FILE_TOTAL_SIZE_KEY, int64(0))
	server.statMap.Put(util.GetToDay()+"_"+CONST_STAT_FILE_COUNT_KEY, int64(0))
	server.statMap.Put(util.GetToDay()+"_"+CONST_STAT_FILE_TOTAL_SIZE_KEY, int64(0))
	server.curDate = util.GetToDay()

	// 数据库
	opts := &opt.Options{
		CompactionTableSize: 1024 * 1024 * 20,
		WriteBuffer:         1024 * 1024 * 20,
	}
	// /data.db
	if server.ldb, err = leveldb.OpenFile(CONST_LEVELDB_FILE_NAME, opts); err != nil {
		fmt.Println(fmt.Sprintf("open db file %s fail,maybe has opening", CONST_LEVELDB_FILE_NAME))
		_ = slog.Error(err)
		panic(err)
	}
	// log.db
	server.logDB, err = leveldb.OpenFile(CONST_LOG_LEVELDB_FILE_NAME, opts)
	if err != nil {
		fmt.Println(fmt.Sprintf("open db file %s fail,maybe has opening", CONST_LOG_LEVELDB_FILE_NAME))
		_ = slog.Error(err)
		panic(err)
	}

	return server, nil
}
