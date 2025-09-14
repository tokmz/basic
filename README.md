# Basic - Go 常用库封装项目

一个采用 Go workspace 模式的项目，用于封装常用的 Go 库，实现模块化设计，便于维护和扩展。

## 🏗️ 项目架构

### Go Workspace 模式

本项目采用 Go 1.18+ 的 workspace 功能，实现多模块协同开发，同时保持各模块的独立性。

```
basic/
├── go.work                 # Go workspace 配置
├── README.md              # 项目总览文档
├── CHANGELOG.md           # 变更日志
├── CONCURRENT_SAFETY.md   # 并发安全文档
└── pkg/                   # 核心包目录
    ├── database/          # 数据库包
    │   ├── README.md      # 数据库包文档
    │   ├── go.mod         # 模块定义
    │   ├── *.go           # 核心实现
    │   ├── *_test.go      # 单元测试
    │   ├── concurrent_test.go # 并发安全测试
    │   └── examples/      # 使用示例
    │       ├── basic/     # 基础用法示例
    │       └── advanced/  # 高级用法示例
    ├── cache/             # 缓存包
    │   ├── README.md      # 缓存包文档
    │   ├── go.mod         # 模块定义
    │   ├── *.go           # 核心实现
    │   └── *_test.go      # 单元测试
    ├── logger/            # 日志包
    │   ├── go.mod         # 模块定义
    │   ├── *.go           # 核心实现
    │   ├── *_test.go      # 单元测试
    │   └── example_usage.go # 使用示例
    ├── config/            # 配置包
    │   ├── README.md      # 配置包文档
    │   ├── go.mod         # 模块定义
    │   ├── *.go           # 核心实现
    │   ├── *_test.go      # 单元测试
    │   └── example_usage.go # 使用示例
    ├── monitor/           # 系统监控包
    │   ├── README.md      # 监控包文档
    │   ├── go.mod         # 模块定义
    │   ├── *.go           # 核心实现
    │   ├── *_test.go      # 单元测试
    │   └── example_usage.go # 使用示例
    └── chi/               # 企业级Web框架
        ├── README.md      # 框架文档
        ├── CHANGELOG.md   # 更新日志
        ├── go.mod         # 模块定义
        ├── *.go           # 核心实现
        ├── *_test.go      # 单元测试
        └── example_usage.go # 使用示例
```

## 📦 已实现的包

### Database 包 - 高级数据库操作封装

基于 GORM 的企业级数据库包，提供以下核心功能：

#### ✨ 核心特性

1. **读写分离** - 自动路由读写操作到不同数据库
   - 写操作 → 主库 (Master)
   - 读操作 → 从库 (Slaves) 
   - 支持多从库负载均衡
   - 可配置权重分配

2. **外部日志集成** - 无缝集成 Zap 日志系统
   - 支持外部 Zap 日志实例
   - 可配置日志级别
   - 结构化日志输出

3. **慢查询监控** - 智能监控查询性能
   - 可配置慢查询阈值
   - 查询类型统计分析
   - 最近慢查询记录
   - 实时统计报告

4. **连接池管理** - 精细化连接池控制
   - 最大连接数配置
   - 空闲连接管理
   - 连接生命周期控制

5. **健康监控** - 主动健康状态监控
   - 定期健康检查
   - 自动故障恢复
   - 健康状态报告
   - 告警机制

#### 🔒 并发安全保证

**重要改进：** 项目已经过专门的并发安全优化，解决了原有的竞态条件问题：

- **线程安全的深拷贝** - 使用 JSON 序列化确保数据完整性
- **原子性统计更新** - SlowQueryMonitor 统计更新完全原子化
- **优化的锁策略** - 减少锁争用，提升并发性能
- **全面的竞态条件测试** - 包含专门的并发安全测试套件

详见 [并发安全文档](CONCURRENT_SAFETY.md)

#### 🚀 快速开始

