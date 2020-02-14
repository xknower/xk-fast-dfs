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
func (server *Service) getRequestURI(action string) string {
	var uri string
	if supportGroupManage {
		uri = "/" + group + "/" + action
	} else {
		uri = "/" + action
	}
	return uri
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
		slog.Error(err)
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

//
func (server *Service) notPermit(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(401)
}

//
func (server *Service) checkAuth(w http.ResponseWriter, r *http.Request) bool {
	var (
		err        error
		req        *httplib.BeegoHTTPRequest
		result     string
		jsonResult en.JsonResult
	)
	if err = r.ParseForm(); err != nil {
		slog.Error(err)
		return false
	}
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
			slog.Error(err)
			return false
		}
		if jsonResult.Data != "ok" {
			slog.Warn(result)
			return false
		}
	} else {
		if result != "ok" {
			slog.Warn(result)
			return false
		}
	}
	return true
}

//
func (server *Service) checkPeerFileExist(peer string, md5sum string, fpath string) (*en.FileInfo, error) {
	var fileInfo en.FileInfo

	req := httplib.Post(fmt.Sprintf("%s%s?md5=%s", peer, server.getRequestURI("check_file_exist"), md5sum))
	req.Param("path", fpath)
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

//
func (server *Service) checkFileExistByInfo(md5s string, fileInfo *en.FileInfo) bool {
	var (
		err      error
		fullpath string
		fi       os.FileInfo
		info     *en.FileInfo
	)
	if fileInfo == nil {
		return false
	}
	if fileInfo.OffSet >= 0 {
		//small file
		if info, err = server.getFileInfoFromLevelDB(fileInfo.Md5); err == nil && info.Md5 == fileInfo.Md5 {
			return true
		} else {
			return false
		}
	}
	fullpath = server.GetFilePathByInfo(fileInfo, true)
	if fi, err = os.Stat(fullpath); err != nil {
		return false
	}
	if fi.Size() == fileInfo.Size {
		return true
	} else {
		return false
	}
}

// 跨域访问配置 [https://blog.csdn.net/yanzisu_congcong/article/details/80552155]
func CrossOrigin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Depth, User-Agent, X-File-Size, X-Requested-With, X-Requested-By, If-Modified-Since, X-File-Name, X-File-Type, Cache-Control, Origin")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Expose-Headers", "Authorization")
}

//
func GetClusterNotPermitMessage(r *http.Request) string {
	var (
		message string
	)
	message = fmt.Sprintf(CONST_MESSAGE_CLUSTER_IP, util.GetClientIp(r))
	return message
}

//
func IsPeer(r *http.Request) bool {
	var (
		ip    string
		peer  string
		bflag bool
	)
	//return true
	ip = util.GetClientIp(r)
	realIp := os.Getenv("GO_FASTDFS_IP")
	if realIp == "" {
		realIp = util.GetPulicIP()
	}
	if ip == "127.0.0.1" || ip == realIp {
		return true
	}
	if util.Contains(ip, AdminIps) {
		return true
	}
	ip = "http://" + ip
	bflag = false
	for _, peer = range peers {
		if strings.HasPrefix(peer, ip) {
			bflag = true
			break
		}
	}
	return bflag
}
