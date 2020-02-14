package server

import (
	"../en"
	"fmt"
	"github.com/astaxie/beego/httplib"
	mapset "github.com/deckarep/golang-set"
	slog "github.com/sjqzhang/seelog"
	dbutil "github.com/syndtr/goleveldb/leveldb/util"
	"mime/multipart"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"
)

// 处理文件上传队列服务 -> 上传文件
func (server *Service) upload(w http.ResponseWriter, r *http.Request) {
	var (
		err error
		ok  bool
		//		pathname     string
		md5sum       string
		fileName     string
		fileInfo     en.FileInfo
		uploadFile   multipart.File
		uploadHeader *multipart.FileHeader
		scene        string
		output       string
		fileResult   en.FileResult
		data         []byte
		code         string
		secret       interface{}
	)
	output = r.FormValue("output")
	if enableCrossOrigin {
		CrossOrigin(w, r)
		if r.Method == http.MethodOptions {
			return
		}
	}

	if authUrl != "" {
		if !server.checkAuth(w, r) {
			slog.Warn("auth fail", r.Form)
			server.notPermit(w, r)
			w.Write([]byte("auth fail"))
			return
		}
	}
	if r.Method == http.MethodPost {
		md5sum = r.FormValue("md5")
		fileName = r.FormValue("filename")
		output = r.FormValue("output")
		if readOnly {
			w.Write([]byte("(error) readonly"))
			return
		}
		if enableCustomPath {
			fileInfo.Path = r.FormValue("path")
			fileInfo.Path = strings.Trim(fileInfo.Path, "/")
		}
		scene = r.FormValue("scene")
		code = r.FormValue("code")
		if scene == "" {
			//Just for Compatibility
			scene = r.FormValue("scenes")
		}
		if enableGoogleAuth && scene != "" {
			if secret, ok = server.sceneMap.GetValue(scene); ok {
				if !server.verifyGoogleCode(secret.(string), code, int64(downloadTokenExpire/30)) {
					server.notPermit(w, r)
					w.Write([]byte("invalid request,error google code"))
					return
				}
			}
		}
		fileInfo.Md5 = md5sum
		fileInfo.ReName = fileName
		fileInfo.OffSet = -1
		if uploadFile, uploadHeader, err = r.FormFile("file"); err != nil {
			slog.Error(err)
			w.Write([]byte(err.Error()))
			return
		}
		fileInfo.Peers = []string{}
		fileInfo.TimeStamp = time.Now().Unix()
		if scene == "" {
			scene = defaultScene
		}
		if output == "" {
			output = "text"
		}
		if !util.Contains(output, []string{"json", "text"}) {
			w.Write([]byte("output just support json or text"))
			return
		}
		fileInfo.Scene = scene
		if _, err = server.checkScene(scene); err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		if err != nil {
			slog.Error(err)
			http.Redirect(w, r, "/", http.StatusMovedPermanently)
			return
		}
		if _, err = server.processUploadFile(uploadFile, uploadHeader, &fileInfo, r); err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		if enableDistinctFile {
			// 文件去重
			if v, _ := server.getFileInfoFromLevelDB(fileInfo.Md5); v != nil && v.Md5 != "" {
				fileResult = server.buildFileResult(v, r)
				if renameFile {
					os.Remove(DOCKER_DIR + fileInfo.Path + "/" + fileInfo.ReName)
				} else {
					os.Remove(DOCKER_DIR + fileInfo.Path + "/" + fileInfo.Name)
				}
				if output == "json" {
					if data, err = json.Marshal(fileResult); err != nil {
						slog.Error(err)
						w.Write([]byte(err.Error()))
					}
					w.Write(data)
				} else {
					w.Write([]byte(fileResult.Url))
				}
				return
			}
		}
		if fileInfo.Md5 == "" {
			slog.Warn(" fileInfo.Md5 is null")
			return
		}
		if md5sum != "" && fileInfo.Md5 != md5sum {
			slog.Warn(" fileInfo.Md5 and md5sum !=")
			return
		}
		if !enableDistinctFile {
			// bugfix filecount stat
			fileInfo.Md5 = util.MD5(server.GetFilePathByInfo(&fileInfo, false))
		}
		if enableMergeSmallFile && fileInfo.Size < CONST_SMALL_FILE_SIZE {
			// 保存小文件
			if err = server.saveSmallFile(&fileInfo); err != nil {
				slog.Error(err)
				return
			}
		}
		server.AppendToFileMd5LogQueue(&fileInfo, CONST_FILE_Md5_FILE_NAME) //maybe slow
		go server.postFileToPeer(&fileInfo)
		if fileInfo.Size <= 0 {
			slog.Error("file size is zero")
			return
		}
		fileResult = server.buildFileResult(&fileInfo, r)
		if output == "json" {
			if data, err = json.Marshal(fileResult); err != nil {
				slog.Error(err)
				w.Write([]byte(err.Error()))
			}
			w.Write(data)
		} else {
			w.Write([]byte(fileResult.Url))
		}
		return
	} else {
		md5sum = r.FormValue("md5")
		output = r.FormValue("output")
		if md5sum == "" {
			w.Write([]byte("(error) if you want to upload fast md5 is require" +
				",and if you want to upload file,you must use post method  "))
			return
		}
		if v, _ := server.getFileInfoFromLevelDB(md5sum); v != nil && v.Md5 != "" {
			fileResult = server.buildFileResult(v, r)
		}
		if output == "json" {
			if data, err = json.Marshal(fileResult); err != nil {
				slog.Error(err)
				w.Write([]byte(err.Error()))
			}
			w.Write(data)
		} else {
			w.Write([]byte(fileResult.Url))
		}
	}
}

