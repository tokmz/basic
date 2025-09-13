# Config Package - 企业级配置管理系统

基于 Viper 的高性能企业级配置管理包，提供完整的配置管理解决方案，支持多环境、加密、验证、热重载等企业级特性。

## 🌟 核心特性

### 🚀 高性能配置管理
- **基于 Viper** - 使用成熟稳定的 Viper 库作为底层引擎
- **多格式支持** - 支持 YAML、JSON、TOML、HCL、INI 等配置格式
- **环境变量集成** - 无缝集成环境变量，支持前缀和键映射
- **远程配置支持** - 支持 etcd、Consul、Firestore 等远程配置源

### 🔐 安全加密功能
- **AES 加密** - 内置 AES-GCM 加密算法保护敏感配置
- **选择性加密** - 可指定特定配置项进行加密
- **透明解密** - 自动加解密，对业务代码透明
- **密钥管理** - 支持配置密钥轮换和管理

### 🌍 多环境管理
- **环境自动检测** - 自动检测当前运行环境
- **环境特定配置** - 支持不同环境使用不同配置文件
- **配置继承** - 支持配置层级和继承机制
- **环境切换** - 支持运行时环境切换

### ✅ 配置验证
- **类型安全** - 强类型配置验证
- **自定义规则** - 支持自定义验证规则
- **范围检查** - 支持数值范围和选项验证
- **必填项检查** - 支持必填配置项验证

### 🔄 热重载机制
- **文件监听** - 监听配置文件变化并自动重载
- **事件通知** - 配置变化事件通知机制
- **回调支持** - 支持配置变化回调函数
- **原子更新** - 保证配置更新的原子性

### 🏭 工厂模式
- **多种管理器** - 支持基础、加密、环境等多种管理器类型
- **自动检测** - 根据配置自动选择合适的管理器类型
- **注册机制** - 支持管理器注册和生命周期管理
- **预设工厂** - 提供开发、测试、生产环境预设

## 📦 安装

```bash
go get github.com/tokmz/basic/pkg/config
```

## 🚀 快速开始

### 基本使用

```go
package main

import (
    "fmt"
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

### Builder模式使用

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

### 快速启动

```go
// 最简单的使用方式
manager, err := config.QuickStart(
    config.WithYAMLFile("./config.yaml"),
    config.WithDevelopment(),
)
if err != nil {
    log.Fatal(err)
}
defer manager.Close()
```

## 🔐 加密配置使用

### 基本加密配置

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

if err != nil {
    log.Fatal(err)
}
defer manager.Close()

// 设置敏感信息（自动加密）
manager.SetString("database.password", "super-secret-password")

// 读取配置（自动解密）
password := manager.GetString("database.password")
fmt.Println("Password:", password)
```

### 支持的加密标记

在配置文件中可以使用特殊标记：

```yaml
database:
  host: localhost
  port: 5432
  # 运行时加密
  password: "${SECRET:actual_password}"
  
api:
  # 已加密的值
  secret: "${ENCRYPTED:base64_encrypted_value}"
```

## 🌍 多环境配置

### 环境检测和管理

```go
// 自动检测当前环境
detector := config.NewEnvironmentDetector()
currentEnv := detector.DetectEnvironment()

// 创建环境管理器
envManager := config.NewEnvironmentManager(currentEnv)

// 为不同环境设置配置路径
envManager.AddConfigPath(config.Development, "./config/dev")
envManager.AddConfigPath(config.Production, "./config/prod")
envManager.AddConfigPath(config.Testing, "./config/test")

// 加载环境特定配置
manager, err := envManager.LoadWithEnvironment(currentEnv, nil)
if err != nil {
    log.Fatal(err)
}
defer manager.Close()
```

### 环境配置文件结构

```
config/
├── dev/
│   ├── config.yaml
│   └── database.yaml
├── prod/
│   ├── config.yaml
│   └── database.yaml
└── test/
    ├── config.yaml
    └── database.yaml
```

## ✅ 配置验证

### 创建验证规则

