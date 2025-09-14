package chi

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// App 企业级应用框架
type App struct {
	config     *Config
	engine     *gin.Engine
	server     *http.Server
	middleware []gin.HandlerFunc
	
	// 组件管理
	logger     Logger
	metrics    *Metrics
	health     *HealthChecker
	wsHub      *WebSocketHub
	
	// 高级功能
	pluginManager    *PluginManager
	cacheManager     *CacheManager
	tracer          Tracer
	governanceManager *GovernanceManager
	dynamicConfig    *DynamicConfigManager
	featureFlags     *FeatureFlagManager
	eventBus         *EventBus
	
	// 生命周期钩子
	beforeStart []func() error
	afterStart  []func() error
	beforeStop  []func() error
	afterStop   []func() error
	
	// 状态
	started   bool
	startTime time.Time
}

// New 创建新的企业级应用
func New(opts ...Option) *App {
	config := DefaultConfig()
	
	// 应用选项
	for _, opt := range opts {
		opt(config)
	}
	
	// 设置Gin模式
	gin.SetMode(config.Mode)
	
	// 创建Gin引擎
	engine := gin.New()
	
	app := &App{
		config:  config,
		engine:  engine,
		metrics: NewMetrics(),
		health:  NewHealthChecker(config.Name, config.Version, config.Environment),
		wsHub:   NewWebSocketHub(),
		
		// 高级功能初始化
		cacheManager:     NewCacheManager(),
		tracer:          NewMemoryTracer(DefaultTracerConfig()),
		governanceManager: NewGovernanceManager(),
		dynamicConfig:    NewDynamicConfigManager(),
		eventBus:         NewEventBus(),
	}
	
	// 初始化插件管理器
	app.pluginManager = NewPluginManager(app)
	
	// 初始化功能开关管理器
	app.featureFlags = NewFeatureFlagManager(app.dynamicConfig)
	
	// 设置事件总线中间件
	if app.logger != nil {
		app.eventBus.Use(LoggingEventMiddleware(app.logger))
	}
	if app.metrics != nil {
		app.eventBus.Use(MetricsEventMiddleware(app.metrics))
	}
	
	// 设置默认中间件
	app.setupDefaultMiddleware()
	
	// 设置默认路由
	app.setupDefaultRoutes()
	
	return app
}

// setupDefaultMiddleware 设置默认中间件
func (a *App) setupDefaultMiddleware() {
	if a.config.EnableRecovery {
		a.engine.Use(gin.Recovery())
	}
	
	if a.config.EnableCORS {
		a.engine.Use(CORSMiddleware())
	}
	
	if a.config.EnableRequestID {
		a.engine.Use(RequestIDMiddleware())
	}
	
	if a.config.EnableMetrics {
		a.engine.Use(MetricsMiddleware(a.metrics))
	}
	
	if a.config.EnableSecurity {
		a.engine.Use(SecurityMiddleware(*DefaultSecurityConfig()))
	}
	
	// 添加自定义中间件
	for _, middleware := range a.middleware {
		a.engine.Use(middleware)
	}
}

