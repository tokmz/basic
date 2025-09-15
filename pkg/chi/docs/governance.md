# 服务治理

## 概述

Chi 框架提供了完整的服务治理解决方案，包括熔断器、限流控制、舱壁模式、负载均衡、超时控制和重试机制等。这些组件协同工作，确保系统在面临故障、高负载或网络问题时仍能保持稳定和可用。

## 服务治理架构

### 整体架构图
```
┌─────────────────────────────────────────────────────────────────┐
│                    Service Governance                           │
│                                                                 │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐          │
│  │  Circuit    │   │    Rate     │   │  Bulkhead   │          │
│  │  Breaker    │   │   Limiter   │   │   Pattern   │          │
│  └─────────────┘   └─────────────┘   └─────────────┘          │
│          │                 │                 │                 │
│          └─────────────────┼─────────────────┘                 │
│                            │                                   │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                Governance Gateway                          ││
│  └─────────────────────────────────────────────────────────────┐│
│                            │                                   │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐          │
│  │  Timeout    │   │    Retry    │   │ Health      │          │
│  │  Control    │   │   Manager   │   │ Monitor     │          │
│  └─────────────┘   └─────────────┘   └─────────────┘          │
└─────────────────────────────────────────────────────────────────┘
```

### 核心组件

1. **Circuit Breaker**: 熔断器，防止故障传播
2. **Rate Limiter**: 限流器，控制请求速率
3. **Bulkhead Pattern**: 舱壁模式，资源隔离
4. **Timeout Control**: 超时控制，避免长时间阻塞
5. **Retry Manager**: 重试管理器，智能重试策略
6. **Health Monitor**: 健康监控，实时状态检测

## 熔断器

### 熔断器设计

```go
package governance

import (
    "context"
    "errors"
    "sync"
    "time"
)

// 熔断器状态
type CircuitState int

const (
    StateClosed CircuitState = iota
    StateOpen
    StateHalfOpen
)

// 熔断器配置
type CircuitBreakerConfig struct {
    MaxRequests         uint32        // 半开状态下的最大请求数
    Interval            time.Duration // 统计窗口时间
    Timeout             time.Duration // 开路超时时间
    FailureThreshold    float64       // 失败率阈值
    SuccessThreshold    uint32        // 半开状态下成功恢复的阈值
}

// 熔断器接口
type CircuitBreaker interface {
    Execute(ctx context.Context, fn func() (interface{}, error)) (interface{}, error)
    State() CircuitState
    Metrics() CircuitMetrics
}

// 熔断器指标
type CircuitMetrics struct {
    Requests          uint64
    TotalSuccesses    uint64
    TotalFailures     uint64
    ConsecutiveSuccesses uint64
    ConsecutiveFailures  uint64
}

// 熔断器实现
type circuitBreaker struct {
    config   CircuitBreakerConfig
    state    CircuitState
    metrics  CircuitMetrics
    expiry   time.Time
    mutex    sync.Mutex
}

func NewCircuitBreaker(config CircuitBreakerConfig) CircuitBreaker {
    return &circuitBreaker{
        config: config,
        state:  StateClosed,
    }
}

func (cb *circuitBreaker) Execute(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
    if !cb.allow() {
        return nil, errors.New("circuit breaker is open")
    }

    result, err := fn()
    cb.record(err == nil)

    return result, err
}

func (cb *circuitBreaker) allow() bool {
    cb.mutex.Lock()
    defer cb.mutex.Unlock()

    switch cb.state {
    case StateClosed:
        return true
    case StateOpen:
        if time.Now().After(cb.expiry) {
            cb.setState(StateHalfOpen)
            return true
        }
        return false
    case StateHalfOpen:
        return cb.metrics.Requests < cb.config.MaxRequests
    }

    return false
}

func (cb *circuitBreaker) record(success bool) {
    cb.mutex.Lock()
    defer cb.mutex.Unlock()

    cb.metrics.Requests++

    if success {
        cb.metrics.TotalSuccesses++
        cb.metrics.ConsecutiveSuccesses++
        cb.metrics.ConsecutiveFailures = 0

        if cb.state == StateHalfOpen &&
           cb.metrics.ConsecutiveSuccesses >= cb.config.SuccessThreshold {
            cb.setState(StateClosed)
        }
    } else {
        cb.metrics.TotalFailures++
        cb.metrics.ConsecutiveFailures++
        cb.metrics.ConsecutiveSuccesses = 0

        if cb.shouldTrip() {
            cb.setState(StateOpen)
        }
    }
}

func (cb *circuitBreaker) shouldTrip() bool {
    if cb.metrics.Requests < 10 { // 最小请求数
        return false
    }

    failureRate := float64(cb.metrics.ConsecutiveFailures) / float64(cb.metrics.Requests)
    return failureRate >= cb.config.FailureThreshold
}

func (cb *circuitBreaker) setState(state CircuitState) {
    cb.state = state
    if state == StateOpen {
        cb.expiry = time.Now().Add(cb.config.Timeout)
    }
    if state == StateClosed {
        cb.metrics = CircuitMetrics{}
    }
}

func (cb *circuitBreaker) State() CircuitState {
    cb.mutex.Lock()
    defer cb.mutex.Unlock()
    return cb.state
}

func (cb *circuitBreaker) Metrics() CircuitMetrics {
    cb.mutex.Lock()
    defer cb.mutex.Unlock()
    return cb.metrics
}
```

