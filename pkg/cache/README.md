# Redis 缓存库

一个功能完整、高性能的 Go Redis 缓存库，提供了丰富的操作接口和高级功能。

## 特性

### 🚀 核心功能
- **完整的 Redis 操作支持**：字符串、哈希表、列表、集合、有序集合
- **Lua 脚本支持**：内置常用脚本，支持自定义脚本管理
- **批量操作**：高效的批量数据处理
- **原子操作**：计数器、分布式锁、限流器等

### 📊 监控与追踪
- **性能监控**：操作统计、慢查询记录、响应时间分析
- **链路追踪**：基于 OpenTelemetry 的分布式追踪
- **健康检查**：连接状态监控和自动恢复

### 🔧 高级功能
- **连接池管理**：智能连接池配置和监控
- **配置热更新**：运行时动态调整配置
- **多种数据结构**：队列、栈、排行榜、过滤器等
- **分布式组件**：分布式锁、限流器、原子计数器

## 快速开始

### 安装

```bash
go get github.com/your-org/basic/pkg/cache
```

### 基础使用

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "basic/pkg/cache"
    "basic/pkg/cache/config"
)

func main() {
    // 使用默认配置
    cfg := config.DefaultConfig()
    
    // 创建缓存客户端
    cacheClient, err := cache.New(cfg)
    if err != nil {
        panic(err)
    }
    defer cacheClient.Close()
    
    ctx := context.Background()
    
    // 基础操作
    expiration := 10 * time.Minute
    err = cacheClient.Set(ctx, "key", "value", &expiration)
    if err != nil {
        panic(err)
    }
    
    value, err := cacheClient.Get(ctx, "key")
    if err != nil {
        panic(err)
    }
    fmt.Println("Value:", value)
}
```

### 使用默认实例

```go
package main

import (
    "context"
    "time"
    
    "basic/pkg/cache"
    "basic/pkg/cache/config"
)

func main() {
    // 初始化默认缓存实例
    cfg := config.DefaultConfig()
    err := cache.Init(cfg)
    if err != nil {
        panic(err)
    }
    
    ctx := context.Background()
    
    // 直接使用全局函数
    expiration := 5 * time.Minute
    cache.Set(ctx, "global_key", "global_value", &expiration)
    
    value, _ := cache.Get(ctx, "global_key")
    fmt.Println("Global value:", value)
}
```

## 配置说明

### 基础配置

```go
cfg := &config.RedisConfig{
    // 连接配置
    Addr:     "localhost:6379",
    Password: "",
    DB:       0,
    
    // 连接池配置
    PoolSize:        20,
    MinIdleConns:    5,
    MaxIdleConns:    10,
    ConnMaxIdleTime: 30 * time.Minute,
    ConnMaxLifetime: 2 * time.Hour,
    
    // 超时配置
    DialTimeout:  10 * time.Second,
    ReadTimeout:  5 * time.Second,
    WriteTimeout: 5 * time.Second,
    
    // 重试配置
    MaxRetries:      5,
    MinRetryBackoff: 10 * time.Millisecond,
    MaxRetryBackoff: 1 * time.Second,
    
    // 默认过期时间
    DefaultExpiration: 12 * time.Hour,
}
```

### 监控配置

```go
cfg.Monitoring = config.MonitoringConfig{
    Enabled:          true,
    MetricsNamespace: "my_app_redis",
    SlowLogThreshold: 50 * time.Millisecond,
    CollectInterval:  5 * time.Second,
}
```

### 链路追踪配置

```go
cfg.Tracing = config.TracingConfig{
    Enabled:     true,
    ServiceName: "my-redis-cache",
    Version:     "1.0.0",
}
```

## 功能详解

### 字符串操作

```go
// 基础操作
cacheClient.Set(ctx, "key", "value", &expiration)
value, _ := cacheClient.Get(ctx, "key")

// 批量操作
pairs := map[string]interface{}{
    "key1": "value1",
    "key2": "value2",
}
cacheClient.MSet(ctx, pairs, &expiration)
values, _ := cacheClient.MGet(ctx, "key1", "key2")

// 原子计数器
counter := cacheClient.NewCounter("page_views")
count, _ := counter.Increment(ctx, 1)
```

### 哈希表操作

```go
// 基础操作
cacheClient.HSet(ctx, "user:1001", "name", "张三", "age", "25")
name, _ := cacheClient.HGet(ctx, "user:1001", "name")
profile, _ := cacheClient.HGetAll(ctx, "user:1001")

