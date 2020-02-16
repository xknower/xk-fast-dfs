package en

// 文件结构定义
// ---------- ---------- ----------
type FileInfo struct {
	Name      string   `json:"name"`   // 文件名称
	ReName    string   `json:"rename"` // 文件完整信息名称(主要信息组成)
	Path      string   `json:"path"`   // 文件路径
	Md5       string   `json:"md5"`    // 文件标识, 文件(路径)Hash
	Size      int64    `json:"size"`   // 文件大小
	Peers     []string `json:"peers"`  // 文件存在的集群节点
	Scene     string   `json:"scene"`
	TimeStamp int64    `json:"timeStamp"` // 文件时间戳
	OffSet    int64    `json:"offset"`    // >=0 small file, -2, -1
	Retry     int      // 重试次数计数器 (文件下载)
	Op        string
}

// 文件操作日志结构定义
// ---------- ---------- ----------
type FileLog struct {
	FileInfo *FileInfo
	FileName string
}

type FileResult struct {
	Url     string `json:"url"`
	Md5     string `json:"md5"`
	Path    string `json:"path"`
	Domain  string `json:"domain"`
	Scene   string `json:"scene"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"mtime"`
	//Just for Compatibility
	Scenes  string `json:"scenes"`
	Retmsg  string `json:"retmsg"`
	Retcode int    `json:"retcode"`
	Src     string `json:"src"`
}

type StatDateFileInfo struct {
	Date      string `json:"date"`
	TotalSize int64  `json:"totalSize"`
	FileCount int64  `json:"fileCount"`
}

type FileInfoResult struct {
	Name    string `json:"name"`
	Md5     string `json:"md5"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"mtime"`
	IsDir   bool   `json:"is_dir"`
}
