// 集权文件操作
// 包括文件保存到集群, 从集群中获取文件, 从集群中查询文件信息, 校验集群文件信息
package server

import (
	"../en"
	"errors"
	"fmt"
	"github.com/astaxie/beego/httplib"
	slog "github.com/sjqzhang/seelog"
	"github.com/syndtr/goleveldb/leveldb"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"
)

// 根据文件 MD5值 (Hash值 - 根据文件路径计算计算Hash), 获取文件信息
// 获取文件信息(从数据库加载文件信息) -> 检测文件并加载到处理队列 | 自动修复文件并同步集群数据服务
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

// 文件文件Hash, 判断文件在数据库是否存在
func (server *Service) isExistFromLevelDB(key string, db *leveldb.DB) (bool, error) {
	return db.Has([]byte(key), nil)
}

// 保存文件信息到数据库
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
	if db == server.ldb {
		// search slow ,write fast, double write logDB (搜索速度慢，写入速度快，双写入日志数据库)
		logDate := util.GetDayFromTimeStamp(fileInfo.TimeStamp)
		logKey := fmt.Sprintf("%s_%s_%s", logDate, CONST_FILE_Md5_FILE_NAME, fileInfo.Md5)
		_ = server.logDB.Put([]byte(logKey), data, nil)
	}
	return fileInfo, nil
}

// 根据文件Hash, 从数据库删除文件
func (server *Service) removeKeyFromLevelDB(key string, db *leveldb.DB) error {
	return db.Delete([]byte(key), nil)
}

// 处理上传文件信息
func (server *Service) processUploadFile(file multipart.File, header *multipart.FileHeader, fileInfo *en.FileInfo, r *http.Request) (*en.FileInfo, error) {
	var (
		err     error
		outFile *os.File
		folder  string
		fi      os.FileInfo
	)
	defer file.Close()

	_, fileInfo.Name = filepath.Split(header.Filename)
	// bugfix for ie upload file contain fullpath
	if len(extensions) > 0 && !util.Contains(path.Ext(fileInfo.Name), extensions) {
		return fileInfo, errors.New("(error)file extension mismatch")
	}

	if renameFile {
		fileInfo.ReName = util.MD5(util.GetUUID()) + path.Ext(fileInfo.Name)
	}
	folder = time.Now().Format("20060102/15/04")
	if peerId != "" {
		folder = fmt.Sprintf(folder+"/%s", peerId)
	}
	if fileInfo.Scene != "" {
		folder = fmt.Sprintf(STORE_DIR+"/%s/%s", fileInfo.Scene, folder)
	} else {
		folder = fmt.Sprintf(STORE_DIR+"/%s", folder)
	}
	if fileInfo.Path != "" {
		if strings.HasPrefix(fileInfo.Path, STORE_DIR) {
			folder = fileInfo.Path
		} else {
			folder = STORE_DIR + "/" + fileInfo.Path
		}
	}
	if !util.FileExists(folder) {
		_ = os.MkdirAll(folder, 0775)
	}
	outPath := fmt.Sprintf(folder+"/%s", fileInfo.Name)
	if fileInfo.ReName != "" {
		outPath = fmt.Sprintf(folder+"/%s", fileInfo.ReName)
	}
	if util.FileExists(outPath) && enableDistinctFile {
		for i := 0; i < 10000; i++ {
			outPath = fmt.Sprintf(folder+"/%d_%s", i, filepath.Base(header.Filename))
			fileInfo.Name = fmt.Sprintf("%d_%s", i, header.Filename)
			if !util.FileExists(outPath) {
				break
			}
		}
	}
	slog.Info(fmt.Sprintf("upload: %s", outPath))
	if outFile, err = os.Create(outPath); err != nil {
		return fileInfo, err
	}
	defer outFile.Close()
	//
	if err != nil {
		_ = slog.Error(err)
		return fileInfo, errors.New("(error)fail," + err.Error())
	}
	if _, err = io.Copy(outFile, file); err != nil {
		_ = slog.Error(err)
		return fileInfo, errors.New("(error)fail," + err.Error())
	}
	if fi, err = outFile.Stat(); err != nil {
		_ = slog.Error(err)
	} else {
		fileInfo.Size = fi.Size()
	}
	if fi.Size() != header.Size {
		return fileInfo, errors.New("(error)file uncomplete")
	}
	v := "" // util.GetFileSum(outFile, Config().FileSumArithmetic)
	if enableDistinctFile {
		v = util.GetFileSum(outFile, fileSumArithmetic)
	} else {
		v = util.MD5(server.analyseFilePathByInfo(fileInfo, false))
	}
	fileInfo.Md5 = v
	//fileInfo.Path = folder //strings.Replace( folder,DOCKER_DIR,"",1)
	fileInfo.Path = strings.Replace(folder, DOCKER_DIR, "", 1)
	fileInfo.Peers = append(fileInfo.Peers, server.host)
	//fmt.Println("upload",fileInfo)
	return fileInfo, nil
}

