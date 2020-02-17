package web

import (
	"../en"
	"bytes"
	"errors"
	"github.com/astaxie/beego/httplib"
	_ "github.com/eventials/go-tus"
	"github.com/nfnt/resize"
	slog "github.com/sjqzhang/seelog"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"strings"
	"time"
)

// 检测是否有下载权限 [token , timestamp, code]
func (hs *HttpServer) checkDownloadAuth(w http.ResponseWriter, r *http.Request) (bool, error) {
	var (
		err      error
		ts       int64
		fileInfo *en.FileInfo
		scene    string
		secret   interface{}
		code     string
		ok       bool
	)
	// 定义功能 -> 检测 Token (文件Token = md5sum+timestamp)
	CheckTokenFunc := func(token string, md5sum string, timestamp string) bool {
		if util.MD5(md5sum+timestamp) != token {
			return false
		}
		return true
	}
	// 判断下载是否需要认证, 默认认证不开启, 开启需要设置 认证URL
	if enableDownloadAuth && authUrl != "" && !IsPeer(r) && !hs.s.CheckAuth(w, r) {
		return false, errors.New("auth fail")
	}
	// 判断下载是否需要认证 Token, 默认不认证 Token
	if downloadUseToken && !IsPeer(r) {
		// 表单中获取 Token
		token := r.FormValue("token")
		// 表单中获取时间戳
		timestamp := r.FormValue("timestamp")
		if token == "" || timestamp == "" {
			return false, errors.New("unvalid request")
		}
		// 下载有效时间
		maxTimestamp := time.Now().Add(time.Second * time.Duration(downloadTokenExpire)).Unix()
		minTimestamp := time.Now().Add(-time.Second * time.Duration(downloadTokenExpire)).Unix()
		if ts, err = strconv.ParseInt(timestamp, 10, 64); err != nil {
			// 验证时间戳 timestamp 无效
			return false, errors.New("unvalid timestamp")
		}
		if ts > maxTimestamp || ts < minTimestamp {
			// 失效
			return false, errors.New("timestamp expire")
		}
		//
		fullPath, smallPath := analyseFilePathFromRequest(w, r)
		var pathMd5 string
		if smallPath != "" {
			pathMd5 = util.MD5(smallPath)
		} else {
			pathMd5 = util.MD5(fullPath)
		}
		// 根据文件 MD5值 (Hash值), 获取文件信息
		if fileInfo, err = hs.s.GetFileInfoFromLevelDB(pathMd5); err != nil {
			// 获取到文件信息错误 => TODO
		} else {
			// 认证 Token
			ok := CheckTokenFunc(token, fileInfo.Md5, timestamp)
			if !ok {
				return ok, errors.New("unvalid token")
			}
			return ok, nil
		}
	}
	// 检测是否需要 Google认证
	if enableGoogleAuth && !IsPeer(r) {
		fullPath := r.RequestURI[len(group)+2 : len(r.RequestURI)]
		fullPath = strings.Split(fullPath, "?")[0] // just path
		scene = strings.Split(fullPath, "/")[0]
		code = r.FormValue("code")
		if secret, ok = hs.s.GetSceneMap().GetValue(scene); ok {
			if !hs.s.VerifyGoogleCode(secret.(string), code, int64(downloadTokenExpire/30)) {
				return false, errors.New("invalid google code")
			}
		}
	}
	return true, nil
}

// ---------- ---------- ---------- ---------- ---------- ---------- ---------- ----------

