package server

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// RouterManager 路由管理器
type RouterManager struct {
	server     *Server
	engine     *gin.Engine
	routes     map[string]*RouteInfo
	groups     map[string]*gin.RouterGroup
	middleware []gin.HandlerFunc
}

// RouteInfo 路由信息
type RouteInfo struct {
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	Handler     string            `json:"handler"`
	Middleware  []string          `json:"middleware"`
	Description string            `json:"description"`
	Tags        []string          `json:"tags"`
	Params      map[string]string `json:"params"`
	CreatedAt   time.Time         `json:"created_at"`
}

// NewRouterManager 创建路由管理器
func NewRouterManager(server *Server) *RouterManager {
	return &RouterManager{
		server: server,
		engine: server.engine,
		routes: make(map[string]*RouteInfo),
		groups: make(map[string]*gin.RouterGroup),
	}
}

// SetupDefaultRoutes 设置默认路由
func (rm *RouterManager) SetupDefaultRoutes() {
	// 健康检查路由
	rm.RegisterRoute("GET", "/api/v1/health", rm.healthHandler, "健康检查接口")

	// 服务器信息路由
	rm.RegisterRoute("GET", "/api/v1/info", rm.serverInfoHandler, "服务器信息接口")

	// 路由列表接口
	rm.RegisterRoute("GET", "/api/v1/routes", rm.routesHandler, "路由列表接口")

	// 监控指标路由
	if rm.server.config.EnableMetrics {
		rm.RegisterRoute("GET", "/api/v1/metrics", rm.metricsHandler, "监控指标接口")
	}

	// 性能分析路由
	if rm.server.config.EnablePprof {
		rm.setupPprofRoutes()
	}

	// 静态文件路由
	if rm.server.config.StaticDir != "" {
		rm.setupStaticFileRoutes()
	}
}

// RegisterRoute 注册单个路由
func (rm *RouterManager) RegisterRoute(method, path string, handler gin.HandlerFunc, description string, middleware ...gin.HandlerFunc) {
	// 创建路由信息
	routeKey := fmt.Sprintf("%s:%s", method, path)
	routeInfo := &RouteInfo{
		Method:      method,
		Path:        path,
		Handler:     "custom_handler",
		Description: description,
		CreatedAt:   time.Now(),
		Params:      make(map[string]string),
	}

	// 记录中间件信息
	for _, mw := range middleware {
		routeInfo.Middleware = append(routeInfo.Middleware, fmt.Sprintf("%T", mw))
	}

	// 存储路由信息
	rm.routes[routeKey] = routeInfo

	// 注册到Gin引擎
	handlers := append(middleware, handler)
	switch strings.ToUpper(method) {
	case "GET":
		rm.engine.GET(path, handlers...)
	case "POST":
		rm.engine.POST(path, handlers...)
	case "PUT":
		rm.engine.PUT(path, handlers...)
	case "DELETE":
		rm.engine.DELETE(path, handlers...)
	case "PATCH":
		rm.engine.PATCH(path, handlers...)
	case "HEAD":
		rm.engine.HEAD(path, handlers...)
	case "OPTIONS":
		rm.engine.OPTIONS(path, handlers...)
	default:
		rm.engine.Any(path, handlers...)
	}
}

// CreateGroup 创建路由组
func (rm *RouterManager) CreateGroup(prefix string, middleware ...gin.HandlerFunc) *gin.RouterGroup {
	group := rm.engine.Group(prefix, middleware...)
	rm.groups[prefix] = group
	return group
}

// GetGroup 获取路由组
func (rm *RouterManager) GetGroup(prefix string) *gin.RouterGroup {
	return rm.groups[prefix]
}

