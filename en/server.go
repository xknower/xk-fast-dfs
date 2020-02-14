package en

import (
	mapset "github.com/deckarep/golang-set"
	"github.com/sjqzhang/goutil"
	"github.com/syndtr/goleveldb/leveldb"
	"net/http"
)

// 服务端定义
type Server interface {
	GetServerName() string     // 获取服务名称
	GetGroupRouteName() string // 获取访问路由名称, 未配置使用服务名称
	GetHost() string
	GetLdb() *leveldb.DB
	GetStatMap() *goutil.CommonMap
	GetSceneMap() *goutil.CommonMap
	GetSumMap() *goutil.CommonMap
	GetQueueUpload() chan WrapReqResp
	GetQueueFromPeers() chan FileInfo
	GetQueueToPeers() chan FileInfo
	GetQueueFileLog() chan *FileLog

	Reload(w http.ResponseWriter, r *http.Request)
	AppendToQueue(fileInfo *FileInfo)
	AppendToDownloadQueue(fileInfo *FileInfo)
	Upload(w http.ResponseWriter, r *http.Request)
	DownloadFromPeer(peer string, fileInfo *FileInfo)

	GetRequestURI(action string) string
	GetMd5sByDate(date string, filename string) (mapset.Set, error)
	NotPermit(w http.ResponseWriter, r *http.Request)
	CheckAuth(w http.ResponseWriter, r *http.Request) bool
	VerifyGoogleCode(secret string, code string, discrepancy int64) bool
	CheckPeerFileExist(peer string, md5sum string, fpath string) (*FileInfo, error)
	CheckFileAndSendToPeer(date string, filename string, isForceUpload bool)

	GetFileInfoFromLevelDB(key string) (*FileInfo, error)
	SaveFileInfoToLevelDB(key string, fileInfo *FileInfo, db *leveldb.DB) (*FileInfo, error)
	SaveFileMd5Log(fileInfo *FileInfo, filename string)
	BackUpMetaDataByDate(date string)
	RemoveKeyFromLevelDB(key string, db *leveldb.DB) error
	AutoRepair(forceRepair bool)
	RepairStatByDate(date string) StatDateFileInfo
	RepairFileInfoFromFile()
}

// 服务端实现
// ---------- ----------
// name  服务器唯一名称
// group 访问路由-分组(路由名称)
// ---------- ----------
type DefaultServer struct {
	name  string
	group string
}

// 获取服务名称
func (s DefaultServer) GetServerName() string {
	return s.name
}

// 获取访问路由名称, 未配置使用服务名称
func (s DefaultServer) GetGroupRouteName() string {
	if s.group == "" {
		return s.name
	}
	return s.group
}

// 初始化服务端
func NewServer(name, group string) *DefaultServer {
	return &DefaultServer{
		name, group,
	}
}
