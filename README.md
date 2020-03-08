# xk-fast-dfs

```项目描述

 项目保存的基本单位为文件实体, 所有对象以文件形式进行描述存储管理。

 集群, 即是分组, 由多个相同分组归属的节点组成, 一个节点表示一个运行实例, 分组名称表示该节点对于与那个分组(集群)。

 -> 该项目, 是对项目 [go-fatdfs][https://github.com/sjqzhang/go-fastdfs.git][commit: eb666f55 , support big file sence and path, 201912041558]
 -> 的学习、整理 和相关代码(业务)优化。
 -> 本项目是单分组集群应用文件系统

 使用: 配置外网下载域, 搭建内部集群(组), 通过 Nginx 转发到内网各节点, 进行处理

```

```开发描述

  基于原项目的二次开发, 总体目标是, 多分组、高可用的分布式文件系统, 以及基于该系统的分布式用户分布和文件存储检索服务
  
  开发阶段
  > 1、鉴于原项目的架构和注释不友好, 在学习的基础上优化架构进行分包和业务注释
  > 2、多分组工程实现以及分布式用户功能实现
  > 3、先阶段工作描述 <目前第一阶段已经完成和最后处理、第二阶段开发正在规划开发文档、文档管理系统, 小程序端正在开发>

```

## 01 项目功能架构描述

``` 项目架构描述 
xk-fast-dfs 项目架构
│
│  xk.go // 启动文件
│
├─conf // 项目配置加载管理
│
├─en   // 实体定义
│  │  file.go
│  │  http.go
│  │  server.go
│  └─store.go
│
├─server // 服务端
│
├─web    // 对外接口实现
│
├─doc    // 项目文档
│
└─static // 静态资源
```

``` 应用部署架构描述 
xk-fast-dfs 部署架构
│
│  xk.exe // 项目运行文件
│
├─conf
│      cfg.json   // 项目配置文件
│
├─data
│  │  stat.json  // 应用状态信息文件
│  │
│  ├─20200214   // 文件信息
│  │      files.md5
│  │      meta.data
│  │
│  ├─data.db // 文件数据信息数据库
│  │
│  └─log.db  // 文件操作日志信息数据库
│
├─files  // 上传文件保存目录
│
├─log    // 应用日志目录
│      access.log
│      tusd.log
│      xk.log     // HTTP 请求日志
│
└─static         // 应用静态资源目录
```

## 02 项目架构及包结构 说明

``` 架构说明
> 包结构划分 : 通用功能部分(全局配置解析加载 + 全局接口和实体定义 + 全局公共工具实现) + 业务功能模块 + 外部接口模块
> 业务功能模块 (server) : 实体结构定义 + 接口方法实现 + 配置初始化 + 业务模块组件初始化 + 业务模块核心组件 + 业务模块交互组件实现
  -- 业务功能实现
  -- 业务模块核心组件 : 文件上传加载交互模块(HTTP File Server) + 后端服务组件(集群交互、业务状态交互、集群数据管理)
> 外部接口模块 : HTTP服务定义及Handler定义 + 配置初始化 + 对接业务功能模块(调用分析并响应相应数据)
```
``` 文件上传流程
客户端上传文件 (Http请求) 
  > (单台)服务器处理请求, 接收上传数据存储在本地 (弱校验, 直接返回成功文件状态为已上传未同步)
  / 同步文件信息到集群(文件所属集群配置, 例如, 配置3台, 则文件同步到三台响应成功, 为成功) > 返回成功
客户端请求同步集群文件状态 (Http请求) > 主机将本地文件状态信息响应给客户端 (文件状态定义同步和其他主机公告) 
```
``` 文件下载流程
客户下载文件 (Http请求)
  > (单台)服务器查询本地数据(集权文件索引是否存在数据, 本地文件存储系统是否存储由文件数据) > 索引不存在同步索引信息,
  / 同步成功, 文件不存在返回响应 / 文件存在 (返回响应信息, 加载下载连接), 请求下载文件(文件过大,直接响应客户端, 等待(一段时间)下载数据) / 文件下载成功
```
``` 集群数据管理同步流程
> 实时数据同步和定时数据同步, 用户发起请求时直接进入处理队列进行处理, 定时同步数据通过定时接口将处理任务压入队列
> 定时同步数据, 数据索引同步, 原始数据同步 (根据配置进行负载均衡同步, 例如, 集群只有三台主机, 配置为三台主备份, 则每台主机都会存储数据)
> 主机划分 : 集群主机 + 备份主机 (集群主机, 对外(备份主机和客户端)提供服务) (备份主机, 也称为用户主机, 单用户使用缓存存储单用户数据, 并划分单独空间备份整个系统数据)
> 数据配置 : 主存储配置(2 个)、备份配置 (副本 Replication (6 个))
```

### A、包功能描述
>###### doc    项目文档
>###### conf   全局环境配置, 项目启动时, 加载解析环境配置
>###### en     定义公共实体类
>###### server 后台服务功能
>###### web    HTTP服务, 提供对外访问接口
>###### static 静态资源存储目录

