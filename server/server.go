package server

import (
	"../conf"
	"../en"
	"../web"
	"bytes"
	"errors"
	"fmt"
	"github.com/astaxie/beego/httplib"
	_ "github.com/eventials/go-tus"
	slog "github.com/sjqzhang/seelog"
	"github.com/sjqzhang/tusd"
	"github.com/sjqzhang/tusd/filestore"
	dbutil "github.com/syndtr/goleveldb/leveldb/util"
	"io"
	"io/ioutil"
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

// 启动服务
// ---------- ---------- ----------
// 01 相关参数配置初始化
// 02 启动相关服务组件
// ---------- ---------- ----------
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
	// -> IP , 获取本机IP地址
	var ip string
	if ip := os.Getenv(GO_FASTDFS_IP); ip == "" {
		ip = util.GetPulicIP()
	}
	// -> HOST
	if host == "" {
		if len(strings.Split(addr, ":")) == 2 {
			host = fmt.Sprintf("http://%s:%s", ip, strings.Split(addr, ":")[1])
			server.host = host
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

	// -> 节点名 peers 手动配置(多个)集群节点
	rex, _ := regexp.Compile("\\d+\\.\\d+\\.\\d+\\.\\d+")
	var prs []string
	for _, peer := range peers {
		if util.Contains(ip, rex.FindAllString(peer, -1)) ||
			util.Contains("127.0.0.1", rex.FindAllString(peer, -1)) {
			continue
		}
		if strings.HasPrefix(peer, "http") {
			prs = append(prs, peer)
		} else {
			prs = append(prs, "http://"+peer)
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

// 启动相关服务组件
// ---------- ---------- ----------
// 00 开始服务 -> 检测文件并加载到处理队列 (定时, 自动检测系统状态 (文件和结点状态))
// 01 开启服务 -> 定期清理及备份数据服务
// 02 开启服务 -> 检测集群状态服务
// 03 开启服务
// 04 开启服务
// 05 开启服务 -> 处理日志队列服务
// 06 开启服务 -> 处理文件下载队列服务
// 07 开启服务 -> 处理文件上传队列服务
// 08 开启服务 -> 清除过期(下载)文件服务
// 09 开启服务 -> 监控文件变更服务
// 10 开启服务
// 11 开启服务 -> 数据迁移服务
// 12 开启服务 -> 数据修复更新服务
// 13 开启服务 -> 定时强制释放内存
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

	// 01 开启服务 -> 定义清理数据与处理文件元数据和日志信息
	go server.cleanAndBackUp()
	// 02 开启服务 -> 检测集群状态服务
	go server.checkClusterStatus()
	// 03 开启服务 -> 检测处理队列文件信息, 并加如处理队列等待处理 (重启)
	go server.loadQueueSendToPeer()
	// 04 开启服务 -> 开启多个 syncWorker, 处理文件上传处理队列
	go server.consumerPostToPeer()
	// 05 开启服务 -> 开启处理, 文件日志处理队列
	go server.consumerLog()
	// 06 开启服务 -> 开启多个 syncWorker, 处理文件下载处理队列
	go server.consumerDownLoad()
	// 07 开启服务 -> 开启多个 uploadWorker, 处理HTTP文件上传处理队列
	go server.consumerUpload()
	// 08 开启服务 -> 清除过期(下载)文件服务
	go server.removeDownloading()

	// 支持按组(集群)管理
	if enableFsnotify {
		// 09 开启服务 -> 监控文件变更并处理
		go server.watchFilesChange()
	}
	// 10 开启服务 -> 加载搜索字典文件
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

// 检测状态文件(stat.json)并格式化
func (server *Service) formatStatInfo() {
	var (
		err  error
		data []byte
		stat map[string]interface{}
	)
	if util.FileExists(CONST_STAT_FILE_NAME) {
		if data, err = util.ReadBinFile(CONST_STAT_FILE_NAME); err != nil {
			_ = slog.Error(err)
		} else {
			if err = json.Unmarshal(data, &stat); err != nil {
				_ = slog.Error(err)
			} else {
				var count int64
				for k, v := range stat {
					switch v.(type) {
					case float64:
						vv := strings.Split(fmt.Sprintf("%f", v), ".")[0]
						if count, err = strconv.ParseInt(vv, 10, 64); err != nil {
							_ = slog.Error(err)
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
	var (
		err       error
		fileInfo  en.FileInfo
		fileCount int64
		fileSize  int64
		stat      en.StatDateFileInfo
	)
	defer func() {
		if re := recover(); re != nil {
			buffer := debug.Stack()
			_ = slog.Error("RepairStatByDate")
			_ = slog.Error(re)
			_ = slog.Error(string(buffer))
		}
	}()
	keyPrefix := "%s_%s_"
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
	server.saveStat()
	stat.Date = date
	stat.FileCount = fileCount
	stat.TotalSize = fileSize
	return stat
}

// 保存状态 -> stat.json
func (server *Service) saveStat() {
	// 定义功能
	SaveStatFunc := func() {
		defer func() {
			if re := recover(); re != nil {
				buffer := debug.Stack()
				_ = slog.Error("SaveStatFunc")
				_ = slog.Error(re)
				_ = slog.Error(string(buffer))
			}
		}()
		stat := server.statMap.Get()
		if v, ok := stat[CONST_STAT_FILE_COUNT_KEY]; ok {
			switch v.(type) {
			case int64, int32, int, float64, float32:
				if v.(int64) >= 0 {
					if data, err := json.Marshal(stat); err != nil {
						_ = slog.Error(err)
					} else {
						util.WriteBinFile(CONST_STAT_FILE_NAME, data)
					}
				}
			}
		}
	}
	SaveStatFunc()
}

// 初始化 Tus
// ---------- ---------- ----------
// 大文件上传, 支持断点续传功能
// 使用本地文件系统, 添加 HTTP Handler (/big/upload/)
// ---------- ---------- ----------
func (server *Service) initTus() {
	var (
		err     error
		fileLog *os.File
	)
	bigDir := CONST_BIG_UPLOAD_PATH_SUFFIX
	if supportGroupManage {
		bigDir = fmt.Sprintf("/%s%s", group, CONST_BIG_UPLOAD_PATH_SUFFIX)
	}
	//
	bigStoreDir := STORE_DIR + "/_big/" + peerId
	_ = os.MkdirAll(bigStoreDir, 0775)
	_ = os.MkdirAll(LOG_DIR, 0775)
	store := filestore.FileStore{
		Path: bigStoreDir,
	}
	//
	if fileLog, err = os.OpenFile(LOG_DIR+"/tusd.log", os.O_CREATE|os.O_RDWR, 0666); err != nil {
		_ = slog.Error(err)
		panic("initTus")
	}
	go func() {
		for {
			if fi, err := fileLog.Stat(); err != nil {
				slog.Error(err)
			} else {
				if fi.Size() > 1024*1024*500 {
					//500M
					_, _ = util.CopyFile(LOG_DIR+"/tusd.log", LOG_DIR+"/tusd.log.2")
					_, _ = fileLog.Seek(0, 0)
					_ = fileLog.Truncate(0)
					_, _ = fileLog.Seek(0, 2)
				}
			}
			time.Sleep(time.Second * 30)
		}
	}()

	//
	l := log.New(fileLog, "[tusd] ", log.LstdFlags)
	//
	composer := tusd.NewStoreComposer()
	// support raw tus upload and download
	store.GetReaderExt = func(id string) (io.Reader, error) {
		var (
			err    error
			offset int64
			length int
			buffer []byte
			fi     *en.FileInfo
			fn     string
		)
		//
		if fi, err = server.getFileInfoFromLevelDB(id); err != nil {
			_ = slog.Error(err)
			return nil, err
		} else {
			if authUrl != "" {
				//
				fileResult := util.JsonEncodePretty(server.buildFileResult(fi, nil))
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
					_ = slog.Error(msg)
					return nil, errors.New(msg)
				}
			}
			return nil, errors.New(fmt.Sprintf("%s not found", fp))
		}
	}
	store.UseIn(composer)

	//
	SetupPreHooksFunc := func(composer *tusd.StoreComposer) {
		composer.UseCore(en.HookDataStore{
			DataStore: composer.Core,
		})
	}
	SetupPreHooksFunc(composer)

	handler, err := tusd.NewHandler(tusd.Config{
		Logger:                  l,
		BasePath:                bigDir,
		StoreComposer:           composer,
		NotifyCompleteUploads:   true,
		RespectForwardedHeaders: true,
	})

	//
	NotifyFunc := func(handler *tusd.Handler) {
		for {
			select {
			// 获取已经上传完成的数据 -> FileInfo
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
				oldFullPath := bigStoreDir + "/" + info.ID + ".bin"
				infoFullPath := bigStoreDir + "/" + info.ID + ".info"
				if md5sum, err = util.GetFileSumByName(oldFullPath, fileSumArithmetic); err != nil {
					_ = slog.Error(err)
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
				fPath := time.Now().Format("/20060102/15/04/")
				if pathCustom != "" {
					fPath = "/" + strings.Replace(pathCustom, ".", "", -1) + "/"
				}
				newFullPath := STORE_DIR + "/" + scene + fPath + peerId + "/" + filename
				if pathCustom != "" {
					newFullPath = STORE_DIR + "/" + scene + fPath + filename
				}
				if fi, err := server.getFileInfoFromLevelDB(md5sum); err != nil {
					_ = slog.Error(err)
				} else {
					tPath := server.analyseFilePathByInfo(fi, true)
					if fi.Md5 != "" && util.FileExists(tPath) {
						if _, err := server.saveFileInfoToLevelDB(info.ID, fi, server.ldb); err != nil {
							_ = slog.Error(err)
						}
						slog.Info(fmt.Sprintf("file is found md5:%s", fi.Md5))
						slog.Info("remove file:", oldFullPath)
						slog.Info("remove file:", infoFullPath)
						_ = os.Remove(oldFullPath)
						_ = os.Remove(infoFullPath)
						continue
					}
				}
				fPath = STORE_DIR_NAME + "/" + defaultScene + fPath + peerId
				_ = os.MkdirAll(DOCKER_DIR+fPath, 0775)
				fileInfo := &en.FileInfo{
					Name:      name,
					Path:      fPath,
					ReName:    filename,
					Size:      info.Size,
					TimeStamp: timeStamp,
					Md5:       md5sum,
					Peers:     []string{server.host},
					OffSet:    -1,
				}
				if err = os.Rename(oldFullPath, newFullPath); err != nil {
					_ = slog.Error(err)
					continue
				}
				slog.Info(fileInfo)
				_ = os.Remove(infoFullPath)
				if _, err = server.saveFileInfoToLevelDB(info.ID, fileInfo, server.ldb); err != nil {
					// assosiate file id
					_ = slog.Error(err)
				}
				// 文件处理信息加入日志处理队列
				server.appendToFileMd5LogQueue(fileInfo, CONST_FILE_Md5_FILE_NAME)
				// 文件保存到集群
				go server.postFileToPeer(fileInfo)

				//
				CallBackFunc := func(info tusd.FileInfo, fileInfo *en.FileInfo) {
					if url, ok := info.MetaData["callback_url"]; ok {
						req := httplib.Post(url)
						req.SetTimeout(time.Second*10, time.Second*10)
						req.Param("info", util.JsonEncodePretty(fileInfo))
						req.Param("id", info.ID)
						if _, err := req.String(); err != nil {
							_ = slog.Error(err)
						}
					}
				}
				go CallBackFunc(info, fileInfo)
			}
		}
	}

	//
	go NotifyFunc(handler)
	if err != nil {
		_ = slog.Error(err)
	}

	// Web Handler Interface -> /big/upload/
	http.Handle(bigDir, http.StripPrefix(bigDir, handler))
}

// 获取文件路径, 文件信息中解析
func (server *Service) analyseFilePathByInfo(fileInfo *en.FileInfo, withDocker bool) string {
	fn := fileInfo.Name
	if fileInfo.ReName != "" {
		fn = fileInfo.ReName
	}
	if withDocker {
		return DOCKER_DIR + fileInfo.Path + "/" + fn
	}
	return fileInfo.Path + "/" + fn
}

// 通过文件信息, 构建文件信息结果数据
func (server *Service) buildFileResult(fileInfo *en.FileInfo, r *http.Request) en.FileResult {
	var (
		fileResult en.FileResult
		domain     string
	)
	host := strings.Replace(host, "http://", "", -1)
	if r != nil {
		host = r.Host
	}
	if !strings.HasPrefix(downloadDomain, "http") {
		if downloadDomain == "" {
			downloadDomain = fmt.Sprintf("http://%s", host)
		} else {
			downloadDomain = fmt.Sprintf("http://%s", downloadDomain)
		}
	}
	if downloadDomain != "" {
		domain = downloadDomain
	} else {
		domain = fmt.Sprintf("http://%s", host)
	}
	//
	outName := fileInfo.Name
	if fileInfo.ReName != "" {
		outName = fileInfo.ReName
	}
	//
	path := strings.Replace(fileInfo.Path, STORE_DIR_NAME+"/", "", 1)
	if supportGroupManage {
		path = group + "/" + path + "/" + outName
	} else {
		path = path + "/" + outName
	}
	downloadUrl := fmt.Sprintf("http://%s/%s", host, path)
	if downloadDomain != "" {
		downloadUrl = fmt.Sprintf("%s/%s", downloadDomain, path)
	}
	fileResult.Url = downloadUrl
	fileResult.Md5 = fileInfo.Md5
	fileResult.Path = "/" + path
	fileResult.Domain = domain
	fileResult.Scene = fileInfo.Scene
	fileResult.Size = fileInfo.Size
	fileResult.ModTime = fileInfo.TimeStamp
	// Just for Compatibility
	fileResult.Src = fileResult.Path
	fileResult.Scenes = fileInfo.Scene
	return fileResult
}

// 配置获取设置重启等操作 [cfg, action]
// ------------------------------------
func (server *Service) reload(w http.ResponseWriter, r *http.Request) {
	var (
		result en.JsonResult
		data   []byte
		cfg    conf.GlobalConfig
	)
	if !web.IsPeer(r) {
		_, _ = w.Write([]byte(web.GetClusterNotPermitMessage(r)))
		return
	}

	result.Status = "fail"
	err := r.ParseForm()
	cfgJson := r.FormValue("cfg")
	action := r.FormValue("action")
	// get 获取配置
	if action == "get" {
		result.Data = conf.Global()
		result.Status = "ok"
		_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
		return
	}
	// set 设置配置, 到配置文件
	if action == "set" {
		if cfgJson == "" {
			result.Message = "(error)parameter cfg(json) require"
			_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
			return
		}
		if err = json.Unmarshal([]byte(cfgJson), &cfg); err != nil {
			_ = slog.Error(err)
			result.Message = err.Error()
			_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
			return
		}
		result.Status = "ok"
		cfgJson = util.JsonEncodePretty(cfg)
		util.WriteFile(CONST_CONF_FILE_NAME, cfgJson)
		_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
		return
	}
	// reload 重新加载配置
	if action == "reload" {
		if data, err = ioutil.ReadFile(CONST_CONF_FILE_NAME); err != nil {
			result.Message = err.Error()
			_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
			return
		}
		if err = json.Unmarshal(data, &cfg); err != nil {
			result.Message = err.Error()
			_, err = w.Write([]byte(util.JsonEncodePretty(result)))
			return
		}
		// 重新解析配置文件
		conf.ParseConfig(CONST_CONF_FILE_NAME)
		//server.initComponent(true)
		result.Status = "ok"
		_, err = w.Write([]byte(util.JsonEncodePretty(result)))
		return
	}
	//
	if action == "" {
		_, err = w.Write([]byte("(error)action support set(json) get reload"))
	}
}