// 检测文件并加载到处理队列 -> 获取MD5文件中保存的文件信息 | 自动修复文件并同步集群数据服务
func (server *Service) getMd5sByDate(date string, filename string) (mapset.Set, error) {
	var (
		keyPrefix string
		md5set    mapset.Set
		keys      []string
	)
	md5set = mapset.NewSet()
	keyPrefix = "%s_%s_"
	keyPrefix = fmt.Sprintf(keyPrefix, date, filename)
	iter := server.logDB.NewIterator(dbutil.BytesPrefix([]byte(keyPrefix)), nil)
	for iter.Next() {
		keys = strings.Split(string(iter.Key()), "_")
		if len(keys) >= 3 {
			md5set.Add(keys[2])
		}
	}
	iter.Release()
	return md5set, nil
}

// 清理 -> 定期清理及备份数据服务
func (server *Service) cleanLogLevelDBByDate(date string, filename string) {
	defer func() {
		if re := recover(); re != nil {
			buffer := debug.Stack()
			slog.Error("CleanLogLevelDBByDate")
			slog.Error(re)
			slog.Error(string(buffer))
		}
	}()
	var (
		err       error
		keyPrefix string
		keys      mapset.Set
	)
	keys = mapset.NewSet()
	keyPrefix = "%s_%s_"
	keyPrefix = fmt.Sprintf(keyPrefix, date, filename)
	iter := server.logDB.NewIterator(dbutil.BytesPrefix([]byte(keyPrefix)), nil)
	for iter.Next() {
		keys.Add(string(iter.Value()))
	}
	iter.Release()
	for key := range keys.Iter() {
		err = server.removeKeyFromLevelDB(key.(string), server.logDB)
		if err != nil {
			slog.Error(err)
		}
	}
}

