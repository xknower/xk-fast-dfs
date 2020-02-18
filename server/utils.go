// 公共方法实现
package server

import (
	"../en"
	"errors"
	"fmt"
	"github.com/astaxie/beego/httplib"
	"github.com/sjqzhang/googleAuthenticator"
	slog "github.com/sjqzhang/seelog"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"
)

// 构建请求URL
func (server *Service) analyseRequestURI(action string) string {
	var uri string
	if supportGroupManage {
		uri = "/" + group + "/" + action
	} else {
		uri = "/" + action
	}
	return uri
}

// 获取文件路径, 文件信息中解析
func (server *Service) analyseFilePathByInfo(fileInfo *en.FileInfo, withDocker bool) string {
	fn := fileInfo.Name
	if fileInfo.ReName != "" {
		fn = fileInfo.ReName
	}
	if withDocker {
		return DOCKER_DIR + fileInfo.Path + "/" + fn
	}
	return fileInfo.Path + "/" + fn
}

// 通过文件信息, 构建文件信息结果数据
func (server *Service) buildFileResult(fileInfo *en.FileInfo, r *http.Request) en.FileUploadResult {
	host := strings.Replace(host, "http://", "", -1)
	if r != nil {
		host = r.Host
	}
	if !strings.HasPrefix(downloadDomain, "http") {
		if downloadDomain == "" {
			downloadDomain = fmt.Sprintf("http://%s", host)
		} else {
			downloadDomain = fmt.Sprintf("http://%s", downloadDomain)
		}
	}
	var domain string
	if downloadDomain != "" {
		domain = downloadDomain
	} else {
		domain = fmt.Sprintf("http://%s", host)
	}
	//
	outName := fileInfo.Name
	if fileInfo.ReName != "" {
		outName = fileInfo.ReName
	}
	//
	p := strings.Replace(fileInfo.Path, STORE_DIR_NAME+"/", "", 1)
	if supportGroupManage {
		p = group + "/" + p + "/" + outName
	} else {
		p = p + "/" + outName
	}
	downloadUrl := fmt.Sprintf("http://%s/%s", host, p)
	if downloadDomain != "" {
		downloadUrl = fmt.Sprintf("%s/%s", downloadDomain, p)
	}

	// 返回构建的文件信息
	var fileResult en.FileUploadResult
	fileResult.Url = downloadUrl
	fileResult.Md5 = fileInfo.Md5
	fileResult.Path = "/" + p
	fileResult.Domain = domain
	fileResult.Scene = fileInfo.Scene
	fileResult.Size = fileInfo.Size
	fileResult.ModTime = fileInfo.TimeStamp
	fileResult.Src = fileResult.Path
	fileResult.Scenes = fileInfo.Scene
	return fileResult
}

// 发送邮件
func (server *Service) sendToMail(to, subject, body, mailType string) error {
	host := mail.Host
	user := mail.User
	password := mail.Password
	hp := strings.Split(host, ":")
	auth := smtp.PlainAuth("", user, password, hp[0])
	var contentType string
	if mailType == "html" {
		contentType = "Content-Type: text/" + mailType + "; charset=UTF-8"
	} else {
		contentType = "Content-Type: text/plain" + "; charset=UTF-8"
	}
	msg := []byte("To: " + to + "\r\nFrom: " + user + ">\r\nSubject: " + "\r\n" + contentType + "\r\n\r\n" + body)
	sendTo := strings.Split(to, ";")
	err := smtp.SendMail(host, auth, user, sendTo, msg)
	return err
}

//
func (server *Service) verifyGoogleCode(secret string, code string, discrepancy int64) bool {
	var (
		goauth *googleAuthenticator.GAuth
	)
	goauth = googleAuthenticator.NewGAuth()
	if ok, err := goauth.VerifyCode(secret, code, discrepancy); ok {
		return ok
	} else {
		_ = slog.Error(err)
		return ok
	}
}

//
func (server *Service) checkScene(scene string) (bool, error) {
	var (
		scenes []string
	)
	if len(scenes) == 0 {
		return true, nil
	}
	for _, s := range scenes {
		scenes = append(scenes, strings.Split(s, ":")[0])
	}
	if !util.Contains(scene, scenes) {
		return false, errors.New("not valid scene")
	}
	return true, nil
}

// 返回 401, 没有权限访问
func (server *Service) notPermit(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(401)
}

// 检测操作权限
func (server *Service) checkAuth(w http.ResponseWriter, r *http.Request) bool {
	var (
		err        error
		req        *httplib.BeegoHTTPRequest
		result     string
		jsonResult en.JsonResult
	)
	if err = r.ParseForm(); err != nil {
		_ = slog.Error(err)
		return false
	}
	//
	req = httplib.Post(authUrl)
	req.SetTimeout(time.Second*10, time.Second*10)
	req.Param("__path__", r.URL.Path)
	req.Param("__query__", r.URL.RawQuery)
	for k, _ := range r.Form {
		req.Param(k, r.FormValue(k))
	}
	for k, v := range r.Header {
		req.Header(k, v[0])
	}
	result, err = req.String()
	result = strings.TrimSpace(result)
	if strings.HasPrefix(result, "{") && strings.HasSuffix(result, "}") {
		if err = json.Unmarshal([]byte(result), &jsonResult); err != nil {
			_ = slog.Error(err)
			return false
		}
		if jsonResult.Data != "ok" {
			_ = slog.Warn(result)
			return false
		}
	} else {
		if result != "ok" {
			_ = slog.Warn(result)
			return false
		}
	}
	return true
}

// 检测文件在集群某个节点是否存在  [peer, md5sum, path]
func (server *Service) checkPeerFileExist(peer string, md5sum string, path string) (*en.FileInfo, error) {
	var fileInfo en.FileInfo
	//
	req := httplib.Post(fmt.Sprintf("%s%s?md5=%s", peer, server.analyseRequestURI("check_file_exist"), md5sum))
	req.Param("path", path)
	req.Param("md5", md5sum)
	req.SetTimeout(time.Second*5, time.Second*10)
	if err := req.ToJSON(&fileInfo); err != nil {
		return &en.FileInfo{}, err
	}
	if fileInfo.Md5 == "" {
		return &fileInfo, errors.New("not found")
	}
	return &fileInfo, nil
}

// 根据文件信息, 查询文件是否存在
func (server *Service) checkFileExistByInfo(md5s string, fileInfo *en.FileInfo) bool {
	var (
		err  error
		fi   os.FileInfo
		info *en.FileInfo
	)
	if fileInfo == nil {
		return false
	}
	if fileInfo.OffSet >= 0 {
		// small file
		if info, err = server.getFileInfoFromLevelDB(fileInfo.Md5); err == nil && info.Md5 == fileInfo.Md5 {
			return true
		} else {
			return false
		}
	}
	//
	fullPath := server.analyseFilePathByInfo(fileInfo, true)
	if fi, err = os.Stat(fullPath); err != nil {
		return false
	}
	if fi.Size() == fileInfo.Size {
		return true
	} else {
		return false
	}
}
