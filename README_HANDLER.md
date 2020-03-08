# 业务功能接口分析

## API通用说明
```
一、统一使用POST请求
二、返回格式统一为json
　　格式如下
    {
	  "status":"ok",
	  "message":"",
	  "data":{}
	}

二、url中的group只有在support_group_manage设置为true才有。
	例如：
	http://10.1.5.9:8080/group/reload
	默认：
	http://10.1.5.9:8080/reload
	说明：url中的group为cfg.json中的group参数值。

```

## 主页
> / 、/group     -> hs.Home

## WEB上传页面
> /upload.html   -> hs.IndexHTML

## 报表页面
> /report        -> hs.ReportHTML

## 上传文件
> /upload        -> hs.Upload
> GET [form-data -> md5 文件标识符, output 输出格式(默认文本下载URL, json 格式输出)] 根据文件标识符查询文件信息, 返回文件下载地址
// {"url":"","md5":"","path":"","domain":"","scene":"","size":0,"mtime":0,"scenes":"","retmsg":"","retcode":0,"src":""}
> POST
// {"url":"http://localhost:8000/group/xx/yy.txt","md5":"","path":"/group/xx/yy.txt","domain":"http://localhost:8000","scene":"default","size":,"mtime":1581909176,"scenes":"default","retmsg":"","retcode":0,"src":"/group/xx/yy.txt"}
// OffSet = -1 , 刚上传文件
``` 文件上传API
http://127.0.0.1:8000/group/upload
参数：
file:上传的文件
scene:场景
output:输出
path:自定义路径
具体请参阅示例代码（用浏览器访问http://127.0.0.1:8080）
```
``` 文件秒传
http://127.0.0.1:8000/group/upload
参数：
md5:文件的摘要
摘要算法要与cfg.json中配置的一样
例子：http://127.0.0.1:8080/upload?md5=430a71f5c5e093a105452819cc48cc9c&output=json
```

## 下载文件
> /group/        ->  hs.Download

## 检测(多个个)文件是否存在, 返回存在的文件信息
> /check_files_exist  -> hs.CheckFilesExist
// [md5s 文件标识符, 多个文件用逗号分隔]
// {"message":"","status":"","data":null}
// {"message":"","status":"","data":[{"name":"012690810016 原始数据.log","rename":"gg","path":"files/vv","md5":"c7458c993c9f913e7da1956d064d58a9","size":729188,"peers":["http://7"],"scene":"default","timeStamp":1581918699,"offset":-1,"Retry":0,"Op":""}]}

## 检测(单个)文件是否存在, 存在返回文件信息
> /check_file_exist   -> hs.CheckFileExist
//  [md5 文件标识 , path 文件路径]
// {"name":"012690810016 原始数据.log","rename":"gg","path":"files/vv","md5":"c7458c993c9f913e7da1956d064d58a9","size":729188,"peers":["http://7"],"scene":"default","timeStamp":1581918699,"offset":-1,"Retry":0,"Op":""}

## 获取文件信息 [md5 文件标识符, path 文件完整路径; 优先使用路径]
> /get_file_info      -> hs.GetFileInfo
// { "data": null, "message": "leveldb: not found", "status": "fail" }
// { "data": { "Op": "", "Retry": 0, "md5": "c7458c993c9f913e7da1956d064d58a9", "name": "012690810016 原始数据.log", "offset": -1, "path": "files/vv", "peers": [ "http://7" ], "rename": "gg", "scene": "default", "size": 729188, "timeStamp": 1581918699 }, "message": "", "status": "ok" }
``` 文件信息
http://127.0.0.1:8000/group/get_file_info
参数：
md5:文件的摘要（md5|sha1） 视配置定
path:文件路径
md5与path二选一
说明：md5或path都是上传文件时返回的信息，要以json方式返回才能看到（参阅浏览器上传）
例子：http://127.0.0.1:8080/get_file_info?md5=430a71f5c5e093a105452819cc48cc9c
```

