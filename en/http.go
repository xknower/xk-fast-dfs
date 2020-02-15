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