#### server 后台服务功能
>###### entity  实体定义
>###### init    服务相关参数变量初始化
>###### inf     对外接口实现, 相关功能提供给其他服务
>###### server  项目组件初始化及启动实现
>###### service 服务组件实现
>###### utils   公共功能实现 (不依赖于状态)

#### 存储解析
>###### [本地数据库 (DB: data + log)]
>###### [集群数据库 (存在在其他节点的数据库)]

#### 队列解析 
>###### [queueUpload    本机上传数据处理队列]
>###### [queueFileLog   本机日志数据处理队列]
>###### [queueToPeers   集群文件上传处理队列]
>###### [queueFromPeers 集群文件下载处理队列]

### B、实体结构说明

#### 配置相关定义
>###### GlobalConfig 全局配置
>###### ServerConfig 服务端配置
>###### WebConfig    Web端配置

#### Server 结构定义
> Server -> Service

#### 业务实体相关定义
>###### FileInfo         文件实体结构
>###### FileLog          文件(操作)日志实体结构
>###### StatDateFileInfo 统计文件状态信息实体结构(相关信息保存)
>###### HookDataStore    存储定义

>###### WrapReqResp  请求处理(HTTP)实体定义 [文件上传队列]

#### WEB 结构定义
>###### HttpServer
>###### HttpHandler

#### HTTP 接口返回结构相关定义
>###### JsonResult   响应结果
>###### HttpError    错误信息结构
>###### StatusCode   错误码
>###### FileResult       文件上传成功, 结果返回文件实体结构定义
>###### FileInfoResult   查询目录文件信息, 结果返回文件实体结构定义

## [03 WEB 接口分析](README_HANDLER.md)

## 04 主要业务流程分析

### A、文件上传业务流程

// 01 文件上传功能 [HTTP文件数据流 -> 集群]
// > service.consumerUpload() [server.queueUpload] > func.upload() -> processUploadFile() - saveSmallFile | postFileToPeer
// -> 加入处理日志队列 server.AppendToFileMd5LogQueue() -> func.saveFileMd5Log() -> func.aveFileInfoToLevelDB

### B、文件下载业务流程

### C、文件查询业务流程 [单文件信息查询、多文件信息查询、文件搜索]

### D、状态查询业务流程

### E、应用启动初始化流程

### F、集群节点间文件管理流程 [文件状态同步, 保证高可用、唯一性]

### G、应用管理业务流程

## 05 第三方服务分析

``` 内部包
> fmt
> flag
> path
> path/filepath

> strings
> strconv
> time
> math/rand
> regexp
> errors
> log

> bytes
> bufio
> io
> io/ioutil
> os
> os/signal

> syscall
> unsafe
> sync
> sync/atomic
> runtime
> runtime/debug

> mime/multipart
> net/http
> net/http/pprof
> net/smtp
> image
> image/jpeg
> image/png
```
``` 外部包
> github.com/astaxie/beego/httplib

> github.com/eventials/go-tus

> github.com/sjqzhang/goutil
> github.com/sjqzhang/seelog
> github.com/sjqzhang/tusd
> github.com/sjqzhang/tusd/filestore
> github.com/sjqzhang/googleAuthenticator
> github.com/sjqzhang/googleAuthenticator

> github.com/syndtr/goleveldb/leveldb
> github.com/syndtr/goleveldb/leveldb/opt
> github.com/syndtr/goleveldb/leveldb/util

> github.com/shirou/gopsutil/disk
> github.com/shirou/gopsutil/mem

> github.com/json-iterator/go
> github.com/deckarep/golang-set
> github.com/radovskyb/watcher
> github.com/nfnt/resize"
```

## 06 服务部署配置

### 服务配置
```xk-fast-dfs.service
[Unit]
Description=fast-dfs service 
Wants=network.target 

[Service]
PIDFile=/var/xkfastdfs/conf/app.pid
Environment="GO_FASTDFS_DIR=/home/xkfastdfs" #/home/xkfastdfs 修改成你的安装路径
ExecStart=/home/xkfastdfs/xkfastdfs $GO_FASTDFS_DIR
ExecReload=/bin/kill -s HUP $MAINPID
ExecStop=/bin/kill -s QUIT $MAINPID
PrivateTmp=true
Restart=always

[Install] 
WantedBy=multi-user.target
```

