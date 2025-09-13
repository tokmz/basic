package chi

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Logger 日志接口
type Logger interface {
	Info(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
}

// Metrics 指标收集器
type Metrics struct {
	mu                sync.RWMutex
	requestCount      int64
	responseTimeSum   int64
	responseTimeCount int64
	errorCount        int64
	statusCounts      map[int]int64
	pathCounts        map[string]int64
	methodCounts      map[string]int64
	
	// 实时指标
	currentRequests int64
	startTime       time.Time
}

// NewMetrics 创建指标收集器
func NewMetrics() *Metrics {
	return &Metrics{
		statusCounts: make(map[int]int64),
		pathCounts:   make(map[string]int64),
		methodCounts: make(map[string]int64),
		startTime:    time.Now(),
	}
}

// MetricsData 指标数据
type MetricsData struct {
	RequestCount     int64             `json:"request_count"`
	ErrorCount       int64             `json:"error_count"`
	AverageResponse  float64           `json:"average_response_ms"`
	CurrentRequests  int64             `json:"current_requests"`
	Uptime           string            `json:"uptime"`
	StatusCounts     map[int]int64     `json:"status_counts"`
	PathCounts       map[string]int64  `json:"path_counts"`
	MethodCounts     map[string]int64  `json:"method_counts"`
	RequestRate      float64           `json:"requests_per_second"`
	ErrorRate        float64           `json:"error_rate_percent"`
}

// Record 记录请求指标
func (m *Metrics) Record(method, path string, status int, duration time.Duration) {
	atomic.AddInt64(&m.requestCount, 1)
	atomic.AddInt64(&m.responseTimeSum, duration.Nanoseconds())
	atomic.AddInt64(&m.responseTimeCount, 1)
	atomic.AddInt64(&m.currentRequests, -1)
	
	if status >= 400 {
		atomic.AddInt64(&m.errorCount, 1)
	}
	
	m.mu.Lock()
	m.statusCounts[status]++
	m.pathCounts[path]++
	m.methodCounts[method]++
	m.mu.Unlock()
}

// IncCurrentRequests 增加当前请求数
func (m *Metrics) IncCurrentRequests() {
	atomic.AddInt64(&m.currentRequests, 1)
}

// GetMetrics 获取指标数据
func (m *Metrics) GetMetrics() MetricsData {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	requestCount := atomic.LoadInt64(&m.requestCount)
	errorCount := atomic.LoadInt64(&m.errorCount)
	responseTimeSum := atomic.LoadInt64(&m.responseTimeSum)
	responseTimeCount := atomic.LoadInt64(&m.responseTimeCount)
	currentRequests := atomic.LoadInt64(&m.currentRequests)
	
	var avgResponse float64
	if responseTimeCount > 0 {
		avgResponse = float64(responseTimeSum/responseTimeCount) / 1e6 // 转换为毫秒
	}
	
	// 计算请求速率
	uptime := time.Since(m.startTime)
	var requestRate float64
	if uptime.Seconds() > 0 {
		requestRate = float64(requestCount) / uptime.Seconds()
	}
	
	// 计算错误率
	var errorRate float64
	if requestCount > 0 {
		errorRate = float64(errorCount) / float64(requestCount) * 100
	}
	
	// 深拷贝map
	statusCounts := make(map[int]int64)
	for k, v := range m.statusCounts {
		statusCounts[k] = v
	}
	
	pathCounts := make(map[string]int64)
	for k, v := range m.pathCounts {
		pathCounts[k] = v
	}
	
	methodCounts := make(map[string]int64)
	for k, v := range m.methodCounts {
		methodCounts[k] = v
	}
	
	return MetricsData{
		RequestCount:    requestCount,
		ErrorCount:      errorCount,
		AverageResponse: avgResponse,
		CurrentRequests: currentRequests,
		Uptime:          uptime.String(),
		StatusCounts:    statusCounts,
		PathCounts:      pathCounts,
		MethodCounts:    methodCounts,
		RequestRate:     requestRate,
		ErrorRate:       errorRate,
	}
}

// Reset 重置指标
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	atomic.StoreInt64(&m.requestCount, 0)
	atomic.StoreInt64(&m.responseTimeSum, 0)
	atomic.StoreInt64(&m.responseTimeCount, 0)
	atomic.StoreInt64(&m.errorCount, 0)
	atomic.StoreInt64(&m.currentRequests, 0)
	
	m.statusCounts = make(map[int]int64)
	m.pathCounts = make(map[string]int64)
	m.methodCounts = make(map[string]int64)
	m.startTime = time.Now()
}

