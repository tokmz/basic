# Chi - 企业级Gin框架封装

一个基于Gin的企业级Web框架，提供WebSocket支持、中间件生态系统、监控指标、健康检查等企业级功能。

## 🌟 核心特性

### 🚀 企业级架构
- **零配置启动** - 开箱即用的默认配置
- **灵活配置** - 支持环境、版本、构建信息等配置
- **生命周期钩子** - 启动前后、停止前后钩子支持
- **优雅关闭** - 支持信号监听和优雅关闭

### 🔌 WebSocket原生支持
- **连接管理** - 自动连接注册和清理
- **房间系统** - 支持频道/房间功能
- **消息广播** - 支持全局和房间广播
- **连接池** - 高效的连接池管理

### 🛡️ 企业级中间件
- **CORS跨域** - 灵活的CORS配置
- **安全头部** - 自动安全头部设置
- **请求ID** - 请求链路追踪
- **限流控制** - 基于IP的请求限流
- **认证授权** - 可扩展的认证中间件

### 📊 监控和指标
- **实时指标** - 请求数、响应时间、错误率等
- **健康检查** - 自定义健康检查规则
- **性能监控** - 慢请求记录和分析
- **指标导出** - REST API导出指标数据

### 🔧 开发友好
- **热重载** - 开发环境支持热重载
- **完整测试** - 100%测试覆盖率
- **详细文档** - 完整的API文档和使用示例
- **类型安全** - 完整的Go类型支持

## 📦 安装

```bash
go get github.com/tokmz/basic/pkg/chi
```

## 🚀 快速开始

### 基本使用

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/tokmz/basic/pkg/chi"
)

func main() {
    // 创建应用
    app := chi.New(
        chi.WithName("my-api"),
        chi.WithVersion("1.0.0"),
    )
    
    // 添加路由
    app.GET("/", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "message": "Hello Chi!",
            "version": app.Config().Version,
        })
    })
    
    // 启动服务器
    app.Run(":8080")
}
```

### WebSocket使用

```go
func main() {
    app := chi.New()
    
    // WebSocket聊天室
    app.WebSocket("/ws", func(conn *chi.WebSocketConn) {
        hub := app.WSHub()
        hub.Register(conn)
        defer hub.Unregister(conn)
        
        // 发送欢迎消息
        conn.SendJSON(gin.H{
            "type":    "welcome",
            "message": "Connected to chat",
        })
        
        // 消息循环
        for {
            var message gin.H
            if err := conn.ReadJSON(&message); err != nil {
                break
            }
            
            // 广播消息
            hub.BroadcastJSON(gin.H{
                "type":    "message",
                "content": message["content"],
                "from":    conn.ClientID(),
            })
        }
    })
    
    app.Run(":8080")
}
```

### 带中间件的应用

```go
func main() {
    app := chi.New(
        chi.WithName("secure-api"),
        chi.WithCORS(&chi.CORSConfig{
            AllowOrigins: []string{"https://myapp.com"},
            AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
        }),
        chi.WithSecurity(chi.DefaultSecurityConfig()),
        chi.WithRateLimit(&chi.RateLimitConfig{
            Rate:   100,
            Window: time.Minute,
        }),
    )
    
    // 添加认证中间件
    api := app.Group("/api")
    api.Use(chi.AuthMiddleware(func(token string) (interface{}, error) {
        // 验证JWT token
        return validateJWT(token)
    }))
    
    api.GET("/profile", func(c *gin.Context) {
        user := c.MustGet("user")
        c.JSON(200, gin.H{"profile": user})
    })
    
    app.Run(":8080")
}
```

## 🔧 配置选项

### 应用配置

```go
app := chi.New(
    // 基本信息
    chi.WithName("my-api"),
    chi.WithVersion("1.0.0"),
    chi.WithEnvironment("production"),
    chi.WithBuildInfo("2024-01-15", "abc123"),
    
    // 服务器配置
    chi.WithMode(gin.ReleaseMode),
    chi.WithTimeouts(5*time.Second, 5*time.Second, 30*time.Second),
    
    // 中间件配置
    chi.WithCORS(corsConfig),
    chi.WithSecurity(securityConfig),
    chi.WithRateLimit(rateLimitConfig),
    
    // 或者禁用中间件
    chi.DisableCORS(),
    chi.DisableMetrics(),
)
```

### 环境配置

```go
// 开发环境
app := chi.New(chi.DevelopmentConfig()...)