### 熔断器使用示例

```go
// 创建熔断器
func createCircuitBreaker() CircuitBreaker {
    config := CircuitBreakerConfig{
        MaxRequests:      10,
        Interval:         time.Minute,
        Timeout:          30 * time.Second,
        FailureThreshold: 0.6, // 60%失败率触发熔断
        SuccessThreshold: 5,   // 5次成功后恢复
    }

    return NewCircuitBreaker(config)
}

// 使用熔断器保护服务调用
func callServiceWithCircuitBreaker() {
    cb := createCircuitBreaker()

    result, err := cb.Execute(context.Background(), func() (interface{}, error) {
        // 实际的服务调用
        return httpClient.Get("https://api.service.com/data")
    })

    if err != nil {
        log.Error("Service call failed:", err)
        return
    }

    log.Info("Service call succeeded:", result)
}

// HTTP中间件集成
func CircuitBreakerMiddleware(cb CircuitBreaker) gin.HandlerFunc {
    return func(c *gin.Context) {
        result, err := cb.Execute(c.Request.Context(), func() (interface{}, error) {
            // 执行后续中间件和处理器
            c.Next()

            // 检查响应状态
            if c.Writer.Status() >= 500 {
                return nil, errors.New("server error")
            }

            return nil, nil
        })

        if err != nil {
            c.JSON(503, gin.H{
                "error": "service temporarily unavailable",
                "state": cb.State().String(),
            })
            c.Abort()
        }
    }
}
```

## 限流控制

### 令牌桶算法

```go
// 令牌桶限流器
type TokenBucketLimiter struct {
    capacity    int64         // 桶容量
    tokens      int64         // 当前令牌数
    refillRate  time.Duration // 令牌补充速率
    lastRefill  time.Time     // 上次补充时间
    mutex       sync.Mutex
}

func NewTokenBucketLimiter(capacity int64, refillRate time.Duration) *TokenBucketLimiter {
    return &TokenBucketLimiter{
        capacity:   capacity,
        tokens:     capacity,
        refillRate: refillRate,
        lastRefill: time.Now(),
    }
}

func (tbl *TokenBucketLimiter) Allow() bool {
    return tbl.AllowN(1)
}

func (tbl *TokenBucketLimiter) AllowN(n int64) bool {
    tbl.mutex.Lock()
    defer tbl.mutex.Unlock()

    tbl.refill()

    if tbl.tokens >= n {
        tbl.tokens -= n
        return true
    }

    return false
}

func (tbl *TokenBucketLimiter) refill() {
    now := time.Now()
    elapsed := now.Sub(tbl.lastRefill)

    if elapsed <= 0 {
        return
    }

    tokensToAdd := int64(elapsed / tbl.refillRate)
    if tokensToAdd > 0 {
        tbl.tokens = min(tbl.capacity, tbl.tokens+tokensToAdd)
        tbl.lastRefill = now
    }
}

func (tbl *TokenBucketLimiter) Tokens() int64 {
    tbl.mutex.Lock()
    defer tbl.mutex.Unlock()

    tbl.refill()
    return tbl.tokens
}
```

