package conf

import (
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"io/ioutil"
	"os"
	"strings"
	"sync/atomic"
	"unsafe"
)

// 配置文件名
var FileName string

//
var ptr unsafe.Pointer

// JSON 解析
var json = jsoniter.ConfigCompatibleWithStandardLibrary

// 全局配置
type GlobalConfig struct {
	AuthUrl            string       `json:"auth_url"`
	EnableCrossOrigin  bool         `json:"enable_cross_origin"`
	Addr               string       `json:"addr"`
	QueueSize          int          `json:"queue_size"`
	PeerId             string       `json:"peer_id"`
	SupportGroupManage bool         `json:"support_group_manage"`
	ReadTimeout        int          `json:"read_timeout"`
	WriteTimeout       int          `json:"write_timeout"`
	IdleTimeout        int          `json:"idle_timeout"`
	ReadHeaderTimeout  int          `json:"read_header_timeout"`
	EnableWebUpload    bool         `json:"enable_web_upload"`
	server             ServerConfig `json:"server"`
	web                WebConfig    `json:"web"`
}

// 服务端配置
type ServerConfig struct {
}

// HTTP WEB 配置
type WebConfig struct {
}

//
func Global() *GlobalConfig {
	return (*GlobalConfig)(atomic.LoadPointer(&ptr))
}

//
func Server() *ServerConfig {
	return &Global().server
}

//
func Web() *WebConfig {
	return &Global().web
}

// 解析配置文件
func ParseConfig(filePath string) {
	var data []byte
	if filePath == "" {
		// 使用默认配置项目
		data = []byte(strings.TrimSpace(CONFIG_JSON))
	} else {
		// 加载配置文件
		file, err := os.Open(filePath)
		if err != nil {
			panic(fmt.Sprintln("open file path:", filePath, "error:", err))
		}
		defer file.Close()
		//
		FileName = filePath
		data, err = ioutil.ReadAll(file)
		if err != nil {
			panic(fmt.Sprintln("file path:", filePath, " read all error:", err))
		}
	}

	// 加载全局配置
	var c GlobalConfig
	if err := json.Unmarshal(data, &c); err != nil {
		panic(fmt.Sprintln("file path:", filePath, "json unmarshal error:", err))
	}

	//
	log.Info(c)
	atomic.StorePointer(&ptr, unsafe.Pointer(&c))
	log.Info("config parse success")
}
