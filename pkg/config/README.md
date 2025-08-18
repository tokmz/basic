# 配置管理包 (Config Package)

基于 Viper 的高性能配置管理包，支持多种配置格式、环境变量覆盖、配置热重载、远程配置中心等功能。

## 特性

- ✅ **多格式支持**: YAML、JSON、TOML、INI 等多种配置文件格式
- ✅ **环境变量覆盖**: 自动读取环境变量，支持优先级覆盖（环境变量 > 配置文件 > 默认值）
- ✅ **多级路径搜索**: 支持多个配置文件搜索路径
- ✅ **配置热重载**: 文件变更监听，自动重新加载配置
- ✅ **远程配置中心**: 支持 etcd、Consul 等远程配置中心（可扩展）
- ✅ **工厂模式**: 支持创建不同环境的配置实例
- ✅ **结构体绑定**: 自动将配置绑定到 Go 结构体
- ✅ **调试模式**: 详细的配置加载过程日志
- ✅ **线程安全**: 并发安全的配置读写操作
- ✅ **配置验证**: 内置配置验证规则
- ✅ **性能优化**: 针对大配置文件的处理优化

## 快速开始

### 安装

```bash
go get basic/pkg/config
```

### 基础用法

```go
package main

import (
    "fmt"
    "log"
    "basic/pkg/config"
)

func main() {
    // 创建配置管理器
    cfg, err := config.NewManager(nil) // 使用默认选项
    if err != nil {
        log.Fatal(err)
    }
    defer cfg.Close()

    // 设置默认值
    cfg.SetDefault("server.port", 8080)
    cfg.SetDefault("app.name", "我的应用")

    // 获取配置值
    port := cfg.GetInt("server.port")
    appName := cfg.GetString("app.name")

    fmt.Printf("应用: %s, 端口: %d\n", appName, port)
}
```

### 高级配置

```go
package main

import (
    "strings"
    "basic/pkg/config"
)

func main() {
    opts := &config.Options{
        Environment:     config.Development,
        ConfigName:      "app",
        ConfigType:      "yaml",
        ConfigPaths:     []string{".", "./config", "$HOME/.config"},
        EnvPrefix:       "APP",
        EnvKeyReplacer:  strings.NewReplacer(".", "_"),
        Debug:           true,
        Defaults: map[string]interface{}{
            "server.host": "localhost",
            "server.port": 8080,
        },
        RemoteProviders: []config.RemoteProvider{
            {
                Provider: "etcd",
                Endpoint: "localhost:2379",
                Path:     "/config/app",
            },
        },
    }

    cfg, err := config.NewManager(opts)
    if err != nil {
        log.Fatal(err)
    }
    defer cfg.Close()

    // 读取配置文件
    if err := cfg.ReadInConfig(); err != nil {
        log.Printf("读取配置文件失败: %v", err)
    }

    // 启用配置监听
    cfg.WatchConfig()
    cfg.OnConfigChange(func() {
        fmt.Println("配置已更新")
    })
}
```

## API 文档

### 核心接口

#### Config 接口

```go
type Config interface {
    // 基础配置操作
    Get(key string) interface{}
    GetString(key string) string
    GetInt(key string) int
    GetInt64(key string) int64
    GetFloat64(key string) float64
    GetBool(key string) bool
    GetStringSlice(key string) []string
    GetStringMap(key string) map[string]interface{}
    GetStringMapString(key string) map[string]string
    GetTime(key string) time.Time
    GetDuration(key string) time.Duration

    // 设置配置
    Set(key string, value interface{})
    SetDefault(key string, value interface{})

    // 结构体绑定
    Unmarshal(rawVal interface{}) error
    UnmarshalKey(key string, rawVal interface{}) error

    // 配置文件操作
    ReadInConfig() error
    WatchConfig()
    OnConfigChange(run func())

    // 远程配置
    AddRemoteProvider(provider, endpoint, path string) error
    ReadRemoteConfig() error
    WatchRemoteConfig() error

    // 调试和状态
    Debug() bool
    SetDebug(debug bool)
    GetConfigFile() string
    AllSettings() map[string]interface{}

    // 生命周期
    Close() error
}
```

### 配置选项

#### Options 结构体

```go
type Options struct {
    Environment     Environment               // 环境类型 (dev/test/prod)
    ConfigName      string                   // 配置文件名（不含扩展名）
    ConfigType      string                   // 配置文件类型 (yaml/json/toml/ini)
    ConfigPaths     []string                 // 配置文件搜索路径
    RemoteProviders []RemoteProvider         // 远程配置提供者
    Debug           bool                     // 调试模式
    Defaults        map[string]interface{}   // 默认配置值
    EnvPrefix       string                   // 环境变量前缀
    EnvKeyReplacer  *strings.Replacer        // 环境变量键名替换器
}
```

#### 默认选项

