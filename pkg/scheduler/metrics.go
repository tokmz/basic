package scheduler

import (
	"sync"
	"time"
)

// SimpleMetricsCollector 简单指标收集器实现
type SimpleMetricsCollector struct {
	counters   map[string]int64
	gauges     map[string]float64
	histograms map[string][]float64
	mutex      sync.RWMutex
}

// NewSimpleMetricsCollector 创建简单指标收集器
func NewSimpleMetricsCollector() MetricsCollector {
	return &SimpleMetricsCollector{
		counters:   make(map[string]int64),
		gauges:     make(map[string]float64),
		histograms: make(map[string][]float64),
	}
}

func (m *SimpleMetricsCollector) IncTasksTotal(taskType string) {
	m.Increment("tasks_total", map[string]string{"type": taskType})
}

func (m *SimpleMetricsCollector) IncTasksSucceeded(taskType string) {
	m.Increment("tasks_succeeded", map[string]string{"type": taskType})
}

func (m *SimpleMetricsCollector) IncTasksFailed(taskType string) {
	m.Increment("tasks_failed", map[string]string{"type": taskType})
}

func (m *SimpleMetricsCollector) IncTasksCanceled(taskType string) {
	m.Increment("tasks_canceled", map[string]string{"type": taskType})
}

func (m *SimpleMetricsCollector) ObserveTaskDuration(taskType string, duration time.Duration) {
	key := m.buildKey("task_duration", map[string]string{"type": taskType})
	m.Histogram(key, duration.Seconds(), nil)
}

func (m *SimpleMetricsCollector) SetActiveTasks(count int) {
	m.Gauge("active_tasks", float64(count), nil)
}

func (m *SimpleMetricsCollector) SetPendingTasks(count int) {
	m.Gauge("pending_tasks", float64(count), nil)
}

func (m *SimpleMetricsCollector) SetRunningTasks(count int) {
	m.Gauge("running_tasks", float64(count), nil)
}

func (m *SimpleMetricsCollector) SetWorkerCount(count int) {
	m.Gauge("worker_count", float64(count), nil)
}

func (m *SimpleMetricsCollector) IncErrors(errorType string) {
	m.Increment("errors_total", map[string]string{"type": errorType})
}

func (m *SimpleMetricsCollector) Increment(name string, labels map[string]string) {
	key := m.buildKey(name, labels)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.counters[key]++
}

func (m *SimpleMetricsCollector) Gauge(name string, value float64, labels map[string]string) {
	key := m.buildKey(name, labels)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.gauges[key] = value
}

func (m *SimpleMetricsCollector) Histogram(name string, value float64, labels map[string]string) {
	key := m.buildKey(name, labels)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if m.histograms[key] == nil {
		m.histograms[key] = make([]float64, 0)
	}
	m.histograms[key] = append(m.histograms[key], value)
	
	// 保持最近1000个样本
	if len(m.histograms[key]) > 1000 {
		m.histograms[key] = m.histograms[key][len(m.histograms[key])-1000:]
	}
}

func (m *SimpleMetricsCollector) GetMetrics() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	metrics := make(map[string]interface{})
	
	// 复制计数器
	counters := make(map[string]int64)
	for k, v := range m.counters {
		counters[k] = v
	}
	metrics["counters"] = counters
	
	// 复制仪表
	gauges := make(map[string]float64)
	for k, v := range m.gauges {
		gauges[k] = v
	}
	metrics["gauges"] = gauges
	
	// 复制直方图统计
	histograms := make(map[string]map[string]float64)
	for k, values := range m.histograms {
		if len(values) == 0 {
			continue
		}
		
		stats := m.calculateHistogramStats(values)
		histograms[k] = stats
	}
	metrics["histograms"] = histograms
	
	return metrics
}

func (m *SimpleMetricsCollector) buildKey(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}
	
	key := name
	for k, v := range labels {
		key += "_" + k + "_" + v
	}
	return key
}

func (m *SimpleMetricsCollector) calculateHistogramStats(values []float64) map[string]float64 {
	if len(values) == 0 {
		return nil
	}
	
	// 排序
	sorted := make([]float64, len(values))
	copy(sorted, values)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	stats := map[string]float64{
		"count": float64(len(values)),
		"min":   sorted[0],
		"max":   sorted[len(sorted)-1],
	}
	
	// 计算平均值
	var sum float64
	for _, v := range values {
		sum += v
	}
	stats["avg"] = sum / float64(len(values))
	
	// 计算百分位数
	stats["p50"] = sorted[len(sorted)*50/100]
	stats["p90"] = sorted[len(sorted)*90/100]
	stats["p95"] = sorted[len(sorted)*95/100]
	stats["p99"] = sorted[len(sorted)*99/100]
	
	return stats
}

// NoOpMetricsCollector 空指标收集器
type NoOpMetricsCollector struct{}

func NewNoOpMetricsCollector() MetricsCollector {
	return &NoOpMetricsCollector{}
}

func (n *NoOpMetricsCollector) IncTasksTotal(taskType string)                        {}
func (n *NoOpMetricsCollector) IncTasksSucceeded(taskType string)                   {}
func (n *NoOpMetricsCollector) IncTasksFailed(taskType string)                      {}
func (n *NoOpMetricsCollector) IncTasksCanceled(taskType string)                    {}
func (n *NoOpMetricsCollector) ObserveTaskDuration(taskType string, duration time.Duration) {}
func (n *NoOpMetricsCollector) SetActiveTasks(count int)                            {}
func (n *NoOpMetricsCollector) SetPendingTasks(count int)                           {}
func (n *NoOpMetricsCollector) SetRunningTasks(count int)                           {}
func (n *NoOpMetricsCollector) SetWorkerCount(count int)                            {}
func (n *NoOpMetricsCollector) IncErrors(errorType string)                          {}
func (n *NoOpMetricsCollector) Increment(name string, labels map[string]string)    {}
func (n *NoOpMetricsCollector) Gauge(name string, value float64, labels map[string]string) {}
func (n *NoOpMetricsCollector) Histogram(name string, value float64, labels map[string]string) {}
func (n *NoOpMetricsCollector) GetMetrics() map[string]interface{}                  { return nil }