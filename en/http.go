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

// 查询目录文件信息, 结果返回文件实体结构定义
type FileInfoResult struct {
	Name    string `json:"name"`
	Md5     string `json:"md5"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"mtime"`
	IsDir   bool   `json:"is_dir"`
}
