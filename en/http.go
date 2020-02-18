// HTTP 响应数据结构定义
package en

import "net/http"

// HTTP 返回结果
type JsonResult struct {
	Message string      `json:"message"`
	Status  string      `json:"status"`
	Data    interface{} `json:"data"`
}

// HTTP 异常状态
type HttpError struct {
	error
	statusCode int
}

// HTTP 异常编码
func (err HttpError) StatusCode() int {
	return err.statusCode
}

// HTTP 异常描述
func (err HttpError) Body() []byte {
	return []byte(err.Error())
}

// HTTP 请求处理实体 [文件上传队列]
type WrapReqResp struct {
	W    *http.ResponseWriter
	R    *http.Request
	Done chan bool
}

// 文件上传成功, 结果返回文件实体结构定义
type FileUploadResult struct {
	Url     string `json:"url"`    // 下载地址
	Md5     string `json:"md5"`    // 文件标识
	Path    string `json:"path"`   // 文件路径 (下载路径)
	Src     string `json:"src"`    // 文件路径 (存储路径)
	Domain  string `json:"domain"` // 下载域, 配置的下载地址或者获取本节点地址
	Scene   string `json:"scene"`  // 文件场景
	Size    int64  `json:"size"`   // 文件大小
	ModTime int64  `json:"mtime"`  // 文件时间戳
	Scenes  string `json:"scenes"`
	Retmsg  string `json:"retmsg"`
	Retcode int    `json:"retcode"`
}

// 查询目录文件信息, 结果返回文件实体结构定义
type FileInfoResult struct {
	Name    string `json:"name"`   // 文件名
	Md5     string `json:"md5"`    // 文件标识
	Path    string `json:"path"`   // 文件路径
	Size    int64  `json:"size"`   // 文件大小
	ModTime int64  `json:"mtime"`  // 时间戳
	IsDir   bool   `json:"is_dir"` // 是否为目录
}