// 下载文件, 根据文件路径, 从集群查找文件并下载, 未找到返回 404
func (hs *HttpServer) downloadNotFound(w http.ResponseWriter, r *http.Request) {
	var (
		err      error
		pathMd5  string
		peer     string
		fileInfo *en.FileInfo
	)
	//
	fullPath, smallPath := analyseFilePathFromRequest(w, r)
	isDownload := true
	if r.FormValue("download") == "" {
		isDownload = defaultDownload
	}
	if r.FormValue("download") == "0" {
		isDownload = false
	}
	if smallPath != "" {
		pathMd5 = util.MD5(smallPath)
	} else {
		pathMd5 = util.MD5(fullPath)
	}
	// 从集权所有节点中, 根据文件路径查询文件
	for _, peer = range peers {
		// 检测文件是否存在, 并获取文件信息
		if fileInfo, err = hs.s.CheckPeerFileExist(peer, pathMd5, fullPath); err != nil {
			slog.Error(err)
			continue
		}
		if fileInfo.Md5 != "" {
			// 查找到文件
			go hs.s.DownloadFromPeer(peer, fileInfo)
			//http.Redirect(w, r, peer+r.RequestURI, 302)
			if isDownload {
				SetDownloadHeader(w, r)
			}
			// 下载文件
			hs.downloadFileToResponse(peer+r.RequestURI, w, r)
			return
		}
	}
	// 文件未找到
	w.WriteHeader(404)
	return
}

// 下载文件, 获取数据返回到响应体
func (hs *HttpServer) downloadFileToResponse(url string, w http.ResponseWriter, r *http.Request) {
	var (
		err  error
		req  *httplib.BeegoHTTPRequest
		resp *http.Response
	)
	req = httplib.Get(url)
	req.SetTimeout(time.Second*20, time.Second*600)
	resp, err = req.DoRequest()
	if err != nil {
		_ = slog.Error(err)
	}
	defer resp.Body.Close()
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		_ = slog.Error(err)
	}
}

// ---------- ---------- ---------- ---------- ---------- ---------- ---------- ----------
// 下载文件 (下载图片) [download, width, height (设置宽高, 表明是图片)]
func (hs *HttpServer) downloadNormalIMGFileByURI(w http.ResponseWriter, r *http.Request) (bool, error) {
	var (
		err       error
		imgWidth  int
		imgHeight int
	)
	isDownload := true
	if r.FormValue("download") == "" {
		isDownload = defaultDownload
	}
	if r.FormValue("download") == "0" {
		isDownload = false
	}
	//
	width := r.FormValue("width")
	height := r.FormValue("height")
	if width != "" {
		imgWidth, err = strconv.Atoi(width)
		if err != nil {
			_ = slog.Error(err)
		}
	}
	if height != "" {
		imgHeight, err = strconv.Atoi(height)
		if err != nil {
			_ = slog.Error(err)
		}
	}
	if isDownload {
		// 执行下载
		SetDownloadHeader(w, r)
	}
	// 分析文件存储路径
	fullPath, _ := analyseFilePathFromRequest(w, r)
	if imgWidth != 0 || imgHeight != 0 {
		// 下载图片
		hs.resizeImage(w, fullPath, uint(imgWidth), uint(imgHeight))
		return true, nil
	}
	// 下载文件
	staticHandler.ServeHTTP(w, r)
	return true, nil
}

// 下载图片文件到, HTTP响应体中, 根据修改图片宽高
func (hs *HttpServer) resizeImage(w http.ResponseWriter, fullPath string, width, height uint) {
	var (
		img     image.Image
		err     error
		imgType string
		file    *os.File
	)
	file, err = os.Open(fullPath)
	if err != nil {
		_ = slog.Error(err)
		return
	}
	img, imgType, err = image.Decode(file)
	if err != nil {
		_ = slog.Error(err)
		return
	}
	_ = file.Close()
	img = resize.Resize(width, height, img, resize.Lanczos3)
	if imgType == "jpg" || imgType == "jpeg" {
		_ = jpeg.Encode(w, img, nil)
	} else if imgType == "png" {
		_ = png.Encode(w, img)
	} else {
		_, _ = file.Seek(0, 0)
		_, _ = io.Copy(w, file)
	}
}