```go
package main

import (
    "time"
    "github.com/tokmz/basic/pkg/database"
)

func main() {
    // 配置数据库
    config := &database.DatabaseConfig{
        Master: database.MasterConfig{
            Driver: "mysql",
            DSN: "user:pass@tcp(localhost:3306)/master_db",
        },
        Slaves: []database.SlaveConfig{
            {
                Driver: "mysql",
                DSN: "user:pass@tcp(localhost:3307)/slave_db",
                Weight: 2, // 权重配置
            },
        },
        // ... 其他配置
    }

    // 创建客户端
    client, err := database.NewClient(config)
    if err != nil {
        panic(err)
    }
    defer client.Close()

    // 使用数据库
    db := client.DB()
    // db.Create(...) -> 路由到主库
    // db.Find(...) -> 路由到从库
}
```

更多详细用法请参考：[Database 包文档](pkg/database/README.md)

### Cache 包 - 企业级缓存系统

基于内存和 Redis 的高性能企业级缓存包，提供多级缓存架构和完整的缓存管理解决方案：

#### ✨ 核心特性

1. **多级缓存架构** - 灵活的缓存层次设计
   - 内存缓存(L1) + Redis缓存(L2)
   - 支持LRU、LFU等多种淘汰策略
   - 自动缓存预热和数据同步
   - 可配置的TTL策略

2. **高并发支持** - 线程安全的缓存操作
   - 读写分离锁机制
   - 原子操作保证数据一致性
   - 支持数十万QPS访问
   - 防缓存击穿和雪崩

3. **完善监控** - 实时性能监控
   - 命中率、错误率统计
   - 响应时间监控
   - 内存使用量跟踪
   - 慢操作告警

4. **分布式一致性** - 多节点缓存同步
   - 最终一致性模式
   - 强一致性模式
   - 节点间数据同步
   - 故障自动恢复

5. **灵活配置** - 多种创建和配置方式
   - Builder模式构建
   - 工厂模式创建
   - 配置文件驱动
   - 运行时动态调整

#### 🚀 快速开始

```go
package main

import (
    "context"
    "time"
    "github.com/tokmz/basic/pkg/cache"
)

func main() {
    // 创建内存LRU缓存
    memCache, err := cache.Create(cache.TypeMemoryLRU)
    if err != nil {
        panic(err)
    }
    defer memCache.Close()
    
    ctx := context.Background()
    
    // 设置缓存
    err = memCache.Set(ctx, "user:123", "John Doe", time.Hour)
    if err != nil {
        panic(err)
    }
    
    // 获取缓存
    value, err := memCache.Get(ctx, "user:123")
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("User: %s\n", value)
    
    // 获取统计信息
    stats := memCache.Stats()
    fmt.Printf("命中率: %.2f%%\n", stats.HitRate*100)
}
```

#### 📁 多级缓存配置示例

```go
// 多级缓存配置
config := cache.DefaultConfig()
config.Type = cache.TypeMultiLevel
config.MultiLevel.EnableL1 = true  // 启用内存缓存
config.MultiLevel.EnableL2 = true  // 启用Redis缓存
config.MultiLevel.L1TTL = 30 * time.Minute
config.MultiLevel.L2TTL = 2 * time.Hour
config.MultiLevel.SyncStrategy = cache.SyncWriteThrough

cache, err := cache.CreateWithConfig(config)
if err != nil {
    panic(err)
}
defer cache.Close()
```

#### 🔔 Builder模式示例

```go
// 使用Builder模式创建缓存
cache, err := cache.NewCacheBuilder().
    WithType(cache.TypeMemoryLRU).
    WithMemoryConfig(cache.MemoryConfig{
        MaxSize:         1000,
        MaxMemory:       50 * 1024 * 1024, // 50MB
        EvictPolicy:     cache.EvictLRU,
        CleanupInterval: 5 * time.Minute,
    }).
    WithMonitoring(cache.MonitoringConfig{
        EnableMetrics:   true,
        EnableLogging:   true,
        SlowThreshold:   100 * time.Millisecond,
    }).
    Build()

if err != nil {
    panic(err)
}
defer cache.Close()
```

#### 🎯 缓存管理器示例

