# 插件系统开发指南

## 概述

Chi 框架提供了强大的插件系统，支持动态加载、卸载和管理插件。插件系统采用标准化的接口设计，提供完整的生命周期管理，使得功能扩展变得简单而安全。

## 插件架构

### 整体设计
```
┌──────────────────────────────────────────────┐
│              Plugin System                    │
│                                              │
│  ┌────────────┐    ┌─────────────────────┐   │
│  │Plugin      │    │  Plugin Registry    │   │
│  │Manager     │◄──▶│                     │   │
│  └────────────┘    └─────────────────────┘   │
│         │                      │             │
│         ▼                      ▼             │
│  ┌────────────┐    ┌─────────────────────┐   │
│  │Plugin      │    │   Dependency        │   │
│  │Loader      │    │   Resolver          │   │
│  └────────────┘    └─────────────────────┘   │
└──────────────────────────────────────────────┘
```

### 核心组件

1. **Plugin Manager**: 插件生命周期管理
2. **Plugin Registry**: 插件注册表，维护所有插件信息
3. **Plugin Loader**: 动态加载和卸载插件
4. **Dependency Resolver**: 插件依赖解析与管理

## 插件接口定义

### 基础插件接口

```go
// Plugin 定义插件的基本接口
type Plugin interface {
    // 获取插件信息
    Info() PluginInfo

    // 初始化插件
    Init(ctx context.Context, config Config) error

    // 启动插件
    Start(ctx context.Context) error

    // 停止插件
    Stop(ctx context.Context) error

    // 清理资源
    Close(ctx context.Context) error
}

// PluginInfo 插件基本信息
type PluginInfo struct {
    Name        string            `json:"name"`
    Version     string            `json:"version"`
    Description string            `json:"description"`
    Author      string            `json:"author"`
    Homepage    string            `json:"homepage"`
    License     string            `json:"license"`
    Tags        []string          `json:"tags"`
    Dependencies []Dependency     `json:"dependencies"`
    Config      ConfigSchema      `json:"config"`
}

// Dependency 插件依赖定义
type Dependency struct {
    Name    string `json:"name"`
    Version string `json:"version"`
    Type    string `json:"type"` // required, optional
}
```

### 扩展接口

```go
// MiddlewarePlugin HTTP中间件插件接口
type MiddlewarePlugin interface {
    Plugin
    Middleware() gin.HandlerFunc
}

// ServicePlugin 服务插件接口
type ServicePlugin interface {
    Plugin
    RegisterService(registry ServiceRegistry) error
}

// EventPlugin 事件插件接口
type EventPlugin interface {
    Plugin
    RegisterEvents(bus EventBus) error
}

// ConfigurablePlugin 可配置插件接口
type ConfigurablePlugin interface {
    Plugin
    Configure(config map[string]interface{}) error
    GetConfig() map[string]interface{}
}
```

## 开发插件

### 1. 创建基础插件

```go
package myplugin

import (
    "context"
    "github.com/your-org/basic/pkg/chi/plugin"
)

// MyPlugin 示例插件
type MyPlugin struct {
    name   string
    config Config
    logger Logger
}

// Info 返回插件信息
func (p *MyPlugin) Info() plugin.PluginInfo {
    return plugin.PluginInfo{
        Name:        "my-plugin",
        Version:     "1.0.0",
        Description: "示例插件",
        Author:      "Your Name",
        License:     "MIT",
        Tags:        []string{"example", "demo"},
        Dependencies: []plugin.Dependency{
            {
                Name:    "logger",
                Version: ">=1.0.0",
                Type:    "required",
            },
        },
    }
}

// Init 初始化插件
func (p *MyPlugin) Init(ctx context.Context, config plugin.Config) error {
    p.logger = config.GetLogger()
    p.logger.Info("MyPlugin initializing...")

    // 解析插件配置
    if err := config.Unmarshal(&p.config); err != nil {
        return fmt.Errorf("failed to parse config: %w", err)
    }

    return nil
}

// Start 启动插件
func (p *MyPlugin) Start(ctx context.Context) error {
    p.logger.Info("MyPlugin starting...")

    // 启动插件逻辑
    go p.backgroundWork(ctx)

    return nil
}

// Stop 停止插件
func (p *MyPlugin) Stop(ctx context.Context) error {
    p.logger.Info("MyPlugin stopping...")

    // 清理资源，停止后台任务

    return nil
}

// Close 关闭插件
func (p *MyPlugin) Close(ctx context.Context) error {
    p.logger.Info("MyPlugin closing...")

    // 最终清理

    return nil
}

func (p *MyPlugin) backgroundWork(ctx context.Context) {
    // 插件的后台工作逻辑
    for {
        select {
        case <-ctx.Done():
            return
        case <-time.After(time.Second * 10):
            p.logger.Debug("Background work tick")
        }
    }
}
```

