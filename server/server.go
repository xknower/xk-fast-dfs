package server

import (
	"../en"
	"../web"
	"bytes"
	"errors"
	"fmt"
	"github.com/astaxie/beego/httplib"
	_ "github.com/eventials/go-tus"
	jsoniter "github.com/json-iterator/go"
	"github.com/sjqzhang/goutil"
	slog "github.com/sjqzhang/seelog"
	"github.com/sjqzhang/tusd"
	"github.com/sjqzhang/tusd/filestore"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	dbutil "github.com/syndtr/goleveldb/leveldb/util"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path"
	"regexp"
	"runtime"
	"runtime/debug"
	"strconv"
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

// 检测并格式化, stat.json 状态文件
func (server *Service) formatStatInfo() {
	var (
		data  []byte
		err   error
		count int64
		stat  map[string]interface{}
	)
	if util.FileExists(CONST_STAT_FILE_NAME) {
		if data, err = util.ReadBinFile(CONST_STAT_FILE_NAME); err != nil {
			slog.Error(err)
		} else {
			if err = json.Unmarshal(data, &stat); err != nil {
				slog.Error(err)
			} else {
				for k, v := range stat {
					switch v.(type) {
					case float64:
						vv := strings.Split(fmt.Sprintf("%f", v), ".")[0]
						if count, err = strconv.ParseInt(vv, 10, 64); err != nil {
							slog.Error(err)
						} else {
							server.statMap.Put(k, count)
						}
					default:
						server.statMap.Put(k, v)
					}
				}
			}
		}
	} else {
		server.repairStatByDate(util.GetToDay())
	}
}

//
func (server *Service) repairStatByDate(date string) en.StatDateFileInfo {
	defer func() {
		if re := recover(); re != nil {
			buffer := debug.Stack()
			slog.Error("RepairStatByDate")
			slog.Error(re)
			slog.Error(string(buffer))
		}
	}()
	var (
		err       error
		keyPrefix string
		fileInfo  en.FileInfo
		fileCount int64
		fileSize  int64
		stat      en.StatDateFileInfo
	)
	keyPrefix = "%s_%s_"
	keyPrefix = fmt.Sprintf(keyPrefix, date, CONST_FILE_Md5_FILE_NAME)
	iter := server.logDB.NewIterator(dbutil.BytesPrefix([]byte(keyPrefix)), nil)
	defer iter.Release()
	for iter.Next() {
		if err = json.Unmarshal(iter.Value(), &fileInfo); err != nil {
			continue
		}
		fileCount = fileCount + 1
		fileSize = fileSize + fileInfo.Size
	}
	server.statMap.Put(date+"_"+CONST_STAT_FILE_COUNT_KEY, fileCount)
	server.statMap.Put(date+"_"+CONST_STAT_FILE_TOTAL_SIZE_KEY, fileSize)
	server.SaveStat()
	stat.Date = date
	stat.FileCount = fileCount
	stat.TotalSize = fileSize
	return stat
}

// 初始化 Tus
func (server *Service) initTus() {
	var (
		err     error
		fileLog *os.File
		bigDir  string
	)
	BIG_DIR := STORE_DIR + "/_big/" + peerId
	os.MkdirAll(BIG_DIR, 0775)
	os.MkdirAll(LOG_DIR, 0775)
	store := filestore.FileStore{
		Path: BIG_DIR,
	}
	if fileLog, err = os.OpenFile(LOG_DIR+"/tusd.log", os.O_CREATE|os.O_RDWR, 0666); err != nil {
		slog.Error(err)
		panic("initTus")
	}

	go func() {
		for {
			if fi, err := fileLog.Stat(); err != nil {
				slog.Error(err)
			} else {
				if fi.Size() > 1024*1024*500 {
					//500M
					util.CopyFile(LOG_DIR+"/tusd.log", LOG_DIR+"/tusd.log.2")
					fileLog.Seek(0, 0)
					fileLog.Truncate(0)
					fileLog.Seek(0, 2)
				}
			}
			time.Sleep(time.Second * 30)
		}
	}()
	l := log.New(fileLog, "[tusd] ", log.LstdFlags)
	bigDir = CONST_BIG_UPLOAD_PATH_SUFFIX
	if supportGroupManage {
		bigDir = fmt.Sprintf("/%s%s", group, CONST_BIG_UPLOAD_PATH_SUFFIX)
	}
	composer := tusd.NewStoreComposer()
	// support raw tus upload and download
	store.GetReaderExt = func(id string) (io.Reader, error) {
		var (
			offset int64
			err    error
			length int
			buffer []byte
			fi     *en.FileInfo
			fn     string
		)
		if fi, err = server.GetFileInfoFromLevelDB(id); err != nil {
			slog.Error(err)
			return nil, err
		} else {
			if authUrl != "" {
				fileResult := util.JsonEncodePretty(server.BuildFileResult(fi, nil))
				bufferReader := bytes.NewBuffer([]byte(fileResult))
				return bufferReader, nil
			}
			fn = fi.Name
			if fi.ReName != "" {
				fn = fi.ReName
			}
			fp := DOCKER_DIR + fi.Path + "/" + fn
			if util.FileExists(fp) {
				slog.Info(fmt.Sprintf("download:%s", fp))
				return os.Open(fp)
			}
			ps := strings.Split(fp, ",")
			if len(ps) > 2 && util.FileExists(ps[0]) {
				if length, err = strconv.Atoi(ps[2]); err != nil {
					return nil, err
				}
				if offset, err = strconv.ParseInt(ps[1], 10, 64); err != nil {
					return nil, err
				}
				if buffer, err = util.ReadFileByOffSet(ps[0], offset, length); err != nil {
					return nil, err
				}
				if buffer[0] == '1' {
					bufferReader := bytes.NewBuffer(buffer[1:])
					return bufferReader, nil
				} else {
					msg := "data no sync"
					slog.Error(msg)
					return nil, errors.New(msg)
				}
			}
			return nil, errors.New(fmt.Sprintf("%s not found", fp))
		}
	}

	store.UseIn(composer)
	SetupPreHooks := func(composer *tusd.StoreComposer) {
		composer.UseCore(web.HookDataStore{
			DataStore: composer.Core,
		})
	}
	SetupPreHooks(composer)
	handler, err := tusd.NewHandler(tusd.Config{
		Logger:                  l,
		BasePath:                bigDir,
		StoreComposer:           composer,
		NotifyCompleteUploads:   true,
		RespectForwardedHeaders: true,
	})
	notify := func(handler *tusd.Handler) {
		for {
			select {
			case info := <-handler.CompleteUploads:
				slog.Info("CompleteUploads", info)
				name := ""
				pathCustom := ""
				scene := defaultScene
				if v, ok := info.MetaData["filename"]; ok {
					name = v
				}
				if v, ok := info.MetaData["scene"]; ok {
					scene = v
				}
				if v, ok := info.MetaData["path"]; ok {
					pathCustom = v
				}
				var err error
				md5sum := ""
				oldFullPath := BIG_DIR + "/" + info.ID + ".bin"
				infoFullPath := BIG_DIR + "/" + info.ID + ".info"
				if md5sum, err = util.GetFileSumByName(oldFullPath, fileSumArithmetic); err != nil {
					slog.Error(err)
					continue
				}
				ext := path.Ext(name)
				filename := md5sum + ext
				if name != "" {
					filename = name
				}
				if renameFile {
					filename = md5sum + ext
				}
				timeStamp := time.Now().Unix()
				fpath := time.Now().Format("/20060102/15/04/")
				if pathCustom != "" {
					fpath = "/" + strings.Replace(pathCustom, ".", "", -1) + "/"
				}
				newFullPath := STORE_DIR + "/" + scene + fpath + peerId + "/" + filename
				if pathCustom != "" {
					newFullPath = STORE_DIR + "/" + scene + fpath + filename
				}
				if fi, err := server.GetFileInfoFromLevelDB(md5sum); err != nil {
					slog.Error(err)
				} else {
					tpath := server.GetFilePathByInfo(fi, true)
					if fi.Md5 != "" && util.FileExists(tpath) {
						if _, err := server.SaveFileInfoToLevelDB(info.ID, fi, server.ldb); err != nil {
							slog.Error(err)
						}
						slog.Info(fmt.Sprintf("file is found md5:%s", fi.Md5))
						slog.Info("remove file:", oldFullPath)
						slog.Info("remove file:", infoFullPath)
						os.Remove(oldFullPath)
						os.Remove(infoFullPath)
						continue
					}
				}
				fpath = STORE_DIR_NAME + "/" + defaultScene + fpath + peerId
				os.MkdirAll(DOCKER_DIR+fpath, 0775)
				fileInfo := &en.FileInfo{
					Name:      name,
					Path:      fpath,
					ReName:    filename,
					Size:      info.Size,
					TimeStamp: timeStamp,
					Md5:       md5sum,
					Peers:     []string{server.host},
					OffSet:    -1,
				}
				if err = os.Rename(oldFullPath, newFullPath); err != nil {
					slog.Error(err)
					continue
				}
				slog.Info(fileInfo)
				os.Remove(infoFullPath)
				if _, err = server.SaveFileInfoToLevelDB(info.ID, fileInfo, server.ldb); err != nil {
					//assosiate file id
					slog.Error(err)
				}
				server.SaveFileMd5Log(fileInfo, CONST_FILE_Md5_FILE_NAME)
				go server.postFileToPeer(fileInfo)
				callBack := func(info tusd.FileInfo, fileInfo *en.FileInfo) {
					if callback_url, ok := info.MetaData["callback_url"]; ok {
						req := httplib.Post(callback_url)
						req.SetTimeout(time.Second*10, time.Second*10)
						req.Param("info", util.JsonEncodePretty(fileInfo))
						req.Param("id", info.ID)
						if _, err := req.String(); err != nil {
							slog.Error(err)
						}
					}
				}
				go callBack(info, fileInfo)
			}
		}
	}
	go notify(handler)
	if err != nil {
		slog.Error(err)
	}
	http.Handle(bigDir, http.StripPrefix(bigDir, handler))
}