```go
// 获取默认管理器
manager := cache.GetDefaultManager()

// 创建并注册多个缓存实例
userCache, _ := manager.CreateAndRegisterCache("users", userConfig)
productCache, _ := manager.CreateAndRegisterCache("products", productConfig)

// 使用缓存
userCache.Set(ctx, "user:123", userData, time.Hour)
productCache.Set(ctx, "product:456", productData, time.Hour)

// 获取所有缓存的统计信息
allStats := manager.GetAllStats()
for name, stats := range allStats {
    fmt.Printf("缓存 %s: 命中率 %.2f%%\n", name, stats.HitRate*100)
}
```

更多详细用法请参考：[Cache 包文档](pkg/cache/README.md)

### Config 包 - 企业级配置管理系统

基于 Viper 的高性能企业级配置管理包，提供完整的配置管理解决方案：

#### ✨ 核心特性

1. **高性能配置管理** - 基于成熟稳定的 Viper 库
   - 多格式支持（YAML、JSON、TOML、HCL、INI）
   - 环境变量无缝集成
   - 远程配置源支持（etcd、Consul、Firestore）
   - 高效的内存缓存机制

2. **安全加密功能** - 保护敏感配置信息
   - AES-GCM 加密算法
   - 选择性配置项加密
   - 透明加解密操作
   - 密钥管理和轮换支持

3. **多环境管理** - 灵活的环境配置支持
   - 自动环境检测
   - 环境特定配置文件
   - 配置继承机制
   - 运行时环境切换

4. **配置验证** - 类型安全和数据验证
   - 强类型配置验证
   - 自定义验证规则
   - 范围和选项检查
   - 必填项验证

5. **热重载机制** - 动态配置更新
   - 文件变化监听
   - 配置变化事件通知
   - 回调函数支持
   - 原子性配置更新

6. **工厂模式** - 灵活的管理器创建
   - 多种管理器类型
   - 自动类型检测
   - 管理器注册机制
   - 预设环境工厂

#### 🚀 快速开始

```go
package main

import (
    "log"
    "github.com/tokmz/basic/pkg/config"
)

func main() {
    // 初始化全局配置
    err := config.Init(config.DevelopmentConfig())
    if err != nil {
        log.Fatal(err)
    }
    defer config.Close()
    
    // 读取配置
    dbHost := config.GetString("database.host")
    dbPort := config.GetInt("database.port")
    debugMode := config.GetBool("debug")
    
    fmt.Printf("Database: %s:%d, Debug: %t\n", dbHost, dbPort, debugMode)
}
```

#### 📁 Builder模式使用

```go
manager, err := config.NewConfigBuilder().
    WithConfigName("myapp").
    WithConfigType("yaml").
    WithConfigPaths("./config").
    WithEnvPrefix("MYAPP").
    WithWatchConfig(true).
    WithDefault("port", 8080).
    Build()

if err != nil {
    log.Fatal(err)
}
defer manager.Close()

port := manager.GetInt("port")
fmt.Printf("Server will start on port: %d\n", port)
```

#### 🔐 加密配置示例

```go
// 指定需要加密的配置键
encryptedKeys := []string{"database.password", "api.secret_key"}

// 创建加密管理器
manager, err := config.NewEncryptedViperManager(
    &config.Config{
        EncryptionKey: "your-encryption-key",
        ConfigName:    "app",
        ConfigType:    "yaml",
    },
    encryptedKeys,
)

// 设置敏感信息（自动加密）
manager.SetString("database.password", "super-secret-password")

// 读取配置（自动解密）
password := manager.GetString("database.password")
```

#### 🌍 多环境配置示例

```go
// 自动检测环境
detector := config.NewEnvironmentDetector()
currentEnv := detector.DetectEnvironment()

// 创建环境管理器
envManager := config.NewEnvironmentManager(currentEnv)
envManager.AddConfigPath(config.Development, "./config/dev")
envManager.AddConfigPath(config.Production, "./config/prod")

// 加载环境特定配置
manager, err := envManager.LoadWithEnvironment(currentEnv, nil)
```

更多详细用法请参考：[Config 包文档](pkg/config/README.md)

### Monitor 包 - 企业级系统监控包

一个功能完整的Go语言系统监控包，提供实时系统资源监控、告警管理、数据持久化等企业级功能：

#### ✨ 核心特性

