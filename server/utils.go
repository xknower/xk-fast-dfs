package server

import (
	"../en"
	"errors"
	"fmt"
	"github.com/astaxie/beego/httplib"
	"github.com/sjqzhang/googleAuthenticator"
	slog "github.com/sjqzhang/seelog"
	"github.com/syndtr/goleveldb/leveldb"
	"io"
	"mime/multipart"
	"net/http"
	"net/smtp"
	"os"
	"path"
	"path/filepath"
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
func (server *Service) saveSmallFile(fileInfo *en.FileInfo) error {
	var (
		err      error
		filename string
		fpath    string
		srcFile  *os.File
		desFile  *os.File
		largeDir string
		destPath string
		reName   string
		fileExt  string
	)
	filename = fileInfo.Name
	fileExt = path.Ext(filename)
	if fileInfo.ReName != "" {
		filename = fileInfo.ReName
	}
	fpath = DOCKER_DIR + fileInfo.Path + "/" + filename
	largeDir = LARGE_DIR + "/" + peerId
	if !util.FileExists(largeDir) {
		os.MkdirAll(largeDir, 0775)
	}
	reName = fmt.Sprintf("%d", util.RandInt(100, 300))
	destPath = largeDir + "/" + reName
	server.lockMap.LockKey(destPath)
	defer server.lockMap.UnLockKey(destPath)
	if util.FileExists(fpath) {
		srcFile, err = os.OpenFile(fpath, os.O_CREATE|os.O_RDONLY, 06666)
		if err != nil {
			return err
		}
		defer srcFile.Close()
		desFile, err = os.OpenFile(destPath, os.O_CREATE|os.O_RDWR, 06666)
		if err != nil {
			return err
		}
		defer desFile.Close()
		fileInfo.OffSet, err = desFile.Seek(0, 2)
		if _, err = desFile.Write([]byte("1")); err != nil {
			//first byte set 1
			return err
		}
		fileInfo.OffSet, err = desFile.Seek(0, 2)
		if err != nil {
			return err
		}
		fileInfo.OffSet = fileInfo.OffSet - 1 //minus 1 byte
		fileInfo.Size = fileInfo.Size + 1
		fileInfo.ReName = fmt.Sprintf("%s,%d,%d,%s", reName, fileInfo.OffSet, fileInfo.Size, fileExt)
		if _, err = io.Copy(desFile, srcFile); err != nil {
			return err
		}
		srcFile.Close()
		os.Remove(fpath)
		fileInfo.Path = strings.Replace(largeDir, DOCKER_DIR, "", 1)
	}
	return nil
}

//
func (server *Service) notPermit(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(401)
}

//
func (server *Service) saveUploadFile(file multipart.File, header *multipart.FileHeader, fileInfo *en.FileInfo, r *http.Request) (*en.FileInfo, error) {
	var (
		err     error
		outFile *os.File
		folder  string
		fi      os.FileInfo
	)
	defer file.Close()
	_, fileInfo.Name = filepath.Split(header.Filename)
	// bugfix for ie upload file contain fullpath
	if len(extensions) > 0 && !util.Contains(path.Ext(fileInfo.Name), extensions) {
		return fileInfo, errors.New("(error)file extension mismatch")
	}

	if renameFile {
		fileInfo.ReName = util.MD5(util.GetUUID()) + path.Ext(fileInfo.Name)
	}
	folder = time.Now().Format("20060102/15/04")
	if peerId != "" {
		folder = fmt.Sprintf(folder+"/%s", peerId)
	}
	if fileInfo.Scene != "" {
		folder = fmt.Sprintf(STORE_DIR+"/%s/%s", fileInfo.Scene, folder)
	} else {
		folder = fmt.Sprintf(STORE_DIR+"/%s", folder)
	}
	if fileInfo.Path != "" {
		if strings.HasPrefix(fileInfo.Path, STORE_DIR) {
			folder = fileInfo.Path
		} else {
			folder = STORE_DIR + "/" + fileInfo.Path
		}
	}
	if !util.FileExists(folder) {
		os.MkdirAll(folder, 0775)
	}
	outPath := fmt.Sprintf(folder+"/%s", fileInfo.Name)
	if fileInfo.ReName != "" {
		outPath = fmt.Sprintf(folder+"/%s", fileInfo.ReName)
	}
	if util.FileExists(outPath) && enableDistinctFile {
		for i := 0; i < 10000; i++ {
			outPath = fmt.Sprintf(folder+"/%d_%s", i, filepath.Base(header.Filename))
			fileInfo.Name = fmt.Sprintf("%d_%s", i, header.Filename)
			if !util.FileExists(outPath) {
				break
			}
		}
	}
	slog.Info(fmt.Sprintf("upload: %s", outPath))
	if outFile, err = os.Create(outPath); err != nil {
		return fileInfo, err
	}
	defer outFile.Close()
	if err != nil {
		slog.Error(err)
		return fileInfo, errors.New("(error)fail," + err.Error())
	}
	if _, err = io.Copy(outFile, file); err != nil {
		slog.Error(err)
		return fileInfo, errors.New("(error)fail," + err.Error())
	}
	if fi, err = outFile.Stat(); err != nil {
		slog.Error(err)
	} else {
		fileInfo.Size = fi.Size()
	}
	if fi.Size() != header.Size {
		return fileInfo, errors.New("(error)file uncomplete")
	}
	v := "" // util.GetFileSum(outFile, Config().FileSumArithmetic)
	if enableDistinctFile {
		v = util.GetFileSum(outFile, fileSumArithmetic)
	} else {
		v = util.MD5(server.GetFilePathByInfo(fileInfo, false))
	}
	fileInfo.Md5 = v
	//fileInfo.Path = folder //strings.Replace( folder,DOCKER_DIR,"",1)
	fileInfo.Path = strings.Replace(folder, DOCKER_DIR, "", 1)
	fileInfo.Peers = append(fileInfo.Peers, server.host)
	//fmt.Println("upload",fileInfo)
	return fileInfo, nil
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
func (server *Service) isExistFromLevelDB(key string, db *leveldb.DB) (bool, error) {
	return db.Has([]byte(key), nil)
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