### 滑动窗口限流

```go
// 滑动窗口限流器
type SlidingWindowLimiter struct {
    capacity    int64
    windowSize  time.Duration
    requests    []time.Time
    mutex       sync.Mutex
}

func NewSlidingWindowLimiter(capacity int64, windowSize time.Duration) *SlidingWindowLimiter {
    return &SlidingWindowLimiter{
        capacity:   capacity,
        windowSize: windowSize,
        requests:   make([]time.Time, 0),
    }
}

func (swl *SlidingWindowLimiter) Allow() bool {
    swl.mutex.Lock()
    defer swl.mutex.Unlock()

    now := time.Now()
    windowStart := now.Add(-swl.windowSize)

    // 清理过期请求
    validRequests := 0
    for i, reqTime := range swl.requests {
        if reqTime.After(windowStart) {
            swl.requests = swl.requests[i:]
            validRequests = len(swl.requests)
            break
        }
    }

    if validRequests == 0 {
        swl.requests = swl.requests[:0]
    }

    // 检查是否允许新请求
    if int64(len(swl.requests)) < swl.capacity {
        swl.requests = append(swl.requests, now)
        return true
    }

    return false
}

func (swl *SlidingWindowLimiter) Count() int {
    swl.mutex.Lock()
    defer swl.mutex.Unlock()

    now := time.Now()
    windowStart := now.Add(-swl.windowSize)

    count := 0
    for _, reqTime := range swl.requests {
        if reqTime.After(windowStart) {
            count++
        }
    }

    return count
}
```

### 多维度限流

```go
// 多维度限流器
type MultiDimensionalLimiter struct {
    limiters map[string]RateLimiter
    mutex    sync.RWMutex
}

func NewMultiDimensionalLimiter() *MultiDimensionalLimiter {
    return &MultiDimensionalLimiter{
        limiters: make(map[string]RateLimiter),
    }
}

func (mdl *MultiDimensionalLimiter) SetLimiter(key string, limiter RateLimiter) {
    mdl.mutex.Lock()
    defer mdl.mutex.Unlock()

    mdl.limiters[key] = limiter
}

func (mdl *MultiDimensionalLimiter) Allow(keys ...string) bool {
    mdl.mutex.RLock()
    defer mdl.mutex.RUnlock()

    // 所有维度都必须通过限流检查
    for _, key := range keys {
        if limiter, exists := mdl.limiters[key]; exists {
            if !limiter.Allow() {
                return false
            }
        }
    }

    return true
}

// 使用示例
func setupMultiDimensionalLimiting() {
    limiter := NewMultiDimensionalLimiter()

    // 按用户限流：每分钟100次
    userLimiter := NewTokenBucketLimiter(100, time.Minute/100)
    limiter.SetLimiter("user:123", userLimiter)

    // 按IP限流：每分钟1000次
    ipLimiter := NewTokenBucketLimiter(1000, time.Minute/1000)
    limiter.SetLimiter("ip:192.168.1.1", ipLimiter)

    // 按API限流：每分钟10000次
    apiLimiter := NewTokenBucketLimiter(10000, time.Minute/10000)
    limiter.SetLimiter("api:/users", apiLimiter)

    // 检查多维度限流
    userID := "user:123"
    clientIP := "ip:192.168.1.1"
    apiPath := "api:/users"

    if limiter.Allow(userID, clientIP, apiPath) {
        // 允许请求
        log.Info("Request allowed")
    } else {
        // 拒绝请求
        log.Warn("Request rate limited")
    }
}
```

