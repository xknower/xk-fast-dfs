// 队列操作, 文件信息入队及出队列
package server

import (
	"../en"
	_ "github.com/eventials/go-tus"
	_ "net/http/pprof"
	"time"
)

// 集群文件下载处理队列 -> 文件信息添加到队列 检测文件并加载到处理队列
func (server *Service) appendToDownloadQueue(fileInfo *en.FileInfo) {
	for (len(server.queueFromPeers) + CONST_QUEUE_SIZE/10) > CONST_QUEUE_SIZE {
		time.Sleep(time.Millisecond * 50)
	}
	server.queueFromPeers <- *fileInfo
}

// 集群文件上传处理队列 -> 文件信息添加到队列 检测文件并加载到处理队列
func (server *Service) appendToQueue(fileInfo *en.FileInfo) {
	for (len(server.queueToPeers) + CONST_QUEUE_SIZE/10) > CONST_QUEUE_SIZE {
		time.Sleep(time.Millisecond * 50)
	}
	server.queueToPeers <- *fileInfo
}

// 文件日志处理队列 -> 文件处理信息加入日志队列
func (server *Service) appendToFileMd5LogQueue(fileInfo *en.FileInfo, filename string) {
	for len(server.queueFileLog)+len(server.queueFileLog)/10 > CONST_QUEUE_SIZE {
		time.Sleep(time.Second * 1)
	}
	server.queueFileLog <- &en.FileLog{FileInfo: fileInfo, FileName: filename}
}