1. **全面的系统监控** - 完整的系统资源监控
   - CPU监控（实时使用率、核心数统计）
   - 内存监控（使用量、可用量、交换分区）
   - 磁盘监控（使用率、I/O统计、挂载点信息）
   - 网络监控（接口状态、流量统计、连接信息）
   - 进程监控（进程列表、资源使用、进程树）
   - 系统负载（负载平均值监控）

2. **智能告警系统** - 灵活的告警规则引擎
   - 多级告警（Info、Warning、Critical）
   - 规则引擎（阈值、持续时间、操作符）
   - 多种通知方式（日志、邮件、Webhook）
   - 告警聚合和去重
   - 完整的告警历史记录

3. **数据持久化** - 高效的数据存储系统
   - 文件存储（支持压缩和轮转）
   - 内存存储（高性能临时存储）
   - 灵活查询（时间范围、标签过滤）
   - 自动数据清理和保留策略

4. **数据聚合分析** - 实时统计和趋势分析
   - 实时聚合计算（平均值、最大值、最小值）
   - 历史趋势分析
   - 缓存优化查询性能
   - 分页查询支持

5. **高度可扩展** - 模块化设计易于扩展
   - 自定义指标收集器
   - 插件式告警处理器
   - 配置驱动的灵活配置
   - 跨平台支持（Linux、macOS、Windows）

#### 🚀 快速开始

```go
package main

import (
    "fmt"
    "log"
    "github.com/tokmz/basic/pkg/monitor"
)

func main() {
    // 1. 创建监控器
    config := monitor.DefaultMonitorConfig()
    monitor := monitor.NewSystemMonitor(config)
    
    // 2. 获取系统信息
    systemInfo, err := monitor.GetSystemInfo()
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Hostname: %s\n", systemInfo.Hostname)
    fmt.Printf("OS: %s\n", systemInfo.OS)
    
    // 3. 获取CPU信息
    if systemInfo.CPU != nil {
        fmt.Printf("CPU Cores: %d, Usage: %.2f%%\n", 
            systemInfo.CPU.Cores, systemInfo.CPU.Usage)
    }
    
    // 4. 获取内存信息
    if systemInfo.Memory != nil {
        fmt.Printf("Memory Usage: %.2f%%\n", systemInfo.Memory.UsagePercent)
    }
}
```

更多详细用法请参考：[Monitor 包文档](pkg/monitor/README.md)

### Chi 包 - 企业级Web框架 🚀

基于 Gin 构建的下一代企业级Web框架，提供完整的插件生态、分布式特性、服务治理和可观测性：

#### ✨ 核心特性

1. **🏗️ 企业级架构**
   - **插件系统** - 动态插件加载/卸载，完整生命周期管理
   - **服务发现** - 内置服务注册与发现，支持负载均衡
   - **配置管理** - 热重载配置，环境特定配置支持
   - **事件驱动** - 企业级事件总线，异步事件处理

2. **🛡️ 服务治理**
   - **熔断器** - 故障隔离与快速恢复
   - **限流控制** - 令牌桶算法，多维度限流
   - **舱壁模式** - 资源隔离与并发控制
   - **负载均衡** - 多种负载均衡策略

3. **📊 可观测性**
   - **分布式追踪** - 完整链路追踪与性能分析
   - **指标收集** - 实时性能指标监控
   - **健康检查** - 多层级健康状态监控
   - **日志集成** - 结构化日志与链路关联

4. **🔌 高级功能**
   - **WebSocket** - 企业级实时通信支持
   - **缓存系统** - 多提供者缓存与统计
   - **功能开关** - 动态功能控制与A/B测试
   - **动态配置** - 运行时配置热更新

#### 🚀 快速开始

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

启动后自动提供管理控制台：
- `http://localhost:8080/health` - 健康检查
- `http://localhost:8080/metrics` - 性能指标  
- `http://localhost:8080/admin/*` - 管理控制台

#### 🎯 管理控制台

Chi框架自动提供强大的管理控制台，包含：