### 2. 中间件插件示例

```go
package auth

import (
    "context"
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/your-org/basic/pkg/chi/plugin"
)

type AuthPlugin struct {
    plugin.BasePlugin
    jwtSecret string
}

func (p *AuthPlugin) Info() plugin.PluginInfo {
    return plugin.PluginInfo{
        Name:        "auth-middleware",
        Version:     "1.0.0",
        Description: "JWT认证中间件插件",
        Author:      "Chi Team",
        License:     "MIT",
        Tags:        []string{"auth", "jwt", "middleware"},
    }
}

func (p *AuthPlugin) Init(ctx context.Context, config plugin.Config) error {
    var authConfig struct {
        JWTSecret string `yaml:"jwt_secret"`
    }

    if err := config.Unmarshal(&authConfig); err != nil {
        return err
    }

    p.jwtSecret = authConfig.JWTSecret
    return nil
}

// Middleware 实现中间件接口
func (p *AuthPlugin) Middleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.JSON(http.StatusUnauthorized, gin.H{
                "error": "missing authorization header",
            })
            c.Abort()
            return
        }

        // JWT验证逻辑
        if !p.validateToken(token) {
            c.JSON(http.StatusUnauthorized, gin.H{
                "error": "invalid token",
            })
            c.Abort()
            return
        }

        c.Next()
    }
}

func (p *AuthPlugin) validateToken(token string) bool {
    // JWT验证实现
    return true
}

// 插件注册函数
func init() {
    plugin.Register("auth-middleware", &AuthPlugin{})
}
```

### 3. 服务插件示例

```go
package userservice

import (
    "context"
    "github.com/your-org/basic/pkg/chi/plugin"
)

type UserServicePlugin struct {
    plugin.BasePlugin
    db Database
}

func (p *UserServicePlugin) Info() plugin.PluginInfo {
    return plugin.PluginInfo{
        Name:        "user-service",
        Version:     "1.0.0",
        Description: "用户服务插件",
        Tags:        []string{"service", "user"},
        Dependencies: []plugin.Dependency{
            {
                Name: "database",
                Version: ">=1.0.0",
                Type: "required",
            },
        },
    }
}

func (p *UserServicePlugin) Init(ctx context.Context, config plugin.Config) error {
    // 获取数据库依赖
    db, err := config.GetDependency("database")
    if err != nil {
        return err
    }

    p.db = db.(Database)
    return nil
}

// RegisterService 注册服务到服务注册表
func (p *UserServicePlugin) RegisterService(registry plugin.ServiceRegistry) error {
    userService := &UserService{db: p.db}

    return registry.Register("user-service", userService)
}

type UserService struct {
    db Database
}

func (s *UserService) GetUser(ctx context.Context, id string) (*User, error) {
    // 用户查询逻辑
    return s.db.FindUser(ctx, id)
}

func (s *UserService) CreateUser(ctx context.Context, user *User) error {
    // 用户创建逻辑
    return s.db.CreateUser(ctx, user)
}
```

## 插件配置

### 配置文件示例

