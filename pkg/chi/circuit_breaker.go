package chi

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// CircuitState 熔断器状态
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreakerConfig 熔断器配置
type CircuitBreakerConfig struct {
	Name                string        `json:"name"`
	MaxRequests         uint32        `json:"max_requests"`          // 半开状态下的最大请求数
	Interval            time.Duration `json:"interval"`              // 统计时间窗口
	Timeout             time.Duration `json:"timeout"`               // 熔断超时时间
	ReadyToTrip         func(counts Counts) bool `json:"-"`     // 触发熔断的条件
	OnStateChange       func(name string, from CircuitState, to CircuitState) `json:"-"` // 状态变化回调
	IsSuccessful        func(err error) bool `json:"-"`        // 判断是否成功的函数
}

// DefaultCircuitBreakerConfig 默认熔断器配置
func DefaultCircuitBreakerConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:        name,
		MaxRequests: 1,
		Interval:    time.Duration(0), // 不重置计数器
		Timeout:     60 * time.Second,
		ReadyToTrip: func(counts Counts) bool {
			return counts.Requests >= 3 && counts.TotalFailures >= 3
		},
		OnStateChange: func(name string, from CircuitState, to CircuitState) {},
		IsSuccessful: func(err error) bool {
			return err == nil
		},
	}
}

// Counts 请求计数
type Counts struct {
	Requests             uint32 `json:"requests"`
	TotalSuccesses       uint32 `json:"total_successes"`
	TotalFailures        uint32 `json:"total_failures"`
	ConsecutiveSuccesses uint32 `json:"consecutive_successes"`
	ConsecutiveFailures  uint32 `json:"consecutive_failures"`
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	name          string
	maxRequests   uint32
	interval      time.Duration
	timeout       time.Duration
	readyToTrip   func(counts Counts) bool
	onStateChange func(name string, from CircuitState, to CircuitState)
	isSuccessful  func(err error) bool

	mutex      sync.RWMutex
	state      CircuitState
	generation uint64
	counts     Counts
	expiry     time.Time
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:          config.Name,
		maxRequests:   config.MaxRequests,
		interval:      config.Interval,
		timeout:       config.Timeout,
		readyToTrip:   config.ReadyToTrip,
		onStateChange: config.OnStateChange,
		isSuccessful:  config.IsSuccessful,
	}

	cb.toNewGeneration(time.Now())
	return cb
}

// Name 获取熔断器名称
func (cb *CircuitBreaker) Name() string {
	return cb.name
}

// State 获取当前状态
func (cb *CircuitBreaker) State() CircuitState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// Counts 获取计数信息
func (cb *CircuitBreaker) Counts() Counts {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.counts
}

// Execute 执行函数
func (cb *CircuitBreaker) Execute(req func() (interface{}, error)) (interface{}, error) {
	generation, err := cb.beforeRequest()
	if err != nil {
		return nil, err
	}

	defer func() {
		e := recover()
		if e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()

	result, err := req()
	cb.afterRequest(generation, cb.isSuccessful(err))
	return result, err
}

// Call 调用函数（简化版Execute）
func (cb *CircuitBreaker) Call(fn func() error) error {
	_, err := cb.Execute(func() (interface{}, error) {
		return nil, fn()
	})
	return err
}

// beforeRequest 请求前检查
func (cb *CircuitBreaker) beforeRequest() (uint64, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)

	if state == StateOpen {
		return generation, errors.New("circuit breaker is open")
	} else if state == StateHalfOpen && cb.counts.Requests >= cb.maxRequests {
		return generation, errors.New("too many requests")
	}

	cb.counts.Requests++
	return generation, nil
}

// afterRequest 请求后处理
func (cb *CircuitBreaker) afterRequest(before uint64, success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)
	if generation != before {
		return
	}

	if success {
		cb.onSuccess(state, now)
	} else {
		cb.onFailure(state, now)
	}
}

// onSuccess 成功处理
func (cb *CircuitBreaker) onSuccess(state CircuitState, now time.Time) {
	cb.counts.TotalSuccesses++
	cb.counts.ConsecutiveSuccesses++
	cb.counts.ConsecutiveFailures = 0

	if state == StateHalfOpen && cb.counts.ConsecutiveSuccesses >= cb.maxRequests {
		cb.setState(StateClosed, now)
	}
}