```go
func DefaultOptions() *Options {
    return &Options{
        Environment: Development,
        ConfigName:  "config",
        ConfigType:  "yaml",
        ConfigPaths: []string{
            ".",
            "./config",
            "./configs",
            "$HOME/.config",
            "/etc",
        },
        Debug:          false,
        Defaults:       make(map[string]interface{}),
        EnvKeyReplacer: strings.NewReplacer(".", "_"),
    }
}
```

### 工厂模式

#### Factory 接口

```go
type Factory interface {
    CreateConfig(env Environment, opts *Options) (Config, error)
    GetConfig(env Environment) (Config, bool)
    ListEnvironments() []Environment
    CloseAll() error
}
```

#### 使用示例

```go
// 获取全局工厂
factory := config.GetGlobalFactory()

// 创建不同环境的配置
devConfig, err := factory.CreateConfig(config.Development, &config.Options{
    ConfigName: "app",
    Debug:      true,
})

prodConfig, err := factory.CreateConfig(config.Production, &config.Options{
    ConfigName: "app",
    Debug:      false,
})

// 获取已创建的配置
if cfg, exists := factory.GetConfig(config.Development); exists {
    fmt.Println("开发环境配置已存在")
}

// 关闭所有配置
factory.CloseAll()
```

### 远程配置

#### RemoteProvider 结构体

```go
type RemoteProvider struct {
    Provider string // 提供者类型 (etcd/consul/mock)
    Endpoint string // 连接端点
    Path     string // 配置路径
}
```

#### 使用示例

```go
// 添加远程配置提供者
cfg.AddRemoteProvider("etcd", "localhost:2379", "/config/app")

// 读取远程配置
if err := cfg.ReadRemoteConfig(); err != nil {
    log.Printf("读取远程配置失败: %v", err)
}

// 监听远程配置变化
if err := cfg.WatchRemoteConfig(); err != nil {
    log.Printf("监听远程配置失败: %v", err)
}
```

### 结构体绑定

#### 配置结构体示例

```go
type AppConfig struct {
    Server   ServerConfig   `mapstructure:"server"`
    Database DatabaseConfig `mapstructure:"database"`
    Redis    RedisConfig    `mapstructure:"redis"`
}

type ServerConfig struct {
    Host         string        `mapstructure:"host"`
    Port         int           `mapstructure:"port"`
    ReadTimeout  time.Duration `mapstructure:"read_timeout"`
    WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type DatabaseConfig struct {
    Driver   string `mapstructure:"driver"`
    DSN      string `mapstructure:"dsn"`
    MaxConns int    `mapstructure:"max_conns"`
}
```

#### 绑定示例

```go
// 绑定整个配置
var appConfig AppConfig
if err := cfg.Unmarshal(&appConfig); err != nil {
    log.Fatal(err)
}

// 绑定部分配置
var serverConfig ServerConfig
if err := cfg.UnmarshalKey("server", &serverConfig); err != nil {
    log.Fatal(err)
}
```

### 配置验证

#### 验证规则

```go
// 创建验证器
validator := config.NewValidator()

// 添加验证规则
validator.AddRule("server.port", config.NewRangeRule(1, 65535))
validator.AddRule("app.name", config.NewRequiredRule())
validator.AddRule("database.driver", config.NewEnumRule([]string{"mysql", "postgres", "sqlite"}))

// 验证配置
if err := validator.Validate(cfg.AllSettings()); err != nil {
    log.Printf("配置验证失败: %v", err)
}
```

## 配置文件示例

### YAML 格式 (config.yaml)

```yaml
app:
  name: "我的应用"
  version: "1.0.0"
  debug: false

server:
  host: "localhost"
  port: 8080
  read_timeout: "30s"
  write_timeout: "30s"
  tls:
    enabled: false
    cert_file: ""
    key_file: ""

database:
  driver: "mysql"
  dsn: "user:password@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"
  max_open_conns: 100
  max_idle_conns: 10
  conn_max_lifetime: "1h"

redis:
  addr: "localhost:6379"
  password: ""
  db: 0

logging:
  level: "info"
  format: "json"
  output: "stdout"
```

### JSON 格式 (config.json)

```json
{
  "app": {
    "name": "我的应用",
    "version": "1.0.0",
    "debug": false
  },
  "server": {
    "host": "localhost",
    "port": 8080,
    "read_timeout": "30s",
    "write_timeout": "30s"
  },
  "database": {
    "driver": "mysql",
    "dsn": "user:password@tcp(localhost:3306)/dbname",
    "max_open_conns": 100
  }
}
```

### TOML 格式 (config.toml)

```toml
[app]
name = "我的应用"
version = "1.0.0"
debug = false

[server]
host = "localhost"
port = 8080
read_timeout = "30s"
write_timeout = "30s"

[database]
driver = "mysql"
dsn = "user:password@tcp(localhost:3306)/dbname"
max_open_conns = 100
```

