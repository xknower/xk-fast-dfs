// 服务端, 组件服务实现
package server

import (
	"../en"
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

// 00 服务 -> 检测文件并加载到处理队列 [date, filename]
func (server *Service) checkFileAndSendToPeer(date string, filename string, isForceUpload bool) {
	defer func() {
		if re := recover(); re != nil {
			buffer := debug.Stack()
			_ = slog.Error("CheckFileAndSendToPeer")
			_ = slog.Error(re)
			_ = slog.Error(string(buffer))
		}
	}()
	var (
		err    error
		md5set mapset.Set
	)
	//
	if md5set, err = server.getMd5sByDate(date, filename); err != nil {
		slog.Error(err)
		return
	}
	//
	md5s := md5set.ToSlice()
	for _, md := range md5s {
		if md == nil {
			continue
		}
		//
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
			// 添加到处理队列, 等待处理
			if filename == CONST_Md5_QUEUE_FILE_NAME {
				server.appendToDownloadQueue(fileInfo)
			} else {
				server.appendToQueue(fileInfo)
			}
		}
	}
}

// 01 服务 -> 定期清理及备份数据服务
func (server *Service) cleanAndBackUp() {
	CleanFunc := func() {
		var filenames []string
		if server.curDate != util.GetToDay() {
			filenames = []string{CONST_Md5_QUEUE_FILE_NAME, CONST_Md5_ERROR_FILE_NAME, CONST_REMOME_Md5_FILE_NAME}
			yesterday := util.GetDayFromTimeStamp(time.Now().AddDate(0, 0, -1).Unix())
			for _, filename := range filenames {
				// 清理日志数据
				server.cleanLogLevelDBByDate(yesterday, filename)
			}
			// 备份及清理数据
			server.backUpMetaDataByDate(yesterday)
			server.curDate = util.GetToDay() // 最后一次处理时间
		}
	}
	//
	go func() {
		for {
			//
			time.Sleep(time.Hour * 6)
			CleanFunc()
		}
	}()
}

// 02 服务 -> 检测集群状态服务, 并告警报警节点状态信息
func (server *Service) checkClusterStatus() {
	// 定义功能
	CheckFunc := func() {
		defer func() {
			if re := recover(); re != nil {
				buffer := debug.Stack()
				_ = slog.Error("CheckClusterStatus")
				_ = slog.Error(re)
				_ = slog.Error(string(buffer))
			}
		}()
		var (
			status  en.JsonResult
			subject string
			body    string
		)
		for _, peer := range peers {
			// 遍历查询集群所有节点, 获取所有节点状态信息
			req := httplib.Get(fmt.Sprintf("%s%s", peer, server.analyseRequestURI("status")))
			req.SetTimeout(time.Second*5, time.Second*5)
			if err := req.ToJSON(&status); err != nil || status.Status != "ok" {
				// 未查询到节点状态或异常, 遍历警告地址, 发送警告信息
				for _, to := range alarmReceivers {
					subject = "fastdfs server error"
					if err != nil {
						body = fmt.Sprintf("%s\nserver:%s\nerror:\n%s", subject, peer, err.Error())
					} else {
						body = fmt.Sprintf("%s\nserver:%s\n", subject, peer)
					}
					// 发送邮件
					if err = server.sendToMail(to, subject, body, "text"); err != nil {
						_ = slog.Error(err)
					}
				}
				//
				if alarmUrl != "" {
					req = httplib.Post(alarmUrl)
					req.SetTimeout(time.Second*10, time.Second*10)
					req.Param("message", body)
					req.Param("subject", subject)
					if _, err = req.String(); err != nil {
						_ = slog.Error(err)
					}
				}
			}
		}
	}
	//
	go func() {
		for {
			time.Sleep(time.Minute * 10)
			CheckFunc()
		}
	}()
}