```yaml
# plugins.yaml
plugins:
  # 认证插件配置
  auth-middleware:
    enabled: true
    priority: 100
    config:
      jwt_secret: "your-secret-key"
      token_expiry: "24h"
      refresh_enabled: true

  # 限流插件配置
  rate-limiter:
    enabled: true
    priority: 90
    config:
      requests_per_second: 100
      burst_size: 200
      strategy: "token_bucket"

  # 用户服务插件
  user-service:
    enabled: true
    priority: 50
    config:
      cache_ttl: "1h"
      batch_size: 100
```

### 动态配置

```go
// 运行时动态配置插件
func configurePlugin() {
    pluginManager := chi.GetPluginManager()

    // 动态配置插件
    config := map[string]interface{}{
        "cache_ttl": "30m",
        "batch_size": 50,
    }

    err := pluginManager.ConfigurePlugin("user-service", config)
    if err != nil {
        log.Error("Failed to configure plugin:", err)
    }
}
```

## 插件生命周期

### 加载流程
```
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│ Load    │───▶│ Init    │───▶│ Start   │───▶│ Ready   │
│ Plugin  │    │ Plugin  │    │ Plugin  │    │ State   │
└─────────┘    └─────────┘    └─────────┘    └─────────┘
     │              │              │              │
     ▼              ▼              ▼              ▼
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│Validate │    │Resolve  │    │Register │    │Monitor  │
│Plugin   │    │Deps     │    │Services │    │Health   │
└─────────┘    └─────────┘    └─────────┘    └─────────┘
```

### 卸载流程
```
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│ Stop    │───▶│Unregister│───▶│ Close   │───▶│ Unload  │
│ Plugin  │    │ Services │    │ Plugin  │    │ Plugin  │
└─────────┘    └─────────┘    └─────────┘    └─────────┘
```

### 生命周期事件

```go
// 监听插件生命周期事件
func setupPluginEventHandlers() {
    pluginManager := chi.GetPluginManager()

    // 插件加载事件
    pluginManager.OnLoad(func(plugin plugin.Plugin) {
        log.Info("Plugin loaded:", plugin.Info().Name)
    })

    // 插件启动事件
    pluginManager.OnStart(func(plugin plugin.Plugin) {
        log.Info("Plugin started:", plugin.Info().Name)
    })

    // 插件停止事件
    pluginManager.OnStop(func(plugin plugin.Plugin) {
        log.Info("Plugin stopped:", plugin.Info().Name)
    })

    // 插件错误事件
    pluginManager.OnError(func(plugin plugin.Plugin, err error) {
        log.Error("Plugin error:", plugin.Info().Name, err)
    })
}
```

## 插件管理

### 基础操作

```go
// 获取插件管理器
pluginManager := chi.GetPluginManager()

// 加载插件
err := pluginManager.LoadPlugin("path/to/plugin.so")
if err != nil {
    log.Error("Failed to load plugin:", err)
}

// 启用插件
err = pluginManager.EnablePlugin("plugin-name")
if err != nil {
    log.Error("Failed to enable plugin:", err)
}

// 禁用插件
err = pluginManager.DisablePlugin("plugin-name")
if err != nil {
    log.Error("Failed to disable plugin:", err)
}

// 卸载插件
err = pluginManager.UnloadPlugin("plugin-name")
if err != nil {
    log.Error("Failed to unload plugin:", err)
}
```

### 插件查询

```go
// 获取所有插件信息
plugins := pluginManager.ListPlugins()
for _, plugin := range plugins {
    fmt.Printf("Plugin: %s v%s - %s\n",
               plugin.Name,
               plugin.Version,
               plugin.Status)
}

// 查询特定插件
plugin, exists := pluginManager.GetPlugin("plugin-name")
if exists {
    fmt.Printf("Plugin info: %+v\n", plugin.Info())
}

// 检查插件状态
status := pluginManager.GetPluginStatus("plugin-name")
fmt.Printf("Plugin status: %s\n", status)
```

## 插件热更新

### 实现热更新

