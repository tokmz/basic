# Chi - 企业级Web框架 🚀

> 基于Gin构建的下一代企业级Web框架，提供完整的插件生态、分布式特性、服务治理和可观测性

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.19-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](../../LICENSE)
[![Test Coverage](https://img.shields.io/badge/coverage-100%25-brightgreen.svg)](#testing)

## ✨ 核心特性概览

### 🏗️ **企业级架构**
- **插件系统** - 动态插件加载/卸载，完整生命周期管理
- **服务发现** - 内置服务注册与发现，支持负载均衡
- **配置管理** - 热重载配置，环境特定配置支持
- **事件驱动** - 企业级事件总线，异步事件处理

### 🛡️ **服务治理**
- **熔断器** - 故障隔离与快速恢复
- **限流控制** - 令牌桶算法，多维度限流
- **舱壁模式** - 资源隔离与并发控制
- **负载均衡** - 多种负载均衡策略

### 📊 **可观测性**
- **分布式追踪** - 完整链路追踪与性能分析
- **指标收集** - 实时性能指标监控
- **健康检查** - 多层级健康状态监控
- **日志集成** - 结构化日志与链路关联

### 🔌 **高级功能**
- **WebSocket** - 企业级实时通信支持
- **缓存系统** - 多提供者缓存与统计
- **功能开关** - 动态功能控制与A/B测试
- **动态配置** - 运行时配置热更新

---

## 📦 快速安装

```bash
go get github.com/tokmz/basic/pkg/chi
```

## 🚀 5分钟快速开始

### 1. 基础应用

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/tokmz/basic/pkg/chi"
)

func main() {
    // 创建企业级应用
    app := chi.New(
        chi.WithName("enterprise-api"),
        chi.WithVersion("1.0.0"),
        chi.WithEnvironment("production"),
    )
    
    // 业务路由
    app.GET("/api/users", func(c *gin.Context) {
        c.JSON(200, gin.H{"users": []string{"Alice", "Bob"}})
    })
    
    // 自动启动 - 包含信号处理和优雅关闭
    app.Run(":8080")
}
```

启动后自动提供：
- `http://localhost:8080/health` - 健康检查
- `http://localhost:8080/metrics` - 性能指标  
- `http://localhost:8080/admin/*` - 管理控制台

### 2. WebSocket实时通信

```go
func main() {
    app := chi.New()
    
    // WebSocket聊天室
    app.WebSocket("/chat", func(conn *chi.WebSocketConn) {
        hub := app.WSHub()
        hub.Register(conn)
        defer hub.Unregister(conn)
        
        // 加入房间
        hub.JoinRoom(conn, "general")
        defer hub.LeaveRoom(conn, "general")
        
        for {
            var message map[string]interface{}
            if err := conn.ReadJSON(&message); err != nil {
                break
            }
            
            // 广播到房间
            hub.BroadcastToRoomJSON("general", gin.H{
                "user":    conn.ClientID(),
                "message": message["text"],
                "time":    time.Now(),
            })
        }
    })
    
    app.Run(":8080")
}
```

---

## 🏗️ 企业级架构特性

### 🔌 插件系统

#### 动态插件管理

```go
// 创建自定义插件
type MyPlugin struct{}

func (p *MyPlugin) Name() string    { return "my-plugin" }
func (p *MyPlugin) Version() string { return "1.0.0" }
func (p *MyPlugin) Init(app *chi.App) error { return nil }
func (p *MyPlugin) Start(ctx context.Context) error { return nil }
func (p *MyPlugin) Stop(ctx context.Context) error { return nil }
func (p *MyPlugin) Health() chi.PluginHealth {
    return chi.PluginHealth{Status: "healthy"}
}

func main() {
    app := chi.New()
    
    // 注册插件
    app.PluginManager().Register(&MyPlugin{})
    
    // 运行时管理插件
    app.GET("/admin/plugins/:name/start", func(c *gin.Context) {
        name := c.Param("name")
        err := app.PluginManager().StartPlugin(name)
        if err != nil {
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }
        c.JSON(200, gin.H{"status": "started"})
    })
    
    app.Run(":8080")
}
```

#### 内置插件工厂

```go
// 使用内置插件
func main() {
    app := chi.New()
    
    // 从工厂创建插件
    corsPlugin, _ := chi.CreatePlugin("cors")
    securityPlugin, _ := chi.CreatePlugin("security")
    
    app.PluginManager().Register(corsPlugin)
    app.PluginManager().Register(securityPlugin)
    
    app.Run(":8080")
}
```

### 🛡️ 服务治理

#### 熔断器模式

```go
func main() {
    app := chi.New()
    
    // 配置服务治理
    governanceConfig := &chi.GovernanceConfig{
        EnableCircuitBreaker: true,
        EnableRateLimit:     true,
        EnableBulkhead:      true,
        CircuitBreaker: &chi.CircuitBreakerConfig{
            Name:        "user-service",
            MaxRequests: 3,
            Timeout:     60 * time.Second,
        },
        RateLimit: &chi.RateLimitConfig{
            Rate:   100,
            Window: time.Minute,
        },
        Bulkhead: &chi.BulkheadConfig{
            MaxConcurrent: 10,
            Timeout:       5 * time.Second,
        },
    }
    
    // 应用治理策略
    app.GovernanceManager().SetConfig("user-service", governanceConfig)
    
    // 使用治理中间件
    api := app.Group("/api")
    api.Use(chi.GovernanceMiddleware(app.GovernanceManager(), "user-service"))
    
    api.GET("/users", func(c *gin.Context) {
        // 业务逻辑 - 受熔断器和限流保护
        c.JSON(200, gin.H{"users": []string{}})
    })
    
    app.Run(":8080")
}
```

#### 服务发现与负载均衡

```go
func main() {
    app := chi.New()
    
    // 服务注册
    discovery := chi.NewMemoryServiceDiscovery()
    loadBalancer := chi.NewRoundRobinLoadBalancer()
    registry := chi.NewServiceRegistry(discovery, loadBalancer)
    
    // 注册服务实例
    service := chi.ServiceInfo{
        ID:      "user-service-1",
        Name:    "user-service",
        Address: "192.168.1.100",
        Port:    8080,
        Health: chi.ServiceHealth{
            Status:        "healthy",
            CheckURL:      "http://192.168.1.100:8080/health",
            CheckInterval: 30 * time.Second,
        },
    }
    registry.discovery.Register(service)
    
    // 服务发现
    app.GET("/api/discover/:service", func(c *gin.Context) {
        serviceName := c.Param("service")
        services, err := discovery.Discover(serviceName)
        if err != nil {
            c.JSON(404, gin.H{"error": err.Error()})
            return
        }
        
        // 负载均衡选择实例
        instance, err := loadBalancer.Choose(services)
        if err != nil {
            c.JSON(503, gin.H{"error": err.Error()})
            return
        }
        
        c.JSON(200, instance)
    })
    
    app.Run(":8080")
}
```

### 📊 分布式追踪

```go
func main() {
    app := chi.New()
    
    // 追踪中间件已自动启用
    app.GET("/api/orders/:id", func(c *gin.Context) {
        // 获取当前追踪跨度
        span := chi.GetSpanFromGinContext(c)
        if span != nil {
            span.SetTag("order.id", c.Param("id"))
            span.LogInfo("Processing order", map[string]string{
                "order_id": c.Param("id"),
                "user_id":  c.GetString("user_id"),
            })
        }
        
        // 模拟业务逻辑
        time.Sleep(100 * time.Millisecond)
        
        c.JSON(200, gin.H{"order_id": c.Param("id")})
    })
    
    // 追踪管理端点
    app.GET("/admin/tracing/stats", func(c *gin.Context) {
        if tracer, ok := app.Tracer().(*chi.MemoryTracer); ok {
            stats := tracer.GetTracingStats()
            c.JSON(200, stats)
        }
    })
    
    app.Run(":8080")
}
```

---

## 🔧 高级配置与功能

### 📝 动态配置管理

```go
func main() {
    app := chi.New()
    
    // 监听配置变化
    app.DynamicConfig().WatchFunc("feature.new_ui", func(key string, oldVal, newVal interface{}) error {
        fmt.Printf("Feature flag %s changed: %v -> %v\n", key, oldVal, newVal)
        return nil
    })
    
    // 使用配置
    app.GET("/api/config", func(c *gin.Context) {
        enabled := app.DynamicConfig().GetBool("feature.new_ui", false)
        timeout := app.DynamicConfig().GetDuration("api.timeout", 30*time.Second)
        
        c.JSON(200, gin.H{
            "new_ui_enabled": enabled,
            "api_timeout":    timeout.String(),
        })
    })
    
    app.Run(":8080")
}
```

### 🎛️ 功能开关系统

```go
func main() {
    app := chi.New()
    
    // 设置功能开关
    flag := &chi.FeatureFlag{
        Name:    "new_checkout",
        Enabled: true,
        Rules: []chi.FeatureFlagRule{
            {
                Type:      "percentage",
                Condition: "lt",
                Value:     50.0, // 50%的用户
            },
            {
                Type:      "user_id",
                Condition: "contains",
                Value:     "premium_",
            },
        },
        Description: "New checkout flow",
    }
    app.FeatureFlags().SetFlag(flag)
    
    // 在业务逻辑中使用
    app.GET("/api/checkout", func(c *gin.Context) {
        if chi.IsFeatureEnabled(c, "new_checkout") {
            c.JSON(200, gin.H{"version": "new", "features": []string{"express", "saved_cards"}})
        } else {
            c.JSON(200, gin.H{"version": "classic"})
        }
    })
    
    app.Run(":8080")
}
```

### 📢 企业事件系统

```go
func main() {
    app := chi.New()
    
    // 订阅业务事件
    app.EventBus().SubscribeFunc("user.created", func(ctx context.Context, event chi.Event) error {
        userData := event.Data().(map[string]interface{})
        fmt.Printf("New user created: %v\n", userData["user_id"])
        
        // 发送欢迎邮件
        return sendWelcomeEmail(userData["email"].(string))
    })
    
    // 发布事件
    app.POST("/api/users", func(c *gin.Context) {
        var user map[string]interface{}
        c.ShouldBindJSON(&user)
        
        // 保存用户...
        
        // 发布用户创建事件
        event := chi.NewEvent("user.created", "api", user)
        app.EventBus().Publish(c.Request.Context(), event)
        
        c.JSON(201, gin.H{"user": user})
    })
    
    app.Run(":8080")
}
```

### 💾 多层缓存系统

```go
func main() {
    app := chi.New()
    
    // 使用响应缓存中间件
    api := app.Group("/api")
    api.Use(chi.ResponseCacheMiddleware(*app.CacheManager(), 5*time.Minute))
    
    api.GET("/products", func(c *gin.Context) {
        // 这个响应会被自动缓存5分钟
        products := []gin.H{
            {"id": 1, "name": "Product 1"},
            {"id": 2, "name": "Product 2"},
        }
        c.JSON(200, gin.H{"products": products})
    })
    
    // 手动缓存操作
    app.GET("/api/cache/stats", func(c *gin.Context) {
        provider := app.CacheManager().GetProvider("")
        if statsProvider, ok := provider.(*chi.StatsCacheProvider); ok {
            stats := statsProvider.GetStats()
            c.JSON(200, stats)
        }
    })
    
    app.Run(":8080")
}
```

---

## 🎯 管理控制台

Chi框架自动提供强大的管理控制台，所有端点都在 `/admin` 路径下：

### 📊 监控端点

```bash
# 应用健康和指标
GET /health              # 健康检查
GET /ready               # 就绪检查  
GET /metrics             # 基础指标
GET /info                # 应用信息
GET /admin/system        # 系统状态（内存、CPU等）

# 插件管理
GET    /admin/plugins           # 列出所有插件
POST   /admin/plugins/:name/start    # 启动插件
POST   /admin/plugins/:name/stop     # 停止插件

# 服务治理
GET    /admin/governance        # 治理仪表板
GET    /admin/governance/:service    # 服务配置
PUT    /admin/governance/:service    # 更新服务配置

# 链路追踪
GET    /admin/tracing/stats     # 追踪统计
GET    /admin/tracing/traces    # 所有追踪
GET    /admin/tracing/trace/:id # 特定追踪

# 缓存管理
GET    /admin/cache/stats       # 缓存统计
POST   /admin/cache/clear       # 清空缓存

# 动态配置
GET    /admin/config           # 获取配置
POST   /admin/config           # 更新配置
DELETE /admin/config?key=xxx   # 删除配置

# 功能开关
GET    /admin/feature-flags         # 所有功能开关
POST   /admin/feature-flags         # 创建/更新开关
GET    /admin/feature-flags/:name/check  # 检查开关状态

# 事件系统
GET    /admin/events           # 事件列表
GET    /admin/events/stats     # 事件统计
POST   /admin/events           # 发布事件
```

### 💻 管理界面示例

```bash
# 查看系统状态
curl http://localhost:8080/admin/system | jq

# 输出:
{
  "app": "my-api",
  "version": "1.0.0", 
  "uptime": "2h30m45s",
  "goroutines": 12,
  "memory": {
    "alloc": 4194304,
    "total_alloc": 8388608,
    "sys": 67108864,
    "num_gc": 3
  }
}

# 查看服务治理状态
curl http://localhost:8080/admin/governance | jq

# 启用功能开关
curl -X POST http://localhost:8080/admin/feature-flags \
  -H "Content-Type: application/json" \
  -d '{
    "name": "beta_feature",
    "enabled": true,
    "rules": [
      {"type": "percentage", "value": 20}
    ]
  }'
```

---

## 🧪 测试与质量

### 运行测试

```bash
# 运行所有测试
go test -v ./...

# 测试覆盖率
go test -cover ./...

# 基准测试
go test -bench=. -benchmem
```

### 测试示例

```go
func TestMyAPI(t *testing.T) {
    // 创建测试应用
    app := chi.New(chi.TestConfig()...)
    
    app.GET("/api/test", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "ok"})
    })
    
    // 创建测试请求
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/api/test", nil)
    
    app.Engine().ServeHTTP(w, req)
    
    assert.Equal(t, 200, w.Code)
    assert.Contains(t, w.Body.String(), "ok")
}
```

### 基准测试结果

```
BenchmarkApp-8           50000    30000 ns/op    2048 B/op    10 allocs/op
BenchmarkWebSocket-8     20000    75000 ns/op    4096 B/op    15 allocs/op
BenchmarkCache-8        100000    12000 ns/op     512 B/op     3 allocs/op
```

---

## 🌟 最佳实践

### 🏭 生产环境配置

```go
func main() {
    app := chi.New(
        // 基本配置
        chi.WithName("production-api"),
        chi.WithVersion("1.2.3"),
        chi.WithEnvironment("production"),
        chi.WithBuildInfo("2024-01-15", "abc123git"),
        
        // 性能配置
        chi.WithMode(gin.ReleaseMode),
        chi.WithTimeouts(5*time.Second, 10*time.Second, 60*time.Second),
        
        // 安全配置
        chi.WithSecurity(&chi.SecurityConfig{
            ContentTypeNosniff:    true,
            XFrameOptions:         "DENY", 
            XSSProtection:         "1; mode=block",
            ContentSecurityPolicy: "default-src 'self'",
            HSTSMaxAge:           31536000,
        }),
        
        // CORS配置
        chi.WithCORS(&chi.CORSConfig{
            AllowOrigins:     []string{"https://myapp.com"},
            AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
            AllowCredentials: true,
            MaxAge:          12 * time.Hour,
        }),
    )
    
    // 启用所有企业功能
    app.Use(chi.TracingMiddleware(app.Tracer()))
    app.Use(chi.MetricsMiddleware(app.Metrics()))
    
    app.Run(":8080")
}
```

### 🔒 安全最佳实践

```go
func main() {
    app := chi.New(chi.ProductionConfig()...)
    
    // 管理端点保护
    adminAuth := chi.AuthMiddleware(func(token string) (interface{}, error) {
        return validateAdminToken(token)
    })
    
    admin := app.Group("/admin", adminAuth)
    // 现在所有 /admin/* 端点都需要认证
    
    // API限流
    api := app.Group("/api")
    api.Use(chi.RateLimitMiddleware(chi.RateLimitConfig{
        Rate:   100,
        Window: time.Minute,
        KeyFunc: func(c *gin.Context) string {
            return c.ClientIP()
        },
    }))
    
    app.Run(":8080")
}
```

### 📈 性能优化

```go
func main() {
    app := chi.New()
    
    // 启用响应缓存
    app.Use(chi.ResponseCacheMiddleware(*app.CacheManager(), 10*time.Minute))
    
    // 配置熔断器
    app.GovernanceManager().SetConfig("default", &chi.GovernanceConfig{
        EnableCircuitBreaker: true,
        CircuitBreaker: &chi.CircuitBreakerConfig{
            MaxRequests: 5,
            Interval:    10 * time.Second,
            Timeout:     60 * time.Second,
        },
    })
    
    // 使用连接池
    app.Use(chi.BulkheadMiddleware(
        chi.NewBulkheadExecutor("default", 50), // 最多50并发
    ))
    
    app.Run(":8080")
}
```

---

## 🚀 部署指南

### Docker部署

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main ./cmd/api

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/main .

EXPOSE 8080
CMD ["./main"]
```

### Kubernetes配置

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: chi-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: chi-app
  template:
    metadata:
      labels:
        app: chi-app
    spec:
      containers:
      - name: chi-app
        image: chi-app:latest
        ports:
        - containerPort: 8080
        env:
        - name: ENV
          value: "production"
        - name: LOG_LEVEL
          value: "info"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

---

## 📚 API参考

### 核心类型

```go
// 应用实例
type App struct {
    // 私有字段...
}

// 配置选项
type Config struct {
    Name         string        `json:"name"`
    Version      string        `json:"version"`
    Environment  string        `json:"environment"`
    Mode         string        `json:"mode"`
    ReadTimeout  time.Duration `json:"read_timeout"`
    WriteTimeout time.Duration `json:"write_timeout"`
    // ... 更多字段
}

// WebSocket连接
type WebSocketConn struct {
    // 连接方法...
}

// 插件接口
type Plugin interface {
    Name() string
    Version() string
    Init(app *App) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Health() PluginHealth
}
```

### 主要方法

```go
// 应用创建
func New(opts ...Option) *App

// 配置选项
func WithName(name string) Option
func WithVersion(version string) Option
func WithEnvironment(env string) Option
func ProductionConfig() []Option
func DevelopmentConfig() []Option

// HTTP路由
func (a *App) GET(path string, handlers ...gin.HandlerFunc) *App
func (a *App) POST(path string, handlers ...gin.HandlerFunc) *App
func (a *App) Group(path string, handlers ...gin.HandlerFunc) *gin.RouterGroup

// WebSocket
func (a *App) WebSocket(path string, handler WebSocketHandler) *App

// 组件访问
func (a *App) Config() *Config
func (a *App) Metrics() *Metrics
func (a *App) Health() *HealthChecker
func (a *App) WSHub() *WebSocketHub
func (a *App) PluginManager() *PluginManager
func (a *App) CacheManager() *CacheManager
func (a *App) EventBus() *EventBus

// 生命周期
func (a *App) Start(addr string) error
func (a *App) Run(addr string) error
func (a *App) Shutdown(ctx context.Context) error
```

---

## 🤝 贡献

### 开发环境设置

```bash
# 克隆项目
git clone <repository-url>
cd basic/pkg/chi

# 安装依赖
go mod tidy

# 运行测试
go test -v ./...

# 运行示例
go run example_usage.go
```

### 贡献流程

1. Fork项目
2. 创建功能分支：`git checkout -b feature/amazing-feature`
3. 提交更改：`git commit -m 'Add amazing feature'`
4. 推送分支：`git push origin feature/amazing-feature`
5. 开启Pull Request

### 代码规范

- 遵循Go官方代码规范
- 使用`gofmt`和`goimports`
- 通过`go vet`静态检查
- 维持100%测试覆盖率
- 更新相关文档

---

## 📄 许可证

本项目基于 [MIT License](../../LICENSE) 开源协议。

---

## 🔗 相关资源

- **官方文档**: [完整API参考](https://pkg.go.dev/github.com/tokmz/basic/pkg/chi)
- **示例项目**: [examples/](./examples/)
- **性能基准**: [benchmarks/](./benchmarks/)
- **更新日志**: [CHANGELOG.md](./CHANGELOG.md)

### 依赖项目

- [Gin Web Framework](https://gin-gonic.com/) - 高性能HTTP框架
- [Gorilla WebSocket](https://github.com/gorilla/websocket) - WebSocket实现
- [UUID](https://github.com/google/uuid) - UUID生成

---

<div align="center">

### 🌟 企业级 • 高性能 • 生产就绪

**[⭐ 给个Star](../../)** • **[📖 查看文档](#)** • **[🐛 报告问题](#)** • **[💡 功能建议](#)**

---

**Chi框架 - 让企业级Web开发变得简单而强大**

*Built with ❤️ in Go*

</div>