package server

import (
	"../en"
	"../web"
	"errors"
	"fmt"
	"github.com/astaxie/beego/httplib"
	slog "github.com/sjqzhang/seelog"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"
)

//
func (server *Service) CheckAuth(w http.ResponseWriter, r *http.Request) bool {
	var (
		err        error
		req        *httplib.BeegoHTTPRequest
		result     string
		jsonResult web.JsonResult
	)
	if err = r.ParseForm(); err != nil {
		slog.Error(err)
		return false
	}
	req = httplib.Post(authUrl)
	req.SetTimeout(time.Second*10, time.Second*10)
	req.Param("__path__", r.URL.Path)
	req.Param("__query__", r.URL.RawQuery)
	for k, _ := range r.Form {
		req.Param(k, r.FormValue(k))
	}
	for k, v := range r.Header {
		req.Header(k, v[0])
	}
	result, err = req.String()
	result = strings.TrimSpace(result)
	if strings.HasPrefix(result, "{") && strings.HasSuffix(result, "}") {
		if err = json.Unmarshal([]byte(result), &jsonResult); err != nil {
			slog.Error(err)
			return false
		}
		if jsonResult.Data != "ok" {
			slog.Warn(result)
			return false
		}
	} else {
		if result != "ok" {
			slog.Warn(result)
			return false
		}
	}
	return true
}

//
func (server *Service) checkPeerFileExist(peer string, md5sum string, fpath string) (*en.FileInfo, error) {
	var fileInfo en.FileInfo

	req := httplib.Post(fmt.Sprintf("%s%s?md5=%s", peer, server.getRequestURI("check_file_exist"), md5sum))
	req.Param("path", fpath)
	req.Param("md5", md5sum)
	req.SetTimeout(time.Second*5, time.Second*10)
	if err := req.ToJSON(&fileInfo); err != nil {
		return &en.FileInfo{}, err
	}
	if fileInfo.Md5 == "" {
		return &fileInfo, errors.New("not found")
	}
	return &fileInfo, nil
}

//
func (server *Service) postFileToPeer(fileInfo *en.FileInfo) {
	var (
		err      error
		peer     string
		filename string
		info     *en.FileInfo
		postURL  string
		result   string
		fi       os.FileInfo
		i        int
		data     []byte
		fpath    string
	)
	defer func() {
		if re := recover(); re != nil {
			buffer := debug.Stack()
			slog.Error("postFileToPeer")
			slog.Error(re)
			slog.Error(string(buffer))
		}
	}()
	//fmt.Println("postFile",fileInfo)
	for i, peer = range peers {
		_ = i
		if fileInfo.Peers == nil {
			fileInfo.Peers = []string{}
		}
		if util.Contains(peer, fileInfo.Peers) {
			continue
		}
		filename = fileInfo.Name
		if fileInfo.ReName != "" {
			filename = fileInfo.ReName
			if fileInfo.OffSet != -1 {
				filename = strings.Split(fileInfo.ReName, ",")[0]
			}
		}
		fpath = DOCKER_DIR + fileInfo.Path + "/" + filename
		if !util.FileExists(fpath) {
			slog.Warn(fmt.Sprintf("file '%s' not found", fpath))
			continue
		} else {
			if fileInfo.Size == 0 {
				if fi, err = os.Stat(fpath); err != nil {
					slog.Error(err)
				} else {
					fileInfo.Size = fi.Size()
				}
			}
		}
		if fileInfo.OffSet != -2 && enableDistinctFile {
			//not migrate file should check or update file
			// where not EnableDistinctFile should check
			if info, err = server.checkPeerFileExist(peer, fileInfo.Md5, ""); info.Md5 != "" {
				fileInfo.Peers = append(fileInfo.Peers, peer)
				if _, err = server.saveFileInfoToLevelDB(fileInfo.Md5, fileInfo, server.ldb); err != nil {
					slog.Error(err)
				}
				continue
			}
		}
		postURL = fmt.Sprintf("%s%s", peer, server.getRequestURI("syncfile_info"))
		b := httplib.Post(postURL)
		b.SetTimeout(time.Second*30, time.Second*30)
		if data, err = json.Marshal(fileInfo); err != nil {
			slog.Error(err)
			return
		}
		b.Param("fileInfo", string(data))
		result, err = b.String()
		if err != nil {
			if fileInfo.Retry <= retryCount {
				fileInfo.Retry = fileInfo.Retry + 1
				server.AppendToQueue(fileInfo)
			}
			slog.Error(err, fmt.Sprintf(" path:%s", fileInfo.Path+"/"+fileInfo.Name))
		}
		if !strings.HasPrefix(result, "http://") || err != nil {
			server.SaveFileMd5Log(fileInfo, CONST_Md5_ERROR_FILE_NAME)
		}
		if strings.HasPrefix(result, "http://") {
			slog.Info(result)
			if !util.Contains(peer, fileInfo.Peers) {
				fileInfo.Peers = append(fileInfo.Peers, peer)
				if _, err = server.saveFileInfoToLevelDB(fileInfo.Md5, fileInfo, server.ldb); err != nil {
					slog.Error(err)
				}
			}
		}
		if err != nil {
			slog.Error(err)
		}
	}
}

// 文件下载 -> 处理文件下载队列服务
func (server *Service) DownloadFromPeer(peer string, fileInfo *en.FileInfo) {
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
	if fileInfo.OffSet != -2 && enableDistinctFile && server.CheckFileExistByInfo(fileInfo.Md5, fileInfo) {
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
			server.AppendToDownloadQueue(fileInfo) //retry
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
			server.AppendToDownloadQueue(fileInfo) //retry
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
		server.SaveFileMd5Log(fileInfo, CONST_FILE_Md5_FILE_NAME)
		return
	}
	if err = req.ToFile(fpathTmp); err != nil {
		server.AppendToDownloadQueue(fileInfo) //retry
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
		server.SaveFileMd5Log(fileInfo, CONST_FILE_Md5_FILE_NAME)
	}
}