## 舱壁模式

### 资源隔离池

```go
// 资源池配置
type PoolConfig struct {
    MaxIdle     int           // 最大空闲连接数
    MaxActive   int           // 最大活跃连接数
    IdleTimeout time.Duration // 空闲连接超时时间
    Wait        bool          // 是否等待可用连接
    WaitTimeout time.Duration // 等待超时时间
}

// 资源池接口
type ResourcePool interface {
    Get(ctx context.Context) (Resource, error)
    Put(resource Resource) error
    Close() error
    Stats() PoolStats
}

// 资源接口
type Resource interface {
    IsValid() bool
    Close() error
}

// 资源池统计
type PoolStats struct {
    ActiveCount int
    IdleCount   int
    WaitCount   int
}

// 资源池实现
type resourcePool struct {
    config      PoolConfig
    factory     func() (Resource, error)
    resources   chan Resource
    activeCount int32
    closed      int32
    mutex       sync.Mutex
    waiters     []chan Resource
}

func NewResourcePool(config PoolConfig, factory func() (Resource, error)) ResourcePool {
    return &resourcePool{
        config:    config,
        factory:   factory,
        resources: make(chan Resource, config.MaxIdle),
        waiters:   make([]chan Resource, 0),
    }
}

func (p *resourcePool) Get(ctx context.Context) (Resource, error) {
    if atomic.LoadInt32(&p.closed) == 1 {
        return nil, errors.New("pool is closed")
    }

    select {
    case resource := <-p.resources:
        if resource.IsValid() {
            atomic.AddInt32(&p.activeCount, 1)
            return resource, nil
        }
        // 资源无效，创建新的
        fallthrough
    default:
        // 检查是否可以创建新资源
        if atomic.LoadInt32(&p.activeCount) < int32(p.config.MaxActive) {
            resource, err := p.factory()
            if err != nil {
                return nil, err
            }
            atomic.AddInt32(&p.activeCount, 1)
            return resource, nil
        }

        // 达到最大连接数，需要等待
        if !p.config.Wait {
            return nil, errors.New("pool is full")
        }

        return p.waitForResource(ctx)
    }
}

func (p *resourcePool) Put(resource Resource) error {
    if atomic.LoadInt32(&p.closed) == 1 {
        resource.Close()
        return errors.New("pool is closed")
    }

    atomic.AddInt32(&p.activeCount, -1)

    if !resource.IsValid() {
        resource.Close()
        return nil
    }

    select {
    case p.resources <- resource:
        return nil
    default:
        // 空闲池已满，关闭资源
        resource.Close()
        return nil
    }
}

func (p *resourcePool) waitForResource(ctx context.Context) (Resource, error) {
    waiter := make(chan Resource, 1)

    p.mutex.Lock()
    p.waiters = append(p.waiters, waiter)
    p.mutex.Unlock()

    timeout := p.config.WaitTimeout
    if timeout == 0 {
        timeout = 30 * time.Second
    }

    select {
    case resource := <-waiter:
        return resource, nil
    case <-ctx.Done():
        p.removeWaiter(waiter)
        return nil, ctx.Err()
    case <-time.After(timeout):
        p.removeWaiter(waiter)
        return nil, errors.New("wait for resource timeout")
    }
}

func (p *resourcePool) removeWaiter(waiter chan Resource) {
    p.mutex.Lock()
    defer p.mutex.Unlock()

    for i, w := range p.waiters {
        if w == waiter {
            p.waiters = append(p.waiters[:i], p.waiters[i+1:]...)
            break
        }
    }
}
```

### 舱壁隔离