// onFailure 失败处理
func (cb *CircuitBreaker) onFailure(state CircuitState, now time.Time) {
	cb.counts.TotalFailures++
	cb.counts.ConsecutiveFailures++
	cb.counts.ConsecutiveSuccesses = 0

	if state == StateClosed && cb.readyToTrip(cb.counts) {
		cb.setState(StateOpen, now)
	} else if state == StateHalfOpen {
		cb.setState(StateOpen, now)
	}
}

// currentState 获取当前状态
func (cb *CircuitBreaker) currentState(now time.Time) (CircuitState, uint64) {
	switch cb.state {
	case StateClosed:
		if !cb.expiry.IsZero() && cb.expiry.Before(now) {
			cb.toNewGeneration(now)
		}
	case StateOpen:
		if cb.expiry.Before(now) {
			cb.setState(StateHalfOpen, now)
		}
	}
	return cb.state, cb.generation
}

// setState 设置状态
func (cb *CircuitBreaker) setState(state CircuitState, now time.Time) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state
	cb.toNewGeneration(now)

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, prev, state)
	}
}

// toNewGeneration 开始新一代
func (cb *CircuitBreaker) toNewGeneration(now time.Time) {
	cb.generation++
	cb.counts = Counts{}

	var zero time.Time
	switch cb.state {
	case StateClosed:
		if cb.interval == 0 {
			cb.expiry = zero
		} else {
			cb.expiry = now.Add(cb.interval)
		}
	case StateOpen:
		cb.expiry = now.Add(cb.timeout)
	default: // StateHalfOpen
		cb.expiry = zero
	}
}

// CircuitBreakerMiddleware 熔断器中间件
func CircuitBreakerMiddleware(cb *CircuitBreaker) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := cb.Call(func() error {
			c.Next()
			
			// 检查响应状态码
			if c.Writer.Status() >= 500 {
				return fmt.Errorf("server error: %d", c.Writer.Status())
			}
			return nil
		})
		
		if err != nil {
			c.JSON(503, gin.H{
				"error":           "Service temporarily unavailable",
				"circuit_breaker": cb.Name(),
				"state":          cb.State().String(),
				"details":        err.Error(),
			})
			c.Abort()
		}
	}
}

// CircuitBreakerManager 熔断器管理器
type CircuitBreakerManager struct {
	mu       sync.RWMutex
	breakers map[string]*CircuitBreaker
}

// NewCircuitBreakerManager 创建熔断器管理器
func NewCircuitBreakerManager() *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*CircuitBreaker),
	}
}

// Register 注册熔断器
func (cbm *CircuitBreakerManager) Register(name string, config CircuitBreakerConfig) *CircuitBreaker {
	cbm.mu.Lock()
	defer cbm.mu.Unlock()
	
	config.Name = name
	cb := NewCircuitBreaker(config)
	cbm.breakers[name] = cb
	return cb
}

// Get 获取熔断器
func (cbm *CircuitBreakerManager) Get(name string) *CircuitBreaker {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()
	
	return cbm.breakers[name]
}

// GetOrCreate 获取或创建熔断器
func (cbm *CircuitBreakerManager) GetOrCreate(name string, config *CircuitBreakerConfig) *CircuitBreaker {
	cbm.mu.Lock()
	defer cbm.mu.Unlock()
	
	if cb, exists := cbm.breakers[name]; exists {
		return cb
	}
	
	if config == nil {
		defaultConfig := DefaultCircuitBreakerConfig(name)
		config = &defaultConfig
	}
	
	config.Name = name
	cb := NewCircuitBreaker(*config)
	cbm.breakers[name] = cb
	return cb
}

// Remove 移除熔断器
func (cbm *CircuitBreakerManager) Remove(name string) {
	cbm.mu.Lock()
	defer cbm.mu.Unlock()
	
	delete(cbm.breakers, name)
}

// List 列出所有熔断器
func (cbm *CircuitBreakerManager) List() map[string]*CircuitBreaker {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()
	
	result := make(map[string]*CircuitBreaker)
	for name, cb := range cbm.breakers {
		result[name] = cb
	}
	return result
}