// 批量操作
hashBatch := cacheClient.NewHashBatch("user:1001")
hashBatch.Set("email", "user@example.com").Set("phone", "13800138000")
hashBatch.Execute(ctx)

// 哈希表计数器
hashCounter := cacheClient.NewHashCounter("stats:daily", "login_count")
count, _ := hashCounter.Increment(ctx, 1)
```

### 列表操作

```go
// 基础操作
cacheClient.LPush(ctx, "messages", "msg1", "msg2")
message, _ := cacheClient.RPop(ctx, "messages")

// 队列操作
queue := cacheClient.NewListQueue("task_queue")
queue.Enqueue(ctx, "task1", "task2")
task, _ := queue.Dequeue(ctx)

// 栈操作
stack := cacheClient.NewListStack("operation_stack")
stack.Push(ctx, "op1", "op2")
operation, _ := stack.Pop(ctx)
```

### 集合操作

```go
// 基础操作
cacheClient.SAdd(ctx, "tags", "go", "redis", "cache")
members, _ := cacheClient.SMembers(ctx, "tags")

// 集合运算
intersection, _ := cacheClient.SInter(ctx, "set1", "set2")
union, _ := cacheClient.SUnion(ctx, "set1", "set2")

// 集合过滤器
filter := cacheClient.NewSetFilter("active_users")
filter.Add(ctx, "user1", "user2")
exists, _ := filter.Contains(ctx, "user1")
```

### 有序集合操作

```go
// 基础操作
members := []redis.Z{
    {Score: 100, Member: "player1"},
    {Score: 200, Member: "player2"},
}
cacheClient.ZAdd(ctx, "leaderboard", members...)
top10, _ := cacheClient.ZRevRange(ctx, "leaderboard", 0, 9)

// 排行榜功能
leaderboard := cacheClient.NewLeaderboard("game_scores")
leaderboard.AddScore(ctx, "player1", 1500)
rank, _ := leaderboard.GetRank(ctx, "player1")
topPlayers, _ := leaderboard.GetTopN(ctx, 10)
```

### Lua 脚本操作

```go
// 直接执行脚本
script := `
    local key = KEYS[1]
    local value = ARGV[1]
    redis.call('SET', key, value)
    return redis.call('GET', key)
`
result, _ := cacheClient.Eval(ctx, script, []string{"test_key"}, "test_value")

// 注册和执行脚本
cacheClient.RegisterScript("my_script", script)
result, _ := cacheClient.ExecuteScript(ctx, "my_script", []string{"key"}, "value")

// 分布式锁
lock := cacheClient.NewDistributedLock("resource_lock", "unique_value", 30*time.Second)
locked, _ := lock.TryLock(ctx)
if locked {
    // 执行业务逻辑
    lock.Unlock(ctx)
}

// 限流器
rateLimiter := cacheClient.NewRateLimiter("api_limit", time.Minute, 100)
allowed, remaining, _ := rateLimiter.Allow(ctx)
```

### 批量操作

```go
// 批量操作器
batchOp := cacheClient.NewBatchOperator()

// 批量设置带过期时间的键值对
kvs := map[string]interface{}{
    "session:1001": "user_data_1",
    "session:1002": "user_data_2",
}
count, _ := batchOp.BatchSetWithExpire(ctx, kvs, 30*time.Minute)

// 多个哈希表的字段获取
hashKeys := []string{"user:1001", "user:1002"}
results, _ := batchOp.MultiHGet(ctx, hashKeys, "name")
```

## 监控和统计

### 获取统计信息

```go
// 获取操作统计
stats := cacheClient.GetStats()
fmt.Printf("总操作数: %d\n", stats.TotalOperations)
fmt.Printf("成功操作数: %d\n", stats.SuccessOperations)
fmt.Printf("平均响应时间: %v\n", stats.AverageDuration)

// 获取慢查询记录
slowQueries := cacheClient.GetSlowQueries()
for _, query := range slowQueries {
    fmt.Printf("慢查询: %s, 键: %s, 耗时: %v\n", 
        query.Operation, query.Key, query.Duration)
}
```

### 配置管理

```go
// 获取当前配置
cfg := cacheClient.GetConfig()