```go
// 热更新插件
func hotReloadPlugin(pluginName string) error {
    pluginManager := chi.GetPluginManager()

    // 停止旧插件
    if err := pluginManager.StopPlugin(pluginName); err != nil {
        return fmt.Errorf("failed to stop plugin: %w", err)
    }

    // 卸载旧插件
    if err := pluginManager.UnloadPlugin(pluginName); err != nil {
        return fmt.Errorf("failed to unload plugin: %w", err)
    }

    // 加载新插件
    if err := pluginManager.LoadPlugin(pluginName); err != nil {
        return fmt.Errorf("failed to load new plugin: %w", err)
    }

    // 启动新插件
    if err := pluginManager.StartPlugin(pluginName); err != nil {
        return fmt.Errorf("failed to start new plugin: %w", err)
    }

    return nil
}
```

### 版本管理

```go
// 插件版本管理
type VersionManager struct {
    plugins map[string][]PluginVersion
}

type PluginVersion struct {
    Version string
    Path    string
    Active  bool
}

// 回滚到指定版本
func (vm *VersionManager) RollbackTo(pluginName, version string) error {
    versions, exists := vm.plugins[pluginName]
    if !exists {
        return fmt.Errorf("plugin %s not found", pluginName)
    }

    for _, v := range versions {
        if v.Version == version {
            return hotReloadPlugin(v.Path)
        }
    }

    return fmt.Errorf("version %s not found", version)
}
```

## 最佳实践

### 1. 插件设计原则

- **单一职责**：每个插件应该专注于一个特定功能
- **松耦合**：插件间应该通过接口通信，避免直接依赖
- **优雅降级**：插件故障不应该影响整个应用
- **配置驱动**：通过配置文件控制插件行为

### 2. 错误处理

```go
func (p *MyPlugin) Start(ctx context.Context) error {
    // 使用recover捕获panic
    defer func() {
        if r := recover(); r != nil {
            p.logger.Error("Plugin panic recovered:", r)
        }
    }()

    // 使用超时控制
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    // 实现重试逻辑
    return retry.Do(func() error {
        return p.doStart(ctx)
    }, retry.Attempts(3))
}
```

### 3. 资源管理

```go
func (p *MyPlugin) Close(ctx context.Context) error {
    // 确保资源被正确释放
    if p.conn != nil {
        p.conn.Close()
    }

    if p.workers != nil {
        close(p.workers)
    }

    // 等待所有goroutine结束
    p.wg.Wait()

    return nil
}
```

### 4. 测试策略

```go
func TestPluginLifecycle(t *testing.T) {
    plugin := &MyPlugin{}

    // 测试初始化
    config := &MockConfig{}
    err := plugin.Init(context.Background(), config)
    assert.NoError(t, err)

    // 测试启动
    err = plugin.Start(context.Background())
    assert.NoError(t, err)

    // 测试停止
    err = plugin.Stop(context.Background())
    assert.NoError(t, err)

    // 测试清理
    err = plugin.Close(context.Background())
    assert.NoError(t, err)
}
```

## 调试与监控

### 插件日志

```go
// 启用插件调试日志
func enablePluginDebugLog(pluginName string) {
    logger := chi.GetLogger()
    logger.SetLevel(pluginName, log.DebugLevel)
}
```

### 性能监控

```go
// 监控插件性能
func monitorPluginPerformance() {
    pluginManager := chi.GetPluginManager()

    // 定期收集插件指标
    go func() {
        ticker := time.NewTicker(30 * time.Second)
        defer ticker.Stop()

        for range ticker.C {
            metrics := pluginManager.GetMetrics()
            for pluginName, metric := range metrics {
                log.Info("Plugin metrics:",
                    "name", pluginName,
                    "cpu", metric.CPUUsage,
                    "memory", metric.MemoryUsage,
                    "goroutines", metric.GoroutineCount)
            }
        }
    }()
}
```

通过以上指南，您可以开发出功能强大、稳定可靠的 Chi 框架插件。插件系统为框架提供了极大的扩展性和灵活性，使得企业级应用的功能定制变得简单而高效。