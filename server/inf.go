package server

// 获取服务名称
func (server Service) GetServerName() string {
	return server.name
}

// 获取访问路由名称, 未配置使用服务名称
func (server Service) GetGroupRouteName() string {
	if server.group == "" {
		return server.name
	}
	return server.group
}