// 动态调整配置
cacheClient.UpdatePoolConfig(20, 5, 15) // poolSize, minIdle, maxIdle
cacheClient.UpdateTimeouts(10*time.Second, 5*time.Second, 5*time.Second)
cacheClient.SetSlowLogThreshold(50 * time.Millisecond)

// 重置统计信息
cacheClient.ResetStats()
cacheClient.ClearSlowQueries()
```

## 高级功能

### 原子计数器

```go
// 带过期时间的原子计数器
atomicCounter := cacheClient.NewAtomicCounter("daily_visits", 24*time.Hour)
count, _ := atomicCounter.IncrBy(ctx, 5)
count, _ = atomicCounter.DecrBy(ctx, 2)
count, _ = atomicCounter.Get(ctx)
```

### Lua 脚本管理器

```go
// 创建脚本管理器
scriptManager := cacheClient.NewLuaScriptManager()

// 注册自定义脚本
customScript := `
    local key = KEYS[1]
    local field = ARGV[1]
    local increment = tonumber(ARGV[2])
    
    local current = redis.call('HGET', key, field)
    if current == false then
        current = 0
    else
        current = tonumber(current)
    end
    
    local new_value = current + increment
    redis.call('HSET', key, field, new_value)
    return new_value
`
scriptManager.RegisterScript("hash_incr", customScript)

// 执行脚本
result, _ := scriptManager.ExecuteScript(ctx, "hash_incr", 
    []string{"stats:counters"}, "page_views", 10)

// 列出所有脚本
scripts := scriptManager.ListScripts()
```

## 最佳实践

### 1. 连接池配置

```go
// 根据应用负载调整连接池大小
cfg.PoolSize = 20        // 最大连接数
cfg.MinIdleConns = 5     // 最小空闲连接数
cfg.MaxIdleConns = 10    // 最大空闲连接数
```

### 2. 超时配置

```go
// 合理设置超时时间
cfg.DialTimeout = 10 * time.Second  // 连接超时
cfg.ReadTimeout = 5 * time.Second   // 读取超时
cfg.WriteTimeout = 5 * time.Second  // 写入超时
```

### 3. 监控配置

```go
// 启用监控和慢查询记录
cfg.Monitoring.Enabled = true
cfg.Monitoring.SlowLogThreshold = 50 * time.Millisecond
```

### 4. 错误处理

```go
// 始终检查错误
value, err := cacheClient.Get(ctx, "key")
if err != nil {
    if err == redis.Nil {
        // 键不存在
        fmt.Println("Key not found")
    } else {
        // 其他错误
        log.Printf("Redis error: %v", err)
    }
}
```

### 5. 上下文使用

```go
// 使用带超时的上下文
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

value, err := cacheClient.Get(ctx, "key")
```

## 性能优化

### 1. 批量操作

```go
// 使用批量操作减少网络往返
pairs := map[string]interface{}{
    "key1": "value1",
    "key2": "value2",
    "key3": "value3",
}
cacheClient.MSet(ctx, pairs, nil)
```

### 2. 管道操作

```go
// 使用 Lua 脚本进行原子操作
script := `
    local results = {}
    for i = 1, #KEYS do
        table.insert(results, redis.call('GET', KEYS[i]))
    end
    return results
`
results, _ := cacheClient.Eval(ctx, script, []string{"key1", "key2", "key3"})
```

### 3. 连接复用

```go
// 复用缓存客户端实例，避免频繁创建连接
var cacheClient cache.Cache

func init() {
    cfg := config.DefaultConfig()
    var err error
    cacheClient, err = cache.New(cfg)
    if err != nil {
        panic(err)
    }
}
```

## 故障排查

### 1. 连接问题

```go
// 检查连接状态
if err := cacheClient.Ping(ctx); err != nil {
    log.Printf("Redis connection failed: %v", err)
}
```

### 2. 性能问题

```go
// 检查慢查询
slowQueries := cacheClient.GetSlowQueries()
for _, query := range slowQueries {
    if query.Duration > 100*time.Millisecond {
        log.Printf("Slow query detected: %s, duration: %v", 
            query.Operation, query.Duration)
    }
}
```

### 3. 内存使用

```go
// 监控统计信息
stats := cacheClient.GetStats()
log.Printf("Total operations: %d, Failed: %d", 
    stats.TotalOperations, stats.FailedOperations)
```

## 示例项目

完整的使用示例请参考 `examples/basic_usage.go` 文件，其中包含了所有功能的详细演示。

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request 来改进这个项目。