// HealthStatus 健康状态
type HealthStatus struct {
	Status      string                 `json:"status"`
	Timestamp   time.Time              `json:"timestamp"`
	Version     string                 `json:"version"`
	Environment string                 `json:"environment"`
	Uptime      string                 `json:"uptime"`
	Checks      map[string]CheckResult `json:"checks"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// CheckResult 检查结果
type CheckResult struct {
	Status    string                 `json:"status"`
	Message   string                 `json:"message"`
	Duration  time.Duration          `json:"duration"`
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// HealthChecker 健康检查器
type HealthChecker struct {
	mu        sync.RWMutex
	checks    map[string]func() CheckResult
	startTime time.Time
	name      string
	version   string
	env       string
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(name, version, env string) *HealthChecker {
	return &HealthChecker{
		checks:    make(map[string]func() CheckResult),
		startTime: time.Now(),
		name:      name,
		version:   version,
		env:       env,
	}
}

// AddCheck 添加健康检查
func (h *HealthChecker) AddCheck(name string, check func() CheckResult) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks[name] = check
}

// RemoveCheck 移除健康检查
func (h *HealthChecker) RemoveCheck(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.checks, name)
}

// Check 执行健康检查
func (h *HealthChecker) Check() HealthStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	status := "healthy"
	checks := make(map[string]CheckResult)
	
	for name, check := range h.checks {
		start := time.Now()
		result := check()
		result.Duration = time.Since(start)
		result.Timestamp = time.Now()
		
		checks[name] = result
		if result.Status != "healthy" {
			status = "unhealthy"
		}
	}
	
	return HealthStatus{
		Status:      status,
		Timestamp:   time.Now(),
		Version:     h.version,
		Environment: h.env,
		Uptime:      time.Since(h.startTime).String(),
		Checks:      checks,
		Details: map[string]interface{}{
			"app_name": h.name,
		},
	}
}

// AddDatabaseCheck 添加数据库健康检查
func (h *HealthChecker) AddDatabaseCheck(name string, pingFunc func() error) {
	h.AddCheck(name, func() CheckResult {
		err := pingFunc()
		if err != nil {
			return CheckResult{
				Status:  "unhealthy",
				Message: fmt.Sprintf("Database connection failed: %v", err),
				Details: map[string]interface{}{
					"error": err.Error(),
				},
			}
		}
		return CheckResult{
			Status:  "healthy",
			Message: "Database connection successful",
		}
	})
}

// AddRedisCheck 添加Redis健康检查
func (h *HealthChecker) AddRedisCheck(name string, pingFunc func() error) {
	h.AddCheck(name, func() CheckResult {
		err := pingFunc()
		if err != nil {
			return CheckResult{
				Status:  "unhealthy",
				Message: fmt.Sprintf("Redis connection failed: %v", err),
				Details: map[string]interface{}{
					"error": err.Error(),
				},
			}
		}
		return CheckResult{
			Status:  "healthy",
			Message: "Redis connection successful",
		}
	})
}

// AddExternalServiceCheck 添加外部服务健康检查
func (h *HealthChecker) AddExternalServiceCheck(name, url string, checkFunc func(string) error) {
	h.AddCheck(name, func() CheckResult {
		err := checkFunc(url)
		if err != nil {
			return CheckResult{
				Status:  "unhealthy",
				Message: fmt.Sprintf("External service %s is unavailable: %v", name, err),
				Details: map[string]interface{}{
					"url":   url,
					"error": err.Error(),
				},
			}
		}
		return CheckResult{
			Status:  "healthy",
			Message: fmt.Sprintf("External service %s is available", name),
			Details: map[string]interface{}{
				"url": url,
			},
		}
	})
}

// DefaultLogger 默认日志器
type DefaultLogger struct{}

func (l *DefaultLogger) Info(msg string, fields ...interface{}) {
	fmt.Printf("[INFO] %s %v\n", msg, fields)
}

func (l *DefaultLogger) Error(msg string, fields ...interface{}) {
	fmt.Printf("[ERROR] %s %v\n", msg, fields)
}

func (l *DefaultLogger) Warn(msg string, fields ...interface{}) {
	fmt.Printf("[WARN] %s %v\n", msg, fields)
}

func (l *DefaultLogger) Debug(msg string, fields ...interface{}) {
	fmt.Printf("[DEBUG] %s %v\n", msg, fields)
}

// MemoryStats 内存统计
type MemoryStats struct {
	Alloc      uint64 `json:"alloc"`
	TotalAlloc uint64 `json:"total_alloc"`
	Sys        uint64 `json:"sys"`
	NumGC      uint32 `json:"num_gc"`
}

// GetMemoryStats 获取内存统计（需要导入runtime包时实现）
func GetMemoryStats() MemoryStats {
	// 这里应该使用runtime.ReadMemStats实现
	// 为了避免导入runtime包，这里返回空值
	return MemoryStats{}
}

// Performance 性能监控
type Performance struct {
	mu           sync.RWMutex
	slowRequests []SlowRequest
	maxSlowReqs  int
}

// SlowRequest 慢请求记录
type SlowRequest struct {
	Method    string        `json:"method"`
	Path      string        `json:"path"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
	UserAgent string        `json:"user_agent"`
	IP        string        `json:"ip"`
}

// NewPerformance 创建性能监控
func NewPerformance(maxSlowReqs int) *Performance {
	return &Performance{
		slowRequests: make([]SlowRequest, 0),
		maxSlowReqs:  maxSlowReqs,
	}
}

// RecordSlowRequest 记录慢请求
func (p *Performance) RecordSlowRequest(method, path, userAgent, ip string, duration time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	slowReq := SlowRequest{
		Method:    method,
		Path:      path,
		Duration:  duration,
		Timestamp: time.Now(),
		UserAgent: userAgent,
		IP:        ip,
	}
	
	p.slowRequests = append(p.slowRequests, slowReq)
	
	// 保持最大记录数
	if len(p.slowRequests) > p.maxSlowReqs {
		p.slowRequests = p.slowRequests[1:]
	}
}

// GetSlowRequests 获取慢请求列表
func (p *Performance) GetSlowRequests() []SlowRequest {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	// 返回副本
	result := make([]SlowRequest, len(p.slowRequests))
	copy(result, p.slowRequests)
	return result
}

// ClearSlowRequests 清空慢请求记录
func (p *Performance) ClearSlowRequests() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.slowRequests = p.slowRequests[:0]
}