```go
// 舱壁隔离器
type BulkheadIsolator struct {
    pools map[string]ResourcePool
    mutex sync.RWMutex
}

func NewBulkheadIsolator() *BulkheadIsolator {
    return &BulkheadIsolator{
        pools: make(map[string]ResourcePool),
    }
}

func (bi *BulkheadIsolator) CreateBulkhead(name string, config PoolConfig, factory func() (Resource, error)) {
    bi.mutex.Lock()
    defer bi.mutex.Unlock()

    bi.pools[name] = NewResourcePool(config, factory)
}

func (bi *BulkheadIsolator) ExecuteInBulkhead(ctx context.Context, bulkheadName string, fn func(Resource) error) error {
    bi.mutex.RLock()
    pool, exists := bi.pools[bulkheadName]
    bi.mutex.RUnlock()

    if !exists {
        return fmt.Errorf("bulkhead %s not found", bulkheadName)
    }

    resource, err := pool.Get(ctx)
    if err != nil {
        return fmt.Errorf("failed to get resource from bulkhead %s: %w", bulkheadName, err)
    }

    defer pool.Put(resource)

    return fn(resource)
}

// 使用示例
func setupBulkheads() {
    isolator := NewBulkheadIsolator()

    // 数据库连接池
    dbConfig := PoolConfig{
        MaxIdle:     10,
        MaxActive:   50,
        IdleTimeout: 5 * time.Minute,
        Wait:        true,
        WaitTimeout: 30 * time.Second,
    }

    isolator.CreateBulkhead("database", dbConfig, func() (Resource, error) {
        return newDatabaseConnection()
    })

    // HTTP客户端池
    httpConfig := PoolConfig{
        MaxIdle:     5,
        MaxActive:   20,
        IdleTimeout: 2 * time.Minute,
        Wait:        false,
    }

    isolator.CreateBulkhead("http-client", httpConfig, func() (Resource, error) {
        return newHTTPClient()
    })

    // 在隔离环境中执行数据库操作
    err := isolator.ExecuteInBulkhead(context.Background(), "database", func(resource Resource) error {
        db := resource.(*DatabaseConnection)
        return db.Query("SELECT * FROM users")
    })

    if err != nil {
        log.Error("Database operation failed:", err)
    }
}
```

## 超时控制

### 超时管理器

```go
// 超时配置
type TimeoutConfig struct {
    Default      time.Duration            // 默认超时时间
    Operations   map[string]time.Duration // 操作特定超时时间
    Progressive  bool                     // 是否启用渐进式超时
    BackoffRate  float64                  // 退避率
}

// 超时管理器
type TimeoutManager struct {
    config TimeoutConfig
    stats  map[string]*TimeoutStats
    mutex  sync.RWMutex
}

// 超时统计
type TimeoutStats struct {
    TotalRequests   int64
    TimeoutCount    int64
    AverageLatency  time.Duration
    LastTimeout     time.Time
}

func NewTimeoutManager(config TimeoutConfig) *TimeoutManager {
    return &TimeoutManager{
        config: config,
        stats:  make(map[string]*TimeoutStats),
    }
}

func (tm *TimeoutManager) Execute(ctx context.Context, operation string, fn func(context.Context) error) error {
    timeout := tm.getTimeout(operation)

    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    start := time.Now()
    err := fn(ctx)
    duration := time.Since(start)

    tm.recordStats(operation, duration, err == context.DeadlineExceeded)

    return err
}

func (tm *TimeoutManager) getTimeout(operation string) time.Duration {
    tm.mutex.RLock()
    defer tm.mutex.RUnlock()

    if timeout, exists := tm.config.Operations[operation]; exists {
        if tm.config.Progressive {
            stats := tm.stats[operation]
            if stats != nil && stats.TimeoutCount > 0 {
                // 渐进式超时：根据历史超时情况调整
                backoffMultiplier := 1.0 + float64(stats.TimeoutCount)*tm.config.BackoffRate
                return time.Duration(float64(timeout) * backoffMultiplier)
            }
        }
        return timeout
    }

    return tm.config.Default
}

func (tm *TimeoutManager) recordStats(operation string, duration time.Duration, isTimeout bool) {
    tm.mutex.Lock()
    defer tm.mutex.Unlock()

    stats, exists := tm.stats[operation]
    if !exists {
        stats = &TimeoutStats{}
        tm.stats[operation] = stats
    }

    stats.TotalRequests++
    if isTimeout {
        stats.TimeoutCount++
        stats.LastTimeout = time.Now()
    }

    // 计算平均延迟
    stats.AverageLatency = time.Duration(
        (int64(stats.AverageLatency)*stats.TotalRequests + int64(duration)) / (stats.TotalRequests + 1),
    )
}
```