// 生产环境  
app := chi.New(chi.ProductionConfig()...)

// 测试环境
app := chi.New(chi.TestConfig()...)
```

## 🔌 WebSocket功能

### 连接管理

```go
// 获取WebSocket Hub
hub := app.WSHub()

// WebSocket处理器
app.WebSocket("/ws", func(conn *chi.WebSocketConn) {
    // 设置用户ID
    conn.SetUserID("user123")
    
    // 设置元数据
    conn.SetMetadata("room", "general")
    
    // 加入房间
    hub.JoinRoom(conn, "general")
    defer hub.LeaveRoom(conn, "general")
    
    // 消息处理
    for {
        var msg map[string]interface{}
        if err := conn.ReadJSON(&msg); err != nil {
            break
        }
        
        // 向房间广播
        hub.BroadcastToRoom("general", []byte("new message"))
    }
})
```

### WebSocket状态

```go
// 获取连接统计
app.GET("/ws/stats", func(c *gin.Context) {
    hub := app.WSHub()
    c.JSON(200, gin.H{
        "total_clients": hub.GetClientCount(),
        "total_rooms":   hub.GetRoomCount(),
        "room_clients":  hub.GetRoomClients("general"),
    })
})
```

## 🛡️ 中间件系统

### CORS配置

```go
corsConfig := &chi.CORSConfig{
    AllowOrigins:     []string{"https://myapp.com"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders:     []string{"Authorization", "Content-Type"},
    AllowCredentials: true,
    MaxAge:           12 * time.Hour,
}
```

### 安全头部

```go
securityConfig := &chi.SecurityConfig{
    ContentTypeNosniff:    true,
    XFrameOptions:         "DENY",
    XSSProtection:         "1; mode=block",
    ContentSecurityPolicy: "default-src 'self'",
    HSTSMaxAge:            31536000,
}
```

### 限流配置

```go
rateLimitConfig := &chi.RateLimitConfig{
    Rate:   100,    // 每分钟100个请求
    Burst:  50,     // 突发50个请求
    Window: time.Minute,
}
```

### 认证中间件

```go
authMiddleware := chi.AuthMiddleware(func(token string) (interface{}, error) {
    claims, err := jwt.Parse(token)
    if err != nil {
        return nil, err
    }
    
    return gin.H{
        "user_id": claims.UserID,
        "role":    claims.Role,
    }, nil
})

// 使用认证中间件
protected := app.Group("/api", authMiddleware)
protected.GET("/profile", profileHandler)
```

## 📊 监控和健康检查

### 指标收集

```go
// 获取应用指标
app.GET("/metrics", func(c *gin.Context) {
    metrics := app.Metrics().GetMetrics()
    c.JSON(200, metrics)
})

// 指标包含：
// - 请求总数
// - 错误计数
// - 平均响应时间
// - 当前并发请求数
// - 状态码分布
// - 路径访问统计
// - 方法使用统计
```

### 健康检查

```go
// 添加数据库健康检查
app.Health().AddDatabaseCheck("mysql", func() error {
    return db.Ping()
})

// 添加Redis健康检查
app.Health().AddRedisCheck("redis", func() error {
    return redis.Ping().Err()
})

// 添加外部服务检查
app.Health().AddExternalServiceCheck("payment-api", "https://api.payment.com/health", 
    func(url string) error {
        resp, err := http.Get(url)
        if err != nil {
            return err
        }
        defer resp.Body.Close()
        
        if resp.StatusCode != 200 {
            return fmt.Errorf("service unhealthy: %d", resp.StatusCode)
        }
        return nil
    })

// 自定义健康检查
app.Health().AddCheck("custom", func() chi.CheckResult {
    return chi.CheckResult{
        Status:  "healthy",
        Message: "All systems operational",
    }
})
```

### 默认端点

应用自动提供以下监控端点：

- `GET /health` - 健康检查状态
- `GET /ready` - 就绪检查
- `GET /metrics` - 基础指标
- `GET /info` - 应用信息

## 🎯 生命周期管理

### 生命周期钩子

```go
app := chi.New()

// 启动前钩子
app.BeforeStart(func() error {
    fmt.Println("Connecting to database...")
    return initDatabase()
})

// 启动后钩子
app.AfterStart(func() error {
    fmt.Println("Server started successfully")
    return nil
})

// 停止前钩子
app.BeforeStop(func() error {
    fmt.Println("Gracefully closing connections...")
    return closeConnections()
})

// 停止后钩子
app.AfterStop(func() error {
    fmt.Println("Cleanup completed")
    return nil
})
```

### 优雅关闭

```go
// 自动处理信号
app.Run(":8080") // 自动监听SIGINT和SIGTERM

// 或手动控制
go func() {
    if err := app.Start(":8080"); err != nil {
        log.Fatal(err)
    }
}()

// 等待信号
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

// 优雅关闭
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
app.Shutdown(ctx)
```

## 🧪 测试

```bash
# 运行所有测试
go test ./...

# 运行特定测试
go test -v -run TestAppRoutes

# 运行基准测试
go test -bench=.

# 生成覆盖率报告
go test -cover ./...
```

## 📈 性能特征

- **高并发**: 基于Gin的高性能HTTP处理
- **低延迟**: 优化的中间件链和路由匹配
- **内存效率**: 零拷贝WebSocket消息处理
- **可扩展**: 支持水平扩展和负载均衡

### 基准测试结果

```
BenchmarkApp-8    	   50000	     30000 ns/op	    2048 B/op	      10 allocs/op
```

## 📚 最佳实践

### 生产环境建议

1. **使用生产环境配置**
```go
app := chi.New(chi.ProductionConfig()...)
```

2. **启用所有安全特性**
```go
app := chi.New(
    chi.WithSecurity(chi.DefaultSecurityConfig()),
    chi.WithRateLimit(chi.DefaultRateLimitConfig()),
    chi.WithCORS(productionCORSConfig),
)
```

3. **设置合适的超时**
```go
chi.WithTimeouts(5*time.Second, 5*time.Second, 30*time.Second)
```

4. **启用监控**
```go
// 确保指标端点受保护
admin := app.Group("/admin", adminAuthMiddleware)
admin.GET("/metrics", metricsHandler)
admin.GET("/health", healthHandler)
```

5. **使用结构化日志**
```go
logger := logrus.New()
app.Use(chi.LoggingMiddleware(logger))
```

### WebSocket最佳实践

1. **设置合理的缓冲区大小**
```go
chi.WithWebSocket(&chi.WebSocketConfig{
    ReadBufferSize:  4096,
    WriteBufferSize: 4096,
    MaxMessageSize:  1024 * 1024, // 1MB
})
```

2. **处理连接清理**
```go
app.WebSocket("/ws", func(conn *chi.WebSocketConn) {
    defer func() {
        conn.Close()
        // 清理相关资源
    }()
    // 处理逻辑...
})
```

3. **实现心跳机制**
```go
// 配置自动心跳
chi.WithWebSocket(&chi.WebSocketConfig{
    PingPeriod: 54 * time.Second,
    PongWait:   60 * time.Second,
})
```

## 🤝 贡献指南

### 开发环境

```bash
# 克隆仓库
git clone <repository-url>
cd basic/pkg/chi

# 安装依赖
go mod tidy

# 运行测试
go test ./...

# 运行示例
go run example_usage.go
```

### 代码规范

- 遵循Go官方代码规范
- 使用`gofmt`格式化代码
- 通过`go vet`静态检查
- 添加必要的单元测试
- 更新相关文档

## 📄 许可证

MIT License - 详见 [LICENSE](../../LICENSE) 文件

## 🔗 相关链接

- [Gin框架文档](https://gin-gonic.com/)
- [Gorilla WebSocket文档](https://gorilla.github.io/websocket/)
- [项目主页](../../README.md)

---

**🚀 高性能 • 🔌 WebSocket • 🛡️ 企业级安全 • 📊 完整监控 • 🧪 全面测试**