// 03 服务 -> 检测处理队列文件信息, 并加如处理队列等待处理 (重启)
func (server *Service) loadQueueSendToPeer() {
	// 加载处理队列文件
	if queue, err := server.loadFileInfoByDate(util.GetToDay(), CONST_Md5_QUEUE_FILE_NAME); err != nil {
		_ = slog.Error(err)
	} else {
		for fileInfo := range queue.Iter() {
			//server.queueFromPeers <- *fileInfo.(*FileInfo)
			server.appendToDownloadQueue(fileInfo.(*en.FileInfo))
		}
	}
}

// 04 服务 -> 开启多个 syncWorker, 处理文件上传处理队列
func (server *Service) consumerPostToPeer() {
	ConsumerFunc := func() {
		for {
			fileInfo := <-server.queueToPeers
			server.postFileToPeer(&fileInfo)
		}
	}
	//
	for i := 0; i < syncWorker; i++ {
		go ConsumerFunc()
	}
}

// 05 服务 -> 开启处理, 文件日志处理队列
func (server *Service) consumerLog() {
	go func() {
		var fileLog *en.FileLog
		for {
			fileLog = <-server.queueFileLog
			server.saveFileMd5Log(fileLog.FileInfo, fileLog.FileName)
		}
	}()
}

// 06 服务 -> 开启多个 syncWorker, 处理文件下载处理队列
func (server *Service) consumerDownLoad() {
	// 定义功能
	ConsumerFunc := func() {
		for {
			fileInfo := <-server.queueFromPeers
			if len(fileInfo.Peers) <= 0 {
				_ = slog.Warn("Peer is null", fileInfo)
				continue
			}
			//
			for _, peer := range fileInfo.Peers {
				if strings.Contains(peer, "127.0.0.1") {
					_ = slog.Warn("sync error with 127.0.0.1", fileInfo)
					continue
				}
				if peer != server.host {
					// 下载文件
					server.downloadFromPeer(peer, &fileInfo)
					break
				}
			}
		}
	}
	//
	for i := 0; i < syncWorker; i++ {
		go ConsumerFunc()
	}
}

