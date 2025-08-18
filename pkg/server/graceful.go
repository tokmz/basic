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
)

// GracefulShutdown 优雅关闭管理器
type GracefulShutdown struct {
	server          *Server
	shutdownTimeout time.Duration
	connections     map[string]*http.Request
	connMutex       sync.RWMutex
	shutdownOnce    sync.Once
	shutdownCh      chan struct{}
	isShuttingDown  bool
}

// NewGracefulShutdown 创建优雅关闭管理器
func NewGracefulShutdown(server *Server, timeout time.Duration) *GracefulShutdown {
	return &GracefulShutdown{
		server:          server,
		shutdownTimeout: timeout,
		connections:     make(map[string]*http.Request),
		shutdownCh:      make(chan struct{}),
	}
}

// Start 启动优雅关闭监听
func (gs *GracefulShutdown) Start() {
	// 监听系统信号
	go gs.listenForShutdownSignals()
	
	// 启动连接跟踪
	gs.trackConnections()
}

// listenForShutdownSignals 监听关闭信号
func (gs *GracefulShutdown) listenForShutdownSignals() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	for sig := range sigCh {
		log.Printf("Received signal: %v", sig)
		gs.initiateShutdown()
		return
	}
}

// initiateShutdown 启动关闭流程
func (gs *GracefulShutdown) initiateShutdown() {
	gs.shutdownOnce.Do(func() {
		log.Println("Initiating graceful shutdown...")
		gs.isShuttingDown = true
		
		// 创建关闭上下文
		ctx, cancel := context.WithTimeout(context.Background(), gs.shutdownTimeout)
		defer cancel()
		
		// 执行关闭流程
		if err := gs.shutdown(ctx); err != nil {
			log.Printf("Graceful shutdown failed: %v", err)
			os.Exit(1)
		}
		
		log.Println("Graceful shutdown completed")
		os.Exit(0)
	})
}

// shutdown 执行关闭流程
func (gs *GracefulShutdown) shutdown(ctx context.Context) error {
	// 1. 停止接收新连接
	log.Println("Step 1: Stopping new connections...")
	if err := gs.server.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}
	
	// 2. 等待现有连接完成
	log.Println("Step 2: Waiting for existing connections to finish...")
	if err := gs.waitForConnections(ctx); err != nil {
		return fmt.Errorf("failed to wait for connections: %w", err)
	}
	
	// 3. 清理资源
	log.Println("Step 3: Cleaning up resources...")
	gs.cleanup()
	
	// 4. 通知关闭完成
	close(gs.shutdownCh)
	
	return nil
}

// waitForConnections 等待现有连接完成
func (gs *GracefulShutdown) waitForConnections(ctx context.Context) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			gs.connMutex.RLock()
			connCount := len(gs.connections)
			gs.connMutex.RUnlock()
			
			if connCount > 0 {
				return fmt.Errorf("shutdown timeout: %d connections still active", connCount)
			}
			return ctx.Err()
			
		case <-ticker.C:
			gs.connMutex.RLock()
			connCount := len(gs.connections)
			gs.connMutex.RUnlock()
			
			if connCount == 0 {
				log.Println("All connections finished")
				return nil
			}
			
			log.Printf("Waiting for %d connections to finish...", connCount)
		}
	}
}

// trackConnections 跟踪活跃连接
func (gs *GracefulShutdown) trackConnections() {
	// 包装原始的HTTP服务器
	originalHandler := gs.server.httpServer.Handler
	
	gs.server.httpServer.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 如果正在关闭，拒绝新连接
		if gs.isShuttingDown {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Server is shutting down"))
			return
		}
		
		// 生成连接ID
		connID := fmt.Sprintf("%s-%d", r.RemoteAddr, time.Now().UnixNano())
		
		// 注册连接
		gs.connMutex.Lock()
		gs.connections[connID] = r
		gs.connMutex.Unlock()
		
		// 处理请求
		defer func() {
			// 注销连接
			gs.connMutex.Lock()
			delete(gs.connections, connID)
			gs.connMutex.Unlock()
		}()
		
		originalHandler.ServeHTTP(w, r)
	})
}

// cleanup 清理资源
func (gs *GracefulShutdown) cleanup() {
	// 清理连接映射
	gs.connMutex.Lock()
	gs.connections = make(map[string]*http.Request)
	gs.connMutex.Unlock()
	
	// 这里可以添加其他资源清理逻辑
	// 例如：关闭数据库连接、清理缓存等
	log.Println("Resources cleaned up")
}

// GetActiveConnections 获取活跃连接数
func (gs *GracefulShutdown) GetActiveConnections() int {
	gs.connMutex.RLock()
	defer gs.connMutex.RUnlock()
	return len(gs.connections)
}

// IsShuttingDown 检查是否正在关闭
func (gs *GracefulShutdown) IsShuttingDown() bool {
	return gs.isShuttingDown
}

// WaitForShutdown 等待关闭完成
func (gs *GracefulShutdown) WaitForShutdown() {
	<-gs.shutdownCh
}

// ForceShutdown 强制关闭
func (gs *GracefulShutdown) ForceShutdown() {
	log.Println("Force shutdown initiated")
	os.Exit(1)
}