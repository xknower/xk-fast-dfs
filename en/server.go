// 服务端接口定义
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

	// 重新加载后台服务
	Reload(w http.ResponseWriter, r *http.Request)

	//
	AppendToQueue(fileInfo *FileInfo)
	AppendToDownloadQueue(fileInfo *FileInfo)

	// 上传文件
	Upload(w http.ResponseWriter, r *http.Request)

	// 文件下载
	DownloadFromPeer(peer string, fileInfo *FileInfo)                               // 从集群(查找到文件的节点 peer)中下载文件
	NotPermit(w http.ResponseWriter, r *http.Request)                               // 返回 401 , 没有访问权限
	CheckAuth(w http.ResponseWriter, r *http.Request) bool                          //
	CheckPeerFileExist(peer string, md5sum string, fpath string) (*FileInfo, error) // 检测文件是否存在, 并获取文件信息
	GetFileInfoFromLevelDB(key string) (*FileInfo, error)                           // 从数据库中查询文件
	VerifyGoogleCode(secret string, code string, discrepancy int64) bool

	RemoveKeyFromLevelDB(key string, db *leveldb.DB) error                                   // 从数据库删除文件信息
	SaveFileInfoToLevelDB(key string, fileInfo *FileInfo, db *leveldb.DB) (*FileInfo, error) // 保存文件信息到数据库
	SaveFileMd5Log(fileInfo *FileInfo, filename string)                                      // 保存文件日志信息数据
	CheckFileAndSendToPeer(date string, filename string, isForceUpload bool)
	//
	AutoRepair(forceRepair bool)
	RepairStatByDate(date string) StatDateFileInfo
	RepairFileInfoFromFile()
	BackUpMetaDataByDate(date string)

	GetRequestURI(action string) string
	GetMd5sByDate(date string, filename string) (mapset.Set, error)
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