// GetStats 获取所有熔断器统计
func (cbm *CircuitBreakerManager) GetStats() map[string]CircuitBreakerStats {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()
	
	stats := make(map[string]CircuitBreakerStats)
	for name, cb := range cbm.breakers {
		stats[name] = CircuitBreakerStats{
			Name:   name,
			State:  cb.State().String(),
			Counts: cb.Counts(),
		}
	}
	return stats
}

// CircuitBreakerStats 熔断器统计
type CircuitBreakerStats struct {
	Name   string        `json:"name"`
	State  string        `json:"state"`
	Counts Counts        `json:"counts"`
}

// RouteCircuitBreakerMiddleware 路由级别熔断器中间件
func RouteCircuitBreakerMiddleware(manager *CircuitBreakerManager, config *CircuitBreakerConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}
		
		// 为每个路由创建独立的熔断器
		cb := manager.GetOrCreate(route, config)
		
		err := cb.Call(func() error {
			c.Next()
			
			// 检查响应状态码
			if c.Writer.Status() >= 500 {
				return fmt.Errorf("server error: %d", c.Writer.Status())
			}
			return nil
		})
		
		if err != nil {
			c.JSON(503, gin.H{
				"error":           "Service temporarily unavailable",
				"route":          route,
				"circuit_breaker": cb.Name(),
				"state":          cb.State().String(),
				"details":        err.Error(),
			})
			c.Abort()
		}
	}
}

// Global circuit breaker manager
var GlobalCircuitBreakerManager = NewCircuitBreakerManager()

// DefaultCircuitBreaker 创建默认熔断器中间件
func DefaultCircuitBreaker(name string) gin.HandlerFunc {
	cb := GlobalCircuitBreakerManager.GetOrCreate(name, nil)
	return CircuitBreakerMiddleware(cb)
}

// BulkheadPattern 舱壁模式实现
type BulkheadExecutor struct {
	name     string
	maxConcurrent int
	semaphore chan struct{}
	stats     BulkheadStats
	mu        sync.RWMutex
}

// BulkheadStats 舱壁统计
type BulkheadStats struct {
	Name            string `json:"name"`
	MaxConcurrent   int    `json:"max_concurrent"`
	ActiveRequests  int    `json:"active_requests"`
	RejectedRequests int64  `json:"rejected_requests"`
	CompletedRequests int64 `json:"completed_requests"`
}

// NewBulkheadExecutor 创建舱壁执行器
func NewBulkheadExecutor(name string, maxConcurrent int) *BulkheadExecutor {
	return &BulkheadExecutor{
		name:          name,
		maxConcurrent: maxConcurrent,
		semaphore:     make(chan struct{}, maxConcurrent),
		stats: BulkheadStats{
			Name:          name,
			MaxConcurrent: maxConcurrent,
		},
	}
}

// Execute 执行函数
func (be *BulkheadExecutor) Execute(ctx context.Context, fn func() error) error {
	select {
	case be.semaphore <- struct{}{}:
		// 获得执行权限
		defer func() {
			<-be.semaphore
			be.mu.Lock()
			be.stats.ActiveRequests--
			be.stats.CompletedRequests++
			be.mu.Unlock()
		}()
		
		be.mu.Lock()
		be.stats.ActiveRequests++
		be.mu.Unlock()
		
		return fn()
		
	case <-ctx.Done():
		be.mu.Lock()
		be.stats.RejectedRequests++
		be.mu.Unlock()
		return ctx.Err()
		
	default:
		// 没有可用资源
		be.mu.Lock()
		be.stats.RejectedRequests++
		be.mu.Unlock()
		return errors.New("bulkhead capacity exceeded")
	}
}

// GetStats 获取舱壁统计
func (be *BulkheadExecutor) GetStats() BulkheadStats {
	be.mu.RLock()
	defer be.mu.RUnlock()
	return be.stats
}

// BulkheadMiddleware 舱壁中间件
func BulkheadMiddleware(executor *BulkheadExecutor) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := executor.Execute(c.Request.Context(), func() error {
			c.Next()
			return nil
		})
		
		if err != nil {
			c.JSON(429, gin.H{
				"error":     "Resource capacity exceeded",
				"bulkhead":  executor.name,
				"details":   err.Error(),
			})
			c.Abort()
		}
	}
}