package cache

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// Monitor 缓存监控器
type Monitor struct {
	config     MonitoringConfig
	logger     *zap.Logger
	metrics    *Metrics
	collectors []Collector
	mu         sync.RWMutex
	stopCh     chan struct{}
	ticker     *time.Ticker
	started    int32
}

// Metrics 缓存指标
type Metrics struct {
	// 基础指标
	TotalRequests int64 `json:"total_requests"`
	Hits          int64 `json:"hits"`
	Misses        int64 `json:"misses"`
	Sets          int64 `json:"sets"`
	Deletes       int64 `json:"deletes"`
	Errors        int64 `json:"errors"`

	// 性能指标
	AvgResponseTime time.Duration `json:"avg_response_time"`
	MaxResponseTime time.Duration `json:"max_response_time"`
	MinResponseTime time.Duration `json:"min_response_time"`
	SlowQueries     int64         `json:"slow_queries"`

	// 资源指标
	MemoryUsage     int64   `json:"memory_usage"`
	CPUUsage        float64 `json:"cpu_usage"`
	ConnectionCount int64   `json:"connection_count"`

	// 统计指标
	HitRate    float64 `json:"hit_rate"`
	ErrorRate  float64 `json:"error_rate"`
	Throughput float64 `json:"throughput"` // QPS

	// 时间戳
	Timestamp time.Time `json:"timestamp"`
}

// Collector 指标收集器接口
type Collector interface {
	Collect() (*Metrics, error)
	Name() string
}

// NewMonitor 创建监控器
func NewMonitor(config MonitoringConfig, logger *zap.Logger) *Monitor {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Monitor{
		config:     config,
		logger:     logger,
		metrics:    &Metrics{},
		collectors: make([]Collector, 0),
		stopCh:     make(chan struct{}),
	}
}

// AddCollector 添加指标收集器
func (m *Monitor) AddCollector(collector Collector) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.collectors = append(m.collectors, collector)
}

// Start 启动监控
func (m *Monitor) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return nil
	}

	if !m.config.EnableMetrics {
		return nil
	}

	// 启动定时收集
	if m.config.MetricsInterval > 0 {
		m.ticker = time.NewTicker(m.config.MetricsInterval)
		go m.collectLoop(ctx)
	}

	m.logger.Info("Cache monitor started",
		zap.Duration("interval", m.config.MetricsInterval),
		zap.Bool("metrics_enabled", m.config.EnableMetrics),
		zap.Bool("logging_enabled", m.config.EnableLogging),
	)

	return nil
}

// Stop 停止监控
func (m *Monitor) Stop() error {
	if !atomic.CompareAndSwapInt32(&m.started, 1, 0) {
		return nil
	}

	if m.ticker != nil {
		m.ticker.Stop()
	}

	close(m.stopCh)

	m.logger.Info("Cache monitor stopped")
	return nil
}

// RecordRequest 记录请求
func (m *Monitor) RecordRequest(operation string, duration time.Duration, err error) {
	if !m.config.EnableMetrics {
		return
	}

	atomic.AddInt64(&m.metrics.TotalRequests, 1)

	// 记录操作类型
	switch operation {
	case "get":
		if err == nil {
			atomic.AddInt64(&m.metrics.Hits, 1)
		} else if err == ErrKeyNotFound {
			atomic.AddInt64(&m.metrics.Misses, 1)
		} else {
			atomic.AddInt64(&m.metrics.Errors, 1)
		}
	case "set":
		if err == nil {
			atomic.AddInt64(&m.metrics.Sets, 1)
		} else {
			atomic.AddInt64(&m.metrics.Errors, 1)
		}
	case "delete":
		if err == nil {
			atomic.AddInt64(&m.metrics.Deletes, 1)
		} else {
			atomic.AddInt64(&m.metrics.Errors, 1)
		}
	}

	// 记录错误
	if err != nil {
		atomic.AddInt64(&m.metrics.Errors, 1)
		if m.config.EnableLogging {
			m.logger.Error("Cache operation failed",
				zap.String("operation", operation),
				zap.Duration("duration", duration),
				zap.Error(err),
			)
		}
	}

	// 记录慢查询
	if duration > m.config.SlowThreshold {
		atomic.AddInt64(&m.metrics.SlowQueries, 1)
		if m.config.EnableLogging {
			m.logger.Warn("Slow cache operation detected",
				zap.String("operation", operation),
				zap.Duration("duration", duration),
				zap.Duration("threshold", m.config.SlowThreshold),
			)
		}
	}

	// 更新响应时间统计
	m.updateResponseTime(duration)
}