// RegisterGroupRoute 在路由组中注册路由
func (rm *RouterManager) RegisterGroupRoute(groupPrefix, method, path string, handler gin.HandlerFunc, description string, middleware ...gin.HandlerFunc) {
	group := rm.GetGroup(groupPrefix)
	if group == nil {
		group = rm.CreateGroup(groupPrefix)
	}

	// 完整路径
	fullPath := groupPrefix + path
	routeKey := fmt.Sprintf("%s:%s", method, fullPath)

	// 创建路由信息
	routeInfo := &RouteInfo{
		Method:      method,
		Path:        fullPath,
		Handler:     "group_handler",
		Description: description,
		CreatedAt:   time.Now(),
		Params:      make(map[string]string),
	}

	// 记录中间件信息
	for _, mw := range middleware {
		routeInfo.Middleware = append(routeInfo.Middleware, fmt.Sprintf("%T", mw))
	}

	// 存储路由信息
	rm.routes[routeKey] = routeInfo

	// 注册到路由组
	handlers := append(middleware, handler)
	switch strings.ToUpper(method) {
	case "GET":
		group.GET(path, handlers...)
	case "POST":
		group.POST(path, handlers...)
	case "PUT":
		group.PUT(path, handlers...)
	case "DELETE":
		group.DELETE(path, handlers...)
	case "PATCH":
		group.PATCH(path, handlers...)
	case "HEAD":
		group.HEAD(path, handlers...)
	case "OPTIONS":
		group.OPTIONS(path, handlers...)
	default:
		group.Any(path, handlers...)
	}
}

// AddGlobalMiddleware 添加全局中间件
func (rm *RouterManager) AddGlobalMiddleware(middleware ...gin.HandlerFunc) {
	rm.middleware = append(rm.middleware, middleware...)
	rm.engine.Use(middleware...)
}

// setupPprofRoutes 设置性能分析路由
func (rm *RouterManager) setupPprofRoutes() {
	pprofGroup := rm.CreateGroup("/debug/pprof")
	
	// 注册pprof路由
	pprofGroup.GET("/", gin.WrapF(pprof.Index))
	pprofGroup.GET("/cmdline", gin.WrapF(pprof.Cmdline))
	pprofGroup.GET("/profile", gin.WrapF(pprof.Profile))
	pprofGroup.POST("/symbol", gin.WrapF(pprof.Symbol))
	pprofGroup.GET("/symbol", gin.WrapF(pprof.Symbol))
	pprofGroup.GET("/trace", gin.WrapF(pprof.Trace))
	pprofGroup.GET("/allocs", gin.WrapH(pprof.Handler("allocs")))
	pprofGroup.GET("/block", gin.WrapH(pprof.Handler("block")))
	pprofGroup.GET("/goroutine", gin.WrapH(pprof.Handler("goroutine")))
	pprofGroup.GET("/heap", gin.WrapH(pprof.Handler("heap")))
	pprofGroup.GET("/mutex", gin.WrapH(pprof.Handler("mutex")))
	pprofGroup.GET("/threadcreate", gin.WrapH(pprof.Handler("threadcreate")))

	// 记录pprof路由信息
	pprofRoutes := []string{
		"/", "/cmdline", "/profile", "/symbol", "/trace",
		"/allocs", "/block", "/goroutine", "/heap", "/mutex", "/threadcreate",
	}

	for _, route := range pprofRoutes {
		routeKey := fmt.Sprintf("GET:/debug/pprof%s", route)
		rm.routes[routeKey] = &RouteInfo{
			Method:      "GET",
			Path:        "/debug/pprof" + route,
			Handler:     "pprof_handler",
			Description: "性能分析接口",
			Tags:        []string{"pprof", "debug"},
			CreatedAt:   time.Now(),
			Params:      make(map[string]string),
		}
	}
}

// setupStaticFileRoutes 设置静态文件路由
func (rm *RouterManager) setupStaticFileRoutes() {
	staticServer := NewStaticFileServer(rm.server.config)
	staticServer.SetupRoutes(rm.engine)

	// 记录静态文件路由信息
	staticPath := rm.server.config.StaticURLPrefix + "/*filepath"
	routeKey := fmt.Sprintf("GET:%s", staticPath)
	rm.routes[routeKey] = &RouteInfo{
		Method:      "GET",
		Path:        staticPath,
		Handler:     "static_file_handler",
		Description: "静态文件服务",
		Tags:        []string{"static", "file"},
		CreatedAt:   time.Now(),
		Params:      map[string]string{"filepath": "文件路径"},
	}
}