```bash
# 系统监控
GET /admin/system          # 系统状态（内存、CPU等）
GET /admin/plugins         # 插件管理
GET /admin/governance      # 服务治理仪表板
GET /admin/tracing/stats   # 链路追踪统计
GET /admin/cache/stats     # 缓存统计
GET /admin/config          # 动态配置
GET /admin/feature-flags   # 功能开关
GET /admin/events          # 事件系统
```

#### 🛡️ 服务治理示例

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

#### 🔌 WebSocket实时通信

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

#### 📢 企业事件系统

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

更多详细用法请参考：[Chi 包文档](pkg/chi/README.md)

### Logger 包 - 企业级日志系统

基于 Zap 的高性能企业级日志包，提供完整的日志管理解决方案：

#### ✨ 核心特性

1. **高性能日志** - 基于 Uber Zap 的高性能日志库
   - 结构化日志记录
   - 零内存分配（生产模式）
   - 异步日志写入支持

2. **日志轮转** - 智能日志文件管理
   - 按文件大小自动轮转
   - 按时间自动清理旧日志
   - 压缩存储节省空间
   - 可配置备份数量

3. **多输出支持** - 灵活的输出配置
   - 控制台输出（支持彩色）
   - 文件输出
   - 多种输出格式（JSON/Console）
   - 分级别输出控制

4. **钩子机制** - 可扩展的告警系统
   - 邮件告警钩子
   - Webhook 告警钩子
   - 数据库日志钩子
   - 自定义钩子支持

5. **中间件集成** - HTTP 中间件支持
   - 请求日志记录
   - 错误恢复中间件
   - 上下文追踪（Trace ID/Request ID）
   - 性能监控

6. **上下文感知** - 智能上下文信息提取
   - 自动提取 Trace ID
   - 自动提取 Request ID
   - 自动提取 User ID
   - 链路追踪支持

#### 🚀 快速开始

```go
package main

import (
    "github.com/tokmz/basic/pkg/logger"
    "go.uber.org/zap"
)

func main() {
    // 初始化全局日志器
    err := logger.InitGlobal(logger.DevelopmentConfig())
    if err != nil {
        panic(err)
    }
    defer logger.Sync()

    // 基本日志记录
    logger.Info("应用启动")
    logger.Infof("应用启动在端口: %d", 8080)

    // 结构化日志
    logger.Info("用户登录",
        zap.String("user_id", "12345"),
        zap.String("username", "john_doe"),
    )

    // 带字段的日志
    logger.WithFields(logger.Fields{
        "module": "auth",
        "action": "login",
    }).Info("用户认证成功")
}
```

#### 📁 生产环境配置示例

```go
// 生产环境配置
config := logger.ProductionConfig("my-service", "production")

// 自定义文件轮转
config.File.Filename = "./logs/app.log"
config.File.Rotation.MaxSize = 100    // 100MB
config.File.Rotation.MaxAge = 30      // 30天
config.File.Rotation.MaxBackups = 10  // 10个备份

// 创建日志器
log, err := logger.New(config)
if err != nil {
    panic(err)
}
defer log.Close()
```

#### 🔔 钩子告警示例

```go
// 创建邮件告警钩子
emailHook := logger.NewEmailHook(
    "smtp.company.com", 587,
    "alerts@company.com", "password",
    []string{"admin@company.com"},
)

// 创建Webhook钩子
webhookHook := logger.NewWebhookHook("https://hooks.slack.com/...")

// 创建带钩子的日志器
log, err := logger.NewWithHooks(
    logger.ProductionConfig("api-service", "production"),
    emailHook, webhookHook,
)

// 错误日志会触发告警
log.Error("数据库连接失败", zap.String("database", "users"))
```

## 🛠️ 开发指南

### 环境要求

- Go 1.24+
- 支持 Go Modules 和 Workspaces

### 本地开发

```bash
# 克隆项目
git clone <repository-url>
cd basic

# Go workspace 会自动识别所有模块
go mod tidy

# 运行测试
go test ./...

# 并发安全测试
go test -race ./...
```

### 添加新包

1. 在 `pkg/` 目录下创建新包
2. 初始化 go.mod: `go mod init github.com/tokmz/basic/pkg/<package-name>`
3. 更新根目录 `go.work` 文件，添加新包路径
4. 实现功能并编写测试
5. 添加使用示例和文档