// ---------- ---------- ---------- ---------- ---------- ---------- ---------- ----------
// 下载文件 -> 小文件
func (hs *HttpServer) downloadSmallFileByURI(w http.ResponseWriter, r *http.Request) (bool, error) {
	var (
		err       error
		data      []byte
		imgWidth  int
		imgHeight int
		notFound  bool
	)
	isDownload := true
	if r.FormValue("download") == "" {
		isDownload = defaultDownload
	}
	if r.FormValue("download") == "0" {
		isDownload = false
	}
	//
	width := r.FormValue("width")
	height := r.FormValue("height")
	if width != "" {
		imgWidth, err = strconv.Atoi(width)
		if err != nil {
			_ = slog.Error(err)
		}
	}
	if height != "" {
		imgHeight, err = strconv.Atoi(height)
		if err != nil {
			_ = slog.Error(err)
		}
	}
	// 小文件传输 (文件名、文件读取大小、文件读取偏移)
	// -> 读取获取文件传输相关信息
	data, notFound, err = hs.getSmallFileByURI(w, r)
	_ = notFound
	if data != nil && string(data[0]) == "1" {
		if isDownload {
			SetDownloadHeader(w, r)
		}
		if imgWidth != 0 || imgHeight != 0 {
			//
			hs.resizeImageByBytes(w, data[1:], uint(imgWidth), uint(imgHeight))
			return true, nil
		}
		_, _ = w.Write(data[1:])
		return true, nil
	}
	return false, errors.New("not found")
}

// 下载文件 -> 小文件
func (hs *HttpServer) getSmallFileByURI(w http.ResponseWriter, r *http.Request) ([]byte, bool, error) {
	var (
		err    error
		data   []byte
		offset int64
		length int
		info   os.FileInfo
	)
	fullPath, _ := analyseFilePathFromRequest(w, r)
	//
	if _, offset, length, err = hs.parseSmallFile(r.RequestURI); err != nil {
		return nil, false, err
	}
	if info, err = os.Stat(fullPath); err != nil {
		return nil, false, err
	}
	if info.Size() < offset+int64(length) {
		return nil, true, errors.New("noFound")
	} else {
		// 获取文件信息
		data, err = util.ReadFileByOffSet(fullPath, offset, length)
		if err != nil {
			return nil, false, err
		}
		return data, false, err
	}
}

// 下载文件, 文件分块传输
// 下载文件 -> 小文件 -> filename 解析出, 文件名文件大小, 文件读取偏移量
func (hs *HttpServer) parseSmallFile(filename string) (string, int64, int, error) {
	//
	err := errors.New("unvalid small file")
	// 检测文件名
	if len(filename) < 3 {
		return filename, -1, -1, err
	}
	if strings.Contains(filename, "/") {
		// 文件路径中, 获取文件名相关信息
		filename = filename[strings.LastIndex(filename, "/")+1:]
	}
	//
	pos := strings.Split(filename, ",")
	if len(pos) < 3 {
		return filename, -1, -1, err
	}
	offset, err := strconv.ParseInt(pos[1], 10, 64)
	if err != nil {
		return filename, -1, -1, err
	}
	//
	var length int
	if length, err = strconv.Atoi(pos[2]); err != nil {
		return filename, offset, -1, err
	}
	if length > CONST_SMALL_FILE_SIZE || offset < 0 {
		err = errors.New("invalid fileSize or offset")
		return filename, -1, -1, err
	}
	return pos[0], offset, length, nil
}

// 文件下载, 根据文件信息下载文件数据到 响应体
func (hs *HttpServer) resizeImageByBytes(w http.ResponseWriter, data []byte, width, height uint) {
	var (
		img     image.Image
		err     error
		imgType string
	)
	reader := bytes.NewReader(data)
	img, imgType, err = image.Decode(reader)
	if err != nil {
		_ = slog.Error(err)
		return
	}
	img = resize.Resize(width, height, img, resize.Lanczos3)
	if imgType == "jpg" || imgType == "jpeg" {
		_ = jpeg.Encode(w, img, nil)
	} else if imgType == "png" {
		_ = png.Encode(w, img)
	} else {
		//
		_, _ = w.Write(data)
	}
}
