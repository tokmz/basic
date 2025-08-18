package monitoring

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Monitor Redis性能监控器
type Monitor struct {
	namespace string
	enabled   bool

	// Prometheus指标
	operationCounter    *prometheus.CounterVec
	operationDuration   *prometheus.HistogramVec
	connectionPoolGauge *prometheus.GaugeVec
	errorCounter        *prometheus.CounterVec
	slowQueryCounter    *prometheus.CounterVec

	// 慢查询配置
	slowLogThreshold time.Duration
	slowQueries      []SlowQuery
	slowQueryMutex   sync.RWMutex

	// 统计数据
	stats      *Stats
	statsMutex sync.RWMutex
}

// Stats 统计数据
type Stats struct {
	TotalOperations   int64              `json:"total_operations"`
	SuccessOperations int64              `json:"success_operations"`
	FailedOperations  int64              `json:"failed_operations"`
	TotalDuration     time.Duration      `json:"total_duration"`
	AverageDuration   time.Duration      `json:"average_duration"`
	OperationStats    map[string]*OpStat `json:"operation_stats"`
	SlowQueryCount    int64              `json:"slow_query_count"`
	LastUpdated       time.Time          `json:"last_updated"`
}

// OpStat 操作统计
type OpStat struct {
	Count        int64         `json:"count"`
	SuccessCount int64         `json:"success_count"`
	FailedCount  int64         `json:"failed_count"`
	TotalTime    time.Duration `json:"total_time"`
	AvgTime      time.Duration `json:"avg_time"`
	MinTime      time.Duration `json:"min_time"`
	MaxTime      time.Duration `json:"max_time"`
}

