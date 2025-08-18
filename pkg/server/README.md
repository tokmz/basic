# Gin 框架 Web 服务技术文档

## 目录

1. [服务概述与架构设计](#1-服务概述与架构设计)
2. [核心功能详细说明](#2-核心功能详细说明)
3. [配置指南](#3-配置指南)
4. [使用示例](#4-使用示例)
5. [API 接口文档](#5-api-接口文档)
6. [部署指南](#6-部署指南)
7. [性能优化](#7-性能优化)
8. [故障排除](#8-故障排除)

---

## 1. 服务概述与架构设计

### 1.1 服务概述

本 Gin 框架 Web 服务是一个高性能、模块化的 HTTP 服务器实现，专为现代 Web 应用程序设计。服务提供了完整的企业级功能，包括优雅关闭、热重启、静态文件服务、路由管理、中间件系统等核心特性。

**主要特性：**
- 🚀 高性能：基于 Gin 框架，提供卓越的性能表现
- 🔧 模块化设计：清晰的代码结构，易于维护和扩展
- 🛡️ 企业级功能：优雅关闭、热重启、监控指标等
- 📁 静态文件服务：高效的静态资源服务能力
- 🔀 灵活路由：强大的路由管理和中间件系统
- 🔒 安全特性：CORS、认证、限流等安全机制
- 📊 可观测性：集成监控、日志、性能分析

### 1.2 技术栈

| 组件 | 技术选型 | 版本要求 | 用途 |
|------|----------|----------|------|
| **Web 框架** | Gin | >= 1.9.0 | HTTP 服务器框架 |
| **配置管理** | Viper | >= 1.15.0 | 配置文件解析 |
| **日志系统** | Zap | >= 1.24.0 | 结构化日志 |
| **数据库 ORM** | GORM | >= 1.25.0 | 数据库操作 |
| **缓存** | Redis | >= 6.0 | 缓存和会话存储 |
| **监控** | Prometheus | >= 2.40.0 | 指标收集 |
| **链路追踪** | OpenTelemetry | >= 1.16.0 | 分布式追踪 |
| **测试框架** | Testify | >= 1.8.0 | 单元测试 |

### 1.3 目录结构

```
/Users/aikzy/Desktop/basic/pkg/server/
├── config.go          # 配置管理
├── server.go          # 服务器核心
├── router.go          # 路由管理
├── middleware.go      # 中间件系统
├── graceful.go        # 优雅关闭
├── restart.go         # 热重启
├── static.go          # 静态文件服务
├── errors.go          # 错误定义
└── examples/          # 使用示例
    ├── basic_server.go
    ├── basic_server_test.go
    └── README.md
```

### 1.4 架构设计理念

#### 1.4.1 分层架构

```
┌─────────────────────────────────────┐
│            应用层 (Handlers)          │
├─────────────────────────────────────┤
│           中间件层 (Middleware)       │
├─────────────────────────────────────┤
│           路由层 (Router)            │
├─────────────────────────────────────┤
│           服务层 (Server)            │
├─────────────────────────────────────┤
│          基础设施层 (Infrastructure)   │
└─────────────────────────────────────┘
```

#### 1.4.2 模块化设计

- **配置模块**：统一的配置管理，支持多种配置源
- **服务器模块**：核心服务器功能，生命周期管理
- **路由模块**：灵活的路由注册和管理
- **中间件模块**：可插拔的中间件系统
- **静态文件模块**：高效的静态资源服务
- **监控模块**：完整的可观测性支持

#### 1.4.3 设计原则

1. **单一职责原则**：每个模块专注于特定功能
2. **开闭原则**：对扩展开放，对修改封闭
3. **依赖倒置原则**：依赖抽象而非具体实现
4. **接口隔离原则**：使用小而专一的接口
5. **组合优于继承**：通过组合实现功能扩展

---

## 2. 核心功能详细说明

### 2.1 优雅关闭 (Graceful Shutdown)

#### 2.1.1 功能概述

优雅关闭功能确保服务器在接收到终止信号时，能够安全地停止服务，完成正在处理的请求，释放资源，避免数据丢失。

#### 2.1.2 工作原理

```go
// GracefulShutdown 结构体
type GracefulShutdown struct {
    server          *http.Server
    shutdownTimeout time.Duration
    signals         []os.Signal
    hooks           []ShutdownHook
    activeConns     int64
    mu              sync.RWMutex
}
```

**关键流程：**

1. **信号监听**：监听 SIGINT、SIGTERM 等系统信号
2. **停止接收新请求**：关闭监听端口，拒绝新连接
3. **等待现有请求完成**：设置超时时间，等待正在处理的请求完成
4. **执行清理钩子**：按顺序执行注册的清理函数
5. **强制退出**：超时后强制关闭所有连接

#### 2.1.3 配置参数

```go
type Config struct {
    // 优雅关闭配置
    EnableGracefulShutdown bool          `mapstructure:"enable_graceful_shutdown"`
    ShutdownTimeout       time.Duration `mapstructure:"shutdown_timeout"`
    ShutdownSignals       []string      `mapstructure:"shutdown_signals"`
}
```

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `EnableGracefulShutdown` | bool | true | 是否启用优雅关闭 |
| `ShutdownTimeout` | time.Duration | 30s | 关闭超时时间 |
| `ShutdownSignals` | []string | ["SIGINT", "SIGTERM"] | 监听的信号列表 |

#### 2.1.4 使用示例

```go
// 创建优雅关闭管理器
graceful := NewGracefulShutdown(server, 30*time.Second)

// 注册清理钩子
graceful.RegisterHook("database", func(ctx context.Context) error {
    return db.Close()
})

graceful.RegisterHook("cache", func(ctx context.Context) error {
    return redis.Close()
})

// 启动监听
graceful.Start()
```

#### 2.1.5 最佳实践

1. **合理设置超时时间**：根据业务特点设置合适的超时时间
2. **注册清理钩子**：确保数据库连接、缓存等资源正确释放
3. **监控活跃连接**：实时监控活跃连接数，便于调试
4. **日志记录**：记录关闭过程中的关键事件

### 2.2 热重启 (Hot Reload)

#### 2.2.1 功能概述

热重启功能允许在不中断服务的情况下重新加载应用程序，实现零停机时间的代码更新和配置变更。

#### 2.2.2 实现原理

```go
// HotRestart 结构体
type HotRestart struct {
    server       *Server
    pidFile      string
    listener     net.Listener
    restartChan  chan os.Signal
    childProcess *os.Process
}
```

**重启流程：**

1. **接收重启信号**：监听 SIGUSR2 信号
2. **启动新进程**：使用相同参数启动新的子进程
3. **传递监听器**：将当前监听器传递给子进程
4. **等待子进程就绪**：确认子进程成功启动
5. **优雅关闭旧进程**：停止接收新请求，处理完现有请求后退出

#### 2.2.3 配置参数

```go
type Config struct {
    // 热重启配置
    EnableHotRestart bool   `mapstructure:"enable_hot_restart"`
    PidFile         string `mapstructure:"pid_file"`
    RestartSignal   string `mapstructure:"restart_signal"`
}
```

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `EnableHotRestart` | bool | false | 是否启用热重启 |
| `PidFile` | string | "/tmp/server.pid" | PID 文件路径 |
| `RestartSignal` | string | "SIGUSR2" | 重启信号 |

#### 2.2.4 使用示例

```go
// 启用热重启
config.EnableHotRestart = true
config.PidFile = "/var/run/myapp.pid"

// 创建服务器
srv, err := server.New(config)
if err != nil {
    log.Fatal(err)
}

// 启动服务器（支持热重启）
if err := srv.Run(); err != nil {
    log.Fatal(err)
}
```

**手动触发重启：**

```bash
# 发送重启信号
kill -USR2 $(cat /var/run/myapp.pid)

# 或使用 systemctl（如果配置了 systemd）
systemctl reload myapp
```

#### 2.2.5 注意事项

1. **文件描述符限制**：确保系统允许足够的文件描述符
2. **内存使用**：重启过程中会短暂存在两个进程
3. **状态同步**：无状态设计，避免进程间状态同步问题
4. **监控告警**：监控重启过程，及时发现异常

### 2.3 静态文件服务 (Static File Serving)

#### 2.3.1 功能概述

静态文件服务提供高效的静态资源服务能力，支持文件缓存、压缩、目录列表、安全检查等企业级功能。

#### 2.3.2 核心特性

```go
// StaticFileServer 结构体
type StaticFileServer struct {
    config           *Config
    fileSystem       http.FileSystem
    compressionLevel int
    cacheMaxAge      time.Duration
    indexFiles       []string
}
```

**主要功能：**

- ✅ **高性能文件服务**：优化的文件读取和传输
- ✅ **智能缓存控制**：基于文件修改时间的缓存策略
- ✅ **内容压缩**：支持 Gzip 压缩，减少传输大小
- ✅ **目录列表**：可配置的目录浏览功能
- ✅ **安全检查**：防止路径遍历攻击
- ✅ **MIME 类型检测**：自动识别文件类型
- ✅ **范围请求**：支持断点续传

#### 2.3.3 配置参数

```go
type Config struct {
    // 静态文件配置
    StaticDir              string        `mapstructure:"static_dir"`
    StaticURLPrefix        string        `mapstructure:"static_url_prefix"`
    EnableDirectoryListing bool          `mapstructure:"enable_directory_listing"`
    StaticCacheMaxAge      time.Duration `mapstructure:"static_cache_max_age"`
    EnableCompression      bool          `mapstructure:"enable_compression"`
    CompressionLevel       int           `mapstructure:"compression_level"`
    IndexFiles             []string      `mapstructure:"index_files"`
}
```

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `StaticDir` | string | "./static" | 静态文件目录 |
| `StaticURLPrefix` | string | "/static" | URL 前缀 |
| `EnableDirectoryListing` | bool | false | 是否启用目录列表 |
| `StaticCacheMaxAge` | time.Duration | 24h | 缓存最大时间 |
| `EnableCompression` | bool | true | 是否启用压缩 |
| `CompressionLevel` | int | 6 | 压缩级别 (1-9) |
| `IndexFiles` | []string | ["index.html"] | 默认索引文件 |

#### 2.3.4 使用示例

```go
// 配置静态文件服务
config := &Config{
    StaticDir:              "./public",
    StaticURLPrefix:        "/assets",
    EnableDirectoryListing: true,
    StaticCacheMaxAge:      24 * time.Hour,
    EnableCompression:      true,
    CompressionLevel:       6,
    IndexFiles:             []string{"index.html", "index.htm"},
}

// 创建静态文件服务器
staticServer := NewStaticFileServer(config)

// 设置路由
staticServer.SetupRoutes(gin.Default())
```

**目录结构示例：**

```
public/
├── css/
│   ├── style.css
│   └── bootstrap.min.css
├── js/
│   ├── app.js
│   └── jquery.min.js
├── images/
│   ├── logo.png
│   └── banner.jpg
└── index.html
```

**访问 URL：**

- `http://localhost:8080/assets/` → 目录列表
- `http://localhost:8080/assets/css/style.css` → CSS 文件
- `http://localhost:8080/assets/images/logo.png` → 图片文件

#### 2.3.5 安全特性

1. **路径遍历防护**：防止 `../` 等路径遍历攻击
2. **文件类型检查**：只允许安全的文件类型
3. **访问控制**：可配置的访问权限
4. **大小限制**：防止大文件攻击

```go
// 安全检查示例
func (sfs *StaticFileServer) isPathSafe(path string) bool {
    // 检查路径遍历
    if strings.Contains(path, "..") {
        return false
    }
    
    // 检查隐藏文件
    if strings.HasPrefix(filepath.Base(path), ".") {
        return false
    }
    
    return true
}
```

### 2.4 路由管理 (Routing Management)

#### 2.4.1 功能概述

路由管理系统提供了灵活、强大的 HTTP 路由功能，支持路由分组、参数化路由、中间件绑定、路由信息查询等高级特性。

#### 2.4.2 核心组件

```go
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
```

#### 2.4.3 路由注册

**基础路由注册：**

```go
// 创建路由管理器
rm := NewRouterManager(server)

// 注册单个路由
rm.RegisterRoute("GET", "/api/users", getUsersHandler, "获取用户列表")
rm.RegisterRoute("POST", "/api/users", createUserHandler, "创建用户")
rm.RegisterRoute("GET", "/api/users/:id", getUserHandler, "获取用户详情")
```

**路由分组：**

```go
// 创建路由组
apiV1 := rm.CreateGroup("/api/v1", authMiddleware())

// 在路由组中注册路由
rm.RegisterGroupRoute("/api/v1", "GET", "/users", getUsersHandler, "获取用户列表")
rm.RegisterGroupRoute("/api/v1", "POST", "/users", createUserHandler, "创建用户")

// 嵌套路由组
usersGroup := rm.CreateGroup("/api/v1/users")
rm.RegisterGroupRoute("/api/v1/users", "GET", "/", getUsersHandler, "获取用户列表")
rm.RegisterGroupRoute("/api/v1/users", "GET", "/:id", getUserHandler, "获取用户详情")
```

#### 2.4.4 参数化路由

```go
// 路径参数
r.GET("/users/:id", func(c *gin.Context) {
    id := c.Param("id")
    c.JSON(200, gin.H{"user_id": id})
})

// 通配符参数
r.GET("/files/*filepath", func(c *gin.Context) {
    filepath := c.Param("filepath")
    c.JSON(200, gin.H{"filepath": filepath})
})

// 查询参数
r.GET("/search", func(c *gin.Context) {
    query := c.Query("q")
    page := c.DefaultQuery("page", "1")
    c.JSON(200, gin.H{"query": query, "page": page})
})
```

#### 2.4.5 路由信息查询

```go
// 获取所有路由信息
routes := rm.GetRoutes()

// 获取路由组信息
groups := rm.GetGroups()

// 路由统计接口
r.GET("/api/v1/routes", rm.routesHandler)
```

**路由信息 API 响应：**

```json
{
  "total": 15,
  "routes": {
    "GET:/api/v1/users": {
      "method": "GET",
      "path": "/api/v1/users",
      "handler": "getUsersHandler",
      "description": "获取用户列表",
      "middleware": ["authMiddleware"],
      "created_at": "2024-01-01T00:00:00Z"
    }
  },
  "groups": {
    "/api/v1": {
      "prefix": "/api/v1",
      "routes_count": 10
    }
  }
}
```

### 2.5 中间件系统

#### 2.5.1 功能概述

中间件系统提供了可插拔的请求处理管道，支持全局中间件、路由级中间件、条件中间件等多种使用方式。

#### 2.5.2 内置中间件

**1. CORS 中间件**

```go
func (s *Server) corsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        if s.config.EnableCORS {
            c.Header("Access-Control-Allow-Origin", s.config.CORSAllowOrigins)
            c.Header("Access-Control-Allow-Methods", s.config.CORSAllowMethods)
            c.Header("Access-Control-Allow-Headers", s.config.CORSAllowHeaders)
            c.Header("Access-Control-Max-Age", fmt.Sprintf("%d", int(s.config.CORSMaxAge.Seconds())))
            
            if c.Request.Method == "OPTIONS" {
                c.AbortWithStatus(http.StatusNoContent)
                return
            }
        }
        c.Next()
    }
}
```

**2. 限流中间件**

```go
func (s *Server) rateLimitMiddleware() gin.HandlerFunc {
    if !s.config.EnableRateLimit {
        return func(c *gin.Context) { c.Next() }
    }
    
    limiter := rate.NewLimiter(rate.Limit(s.config.RateLimit), int(s.config.RateLimit))
    
    return func(c *gin.Context) {
        if !limiter.Allow() {
            c.JSON(http.StatusTooManyRequests, gin.H{
                "error": "请求过于频繁，请稍后再试",
                "code":  "RATE_LIMIT_EXCEEDED",
            })
            c.Abort()
            return
        }
        c.Next()
    }
}
```

**3. 监控中间件**

```go
func (s *Server) metricsMiddleware() gin.HandlerFunc {
    if !s.config.EnableMetrics {
        return func(c *gin.Context) { c.Next() }
    }
    
    return func(c *gin.Context) {
        start := time.Now()
        
        c.Next()
        
        duration := time.Since(start)
        status := c.Writer.Status()
        
        // 记录指标
        requestDuration.WithLabelValues(
            c.Request.Method,
            c.FullPath(),
            fmt.Sprintf("%d", status),
        ).Observe(duration.Seconds())
        
        requestTotal.WithLabelValues(
            c.Request.Method,
            c.FullPath(),
            fmt.Sprintf("%d", status),
        ).Inc()
    }
}
```

**4. 请求日志中间件**

```go
func RequestLoggerMiddleware() gin.HandlerFunc {
    return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
        return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
            param.ClientIP,
            param.TimeStamp.Format(time.RFC1123),
            param.Method,
            param.Path,
            param.Request.Proto,
            param.StatusCode,
            param.Latency,
            param.Request.UserAgent(),
            param.ErrorMessage,
        )
    })
}
```

**5. 安全中间件**

```go
func SecurityMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 安全头设置
        c.Header("X-Frame-Options", "DENY")
        c.Header("X-Content-Type-Options", "nosniff")
        c.Header("X-XSS-Protection", "1; mode=block")
        c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
        
        c.Next()
    }
}
```

#### 2.5.3 自定义中间件

```go
// 认证中间件示例
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.JSON(http.StatusUnauthorized, gin.H{
                "error": "缺少认证令牌",
            })
            c.Abort()
            return
        }
        
        // 验证 token
        if !validateToken(token) {
            c.JSON(http.StatusUnauthorized, gin.H{
                "error": "无效的认证令牌",
            })
            c.Abort()
            return
        }
        
        // 设置用户信息
        userID := getUserIDFromToken(token)
        c.Set("user_id", userID)
        
        c.Next()
    }
}

// 使用中间件
r.Use(AuthMiddleware())

// 路由级中间件
r.GET("/protected", AuthMiddleware(), protectedHandler)

// 路由组中间件
protected := r.Group("/api", AuthMiddleware())
{
    protected.GET("/profile", getProfile)
    protected.POST("/logout", logout)
}
```

---

## 3. 配置指南

### 3.1 配置文件格式

服务支持多种配置文件格式：YAML、JSON、TOML。推荐使用 YAML 格式。

#### 3.1.1 完整配置示例 (config.yaml)

```yaml
# 服务器基础配置
server:
  name: "Gin Web Server"
  host: "0.0.0.0"
  port: 8080
  mode: "release"  # debug, release, test

# 超时配置
timeouts:
  read: "30s"
  write: "30s"
  idle: "60s"
  shutdown: "30s"

# 功能开关
features:
  graceful_shutdown: true
  hot_restart: false
  metrics: true
  pprof: false
  tracing: true

# 静态文件服务
static:
  dir: "./static"
  url_prefix: "/static"
  enable_directory_listing: false
  cache_max_age: "24h"
  enable_compression: true
  compression_level: 6
  index_files:
    - "index.html"
    - "index.htm"

# CORS 配置
cors:
  enable: true
  allow_origins: "*"
  allow_methods: "GET,POST,PUT,DELETE,OPTIONS"
  allow_headers: "Origin,Content-Type,Authorization"
  max_age: "12h"

# 限流配置
rate_limit:
  enable: true
  rate: 100
  window: "1m"

# 日志配置
logging:
  level: "info"  # debug, info, warn, error
  format: "json"  # json, text
  output: "stdout"  # stdout, stderr, file
  file_path: "/var/log/server.log"
  max_size: 100  # MB
  max_backups: 3
  max_age: 28  # days
  compress: true

# 数据库配置
database:
  driver: "mysql"
  host: "localhost"
  port: 3306
  username: "root"
  password: "password"
  database: "myapp"
  charset: "utf8mb4"
  max_open_conns: 100
  max_idle_conns: 10
  conn_max_lifetime: "1h"

# Redis 配置
redis:
  host: "localhost"
  port: 6379
  password: ""
  database: 0
  pool_size: 10
  min_idle_conns: 5
  dial_timeout: "5s"
  read_timeout: "3s"
  write_timeout: "3s"

# 监控配置
monitoring:
  prometheus:
    enable: true
    path: "/metrics"
    namespace: "gin_server"
  
  jaeger:
    enable: true
    endpoint: "http://localhost:14268/api/traces"
    service_name: "gin-web-server"
    sample_rate: 0.1

# 安全配置
security:
  enable_https: false
  cert_file: "/path/to/cert.pem"
  key_file: "/path/to/key.pem"
  enable_csrf: false
  csrf_secret: "your-csrf-secret"
```

### 3.2 环境变量配置

支持通过环境变量覆盖配置文件中的设置：

```bash
# 服务器配置
export SERVER_HOST=0.0.0.0
export SERVER_PORT=8080
export SERVER_MODE=release

# 数据库配置
export DATABASE_HOST=localhost
export DATABASE_PORT=3306
export DATABASE_USERNAME=root
export DATABASE_PASSWORD=secret
export DATABASE_DATABASE=myapp

# Redis 配置
export REDIS_HOST=localhost
export REDIS_PORT=6379
export REDIS_PASSWORD=

# 功能开关
export FEATURES_GRACEFUL_SHUTDOWN=true
export FEATURES_HOT_RESTART=false
export FEATURES_METRICS=true
```

### 3.3 配置加载优先级

配置加载按以下优先级顺序：

1. **命令行参数** (最高优先级)
2. **环境变量**
3. **配置文件**
4. **默认值** (最低优先级)

### 3.4 配置验证

```go
// 配置验证示例
func (c *Config) Validate() error {
    if c.Port <= 0 || c.Port > 65535 {
        return ErrInvalidPort
    }
    
    if c.ReadTimeout <= 0 {
        return ErrInvalidTimeout
    }
    
    if c.EnableRateLimit && c.RateLimit <= 0 {
        return ErrInvalidRateLimit
    }
    
    if c.StaticDir != "" {
        if _, err := os.Stat(c.StaticDir); os.IsNotExist(err) {
            return fmt.Errorf("静态文件目录不存在: %s", c.StaticDir)
        }
    }
    
    return nil
}
```

### 3.5 配置热重载

```go
// 配置热重载示例
func (s *Server) WatchConfig() {
    viper.WatchConfig()
    viper.OnConfigChange(func(e fsnotify.Event) {
        log.Printf("配置文件发生变化: %s", e.Name)
        
        // 重新加载配置
        newConfig := &Config{}
        if err := viper.Unmarshal(newConfig); err != nil {
            log.Printf("配置重载失败: %v", err)
            return
        }
        
        // 验证新配置
        if err := newConfig.Validate(); err != nil {
            log.Printf("新配置验证失败: %v", err)
            return
        }
        
        // 应用新配置
        s.updateConfig(newConfig)
        log.Println("配置重载成功")
    })
}
```

---

## 4. 使用示例

### 4.1 基础服务器启动

```go
package main

import (
    "log"
    "time"
    
    "github.com/gin-gonic/gin"
    "your-project/pkg/server"
)

func main() {
    // 创建配置
    config := server.DefaultConfig()
    config.Name = "My Web Server"
    config.Port = 8080
    config.EnableGracefulShutdown = true
    config.EnableMetrics = true
    
    // 创建服务器
    srv, err := server.New(config)
    if err != nil {
        log.Fatal("创建服务器失败:", err)
    }
    
    // 添加路由
    srv.Engine().GET("/", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "Hello World"})
    })
    
    // 启动服务器
    log.Printf("服务器启动在 %s", config.GetAddr())
    if err := srv.Run(); err != nil {
        log.Fatal("服务器启动失败:", err)
    }
}
```

### 4.2 完整的 REST API 示例

```go
package main

import (
    "net/http"
    "strconv"
    "time"
    
    "github.com/gin-gonic/gin"
    "your-project/pkg/server"
)

// User 用户模型
type User struct {
    ID        int       `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
}

// 模拟数据存储
var users = []User{
    {ID: 1, Name: "张三", Email: "zhangsan@example.com", CreatedAt: time.Now()},
    {ID: 2, Name: "李四", Email: "lisi@example.com", CreatedAt: time.Now()},
}
var nextID = 3

func main() {
    // 配置服务器
    config := server.DefaultConfig()
    config.Name = "User API Server"
    config.Port = 8080
    config.EnableCORS = true
    config.EnableRateLimit = true
    config.RateLimit = 100
    config.RateLimitWindow = time.Minute
    
    // 创建服务器
    srv, err := server.New(config)
    if err != nil {
        log.Fatal(err)
    }
    
    // 设置路由
    setupRoutes(srv.Engine())
    
    // 启动服务器
    srv.Run()
}

func setupRoutes(r *gin.Engine) {
    // API 版本组
    v1 := r.Group("/api/v1")
    {
        // 用户路由
        users := v1.Group("/users")
        {
            users.GET("/", getUsers)
            users.POST("/", createUser)
            users.GET("/:id", getUser)
            users.PUT("/:id", updateUser)
            users.DELETE("/:id", deleteUser)
        }
    }
}

// 获取用户列表
func getUsers(c *gin.Context) {
    // 分页参数
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
    
    // 搜索参数
    search := c.Query("search")
    
    // 过滤用户
    filteredUsers := users
    if search != "" {
        filteredUsers = []User{}
        for _, user := range users {
            if strings.Contains(user.Name, search) || strings.Contains(user.Email, search) {
                filteredUsers = append(filteredUsers, user)
            }
        }
    }
    
    // 分页
    start := (page - 1) * size
    end := start + size
    if start > len(filteredUsers) {
        start = len(filteredUsers)
    }
    if end > len(filteredUsers) {
        end = len(filteredUsers)
    }
    
    result := filteredUsers[start:end]
    
    c.JSON(http.StatusOK, gin.H{
        "data": result,
        "pagination": gin.H{
            "page":  page,
            "size":  size,
            "total": len(filteredUsers),
        },
        "message": "获取用户列表成功",
    })
}

// 创建用户
func createUser(c *gin.Context) {
    var req struct {
        Name  string `json:"name" binding:"required"`
        Email string `json:"email" binding:"required,email"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "请求参数错误",
            "details": err.Error(),
        })
        return
    }
    
    // 检查邮箱是否已存在
    for _, user := range users {
        if user.Email == req.Email {
            c.JSON(http.StatusConflict, gin.H{
                "error": "邮箱已存在",
            })
            return
        }
    }
    
    // 创建新用户
    user := User{
        ID:        nextID,
        Name:      req.Name,
        Email:     req.Email,
        CreatedAt: time.Now(),
    }
    nextID++
    
    users = append(users, user)
    
    c.JSON(http.StatusCreated, gin.H{
        "data": user,
        "message": "用户创建成功",
    })
}

// 获取用户详情
func getUser(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "无效的用户ID",
        })
        return
    }
    
    for _, user := range users {
        if user.ID == id {
            c.JSON(http.StatusOK, gin.H{
                "data": user,
                "message": "获取用户详情成功",
            })
            return
        }
    }
    
    c.JSON(http.StatusNotFound, gin.H{
        "error": "用户不存在",
    })
}

// 更新用户
func updateUser(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "无效的用户ID",
        })
        return
    }
    
    var req struct {
        Name  string `json:"name"`
        Email string `json:"email"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "请求参数错误",
            "details": err.Error(),
        })
        return
    }
    
    for i, user := range users {
        if user.ID == id {
            if req.Name != "" {
                users[i].Name = req.Name
            }
            if req.Email != "" {
                users[i].Email = req.Email
            }
            
            c.JSON(http.StatusOK, gin.H{
                "data": users[i],
                "message": "用户更新成功",
            })
            return
        }
    }
    
    c.JSON(http.StatusNotFound, gin.H{
        "error": "用户不存在",
    })
}

// 删除用户
func deleteUser(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "无效的用户ID",
        })
        return
    }
    
    for i, user := range users {
        if user.ID == id {
            users = append(users[:i], users[i+1:]...)
            c.JSON(http.StatusOK, gin.H{
                "message": "用户删除成功",
            })
            return
        }
    }
    
    c.JSON(http.StatusNotFound, gin.H{
        "error": "用户不存在",
    })
}
```

### 4.3 中间件使用示例

```go
package main

import (
    "fmt"
    "net/http"
    "time"
    
    "github.com/gin-gonic/gin"
    "your-project/pkg/server"
)

func main() {
    config := server.DefaultConfig()
    srv, _ := server.New(config)
    r := srv.Engine()
    
    // 全局中间件
    r.Use(LoggerMiddleware())
    r.Use(ErrorHandlerMiddleware())
    r.Use(CORSMiddleware())
    
    // 公开路由
    r.GET("/", homeHandler)
    r.POST("/login", loginHandler)
    
    // 需要认证的路由组
    auth := r.Group("/api")
    auth.Use(AuthMiddleware())
    {
        auth.GET("/profile", profileHandler)
        auth.POST("/logout", logoutHandler)
        
        // 需要管理员权限的路由
        admin := auth.Group("/admin")
        admin.Use(AdminMiddleware())
        {
            admin.GET("/users", adminUsersHandler)
            admin.DELETE("/users/:id", adminDeleteUserHandler)
        }
    }
    
    srv.Run()
}

// 日志中间件
func LoggerMiddleware() gin.HandlerFunc {
    return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
        return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
            param.ClientIP,
            param.TimeStamp.Format(time.RFC1123),
            param.Method,
            param.Path,
            param.Request.Proto,
            param.StatusCode,
            param.Latency,
            param.Request.UserAgent(),
            param.ErrorMessage,
        )
    })
}

