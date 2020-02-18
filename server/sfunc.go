package server

import (
	"../en"
	"../web"
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

// HTTP文件上传处理 (HTTP文件上传处理队列)
// [filename 文件名, path 上传路径 file 文件内容 | md5 文件唯一标识符, output 输出格式 (json , 默认text下载URL)]
func (server *Service) upload(w http.ResponseWriter, r *http.Request) {
	var (
		err          error
		ok           bool
		fileInfo     en.FileInfo
		uploadFile   multipart.File
		uploadHeader *multipart.FileHeader
		fileResult   en.FileUploadResult
		data         []byte
		secret       interface{}
	)

	if enableCrossOrigin {
		web.CrossOrigin(w, r)
		if r.Method == http.MethodOptions {
			return
		}
	}
	if authUrl != "" {
		if !server.checkAuth(w, r) {
			_ = slog.Warn("auth fail", r.Form)
			server.notPermit(w, r)
			_, _ = w.Write([]byte("auth fail"))
			return
		}
	}
	if r.Method == http.MethodPost {
		// POST -> 上传文件 filename path scene code file | md5 output
		fileName := r.FormValue("filename")
		md5sum := r.FormValue("md5")
		//
		fileInfo.Md5 = md5sum
		fileInfo.ReName = fileName
		fileInfo.OffSet = -1
		if readOnly {
			// 该节点只读, 不支持上传文件
			_, _ = w.Write([]byte("(error) readonly"))
			return
		}
		if enableCustomPath {
			// 支持非日期路径
			fileInfo.Path = r.FormValue("path")
			fileInfo.Path = strings.Trim(fileInfo.Path, "/")
		}
		//
		scene := r.FormValue("scene")
		if scene == "" {
			//Just for Compatibility
			scene = r.FormValue("scenes")
			if scene == "" {
				scene = defaultScene
			}
		}
		if _, err = server.checkScene(scene); err != nil {
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		fileInfo.Scene = scene
		if enableGoogleAuth && scene != "" {
			code := r.FormValue("code")
			if secret, ok = server.sceneMap.GetValue(scene); ok {
				if !server.verifyGoogleCode(secret.(string), code, int64(downloadTokenExpire/30)) {
					server.notPermit(w, r)
					_, _ = w.Write([]byte("invalid request,error google code"))
					return
				}
			}
		}
		if uploadFile, uploadHeader, err = r.FormFile("file"); err != nil {
			_ = slog.Error(err)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		fileInfo.Peers = []string{}
		fileInfo.TimeStamp = time.Now().Unix()
		if err != nil {
			_ = slog.Error(err)
			http.Redirect(w, r, "/", http.StatusMovedPermanently)
			return
		}
		// 输出格式
		output := r.FormValue("output")
		if output == "" {
			output = "text"
		}
		if !util.Contains(output, []string{"json", "text"}) {
			_, _ = w.Write([]byte("output just support json or text"))
			return
		}
		//
		if _, err = server.processUploadFile(uploadFile, uploadHeader, &fileInfo, r); err != nil {
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		if enableDistinctFile {
			// 文件去重
			if v, _ := server.getFileInfoFromLevelDB(fileInfo.Md5); v != nil && v.Md5 != "" {
				fileResult = server.buildFileResult(v, r)
				if renameFile {
					_ = os.Remove(DOCKER_DIR + fileInfo.Path + "/" + fileInfo.ReName)
				} else {
					_ = os.Remove(DOCKER_DIR + fileInfo.Path + "/" + fileInfo.Name)
				}
				if output == "json" {
					if data, err = json.Marshal(fileResult); err != nil {
						_ = slog.Error(err)
						_, _ = w.Write([]byte(err.Error()))
					}
					_, _ = w.Write(data)
				} else {
					_, _ = w.Write([]byte(fileResult.Url))
				}
				return
			}
		}
		if fileInfo.Md5 == "" {
			_ = slog.Warn(" fileInfo.Md5 is null")
			return
		}
		if md5sum != "" && fileInfo.Md5 != md5sum {
			_ = slog.Warn(" fileInfo.Md5 and md5sum !=")
			return
		}
		if !enableDistinctFile {
			// bugfix filecount stat
			fileInfo.Md5 = util.MD5(server.analyseFilePathByInfo(&fileInfo, false))
		}
		if enableMergeSmallFile && fileInfo.Size < CONST_SMALL_FILE_SIZE {
			// 保存小文件
			if err = server.saveSmallFile(&fileInfo); err != nil {
				_ = slog.Error(err)
				return
			}
		}
		//
		server.appendToFileMd5LogQueue(&fileInfo, CONST_FILE_Md5_FILE_NAME) //maybe slow
		go server.postFileToPeer(&fileInfo)
		if fileInfo.Size <= 0 {
			_ = slog.Error("file size is zero")
			return
		}
		fileResult = server.buildFileResult(&fileInfo, r)
		if output == "json" {
			if data, err = json.Marshal(fileResult); err != nil {
				_ = slog.Error(err)
				_, _ = w.Write([]byte(err.Error()))
			}
			_, _ = w.Write(data)
		} else {
			_, _ = w.Write([]byte(fileResult.Url))
		}
		return
	} else {
		// GET -> fast md5
		md5sum := r.FormValue("md5")
		output := r.FormValue("output")
		if md5sum == "" {
			_, _ = w.Write([]byte("(error) if you want to upload fast md5 is require" +
				",and if you want to upload file,you must use post method  "))
			return
		}
		//
		if v, _ := server.getFileInfoFromLevelDB(md5sum); v != nil && v.Md5 != "" {
			fileResult = server.buildFileResult(v, r)
		}
		if output == "json" {
			if data, err = json.Marshal(fileResult); err != nil {
				_ = slog.Error(err)
				_, _ = w.Write([]byte(err.Error()))
			}
			_, _ = w.Write(data)
		} else {
			_, _ = w.Write([]byte(fileResult.Url))
		}
	}
}

// 检测文件并加载到处理队列 -> 获取MD5文件中保存的文件信息 | 自动修复文件并同步集群数据服务
func (server *Service) getMd5sByDate(date string, filename string) (mapset.Set, error) {
	var (
		keys []string
	)
	md5set := mapset.NewSet()
	keyPrefix := fmt.Sprintf("%s_%s_", date, filename)
	// 根据key, 从数据库查询相关文件信息
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

// 清理(文件处理日志数据库)数据 [date, filename]
func (server *Service) cleanLogLevelDBByDate(date string, filename string) {
	defer func() {
		if re := recover(); re != nil {
			buffer := debug.Stack()
			_ = slog.Error("CleanLogLevelDBByDate")
			_ = slog.Error(re)
			_ = slog.Error(string(buffer))
		}
	}()
	var (
		err error
	)
	keys := mapset.NewSet()
	keyPrefix := fmt.Sprintf("%s_%s_", date, filename)
	//
	iter := server.logDB.NewIterator(dbutil.BytesPrefix([]byte(keyPrefix)), nil)
	for iter.Next() {
		keys.Add(string(iter.Value()))
	}
	iter.Release()
	//
	for key := range keys.Iter() {
		// 清楚数据, 文件处理日志数据库
		err = server.removeKeyFromLevelDB(key.(string), server.logDB)
		if err != nil {
			_ = slog.Error(err)
		}
	}
}

// 整理日志和元数据 -> 根据查询文件信息数据, 处理文件日志信息数据和文件元数据
func (server *Service) backUpMetaDataByDate(date string) {
	defer func() {
		if re := recover(); re != nil {
			buffer := debug.Stack()
			_ = slog.Error("BackUpMetaDataByDate")
			_ = slog.Error(re)
			_ = slog.Error(string(buffer))
		}
	}()
	var (
		err      error
		fileInfo en.FileInfo // 文件信息数据
		fileLog  *os.File    // 文件处理日志文件
		fileMeta *os.File    // 文件元数据文件
	)
	logFileName := DATA_DIR + "/" + date + "/" + CONST_FILE_Md5_FILE_NAME
	metaFileName := DATA_DIR + "/" + date + "/" + "meta.data"

	server.lockMap.LockKey(logFileName)
	defer server.lockMap.UnLockKey(logFileName)

	_ = os.MkdirAll(DATA_DIR+"/"+date, 0775)
	if util.IsExist(logFileName) {
		_ = os.Remove(logFileName)
	}

	if util.IsExist(metaFileName) {
		_ = os.Remove(metaFileName)
	}
	fileLog, err = os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0664)
	if err != nil {
		_ = slog.Error(err)
		return
	}
	defer fileLog.Close()
	fileMeta, err = os.OpenFile(metaFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0664)
	if err != nil {
		_ = slog.Error(err)
		return
	}
	defer fileMeta.Close()

	keyPrefix := fmt.Sprintf("%s_%s_", date, CONST_FILE_Md5_FILE_NAME)
	iter := server.logDB.NewIterator(dbutil.BytesPrefix([]byte(keyPrefix)), nil)
	defer iter.Release()
	for iter.Next() {
		if err = json.Unmarshal(iter.Value(), &fileInfo); err != nil {
			continue
		}
		name := fileInfo.Name
		if fileInfo.ReName != "" {
			name = fileInfo.ReName
		}
		msg := fmt.Sprintf("%s\t%s\n", fileInfo.Md5, string(iter.Value()))
		if _, err = fileMeta.WriteString(msg); err != nil {
			_ = slog.Error(err)
		}
		msg = fmt.Sprintf("%s\t%s\n", util.MD5(fileInfo.Path+"/"+name), string(iter.Value()))
		if _, err = fileMeta.WriteString(msg); err != nil {
			_ = slog.Error(err)
		}
		msg = fmt.Sprintf("%s|%d|%d|%s\n", fileInfo.Md5, fileInfo.Size, fileInfo.TimeStamp, fileInfo.Path+"/"+name)
		if _, err = fileLog.WriteString(msg); err != nil {
			_ = slog.Error(err)
		}
	}

	var fi os.FileInfo
	if fi, err = fileLog.Stat(); err != nil {
		_ = slog.Error(err)
	} else if fi.Size() == 0 {
		_ = fileLog.Close()
		_ = os.Remove(logFileName)
	}
	if fi, err = fileMeta.Stat(); err != nil {
		_ = slog.Error(err)
	} else if fi.Size() == 0 {
		_ = fileMeta.Close()
		_ = os.Remove(metaFileName)
	}
}

//
func (server *Service) loadFileInfoByDate(date string, filename string) (mapset.Set, error) {
	defer func() {
		if re := recover(); re != nil {
			buffer := debug.Stack()
			_ = slog.Error("LoadFileInfoByDate")
			_ = slog.Error(re)
			_ = slog.Error(string(buffer))
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

// 保存操作文件信息日志 -> 文件日志处理队列
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
			_ = slog.Error("saveFileMd5Log")
			_ = slog.Error(re)
			_ = slog.Error(string(buffer))
		}
	}()
	if fileInfo == nil || fileInfo.Md5 == "" || filename == "" {
		_ = slog.Warn("saveFileMd5Log", fileInfo, filename)
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
		// files.md5 -> 上传文件, 保存数据和相关操作日志
		//server.searchMap.Put(fileInfo.Md5, fileInfo.Name)
		if ok, err = server.isExistFromLevelDB(fileInfo.Md5, server.ldb); !ok {
			server.statMap.AddCountInt64(logDate+"_"+CONST_STAT_FILE_COUNT_KEY, 1)
			server.statMap.AddCountInt64(logDate+"_"+CONST_STAT_FILE_TOTAL_SIZE_KEY, fileInfo.Size)
			server.saveStat()
		}
		// 保存日志信息
		if _, err = server.saveFileInfoToLevelDB(logKey, fileInfo, server.logDB); err != nil {
			_ = slog.Error(err)
		}
		// 保存数据信息 (MD5为KEY)
		if _, err := server.saveFileInfoToLevelDB(fileInfo.Md5, fileInfo, server.ldb); err != nil {
			_ = slog.Error("saveToLevelDB", err, fileInfo)
		}
		// 保存数据信息 (路径为KEY)
		if _, err = server.saveFileInfoToLevelDB(util.MD5(fullpath), fileInfo, server.ldb); err != nil {
			_ = slog.Error("saveToLevelDB", err, fileInfo)
		}
		return
	}
	if filename == CONST_REMOME_Md5_FILE_NAME {
		// removes.md5 -> 文件删除, 保存删除日志
		//server.searchMap.Remove(fileInfo.Md5)
		if ok, err = server.isExistFromLevelDB(fileInfo.Md5, server.ldb); ok {
			server.statMap.AddCountInt64(logDate+"_"+CONST_STAT_FILE_COUNT_KEY, -1)
			server.statMap.AddCountInt64(logDate+"_"+CONST_STAT_FILE_TOTAL_SIZE_KEY, -fileInfo.Size)
			server.saveStat()
		}
		_ = server.removeKeyFromLevelDB(logKey, server.logDB)
		md5Path = util.MD5(fullpath)
		if err := server.removeKeyFromLevelDB(fileInfo.Md5, server.ldb); err != nil {
			_ = slog.Error("RemoveKeyFromLevelDB", err, fileInfo)
		}
		if err = server.removeKeyFromLevelDB(md5Path, server.ldb); err != nil {
			_ = slog.Error("RemoveKeyFromLevelDB", err, fileInfo)
		}
		// remove files.md5 for stat info(repair from logDB)
		logKey = fmt.Sprintf("%s_%s_%s", logDate, CONST_FILE_Md5_FILE_NAME, fileInfo.Md5)
		_ = server.removeKeyFromLevelDB(logKey, server.logDB)
		return
	}

	// 下载队列和错误队列 -> 保存操作日志 (不涉及数据的处理和状态变化)
	_, _ = server.saveFileInfoToLevelDB(logKey, fileInfo, server.logDB)
}

// 下载文件(从集群其他节点) -> (文件下载处理队列)
func (server *Service) downloadFromPeer(peer string, fileInfo *en.FileInfo) {
	var (
		err  error
		fi   os.FileInfo
		sum  string
		data []byte
	)
	if readOnly {
		_ = slog.Warn("ReadOnly", fileInfo)
		return
	}
	if retryCount > 0 && fileInfo.Retry >= retryCount {
		_ = slog.Error("DownloadFromPeer Error ", fileInfo)
		return
	} else {
		fileInfo.Retry = fileInfo.Retry + 1
	}

	//
	filename := fileInfo.Name
	if fileInfo.ReName != "" {
		filename = fileInfo.ReName
	}
	// 检测文件是否存在 (文件去重)
	if fileInfo.OffSet != -2 && enableDistinctFile && server.checkFileExistByInfo(fileInfo.Md5, fileInfo) {
		// ignore migrate file
		slog.Info(fmt.Sprintf("DownloadFromPeer file Exist, path:%s", fileInfo.Path+"/"+fileInfo.Name))
		return
	}
	// 检测文件是否存在 (文件不去重)
	pathTmp := server.analyseFilePathByInfo(fileInfo, true)
	if (!enableDistinctFile || fileInfo.OffSet == -2) && util.FileExists(pathTmp) {
		// ignore migrate file
		if fi, err = os.Stat(pathTmp); err == nil {
			if fi.ModTime().Unix() > fileInfo.TimeStamp {
				slog.Info(fmt.Sprintf("ignore file sync path:%s", server.analyseFilePathByInfo(fileInfo, false)))
				fileInfo.TimeStamp = fi.ModTime().Unix()
				// 检测本地文件修改时间将最新版本发布到集群
				server.postFileToPeer(fileInfo) // keep newer
				return
			}
			_ = os.Remove(pathTmp)
		}
	}

	//
	if _, err = os.Stat(fileInfo.Path); err != nil {
		_ = os.MkdirAll(DOCKER_DIR+fileInfo.Path, 0775)
	}
	//fmt.Println("downloadFromPeer",fileInfo)
	p := strings.Replace(fileInfo.Path, STORE_DIR_NAME+"/", "", 1)
	//filename= util.UrlEncode(filename)
	downloadUrl := peer + "/" + group + "/" + p + "/" + filename
	slog.Info("DownloadFromPeer: ", downloadUrl)
	fPath := DOCKER_DIR + fileInfo.Path + "/" + filename
	fPathTmp := DOCKER_DIR + fileInfo.Path + "/" + fmt.Sprintf("%s_%s", "tmp_", filename)
	timeout := fileInfo.Size/1024/1024/1 + 30
	if syncTimeout > 0 {
		timeout = syncTimeout
	}

	server.lockMap.LockKey(fPath)
	defer server.lockMap.UnLockKey(fPath)
	downloadKey := fmt.Sprintf("downloading_%d_%s", time.Now().Unix(), fPath)
	_ = server.ldb.Put([]byte(downloadKey), []byte(""), nil)
	defer func() {
		_ = server.ldb.Delete([]byte(downloadKey), nil)
	}()

	if fileInfo.OffSet == -2 {
		//migrate file
		if fi, err = os.Stat(fPath); err == nil && fi.Size() == fileInfo.Size {
			//prevent double download
			_, _ = server.saveFileInfoToLevelDB(fileInfo.Md5, fileInfo, server.ldb)
			//slog.Info(fmt.Sprintf("file '%s' has download", fpath))
			return
		}
		//
		req := httplib.Get(downloadUrl)
		req.SetTimeout(time.Second*30, time.Second*time.Duration(timeout))
		if err = req.ToFile(fPathTmp); err != nil {
			// retry
			server.appendToDownloadQueue(fileInfo)
			_ = os.Remove(fPathTmp)
			_ = slog.Error(err, fPathTmp)
			return
		}
		//
		if os.Rename(fPathTmp, fPath) == nil {
			//server.SaveFileMd5Log(fileInfo, CONST_FILE_Md5_FILE_NAME)
			_, _ = server.saveFileInfoToLevelDB(fileInfo.Md5, fileInfo, server.ldb)
		}
		return
	}

	//
	req := httplib.Get(downloadUrl)
	req.SetTimeout(time.Second*30, time.Second*time.Duration(timeout))
	if fileInfo.OffSet >= 0 {
		//small file download
		data, err = req.Bytes()
		if err != nil {
			// retry
			server.appendToDownloadQueue(fileInfo)
			_ = slog.Error(err)
			return
		}
		data2 := make([]byte, len(data)+1)
		data2[0] = '1'
		for i, v := range data {
			data2[i+1] = v
		}
		data = data2
		if int64(len(data)) != fileInfo.Size {
			_ = slog.Warn("file size is error")
			return
		}
		fPath = strings.Split(fPath, ",")[0]
		err = util.WriteFileByOffSet(fPath, fileInfo.OffSet, data)
		if err != nil {
			_ = slog.Warn(err)
			return
		}
		//
		server.appendToFileMd5LogQueue(fileInfo, CONST_FILE_Md5_FILE_NAME)
		return
	}

	if err = req.ToFile(fPathTmp); err != nil {
		// retry
		server.appendToDownloadQueue(fileInfo)
		_ = os.Remove(fPathTmp)
		_ = slog.Error(err)
		return
	}

	if fi, err = os.Stat(fPathTmp); err != nil {
		_ = os.Remove(fPathTmp)
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
		_ = slog.Error("file sum check error")
		_ = os.Remove(fPathTmp)
		return
	}

	if os.Rename(fPathTmp, fPath) == nil {
		server.appendToFileMd5LogQueue(fileInfo, CONST_FILE_Md5_FILE_NAME)
	}
}
