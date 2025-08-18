package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// Server Gin Web服务器封装
type Server struct {
	config     *Config
	engine     *gin.Engine
	httpServer *http.Server
	mu         sync.RWMutex
	running    bool
	shutdownCh chan struct{}
	restartCh  chan struct{}
}

// New 创建新的服务器实例
func New(config *Config) *Server {
	if config == nil {
		config = DefaultConfig()
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	// 设置Gin模式
	if !config.EnablePprof {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建Gin引擎
	engine := gin.New()

	// 创建HTTP服务器
	httpServer := &http.Server{
		Addr:         config.GetAddr(),
		Handler:      engine,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	}

	return &Server{
		config:     config,
		engine:     engine,
		httpServer: httpServer,
		shutdownCh: make(chan struct{}),
		restartCh:  make(chan struct{}),
	}
}

// Engine 获取Gin引擎实例
func (s *Server) Engine() *gin.Engine {
	return s.engine
}

// Config 获取服务器配置
func (s *Server) Config() *Config {
	return s.config
}

// IsRunning 检查服务器是否正在运行
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// Run 启动服务器
func (s *Server) Run() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return ErrServerAlreadyRunning
	}
	s.running = true
	s.mu.Unlock()

	// 设置中间件
	s.setupMiddleware()

	// 设置路由
	s.setupRoutes()

	// 启动信号监听
	if s.config.EnableGracefulShutdown || s.config.EnableHotRestart {
		go s.handleSignals()
	}

	// 启动HTTP服务器
	log.Printf("Starting server on %s", s.config.GetAddr())
	err := s.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown 优雅关闭服务器
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return ErrServerNotStarted
	}
	s.running = false
	s.mu.Unlock()

	log.Println("Shutting down server...")

	// 关闭HTTP服务器
	err := s.httpServer.Shutdown(ctx)
	if err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	// 通知关闭完成
	close(s.shutdownCh)
	log.Println("Server shutdown complete")

	return nil
}

// handleSignals 处理系统信号
func (s *Server) handleSignals() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// 如果启用热重启，也监听重启信号
	if s.config.EnableHotRestart {
		signal.Notify(sigCh, syscall.SIGUSR2)
	}

	for sig := range sigCh {
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			if s.config.EnableGracefulShutdown {
				s.gracefulShutdown()
			} else {
				os.Exit(0)
			}
		case syscall.SIGUSR2:
			if s.config.EnableHotRestart {
				s.hotRestart()
			}
		}
	}
}

// gracefulShutdown 执行优雅关闭
func (s *Server) gracefulShutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		log.Printf("Graceful shutdown failed: %v", err)
		os.Exit(1)
	}

	os.Exit(0)
}

// hotRestart 执行热重启
func (s *Server) hotRestart() {
	log.Println("Hot restart triggered")
	// 热重启逻辑将在 restart.go 中实现
	close(s.restartCh)
}

// setupMiddleware 设置中间件
func (s *Server) setupMiddleware() {
	// 基础中间件
	s.engine.Use(gin.Logger())
	s.engine.Use(gin.Recovery())

	// CORS中间件
	if s.config.EnableCORS {
		s.engine.Use(s.corsMiddleware())
	}

	// 限流中间件
	if s.config.EnableRateLimit {
		s.engine.Use(s.rateLimitMiddleware())
	}

	// 监控中间件
	if s.config.EnableMetrics {
		s.engine.Use(s.metricsMiddleware())
	}
}

// setupRoutes 设置基础路由
func (s *Server) setupRoutes() {
	// API路由组
	apiV1 := s.engine.Group("/api/v1")
	{
		// 健康检查
		apiV1.GET("/health", s.healthHandler)

		// 监控指标
		if s.config.EnableMetrics {
			apiV1.GET("/metrics", s.metricsHandler)
		}

		// 性能分析
		if s.config.EnablePprof {
			s.setupPprofRoutes(apiV1)
		}
	}

	// 静态文件服务
	if s.config.StaticDir != "" {
		s.setupStaticRoutes()
	}
}

// healthHandler 健康检查处理器
func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
		"name":      s.config.Name,
	})
}

// metricsHandler 监控指标处理器
func (s *Server) metricsHandler(c *gin.Context) {
	// 监控指标逻辑将在后续实现
	c.String(http.StatusOK, "# Metrics endpoint\n")
}

// setupPprofRoutes 设置性能分析路由
func (s *Server) setupPprofRoutes(group *gin.RouterGroup) {
	// pprof路由将在后续实现
	group.GET("/pprof/*action", func(c *gin.Context) {
		c.String(http.StatusOK, "pprof endpoint")
	})
}

// setupStaticRoutes 设置静态文件路由
func (s *Server) setupStaticRoutes() {
	// 静态文件服务逻辑将在 static.go 中实现
	s.engine.Static(s.config.StaticURLPrefix, s.config.StaticDir)
}