## 重试机制

### 智能重试管理器

```go
// 重试策略
type RetryStrategy interface {
    NextDelay(attempt int) time.Duration
    ShouldRetry(err error, attempt int) bool
    MaxAttempts() int
}

// 指数退避策略
type ExponentialBackoffStrategy struct {
    BaseDelay    time.Duration
    MaxDelay     time.Duration
    Multiplier   float64
    MaxAttempts_ int
    Jitter       bool
}

func (ebs *ExponentialBackoffStrategy) NextDelay(attempt int) time.Duration {
    delay := time.Duration(float64(ebs.BaseDelay) * math.Pow(ebs.Multiplier, float64(attempt)))

    if delay > ebs.MaxDelay {
        delay = ebs.MaxDelay
    }

    if ebs.Jitter {
        // 添加随机抖动，避免惊群效应
        jitter := time.Duration(rand.Int63n(int64(delay) / 2))
        delay = delay/2 + jitter
    }

    return delay
}

func (ebs *ExponentialBackoffStrategy) ShouldRetry(err error, attempt int) bool {
    if attempt >= ebs.MaxAttempts_ {
        return false
    }

    // 检查错误类型是否应该重试
    if isRetryableError(err) {
        return true
    }

    return false
}

func (ebs *ExponentialBackoffStrategy) MaxAttempts() int {
    return ebs.MaxAttempts_
}

// 重试管理器
type RetryManager struct {
    strategy RetryStrategy
    logger   Logger
}

func NewRetryManager(strategy RetryStrategy, logger Logger) *RetryManager {
    return &RetryManager{
        strategy: strategy,
        logger:   logger,
    }
}

func (rm *RetryManager) Execute(ctx context.Context, fn func() error) error {
    var lastErr error

    for attempt := 0; attempt < rm.strategy.MaxAttempts(); attempt++ {
        if attempt > 0 {
            delay := rm.strategy.NextDelay(attempt - 1)
            rm.logger.Debug("Retrying operation", "attempt", attempt, "delay", delay)

            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-time.After(delay):
                // 继续执行
            }
        }

        err := fn()
        if err == nil {
            if attempt > 0 {
                rm.logger.Info("Operation succeeded after retry", "attempts", attempt+1)
            }
            return nil
        }

        lastErr = err
        if !rm.strategy.ShouldRetry(err, attempt) {
            rm.logger.Info("Operation failed, not retryable", "error", err, "attempts", attempt+1)
            break
        }

        rm.logger.Warn("Operation failed, will retry", "error", err, "attempt", attempt+1)
    }

    return fmt.Errorf("operation failed after %d attempts: %w", rm.strategy.MaxAttempts(), lastErr)
}

// 可重试错误判断
func isRetryableError(err error) bool {
    if err == nil {
        return false
    }

    // 网络错误通常是可重试的
    if netErr, ok := err.(net.Error); ok {
        return netErr.Temporary() || netErr.Timeout()
    }

    // HTTP状态码判断
    if httpErr, ok := err.(*HTTPError); ok {
        return httpErr.StatusCode >= 500 && httpErr.StatusCode < 600
    }

    // 数据库连接错误
    if isConnectionError(err) {
        return true
    }

    return false
}
```

## 治理中间件

### 统一治理中间件