// 备份 -> 定期清理及备份数据服务
func (server *Service) backUpMetaDataByDate(date string) {
	defer func() {
		if re := recover(); re != nil {
			buffer := debug.Stack()
			slog.Error("BackUpMetaDataByDate")
			slog.Error(re)
			slog.Error(string(buffer))
		}
	}()
	var (
		err          error
		keyPrefix    string
		msg          string
		name         string
		fileInfo     en.FileInfo
		logFileName  string
		fileLog      *os.File
		fileMeta     *os.File
		metaFileName string
		fi           os.FileInfo
	)
	logFileName = DATA_DIR + "/" + date + "/" + CONST_FILE_Md5_FILE_NAME
	server.lockMap.LockKey(logFileName)
	defer server.lockMap.UnLockKey(logFileName)
	metaFileName = DATA_DIR + "/" + date + "/" + "meta.data"
	os.MkdirAll(DATA_DIR+"/"+date, 0775)
	if util.IsExist(logFileName) {
		os.Remove(logFileName)
	}
	if util.IsExist(metaFileName) {
		os.Remove(metaFileName)
	}
	fileLog, err = os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0664)
	if err != nil {
		slog.Error(err)
		return
	}
	defer fileLog.Close()
	fileMeta, err = os.OpenFile(metaFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0664)
	if err != nil {
		slog.Error(err)
		return
	}
	defer fileMeta.Close()
	keyPrefix = "%s_%s_"
	keyPrefix = fmt.Sprintf(keyPrefix, date, CONST_FILE_Md5_FILE_NAME)
	iter := server.logDB.NewIterator(dbutil.BytesPrefix([]byte(keyPrefix)), nil)
	defer iter.Release()
	for iter.Next() {
		if err = json.Unmarshal(iter.Value(), &fileInfo); err != nil {
			continue
		}
		name = fileInfo.Name
		if fileInfo.ReName != "" {
			name = fileInfo.ReName
		}
		msg = fmt.Sprintf("%s\t%s\n", fileInfo.Md5, string(iter.Value()))
		if _, err = fileMeta.WriteString(msg); err != nil {
			slog.Error(err)
		}
		msg = fmt.Sprintf("%s\t%s\n", util.MD5(fileInfo.Path+"/"+name), string(iter.Value()))
		if _, err = fileMeta.WriteString(msg); err != nil {
			slog.Error(err)
		}
		msg = fmt.Sprintf("%s|%d|%d|%s\n", fileInfo.Md5, fileInfo.Size, fileInfo.TimeStamp, fileInfo.Path+"/"+name)
		if _, err = fileLog.WriteString(msg); err != nil {
			slog.Error(err)
		}
	}
	if fi, err = fileLog.Stat(); err != nil {
		slog.Error(err)
	} else if fi.Size() == 0 {
		fileLog.Close()
		os.Remove(logFileName)
	}
	if fi, err = fileMeta.Stat(); err != nil {
		slog.Error(err)
	} else if fi.Size() == 0 {
		fileMeta.Close()
		os.Remove(metaFileName)
	}
}

//
func (server *Service) loadFileInfoByDate(date string, filename string) (mapset.Set, error) {
	defer func() {
		if re := recover(); re != nil {
			buffer := debug.Stack()
			slog.Error("LoadFileInfoByDate")
			slog.Error(re)
			slog.Error(string(buffer))
		}
	}()
	var (
		err       error
		keyPrefix string
		fileInfos mapset.Set
	)
	fileInfos = mapset.NewSet()
	keyPrefix = "%s_%s_"
	keyPrefix = fmt.Sprintf(keyPrefix, date, filename)
	iter := server.logDB.NewIterator(dbutil.BytesPrefix([]byte(keyPrefix)), nil)
	for iter.Next() {
		var fileInfo en.FileInfo
		if err = json.Unmarshal(iter.Value(), &fileInfo); err != nil {
			continue
		}
		fileInfos.Add(&fileInfo)
	}
	iter.Release()
	return fileInfos, nil
}

