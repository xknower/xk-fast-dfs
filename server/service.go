package server

import (
	"../en"
	"../web"
	"bufio"
	"fmt"
	"github.com/astaxie/beego/httplib"
	mapset "github.com/deckarep/golang-set"
	"github.com/radovskyb/watcher"
	slog "github.com/sjqzhang/seelog"
	dbutil "github.com/syndtr/goleveldb/leveldb/util"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

// 00 服务 -> 检测文件并加载到处理队列
func (server *Service) checkFileAndSendToPeer(date string, filename string, isForceUpload bool) {
	var (
		md5set mapset.Set
		err    error
		md5s   []interface{}
	)
	defer func() {
		if re := recover(); re != nil {
			buffer := debug.Stack()
			slog.Error("CheckFileAndSendToPeer")
			slog.Error(re)
			slog.Error(string(buffer))
		}
	}()
	if md5set, err = server.GetMd5sByDate(date, filename); err != nil {
		slog.Error(err)
		return
	}
	md5s = md5set.ToSlice()
	for _, md := range md5s {
		if md == nil {
			continue
		}
		if fileInfo, _ := server.getFileInfoFromLevelDB(md.(string)); fileInfo != nil && fileInfo.Md5 != "" {
			if isForceUpload {
				fileInfo.Peers = []string{}
			}
			if len(fileInfo.Peers) > len(peers) {
				continue
			}
			if !util.Contains(server.host, fileInfo.Peers) {
				fileInfo.Peers = append(fileInfo.Peers, server.host) // peer is null
			}
			if filename == CONST_Md5_QUEUE_FILE_NAME {
				server.AppendToDownloadQueue(fileInfo)
			} else {
				server.AppendToQueue(fileInfo)
			}
		}
	}
}

// 01 服务 -> 定期清理及备份数据服务
func (server *Service) cleanAndBackUp() {
	Clean := func() {
		var (
			filenames []string
			yesterday string
		)
		if server.curDate != util.GetToDay() {
			filenames = []string{CONST_Md5_QUEUE_FILE_NAME, CONST_Md5_ERROR_FILE_NAME, CONST_REMOME_Md5_FILE_NAME}
			yesterday = util.GetDayFromTimeStamp(time.Now().AddDate(0, 0, -1).Unix())
			for _, filename := range filenames {
				server.cleanLogLevelDBByDate(yesterday, filename)
			}
			server.BackUpMetaDataByDate(yesterday)
			server.curDate = util.GetToDay()
		}
	}
	go func() {
		for {
			time.Sleep(time.Hour * 6)
			Clean()
		}
	}()
}

// 02 服务 -> 检测集群状态服务
func (server *Service) checkClusterStatus() {
	// 定义功能
	CheckFunc := func() {
		defer func() {
			if re := recover(); re != nil {
				buffer := debug.Stack()
				slog.Error("CheckClusterStatus")
				slog.Error(re)
				slog.Error(string(buffer))
			}
		}()
		var (
			status  web.JsonResult
			err     error
			subject string
			body    string
			req     *httplib.BeegoHTTPRequest
		)
		for _, peer := range peers {
			req = httplib.Get(fmt.Sprintf("%s%s", peer, server.getRequestURI("status")))
			req.SetTimeout(time.Second*5, time.Second*5)
			err = req.ToJSON(&status)
			if err != nil || status.Status != "ok" {
				for _, to := range alarmReceivers {
					subject = "fastdfs server error"
					if err != nil {
						body = fmt.Sprintf("%s\nserver:%s\nerror:\n%s", subject, peer, err.Error())
					} else {
						body = fmt.Sprintf("%s\nserver:%s\n", subject, peer)
					}
					if err = server.sendToMail(to, subject, body, "text"); err != nil {
						slog.Error(err)
					}
				}
				if alarmUrl != "" {
					req = httplib.Post(alarmUrl)
					req.SetTimeout(time.Second*10, time.Second*10)
					req.Param("message", body)
					req.Param("subject", subject)
					if _, err = req.String(); err != nil {
						slog.Error(err)
					}
				}
			}
		}
	}
	go func() {
		for {
			time.Sleep(time.Minute * 10)
			CheckFunc()
		}
	}()
}

// 03 服务
func (server *Service) loadQueueSendToPeer() {
	if queue, err := server.loadFileInfoByDate(util.GetToDay(), CONST_Md5_QUEUE_FILE_NAME); err != nil {
		slog.Error(err)
	} else {
		for fileInfo := range queue.Iter() {
			//server.queueFromPeers <- *fileInfo.(*FileInfo)
			server.AppendToDownloadQueue(fileInfo.(*en.FileInfo))
		}
	}
}

// 04 服务
func (server *Service) consumerPostToPeer() {
	ConsumerFunc := func() {
		for {
			fileInfo := <-server.queueToPeers
			server.postFileToPeer(&fileInfo)
		}
	}
	for i := 0; i < syncWorker; i++ {
		go ConsumerFunc()
	}
}

// 05 服务 -> 处理日志队列服务
func (server *Service) consumerLog() {
	go func() {
		var fileLog *en.FileLog
		for {
			fileLog = <-server.queueFileLog
			server.saveFileMd5Log(fileLog.FileInfo, fileLog.FileName)
		}
	}()
}

// 06 服务 -> 处理文件下载队列服务
func (server *Service) consumerDownLoad() {
	// 定义功能
	ConsumerFunc := func() {
		for {
			fileInfo := <-server.queueFromPeers
			if len(fileInfo.Peers) <= 0 {
				slog.Warn("Peer is null", fileInfo)
				continue
			}
			for _, peer := range fileInfo.Peers {
				if strings.Contains(peer, "127.0.0.1") {
					slog.Warn("sync error with 127.0.0.1", fileInfo)
					continue
				}
				if peer != server.host {
					server.DownloadFromPeer(peer, &fileInfo)
					break
				}
			}
		}
	}
	for i := 0; i < syncWorker; i++ {
		go ConsumerFunc()
	}
}

// 07 服务 -> 处理文件上传队列服务
func (server *Service) consumerUpload() {
	// 定义功能
	ConsumerFunc := func() {
		for {
			wr := <-server.queueUpload
			server.upload(*wr.W, wr.R)
			server.rtMap.AddCountInt64(CONST_UPLOAD_COUNTER_KEY, wr.R.ContentLength)
			if v, ok := server.rtMap.GetValue(CONST_UPLOAD_COUNTER_KEY); ok {
				if v.(int64) > 1*1024*1024*1024 {
					var _v int64
					server.rtMap.Put(CONST_UPLOAD_COUNTER_KEY, _v)
					debug.FreeOSMemory()
				}
			}
			wr.Done <- true
		}
	}
	for i := 0; i < uploadWorker; i++ {
		go ConsumerFunc()
	}
}

// 08 服务 -> 删除下载的文件(超出保存时长)服务
func (server *Service) removeDownloading() {
	// 定义功能
	RemoveDownloadFunc := func() {
		for {
			iter := server.ldb.NewIterator(dbutil.BytesPrefix([]byte("downloading_")), nil)
			for iter.Next() {
				key := iter.Key()
				keys := strings.Split(string(key), "_")
				if len(keys) == 3 {
					if t, err := strconv.ParseInt(keys[1], 10, 64); err == nil && time.Now().Unix()-t > 60*10 {
						os.Remove(DOCKER_DIR + keys[2])
					}
				}
			}
			iter.Release()
			time.Sleep(time.Minute * 3)
		}
	}
	go RemoveDownloadFunc()
}

// 09 服务 -> 监控文件变更服务
func (server *Service) watchFilesChange() {
	var (
		w        *watcher.Watcher
		fileInfo en.FileInfo
		curDir   string
		err      error
		qchan    chan *en.FileInfo
		isLink   bool
	)

	qchan = make(chan *en.FileInfo, 10000)
	w = watcher.New()
	w.FilterOps(watcher.Create)
	//w.FilterOps(watcher.Create, watcher.Remove)
	curDir, err = filepath.Abs(filepath.Dir(STORE_DIR_NAME))
	if err != nil {
		slog.Error(err)
	}
	go func() {
		for {
			select {
			case event := <-w.Event:
				if event.IsDir() {
					continue
				}

				fpath := strings.Replace(event.Path, curDir+string(os.PathSeparator), "", 1)
				if isLink {
					fpath = strings.Replace(event.Path, curDir, STORE_DIR_NAME, 1)
				}
				fpath = strings.Replace(fpath, string(os.PathSeparator), "/", -1)
				sum := util.MD5(fpath)
				fileInfo = en.FileInfo{
					Size:      event.Size(),
					Name:      event.Name(),
					Path:      strings.TrimSuffix(fpath, "/"+event.Name()), // files/default/20190927/xxx
					Md5:       sum,
					TimeStamp: event.ModTime().Unix(),
					Peers:     []string{server.host},
					OffSet:    -2,
					Op:        event.Op.String(),
				}
				slog.Info(fmt.Sprintf("WatchFilesChange op:%s path:%s", event.Op.String(), fpath))
				qchan <- &fileInfo
				//server.AppendToQueue(&fileInfo)
			case err := <-w.Error:
				slog.Error(err)
			case <-w.Closed:
				return
			}
		}
	}()

	go func() {
		for {
			c := <-qchan
			if time.Now().Unix()-c.TimeStamp < 3 {
				qchan <- c
				time.Sleep(time.Second * 1)
				continue
			} else {
				//if c.op == watcher.Remove.String() {
				//	req := httplib.Post(fmt.Sprintf("%s%s?md5=%s", server.host, server.getRequestURI("delete"), c.Md5))
				//	req.Param("md5", c.Md5)
				//	req.SetTimeout(time.Second*5, time.Second*10)
				//	slog.Infof(req.String())
				//}
				if c.Op == watcher.Create.String() {
					slog.Info(fmt.Sprintf("Syncfile Add to Queue path:%s", fileInfo.Path+"/"+fileInfo.Name))
					server.AppendToQueue(c)
					server.saveFileInfoToLevelDB(c.Md5, c, server.ldb)
				}
			}
		}
	}()

	if dir, err := os.Readlink(STORE_DIR_NAME); err == nil {
		if strings.HasSuffix(dir, string(os.PathSeparator)) {
			dir = strings.TrimSuffix(dir, string(os.PathSeparator))
		}
		curDir = dir
		isLink = true
		if err := w.AddRecursive(dir); err != nil {
			slog.Error(err)
		}
		w.Ignore(dir + "/_tmp/")
		w.Ignore(dir + "/" + LARGE_DIR_NAME + "/")
	}

	if err := w.AddRecursive("./" + STORE_DIR_NAME); err != nil {
		slog.Error(err)
	}
	w.Ignore("./" + STORE_DIR_NAME + "/_tmp/")
	w.Ignore("./" + STORE_DIR_NAME + "/" + LARGE_DIR_NAME + "/")
	if err := w.Start(time.Millisecond * 100); err != nil {
		slog.Error(err)
	}
}

// 10 服务 -> 加载搜索字典文件
func (server *Service) loadSearchDict() {
	go func() {
		slog.Info("Load search dict ....")
		f, err := os.Open(CONST_SEARCH_FILE_NAME)
		if err != nil {
			slog.Error(err)
			return
		}
		defer f.Close()

		//
		r := bufio.NewReader(f)
		for {
			line, isPreFix, err := r.ReadLine()
			for isPreFix && err == nil {
				kvs := strings.Split(string(line), "\t")
				if len(kvs) == 2 {
					server.searchMap.Put(kvs[0], kvs[1])
				}
			}
		}
		slog.Info("finish load search dict")
	}()
}

// 11 服务 -> 数据迁移服务
func (server *Service) repairFileInfoFromFile() {
	var (
		pathPrefix string
		err        error
		fi         os.FileInfo
	)
	defer func() {
		if re := recover(); re != nil {
			buffer := debug.Stack()
			slog.Error("RepairFileInfoFromFile")
			slog.Error(re)
			slog.Error(string(buffer))
		}
	}()

	// 获取锁
	if server.lockMap.IsLock("RepairFileInfoFromFile") {
		slog.Warn("Lock RepairFileInfoFromFile")
		return
	}
	server.lockMap.LockKey("RepairFileInfoFromFile")
	defer server.lockMap.UnLockKey("RepairFileInfoFromFile")

	// 定义功能
	HandleFunc := func(filePath string, f os.FileInfo, err error) error {
		var (
			files    []os.FileInfo
			fi       os.FileInfo
			fileInfo en.FileInfo
			sum      string
			pathMd5  string
		)
		if f.IsDir() {
			files, err = ioutil.ReadDir(filePath)

			if err != nil {
				return err
			}
			for _, fi = range files {
				if fi.IsDir() || fi.Size() == 0 {
					continue
				}
				filePath = strings.Replace(filePath, "\\", "/", -1)
				if DOCKER_DIR != "" {
					filePath = strings.Replace(filePath, DOCKER_DIR, "", 1)
				}
				if pathPrefix != "" {
					filePath = strings.Replace(filePath, pathPrefix, STORE_DIR_NAME, 1)
				}
				if strings.HasPrefix(filePath, STORE_DIR_NAME+"/"+LARGE_DIR_NAME) {
					slog.Info(fmt.Sprintf("ignore small file file %s", filePath+"/"+fi.Name()))
					continue
				}
				pathMd5 = util.MD5(filePath + "/" + fi.Name())
				//if finfo, _ := server.GetFileInfoFromLevelDB(pathMd5); finfo != nil && finfo.Md5 != "" {
				//	slog.Info(fmt.Sprintf("exist ignore file %s", file_path+"/"+fi.Name()))
				//	continue
				//}
				//sum, err = util.GetFileSumByName(file_path+"/"+fi.Name(), Config().FileSumArithmetic)
				sum = pathMd5
				if err != nil {
					slog.Error(err)
					continue
				}
				fileInfo = en.FileInfo{
					Size:      fi.Size(),
					Name:      fi.Name(),
					Path:      filePath,
					Md5:       sum,
					TimeStamp: fi.ModTime().Unix(),
					Peers:     []string{server.host},
					OffSet:    -2,
				}
				//slog.Info(fileInfo)
				slog.Info(filePath, "/", fi.Name())
				server.AppendToQueue(&fileInfo)
				//server.postFileToPeer(&fileInfo)
				server.saveFileInfoToLevelDB(fileInfo.Md5, &fileInfo, server.ldb)
				//server.SaveFileMd5Log(&fileInfo, CONST_FILE_Md5_FILE_NAME)
			}
		}
		return nil
	}

	//
	pathname := STORE_DIR
	pathPrefix, err = os.Readlink(pathname)
	if err == nil {
		//link
		pathname = pathPrefix
		if strings.HasSuffix(pathPrefix, "/") {
			//bugfix fullpath
			pathPrefix = pathPrefix[0 : len(pathPrefix)-1]
		}
	}
	fi, err = os.Stat(pathname)
	if err != nil {
		slog.Error(err)
	}
	if fi.IsDir() {
		filepath.Walk(pathname, HandleFunc)
	}
	slog.Info("RepairFileInfoFromFile is finish.")
}

// 12 服务 -> 自动修复文件并同步集群数据服务
func (server *Service) autoRepair(forceRepair bool) {
	// 获取锁
	if server.lockMap.IsLock("AutoRepair") {
		slog.Warn("Lock AutoRepair")
		return
	}
	server.lockMap.LockKey("AutoRepair")
	defer server.lockMap.UnLockKey("AutoRepair")

	// 定义自动修复功能
	AutoRepairFunc := func(forceRepair bool) {
		var (
			dateStats []en.StatDateFileInfo
			err       error
			countKey  string
			md5s      string
			localSet  mapset.Set
			remoteSet mapset.Set
			allSet    mapset.Set
			tmpSet    mapset.Set
			fileInfo  *en.FileInfo
		)
		defer func() {
			if re := recover(); re != nil {
				buffer := debug.Stack()
				slog.Error("AutoRepair")
				slog.Error(re)
				slog.Error(string(buffer))
			}
		}()

		// 定义更新数据功能
		Update := func(peer string, dateStat en.StatDateFileInfo) {
			// 从远端拉数据过来
			req := httplib.Get(fmt.Sprintf("%s%s?date=%s&force=%s", peer, server.getRequestURI("sync"), dateStat.Date, "1"))
			req.SetTimeout(time.Second*5, time.Second*5)
			if _, err = req.String(); err != nil {
				slog.Error(err)
			}
			slog.Info(fmt.Sprintf("syn file from %s date %s", peer, dateStat.Date))
		}
		for _, peer := range peers {
			req := httplib.Post(fmt.Sprintf("%s%s", peer, server.getRequestURI("stat")))
			req.Param("inner", "1")
			req.SetTimeout(time.Second*5, time.Second*15)
			if err = req.ToJSON(&dateStats); err != nil {
				slog.Error(err)
				continue
			}
			for _, dateStat := range dateStats {
				if dateStat.Date == "all" {
					continue
				}
				countKey = dateStat.Date + "_" + CONST_STAT_FILE_COUNT_KEY
				if v, ok := server.statMap.GetValue(countKey); ok {
					switch v.(type) {
					case int64:
						if v.(int64) != dateStat.FileCount || forceRepair {
							// 不相等,找差异
							// TODO
							req := httplib.Post(fmt.Sprintf("%s%s", peer, server.getRequestURI("get_md5s_by_date")))
							req.SetTimeout(time.Second*15, time.Second*60)
							req.Param("date", dateStat.Date)
							if md5s, err = req.String(); err != nil {
								continue
							}
							if localSet, err = server.GetMd5sByDate(dateStat.Date, CONST_FILE_Md5_FILE_NAME); err != nil {
								slog.Error(err)
								continue
							}
							remoteSet = util.StrToMapSet(md5s, ",")
							allSet = localSet.Union(remoteSet)
							md5s = util.MapSetToStr(allSet.Difference(localSet), ",")
							req = httplib.Post(fmt.Sprintf("%s%s", peer, server.getRequestURI("receive_md5s")))
							req.SetTimeout(time.Second*15, time.Second*60)
							req.Param("md5s", md5s)
							req.String()
							tmpSet = allSet.Difference(remoteSet)
							for v := range tmpSet.Iter() {
								if v != nil {
									if fileInfo, err = server.getFileInfoFromLevelDB(v.(string)); err != nil {
										slog.Error(err)
										continue
									}
									server.AppendToQueue(fileInfo)
								}
							}
							//Update(peer,dateStat)
						}
					}
				} else {
					Update(peer, dateStat)
				}
			}
		}
	}
	AutoRepairFunc(forceRepair)
}