// 错误处理中间件
func ErrorHandlerMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()
        
        // 处理错误
        if len(c.Errors) > 0 {
            err := c.Errors.Last()
            
            switch err.Type {
            case gin.ErrorTypeBind:
                c.JSON(http.StatusBadRequest, gin.H{
                    "error": "请求参数错误",
                    "details": err.Error(),
                })
            case gin.ErrorTypePublic:
                c.JSON(http.StatusInternalServerError, gin.H{
                    "error": "服务器内部错误",
                })
            default:
                c.JSON(http.StatusInternalServerError, gin.H{
                    "error": "未知错误",
                })
            }
        }
    }
}

// CORS 中间件
func CORSMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Access-Control-Allow-Origin", "*")
        c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
        
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(http.StatusNoContent)
            return
        }
        
        c.Next()
    }
}

// 认证中间件
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.JSON(http.StatusUnauthorized, gin.H{
                "error": "缺少认证令牌",
            })
            c.Abort()
            return
        }
        
        // 验证 token（这里简化处理）
        if !isValidToken(token) {
            c.JSON(http.StatusUnauthorized, gin.H{
                "error": "无效的认证令牌",
            })
            c.Abort()
            return
        }
        
        // 设置用户信息
        userID := getUserIDFromToken(token)
        c.Set("user_id", userID)
        c.Set("user_role", getUserRoleFromToken(token))
        
        c.Next()
    }
}

