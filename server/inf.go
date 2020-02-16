package server

import (
	"../en"
	mapset "github.com/deckarep/golang-set"
	"github.com/sjqzhang/goutil"
	"github.com/syndtr/goleveldb/leveldb"
	"net/http"
)

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

func (server Service) GetHost() string {
	return server.host
}

func (server Service) GetLdb() *leveldb.DB {
	return server.ldb
}

func (server Service) GetStatMap() *goutil.CommonMap {
	return server.statMap
}

func (server Service) GetSceneMap() *goutil.CommonMap {
	return server.sceneMap
}

func (server Service) GetSumMap() *goutil.CommonMap {
	return server.sumMap
}

func (server Service) GetQueueUpload() chan en.WrapReqResp {
	return server.queueUpload
}

func (server Service) GetQueueFromPeers() chan en.FileInfo {
	return server.queueFromPeers
}

func (server Service) GetQueueToPeers() chan en.FileInfo {
	return server.queueToPeers
}

func (server Service) GetQueueFileLog() chan *en.FileLog {
	return server.queueFileLog
}

func (server Service) Reload(w http.ResponseWriter, r *http.Request) {
	server.reload(w, r)
}

func (server Service) AppendToQueue(fileInfo *en.FileInfo) {
	server.appendToQueue(fileInfo)
}

func (server Service) AppendToDownloadQueue(fileInfo *en.FileInfo) {
	server.appendToDownloadQueue(fileInfo)
}

// 上传文件
func (server Service) Upload(w http.ResponseWriter, r *http.Request) {
	server.upload(w, r)
}

// 从集群(查找到文件的节点 peer)中下载文件
func (server Service) DownloadFromPeer(peer string, fileInfo *en.FileInfo) {
	server.downloadFromPeer(peer, fileInfo)
}

func (server Service) GetRequestURI(action string) string {
	return server.getRequestURI(action)
}

func (server Service) GetMd5sByDate(date string, filename string) (mapset.Set, error) {
	return server.getMd5sByDate(date, filename)
}

func (server Service) NotPermit(w http.ResponseWriter, r *http.Request) {
	server.notPermit(w, r)
}

func (server Service) CheckAuth(w http.ResponseWriter, r *http.Request) bool {
	return server.checkAuth(w, r)
}

// 检测文件是否存在, 并获取文件信息
func (server Service) CheckPeerFileExist(peer string, md5sum string, fpath string) (*en.FileInfo, error) {
	return server.checkPeerFileExist(peer, md5sum, fpath)
}

func (server Service) VerifyGoogleCode(secret string, code string, discrepancy int64) bool {
	return server.verifyGoogleCode(secret, code, discrepancy)
}

func (server Service) CheckFileAndSendToPeer(date string, filename string, isForceUpload bool) {
	server.checkFileAndSendToPeer(date, filename, isForceUpload)
}

func (server Service) GetFileInfoFromLevelDB(key string) (*en.FileInfo, error) {
	return server.getFileInfoFromLevelDB(key)
}

func (server Service) SaveFileInfoToLevelDB(key string, fileInfo *en.FileInfo, db *leveldb.DB) (*en.FileInfo, error) {
	return server.saveFileInfoToLevelDB(key, fileInfo, db)
}

func (server Service) SaveFileMd5Log(fileInfo *en.FileInfo, filename string) {
	server.appendToFileMd5LogQueue(fileInfo, filename)
}

func (server Service) BackUpMetaDataByDate(date string) {
	server.backUpMetaDataByDate(date)
}

func (server Service) RemoveKeyFromLevelDB(key string, db *leveldb.DB) error {
	return server.removeKeyFromLevelDB(key, db)
}

func (server Service) AutoRepair(forceRepair bool) {
	server.autoRepair(forceRepair)
}

func (server Service) RepairStatByDate(date string) en.StatDateFileInfo {
	return server.repairStatByDate(date)
}

func (server Service) RepairFileInfoFromFile() {
	server.repairFileInfoFromFile()
}