// 保存操作文件信息日志 -> 处理日志队列服务
func (server *Service) saveFileMd5Log(fileInfo *en.FileInfo, filename string) {
	var (
		err      error
		outname  string
		logDate  string
		ok       bool
		fullpath string
		md5Path  string
		logKey   string
	)
	defer func() {
		if re := recover(); re != nil {
			buffer := debug.Stack()
			slog.Error("saveFileMd5Log")
			slog.Error(re)
			slog.Error(string(buffer))
		}
	}()
	if fileInfo == nil || fileInfo.Md5 == "" || filename == "" {
		slog.Warn("saveFileMd5Log", fileInfo, filename)
		return
	}
	logDate = util.GetDayFromTimeStamp(fileInfo.TimeStamp)
	outname = fileInfo.Name
	if fileInfo.ReName != "" {
		outname = fileInfo.ReName
	}
	fullpath = fileInfo.Path + "/" + outname
	logKey = fmt.Sprintf("%s_%s_%s", logDate, filename, fileInfo.Md5)
	if filename == CONST_FILE_Md5_FILE_NAME {
		//server.searchMap.Put(fileInfo.Md5, fileInfo.Name)
		if ok, err = server.isExistFromLevelDB(fileInfo.Md5, server.ldb); !ok {
			server.statMap.AddCountInt64(logDate+"_"+CONST_STAT_FILE_COUNT_KEY, 1)
			server.statMap.AddCountInt64(logDate+"_"+CONST_STAT_FILE_TOTAL_SIZE_KEY, fileInfo.Size)
			server.saveStat()
		}
		if _, err = server.saveFileInfoToLevelDB(logKey, fileInfo, server.logDB); err != nil {
			slog.Error(err)
		}
		if _, err := server.saveFileInfoToLevelDB(fileInfo.Md5, fileInfo, server.ldb); err != nil {
			slog.Error("saveToLevelDB", err, fileInfo)
		}
		if _, err = server.saveFileInfoToLevelDB(util.MD5(fullpath), fileInfo, server.ldb); err != nil {
			slog.Error("saveToLevelDB", err, fileInfo)
		}
		return
	}
	if filename == CONST_REMOME_Md5_FILE_NAME {
		//server.searchMap.Remove(fileInfo.Md5)
		if ok, err = server.isExistFromLevelDB(fileInfo.Md5, server.ldb); ok {
			server.statMap.AddCountInt64(logDate+"_"+CONST_STAT_FILE_COUNT_KEY, -1)
			server.statMap.AddCountInt64(logDate+"_"+CONST_STAT_FILE_TOTAL_SIZE_KEY, -fileInfo.Size)
			server.saveStat()
		}
		server.removeKeyFromLevelDB(logKey, server.logDB)
		md5Path = util.MD5(fullpath)
		if err := server.removeKeyFromLevelDB(fileInfo.Md5, server.ldb); err != nil {
			slog.Error("RemoveKeyFromLevelDB", err, fileInfo)
		}
		if err = server.removeKeyFromLevelDB(md5Path, server.ldb); err != nil {
			slog.Error("RemoveKeyFromLevelDB", err, fileInfo)
		}
		// remove files.md5 for stat info(repair from logDB)
		logKey = fmt.Sprintf("%s_%s_%s", logDate, CONST_FILE_Md5_FILE_NAME, fileInfo.Md5)
		server.removeKeyFromLevelDB(logKey, server.logDB)
		return
	}
	server.saveFileInfoToLevelDB(logKey, fileInfo, server.logDB)
}