// setupDefaultRoutes 设置默认路由
func (a *App) setupDefaultRoutes() {
	// 健康检查
	a.engine.GET("/health", func(c *gin.Context) {
		health := a.health.Check()
		statusCode := 200
		if health.Status != "healthy" {
			statusCode = 503
		}
		c.JSON(statusCode, health)
	})
	
	// 就绪检查
	a.engine.GET("/ready", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ready",
			"app":    a.config.Name,
			"version": a.config.Version,
		})
	})
	
	// 指标端点
	if a.config.EnableMetrics {
		a.engine.GET("/metrics", func(c *gin.Context) {
			c.JSON(200, a.metrics.GetMetrics())
		})
		
		// 高级指标端点
		a.engine.GET("/metrics/tracing", func(c *gin.Context) {
			if memTracer, ok := a.tracer.(*MemoryTracer); ok {
				c.JSON(200, memTracer.GetTracingStats())
			} else {
				c.JSON(200, gin.H{"message": "tracing stats not available"})
			}
		})
		
		// 缓存统计端点
		a.engine.GET("/metrics/cache", func(c *gin.Context) {
			// TODO: 实现缓存统计获取
			c.JSON(200, gin.H{"message": "cache stats"})
		})
	}
	
	// 应用信息
	a.engine.GET("/info", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"name":        a.config.Name,
			"version":     a.config.Version,
			"environment": a.config.Environment,
			"build_time":  a.config.BuildTime,
			"git_commit":  a.config.GitCommit,
		})
	})
	
	// 管理端点组
	admin := a.engine.Group("/admin")
	{
		// 插件管理
		admin.GET("/plugins", func(c *gin.Context) {
			plugins := a.pluginManager.ListPlugins()
			c.JSON(200, gin.H{"plugins": plugins})
		})
		
		admin.POST("/plugins/:name/start", func(c *gin.Context) {
			name := c.Param("name")
			if err := a.pluginManager.StartPlugin(name); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"message": "plugin started"})
		})
		
		admin.POST("/plugins/:name/stop", func(c *gin.Context) {
			name := c.Param("name")
			if err := a.pluginManager.StopPlugin(name); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"message": "plugin stopped"})
		})
		
		// 缓存管理
		admin.GET("/cache/stats", func(c *gin.Context) {
			provider := a.cacheManager.GetProvider("")
			if statsProvider, ok := provider.(*StatsCacheProvider); ok {
				stats := statsProvider.GetStats()
				c.JSON(200, stats)
			} else {
				c.JSON(200, gin.H{"message": "stats not available"})
			}
		})
		
		admin.POST("/cache/clear", func(c *gin.Context) {
			provider := a.cacheManager.GetProvider("")
			if err := provider.Clear(c.Request.Context()); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"message": "cache cleared"})
		})
		
		// 治理管理
		admin.GET("/governance", GovernanceDashboardHandler(a.governanceManager))
		admin.GET("/governance/:service", ConfigurationAPI(a.governanceManager))
		admin.PUT("/governance/:service", ConfigurationAPI(a.governanceManager))
		admin.POST("/governance/:service", ConfigurationAPI(a.governanceManager))
		
		// 链路追踪管理
		admin.GET("/tracing/stats", func(c *gin.Context) {
			if memTracer, ok := a.tracer.(*MemoryTracer); ok {
				stats := memTracer.GetTracingStats()
				c.JSON(200, stats)
			} else {
				c.JSON(200, gin.H{"message": "tracing stats not available"})
			}
		})
		
		admin.GET("/tracing/traces", func(c *gin.Context) {
			if memTracer, ok := a.tracer.(*MemoryTracer); ok {
				traces := memTracer.GetAllTraces()
				c.JSON(200, traces)
			} else {
				c.JSON(200, gin.H{"message": "traces not available"})
			}
		})
		
		admin.GET("/tracing/trace/:id", func(c *gin.Context) {
			traceID := c.Param("id")
			if memTracer, ok := a.tracer.(*MemoryTracer); ok {
				trace := memTracer.GetTrace(traceID)
				if trace == nil {
					c.JSON(404, gin.H{"error": "trace not found"})
					return
				}
				c.JSON(200, trace)
			} else {
				c.JSON(200, gin.H{"message": "tracing not available"})
			}
		})
		
		// 动态配置管理
		admin.GET("/config", ConfigAPI(a.dynamicConfig))
		admin.POST("/config", ConfigAPI(a.dynamicConfig))
		admin.PUT("/config", ConfigAPI(a.dynamicConfig))
		admin.DELETE("/config", ConfigAPI(a.dynamicConfig))
		
		// 功能开关管理
		admin.GET("/feature-flags", func(c *gin.Context) {
			flags := a.featureFlags.GetAllFlags()
			c.JSON(200, gin.H{"feature_flags": flags})
		})
		
		admin.POST("/feature-flags", func(c *gin.Context) {
			var flag FeatureFlag
			if err := c.ShouldBindJSON(&flag); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}
			
			a.featureFlags.SetFlag(&flag)
			c.JSON(200, gin.H{"message": "feature flag updated"})
		})
		
		admin.GET("/feature-flags/:name/check", func(c *gin.Context) {
			flagName := c.Param("name")
			context := map[string]interface{}{
				"user_id": c.Query("user_id"),
				"ip":      c.ClientIP(),
			}
			
			enabled := a.featureFlags.IsEnabled(flagName, context)
			c.JSON(200, gin.H{
				"flag":    flagName,
				"enabled": enabled,
				"context": context,
			})
		})
		
		// 事件系统管理
		admin.GET("/events", func(c *gin.Context) {
			filters := EventFilters{
				EventType: c.Query("type"),
				Source:    c.Query("source"),
				Limit:     10,
			}
			
			if limitStr := c.Query("limit"); limitStr != "" {
				if limit, err := time.ParseDuration(limitStr); err == nil {
					filters.Limit = int(limit)
				}
			}
			
			events, err := a.eventBus.eventStore.List(filters)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			
			c.JSON(200, gin.H{"events": events})
		})
		
		admin.GET("/events/stats", func(c *gin.Context) {
			stats := a.eventBus.Stats()
			c.JSON(200, stats)
		})
		
		admin.POST("/events", func(c *gin.Context) {
			var request struct {
				Type   string      `json:"type" binding:"required"`
				Data   interface{} `json:"data"`
				Source string      `json:"source"`
			}
			
			if err := c.ShouldBindJSON(&request); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}
			
			source := request.Source
			if source == "" {
				source = "admin_api"
			}
			
			event := NewEvent(request.Type, source, request.Data)
			if err := a.eventBus.Publish(c.Request.Context(), event); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			
			c.JSON(200, gin.H{"message": "event published", "event_id": event.ID()})
		})
		
		// 系统状态
		admin.GET("/system", func(c *gin.Context) {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			
			c.JSON(200, gin.H{
				"app":        a.config.Name,
				"version":    a.config.Version,
				"uptime":     time.Since(a.startTime).String(),
				"goroutines": runtime.NumGoroutine(),
				"memory": gin.H{
					"alloc":       m.Alloc,
					"total_alloc": m.TotalAlloc,
					"sys":         m.Sys,
					"num_gc":      m.NumGC,
				},
			})
		})
	}
	
	// 404处理
	a.engine.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{
			"error":   "Route not found",
			"path":    c.Request.URL.Path,
			"method":  c.Request.Method,
		})
	})
	
	// 405处理
	a.engine.NoMethod(func(c *gin.Context) {
		c.JSON(405, gin.H{
			"error":  "Method not allowed",
			"path":   c.Request.URL.Path,
			"method": c.Request.Method,
		})
	})
}

