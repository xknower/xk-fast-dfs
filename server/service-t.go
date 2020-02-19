package server

import (
	"../en"
	"../web"
	"errors"
	"fmt"
	_ "github.com/eventials/go-tus"
	"github.com/sjqzhang/goutil"
	slog "github.com/sjqzhang/seelog"
	"github.com/syndtr/goleveldb/leveldb"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// 构造请求(下载)域
func (server *Service) getServerURI(r *http.Request) string {
	return fmt.Sprintf("http://%s/", r.Host)
}

// 根据文件路径, 直接读取文件数据并返回 [fPath]
func (server *Service) getMd5File(w http.ResponseWriter, r *http.Request) {
	var (
		err  error
		date string
	)
	if !web.IsPeer(r) {
		return
	}
	fPath := DATA_DIR + "/" + date + "/" + CONST_FILE_Md5_FILE_NAME
	if !util.FileExists(fPath) {
		w.WriteHeader(404)
		return
	}

	//
	var data []byte
	if data, err = ioutil.ReadFile(fPath); err != nil {
		w.WriteHeader(500)
		return
	}
	_, _ = w.Write(data)
}

// 根据文件路径, 获取文件并解析, 获取KEY (内容以键值对按行存储) [date, filename]
func (server *Service) getMd5sMapByDate(date string, filename string) (*goutil.CommonMap, error) {
	var (
		err   error
		fPath string
	)
	result := goutil.NewCommonMap(0)
	if filename == "" {
		fPath = DATA_DIR + "/" + date + "/" + CONST_FILE_Md5_FILE_NAME
	} else {
		fPath = DATA_DIR + "/" + date + "/" + filename
	}
	if !util.FileExists(fPath) {
		return result, errors.New(fmt.Sprintf("fpath %s not found", fPath))
	}

	// 根据路径读取文件数据
	var data []byte
	if data, err = ioutil.ReadFile(fPath); err != nil {
		return result, err
	}

	// 解析数据成 Map
	content := string(data)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		cols := strings.Split(line, "|")
		if len(cols) > 2 {
			if _, err = strconv.ParseInt(cols[1], 10, 64); err != nil {
				continue
			}
			// 获取 KEY
			result.Add(cols[0])
		}
	}
	return result, nil
}

//
func (server *Service) BenchMark(w http.ResponseWriter, r *http.Request) {
	t := time.Now()
	batch := new(leveldb.Batch)

	// 随机生成文件信息, 写入数据库
	for i := 0; i < 100000000; i++ {
		f := en.FileInfo{}
		f.Peers = []string{"http://192.168.0.1", "http://192.168.2.5"}
		f.Path = "20190201/19/02"
		s := strconv.Itoa(i)
		s = util.MD5(s)
		f.Name = s
		f.Md5 = s
		if data, err := json.Marshal(&f); err == nil {
			batch.Put([]byte(s), data)
		}

		//
		if i%10000 == 0 {
			if batch.Len() > 0 {
				// 写入
				_ = server.ldb.Write(batch, nil)
				// batch = new(leveldb.Batch)
				batch.Reset()
			}
			fmt.Println(i, time.Since(t).Seconds())
		}
		//fmt.Println(server.GetFileInfoFromLevelDB(s))
	}

	util.WriteFile("time.txt", time.Since(t).String())
	fmt.Println(time.Since(t).String())
}

//
func (server *Service) registerExit() {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		for s := range c {
			switch s {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				_ = server.ldb.Close()
				slog.Info("Exit", s)
				os.Exit(1)
			}
		}
	}()
}

// 保存 searchMap 内容到文件 (search.txt)
func (server *Service) saveSearchDict() {
	var (
		err        error
		fp         *os.File
		searchDict map[string]interface{}
	)

	server.lockMap.LockKey(CONST_SEARCH_FILE_NAME)
	defer server.lockMap.UnLockKey(CONST_SEARCH_FILE_NAME)

	searchDict = server.searchMap.Get()
	if fp, err = os.OpenFile(CONST_SEARCH_FILE_NAME, os.O_RDWR, 0755); err != nil {
		_ = slog.Error(err)
		return
	}
	defer fp.Close()

	//
	for k, v := range searchDict {
		_, _ = fp.WriteString(fmt.Sprintf("%s\t%s", k, v.(string)))
	}
}

// Notice: performance is poor,just for low capacity,but low memory , if you want to high performance,use searchMap for search,but memory ....
func (server *Service) searchDict(kw string) []en.FileInfo {
	var (
		fileInfos []en.FileInfo
	)
	for dict := range server.searchMap.Iter() {
		if strings.Contains(dict.Val.(string), kw) {
			if fileInfo, _ := server.getFileInfoFromLevelDB(dict.Key); fileInfo != nil {
				fileInfos = append(fileInfos, *fileInfo)
			}
		}
	}
	return fileInfos
}

//
func (server *Service) heartBeat(w http.ResponseWriter, r *http.Request) {
}

func (server *Service) test() {
	TestLockFunc := func() {
		wg := sync.WaitGroup{}
		tt := func(i int, wg *sync.WaitGroup) {
			//if server.lockMap.IsLock("xx") {
			//	return
			//}
			//fmt.Println("timeer len",len(server.lockMap.Get()))
			//time.Sleep(time.Nanosecond*10)
			server.lockMap.LockKey("xx")
			defer server.lockMap.UnLockKey("xx")
			//time.Sleep(time.Nanosecond*1)
			//fmt.Println("xx", i)
			wg.Done()
		}
		go func() {
			for {
				time.Sleep(time.Second * 1)
				fmt.Println("timeer len", len(server.lockMap.Get()), server.lockMap.Get())
			}
		}()
		fmt.Println(len(server.lockMap.Get()))
		for i := 0; i < 10000; i++ {
			wg.Add(1)
			go tt(i, &wg)
		}
		fmt.Println(len(server.lockMap.Get()))
		fmt.Println(len(server.lockMap.Get()))
		server.lockMap.LockKey("abc")
		fmt.Println("lock")
		time.Sleep(time.Second * 5)
		server.lockMap.UnLockKey("abc")
		server.lockMap.LockKey("abc")
		server.lockMap.UnLockKey("abc")
	}
	_ = TestLockFunc

	TestFileFunc := func() {
		var (
			err error
			f   *os.File
		)
		f, err = os.OpenFile("tt", os.O_CREATE|os.O_RDWR, 0777)
		if err != nil {
			fmt.Println(err)
		}
		_, _ = f.WriteAt([]byte("1"), 100)
		_, _ = f.Seek(0, 2)
		_, _ = f.Write([]byte("2"))
		//fmt.Println(f.Seek(0, 2))
		//fmt.Println(f.Seek(3, 2))
		//fmt.Println(f.Seek(3, 0))
		//fmt.Println(f.Seek(3, 1))
		//fmt.Println(f.Seek(3, 0))
		//f.Write([]byte("1"))
	}
	_ = TestFileFunc
	//testFile()
	//testLock()
}