// 07 服务 -> 开启多个 uploadWorker, 处理HTTP文件上传处理队列
func (server *Service) consumerUpload() {
	ConsumerFunc := func() {
		for {
			wr := <-server.queueUpload
			//
			server.upload(*wr.W, wr.R)
			//
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
	//
	for i := 0; i < uploadWorker; i++ {
		go ConsumerFunc()
	}
}

// 08 服务 -> 删除下载的文件服务
func (server *Service) removeDownloading() {
	// 定义功能
	RemoveDownloadFunc := func() {
		for {
			// 文件信息数据库
			iter := server.ldb.NewIterator(dbutil.BytesPrefix([]byte("downloading_")), nil)
			for iter.Next() {
				key := iter.Key()
				keys := strings.Split(string(key), "_")
				if len(keys) == 3 {
					if t, err := strconv.ParseInt(keys[1], 10, 64); err == nil && time.Now().Unix()-t > 60*10 {
						// 移除
						_ = os.Remove(DOCKER_DIR + keys[2])
					}
				}
			}
			iter.Release()
			time.Sleep(time.Minute * 3)
		}
	}
	go RemoveDownloadFunc()
}

// 09 服务 -> 监控文件变更并处理服务
func (server *Service) watchFilesChange() {
	var (
		err      error
		w        *watcher.Watcher
		fileInfo en.FileInfo
		curDir   string
		isLink   bool
	)
	qChan := make(chan *en.FileInfo, 10000)
	w = watcher.New()
	w.FilterOps(watcher.Create)
	//w.FilterOps(watcher.Create, watcher.Remove)

	//
	curDir, err = filepath.Abs(filepath.Dir(STORE_DIR_NAME))
	if err != nil {
		_ = slog.Error(err)
	}

	//
	go func() {
		for {
			select {
			case event := <-w.Event:
				if event.IsDir() {
					continue
				}
				fPath := strings.Replace(event.Path, curDir+string(os.PathSeparator), "", 1)
				if isLink {
					fPath = strings.Replace(event.Path, curDir, STORE_DIR_NAME, 1)
				}
				fPath = strings.Replace(fPath, string(os.PathSeparator), "/", -1)
				sum := util.MD5(fPath)
				fileInfo = en.FileInfo{
					Size:      event.Size(),
					Name:      event.Name(),
					Path:      strings.TrimSuffix(fPath, "/"+event.Name()), // files/default/20190927/xxx
					Md5:       sum,
					TimeStamp: event.ModTime().Unix(),
					Peers:     []string{server.host},
					OffSet:    -2,
					Op:        event.Op.String(),
				}
				slog.Info(fmt.Sprintf("WatchFilesChange op:%s path:%s", event.Op.String(), fPath))
				qChan <- &fileInfo
				//server.AppendToQueue(&fileInfo)
			case err := <-w.Error:
				_ = slog.Error(err)
			case <-w.Closed:
				return
			}
		}
	}()

	go func() {
		for {
			c := <-qChan
			if time.Now().Unix()-c.TimeStamp < 3 {
				qChan <- c
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
					// 加入文件上传处理队列
					server.appendToQueue(c)
					// 保存文件信息到数据库
					_, _ = server.saveFileInfoToLevelDB(c.Md5, c, server.ldb)
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
			_ = slog.Error(err)
		}
		_ = w.Ignore(dir + "/_tmp/")
		_ = w.Ignore(dir + "/" + LARGE_DIR_NAME + "/")
	}

	if err := w.AddRecursive("./" + STORE_DIR_NAME); err != nil {
		_ = slog.Error(err)
	}

	_ = w.Ignore("./" + STORE_DIR_NAME + "/_tmp/")
	_ = w.Ignore("./" + STORE_DIR_NAME + "/" + LARGE_DIR_NAME + "/")
	if err := w.Start(time.Millisecond * 100); err != nil {
		_ = slog.Error(err)
	}
}

// 10 服务 -> 加载搜索字典文件
func (server *Service) loadSearchDict() {
	go func() {
		slog.Info("Load search dict ....")
		f, err := os.Open(CONST_SEARCH_FILE_NAME)
		if err != nil {
			_ = slog.Error(err)
			return
		}
		defer f.Close()

		//
		r := bufio.NewReader(f)
		PutFunc := func() {
			for {
				line, isPreFix, err := r.ReadLine()
				for isPreFix && err == nil {
					kvs := strings.Split(string(line), "\t")
					if len(kvs) == 2 {
						// 加入
						server.searchMap.Put(kvs[0], kvs[1])
					}
				}
			}
		}
		go PutFunc()

		slog.Info("finish load search dict")
	}()
}

// 11 服务 -> 数据迁移服务
func (server *Service) repairFileInfoFromFile() {
	defer func() {
		if re := recover(); re != nil {
			buffer := debug.Stack()
			_ = slog.Error("RepairFileInfoFromFile")
			_ = slog.Error(re)
			_ = slog.Error(string(buffer))
		}
	}()
	var (
		err        error
		pathPrefix string
		fi         os.FileInfo
	)

	// 获取锁
	if server.lockMap.IsLock("RepairFileInfoFromFile") {
		_ = slog.Warn("Lock RepairFileInfoFromFile")
		return
	}
	server.lockMap.LockKey("RepairFileInfoFromFile")
	defer server.lockMap.UnLockKey("RepairFileInfoFromFile")

	// 定义功能
	HandleFunc := func(filePath string, f os.FileInfo, err error) error {
		var (
			files []os.FileInfo
			fi    os.FileInfo
			sum   string
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
				//
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

				// 计算文件Hash值, 根据文件路径
				pathMd5 := util.MD5(filePath + "/" + fi.Name())
				//if finfo, _ := server.GetFileInfoFromLevelDB(pathMd5); finfo != nil && finfo.Md5 != "" {
				//	slog.Info(fmt.Sprintf("exist ignore file %s", file_path+"/"+fi.Name()))
				//	continue
				//}
				//sum, err = util.GetFileSumByName(file_path+"/"+fi.Name(), Config().FileSumArithmetic)
				sum = pathMd5
				if err != nil {
					_ = slog.Error(err)
					continue
				}
				fileInfo := en.FileInfo{
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
				// 加载到处理队列
				server.appendToQueue(&fileInfo)
				//server.postFileToPeer(&fileInfo)
				// 保存信息到数据库
				_, _ = server.saveFileInfoToLevelDB(fileInfo.Md5, &fileInfo, server.ldb)
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
		_ = slog.Error(err)
	}
	if fi.IsDir() {
		_ = filepath.Walk(pathname, HandleFunc)
	}
	slog.Info("RepairFileInfoFromFile is finish.")
}

// 12 服务 -> 自动修复文件并同步集群数据服务
func (server *Service) autoRepair(forceRepair bool) {
	// 获取锁
	if server.lockMap.IsLock("AutoRepair") {
		_ = slog.Warn("Lock AutoRepair")
		return
	}
	server.lockMap.LockKey("AutoRepair")
	defer server.lockMap.UnLockKey("AutoRepair")

	// 定义自动修复功能
	AutoRepairFunc := func(forceRepair bool) {
		defer func() {
			if re := recover(); re != nil {
				buffer := debug.Stack()
				_ = slog.Error("AutoRepair")
				_ = slog.Error(re)
				_ = slog.Error(string(buffer))
			}
		}()
		var (
			err       error
			dateStats []en.StatDateFileInfo
			md5s      string
			localSet  mapset.Set
			fileInfo  *en.FileInfo
		)

		// 定义更新数据功能 (从其他节点获取数据)
		Update := func(peer string, dateStat en.StatDateFileInfo) {
			// 从远端拉数据过来
			req := httplib.Get(fmt.Sprintf("%s%s?date=%s&force=%s", peer, server.analyseRequestURI("sync"), dateStat.Date, "1"))
			req.SetTimeout(time.Second*5, time.Second*5)
			if _, err = req.String(); err != nil {
				_ = slog.Error(err)
			}
			slog.Info(fmt.Sprintf("syn file from %s date %s", peer, dateStat.Date))
		}

		// 遍历集权所有节点状态,
		for _, peer := range peers {
			req := httplib.Post(fmt.Sprintf("%s%s", peer, server.analyseRequestURI("stat")))
			req.Param("inner", "1")
			req.SetTimeout(time.Second*5, time.Second*15)
			if err = req.ToJSON(&dateStats); err != nil {
				_ = slog.Error(err)
				continue
			}
			//
			for _, dateStat := range dateStats {
				if dateStat.Date == "all" {
					continue
				}
				countKey := dateStat.Date + "_" + CONST_STAT_FILE_COUNT_KEY
				if v, ok := server.statMap.GetValue(countKey); ok {
					switch v.(type) {
					case int64:
						if v.(int64) != dateStat.FileCount || forceRepair {
							// 不相等,找差异 TODO
							req := httplib.Post(fmt.Sprintf("%s%s", peer, server.analyseRequestURI("get_md5s_by_date")))
							req.SetTimeout(time.Second*15, time.Second*60)
							req.Param("date", dateStat.Date)
							if md5s, err = req.String(); err != nil {
								continue
							}
							if localSet, err = server.getMd5sByDate(dateStat.Date, CONST_FILE_Md5_FILE_NAME); err != nil {
								_ = slog.Error(err)
								continue
							}
							remoteSet := util.StrToMapSet(md5s, ",")
							allSet := localSet.Union(remoteSet)
							md5s = util.MapSetToStr(allSet.Difference(localSet), ",")
							req = httplib.Post(fmt.Sprintf("%s%s", peer, server.analyseRequestURI("receive_md5s")))
							req.SetTimeout(time.Second*15, time.Second*60)
							req.Param("md5s", md5s)
							_, _ = req.String()
							tmpSet := allSet.Difference(remoteSet)
							for v := range tmpSet.Iter() {
								if v != nil {
									if fileInfo, err = server.getFileInfoFromLevelDB(v.(string)); err != nil {
										_ = slog.Error(err)
										continue
									}
									server.appendToQueue(fileInfo)
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