// Use 添加中间件
func (a *App) Use(middleware ...gin.HandlerFunc) *App {
	a.middleware = append(a.middleware, middleware...)
	if a.started {
		a.engine.Use(middleware...)
	}
	return a
}

// GET 注册GET路由
func (a *App) GET(path string, handlers ...gin.HandlerFunc) *App {
	a.engine.GET(path, handlers...)
	return a
}

// POST 注册POST路由
func (a *App) POST(path string, handlers ...gin.HandlerFunc) *App {
	a.engine.POST(path, handlers...)
	return a
}

// PUT 注册PUT路由
func (a *App) PUT(path string, handlers ...gin.HandlerFunc) *App {
	a.engine.PUT(path, handlers...)
	return a
}

// DELETE 注册DELETE路由
func (a *App) DELETE(path string, handlers ...gin.HandlerFunc) *App {
	a.engine.DELETE(path, handlers...)
	return a
}

// PATCH 注册PATCH路由
func (a *App) PATCH(path string, handlers ...gin.HandlerFunc) *App {
	a.engine.PATCH(path, handlers...)
	return a
}

// HEAD 注册HEAD路由
func (a *App) HEAD(path string, handlers ...gin.HandlerFunc) *App {
	a.engine.HEAD(path, handlers...)
	return a
}