### 代码质量标准

- **单元测试覆盖率** ≥ 80%
- **并发安全测试** 必须通过 `go test -race`
- **代码文档** 所有公开 API 必须有完整注释
- **错误处理** 统一的错误处理和日志记录
- **性能基准** 关键路径需要性能基准测试

## 🧪 测试策略

### 测试类型

1. **单元测试** - `*_test.go`
   - 功能正确性验证
   - 边界条件测试
   - 错误场景覆盖

2. **并发安全测试** - `concurrent_test.go`
   - 竞态条件检测
   - 高并发场景验证  
   - 内存安全测试

3. **集成测试** - `examples/`
   - 实际使用场景验证
   - 多组件协作测试

4. **性能测试** - `Benchmark*`
   - 关键路径性能验证
   - 内存使用分析
   - 并发性能测试

### 运行测试

```bash
# 所有测试
go test ./...

# 并发安全测试
go test -race ./...

# 性能基准测试
go test -bench=. ./...

# 测试覆盖率
go test -cover ./...

# 详细测试报告
go test -v -race -cover ./...
```

## 📊 项目状态

### Database 包状态

| 功能模块 | 实现状态 | 测试状态 | 并发安全 |
|----------|----------|----------|----------|
| 读写分离 | ✅ 完成 | ✅ 通过 | ✅ 安全 |
| 日志集成 | ✅ 完成 | ✅ 通过 | ✅ 安全 |
| 慢查询监控 | ✅ 完成 | ✅ 通过 | ✅ 安全 |
| 连接池管理 | ✅ 完成 | ✅ 通过 | ✅ 安全 |
| 健康监控 | ✅ 完成 | ✅ 通过 | ✅ 安全 |

### Logger 包状态

| 功能模块 | 实现状态 | 测试状态 | 企业特性 |
|----------|----------|----------|----------|
| 核心日志功能 | ✅ 完成 | ✅ 通过 | ✅ 支持 |
| 日志轮转 | ✅ 完成 | ✅ 通过 | ✅ 支持 |
| 多输出格式 | ✅ 完成 | ✅ 通过 | ✅ 支持 |
| 钩子机制 | ✅ 完成 | ✅ 通过 | ✅ 支持 |
| HTTP中间件 | ✅ 完成 | ✅ 通过 | ✅ 支持 |
| 上下文追踪 | ✅ 完成 | ✅ 通过 | ✅ 支持 |

### Cache 包状态

| 功能模块 | 实现状态 | 测试状态 | 并发安全 |
|----------|----------|----------|----------|
| 内存缓存(LRU/LFU) | ✅ 完成 | ✅ 通过 | ✅ 安全 |
| Redis缓存 | ✅ 完成 | ✅ 通过 | ✅ 安全 |
| 多级缓存 | ✅ 完成 | ✅ 通过 | ✅ 安全 |
| 分布式缓存 | ✅ 完成 | ✅ 通过 | ✅ 安全 |
| 监控统计 | ✅ 完成 | ✅ 通过 | ✅ 安全 |
| 缓存管理器 | ✅ 完成 | ✅ 通过 | ✅ 安全 |

### Config 包状态

| 功能模块 | 实现状态 | 测试状态 | 企业特性 |
|----------|----------|----------|----------|
| 基础配置管理 | ✅ 完成 | ✅ 通过 | ✅ 支持 |
| 多格式支持 | ✅ 完成 | ✅ 通过 | ✅ 支持 |
| 环境变量集成 | ✅ 完成 | ✅ 通过 | ✅ 支持 |
| 配置加密 | ✅ 完成 | ✅ 通过 | ✅ 支持 |
| 多环境管理 | ✅ 完成 | ✅ 通过 | ✅ 支持 |
| 配置验证 | ✅ 完成 | ✅ 通过 | ✅ 支持 |
| 热重载机制 | ✅ 完成 | ✅ 通过 | ✅ 支持 |
| 工厂模式 | ✅ 完成 | ✅ 通过 | ✅ 支持 |

### Monitor 包状态