// healthHandler 健康检查处理器
func (rm *RouterManager) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
		"name":      rm.server.config.Name,
		"uptime":    time.Since(time.Now()).String(), // 这里应该记录服务启动时间
	})
}

// serverInfoHandler 服务器信息处理器
func (rm *RouterManager) serverInfoHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"server": gin.H{
			"name":    rm.server.config.Name,
			"version": "1.0.0",
			"port":    rm.server.config.Port,
			"host":    rm.server.config.Host,
		},
		"features": gin.H{
			"graceful_shutdown": rm.server.config.EnableGracefulShutdown,
			"hot_restart":       rm.server.config.EnableHotRestart,
			"metrics":           rm.server.config.EnableMetrics,
			"pprof":             rm.server.config.EnablePprof,
			"static_files":      rm.server.config.StaticDir != "",
			"cors":              rm.server.config.EnableCORS,
			"rate_limit":        rm.server.config.EnableRateLimit,
		},
		"routes_count": len(rm.routes),
		"groups_count": len(rm.groups),
	})
}

// routesHandler 路由列表处理器
func (rm *RouterManager) routesHandler(c *gin.Context) {
	// 获取查询参数
	method := c.Query("method")
	tag := c.Query("tag")

	// 过滤路由
	filteredRoutes := make(map[string]*RouteInfo)
	for key, route := range rm.routes {
		// 按方法过滤
		if method != "" && !strings.EqualFold(route.Method, method) {
			continue
		}

		// 按标签过滤
		if tag != "" {
			found := false
			for _, routeTag := range route.Tags {
				if strings.EqualFold(routeTag, tag) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		filteredRoutes[key] = route
	}

	c.JSON(http.StatusOK, gin.H{
		"total":  len(filteredRoutes),
		"routes": filteredRoutes,
		"groups": rm.getGroupInfo(),
	})
}

// metricsHandler 监控指标处理器
func (rm *RouterManager) metricsHandler(c *gin.Context) {
	// 这里应该集成Prometheus或其他监控系统
	// 目前返回基本的路由统计信息
	c.String(http.StatusOK, `# HELP gin_routes_total Total number of registered routes
# TYPE gin_routes_total counter
gin_routes_total %d

# HELP gin_groups_total Total number of route groups
# TYPE gin_groups_total counter
gin_groups_total %d
`,
		len(rm.routes), len(rm.groups))
}

// getGroupInfo 获取路由组信息
func (rm *RouterManager) getGroupInfo() map[string]interface{} {
	groupInfo := make(map[string]interface{})
	for prefix := range rm.groups {
		// 统计该组下的路由数量
		count := 0
		for _, route := range rm.routes {
			if strings.HasPrefix(route.Path, prefix) {
				count++
			}
		}
		groupInfo[prefix] = gin.H{
			"prefix":      prefix,
			"routes_count": count,
		}
	}
	return groupInfo
}

// GetRoutes 获取所有路由信息
func (rm *RouterManager) GetRoutes() map[string]*RouteInfo {
	return rm.routes
}

// GetGroups 获取所有路由组
func (rm *RouterManager) GetGroups() map[string]*gin.RouterGroup {
	return rm.groups
}

// RemoveRoute 移除路由（注意：Gin不支持动态移除路由，这里只是从记录中移除）
func (rm *RouterManager) RemoveRoute(method, path string) {
	routeKey := fmt.Sprintf("%s:%s", method, path)
	delete(rm.routes, routeKey)
}

// ClearRoutes 清空路由记录
func (rm *RouterManager) ClearRoutes() {
	rm.routes = make(map[string]*RouteInfo)
}