// OPTIONS 注册OPTIONS路由
func (a *App) OPTIONS(path string, handlers ...gin.HandlerFunc) *App {
	a.engine.OPTIONS(path, handlers...)
	return a
}

// Group 创建路由组
func (a *App) Group(path string, handlers ...gin.HandlerFunc) *gin.RouterGroup {
	return a.engine.Group(path, handlers...)
}

// WebSocket 注册WebSocket路由
func (a *App) WebSocket(path string, handler WebSocketHandler) *App {
	a.engine.GET(path, WebSocketUpgradeHandler(a.config.WebSocket, handler))
	return a
}

// Static 提供静态文件服务
func (a *App) Static(relativePath, root string) *App {
	a.engine.Static(relativePath, root)
	return a
}

// StaticFile 提供单个静态文件
func (a *App) StaticFile(relativePath, filepath string) *App {
	a.engine.StaticFile(relativePath, filepath)
	return a
}

// Engine 获取底层Gin引擎
func (a *App) Engine() *gin.Engine {
	return a.engine
}

// Config 获取配置
func (a *App) Config() *Config {
	return a.config
}

// Metrics 获取指标
func (a *App) Metrics() *Metrics {
	return a.metrics
}

// Health 获取健康检查器
func (a *App) Health() *HealthChecker {
	return a.health
}

// WSHub 获取WebSocket Hub
func (a *App) WSHub() *WebSocketHub {
	return a.wsHub
}

// PluginManager 获取插件管理器
func (a *App) PluginManager() *PluginManager {
	return a.pluginManager
}

// CacheManager 获取缓存管理器
func (a *App) CacheManager() *CacheManager {
	return a.cacheManager
}

// Tracer 获取链路追踪器
func (a *App) Tracer() Tracer {
	return a.tracer
}

// GovernanceManager 获取治理管理器
func (a *App) GovernanceManager() *GovernanceManager {
	return a.governanceManager
}

// DynamicConfig 获取动态配置管理器
func (a *App) DynamicConfig() *DynamicConfigManager {
	return a.dynamicConfig
}

// FeatureFlags 获取功能开关管理器
func (a *App) FeatureFlags() *FeatureFlagManager {
	return a.featureFlags
}

// EventBus 获取事件总线
func (a *App) EventBus() *EventBus {
	return a.eventBus
}

// BeforeStart 注册启动前钩子
func (a *App) BeforeStart(fn func() error) *App {
	a.beforeStart = append(a.beforeStart, fn)
	return a
}

// AfterStart 注册启动后钩子
func (a *App) AfterStart(fn func() error) *App {
	a.afterStart = append(a.afterStart, fn)
	return a
}

// BeforeStop 注册停止前钩子
func (a *App) BeforeStop(fn func() error) *App {
	a.beforeStop = append(a.beforeStop, fn)
	return a
}

// AfterStop 注册停止后钩子
func (a *App) AfterStop(fn func() error) *App {
	a.afterStop = append(a.afterStop, fn)
	return a
}