// SlowQuery 慢查询记录
type SlowQuery struct {
	Operation string        `json:"operation"`
	Key       string        `json:"key"`
	Args      []interface{} `json:"args"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
	Error     string        `json:"error,omitempty"`
}

// NewMonitor 创建新的监控器
func NewMonitor(namespace string, enabled bool, slowLogThreshold time.Duration) *Monitor {
	m := &Monitor{
		namespace:        namespace,
		enabled:          enabled,
		slowLogThreshold: slowLogThreshold,
		slowQueries:      make([]SlowQuery, 0),
		stats: &Stats{
			OperationStats: make(map[string]*OpStat),
			LastUpdated:    time.Now(),
		},
	}

	if enabled {
		m.initPrometheusMetrics()
	}

	return m
}

// initPrometheusMetrics 初始化Prometheus指标
func (m *Monitor) initPrometheusMetrics() {
	// 操作计数器
	m.operationCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: m.namespace,
			Name:      "operations_total",
			Help:      "Total number of Redis operations",
		},
		[]string{"operation", "status"},
	)

	// 操作耗时直方图
	m.operationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: m.namespace,
			Name:      "operation_duration_seconds",
			Help:      "Duration of Redis operations in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	// 连接池状态
	m.connectionPoolGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: m.namespace,
			Name:      "connection_pool",
			Help:      "Redis connection pool status",
		},
		[]string{"type"},
	)

	// 错误计数器
	m.errorCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: m.namespace,
			Name:      "errors_total",
			Help:      "Total number of Redis errors",
		},
		[]string{"operation", "error_type"},
	)

	// 慢查询计数器
	m.slowQueryCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: m.namespace,
			Name:      "slow_queries_total",
			Help:      "Total number of slow Redis queries",
		},
		[]string{"operation"},
	)
}

// RecordOperation 记录操作指标
func (m *Monitor) RecordOperation(ctx context.Context, operation, key string, duration time.Duration, err error, args ...interface{}) {
	if !m.enabled {
		return
	}

	// 更新统计数据
	m.updateStats(operation, duration, err)

	// 记录Prometheus指标
	if m.operationCounter != nil {
		status := "success"
		if err != nil {
			status = "error"
		}
		m.operationCounter.WithLabelValues(operation, status).Inc()
	}

	if m.operationDuration != nil {
		m.operationDuration.WithLabelValues(operation).Observe(duration.Seconds())
	}

	// 记录错误
	if err != nil && m.errorCounter != nil {
		errorType := "unknown"
		if err != nil {
			errorType = "redis_error"
		}
		m.errorCounter.WithLabelValues(operation, errorType).Inc()
	}

	// 检查慢查询
	if duration >= m.slowLogThreshold {
		m.recordSlowQuery(operation, key, duration, err, args...)
	}
}

// updateStats 更新统计数据
func (m *Monitor) updateStats(operation string, duration time.Duration, err error) {
	m.statsMutex.Lock()
	defer m.statsMutex.Unlock()

	m.stats.TotalOperations++
	m.stats.TotalDuration += duration
	m.stats.AverageDuration = m.stats.TotalDuration / time.Duration(m.stats.TotalOperations)
	m.stats.LastUpdated = time.Now()

	if err == nil {
		m.stats.SuccessOperations++
	} else {
		m.stats.FailedOperations++
	}

	// 更新操作级别统计
	if m.stats.OperationStats[operation] == nil {
		m.stats.OperationStats[operation] = &OpStat{
			MinTime: duration,
			MaxTime: duration,
		}
	}

	opStat := m.stats.OperationStats[operation]
	opStat.Count++
	opStat.TotalTime += duration
	opStat.AvgTime = opStat.TotalTime / time.Duration(opStat.Count)

	if err == nil {
		opStat.SuccessCount++
	} else {
		opStat.FailedCount++
	}

	if duration < opStat.MinTime {
		opStat.MinTime = duration
	}
	if duration > opStat.MaxTime {
		opStat.MaxTime = duration
	}
}

// recordSlowQuery 记录慢查询
func (m *Monitor) recordSlowQuery(operation, key string, duration time.Duration, err error, args ...interface{}) {
	m.slowQueryMutex.Lock()
	defer m.slowQueryMutex.Unlock()

	slowQuery := SlowQuery{
		Operation: operation,
		Key:       key,
		Args:      args,
		Duration:  duration,
		Timestamp: time.Now(),
	}

	if err != nil {
		slowQuery.Error = err.Error()
	}

	m.slowQueries = append(m.slowQueries, slowQuery)

	// 限制慢查询记录数量
	if len(m.slowQueries) > 1000 {
		m.slowQueries = m.slowQueries[len(m.slowQueries)-500:]
	}

	m.stats.SlowQueryCount++

	// 记录Prometheus指标
	if m.slowQueryCounter != nil {
		m.slowQueryCounter.WithLabelValues(operation).Inc()
	}
}

// UpdateConnectionPool 更新连接池指标
func (m *Monitor) UpdateConnectionPool(totalConns, idleConns, activeConns int) {
	if !m.enabled || m.connectionPoolGauge == nil {
		return
	}

	m.connectionPoolGauge.WithLabelValues("total").Set(float64(totalConns))
	m.connectionPoolGauge.WithLabelValues("idle").Set(float64(idleConns))
	m.connectionPoolGauge.WithLabelValues("active").Set(float64(activeConns))
}

// GetStats 获取统计数据
func (m *Monitor) GetStats() *Stats {
	m.statsMutex.RLock()
	defer m.statsMutex.RUnlock()

	// 深拷贝统计数据
	stats := &Stats{
		TotalOperations:   m.stats.TotalOperations,
		SuccessOperations: m.stats.SuccessOperations,
		FailedOperations:  m.stats.FailedOperations,
		TotalDuration:     m.stats.TotalDuration,
		AverageDuration:   m.stats.AverageDuration,
		SlowQueryCount:    m.stats.SlowQueryCount,
		LastUpdated:       m.stats.LastUpdated,
		OperationStats:    make(map[string]*OpStat),
	}

	for op, opStat := range m.stats.OperationStats {
		stats.OperationStats[op] = &OpStat{
			Count:        opStat.Count,
			SuccessCount: opStat.SuccessCount,
			FailedCount:  opStat.FailedCount,
			TotalTime:    opStat.TotalTime,
			AvgTime:      opStat.AvgTime,
			MinTime:      opStat.MinTime,
			MaxTime:      opStat.MaxTime,
		}
	}

	return stats
}

// GetSlowQueries 获取慢查询记录
func (m *Monitor) GetSlowQueries(limit int) []SlowQuery {
	m.slowQueryMutex.RLock()
	defer m.slowQueryMutex.RUnlock()

	if limit <= 0 || limit > len(m.slowQueries) {
		limit = len(m.slowQueries)
	}

	// 返回最新的慢查询记录
	start := len(m.slowQueries) - limit
	if start < 0 {
		start = 0
	}

	result := make([]SlowQuery, limit)
	copy(result, m.slowQueries[start:])
	return result
}

// ResetStats 重置统计数据
func (m *Monitor) ResetStats() {
	m.statsMutex.Lock()
	defer m.statsMutex.Unlock()

	m.stats = &Stats{
		OperationStats: make(map[string]*OpStat),
		LastUpdated:    time.Now(),
	}
}

// ClearSlowQueries 清空慢查询记录
func (m *Monitor) ClearSlowQueries() {
	m.slowQueryMutex.Lock()
	defer m.slowQueryMutex.Unlock()

	m.slowQueries = make([]SlowQuery, 0)
}

// SetSlowLogThreshold 设置慢查询阈值
func (m *Monitor) SetSlowLogThreshold(threshold time.Duration) {
	m.slowLogThreshold = threshold
}

// IsEnabled 检查监控是否启用
func (m *Monitor) IsEnabled() bool {
	return m.enabled
}

// Enable 启用监控
func (m *Monitor) Enable() {
	m.enabled = true
	if m.operationCounter == nil {
		m.initPrometheusMetrics()
	}
}

// Disable 禁用监控
func (m *Monitor) Disable() {
	m.enabled = false
}