```go
validator := config.NewDefaultValidator()

// 添加验证规则
validator.AddRule(config.ValidationRule{
    Key:      "server.port",
    Required: true,
    Type:     "int",
    MinValue: 1024,
    MaxValue: 65535,
})

validator.AddRule(config.ValidationRule{
    Key:     "log.level",
    Type:    "string",
    Options: []interface{}{"debug", "info", "warn", "error"},
})

// 自定义验证函数
validator.AddRule(config.ValidationRule{
    Key:      "timeout",
    Type:     "duration",
    CustomFunc: func(value interface{}) error {
        if dur, ok := value.(time.Duration); ok {
            if dur < time.Second || dur > time.Hour {
                return fmt.Errorf("timeout must be between 1s and 1h")
            }
        }
        return nil
    },
})

// 验证配置
if err := manager.Validate(validator); err != nil {
    log.Printf("Config validation failed: %v", err)
}
```

## 🔄 配置热重载

### 监听配置变化

```go
// 启用配置文件监听
config := config.DevelopmentConfig()
config.WatchConfig = true

manager := config.NewViperManager(config)
manager.Load()

// 添加变化监听器
manager.Watch(func(event config.Event) {
    fmt.Printf("Config changed: %s = %v (type: %s)\n", 
        event.Key, event.Value, event.Type)
        
    // 根据配置变化执行相应操作
    switch event.Key {
    case "log.level":
        // 更新日志级别
        updateLogLevel(event.Value.(string))
    case "database.pool_size":
        // 调整数据库连接池大小
        adjustDBPool(event.Value.(int))
    }
})
```

## 📊 结构体配置

### 配置解析到结构体

```go
type DatabaseConfig struct {
    Host     string        `mapstructure:"host"`
    Port     int           `mapstructure:"port"`
    Username string        `mapstructure:"username"`
    Password string        `mapstructure:"password"`
    Timeout  time.Duration `mapstructure:"timeout"`
}

type AppConfig struct {
    Debug    bool           `mapstructure:"debug"`
    Port     int            `mapstructure:"port"`
    Database DatabaseConfig `mapstructure:"database"`
}

// 解析整个配置
var appConfig AppConfig
if err := manager.Unmarshal(&appConfig); err != nil {
    log.Fatal(err)
}

// 只解析特定部分
var dbConfig DatabaseConfig
if err := manager.UnmarshalKey("database", &dbConfig); err != nil {
    log.Fatal(err)
}
```

## 🏭 工厂模式使用

### 管理器类型

```go
// 基础Viper管理器
basicManager, err := config.CreateManager(config.ViperManagerType, config.DefaultConfig())

// 加密管理器
encryptedManager, err := config.CreateManagerWithOptions(
    config.EncryptedManagerType, 
    encConfig, 
    []string{"secret.key"},
)

// 环境管理器
envManager, err := config.CreateManagerWithOptions(
    config.EnvironmentManagerType, 
    nil, 
    config.Production,
)

// 自动检测类型
autoManager, err := config.CreateAuto(someConfig)
```

### 管理器注册表

```go
registry := config.GetGlobalRegistry()

// 注册管理器
registry.Register("app", appManager)
registry.Register("cache", cacheManager)

// 获取管理器
if manager, exists := registry.Get("app"); exists {
    port := manager.GetInt("server.port")
}

// 获取或创建管理器
manager, err := registry.GetOrCreate(
    "database", 
    config.ViperManagerType, 
    dbConfig,
)
```

## 🛠️ 高级特性

### 预设工厂

```go
factory := config.GetPresetFactory()

// 快速创建不同环境的管理器
devManager, _ := factory.CreateDevelopment("./config/dev")
prodManager, _ := factory.CreateProduction("./config/prod")
testManager, _ := factory.CreateTesting("./config/test")

// 创建加密管理器
encManager, _ := factory.CreateEncrypted(
    "encryption-key",
    []string{"db.password"},
    nil,
)
```

### 错误处理

```go
// 自定义错误处理器
errorHandler := config.NewDefaultErrorHandler(func(format string, args ...interface{}) {
    log.Printf("[CONFIG] "+format, args...)
})

// 处理配置错误
err := manager.Load()
if err != nil {
    errorHandler.HandleError(err)
    
    // 检查错误类型
    if config.IsConfigError(err) {
        fmt.Println("This is a config error")
    }
    
    if config.IsConfigErrorWithCode(err, "CONFIG_NOT_FOUND") {
        fmt.Println("Config file not found")
    }
}
```