## 列出目录下文件及目录信息 [dir 显示目录, 为空显示根目录]
> /list_dir      -> hs.ListDir
// { "data": null, "message": "open files/x: The system cannot find the file specified.", "status": "" }
// { "data": [ { "is_dir": true, "md5": "78ecb468e7f5e66ec081be9f3fc7863a", "mtime": 1581907374, "name": "_big", "path": "", "size": 0 }, { "is_dir": true, "md5": "c495b986a2bd8d318c92825ed3563ec9", "mtime": 1581906819, "name": "_tmp", "path": "", "size": 0 }, { "is_dir": true, "md5": "25e702e295075eb9b80be634aa88b301", "mtime": 1581906822, "name": "default", "path": "", "size": 0 }, { "is_dir": true, "md5": "4e424fe354b60dd8ce66496f9270627e", "mtime": 1581918699, "name": "vv", "path": "", "size": 0 }, { "is_dir": true, "md5": "1aeefc7712e9ef50f63ae10d94229468", "mtime": 1581908958, "name": "yy", "path": "", "size": 0 } ], "message": "", "status": "ok" }
``` 文件列表
http://127.0.0.1:8000/group/list_dir
参数：
dir：要查看文件列表的目录名
例子：http://127.0.0.1:8080/list_dir?dir=default
```

## 搜索文件
> /search        -> hs.Search
// { "data": [ { "Op": "", "Retry": 0, "md5": "c7458c993c9f913e7da1956d064d58a9", "name": "012690810016 原始数据.log", "offset": -1, "path": "files/vv", "peers": [ "http://7" ], "rename": "gg", "scene": "default", "size": 729188, "timeStamp": 1581918699 } ], "message": "", "status": "ok" }

## 删除文件
> /delete        -> hs.RemoveFile
// [md5 文件标识符, path 文件完整路径; 优先使用文件标识符]
// {"message":"remove success","status":"ok","data":null}
``` 文件删除
http://127.0.0.1:8000/group/delete
参数：
md5:文件的摘要（md5|sha1） 视配置定
path:文件路径
md5与path二选一
说明：md5或path都是上传文件时返回的信息，要以json方式返回才能看到（参阅浏览器上传）
http://127.0.0.1:8080/delete?md5=430a71f5c5e093a105452819cc48cc9c
```

## 清除空目录
> /remove_empty_dir   -> hs.RemoveEmptyDir

## 获取服务文件数量信息(按日期统计) [echart 操作标识 1 输出图表统计信息, inner 操作标识 1 返回纯数据]
> /stat               -> hs.Stat ## 文件统计信息API
```
http://127.0.0.1:8000/group/stat

```

## 获取应用运行状态及文件状态信息
> /status             -> hs.Status

## 上传文件信息, 异步从集权下载文件, 获取下载地址
> /syncfile_info      -> hs.SyncFileInfo

## 
> /sync               -> hs.Sync

##
> /get_md5s_by_date   -> hs.GetMd5sForWeb

##
> /receive_md5s       -> hs.ReceiveMd5s

## 执行文件修复
> /repair             -> hs.Repair
``` 同步失败修复
http://127.0.0.1:8000/group/repair
参数：
force:是否强行修复(0|1)
例子：http://127.0.0.1:8080/repair?force=1
```

## 执行
> /repair_stat        -> hs.RepairStatWeb
``` 修复统计信息
http://127.0.0.1:8000/group/repair_stat
参数：
date:要修复的日期，格式如：20190725
例子：http://127.0.0.1:8080/repair_stat?date=20190725
```

## 数据迁移
> /repair_fileinfo    -> hs.RepairFileInfo
``` 从文件目录中修复元数据 (性能较差)
http://127.0.0.1:8000/group/repair_fileinfo
需要开启搬迁功能，修改cfg.json配置文件中的 enable_migrate 设为true 
```

## 整理日志和元数据
> /backup             -> hs.BackUp

## 谷歌验证
> /gen_google_code    -> hs.GenGoogleCode
> /gen_google_secret  -> hs.GenGoogleSecret

## 重新加载配置
> /reload             -> hs.Reload
``` 配置管理API
http://127.0.0.1:8000/group/reload

参数：
action: set(修改参数),get获取参数,reload重新加载参数
cfg:json参数　与 action=set配合完成参数设置
```