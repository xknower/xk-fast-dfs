package server

import (
	"../en"
	"fmt"
	mapset "github.com/deckarep/golang-set"
	slog "github.com/sjqzhang/seelog"
	"github.com/syndtr/goleveldb/leveldb"
	dbutil "github.com/syndtr/goleveldb/leveldb/util"
	"os"
	"runtime/debug"
	"strings"
)

// 检测文件并加载到处理队列 -> 获取MD5文件中保存的文件信息 | 自动修复文件并同步集群数据服务
func (server *Service) GetMd5sByDate(date string, filename string) (mapset.Set, error) {
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
		err = server.RemoveKeyFromLevelDB(key.(string), server.logDB)
		if err != nil {
			slog.Error(err)
		}
	}
}

//
func (server *Service) RemoveKeyFromLevelDB(key string, db *leveldb.DB) error {
	return db.Delete([]byte(key), nil)
}

// 备份 -> 定期清理及备份数据服务
func (server *Service) BackUpMetaDataByDate(date string) {
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

//
func (server *Service) IsExistFromLevelDB(key string, db *leveldb.DB) (bool, error) {
	return db.Has([]byte(key), nil)
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
		if ok, err = server.IsExistFromLevelDB(fileInfo.Md5, server.ldb); !ok {
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
		if ok, err = server.IsExistFromLevelDB(fileInfo.Md5, server.ldb); ok {
			server.statMap.AddCountInt64(logDate+"_"+CONST_STAT_FILE_COUNT_KEY, -1)
			server.statMap.AddCountInt64(logDate+"_"+CONST_STAT_FILE_TOTAL_SIZE_KEY, -fileInfo.Size)
			server.saveStat()
		}
		server.RemoveKeyFromLevelDB(logKey, server.logDB)
		md5Path = util.MD5(fullpath)
		if err := server.RemoveKeyFromLevelDB(fileInfo.Md5, server.ldb); err != nil {
			slog.Error("RemoveKeyFromLevelDB", err, fileInfo)
		}
		if err = server.RemoveKeyFromLevelDB(md5Path, server.ldb); err != nil {
			slog.Error("RemoveKeyFromLevelDB", err, fileInfo)
		}
		// remove files.md5 for stat info(repair from logDB)
		logKey = fmt.Sprintf("%s_%s_%s", logDate, CONST_FILE_Md5_FILE_NAME, fileInfo.Md5)
		server.RemoveKeyFromLevelDB(logKey, server.logDB)
		return
	}
	server.saveFileInfoToLevelDB(logKey, fileInfo, server.logDB)
}