## 📝 配置文件示例

### 应用配置 (config.yaml)

```yaml
# 应用基本配置
app:
  name: "MyApp"
  version: "1.0.0"
  debug: true

# 服务器配置
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: "30s"
  write_timeout: "30s"

# 数据库配置
database:
  host: "localhost"
  port: 5432
  username: "app_user"
  password: "${SECRET:db_password_here}"
  ssl_mode: "require"
  max_connections: 20

# Redis配置
redis:
  host: "localhost"
  port: 6379
  db: 0
  password: "${SECRET:redis_password_here}"

# 日志配置
log:
  level: "info"
  format: "json"
  output: "stdout"

# API配置
api:
  secret_key: "${SECRET:api_secret_key_here}"
  rate_limit: 1000
  timeout: "10s"
```

### 环境变量配置

```bash
# 应用环境变量
export MYAPP_APP_NAME="MyApp Production"
export MYAPP_SERVER_PORT=80
export MYAPP_DATABASE_HOST="prod-db.example.com"
export MYAPP_LOG_LEVEL="warn"
```

## 🧪 测试

```bash
# 运行所有测试
go test ./...

# 运行特定测试
go test -v -run TestConfig

# 运行基准测试
go test -bench=.

# 生成测试覆盖率报告
go test -cover ./...
```

## 📚 API文档

### 核心接口

#### Manager接口
```go
type Manager interface {
    Load() error
    Reload() error
    Get(key string) interface{}
    GetString(key string) string
    GetInt(key string) int
    GetBool(key string) bool
    GetFloat64(key string) float64
    GetDuration(key string) time.Duration
    GetStringSlice(key string) []string
    Set(key string, value interface{})
    IsSet(key string) bool
    Unmarshal(rawVal interface{}) error
    UnmarshalKey(key string, rawVal interface{}) error
    Watch(callback func(event Event)) error
    StopWatch()
    Validate(validator Validator) error
    GetAllSettings() map[string]interface{}
    Close() error
}
```

#### 全局函数
```go
// 初始化
func Init(config *Config) error
func InitWithEncryption(config *Config, encryptedKeys []string) error
func InitWithEnvironment(env Environment, baseConfig *Config) error

// 读取配置
func Get(key string) interface{}
func GetString(key string) string
func GetInt(key string) int
func GetBool(key string) bool
func GetFloat64(key string) float64
func GetDuration(key string) time.Duration
func GetStringSlice(key string) []string

// 设置配置
func Set(key string, value interface{})
func IsSet(key string) bool

// 结构体解析
func Unmarshal(rawVal interface{}) error
func UnmarshalKey(key string, rawVal interface{}) error

// 监听和验证
func Watch(callback func(event Event)) error
func Validate(validator Validator) error

// 生命周期
func Reload() error
func Close() error
```

## 🤝 贡献指南

### 开发环境设置

```bash
# 克隆仓库
git clone <repository-url>
cd basic/pkg/config

# 安装依赖
go mod tidy

# 运行测试
go test ./...
```

### 代码规范

- 遵循 Go 官方代码规范
- 使用 `gofmt` 格式化代码
- 通过 `go vet` 静态检查
- 添加必要的单元测试
- 更新相关文档

### 提交规范

```
<type>(<scope>): <subject>

<body>

<footer>
```

类型说明：
- `feat`: 新功能
- `fix`: Bug修复
- `docs`: 文档更新
- `test`: 测试相关
- `refactor`: 重构代码

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

## 🔗 相关链接

- [Viper 官方文档](https://github.com/spf13/viper)
- [Go Modules 文档](https://golang.org/ref/mod)
- [项目主页](../../README.md)

## 📞 支持与反馈

- 提交 Issues
- 参与 Discussions
- 贡献代码

---

**🚀 高性能 • 🔒 安全可靠 • 🌍 多环境 • ✅ 类型安全 • 🔄 热重载**