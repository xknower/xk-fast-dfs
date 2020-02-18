// 业务实体结构定义 (文件数据结构, 文件操作日志结构)
package en

// 文件结构定义
// ---------- ---------- ----------
type FileInfo struct {
	Name      string   `json:"name"`      // 文件名称(上传的文件名称)
	ReName    string   `json:"rename"`    // 文件完整信息名称(主要信息组成), 命名保存文件的名称
	Path      string   `json:"path"`      // 文件路径
	Md5       string   `json:"md5"`       // 文件标识 (文件(路径)Hash)
	Size      int64    `json:"size"`      // 文件大小
	Peers     []string `json:"peers"`     // 文件存在的集群节点
	Scene     string   `json:"scene"`     // 文件场景
	TimeStamp int64    `json:"timeStamp"` // 文件时间戳
	OffSet    int64    `json:"offset"`    // -1 上传后新建的文件 , -2 文件变更后的文件 | >= 0 small file | 文件偏移, 小文件合并时, 包含文件数的偏移 , 值大于等于0
	Retry     int      // 重试次数计数器 (文件下载)
	Op        string   // 操作描述, 用于描述监视过程中发生的事件类型
}

// 文件操作日志结构定义
// ---------- ---------- ----------
type FileLog struct {
	FileInfo *FileInfo // 文件信息
	FileName string    // 文件处理信息保存的文件名 (根据处理不同, 存放在不同的文件)
}

// 状态文件内容结构定义 [统计(按日期)文件数量和大小]
type StatDateFileInfo struct {
	Date      string `json:"date"`
	TotalSize int64  `json:"totalSize"`
	FileCount int64  `json:"fileCount"`
}
