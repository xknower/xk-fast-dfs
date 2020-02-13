package server

import (
	"../en"
	"../web"
	"errors"
	"fmt"
	slog "github.com/sjqzhang/seelog"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

//
func (server *Service) upload(w http.ResponseWriter, r *http.Request) {
	var (
		err error
		ok  bool
		//		pathname     string
		md5sum       string
		fileName     string
		fileInfo     en.FileInfo
		uploadFile   multipart.File
		uploadHeader *multipart.FileHeader
		scene        string
		output       string
		fileResult   en.FileResult
		data         []byte
		code         string
		secret       interface{}
	)
	output = r.FormValue("output")
	if enableCrossOrigin {
		web.CrossOrigin(w, r)
		if r.Method == http.MethodOptions {
			return
		}
	}

	if authUrl != "" {
		if !server.CheckAuth(w, r) {
			slog.Warn("auth fail", r.Form)
			server.NotPermit(w, r)
			w.Write([]byte("auth fail"))
			return
		}
	}
	if r.Method == http.MethodPost {
		md5sum = r.FormValue("md5")
		fileName = r.FormValue("filename")
		output = r.FormValue("output")
		if readOnly {
			w.Write([]byte("(error) readonly"))
			return
		}
		if enableCustomPath {
			fileInfo.Path = r.FormValue("path")
			fileInfo.Path = strings.Trim(fileInfo.Path, "/")
		}
		scene = r.FormValue("scene")
		code = r.FormValue("code")
		if scene == "" {
			//Just for Compatibility
			scene = r.FormValue("scenes")
		}
		if enableGoogleAuth && scene != "" {
			if secret, ok = server.sceneMap.GetValue(scene); ok {
				if !server.VerifyGoogleCode(secret.(string), code, int64(downloadTokenExpire/30)) {
					server.NotPermit(w, r)
					w.Write([]byte("invalid request,error google code"))
					return
				}
			}
		}
		fileInfo.Md5 = md5sum
		fileInfo.ReName = fileName
		fileInfo.OffSet = -1
		if uploadFile, uploadHeader, err = r.FormFile("file"); err != nil {
			slog.Error(err)
			w.Write([]byte(err.Error()))
			return
		}
		fileInfo.Peers = []string{}
		fileInfo.TimeStamp = time.Now().Unix()
		if scene == "" {
			scene = defaultScene
		}
		if output == "" {
			output = "text"
		}
		if !util.Contains(output, []string{"json", "text"}) {
			w.Write([]byte("output just support json or text"))
			return
		}
		fileInfo.Scene = scene
		if _, err = server.checkScene(scene); err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		if err != nil {
			slog.Error(err)
			http.Redirect(w, r, "/", http.StatusMovedPermanently)
			return
		}
		if _, err = server.saveUploadFile(uploadFile, uploadHeader, &fileInfo, r); err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		if enableDistinctFile {
			if v, _ := server.GetFileInfoFromLevelDB(fileInfo.Md5); v != nil && v.Md5 != "" {
				fileResult = server.BuildFileResult(v, r)
				if renameFile {
					os.Remove(DOCKER_DIR + fileInfo.Path + "/" + fileInfo.ReName)
				} else {
					os.Remove(DOCKER_DIR + fileInfo.Path + "/" + fileInfo.Name)
				}
				if output == "json" {
					if data, err = json.Marshal(fileResult); err != nil {
						slog.Error(err)
						w.Write([]byte(err.Error()))
					}
					w.Write(data)
				} else {
					w.Write([]byte(fileResult.Url))
				}
				return
			}
		}
		if fileInfo.Md5 == "" {
			slog.Warn(" fileInfo.Md5 is null")
			return
		}
		if md5sum != "" && fileInfo.Md5 != md5sum {
			slog.Warn(" fileInfo.Md5 and md5sum !=")
			return
		}
		if !enableDistinctFile {
			// bugfix filecount stat
			fileInfo.Md5 = util.MD5(server.GetFilePathByInfo(&fileInfo, false))
		}
		if enableMergeSmallFile && fileInfo.Size < CONST_SMALL_FILE_SIZE {
			if err = server.saveSmallFile(&fileInfo); err != nil {
				slog.Error(err)
				return
			}
		}
		server.saveFileMd5Log(&fileInfo, CONST_FILE_Md5_FILE_NAME) //maybe slow
		go server.postFileToPeer(&fileInfo)
		if fileInfo.Size <= 0 {
			slog.Error("file size is zero")
			return
		}
		fileResult = server.BuildFileResult(&fileInfo, r)
		if output == "json" {
			if data, err = json.Marshal(fileResult); err != nil {
				slog.Error(err)
				w.Write([]byte(err.Error()))
			}
			w.Write(data)
		} else {
			w.Write([]byte(fileResult.Url))
		}
		return
	} else {
		md5sum = r.FormValue("md5")
		output = r.FormValue("output")
		if md5sum == "" {
			w.Write([]byte("(error) if you want to upload fast md5 is require" +
				",and if you want to upload file,you must use post method  "))
			return
		}
		if v, _ := server.GetFileInfoFromLevelDB(md5sum); v != nil && v.Md5 != "" {
			fileResult = server.BuildFileResult(v, r)
		}
		if output == "json" {
			if data, err = json.Marshal(fileResult); err != nil {
				slog.Error(err)
				w.Write([]byte(err.Error()))
			}
			w.Write(data)
		} else {
			w.Write([]byte(fileResult.Url))
		}
	}
}

//
func (server *Service) NotPermit(w http.ResponseWriter, r *http.Request) {
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
func (server *Service) BuildFileResult(fileInfo *en.FileInfo, r *http.Request) en.FileResult {
	var (
		outname     string
		fileResult  en.FileResult
		p           string
		downloadUrl string
		domain      string
		host        string
	)
	host = strings.Replace(host, "http://", "", -1)
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
	if downloadDomain != "" {
		domain = downloadDomain
	} else {
		domain = fmt.Sprintf("http://%s", host)
	}
	outname = fileInfo.Name
	if fileInfo.ReName != "" {
		outname = fileInfo.ReName
	}
	p = strings.Replace(fileInfo.Path, STORE_DIR_NAME+"/", "", 1)
	if supportGroupManage {
		p = group + "/" + p + "/" + outname
	} else {
		p = p + "/" + outname
	}
	downloadUrl = fmt.Sprintf("http://%s/%s", host, p)
	if downloadDomain != "" {
		downloadUrl = fmt.Sprintf("%s/%s", downloadDomain, p)
	}
	fileResult.Url = downloadUrl
	fileResult.Md5 = fileInfo.Md5
	fileResult.Path = "/" + p
	fileResult.Domain = domain
	fileResult.Scene = fileInfo.Scene
	fileResult.Size = fileInfo.Size
	fileResult.ModTime = fileInfo.TimeStamp
	// Just for Compatibility
	fileResult.Src = fileResult.Path
	fileResult.Scenes = fileInfo.Scene
	return fileResult
}
