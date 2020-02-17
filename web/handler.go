package web

import (
	"../en"
	"fmt"
	"github.com/astaxie/beego/httplib"
	mapset "github.com/deckarep/golang-set"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"github.com/sjqzhang/googleAuthenticator"
	slog "github.com/sjqzhang/seelog"
	"io"
	"io/ioutil"
	random "math/rand"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

//
func (hs *HttpServer) Home(w http.ResponseWriter, r *http.Request) {
	// [ ""、/ 、 /group 、/group/ -> (跳转到主页 Index)]
	fmt.Printf(" 请求地址 => %s \n", r.RequestURI)
	if r.RequestURI == "/" || r.RequestURI == "" ||
		r.RequestURI == "/"+group ||
		r.RequestURI == "/"+group+"/" {
		hs.IndexHTML(w, r)
		return
	}
	w.WriteHeader(200)
	return
}

// 上传页面
func (hs *HttpServer) IndexHTML(w http.ResponseWriter, r *http.Request) {
	var uploadUrl = "/upload"
	var uploadBigUrl = CONST_BIG_UPLOAD_PATH_SUFFIX
	// 上传页面
	var uppy = UPPY_HTML
	uppyFileName := STATIC_DIR + "/uppy.html"
	if enableWebUpload {
		if supportGroupManage {
			uploadUrl = fmt.Sprintf("/%s/upload", group)
			uploadBigUrl = fmt.Sprintf("/%s%s", group, CONST_BIG_UPLOAD_PATH_SUFFIX)
		}
		if util.IsExist(uppyFileName) {
			// 检测上传页面是否存在, 存在使用静态资源文件
			if data, err := util.ReadBinFile(uppyFileName); err != nil {
				_ = slog.Error(err)
			} else {
				uppy = string(data)
			}
		} else {
			util.WriteFile(uppyFileName, uppy)
		}
		_, _ = fmt.Fprintf(w, fmt.Sprintf(uppy, uploadUrl, defaultScene, uploadBigUrl))
	} else {
		// 不支持 Web 页面上传文件
		_, _ = w.Write([]byte("web upload deny"))
	}
}

// 报表页面
func (hs *HttpServer) ReportHTML(w http.ResponseWriter, r *http.Request) {
	var (
		result en.JsonResult
	)
	fmt.Printf(" 请求地址 => %s \n", r.RequestURI)
	if !IsPeer(r) {
		_, _ = w.Write([]byte(GetClusterNotPermitMessage(r)))
		return
	}
	result.Status = "ok"
	//
	reportFileName := STATIC_DIR + "/report.html"
	if util.IsExist(reportFileName) {
		if data, err := util.ReadBinFile(reportFileName); err != nil {
			_ = slog.Error(err)
			result.Message = err.Error()
			_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
			return
		} else {
			// 返回页面
			html := string(data)
			if supportGroupManage {
				html = strings.Replace(html, "{group}", "/"+group, 1)
			} else {
				html = strings.Replace(html, "{group}", "", 1)
			}
			_, _ = w.Write([]byte(html))
			return
		}
	} else {
		_, _ = w.Write([]byte(fmt.Sprintf("%s is not found", reportFileName)))
	}
}

// 上传文件 [filename 文件名, path 上传路径 , file 文件内容, output=json]
func (hs *HttpServer) Upload(w http.ResponseWriter, r *http.Request) {
	var (
		err    error
		fpTmp  *os.File
		fpBody *os.File
	)
	if r.Method == http.MethodGet {
		// 上传 fast md5, 上传文件必须使用 POST
		hs.s.Upload(w, r)
		return
	}
	// 临时上传目录
	folder := STORE_DIR + "/_tmp/" + time.Now().Format("20060102")
	_ = os.MkdirAll(folder, 0777)
	fn := folder + "/" + util.GetUUID() // 随机文件名
	defer func() {
		_ = os.Remove(fn)
	}()
	fpTmp, err = os.OpenFile(fn, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		_ = slog.Error(err)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	defer fpTmp.Close()
	// 上传文件保存到本地节点
	if _, err = io.Copy(fpTmp, r.Body); err != nil {
		_ = slog.Error(err)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	fpBody, err = os.Open(fn)
	r.Body = fpBody
	// 上传文件到集群 -> 文件加入到上传队列, 等待处理完毕
	done := make(chan bool, 1)
	hs.s.GetQueueUpload() <- en.WrapReqResp{&w, r, done}
	<-done
}

// 下载文件
func (hs *HttpServer) Download(w http.ResponseWriter, r *http.Request) {
	var (
		err error
		ok  bool
		fi  os.FileInfo
	)

	// 检测是否有下载权限
	if ok, err = hs.checkDownloadAuth(w, r); !ok {
		_ = slog.Error(err)
		hs.s.NotPermit(w, r)
		return
	}
	if enableCrossOrigin {
		// 支持跨域下载文件
		CrossOrigin(w, r)
	}
	// 获取文件路径 (fullPath 文件保存文件路径)
	fullPath, smallPath := analyseFilePathFromRequest(w, r)
	if smallPath == "" {
		if fi, err = os.Stat(fullPath); err != nil {
			// 下载文件
			hs.downloadNotFound(w, r)
			return
		}
		if !showDir && fi.IsDir() {
			// 路径非文件, 是个目录, 展示目录无权限
			_, _ = w.Write([]byte("list dir deny"))
			return
		}
		// 下载文件
		_, _ = hs.downloadNormalIMGFileByURI(w, r)
		return
	}
	if smallPath != "" {
		if ok, err = hs.downloadSmallFileByURI(w, r); !ok {
			hs.downloadNotFound(w, r)
			return
		}
		return
	}
}

// 检测(多个)文件是否存在, 并返回查询到的文件信息 [md5s 文件标识符, 多个文件用逗号分隔]
func (hs *HttpServer) CheckFilesExist(w http.ResponseWriter, r *http.Request) {
	var (
		err       error
		data      []byte
		fileInfo  *en.FileInfo
		fileInfos []*en.FileInfo // 查询文件信息列表
		result    en.JsonResult  // 返回结果
	)
	md5sum := r.FormValue("md5s")
	md5s := strings.Split(md5sum, ",")
	for _, m := range md5s {
		// 数据库中查询文件, 通过文件Hash标识
		if fileInfo, err = hs.s.GetFileInfoFromLevelDB(m); fileInfo != nil {
			if fileInfo.OffSet != -1 {
				if data, err = json.Marshal(fileInfo); err != nil {
					_ = slog.Error(err)
				}
				//w.Write(data)
				//return
				fileInfos = append(fileInfos, fileInfo)
				continue
			}
			// 文件(本地节点)绝对路径
			fPath := DOCKER_DIR + fileInfo.Path + "/" + fileInfo.Name
			if fileInfo.ReName != "" {
				fPath = DOCKER_DIR + fileInfo.Path + "/" + fileInfo.ReName
			}
			if util.IsExist(fPath) {
				if data, err = json.Marshal(fileInfo); err == nil {
					fileInfos = append(fileInfos, fileInfo)
					//w.Write(data)
					//return
					continue
				} else {
					_ = slog.Error(err)
				}
			} else {
				if fileInfo.OffSet == -1 {
					// 检测不到文件, 判断文件已经删除 [when file delete,delete from leveldb]
					_ = hs.s.RemoveKeyFromLevelDB(md5sum, hs.s.GetLdb())
				}
			}
		}
	}
	result.Data = fileInfos
	data, _ = json.Marshal(result)
	_, _ = w.Write(data)
	return
}

// 检测(单个)文件是否存在, 存在返回文件信息 [md5 文件标识 , path 文件(绝对)路径]
func (hs *HttpServer) CheckFileExist(w http.ResponseWriter, r *http.Request) {
	var (
		data     []byte
		err      error
		fileInfo *en.FileInfo
		fi       os.FileInfo
	)
	md5sum := r.FormValue("md5")
	fPath := r.FormValue("path")
	//
	if fileInfo, err = hs.s.GetFileInfoFromLevelDB(md5sum); fileInfo != nil {
		if fileInfo.OffSet != -1 {
			if data, err = json.Marshal(fileInfo); err != nil {
				_ = slog.Error(err)
			}
			_, _ = w.Write(data)
			return
		}
		//
		fPath = DOCKER_DIR + fileInfo.Path + "/" + fileInfo.Name
		if fileInfo.ReName != "" {
			fPath = DOCKER_DIR + fileInfo.Path + "/" + fileInfo.ReName
		}
		if util.IsExist(fPath) {
			if data, err = json.Marshal(fileInfo); err == nil {
				// 返回查询到的文件信息
				_, _ = w.Write(data)
				return
			} else {
				_ = slog.Error(err)
			}
		} else {
			if fileInfo.OffSet == -1 {
				// when file delete,delete from leveldb
				_ = hs.s.RemoveKeyFromLevelDB(md5sum, hs.s.GetLdb())
			}
		}
	} else {
		if fPath != "" {
			fPath = strings.Replace(fPath, "/"+group+"/", STORE_DIR_NAME+"/", 1)
			fPath = strings.Replace(fPath, group+"/", STORE_DIR_NAME+"/", 1)
			if fi, err = os.Stat(fPath); err == nil {
				sum := util.MD5(fPath)
				//if Config().EnableDistinctFile {
				//	sum, err = util.GetFileSumByName(fpath, Config().FileSumArithmetic)
				//	if err != nil {
				//		slog.Error(err)
				//	}
				//}
				fileInfo = &en.FileInfo{
					Path:      path.Dir(fPath),
					Name:      path.Base(fPath),
					Size:      fi.Size(),
					Md5:       sum,
					Peers:     []string{host},
					OffSet:    -1, //very important
					TimeStamp: fi.ModTime().Unix(),
				}
				data, err = json.Marshal(fileInfo)
				_, _ = w.Write(data)
				return
			}
		}
	}
	// 返回空数据
	data, _ = json.Marshal(en.FileInfo{})
	_, _ = w.Write(data)
	return
}

// 获取文件信息 [md5 文件标识符, path 文件完整下载路径]
func (hs *HttpServer) GetFileInfo(w http.ResponseWriter, r *http.Request) {
	var (
		err      error
		fileInfo *en.FileInfo
		result   en.JsonResult
	)
	//
	md5sum := r.FormValue("md5")
	fPath := r.FormValue("path")
	result.Status = "fail"
	if !IsPeer(r) {
		_, _ = w.Write([]byte(GetClusterNotPermitMessage(r)))
		return
	}
	if fPath != "" {
		fPath = strings.Replace(fPath, "/"+group+"/", STORE_DIR_NAME+"/", 1)
		fPath = strings.Replace(fPath, group+"/", STORE_DIR_NAME+"/", 1)
		md5sum = util.MD5(fPath)
	}
	//
	if fileInfo, err = hs.s.GetFileInfoFromLevelDB(md5sum); err != nil {
		_ = slog.Error(err)
		result.Message = err.Error()
		_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
		return
	}
	result.Status = "ok"
	result.Data = fileInfo
	_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
	return
}

// 列出目录下文件及目录信息 [dir 显示目录, 为空显示根目录]
func (hs *HttpServer) ListDir(w http.ResponseWriter, r *http.Request) {
	var (
		err         error
		result      en.JsonResult
		filesInfo   []os.FileInfo
		filesResult []en.FileInfoResult
		tmpDir      string
	)
	if !IsPeer(r) {
		result.Message = GetClusterNotPermitMessage(r)
		_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
		return
	}
	dir := r.FormValue("dir")
	//if dir == "" {
	//	result.Message = "dir can't null"
	//	w.Write([]byte(util.JsonEncodePretty(result)))
	//	return
	//}
	dir = strings.Replace(dir, ".", "", -1)
	if tmpDir, err = os.Readlink(dir); err == nil {
		dir = tmpDir
	}
	// 读取目录, 获取目录列表
	if filesInfo, err = ioutil.ReadDir(DOCKER_DIR + STORE_DIR_NAME + "/" + dir); err != nil {
		_ = slog.Error(err)
		result.Message = err.Error()
		_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
		return
	}
	//
	for _, f := range filesInfo {
		fi := en.FileInfoResult{
			Name:    f.Name(),
			Size:    f.Size(),
			IsDir:   f.IsDir(),
			ModTime: f.ModTime().Unix(),
			Path:    dir,
			Md5:     util.MD5(strings.Replace(STORE_DIR_NAME+"/"+dir+"/"+f.Name(), "//", "/", -1)),
		}
		filesResult = append(filesResult, fi)
	}
	result.Status = "ok"
	result.Data = filesResult
	_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
	return
}

// 根据关键字, 搜索含有该关键字名称的文件 [kw 搜索关键词(匹配文件名中包含该关键词的文件, 为空返回所有文件信息)]
func (hs *HttpServer) Search(w http.ResponseWriter, r *http.Request) {
	var (
		err       error
		result    en.JsonResult
		fileInfos []en.FileInfo // 搜索到的所有文件的文件信息
		md5s      []string      // 搜索到的所有的Hash标识
	)
	// 关键字
	kw := r.FormValue("kw")
	if !IsPeer(r) {
		result.Message = GetClusterNotPermitMessage(r)
		_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
		return
	}
	// 编辑文件信息数据库数据, 根据关键搜索关联的文件信息
	iter := hs.s.GetLdb().NewIterator(nil, nil)
	var count int
	for iter.Next() {
		var fileInfo en.FileInfo
		value := iter.Value()
		if err = json.Unmarshal(value, &fileInfo); err != nil {
			_ = slog.Error(err)
			continue
		}
		// 判断文件名中, 是否包含搜索的关键字
		if strings.Contains(fileInfo.Name, kw) && !util.Contains(fileInfo.Md5, md5s) {
			count = count + 1
			fileInfos = append(fileInfos, fileInfo)
			md5s = append(md5s, fileInfo.Md5)
		}
		if count >= 100 {
			break
		}
	}
	iter.Release()
	err = iter.Error()
	if err != nil {
		_ = slog.Error()
	}
	//fileInfos=hs.s.earchDict(kw) // serch file from map for huge capacity
	result.Status = "ok"
	result.Data = fileInfos
	_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
}

// 移除文件 [md5 文件标识符, path 文件完整路径; 优先使用文件标识符]
func (hs *HttpServer) RemoveFile(w http.ResponseWriter, r *http.Request) {
	var (
		err      error
		result   en.JsonResult
		fileInfo *en.FileInfo
		delUrl   string
	)
	if !IsPeer(r) {
		_, _ = w.Write([]byte(GetClusterNotPermitMessage(r)))
		return
	}
	md5 := r.FormValue("md5")
	fPath := r.FormValue("path")
	inner := r.FormValue("inner") // 操作标识
	result.Status = "fail"
	// 检测权限
	if authUrl != "" && !hs.s.CheckAuth(w, r) {
		hs.s.NotPermit(w, r)
		return
	}
	if fPath != "" && md5 == "" {
		fPath = strings.Replace(fPath, "/"+group+"/", STORE_DIR_NAME+"/", 1)
		fPath = strings.Replace(fPath, group+"/", STORE_DIR_NAME+"/", 1)
		md5 = util.MD5(fPath)
	}
	if len(md5) < 32 {
		result.Message = "md5 unvalid"
		_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
		return
	}
	// 执行异步, 调用集群所有节点执行删除文件接口
	if inner != "1" {
		for _, peer := range peers {
			DelFileFunc := func(peer string, md5 string, fileInfo *en.FileInfo) {
				// 遍历集权节点, 构造删除URL, 对集群所有节点进行删除操作
				delUrl = fmt.Sprintf("%s%s", peer, hs.s.GetRequestURI("delete"))
				req := httplib.Post(delUrl)
				req.Param("md5", md5)
				req.Param("inner", "1")
				req.SetTimeout(time.Second*5, time.Second*10)
				if _, err = req.String(); err != nil {
					_ = slog.Error(err)
				}
			}
			//
			go DelFileFunc(peer, md5, fileInfo)
		}
	}
	// 检测并删除文件
	if fileInfo, err = hs.s.GetFileInfoFromLevelDB(md5); err != nil {
		result.Message = err.Error()
		_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
		return
	}
	if fileInfo.OffSet >= 0 {
		result.Message = "small file delete not support"
		_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
		return
	}
	name := fileInfo.Name
	if fileInfo.ReName != "" {
		name = fileInfo.ReName
	}
	fPath = fileInfo.Path + "/" + name
	if fileInfo.Path != "" && util.FileExists(DOCKER_DIR+fPath) {
		// 保存删除操作日志信息
		hs.s.SaveFileMd5Log(fileInfo, CONST_REMOME_Md5_FILE_NAME)
		// 执行删除文件
		if err = os.Remove(DOCKER_DIR + fPath); err != nil {
			result.Message = err.Error()
			_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
			return
		} else {
			// 删除成功
			result.Message = "remove success"
			result.Status = "ok"
			_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
			return
		}
	}
	result.Message = "fail remove"
	_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
}

// 移除空目录
func (hs *HttpServer) RemoveEmptyDir(w http.ResponseWriter, r *http.Request) {
	var (
		result en.JsonResult
	)
	result.Status = "ok"
	if IsPeer(r) {
		//
		go util.RemoveEmptyDir(DATA_DIR)
		//
		go util.RemoveEmptyDir(STORE_DIR)
		result.Message = "clean job start ..,don't try again!!!"
		_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
	} else {
		result.Message = GetClusterNotPermitMessage(r)
		_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
	}
}

// 根据文件信息, 异步从集群下载文件, 获取文件下载地址 (queue.md5) [fileInfo JSON格式]
func (hs *HttpServer) SyncFileInfo(w http.ResponseWriter, r *http.Request) {
	var (
		err      error
		fileInfo en.FileInfo
		filename string
	)
	if !IsPeer(r) {
		return
	}
	fileInfoStr := r.FormValue("fileInfo")
	//
	if err = json.Unmarshal([]byte(fileInfoStr), &fileInfo); err != nil {
		_, _ = w.Write([]byte(GetClusterNotPermitMessage(r)))
		_ = slog.Error(err)
		return
	}
	if fileInfo.OffSet == -2 {
		// 保存文件信息到数据库, 优化迁移 optimize migrate
		_, _ = hs.s.SaveFileInfoToLevelDB(fileInfo.Md5, &fileInfo, hs.s.GetLdb())
	} else {
		//
		hs.s.SaveFileMd5Log(&fileInfo, CONST_Md5_QUEUE_FILE_NAME)
	}
	// 文件添加到下载队列, 等待(异步, 从集群)下载
	hs.s.AppendToDownloadQueue(&fileInfo)
	// 构造下载地址
	filename = fileInfo.Name
	if fileInfo.ReName != "" {
		filename = fileInfo.ReName
	}
	p := strings.Replace(fileInfo.Path, STORE_DIR+"/", "", 1)
	downloadUrl := fmt.Sprintf("http://%s/%s", r.Host, group+"/"+p+"/"+filename)
	slog.Info("SyncFileInfo: ", downloadUrl)
	_, _ = w.Write([]byte(downloadUrl))
}

// 异步下载文件(files.md5 , errors.md5) [date 日志, force 是否暴力处理, inner 操作标识]
func (hs *HttpServer) Sync(w http.ResponseWriter, r *http.Request) {
	var (
		result en.JsonResult
	)
	if !IsPeer(r) {
		result.Message = "client must be in cluster"
		_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
		return
	}
	result.Status = "fail"
	//
	isForceUpload := false
	date := r.FormValue("date")
	force := r.FormValue("force")
	inner := r.FormValue("inner")
	//
	if force == "1" {
		isForceUpload = true
	}
	if inner != "1" {
		for _, peer := range peers {
			//
			req := httplib.Post(peer + hs.s.GetRequestURI("sync"))
			req.Param("force", force)
			req.Param("inner", "1")
			req.Param("date", date)
			if _, err := req.String(); err != nil {
				_ = slog.Error(err)
			}
		}
	}
	//
	if date == "" {
		result.Message = "require paramete date &force , ?date=20181230"
		_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
		return
	}
	date = strings.Replace(date, ".", "", -1)
	if isForceUpload {
		// files.md5
		go hs.s.CheckFileAndSendToPeer(date, CONST_FILE_Md5_FILE_NAME, isForceUpload)
	} else {
		// errors.md5 检测文件, 并加载到下载处理队列
		go hs.s.CheckFileAndSendToPeer(date, CONST_Md5_ERROR_FILE_NAME, isForceUpload)
	}
	result.Status = "ok"
	result.Message = "job is running"
	_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
}

// 查询日期, files.md5 中所有符合该文件的 文件标识符 [date 日期(2020-217)]
func (hs *HttpServer) GetMd5sForWeb(w http.ResponseWriter, r *http.Request) {
	var (
		err    error
		result mapset.Set
		lines  []string
		md5s   []interface{}
	)
	if !IsPeer(r) {
		w.Write([]byte(GetClusterNotPermitMessage(r)))
		return
	}
	date := r.FormValue("date")
	// files.md5
	if result, err = hs.s.GetMd5sByDate(date, CONST_FILE_Md5_FILE_NAME); err != nil {
		_ = slog.Error(err)
		return
	}
	md5s = result.ToSlice()
	for _, line := range md5s {
		if line != nil && line != "" {
			lines = append(lines, line.(string))
		}
	}
	_, _ = w.Write([]byte(strings.Join(lines, ",")))
}

// 根据文件标识符查询文件, 并添加到 文件上传处理队列 等待处理 [md5s 文件标识符, 多个用逗号隔开]
func (hs *HttpServer) ReceiveMd5s(w http.ResponseWriter, r *http.Request) {
	var (
		err      error
		fileInfo *en.FileInfo
	)
	if !IsPeer(r) {
		_ = slog.Warn(fmt.Sprintf("ReceiveMd5s %s", util.GetClientIp(r)))
		_, _ = w.Write([]byte(GetClusterNotPermitMessage(r)))
		return
	}
	md5str := r.FormValue("md5s")
	md5s := strings.Split(md5str, ",")
	// 定义功能
	AppendFunc := func(md5s []string) {
		for _, m := range md5s {
			if m != "" {
				//
				if fileInfo, err = hs.s.GetFileInfoFromLevelDB(m); err != nil {
					_ = slog.Error(err)
					continue
				}
				//
				hs.s.AppendToQueue(fileInfo)
			}
		}
	}
	go AppendFunc(md5s)
}

// 获取服务状态信息 [echart 操作标识 1 输出图表统计信息, inner 操作标识 1 返回纯数据]
func (hs *HttpServer) Stat(w http.ResponseWriter, r *http.Request) {
	var (
		result   en.JsonResult
		category []string
		barCount []int64
		barSize  []int64
		dataMap  map[string]interface{}
	)
	if !IsPeer(r) {
		result.Message = GetClusterNotPermitMessage(r)
		_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
		return
	}
	// 获取服务状态信息数据
	data := hs.getStat()
	result.Status = "ok"
	result.Data = data
	//
	eChart := r.FormValue("echart")
	if eChart == "1" {
		dataMap = make(map[string]interface{}, 3)
		for _, v := range data {
			barCount = append(barCount, v.FileCount)
			barSize = append(barSize, v.TotalSize)
			category = append(category, v.Date)
		}
		dataMap["category"] = category
		dataMap["barCount"] = barCount
		dataMap["barSize"] = barSize
		result.Data = dataMap
	}
	//
	inner := r.FormValue("inner")
	if inner == "1" {
		_, _ = w.Write([]byte(util.JsonEncodePretty(result.Data)))
	} else {
		_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
	}
}

// 获取应用状态信息
func (hs *HttpServer) Status(w http.ResponseWriter, r *http.Request) {
	var (
		err      error
		status   en.JsonResult
		appDir   string
		diskInfo *disk.UsageStat
		memInfo  *mem.VirtualMemoryStat
	)
	// 系统运行(内存)状态
	memStat := new(runtime.MemStats)
	runtime.ReadMemStats(memStat)
	today := util.GetToDay()
	sts := make(map[string]interface{})
	// 队列状态
	sts["Fs.QueueFromPeers"] = len(hs.s.GetQueueFromPeers())
	sts["Fs.QueueToPeers"] = len(hs.s.GetQueueToPeers())
	sts["Fs.QueueFileLog"] = len(hs.s.GetQueueFileLog())
	//
	for _, k := range []string{CONST_FILE_Md5_FILE_NAME, CONST_Md5_ERROR_FILE_NAME, CONST_Md5_QUEUE_FILE_NAME} {
		k2 := fmt.Sprintf("%s_%s", today, k)
		//
		if v, ok := hs.s.GetSumMap().GetValue(k2); ok {
			sumset := v.(mapset.Set)
			if k == CONST_Md5_QUEUE_FILE_NAME {
				sts["Fs.QueueSetSize"] = sumset.Cardinality()
			}
			if k == CONST_Md5_ERROR_FILE_NAME {
				sts["Fs.ErrorSetSize"] = sumset.Cardinality()
			}
			if k == CONST_FILE_Md5_FILE_NAME {
				sts["Fs.FileSetSize"] = sumset.Cardinality()
			}
		}
	}
	// 配置信息
	sts["Fs.AutoRepair"] = autoRepair
	sts["Fs.QueueUpload"] = len(hs.s.GetQueueUpload())
	sts["Fs.RefreshInterval"] = refreshInterval
	sts["Fs.Peers"] = peers
	sts["Fs.Local"] = hs.s.GetHost()
	sts["Fs.FileStats"] = hs.getStat()
	sts["Fs.ShowDir"] = showDir
	sts["Sys.NumGoroutine"] = runtime.NumGoroutine()
	sts["Sys.NumCpu"] = runtime.NumCPU()
	sts["Sys.Alloc"] = memStat.Alloc
	sts["Sys.TotalAlloc"] = memStat.TotalAlloc
	sts["Sys.HeapAlloc"] = memStat.HeapAlloc
	sts["Sys.Frees"] = memStat.Frees
	sts["Sys.HeapObjects"] = memStat.HeapObjects
	sts["Sys.NumGC"] = memStat.NumGC
	sts["Sys.GCCPUFraction"] = memStat.GCCPUFraction
	sts["Sys.GCSys"] = memStat.GCSys
	//sts["Sys.MemInfo"] = memStat
	appDir, err = filepath.Abs(".")
	if err != nil {
		_ = slog.Error(err)
	}
	diskInfo, err = disk.Usage(appDir)
	if err != nil {
		_ = slog.Error(err)
	}
	sts["Sys.DiskInfo"] = diskInfo
	memInfo, err = mem.VirtualMemory()
	if err != nil {
		_ = slog.Error(err)
	}
	sts["Sys.MemInfo"] = memInfo
	status.Status = "ok"
	status.Data = sts
	_, _ = w.Write([]byte(util.JsonEncodePretty(status)))
}

// 执行修复文件并同步集群数据 [force]
func (hs *HttpServer) Repair(w http.ResponseWriter, r *http.Request) {
	var (
		result      en.JsonResult
		force       string
		forceRepair bool
	)
	//
	if !IsPeer(r) {
		result.Message = GetClusterNotPermitMessage(r)
		_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
		return
	}

	result.Status = "ok"
	force = r.FormValue("force")
	if force == "1" {
		forceRepair = true
	}

	// 自动修复文件并同步集群数据服务
	go hs.s.AutoRepair(forceRepair)
	result.Message = "repair job start..."
	_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
}

// 执行该日期统一文件状态数据, 检测状态文件(stat.json)数据, 并修复 [date, inner]
func (hs *HttpServer) RepairStatWeb(w http.ResponseWriter, r *http.Request) {
	var (
		result en.JsonResult
		date   string
		inner  string
	)
	if !IsPeer(r) {
		result.Message = GetClusterNotPermitMessage(r)
		_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
		return
	}
	//
	date = r.FormValue("date")
	inner = r.FormValue("inner")
	//
	if ok, err := regexp.MatchString("\\d{8}", date); err != nil || !ok {
		result.Message = "invalid date"
		_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
		return
	}
	if date == "" || len(date) != 8 {
		date = util.GetToDay()
	}
	if inner != "1" {
		for _, peer := range peers {
			//
			req := httplib.Post(peer + hs.s.GetRequestURI("repair_stat"))
			req.Param("inner", "1")
			req.Param("date", date)
			if _, err := req.String(); err != nil {
				_ = slog.Error(err)
			}
		}
	}
	// 检测状态文件(stat.json)数据, 并修复
	result.Data = hs.s.RepairStatByDate(date)
	result.Status = "ok"
	_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
}

// 数据迁移
func (hs *HttpServer) RepairFileInfo(w http.ResponseWriter, r *http.Request) {
	var (
		result en.JsonResult
	)
	if !IsPeer(r) {
		_, _ = w.Write([]byte(GetClusterNotPermitMessage(r)))
		return
	}
	//
	if !enableMigrate {
		_, _ = w.Write([]byte("please set enable_migrate=true"))
		return
	}
	//
	//
	go hs.s.RepairFileInfoFromFile()
	result.Status = "ok"
	result.Message = "repair job start,don't try again,very danger "
	_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
}

// 整理日志和元数据, 根据查询文件信息数据, 处理文件日志信息数据和文件元数据 [date, inner]
func (hs *HttpServer) BackUp(w http.ResponseWriter, r *http.Request) {
	var (
		err    error
		result en.JsonResult
	)
	if !IsPeer(r) {
		result.Message = GetClusterNotPermitMessage(r)
		_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
		return
	}
	//
	result.Status = "ok"
	date := r.FormValue("date")
	inner := r.FormValue("inner")
	if date == "" {
		date = util.GetToDay()
	}
	if inner != "1" {
		// 针对集群所有节点处理
		for _, peer := range peers {
			backUp := func(peer string, date string) {
				url := fmt.Sprintf("%s%s", peer, hs.s.GetRequestURI("backup"))
				req := httplib.Post(url)
				req.Param("date", date)
				req.Param("inner", "1")
				req.SetTimeout(time.Second*5, time.Second*600)
				if _, err = req.String(); err != nil {
					slog.Error(err)
				}
			}

			go backUp(peer, date)
		}
	}
	// 执行备份服务
	go hs.s.BackUpMetaDataByDate(date)
	result.Message = "back job start..."
	_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
}

//
func (hs *HttpServer) GenGoogleCode(w http.ResponseWriter, r *http.Request) {
	var (
		err    error
		result en.JsonResult
		secret string
		goauth *googleAuthenticator.GAuth
	)
	if !IsPeer(r) {
		result.Message = GetClusterNotPermitMessage(r)
		_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
		return
	}
	//
	result.Status = "ok"
	result.Message = "ok"
	//
	goauth = googleAuthenticator.NewGAuth()
	secret = r.FormValue("secret")
	if result.Data, err = goauth.GetCode(secret); err != nil {
		result.Message = err.Error()
		_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
		return
	}
	_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
}

//
func (hs *HttpServer) GenGoogleSecret(w http.ResponseWriter, r *http.Request) {
	var (
		result en.JsonResult
	)
	if !IsPeer(r) {
		result.Message = GetClusterNotPermitMessage(r)
		_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
		return
	}
	result.Status = "ok"
	result.Message = "ok"
	//
	GetSeed := func(length int) string {
		seeds := "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
		s := ""
		random.Seed(time.Now().UnixNano())
		for i := 0; i < length; i++ {
			s += string(seeds[random.Intn(32)])
		}
		return s
	}
	//
	result.Data = GetSeed(16)
	_, _ = w.Write([]byte(util.JsonEncodePretty(result)))
}