// 从集群中下载文件 -> 处理文件下载队列服务
func (server *Service) downloadFromPeer(peer string, fileInfo *en.FileInfo) {
	var (
		err         error
		filename    string
		fpath       string
		fpathTmp    string
		fi          os.FileInfo
		sum         string
		data        []byte
		downloadUrl string
	)
	if readOnly {
		slog.Warn("ReadOnly", fileInfo)
		return
	}
	if retryCount > 0 && fileInfo.Retry >= retryCount {
		slog.Error("DownloadFromPeer Error ", fileInfo)
		return
	} else {
		fileInfo.Retry = fileInfo.Retry + 1
	}
	filename = fileInfo.Name
	if fileInfo.ReName != "" {
		filename = fileInfo.ReName
	}
	if fileInfo.OffSet != -2 && enableDistinctFile && server.checkFileExistByInfo(fileInfo.Md5, fileInfo) {
		// ignore migrate file
		slog.Info(fmt.Sprintf("DownloadFromPeer file Exist, path:%s", fileInfo.Path+"/"+fileInfo.Name))
		return
	}
	if (!enableDistinctFile || fileInfo.OffSet == -2) && util.FileExists(server.GetFilePathByInfo(fileInfo, true)) {
		// ignore migrate file
		if fi, err = os.Stat(server.GetFilePathByInfo(fileInfo, true)); err == nil {
			if fi.ModTime().Unix() > fileInfo.TimeStamp {
				slog.Info(fmt.Sprintf("ignore file sync path:%s", server.GetFilePathByInfo(fileInfo, false)))
				fileInfo.TimeStamp = fi.ModTime().Unix()
				server.postFileToPeer(fileInfo) // keep newer
				return
			}
			os.Remove(server.GetFilePathByInfo(fileInfo, true))
		}
	}
	if _, err = os.Stat(fileInfo.Path); err != nil {
		os.MkdirAll(DOCKER_DIR+fileInfo.Path, 0775)
	}
	//fmt.Println("downloadFromPeer",fileInfo)
	p := strings.Replace(fileInfo.Path, STORE_DIR_NAME+"/", "", 1)
	//filename= util.UrlEncode(filename)
	downloadUrl = peer + "/" + group + "/" + p + "/" + filename
	slog.Info("DownloadFromPeer: ", downloadUrl)
	fpath = DOCKER_DIR + fileInfo.Path + "/" + filename
	fpathTmp = DOCKER_DIR + fileInfo.Path + "/" + fmt.Sprintf("%s_%s", "tmp_", filename)
	timeout := fileInfo.Size/1024/1024/1 + 30
	if syncTimeout > 0 {
		timeout = syncTimeout
	}
	server.lockMap.LockKey(fpath)
	defer server.lockMap.UnLockKey(fpath)
	download_key := fmt.Sprintf("downloading_%d_%s", time.Now().Unix(), fpath)
	server.ldb.Put([]byte(download_key), []byte(""), nil)
	defer func() {
		server.ldb.Delete([]byte(download_key), nil)
	}()
	if fileInfo.OffSet == -2 {
		//migrate file
		if fi, err = os.Stat(fpath); err == nil && fi.Size() == fileInfo.Size {
			//prevent double download
			server.saveFileInfoToLevelDB(fileInfo.Md5, fileInfo, server.ldb)
			//slog.Info(fmt.Sprintf("file '%s' has download", fpath))
			return
		}
		req := httplib.Get(downloadUrl)
		req.SetTimeout(time.Second*30, time.Second*time.Duration(timeout))
		if err = req.ToFile(fpathTmp); err != nil {
			server.appendToDownloadQueue(fileInfo) //retry
			os.Remove(fpathTmp)
			slog.Error(err, fpathTmp)
			return
		}
		if os.Rename(fpathTmp, fpath) == nil {
			//server.SaveFileMd5Log(fileInfo, CONST_FILE_Md5_FILE_NAME)
			server.saveFileInfoToLevelDB(fileInfo.Md5, fileInfo, server.ldb)
		}
		return
	}
	req := httplib.Get(downloadUrl)
	req.SetTimeout(time.Second*30, time.Second*time.Duration(timeout))
	if fileInfo.OffSet >= 0 {
		//small file download
		data, err = req.Bytes()
		if err != nil {
			server.appendToDownloadQueue(fileInfo) //retry
			slog.Error(err)
			return
		}
		data2 := make([]byte, len(data)+1)
		data2[0] = '1'
		for i, v := range data {
			data2[i+1] = v
		}
		data = data2
		if int64(len(data)) != fileInfo.Size {
			slog.Warn("file size is error")
			return
		}
		fpath = strings.Split(fpath, ",")[0]
		err = util.WriteFileByOffSet(fpath, fileInfo.OffSet, data)
		if err != nil {
			slog.Warn(err)
			return
		}
		server.AppendToFileMd5LogQueue(fileInfo, CONST_FILE_Md5_FILE_NAME)
		return
	}
	if err = req.ToFile(fpathTmp); err != nil {
		server.appendToDownloadQueue(fileInfo) //retry
		os.Remove(fpathTmp)
		slog.Error(err)
		return
	}
	if fi, err = os.Stat(fpathTmp); err != nil {
		os.Remove(fpathTmp)
		return
	}
	_ = sum
	//if Config().EnableDistinctFile {
	//	//DistinctFile
	//	if sum, err = util.GetFileSumByName(fpathTmp, Config().FileSumArithmetic); err != nil {
	//		slog.Error(err)
	//		return
	//	}
	//} else {
	//	//DistinctFile By path
	//	sum = util.MD5(server.GetFilePathByInfo(fileInfo, false))
	//}
	if fi.Size() != fileInfo.Size { //  maybe has bug remove || sum != fileInfo.Md5
		slog.Error("file sum check error")
		os.Remove(fpathTmp)
		return
	}
	if os.Rename(fpathTmp, fpath) == nil {
		server.AppendToFileMd5LogQueue(fileInfo, CONST_FILE_Md5_FILE_NAME)
	}
}