// Start 启动应用
func (a *App) Start(addr string) error {
	// 执行启动前钩子
	for _, fn := range a.beforeStart {
		if err := fn(); err != nil {
			return fmt.Errorf("before start hook failed: %w", err)
		}
	}
	
	// 启动WebSocket Hub
	go a.wsHub.Run()
	
	// 创建HTTP服务器
	a.server = &http.Server{
		Addr:         addr,
		Handler:      a.engine,
		ReadTimeout:  a.config.ReadTimeout,
		WriteTimeout: a.config.WriteTimeout,
		IdleTimeout:  a.config.IdleTimeout,
	}
	
	a.started = true
	a.startTime = time.Now()
	
	// 执行启动后钩子
	for _, fn := range a.afterStart {
		if err := fn(); err != nil {
			return fmt.Errorf("after start hook failed: %w", err)
		}
	}
	
	fmt.Printf("🚀 %s v%s is starting on %s\n", a.config.Name, a.config.Version, addr)
	fmt.Printf("📊 Metrics: http://localhost%s/metrics\n", addr)
	fmt.Printf("🏥 Health: http://localhost%s/health\n", addr)
	fmt.Printf("ℹ️  Info: http://localhost%s/info\n", addr)
	
	// 发布应用启动事件
	if a.eventBus != nil {
		startEvent := NewEvent(EventTypeAppStarted, "app", map[string]interface{}{
			"name":    a.config.Name,
			"version": a.config.Version,
			"address": addr,
		})
		a.eventBus.Publish(context.Background(), startEvent)
	}
	
	return a.server.ListenAndServe()
}

// StartTLS 启动HTTPS服务
func (a *App) StartTLS(addr, certFile, keyFile string) error {
	// 执行启动前钩子
	for _, fn := range a.beforeStart {
		if err := fn(); err != nil {
			return fmt.Errorf("before start hook failed: %w", err)
		}
	}
	
	// 启动WebSocket Hub
	go a.wsHub.Run()
	
	// 创建HTTP服务器
	a.server = &http.Server{
		Addr:         addr,
		Handler:      a.engine,
		ReadTimeout:  a.config.ReadTimeout,
		WriteTimeout: a.config.WriteTimeout,
		IdleTimeout:  a.config.IdleTimeout,
	}
	
	a.started = true
	a.startTime = time.Now()
	
	// 执行启动后钩子
	for _, fn := range a.afterStart {
		if err := fn(); err != nil {
			return fmt.Errorf("after start hook failed: %w", err)
		}
	}
	
	fmt.Printf("🔒 %s v%s is starting on %s (HTTPS)\n", a.config.Name, a.config.Version, addr)
	
	return a.server.ListenAndServeTLS(certFile, keyFile)
}

// Shutdown 优雅关闭应用
func (a *App) Shutdown(ctx context.Context) error {
	if a.server == nil {
		return nil
	}
	
	// 执行停止前钩子
	for _, fn := range a.beforeStop {
		if err := fn(); err != nil {
			fmt.Printf("before stop hook failed: %v\n", err)
		}
	}
	
	// 关闭HTTP服务器
	err := a.server.Shutdown(ctx)
	
	// 执行停止后钩子
	for _, fn := range a.afterStop {
		if err := fn(); err != nil {
			fmt.Printf("after stop hook failed: %v\n", err)
		}
	}
	
	a.started = false
	fmt.Println("✅ Application shutdown completed")
	
	// 发布应用停止事件
	if a.eventBus != nil {
		stopEvent := NewEvent(EventTypeAppStopped, "app", map[string]interface{}{
			"name":    a.config.Name,
			"version": a.config.Version,
		})
		a.eventBus.Publish(context.Background(), stopEvent)
	}
	
	return err
}

// Run 运行应用（自动处理信号）
func (a *App) Run(addr string) error {
	// 创建信号通道
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	
	// 启动服务器
	go func() {
		if err := a.Start(addr); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Server start failed: %v\n", err)
			os.Exit(1)
		}
	}()
	
	// 等待信号
	<-quit
	fmt.Println("\n🛑 Shutting down server...")
	
	// 创建超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// 优雅关闭
	return a.Shutdown(ctx)
}

// RunTLS 运行HTTPS应用（自动处理信号）
func (a *App) RunTLS(addr, certFile, keyFile string) error {
	// 创建信号通道
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	
	// 启动服务器
	go func() {
		if err := a.StartTLS(addr, certFile, keyFile); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Server start failed: %v\n", err)
			os.Exit(1)
		}
	}()
	
	// 等待信号
	<-quit
	fmt.Println("\n🛑 Shutting down server...")
	
	// 创建超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// 优雅关闭
	return a.Shutdown(ctx)
}