| 功能模块 | 实现状态 | 测试状态 | 并发安全 |
|----------|----------|----------|----------|
| 系统资源监控 | ✅ 完成 | ✅ 通过 | ✅ 安全 |
| 网络监控 | ✅ 完成 | ✅ 通过 | ✅ 安全 |
| 进程监控 | ✅ 完成 | ✅ 通过 | ✅ 安全 |
| 数据聚合分析 | ✅ 完成 | ✅ 通过 | ✅ 安全 |
| 智能告警系统 | ✅ 完成 | ✅ 通过 | ✅ 安全 |
| 数据持久化 | ✅ 完成 | ✅ 通过 | ✅ 安全 |
| 监控器接口 | ✅ 完成 | ✅ 通过 | ✅ 安全 |
| 可扩展架构 | ✅ 完成 | ✅ 通过 | ✅ 安全 |

### Chi 包状态

| 功能模块 | 实现状态 | 测试状态 | 企业特性 |
|----------|----------|----------|----------|
| 基础Web框架 | ✅ 完成 | ✅ 通过 | ✅ 支持 |
| WebSocket支持 | ✅ 完成 | ✅ 通过 | ✅ 支持 |
| 插件系统 | ✅ 完成 | ✅ 通过 | ✅ 支持 |
| 服务发现 | ✅ 完成 | ✅ 通过 | ✅ 支持 |
| 熔断器 | ✅ 完成 | ✅ 通过 | ✅ 支持 |
| 限流控制 | ✅ 完成 | ✅ 通过 | ✅ 支持 |
| 舱壁模式 | ✅ 完成 | ✅ 通过 | ✅ 支持 |
| 分布式追踪 | ✅ 完成 | ✅ 通过 | ✅ 支持 |
| 缓存系统 | ✅ 完成 | ✅ 通过 | ✅ 支持 |
| 动态配置 | ✅ 完成 | ✅ 通过 | ✅ 支持 |
| 功能开关 | ✅ 完成 | ✅ 通过 | ✅ 支持 |
| 事件系统 | ✅ 完成 | ✅ 通过 | ✅ 支持 |
| 管理控制台 | ✅ 完成 | ✅ 通过 | ✅ 支持 |

### 规划中的包

- **Server 包** - HTTP/gRPC 服务框架
- **Message 包** - 消息队列封装
- **Middleware 包** - 通用中间件

## 🤝 贡献指南

### 贡献流程

1. Fork 项目
2. 创建功能分支: `git checkout -b feature/new-package`
3. 提交更改: `git commit -am 'Add new package'`
4. 推送分支: `git push origin feature/new-package`
5. 提交 Pull Request

### 代码规范

- 遵循 Go 官方代码规范
- 使用 `gofmt` 格式化代码
- 通过 `go vet` 静态检查
- 添加必要的单元测试
- 更新相关文档

### 提交消息格式

```
<type>(<scope>): <subject>

<body>

<footer>
```

类型说明：
- `feat`: 新功能
- `fix`: 修复 bug
- `docs`: 文档更新
- `style`: 代码格式调整
- `refactor`: 重构代码
- `test`: 测试相关
- `chore`: 构建/工具相关

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

## 🔗 相关链接

- [Database 包文档](pkg/database/README.md)
- [Cache 包文档](pkg/cache/README.md)
- [Config 包文档](pkg/config/README.md)
- [Logger 包文档](pkg/logger/)
- [Monitor 包文档](pkg/monitor/README.md)
- [Chi 框架文档](pkg/chi/README.md) ⭐ **新增**
- [Chi 更新日志](pkg/chi/CHANGELOG.md)
- [并发安全指南](CONCURRENT_SAFETY.md)
- [变更日志](CHANGELOG.md)
- [Go Workspace 官方文档](https://go.dev/doc/tutorial/workspaces)

## 📞 支持与反馈

如遇问题或有建议，欢迎：

- 提交 [Issues](../../issues)
- 发起 [Pull Requests](../../pulls)
- 参与 [Discussions](../../discussions)

---

**⚡ 高性能 • 🔒 线程安全 • 📦 模块化 • 🚀 生产就绪**