# Logger - 企业级Go日志库

基于Zap的高性能、功能丰富的企业级日志库，支持日志分割、多种输出格式、钩子机制等高级功能。

## 特性

- 🚀 **高性能**: 基于uber-go/zap，零分配高性能日志记录
- 📁 **日志分割**: 基于大小、时间的自动日志轮转和压缩
- 🎯 **多输出**: 支持控制台、文件同时输出
- 🔧 **多格式**: JSON和Console两种输出格式
- 📊 **结构化**: 支持结构化字段和上下文日志
- 🎣 **钩子机制**: 支持邮件、Webhook、数据库等告警钩子
- 🌐 **HTTP中间件**: 内置HTTP请求日志中间件
- ⚙️ **灵活配置**: 支持开发、生产等预设配置
- 🔄 **动态级别**: 运行时动态调整日志级别
- 🧵 **并发安全**: 完全的并发安全设计

## 快速开始

### 安装

```bash
go get github.com/tokmz/basic/pkg/logger
```

### 基本使用

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
}
```

## 配置

### 预设配置

```go
// 开发环境配置 - 彩色控制台输出，Debug级别
config := logger.DevelopmentConfig()

// 生产环境配置 - JSON格式文件输出，Info级别
config := logger.ProductionConfig("my-service", "production")

// 默认配置
config := logger.DefaultConfig()
```

### 自定义配置

```go
config := &logger.Config{
    Level:        logger.InfoLevel,
    Format:       logger.JSONFormat,
    EnableCaller: true,
    EnableStack:  true,
    TimeFormat:   time.RFC3339,
    Console: logger.ConsoleConfig{
        Enabled:    true,
        ColoredOut: true,
    },
    File: logger.FileConfig{
        Enabled:  true,
        Filename: "./logs/app.log",
        Rotation: logger.RotationConfig{
            MaxSize:    100, // 100MB
            MaxAge:     30,  // 30天
            MaxBackups: 10,  // 10个备份
            Compress:   true,
        },
    },
    ServiceName: "my-service",
    ServiceEnv:  "production",
}

logger, err := logger.New(config)
```

## 日志级别

支持以下日志级别：
- `DEBUG` - 调试信息
- `INFO` - 常规信息  
- `WARN` - 警告信息
- `ERROR` - 错误信息
- `PANIC` - 严重错误（会panic）
- `FATAL` - 致命错误（会exit）

```go
logger.Debug("调试信息")
logger.Info("常规信息")
logger.Warn("警告信息")
logger.Error("错误信息")

// 格式化日志
logger.Infof("用户 %s 登录成功", username)
logger.Errorf("数据库连接失败: %v", err)
```

## 结构化日志

### 使用zap.Field

```go
logger.Info("用户操作",
    zap.String("user_id", "12345"),
    zap.String("action", "create_order"),
    zap.Float64("amount", 99.99),
    zap.Duration("duration", time.Millisecond*150),
)
```

### 使用Fields映射

```go
logger.WithFields(logger.Fields{
    "user_id": "12345",
    "action":  "login",
    "ip":      "192.168.1.100",
}).Info("用户登录")
```

## 上下文日志

```go
// 创建带追踪信息的上下文
ctx := context.Background()
ctx = logger.WithTraceID(ctx, "trace-123")
ctx = logger.WithRequestID(ctx, "req-456")
ctx = logger.WithUserID(ctx, "user-789")

// 使用上下文记录日志
logger.WithContext(ctx).Info("处理用户请求")
```

## 日志分割

自动基于大小和时间进行日志分割：

```go
config.File.Rotation = logger.RotationConfig{
    MaxSize:    100, // 单文件最大100MB
    MaxAge:     30,  // 保留30天
    MaxBackups: 10,  // 最多10个备份文件
    Compress:   true, // 压缩旧文件
}
```

## 钩子机制

支持在特定日志级别触发自定义钩子：

### 邮件告警钩子

```go
emailHook := logger.NewEmailHook(
    "smtp.example.com", 587,
    "app@example.com", "password",
    []string{"admin@example.com"},
)

