package server

import (
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
	"github.com/syndtr/goleveldb/leveldb"
	dbutil "github.com/syndtr/goleveldb/leveldb/util"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

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
				slog.Error("SaveStatFunc")
				slog.Error(re)
				slog.Error(string(buffer))
			}
		}()
		stat := server.statMap.Get()
		if v, ok := stat[CONST_STAT_FILE_COUNT_KEY]; ok {
			switch v.(type) {
			case int64, int32, int, float64, float32:
				if v.(int64) >= 0 {
					if data, err := json.Marshal(stat); err != nil {
						slog.Error(err)
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
		//
		if fi, err = server.getFileInfoFromLevelDB(id); err != nil {
			slog.Error(err)
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
				if fi, err := server.getFileInfoFromLevelDB(md5sum); err != nil {
					slog.Error(err)
				} else {
					tpath := server.GetFilePathByInfo(fi, true)
					if fi.Md5 != "" && util.FileExists(tpath) {
						if _, err := server.saveFileInfoToLevelDB(info.ID, fi, server.ldb); err != nil {
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
				if _, err = server.saveFileInfoToLevelDB(info.ID, fileInfo, server.ldb); err != nil {
					//assosiate file id
					slog.Error(err)
				}
				server.SaveFileMd5Log(fileInfo, CONST_FILE_Md5_FILE_NAME)
				//
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

// 获取文件信息 -> 检测文件并加载到处理队列 | 自动修复文件并同步集群数据服务
func (server *Service) getFileInfoFromLevelDB(key string) (*en.FileInfo, error) {
	var (
		err      error
		data     []byte
		fileInfo en.FileInfo
	)
	if data, err = server.ldb.Get([]byte(key), nil); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(data, &fileInfo); err != nil {
		return nil, err
	}
	return &fileInfo, nil
}

//
func (server *Service) saveFileInfoToLevelDB(key string, fileInfo *en.FileInfo, db *leveldb.DB) (*en.FileInfo, error) {
	var (
		err  error
		data []byte
	)
	if fileInfo == nil || db == nil {
		return nil, errors.New("fileInfo is null or db is null")
	}
	if data, err = json.Marshal(fileInfo); err != nil {
		return fileInfo, err
	}
	if err = db.Put([]byte(key), data, nil); err != nil {
		return fileInfo, err
	}
	if db == server.ldb { //search slow ,write fast, double write logDB
		logDate := util.GetDayFromTimeStamp(fileInfo.TimeStamp)
		logKey := fmt.Sprintf("%s_%s_%s", logDate, CONST_FILE_Md5_FILE_NAME, fileInfo.Md5)
		server.logDB.Put([]byte(logKey), data, nil)
	}
	return fileInfo, nil
}

//
func (server *Service) buildFileResult(fileInfo *en.FileInfo, r *http.Request) en.FileResult {
	var (
		outname     string
		fileResult  en.FileResult
		p           string
		downloadUrl string
		domain      string
		host        string
	)
	host = strings.Replace(host, "http://", "", -1)
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
	outname = fileInfo.Name
	if fileInfo.ReName != "" {
		outname = fileInfo.ReName
	}
	p = strings.Replace(fileInfo.Path, STORE_DIR_NAME+"/", "", 1)
	if supportGroupManage {
		p = group + "/" + p + "/" + outname
	} else {
		p = p + "/" + outname
	}
	downloadUrl = fmt.Sprintf("http://%s/%s", host, p)
	if downloadDomain != "" {
		downloadUrl = fmt.Sprintf("%s/%s", downloadDomain, p)
	}
	fileResult.Url = downloadUrl
	fileResult.Md5 = fileInfo.Md5
	fileResult.Path = "/" + p
	fileResult.Domain = domain
	fileResult.Scene = fileInfo.Scene
	fileResult.Size = fileInfo.Size
	fileResult.ModTime = fileInfo.TimeStamp
	// Just for Compatibility
	fileResult.Src = fileResult.Path
	fileResult.Scenes = fileInfo.Scene
	return fileResult
}