### nginx
>#### 一、如果不知道文件大小时，需设置 client_max_body_size 0;
>#### 二、如果要开启tus，并使用nginx反向代理, 需要设置proxy_redirect 并且重定向
 (proxy_redirect ~/(\w+)/big/upload/(.*) /$1/big/upload/$2;  #继点续传一定要设置(注意))

#### 配置文件
``` xk-fast-dfs.conf
worker_processes  1;
events {
        worker_connections  1024;
}
http {
        include       mime.types;
        default_type  application/html;
        log_format  main  '$remote_addr - $remote_user [$time_local] "$request" '
                      '$status $body_bytes_sent "$http_referer" '
                      '"$http_user_agent" "$http_x_forwarded_for"';
        access_log  /var/log/nginx/access.log  main;
        error_log  /var/log/nginx/error.log  error;
        sendfile        on;
        keepalive_timeout  65;
		client_max_body_size 0; 
		proxy_redirect ~/big/upload/(.*) /big/upload/$1;  #继点续传一定要设置(注意)
        upstream go-fastdfs {
                server 10.1.14.36:8080;
                server 10.1.14.37:8080;
                ip_hash;     #notice:very important(注意)
        }
        server {
                listen       80;
                server_name  localhost;
                location / {
                    proxy_set_header Host $host; #notice:very important(注意)
                    proxy_set_header X-Real-IP $remote_addr; #notice:very important(注意)
                    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for; #notice:very important(注意)
                    proxy_pass http://go-fastdfs;
                }

        }
}
```
``` xk-fast-dfs-cluster.conf
worker_processes  1;
events {
        worker_connections  1024;
}
http {
        include       mime.types;
        default_type  application/html;
        log_format  main  '$remote_addr - $remote_user [$time_local] "$request" '
                      '$status $body_bytes_sent "$http_referer" '
                      '"$http_user_agent" "$http_x_forwarded_for"';
        access_log  /var/log/nginx/access.log  main;
        error_log  /var/log/nginx/error.log  error;
        sendfile        on;
        keepalive_timeout  65;
        #rewrite_log on;
        client_max_body_size 0;
        proxy_redirect ~/(\w+)/big/upload/(.*) /$1/big/upload/$2;  #继点续传一定要设置(注意)
        upstream gofastdfs-group1 {
                server 10.1.51.70:8082;
                server 10.1.14.37:8080;
                ip_hash;     #notice:very important(注意)
        }
		upstream gofastdfs-group2 {
		        server 10.1.51.70:8083;
                server 10.1.14.36:8083;
                ip_hash;     #notice:very important(注意)
        }
        server {
                listen       8000;
                server_name  localhost;
                location /group1 { #以下header要设置
                    proxy_set_header Host $host;
                    proxy_set_header X-Real-IP $remote_addr;
                    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for; 
                    proxy_pass http://gofastdfs-group1;
                }
                location /group2 {#以下header要设置
                    proxy_set_header Host $host; 
                    proxy_set_header X-Real-IP $remote_addr; 
                    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for; 
                    proxy_pass http://gofastdfs-group2;
                }

        }
}
```
```xk-fast-dfs-big-cluster.conf
worker_processes  auto;
events {
        worker_connections  1024;
}
http {
        include       mime.types;
        default_type  application/html;
        log_format  main  '$remote_addr - $remote_user [$time_local] "$request" '
                      '$status $body_bytes_sent "$http_referer" '
                      '"$http_user_agent" "$http_x_forwarded_for"';
        access_log  /var/log/nginx/access.log  main;
        error_log  /var/log/nginx/error.log  error;
        sendfile        on;
        keepalive_timeout  65;
        rewrite_log on;
        client_max_body_size 0;
        proxy_redirect ~/(\w+)/big/upload/(.*) /$1/big/upload/$2;  #继点续传一定要设置(注意)
        #以下是一下横向扩展的配置，当前统一大集群容量不够时，只要增加一个小集群，也就是增加一个
        #upstream ,一个小群集内按业务需求设定副本数，也就是机器数。
        upstream gofastdfs-group1 {
                server 10.1.51.70:8082;
                #server 10.1.14.37:8082;
                ip_hash;     #notice:very important(注意)
        }
	upstream gofastdfs-group2 {
		server 10.1.51.70:8083;
                #server 10.1.14.36:8083;
                ip_hash;     #notice:very important(注意)
        }
       
        server {
                listen       8001;
                server_name  localhost;

	
		if ( $request_uri ~ /godfs/group ) {
                    # 注意group会随组的前缀改变而改变
		    rewrite ^/godfs/(.*)$ /$1 last;
                }
                location ~ /group(\d) { 
                    #统一在url前增加godfs,以便统一出入口。
                    proxy_set_header Host $host;
                    proxy_set_header X-Real-IP $remote_addr;
                    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for; 
                    proxy_pass http://gofastdfs-group$1;
                }
                location ~ /godfs/upload { 
                    #这是一个横向扩展配置，前期可能只有一个集群group1,当group1满后，只需将上传指向group2,
                    #也就是将rewrite , proxy_pass 中的group1改为group2即可。
                    proxy_set_header Host $host;
                    proxy_set_header X-Real-IP $remote_addr;
                    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for; 
                    rewrite ^/godfs/upload /group1/upload break;
                    proxy_pass http://gofastdfs-group1;
                }
                location ~ /godfs/big/upload { 
                    #以上上类似。
                    proxy_set_header Host $host;
                    proxy_set_header X-Real-IP $remote_addr;
                    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for; 
                    rewrite ^/godfs/upload /group1/big/upload break;
                    proxy_pass http://gofastdfs-group1;
                }

        }
}
```