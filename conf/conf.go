package conf

// 默认配置文件
const CONFIG_JSON = `{
	"本主机地址": "本机http地址,默认自动生成(注意端口必须与addr中的端口一致），必段为内网，自动生成不为内网请自行修改，下同",
	"host": "%s",
	"绑定端号": "端口",
	"addr": ":8000",
	"组号": "用于区别不同的集群(上传或下载)与support_group_manage配合使用,带在下载路径中",
	"group": "group",
	"集群": "集群列表,注意为了高可用，IP必须不能是同一个,同一不会自动备份，且不能为127.0.0.1,且必须为内网IP，默认自动生成",
	"peers": ["%s"],
	"默认场景": "默认default",
	"default_scene": "default",
	"是否开启跨站访问": "默认开启",
	"enable_cross_origin": true,
	"是否开启Google认证，实现安全的上传、下载": "默认不开启",
	"enable_google_auth": false,
	"是否启用迁移": "默认不启用",
	"enable_migrate": false,
	"是否支持按组（集群）管理,主要用途是Nginx支持多集群": "默认支持,不支持时路径为http://10.1.5.4:8080/action,支持时为http://10.1.5.4:8080/group(配置中的group参数)/action,action为动作名，如status,delete,sync等",
	"support_group_manage": true,
	"认证url": "当url不为空时生效,注意:普通上传中使用http参数 auth_token 作为认证参数, 在断点续传中通过HTTP头Upload-Metadata中的auth_token作为认证参数,认证流程参考认证架构图",
	"auth_url": "",
	"下载token过期时间": "单位秒",
	"download_token_expire": 600,
	"重试同步失败文件的时间": "单位秒",
	"refresh_interval": 1800,
	"是否自动修复": "在超过1亿文件时出现性能问题，取消此选项，请手动按天同步，请查看FAQ",
	"auto_repair": true,
	"邮件配置": "",
	"mail": {
		"user": "abc@163.com",
		"password": "abc",
		"host": "smtp.163.com:25"
	},
	"s": {
		"服务名": "用于区别不同的主机",
		"name": "server",
		"PeerID": "集群内唯一,请使用0-9的单字符，默认自动生成",
		"peer_id": "%s",
		"场景列表": "当设定后，用户指的场景必项在列表中，默认不做限制(注意：如果想开启场景认功能，格式如下：'场景名:googleauth_secret' 如 default:N7IET373HB2C5M6D ",
		"scenes": [],
		"本机是否只读": "默认可读可写",
		"read_only": false,
		"是否自动重命名": "默认不自动重命名,使用原文件名",
		"rename_file": false,
		"同步单一文件超时时间（单位秒）": "默认为0,程序自动计算，在特殊情况下，自已设定",
		"sync_timeout": 0,
		"下载域名": "用于外网下载文件的域名,不包含http://",
		"download_domain": "",
		"是否支持非日期路径": "默认支持非日期路径,也即支持自定义路径,需要上传文件时指定path",
		"enable_custom_path": true,
		"文件是否去重": "默认去重",
		"enable_distinct_file": true,
		"是否开启断点续传": "默认开启",
		"enable_tus": true,
		"是否合并小文件": "默认不合并,合并可以解决inode不够用的情况（当前对于小于1M文件）进行合并",
		"enable_merge_small_file": false,
		"告警接收URL": "方法post,参数:subject,message",
		"alarm_url": "",
		"告警接收邮件列表": "接收人数组",
		"alarm_receivers": [],
		"文件去重算法md5可能存在冲突，默认md5": "sha1|md5",
		"file_sum_arithmetic": "md5",
		"允许后缀名": "允许可以上传的文件后缀名，如jpg,jpeg,png等。留空允许所有。",
		"extensions": []
	},
	"w": {
		"管理ip列表": "用于管理集的ip白名单,",
		"admin_ips": ["127.0.0.1"],
		"是否支持web上传,方便调试": "默认支持web上传",
		"enable_web_upload": true,
		"下载是否认证": "默认不认证(注意此选项是在auth_url不为空的情况下生效)",
		"enable_download_auth": false,
		"是否显示目录": "默认显示,方便调试用,上线时请关闭",
		"show_dir": true,
		"默认是否下载": "默认下载",
		"default_download": true,
		"下载是否需带token": "真假",
		"download_use_token": false
	}
}
	`

// Web 上传页面
const UPLOAD_UPPY_HTML = `<html>
			  <head>
				<meta charset="utf-8" />
				<title>go-fastdfs</title>
				<style>form { bargin } .form-line { display:block;height: 30px;margin:8px; } #stdUpload {background: #fafafa;border-radius: 10px;width: 745px; }</style>
				<link href="https://transloadit.edgly.net/releases/uppy/v0.30.0/dist/uppy.min.css" rel="stylesheet"></head>

			  <body>
                <div>标准上传(强列建议使用这种方式)</div>
				<div id="stdUpload">
				  <form action="%s" method="post" enctype="multipart/form-data">
					<span class="form-line">文件(file):
					  <input type="file" id="file" name="file" /></span>
					<span class="form-line">场景(scene):
					  <input type="text" id="scene" name="scene" value="%s" /></span>
					<span class="form-line">文件名(filename):
					  <input type="text" id="filename" name="filename" value="" /></span>
					<span class="form-line">输出(output):
					  <input type="text" id="output" name="output" value="json" /></span>
					<span class="form-line">自定义路径(path):
					  <input type="text" id="path" name="path" value="" /></span>
	              <span class="form-line">google认证码(code):
					  <input type="text" id="code" name="code" value="" /></span>
					 <span class="form-line">自定义认证(auth_token):
					  <input type="text" id="auth_token" name="auth_token" value="" /></span>
					<input type="submit" name="submit" value="upload" />
                </form>

				</div>
                 <div>断点续传 (如果文件很大时可以考虑)</div>
				<div>
				  <div id="drag-drop-area"></div>
				  <script src="https://transloadit.edgly.net/releases/uppy/v0.30.0/dist/uppy.min.js"></script>
				  <script>var uppy = Uppy.Core().use(Uppy.Dashboard, {
					  inline: true,
					  target: '#drag-drop-area'
					}).use(Uppy.Tus, {
					  endpoint: '%s'
					})
					uppy.on('complete', (result) => {
					 // console.log(result) console.log('Upload complete! We’ve uploaded these files:', result.successful)
					})
					//uppy.setMeta({ auth_token: '9ee60e59-cb0f-4578-aaba-29b9fc2919ca',callback_url:'http://127.0.0.1/callback' ,filename:'自定义文件名','path':'自定义path',scene:'自定义场景' })//这里是传递上传的认证参数,callback_url参数中 id为文件的ID,info 文转的基本信息json
					uppy.setMeta({ auth_token: '9ee60e59-cb0f-4578-aaba-29b9fc2919ca',callback_url:'http://127.0.0.1/callback'})//自定义参数与普通上传类似（虽然支持自定义，建议不要自定义，海量文件情况下，自定义很可能给自已给埋坑）
                </script>
				</div>
			  </body>
			</html>`