hookLogger, err := logger.NewWithHooks(config, emailHook)
```

### Webhook钩子

```go
webhookHook := logger.NewWebhookHook("https://webhook.example.com/alert")
hookLogger, err := logger.NewWithHooks(config, webhookHook)
```

### 自定义钩子

```go
type CustomHook struct{}

func (h *CustomHook) Levels() []logger.LogLevel {
    return []logger.LogLevel{logger.ErrorLevel}
}

func (h *CustomHook) Fire(entry *logger.Entry) error {
    // 自定义处理逻辑
    fmt.Printf("Custom hook fired: %s\n", entry.Message)
    return nil
}
```

## HTTP中间件

内置HTTP请求日志中间件：

```go
import (
    "net/http"
    "github.com/tokmz/basic/pkg/logger"
)

func main() {
    logger, _ := logger.New(logger.DefaultConfig())
    
    mux := http.NewServeMux()
    mux.HandleFunc("/", handler)
    
    // 添加日志中间件
    handler := logger.HTTPMiddleware(logger)(mux)
    handler = logger.RecoveryMiddleware(logger)(handler)
    
    http.ListenAndServe(":8080", handler)
}
```

## 性能

该日志库基于zap构建，具有优异的性能特征：

```
BenchmarkLogger_Info              1000000    1500 ns/op      0 allocs/op
BenchmarkLogger_InfoWithFields     500000    2100 ns/op      0 allocs/op
BenchmarkGlobal_Info              1000000    1600 ns/op      0 allocs/op
```

## 错误处理

内置错误类型和错误码：

```go
if err != nil {
    if loggerErr, ok := err.(*logger.LoggerError); ok {
        fmt.Printf("Logger error [%s]: %s\n", loggerErr.Code, loggerErr.Message)
    }
}
```

## 最佳实践

### 1. 使用全局日志器

```go
// 应用启动时初始化
logger.InitGlobal(config)

// 在任何地方使用
logger.Info("message")
logger.WithFields(fields).Error("error occurred")
```

### 2. 结构化日志字段

```go
// 好的实践 - 使用一致的字段名
logger.Info("API调用",
    zap.String("method", "GET"),
    zap.String("endpoint", "/api/users"),
    zap.Int("status_code", 200),
    zap.Duration("duration", duration),
)
```

### 3. 上下文传递

```go
func ProcessRequest(ctx context.Context) {
    logger := logger.WithContext(ctx)
    logger.Info("开始处理请求")
    
    // 传递给子函数
    processUser(ctx, userID)
}

func processUser(ctx context.Context, userID string) {
    logger.WithContext(ctx).WithFields(logger.Fields{
        "user_id": userID,
    }).Info("处理用户")
}
```

### 4. 错误日志

```go
if err != nil {
    logger.Error("操作失败",
        zap.Error(err),
        zap.String("operation", "create_user"),
        zap.String("user_id", userID),
    )
    return err
}
```

## 配置示例

### 开发环境

```yaml
level: debug
format: console
enable_caller: true
enable_stack: false
console:
  enabled: true
  colored_out: true
file:
  enabled: false
service_name: my-app
service_env: development
```

### 生产环境

```yaml
level: info
format: json
enable_caller: true
enable_stack: true
console:
  enabled: false
file:
  enabled: true
  filename: ./logs/app.log
  rotation:
    max_size: 100
    max_age: 30
    max_backups: 10
    compress: true
service_name: my-app
service_env: production
enable_sampler: true
```

## 依赖

- [go.uber.org/zap](https://github.com/uber-go/zap) - 高性能日志库
- [gopkg.in/natefinch/lumberjack.v2](https://github.com/natefinch/lumberjack) - 日志轮转

## 许可证

MIT License

## 贡献

欢迎提交Issue和Pull Request来改进这个项目。