// 管理员权限中间件
func AdminMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        role, exists := c.Get("user_role")
        if !exists || role != "admin" {
            c.JSON(http.StatusForbidden, gin.H{
                "error": "权限不足",
            })
            c.Abort()
            return
        }
        
        c.Next()
    }
}

// 处理器函数
func homeHandler(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"message": "Welcome to Gin Server"})
}

func loginHandler(c *gin.Context) {
    var req struct {
        Username string `json:"username" binding:"required"`
        Password string `json:"password" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.Error(err)
        return
    }
    
    // 验证用户名密码（简化处理）
    if req.Username == "admin" && req.Password == "password" {
        token := generateToken(req.Username, "admin")
        c.JSON(http.StatusOK, gin.H{
            "token": token,
            "message": "登录成功",
        })
    } else {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error": "用户名或密码错误",
        })
    }
}

func profileHandler(c *gin.Context) {
    userID, _ := c.Get("user_id")
    c.JSON(http.StatusOK, gin.H{
        "user_id": userID,
        "message": "用户信息",
    })
}

func logoutHandler(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"message": "退出成功"})
}

func adminUsersHandler(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"message": "管理员用户列表"})
}

func adminDeleteUserHandler(c *gin.Context) {
    userID := c.Param("id")
    c.JSON(http.StatusOK, gin.H{
        "message": fmt.Sprintf("删除用户 %s 成功", userID),
    })
}

// 辅助函数（简化实现）
func isValidToken(token string) bool {
    return token == "Bearer valid-token"
}

func getUserIDFromToken(token string) string {
    return "12345"
}

func getUserRoleFromToken(token string) string {
    return "admin"
}

func generateToken(username, role string) string {
    return "Bearer valid-token"
}
```

### 4.4 文件上传示例

```go
package main

import (
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "time"
    
    "github.com/gin-gonic/gin"
    "your-project/pkg/server"
)

func main() {
    config := server.DefaultConfig()
    srv, _ := server.New(config)
    r := srv.Engine()
    
    // 设置文件上传大小限制（32MB）
    r.MaxMultipartMemory = 32 << 20
    
    // 文件上传路由
    r.POST("/upload", uploadFileHandler)
    r.POST("/upload/multiple", uploadMultipleFilesHandler)
    
    // 文件下载路由
    r.GET("/download/:filename", downloadFileHandler)
    
    // 文件列表路由
    r.GET("/files", listFilesHandler)
    
    srv.Run()
}

// 单文件上传
func uploadFileHandler(c *gin.Context) {
    file, header, err := c.Request.FormFile("file")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "文件上传失败",
            "details": err.Error(),
        })
        return
    }
    defer file.Close()
    
    // 验证文件类型
    if !isAllowedFileType(header.Filename) {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "不支持的文件类型",
        })
        return
    }
    
    // 验证文件大小（10MB）
    if header.Size > 10*1024*1024 {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "文件大小超过限制（10MB）",
        })
        return
    }
    
    // 生成唯一文件名
    filename := generateUniqueFilename(header.Filename)
    filepath := filepath.Join("./uploads", filename)
    
    // 确保上传目录存在
    if err := os.MkdirAll("./uploads", 0755); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "创建上传目录失败",
        })
        return
    }
    
    // 保存文件
    dst, err := os.Create(filepath)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "保存文件失败",
        })
        return
    }
    defer dst.Close()
    
    if _, err := io.Copy(dst, file); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "写入文件失败",
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "data": gin.H{
            "filename":     filename,
            "original_name": header.Filename,
            "size":         header.Size,
            "content_type": header.Header.Get("Content-Type"),
            "upload_time":  time.Now(),
            "download_url": fmt.Sprintf("/download/%s", filename),
        },
        "message": "文件上传成功",
    })
}

// 多文件上传
func uploadMultipleFilesHandler(c *gin.Context) {
    form, err := c.MultipartForm()
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "解析表单失败",
        })
        return
    }
    
    files := form.File["files"]
    if len(files) == 0 {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "没有选择文件",
        })
        return
    }
    
    var uploadedFiles []gin.H
    var errors []string
    
    for _, fileHeader := range files {
        file, err := fileHeader.Open()
        if err != nil {
            errors = append(errors, fmt.Sprintf("%s: 打开文件失败", fileHeader.Filename))
            continue
        }
        
        // 验证文件
        if !isAllowedFileType(fileHeader.Filename) {
            errors = append(errors, fmt.Sprintf("%s: 不支持的文件类型", fileHeader.Filename))
            file.Close()
            continue
        }
        
        if fileHeader.Size > 10*1024*1024 {
            errors = append(errors, fmt.Sprintf("%s: 文件大小超过限制", fileHeader.Filename))
            file.Close()
            continue
        }
        
        // 保存文件
        filename := generateUniqueFilename(fileHeader.Filename)
        filepath := filepath.Join("./uploads", filename)
        
        dst, err := os.Create(filepath)
        if err != nil {
            errors = append(errors, fmt.Sprintf("%s: 保存文件失败", fileHeader.Filename))
            file.Close()
            continue
        }
        
        if _, err := io.Copy(dst, file); err != nil {
            errors = append(errors, fmt.Sprintf("%s: 写入文件失败", fileHeader.Filename))
            dst.Close()
        file.Close()
        continue
        }
        
        dst.Close()
        file.Close()
        
        uploadedFiles = append(uploadedFiles, gin.H{
            "filename":     filename,
            "original_name": fileHeader.Filename,
            "size":         fileHeader.Size,
            "content_type": fileHeader.Header.Get("Content-Type"),
            "upload_time":  time.Now(),
            "download_url": fmt.Sprintf("/download/%s", filename),
        })
    }
    
    response := gin.H{
        "data": gin.H{
            "uploaded_files": uploadedFiles,
            "total_count":    len(uploadedFiles),
        },
        "message": fmt.Sprintf("成功上传 %d 个文件", len(uploadedFiles)),
    }
    
    if len(errors) > 0 {
        response["errors"] = errors
        response["error_count"] = len(errors)
    }
    
    c.JSON(http.StatusOK, response)
}

// 文件下载
func downloadFileHandler(c *gin.Context) {
    filename := c.Param("filename")
    filepath := filepath.Join("./uploads", filename)
    
    // 检查文件是否存在
    if _, err := os.Stat(filepath); os.IsNotExist(err) {
        c.JSON(http.StatusNotFound, gin.H{
            "error": "文件不存在",
        })
        return
    }
    
    // 设置下载头
    c.Header("Content-Description", "File Transfer")
    c.Header("Content-Transfer-Encoding", "binary")
    c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
    c.Header("Content-Type", "application/octet-stream")
    
    c.File(filepath)
}

// 文件列表
func listFilesHandler(c *gin.Context) {
    files, err := os.ReadDir("./uploads")
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "读取文件列表失败",
        })
        return
    }
    
    var fileList []gin.H
    for _, file := range files {
        if !file.IsDir() {
            info, err := file.Info()
            if err != nil {
                continue
            }
            
            fileList = append(fileList, gin.H{
                "filename":    file.Name(),
                "size":        info.Size(),
                "modified":    info.ModTime(),
                "download_url": fmt.Sprintf("/download/%s", file.Name()),
            })
        }
    }
    
    c.JSON(http.StatusOK, gin.H{
        "data": fileList,
        "total": len(fileList),
        "message": "获取文件列表成功",
    })
}

// 辅助函数
func isAllowedFileType(filename string) bool {
    allowedExts := []string{".jpg", ".jpeg", ".png", ".gif", ".pdf", ".doc", ".docx", ".txt"}
    ext := strings.ToLower(filepath.Ext(filename))
    
    for _, allowedExt := range allowedExts {
        if ext == allowedExt {
            return true
        }
    }
    return false
}

func generateUniqueFilename(originalName string) string {
    ext := filepath.Ext(originalName)
    name := strings.TrimSuffix(originalName, ext)
    timestamp := time.Now().Unix()
    return fmt.Sprintf("%s_%d%s", name, timestamp, ext)
}
```

### 4.5 WebSocket 示例

```go
package main

import (
    "log"
    "net/http"
    "sync"
    
    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
    "your-project/pkg/server"
)

// WebSocket 升级器
var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true // 允许所有来源
    },
}

// 客户端连接管理
type Client struct {
    conn   *websocket.Conn
    send   chan []byte
    hub    *Hub
    userID string
}

// 连接中心
type Hub struct {
    clients    map[*Client]bool
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
    mu         sync.RWMutex
}

func newHub() *Hub {
    return &Hub{
        clients:    make(map[*Client]bool),
        broadcast:  make(chan []byte),
        register:   make(chan *Client),
        unregister: make(chan *Client),
    }
}

func (h *Hub) run() {
    for {
        select {
        case client := <-h.register:
            h.mu.Lock()
            h.clients[client] = true
            h.mu.Unlock()
            log.Printf("客户端 %s 已连接，当前连接数: %d", client.userID, len(h.clients))
            
        case client := <-h.unregister:
            h.mu.Lock()
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                close(client.send)
            }
            h.mu.Unlock()
            log.Printf("客户端 %s 已断开，当前连接数: %d", client.userID, len(h.clients))
            
        case message := <-h.broadcast:
            h.mu.RLock()
            for client := range h.clients {
                select {
                case client.send <- message:
                default:
                    close(client.send)
                    delete(h.clients, client)
                }
            }
            h.mu.RUnlock()
        }
    }
}

func main() {
    config := server.DefaultConfig()
    srv, _ := server.New(config)
    r := srv.Engine()
    
    // 创建 WebSocket 中心
    hub := newHub()
    go hub.run()
    
    // WebSocket 路由
    r.GET("/ws", func(c *gin.Context) {
        handleWebSocket(hub, c)
    })
    
    // 广播消息接口
    r.POST("/broadcast", func(c *gin.Context) {
        var req struct {
            Message string `json:"message" binding:"required"`
        }
        
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        
        hub.broadcast <- []byte(req.Message)
        c.JSON(http.StatusOK, gin.H{"message": "广播成功"})
    })
    
    srv.Run()
}

func handleWebSocket(hub *Hub, c *gin.Context) {
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        log.Printf("WebSocket 升级失败: %v", err)
        return
    }
    
    userID := c.Query("user_id")
    if userID == "" {
        userID = "anonymous"
    }
    
    client := &Client{
        conn:   conn,
        send:   make(chan []byte, 256),
        hub:    hub,
        userID: userID,
    }
    
    client.hub.register <- client
    
    go client.writePump()
    go client.readPump()
}

func (c *Client) readPump() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
    }()
    
    for {
        _, message, err := c.conn.ReadMessage()
        if err != nil {
            break
        }
        
        // 广播消息给所有客户端
        c.hub.broadcast <- message
    }
}

func (c *Client) writePump() {
    defer c.conn.Close()
    
    for {
        select {
        case message, ok := <-c.send:
            if !ok {
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }
            
            if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
                return
            }
        }
    }
}
```

---

## 5. API 接口文档

### 5.1 健康检查接口

**GET /health**

检查服务器健康状态。

**响应示例：**
```json
{
  "status": "ok",
  "timestamp": "2024-01-01T12:00:00Z",
  "uptime": "2h30m15s",
  "version": "1.0.0"
}
```

### 5.2 服务器信息接口

**GET /info**

获取服务器基本信息。

**响应示例：**
```json
{
  "name": "Gin Web Server",
  "version": "1.0.0",
  "go_version": "go1.21.0",
  "gin_version": "v1.9.1",
  "start_time": "2024-01-01T10:00:00Z",
  "uptime": "2h30m15s"
}
```

### 5.3 监控指标接口

**GET /metrics**

获取 Prometheus 格式的监控指标。

### 5.4 路由信息接口

**GET /routes**

获取所有注册的路由信息。

**响应示例：**
```json
{
  "total": 15,
  "routes": {
    "GET:/api/v1/users": {
      "method": "GET",
      "path": "/api/v1/users",
      "handler": "getUsersHandler",
      "description": "获取用户列表",
      "middleware": ["authMiddleware"],
      "created_at": "2024-01-01T00:00:00Z"
    }
  }
}
```

---

## 6. 部署指南

### 6.1 Docker 部署

**Dockerfile:**

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/server .
COPY --from=builder /app/config.yaml .

EXPOSE 8080
CMD ["./server"]
```

**docker-compose.yml:**

```yaml
version: '3.8'

services:
  web:
    build: .
    ports:
      - "8080:8080"
    environment:
      - SERVER_MODE=release
      - DATABASE_HOST=db
      - REDIS_HOST=redis
    depends_on:
      - db
      - redis
    volumes:
      - ./static:/app/static
      - ./uploads:/app/uploads

  db:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: password
      MYSQL_DATABASE: myapp
    ports:
      - "3306:3306"
    volumes:
      - db_data:/var/lib/mysql

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

volumes:
  db_data:
```

### 6.2 Kubernetes 部署

**deployment.yaml:**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gin-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: gin-server
  template:
    metadata:
      labels:
        app: gin-server
    spec:
      containers:
      - name: gin-server
        image: gin-server:latest
        ports:
        - containerPort: 8080
        env:
        - name: SERVER_MODE
          value: "release"
        - name: DATABASE_HOST
          value: "mysql-service"
        - name: REDIS_HOST
          value: "redis-service"
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: gin-server-service
spec:
  selector:
    app: gin-server
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: LoadBalancer
```

### 6.3 系统服务部署

**systemd 服务文件 (/etc/systemd/system/gin-server.service):**

```ini
[Unit]
Description=Gin Web Server
After=network.target

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=/opt/gin-server
ExecStart=/opt/gin-server/server
ExecReload=/bin/kill -USR2 $MAINPID
Restart=always
RestartSec=5
Environment=SERVER_MODE=release
Environment=CONFIG_FILE=/opt/gin-server/config.yaml

[Install]
WantedBy=multi-user.target
```

**启动服务：**

```bash
sudo systemctl daemon-reload
sudo systemctl enable gin-server
sudo systemctl start gin-server
sudo systemctl status gin-server
```

---

## 7. 性能优化

### 7.1 性能调优建议

1. **连接池优化**
   - 合理设置数据库连接池大小
   - 配置 Redis 连接池参数
   - 监控连接池使用情况

2. **缓存策略**
   - 使用 Redis 缓存热点数据
   - 实施多级缓存架构
   - 合理设置缓存过期时间

3. **静态资源优化**
   - 启用 Gzip 压缩
   - 设置合适的缓存头
   - 使用 CDN 加速

4. **数据库优化**
   - 优化 SQL 查询
   - 建立合适的索引
   - 使用读写分离

### 7.2 监控指标

**关键性能指标：**

- **响应时间**：P50、P95、P99 响应时间
- **吞吐量**：每秒请求数 (RPS)
- **错误率**：4xx、5xx 错误比例
- **资源使用**：CPU、内存、磁盘使用率
- **数据库性能**：连接数、查询时间
- **缓存命中率**：Redis 缓存命中率

---

## 8. 故障排除

### 8.1 常见问题

**1. 服务启动失败**

```bash
# 检查端口占用
lsof -i :8080

# 检查配置文件
./server --config-check

# 查看详细日志
./server --log-level debug
```

**2. 内存泄漏**

```bash
# 启用 pprof
curl http://localhost:8080/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# 查看 goroutine
curl http://localhost:8080/debug/pprof/goroutine?debug=1
```

**3. 数据库连接问题**

```bash
# 检查数据库连接
mysql -h localhost -u root -p

# 查看连接池状态
curl http://localhost:8080/debug/vars
```

### 8.2 日志分析

**日志级别：**
- `DEBUG`：详细调试信息
- `INFO`：一般信息
- `WARN`：警告信息
- `ERROR`：错误信息

**日志格式：**
```json
{
  "level": "info",
  "timestamp": "2024-01-01T12:00:00Z",
  "message": "请求处理完成",
  "method": "GET",
  "path": "/api/users",
  "status": 200,
  "latency": "15ms",
  "user_id": "12345"
}
```

### 8.3 性能分析

**使用 pprof 进行性能分析：**

```bash
# CPU 分析
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30

# 内存分析
go tool pprof http://localhost:8080/debug/pprof/heap

# 阻塞分析
go tool pprof http://localhost:8080/debug/pprof/block
```

---

## 总结

本技术文档详细介绍了基于 Gin 框架的 Web 服务的各个方面，包括：

- ✅ **完整的架构设计**：模块化、可扩展的系统架构
- ✅ **核心功能实现**：优雅关闭、热重启、静态文件服务、路由管理
- ✅ **企业级特性**：监控、日志、安全、性能优化
- ✅ **详细的配置指南**：支持多种配置方式和环境
- ✅ **丰富的使用示例**：从基础到高级的完整示例
- ✅ **部署和运维**：Docker、Kubernetes、系统服务部署
- ✅ **故障排除**：常见问题解决方案和性能分析

该服务具备了生产环境所需的所有特性，可以直接用于构建高性能、可靠的 Web 应用程序。通过合理的配置和优化，能够满足各种规模的业务需求。