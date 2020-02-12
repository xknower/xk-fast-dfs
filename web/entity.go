package web

import (
	"../conf"
	"errors"
	"fmt"
	"github.com/astaxie/beego/httplib"
	_ "github.com/eventials/go-tus"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sjqzhang/seelog"
	"github.com/sjqzhang/tusd"
	_ "net/http/pprof"
	"strings"
	"time"
)

// JSON 解析
var json = jsoniter.ConfigCompatibleWithStandardLibrary

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

func (err HttpError) StatusCode() int {
	return err.statusCode
}

func (err HttpError) Body() []byte {
	return []byte(err.Error())
}

// 存储对象
type HookDataStore struct {
	tusd.DataStore
}

// 上传文件
func (store HookDataStore) NewUpload(info tusd.FileInfo) (id string, err error) {
	var jsonResult JsonResult
	if conf.Global().AuthUrl != "" {
		if auth_token, ok := info.MetaData["auth_token"]; !ok {
			msg := "token auth fail,auth_token is not in http header Upload-Metadata," +
				"in uppy uppy.setMeta({ auth_token: '9ee60e59-cb0f-4578-aaba-29b9fc2919ca' })"
			log.Error(msg, fmt.Sprintf("current header:%v", info.MetaData))
			return "", HttpError{error: errors.New(msg), statusCode: 401}
		} else {
			req := httplib.Post(conf.Global().AuthUrl)
			req.Param("auth_token", auth_token)
			req.SetTimeout(time.Second*5, time.Second*10)
			content, err := req.String()
			content = strings.TrimSpace(content)
			if strings.HasPrefix(content, "{") && strings.HasSuffix(content, "}") {
				if err = json.Unmarshal([]byte(content), &jsonResult); err != nil {
					log.Error(err)
					return "", HttpError{error: errors.New(err.Error() + content), statusCode: 401}
				}
				if jsonResult.Data != "ok" {
					return "", HttpError{error: errors.New(content), statusCode: 401}
				}
			} else {
				if err != nil {
					log.Error(err)
					return "", err
				}
				if strings.TrimSpace(content) != "ok" {
					return "", HttpError{error: errors.New(content), statusCode: 401}
				}
			}
		}
	}
	return store.DataStore.NewUpload(info)
}
