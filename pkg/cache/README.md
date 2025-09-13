# Enterprise Cache Package

企业级Go缓存包，支持多级缓存架构、高并发访问、灵活的缓存策略配置、完善的监控和日志功能，以及分布式环境下的缓存一致性。

## 特性

### 🚀 核心功能
- **多级缓存架构**：支持内存(L1) + Redis(L2)多级缓存
- **线程安全**：支持高并发访问，内置读写锁和原子操作
- **灵活缓存策略**：支持LRU、LFU、TTL等多种淘汰策略
- **完善监控**：实时统计命中率、错误率、响应时间等关键指标
- **结构化日志**：基于zap的高性能日志系统
- **分布式一致性**：支持最终一致性和强一致性模式
- **简洁API**：易用的接口设计，支持Builder模式和工厂模式

### 📊 缓存类型
- `TypeMemoryLRU`：内存LRU缓存
- `TypeMemoryLFU`：内存LFU缓存  
- `TypeRedisSingle`：Redis单点缓存
- `TypeRedisCluster`：Redis集群缓存
- `TypeMultiLevel`：多级缓存

### ⚡ 性能特性
- 内存缓存：支持数十万QPS
- Redis缓存：连接池复用，支持集群模式
- LRU算法：O(1)时间复杂度的Get/Set操作
- 并发安全：读写分离锁，最小化锁竞争
- 内存管理：自动TTL过期清理，内存使用限制

## 快速开始

### 基本使用

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/tokmz/basic/pkg/cache"
)

func main() {
    // 创建内存缓存
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

### Builder模式

```go
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
```

### 多级缓存

```go
config := cache.DefaultConfig()
config.Type = cache.TypeMultiLevel
config.MultiLevel.EnableL1 = true  // 启用内存缓存
config.MultiLevel.EnableL2 = true  // 启用Redis缓存
config.MultiLevel.L1TTL = 30 * time.Minute
config.MultiLevel.L2TTL = 2 * time.Hour
config.MultiLevel.SyncStrategy = cache.SyncWriteThrough

cache, err := cache.CreateWithConfig(config)
```

### 缓存管理器

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

## 配置详解

### 内存缓存配置

```go
memoryConfig := cache.MemoryConfig{
    MaxSize:         1000,                    // 最大缓存项数量
    MaxMemory:       100 * 1024 * 1024,      // 最大内存使用量(100MB)
    EvictPolicy:     cache.EvictLRU,         // 淘汰策略
    CleanupInterval: 10 * time.Minute,       // 过期清理间隔
    EnableStats:     true,                   // 启用统计
}
```

### Redis缓存配置

```go
redisConfig := cache.RedisConfig{
    Addrs:        []string{"localhost:6379"}, // Redis地址
    Password:     "password",                 // 密码
    DB:           0,                         // 数据库编号
    PoolSize:     10,                        // 连接池大小
    MinIdleConns: 5,                         // 最小空闲连接
    MaxRetries:   3,                         // 最大重试次数
    DialTimeout:  5 * time.Second,           // 连接超时
    ReadTimeout:  3 * time.Second,           // 读超时
    WriteTimeout: 3 * time.Second,           // 写超时
    ClusterMode:  false,                     // 是否集群模式
    KeyPrefix:    "myapp:",                  // 键前缀
}
```

### 监控配置

```go
monitoringConfig := cache.MonitoringConfig{
    EnableMetrics:   true,                    // 启用指标收集
    MetricsInterval: 30 * time.Second,        // 指标收集间隔
    EnableLogging:   true,                    // 启用日志
    LogLevel:        "info",                  // 日志级别
    SlowThreshold:   100 * time.Millisecond,  // 慢操作阈值
}
```

## 高级功能

### 分布式缓存

```go
config := cache.DefaultConfig()
config.Distributed = cache.DistributedConfig{
    EnableSync:       true,                      // 启用分布式同步
    NodeID:          "node-1",                   // 节点ID
    SyncInterval:    5 * time.Second,            // 同步间隔
    ConsistencyLevel: cache.ConsistencyEventual, // 一致性级别
    BroadcastAddr:   "localhost:8080",           // 广播地址
}

distCache, err := cache.NewDistributedCache(config)
if err != nil {
    panic(err)
}

// 启动分布式缓存
err = distCache.Start(context.Background())
if err != nil {
    panic(err)
}
```

### 自定义监控

```go
// 创建自定义收集器
type CustomCollector struct {
    name string
}

func (c *CustomCollector) Name() string {
    return c.name
}

func (c *CustomCollector) Collect() (*cache.Metrics, error) {
    return &cache.Metrics{
        MemoryUsage: getCurrentMemoryUsage(),
        CPUUsage:    getCurrentCPUUsage(),
    }, nil
}

// 添加到监控器
monitor.AddCollector(&CustomCollector{name: "custom"})
```

### 事件处理

```go
// 订阅缓存事件
eventBus.Subscribe("cache.set", func(event *cache.CacheEvent) error {
    fmt.Printf("缓存设置: key=%s, node=%s\n", event.Key, event.NodeID)
    return nil
})

eventBus.Subscribe("cache.delete", func(event *cache.CacheEvent) error {
    fmt.Printf("缓存删除: key=%s, node=%s\n", event.Key, event.NodeID)
    return nil
})
```

## 性能优化建议

### 1. 内存缓存优化
- 合理设置`MaxSize`和`MaxMemory`限制
- 使用合适的淘汰策略(LRU适合大多数场景)
- 定期清理过期数据，避免内存泄漏

### 2. Redis缓存优化
- 配置合适的连接池大小
- 启用连接复用减少连接开销
- 使用键前缀避免冲突

### 3. 多级缓存优化
- L1缓存设置较短TTL，L2缓存设置较长TTL
- 根据业务特点选择同步策略
- 监控各级缓存命中率

### 4. 并发优化
- 读多写少场景使用读写锁
- 避免在锁内执行耗时操作
- 使用singleflight防止缓存击穿

## 错误处理

```go
value, err := cache.Get(ctx, "key")
if err != nil {
    switch err {
    case cache.ErrKeyNotFound:
        // 处理键不存在
    case cache.ErrCacheClosed:
        // 处理缓存已关闭
    default:
        // 处理其他错误
    }
}
```

## 测试

```bash
# 运行所有测试
go test ./...

# 运行特定测试
go test -run TestMemoryCache

# 运行性能测试
go test -bench=BenchmarkMemoryCache

# 查看测试覆盖率
go test -cover ./...
```

## 许可证

MIT License

## 贡献指南

1. Fork项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开Pull Request

## 更新日志

### v1.0.0
- 初始版本发布
- 支持内存和Redis缓存
- 实现多级缓存架构
- 添加完善的监控和日志功能
- 支持分布式一致性