```go
// 治理配置
type GovernanceConfig struct {
    CircuitBreaker  *CircuitBreakerConfig
    RateLimiter     *RateLimiterConfig
    Timeout         *TimeoutConfig
    Retry           *RetryConfig
    Bulkhead        *BulkheadConfig
}

// 治理中间件
func GovernanceMiddleware(config GovernanceConfig) gin.HandlerFunc {
    var circuitBreaker CircuitBreaker
    var rateLimiter RateLimiter
    var timeoutManager *TimeoutManager
    var retryManager *RetryManager
    var bulkheadIsolator *BulkheadIsolator

    // 初始化各个组件
    if config.CircuitBreaker != nil {
        circuitBreaker = NewCircuitBreaker(*config.CircuitBreaker)
    }

    if config.RateLimiter != nil {
        rateLimiter = NewTokenBucketLimiter(
            config.RateLimiter.Capacity,
            config.RateLimiter.RefillRate,
        )
    }

    if config.Timeout != nil {
        timeoutManager = NewTimeoutManager(*config.Timeout)
    }

    if config.Retry != nil {
        strategy := &ExponentialBackoffStrategy{
            BaseDelay:   config.Retry.BaseDelay,
            MaxDelay:    config.Retry.MaxDelay,
            Multiplier:  config.Retry.Multiplier,
            MaxAttempts_: config.Retry.MaxAttempts,
            Jitter:      config.Retry.Jitter,
        }
        retryManager = NewRetryManager(strategy, logger)
    }

    if config.Bulkhead != nil {
        bulkheadIsolator = NewBulkheadIsolator()
        // 初始化舱壁...
    }

    return func(c *gin.Context) {
        // 1. 限流检查
        if rateLimiter != nil {
            userID := c.GetHeader("User-ID")
            clientIP := c.ClientIP()

            if !rateLimiter.Allow(userID, clientIP) {
                c.JSON(429, gin.H{
                    "error": "rate limit exceeded",
                    "retry_after": "60s",
                })
                c.Abort()
                return
            }
        }

        // 2. 熔断器保护
        if circuitBreaker != nil {
            _, err := circuitBreaker.Execute(c.Request.Context(), func() (interface{}, error) {
                c.Next()
                if c.Writer.Status() >= 500 {
                    return nil, errors.New("server error")
                }
                return nil, nil
            })

            if err != nil {
                c.JSON(503, gin.H{
                    "error": "service unavailable",
                    "circuit_state": circuitBreaker.State(),
                })
                c.Abort()
                return
            }
        } else {
            c.Next()
        }
    }
}

// 使用示例
func setupGovernanceMiddleware(r *gin.Engine) {
    config := GovernanceConfig{
        CircuitBreaker: &CircuitBreakerConfig{
            MaxRequests:      10,
            Interval:         time.Minute,
            Timeout:          30 * time.Second,
            FailureThreshold: 0.6,
            SuccessThreshold: 5,
        },
        RateLimiter: &RateLimiterConfig{
            Capacity:   100,
            RefillRate: time.Second,
        },
        Timeout: &TimeoutConfig{
            Default: 30 * time.Second,
            Operations: map[string]time.Duration{
                "database": 10 * time.Second,
                "cache":    2 * time.Second,
                "api":      15 * time.Second,
            },
        },
    }

    r.Use(GovernanceMiddleware(config))
}
```

## 监控和指标

### 治理指标收集

