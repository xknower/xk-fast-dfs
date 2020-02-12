package conf

// 全局静态配置
const (
	STORE_DIR_NAME  = "files"
	LOG_DIR_NAME    = "log"
	DATA_DIR_NAME   = "data"
	CONF_DIR_NAME   = "conf"
	STATIC_DIR_NAME = "static"
	//
	CONST_STAT_FILE_COUNT_KEY      = "fileCount"
	CONST_BIG_UPLOAD_PATH_SUFFIX   = "/big/upload/"
	CONST_STAT_FILE_TOTAL_SIZE_KEY = "totalSize"
	CONST_Md5_ERROR_FILE_NAME      = "errors.md5"
	CONST_Md5_QUEUE_FILE_NAME      = "queue.md5"
	CONST_FILE_Md5_FILE_NAME       = "files.md5"
	CONST_REMOME_Md5_FILE_NAME     = "removes.md5"
	CONST_SMALL_FILE_SIZE          = 1024 * 1024
	CONST_MESSAGE_CLUSTER_IP       = "Can only be called by the cluster ip or 127.0.0.1 or admin_ips(cfg.json),current ip:%s"
)

// 项目运行目录
var (
	FOLDERS = []string{DirData, DirStore, DirConf, DirStatic}
	//
	DirData = DATA_DIR_NAME
	//
	DirStore = STORE_DIR_NAME
	//
	DirConf = CONF_DIR_NAME
	//
	DirStatic = STATIC_DIR_NAME
)

// 全局变量配置
var (
	DirDocker               = ""
	DirLog                  = LOG_DIR_NAME
	DirLargeName            = "haystack"
	DirLarge                = DirStore + "/haystack"
	CONSTLevelDBFileName    = DirData + "/fileserver.db"
	CONSTLevelDBFileNameLog = DirData + "/log.db"
	CONSTStatFileName       = DirData + "/stat.json"
	CONSTConfFileName       = DirConf + "/cfg.json"
	CONSTSearchFileName     = DirData + "/search.txt"
	CONSTUploadCounterKey   = "__CONST_UPLOAD_COUNTER_KEY__"
	LogConfigStr            = `
<seelog type="asynctimer" asyncinterval="1000" minlevel="trace" maxlevel="error">  
	<outputs formatid="common">  
		<buffered formatid="common" size="1048576" flushperiod="1000">  
			<rollingfile type="size" filename="{DirDocker}log/xk.log" maxsize="104857600" maxrolls="10"/>  
		</buffered>
	</outputs>  	  
	 <formats>
		 <format id="common" format="%Date %Time [%LEV] [%File:%Line] [%Func] %Msg%n" />  
	 </formats>  
</seelog>
`
	LogAccessConfigStr = `
<seelog type="asynctimer" asyncinterval="1000" minlevel="trace" maxlevel="error">  
	<outputs formatid="common">  
		<buffered formatid="common" size="1048576" flushperiod="1000">  
			<rollingfile type="size" filename="{DirDocker}log/access.log" maxsize="104857600" maxrolls="10"/>  
		</buffered>
	</outputs>  	  
	 <formats>
		 <format id="common" format="%Date %Time [%LEV] [%File:%Line] [%Func] %Msg%n" />  
	 </formats>  
</seelog>
`
)
