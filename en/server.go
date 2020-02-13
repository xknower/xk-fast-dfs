package en

// 服务端定义
type Server interface {
	GetServerName() string     // 获取服务名称
	GetGroupRouteName() string // 获取访问路由名称, 未配置使用服务名称
}

// 服务端实现
// ---------- ----------
// name  服务器唯一名称
// group 访问路由-分组(路由名称)
// ---------- ----------
type DefaultServer struct {
	name  string
	group string
}

// 获取服务名称
func (s DefaultServer) GetServerName() string {
	return s.name
}

// 获取访问路由名称, 未配置使用服务名称
func (s DefaultServer) GetGroupRouteName() string {
	if s.group == "" {
		return s.name
	}
	return s.group
}

// 初始化服务端
func NewServer(name, group string) *DefaultServer {
	return &DefaultServer{
		name, group,
	}
}