## 环境变量

### 环境变量命名规则

配置键 `server.port` 对应环境变量 `APP_SERVER_PORT`（假设前缀为 `APP`）

### 环境变量示例

```bash
# 应用配置
export APP_APP_NAME="生产环境应用"
export APP_APP_DEBUG="false"

# 服务器配置
export APP_SERVER_HOST="0.0.0.0"
export APP_SERVER_PORT="80"

# 数据库配置
export APP_DATABASE_DSN="user:pass@tcp(prod-db:3306)/prod_db"
export APP_DATABASE_MAX_OPEN_CONNS="200"

# Redis 配置
export APP_REDIS_ADDR="redis-cluster:6379"
export APP_REDIS_PASSWORD="secret"
```

## 最佳实践

### 1. 配置文件组织

```
project/
├── config/
│   ├── config.yaml          # 默认配置
│   ├── config.dev.yaml      # 开发环境配置
│   ├── config.test.yaml     # 测试环境配置
│   └── config.prod.yaml     # 生产环境配置
├── pkg/
└── main.go
```

### 2. 环境特定配置

```go
func createConfig(env string) (config.Config, error) {
    opts := &config.Options{
        Environment: config.Environment(env),
        ConfigName:  "config",
        ConfigType:  "yaml",
        ConfigPaths: []string{"./config", "."},
        EnvPrefix:   "APP",
        Debug:       env == "dev",
    }
    
    return config.NewManager(opts)
}
```

### 3. 配置验证

```go
func validateConfig(cfg config.Config) error {
    validator := config.NewValidator()
    
    // 必需配置
    validator.AddRule("server.port", config.NewRangeRule(1, 65535))
    validator.AddRule("database.dsn", config.NewRequiredRule())
    
    // 枚举值验证
    validator.AddRule("logging.level", 
        config.NewEnumRule([]string{"debug", "info", "warn", "error"}))
    
    return validator.Validate(cfg.AllSettings())
}
```

### 4. 优雅关闭

```go
func main() {
    cfg, err := config.NewManager(opts)
    if err != nil {
        log.Fatal(err)
    }
    
    // 注册优雅关闭
    defer func() {
        if err := cfg.Close(); err != nil {
            log.Printf("关闭配置管理器失败: %v", err)
        }
    }()
    
    // 应用逻辑...
}
```

### 5. 配置热重载处理

```go
func setupConfigReload(cfg config.Config, app *App) {
    cfg.WatchConfig()
    cfg.OnConfigChange(func() {
        log.Println("配置文件已更新，重新加载...")
        
        // 重新绑定配置
        var newConfig AppConfig
        if err := cfg.Unmarshal(&newConfig); err != nil {
            log.Printf("重新加载配置失败: %v", err)
            return
        }
        
        // 更新应用配置
        app.UpdateConfig(newConfig)
        log.Println("配置重新加载完成")
    })
}
```

## 性能优化

### 1. 配置缓存

配置管理器内部使用读写锁，支持高并发读取操作。

### 2. 大配置文件处理

- 使用流式解析，避免一次性加载整个文件到内存
- 支持配置分片，按需加载
- 内置配置压缩支持

### 3. 远程配置优化

- 连接池管理
- 配置缓存和增量更新
- 故障转移和重试机制

## 故障排除

### 常见问题

1. **配置文件未找到**
   ```
   [CONFIG] Config file not found, using defaults and environment variables
   ```
   解决方案：检查配置文件路径和文件名是否正确。

2. **环境变量未生效**
   ```
   确保环境变量名称符合命名规则：前缀_键名（用下划线替换点号）
   ```

3. **远程配置连接失败**
   ```
   [CONFIG] Failed to connect to remote config: connection refused
   ```
   解决方案：检查远程配置中心是否可访问，网络连接是否正常。

4. **配置验证失败**
   ```
   [CONFIG] Validation failed: server.port must be between 1 and 65535
   ```
   解决方案：检查配置值是否符合验证规则。

### 调试模式

启用调试模式可以查看详细的配置加载过程：

```go
opts := &config.Options{
    Debug: true,
}

cfg, err := config.NewManager(opts)
// 或者动态启用
cfg.SetDebug(true)
```

调试输出示例：
```
[CONFIG] Added config path: .
[CONFIG] Added config path: ./config
[CONFIG] Looking for config file: config.dev.yaml
[CONFIG] Loaded config file: ./config/config.dev.yaml
[CONFIG] Get key=server.port, value=8080
[CONFIG] Set key=runtime.start_time, value=2024-01-01T12:00:00Z
```

## 贡献指南

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 打开 Pull Request

## 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 更新日志

### v1.0.0
- 初始版本发布
- 支持多种配置格式
- 环境变量覆盖
- 配置热重载
- 远程配置中心支持
- 工厂模式
- 结构体绑定
- 配置验证
- 调试模式