```go
// 治理指标
type GovernanceMetrics struct {
    CircuitBreakerMetrics map[string]CircuitMetrics
    RateLimiterMetrics    map[string]RateLimitMetrics
    TimeoutMetrics        map[string]TimeoutStats
    RetryMetrics          map[string]RetryStats
    BulkheadMetrics       map[string]BulkheadStats
}

// 指标收集器
type MetricsCollector struct {
    prometheus *prometheus.Registry
    metrics    GovernanceMetrics
    mutex      sync.RWMutex
}

func NewMetricsCollector() *MetricsCollector {
    collector := &MetricsCollector{
        prometheus: prometheus.NewRegistry(),
        metrics:    GovernanceMetrics{},
    }

    collector.registerPrometheusMetrics()
    return collector
}

func (mc *MetricsCollector) registerPrometheusMetrics() {
    // 熔断器指标
    circuitBreakerState := prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "circuit_breaker_state",
            Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
        },
        []string{"name"},
    )

    circuitBreakerRequests := prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "circuit_breaker_requests_total",
            Help: "Total number of requests through circuit breaker",
        },
        []string{"name", "state", "result"},
    )

    // 限流器指标
    rateLimiterRequests := prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "rate_limiter_requests_total",
            Help: "Total number of rate limiter requests",
        },
        []string{"limiter", "result"},
    )

    // 超时指标
    timeoutDuration := prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "operation_timeout_duration_seconds",
            Help: "Operation timeout duration in seconds",
        },
        []string{"operation"},
    )

    mc.prometheus.MustRegister(
        circuitBreakerState,
        circuitBreakerRequests,
        rateLimiterRequests,
        timeoutDuration,
    )
}

// 健康检查
func (mc *MetricsCollector) HealthCheck() map[string]interface{} {
    mc.mutex.RLock()
    defer mc.mutex.RUnlock()

    health := map[string]interface{}{
        "circuit_breakers": make(map[string]interface{}),
        "rate_limiters":    make(map[string]interface{}),
        "bulkheads":        make(map[string]interface{}),
    }

    // 熔断器健康状态
    for name, metrics := range mc.metrics.CircuitBreakerMetrics {
        health["circuit_breakers"].(map[string]interface{})[name] = map[string]interface{}{
            "state":            metrics.State,
            "failure_rate":     float64(metrics.TotalFailures) / float64(metrics.Requests),
            "requests":         metrics.Requests,
            "consecutive_failures": metrics.ConsecutiveFailures,
        }
    }

    return health
}
```

## 配置示例

### 完整治理配置

```yaml
# governance-config.yaml
governance:
  # 全局开关
  enabled: true

  # 熔断器配置
  circuit_breaker:
    enabled: true
    default_config:
      max_requests: 10
      interval: "1m"
      timeout: "30s"
      failure_threshold: 0.6
      success_threshold: 5

    # 服务特定配置
    services:
      user-service:
        max_requests: 20
        failure_threshold: 0.7
      payment-service:
        max_requests: 5
        failure_threshold: 0.4
        timeout: "10s"

  # 限流配置
  rate_limiter:
    enabled: true
    default_limits:
      requests_per_second: 100
      burst_size: 200

    # 多维度限流
    dimensions:
      # 用户限流
      user:
        requests_per_minute: 1000
        burst_size: 1500

      # IP限流
      ip:
        requests_per_minute: 5000
        burst_size: 7500

      # API限流
      api:
        "/api/users":
          requests_per_second: 50
        "/api/orders":
          requests_per_second: 30

  # 超时控制
  timeout:
    enabled: true
    default_timeout: "30s"
    progressive: true
    backoff_rate: 0.1

    operations:
      database: "10s"
      cache: "2s"
      external_api: "15s"
      file_upload: "60s"

  # 重试机制
  retry:
    enabled: true
    max_attempts: 3
    base_delay: "100ms"
    max_delay: "10s"
    multiplier: 2.0
    jitter: true

    # 可重试错误
    retryable_errors:
      - "network_error"
      - "timeout_error"
      - "service_unavailable"

  # 舱壁模式
  bulkhead:
    enabled: true
    pools:
      database:
        max_active: 50
        max_idle: 10
        idle_timeout: "5m"
        wait: true
        wait_timeout: "30s"

      http_client:
        max_active: 20
        max_idle: 5
        idle_timeout: "2m"
        wait: false

  # 监控配置
  monitoring:
    metrics_enabled: true
    health_check_enabled: true
    prometheus_enabled: true

    # 告警阈值
    alerts:
      circuit_breaker_open: true
      rate_limit_exceeded: true
      timeout_rate_high: 0.1 # 10%
      bulk_head_exhausted: true
```

通过以上完整的服务治理方案，Chi框架能够提供企业级的稳定性保证，确保系统在各种异常情况下都能保持良好的服务质量和用户体验。