// GetMetrics 获取当前指标
func (m *Monitor) GetMetrics() *Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 复制指标
	metrics := &Metrics{
		TotalRequests:   atomic.LoadInt64(&m.metrics.TotalRequests),
		Hits:            atomic.LoadInt64(&m.metrics.Hits),
		Misses:          atomic.LoadInt64(&m.metrics.Misses),
		Sets:            atomic.LoadInt64(&m.metrics.Sets),
		Deletes:         atomic.LoadInt64(&m.metrics.Deletes),
		Errors:          atomic.LoadInt64(&m.metrics.Errors),
		SlowQueries:     atomic.LoadInt64(&m.metrics.SlowQueries),
		AvgResponseTime: m.metrics.AvgResponseTime,
		MaxResponseTime: m.metrics.MaxResponseTime,
		MinResponseTime: m.metrics.MinResponseTime,
		MemoryUsage:     atomic.LoadInt64(&m.metrics.MemoryUsage),
		ConnectionCount: atomic.LoadInt64(&m.metrics.ConnectionCount),
		CPUUsage:        m.metrics.CPUUsage,
		Timestamp:       time.Now(),
	}

	// 计算派生指标
	if metrics.TotalRequests > 0 {
		metrics.HitRate = float64(metrics.Hits) / float64(metrics.TotalRequests)
		metrics.ErrorRate = float64(metrics.Errors) / float64(metrics.TotalRequests)
	}

	return metrics
}

// Reset 重置指标
func (m *Monitor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	atomic.StoreInt64(&m.metrics.TotalRequests, 0)
	atomic.StoreInt64(&m.metrics.Hits, 0)
	atomic.StoreInt64(&m.metrics.Misses, 0)
	atomic.StoreInt64(&m.metrics.Sets, 0)
	atomic.StoreInt64(&m.metrics.Deletes, 0)
	atomic.StoreInt64(&m.metrics.Errors, 0)
	atomic.StoreInt64(&m.metrics.SlowQueries, 0)
	atomic.StoreInt64(&m.metrics.MemoryUsage, 0)
	atomic.StoreInt64(&m.metrics.ConnectionCount, 0)

	m.metrics.AvgResponseTime = 0
	m.metrics.MaxResponseTime = 0
	m.metrics.MinResponseTime = 0
	m.metrics.CPUUsage = 0
}

// 私有方法

// collectLoop 收集循环
func (m *Monitor) collectLoop(ctx context.Context) {
	for {
		select {
		case <-m.ticker.C:
			m.collect(ctx)
		case <-m.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// collect 收集指标
func (m *Monitor) collect(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 从收集器收集指标
	for _, collector := range m.collectors {
		metrics, err := collector.Collect()
		if err != nil {
			m.logger.Error("Failed to collect metrics",
				zap.String("collector", collector.Name()),
				zap.Error(err),
			)
			continue
		}

		// 合并指标
		m.mergeMetrics(metrics)
	}

	// 记录指标日志
	if m.config.EnableLogging {
		currentMetrics := m.GetMetrics()
		m.logger.Info("Cache metrics collected",
			zap.Int64("total_requests", currentMetrics.TotalRequests),
			zap.Int64("hits", currentMetrics.Hits),
			zap.Int64("misses", currentMetrics.Misses),
			zap.Float64("hit_rate", currentMetrics.HitRate),
			zap.Int64("errors", currentMetrics.Errors),
			zap.Float64("error_rate", currentMetrics.ErrorRate),
			zap.Duration("avg_response_time", currentMetrics.AvgResponseTime),
			zap.Int64("slow_queries", currentMetrics.SlowQueries),
		)
	}
}

// mergeMetrics 合并指标
func (m *Monitor) mergeMetrics(metrics *Metrics) {
	if metrics == nil {
		return
	}

	// 累加计数指标
	atomic.AddInt64(&m.metrics.MemoryUsage, metrics.MemoryUsage)
	atomic.AddInt64(&m.metrics.ConnectionCount, metrics.ConnectionCount)

	// 更新CPU使用率
	m.metrics.CPUUsage = metrics.CPUUsage
}

// updateResponseTime 更新响应时间统计
func (m *Monitor) updateResponseTime(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 更新最大响应时间
	if duration > m.metrics.MaxResponseTime {
		m.metrics.MaxResponseTime = duration
	}

	// 更新最小响应时间
	if m.metrics.MinResponseTime == 0 || duration < m.metrics.MinResponseTime {
		m.metrics.MinResponseTime = duration
	}

	// 简单的平均响应时间计算(可以改进为移动平均)
	totalRequests := atomic.LoadInt64(&m.metrics.TotalRequests)
	if totalRequests > 0 {
		currentAvg := m.metrics.AvgResponseTime
		m.metrics.AvgResponseTime = time.Duration(
			(int64(currentAvg)*(totalRequests-1) + int64(duration)) / totalRequests,
		)
	}
}

// CacheCollector 缓存收集器
type CacheCollector struct {
	name  string
	cache Cache
}

// NewCacheCollector 创建缓存收集器
func NewCacheCollector(name string, cache Cache) *CacheCollector {
	return &CacheCollector{
		name:  name,
		cache: cache,
	}
}

// Name 返回收集器名称
func (c *CacheCollector) Name() string {
	return c.name
}

// Collect 收集缓存指标
func (c *CacheCollector) Collect() (*Metrics, error) {
	stats := c.cache.Stats()

	return &Metrics{
		TotalRequests: stats.TotalRequests,
		Hits:          stats.Hits,
		Misses:        stats.Misses,
		Sets:          stats.Sets,
		Deletes:       stats.Deletes,
		Errors:        stats.Errors,
		HitRate:       stats.HitRate,
		Timestamp:     time.Now(),
	}, nil
}