// 保存小文件
func (server *Service) saveSmallFile(fileInfo *en.FileInfo) error {
	var (
		err      error
		filename string
		fpath    string
		srcFile  *os.File
		desFile  *os.File
		largeDir string
		destPath string
		reName   string
		fileExt  string
	)
	filename = fileInfo.Name
	fileExt = path.Ext(filename)
	if fileInfo.ReName != "" {
		filename = fileInfo.ReName
	}
	fpath = DOCKER_DIR + fileInfo.Path + "/" + filename
	largeDir = LARGE_DIR + "/" + peerId
	if !util.FileExists(largeDir) {
		_ = os.MkdirAll(largeDir, 0775)
	}
	reName = fmt.Sprintf("%d", util.RandInt(100, 300))
	destPath = largeDir + "/" + reName
	server.lockMap.LockKey(destPath)
	defer server.lockMap.UnLockKey(destPath)
	if util.FileExists(fpath) {
		srcFile, err = os.OpenFile(fpath, os.O_CREATE|os.O_RDONLY, 06666)
		if err != nil {
			return err
		}
		defer srcFile.Close()
		desFile, err = os.OpenFile(destPath, os.O_CREATE|os.O_RDWR, 06666)
		if err != nil {
			return err
		}
		defer desFile.Close()
		fileInfo.OffSet, err = desFile.Seek(0, 2)
		if _, err = desFile.Write([]byte("1")); err != nil {
			//first byte set 1
			return err
		}
		fileInfo.OffSet, err = desFile.Seek(0, 2)
		if err != nil {
			return err
		}
		fileInfo.OffSet = fileInfo.OffSet - 1 //minus 1 byte
		fileInfo.Size = fileInfo.Size + 1
		fileInfo.ReName = fmt.Sprintf("%s,%d,%d,%s", reName, fileInfo.OffSet, fileInfo.Size, fileExt)
		if _, err = io.Copy(desFile, srcFile); err != nil {
			return err
		}
		_ = srcFile.Close()
		_ = os.Remove(fpath)
		fileInfo.Path = strings.Replace(largeDir, DOCKER_DIR, "", 1)
	}
	return nil
}

// 文件保存到集群 (文件上传处理队列)
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
			_ = slog.Error("postFileToPeer")
			_ = slog.Error(re)
			_ = slog.Error(string(buffer))
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
			_ = slog.Warn(fmt.Sprintf("file '%s' not found", fpath))
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
					_ = slog.Error(err)
				}
				continue
			}
		}
		postURL = fmt.Sprintf("%s%s", peer, server.analyseRequestURI("syncfile_info"))
		b := httplib.Post(postURL)
		b.SetTimeout(time.Second*30, time.Second*30)
		if data, err = json.Marshal(fileInfo); err != nil {
			_ = slog.Error(err)
			return
		}
		b.Param("fileInfo", string(data))
		result, err = b.String()
		if err != nil {
			if fileInfo.Retry <= retryCount {
				fileInfo.Retry = fileInfo.Retry + 1
				server.appendToQueue(fileInfo)
			}
			_ = slog.Error(err, fmt.Sprintf(" path:%s", fileInfo.Path+"/"+fileInfo.Name))
		}
		if !strings.HasPrefix(result, "http://") || err != nil {
			server.appendToFileMd5LogQueue(fileInfo, CONST_Md5_ERROR_FILE_NAME)
		}
		if strings.HasPrefix(result, "http://") {
			slog.Info(result)
			if !util.Contains(peer, fileInfo.Peers) {
				fileInfo.Peers = append(fileInfo.Peers, peer)
				if _, err = server.saveFileInfoToLevelDB(fileInfo.Md5, fileInfo, server.ldb); err != nil {
					_ = slog.Error(err)
				}
			}
		}
		if err != nil {
			_ = slog.Error(err)
		}
	}
}
