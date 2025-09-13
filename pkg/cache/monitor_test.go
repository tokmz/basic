package cache

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

// MockCollector 模拟收集器
type MockCollector struct {
	name    string
	metrics *Metrics
	err     error
}

func (m *MockCollector) Name() string {
	return m.name
}

func (m *MockCollector) Collect() (*Metrics, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.metrics, nil
}

// MockCache 模拟缓存
type MockCache struct {
	stats CacheStats
}

func (m *MockCache) Get(ctx context.Context, key string) (interface{}, error) {
	return nil, ErrKeyNotFound
}

func (m *MockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return nil
}

func (m *MockCache) Delete(ctx context.Context, key string) error {
	return nil
}

func (m *MockCache) Exists(ctx context.Context, key string) (bool, error) {
	return false, nil
}

func (m *MockCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	return nil, nil
}

func (m *MockCache) Clear(ctx context.Context) error {
	return nil
}

func (m *MockCache) Stats() CacheStats {
	return m.stats
}

func (m *MockCache) Close() error {
	return nil
}

func TestNewMonitor(t *testing.T) {
	config := MonitoringConfig{
		EnableMetrics:   true,
		MetricsInterval: 30 * time.Second,
		EnableLogging:   true,
		LogLevel:        "info",
		SlowThreshold:   100 * time.Millisecond,
	}

	tests := []struct {
		name   string
		config MonitoringConfig
		logger *zap.Logger
	}{
		{
			name:   "With logger",
			config: config,
			logger: zap.NewNop(),
		},
		{
			name:   "Without logger",
			config: config,
			logger: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor := NewMonitor(tt.config, tt.logger)

			if monitor == nil {
				t.Fatal("NewMonitor should not return nil")
			}

			if monitor.config.EnableMetrics != tt.config.EnableMetrics {
				t.Errorf("Expected EnableMetrics %v, got %v", tt.config.EnableMetrics, monitor.config.EnableMetrics)
			}

			if monitor.config.MetricsInterval != tt.config.MetricsInterval {
				t.Errorf("Expected MetricsInterval %v, got %v", tt.config.MetricsInterval, monitor.config.MetricsInterval)
			}

			if monitor.metrics == nil {
				t.Error("Monitor metrics should be initialized")
			}

			if monitor.collectors == nil {
				t.Error("Monitor collectors should be initialized")
			}

			if monitor.stopCh == nil {
				t.Error("Monitor stopCh should be initialized")
			}
		})
	}
}

func TestMonitor_AddCollector(t *testing.T) {
	monitor := NewMonitor(MonitoringConfig{}, zap.NewNop())

	collector1 := &MockCollector{name: "collector1"}
	collector2 := &MockCollector{name: "collector2"}

	monitor.AddCollector(collector1)
	monitor.AddCollector(collector2)

	if len(monitor.collectors) != 2 {
		t.Errorf("Expected 2 collectors, got %d", len(monitor.collectors))
	}

	if monitor.collectors[0].Name() != "collector1" {
		t.Errorf("Expected first collector name 'collector1', got '%s'", monitor.collectors[0].Name())
	}

	if monitor.collectors[1].Name() != "collector2" {
		t.Errorf("Expected second collector name 'collector2', got '%s'", monitor.collectors[1].Name())
	}
}

func TestMonitor_StartStop(t *testing.T) {
	config := MonitoringConfig{
		EnableMetrics:   true,
		MetricsInterval: 50 * time.Millisecond,
		EnableLogging:   false,
	}

	monitor := NewMonitor(config, zap.NewNop())
	ctx := context.Background()

	// 测试启动
	err := monitor.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start monitor: %v", err)
	}

	// 验证启动状态
	if monitor.started != 1 {
		t.Error("Monitor should be marked as started")
	}

	if monitor.ticker == nil {
		t.Error("Monitor ticker should be initialized")
	}

	// 测试重复启动
	err = monitor.Start(ctx)
	if err != nil {
		t.Error("Repeated start should not return error")
	}

	// 等待一点时间确保监控器运行
	time.Sleep(10 * time.Millisecond)

	// 测试停止
	err = monitor.Stop()
	if err != nil {
		t.Fatalf("Failed to stop monitor: %v", err)
	}

	// 验证停止状态
	if monitor.started != 0 {
		t.Error("Monitor should be marked as stopped")
	}

	// 测试重复停止
	err = monitor.Stop()
	if err != nil {
		t.Error("Repeated stop should not return error")
	}
}

func TestMonitor_StartWithDisabledMetrics(t *testing.T) {
	config := MonitoringConfig{
		EnableMetrics:   false,
		MetricsInterval: 50 * time.Millisecond,
	}

	monitor := NewMonitor(config, zap.NewNop())
	ctx := context.Background()

	err := monitor.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start monitor with disabled metrics: %v", err)
	}

	// 验证ticker没有被创建
	if monitor.ticker != nil {
		t.Error("Ticker should not be created when metrics are disabled")
	}

	monitor.Stop()
}

func TestMonitor_RecordRequest(t *testing.T) {
	config := MonitoringConfig{
		EnableMetrics: true,
		SlowThreshold: 100 * time.Millisecond,
		EnableLogging: false,
	}

	monitor := NewMonitor(config, zap.NewNop())

	tests := []struct {
		name      string
		operation string
		duration  time.Duration
		err       error
		checkFunc func(*testing.T, *Monitor)
	}{
		{
			name:      "Successful GET operation",
			operation: "get",
			duration:  50 * time.Millisecond,
			err:       nil,
			checkFunc: func(t *testing.T, m *Monitor) {
				metrics := m.GetMetrics()
				if metrics.Hits != 1 {
					t.Errorf("Expected 1 hit, got %d", metrics.Hits)
				}
				if metrics.TotalRequests != 1 {
					t.Errorf("Expected 1 total request, got %d", metrics.TotalRequests)
				}
			},
		},
		{
			name:      "GET operation with key not found",
			operation: "get",
			duration:  30 * time.Millisecond,
			err:       ErrKeyNotFound,
			checkFunc: func(t *testing.T, m *Monitor) {
				metrics := m.GetMetrics()
				if metrics.Misses != 1 {
					t.Errorf("Expected 1 miss, got %d", metrics.Misses)
				}
			},
		},
		{
			name:      "GET operation with other error",
			operation: "get",
			duration:  40 * time.Millisecond,
			err:       errors.New("other error"),
			checkFunc: func(t *testing.T, m *Monitor) {
				metrics := m.GetMetrics()
				if metrics.Errors == 0 {
					t.Error("Expected error count > 0")
				}
			},
		},
		{
			name:      "Successful SET operation",
			operation: "set",
			duration:  60 * time.Millisecond,
			err:       nil,
			checkFunc: func(t *testing.T, m *Monitor) {
				metrics := m.GetMetrics()
				if metrics.Sets != 1 {
					t.Errorf("Expected 1 set, got %d", metrics.Sets)
				}
			},
		},
		{
			name:      "Failed SET operation",
			operation: "set",
			duration:  70 * time.Millisecond,
			err:       errors.New("set failed"),
			checkFunc: func(t *testing.T, m *Monitor) {
				metrics := m.GetMetrics()
				if metrics.Errors == 0 {
					t.Error("Expected error count > 0")
				}
			},
		},
		{
			name:      "Successful DELETE operation",
			operation: "delete",
			duration:  45 * time.Millisecond,
			err:       nil,
			checkFunc: func(t *testing.T, m *Monitor) {
				metrics := m.GetMetrics()
				if metrics.Deletes != 1 {
					t.Errorf("Expected 1 delete, got %d", metrics.Deletes)
				}
			},
		},
		{
			name:      "Slow operation",
			operation: "get",
			duration:  150 * time.Millisecond, // 超过100ms阈值
			err:       nil,
			checkFunc: func(t *testing.T, m *Monitor) {
				metrics := m.GetMetrics()
				if metrics.SlowQueries != 1 {
					t.Errorf("Expected 1 slow query, got %d", metrics.SlowQueries)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 重置监控器
			monitor.Reset()

			monitor.RecordRequest(tt.operation, tt.duration, tt.err)

			if tt.checkFunc != nil {
				tt.checkFunc(t, monitor)
			}
		})
	}
}

func TestMonitor_RecordRequestWithDisabledMetrics(t *testing.T) {
	config := MonitoringConfig{
		EnableMetrics: false,
	}

	monitor := NewMonitor(config, zap.NewNop())

	// 记录请求
	monitor.RecordRequest("get", 50*time.Millisecond, nil)

	// 验证指标没有被更新
	metrics := monitor.GetMetrics()
	if metrics.TotalRequests != 0 {
		t.Error("Metrics should not be updated when disabled")
	}
}

func TestMonitor_GetMetrics(t *testing.T) {
	config := MonitoringConfig{
		EnableMetrics: true,
		SlowThreshold: 100 * time.Millisecond,
	}

	monitor := NewMonitor(config, zap.NewNop())

	// 记录一些操作
	monitor.RecordRequest("get", 50*time.Millisecond, nil)                 // hit
	monitor.RecordRequest("get", 40*time.Millisecond, ErrKeyNotFound)      // miss
	monitor.RecordRequest("set", 30*time.Millisecond, nil)                 // set
	monitor.RecordRequest("delete", 20*time.Millisecond, nil)              // delete
	monitor.RecordRequest("get", 150*time.Millisecond, nil)                // slow hit
	monitor.RecordRequest("get", 60*time.Millisecond, errors.New("error")) // error

	metrics := monitor.GetMetrics()

	// 验证基础计数
	if metrics.TotalRequests != 6 {
		t.Errorf("Expected 6 total requests, got %d", metrics.TotalRequests)
	}

	if metrics.Hits != 2 {
		t.Errorf("Expected 2 hits, got %d", metrics.Hits)
	}

	if metrics.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", metrics.Misses)
	}

	if metrics.Sets != 1 {
		t.Errorf("Expected 1 set, got %d", metrics.Sets)
	}

	if metrics.Deletes != 1 {
		t.Errorf("Expected 1 delete, got %d", metrics.Deletes)
	}

	if metrics.SlowQueries != 1 {
		t.Errorf("Expected 1 slow query, got %d", metrics.SlowQueries)
	}

	if metrics.Errors == 0 {
		t.Error("Expected error count > 0")
	}

	// 验证计算的指标
	expectedHitRate := float64(2) / float64(6)
	if metrics.HitRate != expectedHitRate {
		t.Errorf("Expected hit rate %f, got %f", expectedHitRate, metrics.HitRate)
	}

	if metrics.ErrorRate == 0 {
		t.Error("Expected error rate > 0")
	}

	// 验证响应时间
	if metrics.MaxResponseTime == 0 {
		t.Error("Expected max response time > 0")
	}

	if metrics.MinResponseTime == 0 {
		t.Error("Expected min response time > 0")
	}

	if metrics.AvgResponseTime == 0 {
		t.Error("Expected avg response time > 0")
	}

	// 验证时间戳
	if metrics.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
}

func TestMonitor_Reset(t *testing.T) {
	config := MonitoringConfig{
		EnableMetrics: true,
	}

	monitor := NewMonitor(config, zap.NewNop())

	// 记录一些操作
	monitor.RecordRequest("get", 50*time.Millisecond, nil)
	monitor.RecordRequest("set", 40*time.Millisecond, nil)

	// 验证有数据
	metrics := monitor.GetMetrics()
	if metrics.TotalRequests == 0 {
		t.Error("Expected some requests before reset")
	}

	// 重置
	monitor.Reset()

	// 验证重置后数据清零
	metrics = monitor.GetMetrics()
	if metrics.TotalRequests != 0 {
		t.Errorf("Expected 0 total requests after reset, got %d", metrics.TotalRequests)
	}

	if metrics.Hits != 0 {
		t.Errorf("Expected 0 hits after reset, got %d", metrics.Hits)
	}

	if metrics.Sets != 0 {
		t.Errorf("Expected 0 sets after reset, got %d", metrics.Sets)
	}

	if metrics.AvgResponseTime != 0 {
		t.Errorf("Expected 0 avg response time after reset, got %v", metrics.AvgResponseTime)
	}

	if metrics.MaxResponseTime != 0 {
		t.Errorf("Expected 0 max response time after reset, got %v", metrics.MaxResponseTime)
	}

	if metrics.MinResponseTime != 0 {
		t.Errorf("Expected 0 min response time after reset, got %v", metrics.MinResponseTime)
	}
}

func TestMonitor_CollectWithCollectors(t *testing.T) {
	config := MonitoringConfig{
		EnableMetrics:   true,
		MetricsInterval: 10 * time.Millisecond,
		EnableLogging:   false,
	}

	monitor := NewMonitor(config, zap.NewNop())

	// 添加成功的收集器
	successCollector := &MockCollector{
		name: "success",
		metrics: &Metrics{
			MemoryUsage:     1024,
			ConnectionCount: 5,
			CPUUsage:        0.5,
		},
	}

	// 添加失败的收集器
	failCollector := &MockCollector{
		name: "fail",
		err:  errors.New("collection failed"),
	}

	monitor.AddCollector(successCollector)
	monitor.AddCollector(failCollector)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// 启动监控
	err := monitor.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start monitor: %v", err)
	}

	// 等待一些收集周期
	time.Sleep(50 * time.Millisecond)

	monitor.Stop()

	// 验证成功收集器的数据被合并
	metrics := monitor.GetMetrics()
	if metrics.MemoryUsage == 0 {
		t.Error("Expected memory usage to be updated from collector")
	}

	if metrics.ConnectionCount == 0 {
		t.Error("Expected connection count to be updated from collector")
	}
}

func TestMonitor_ConcurrentAccess(t *testing.T) {
	config := MonitoringConfig{
		EnableMetrics: true,
		SlowThreshold: 100 * time.Millisecond,
	}

	monitor := NewMonitor(config, zap.NewNop())

	// 并发记录请求
	const goroutines = 10
	const operations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < operations; i++ {
				monitor.RecordRequest("get", 50*time.Millisecond, nil)
			}
		}(g)
	}

	// 并发获取指标
	go func() {
		for i := 0; i < 100; i++ {
			monitor.GetMetrics()
			time.Sleep(time.Millisecond)
		}
	}()

	// 并发重置
	go func() {
		for i := 0; i < 10; i++ {
			monitor.Reset()
			time.Sleep(10 * time.Millisecond)
		}
	}()

	wg.Wait()

	// 验证没有panic和数据竞争
	metrics := monitor.GetMetrics()
	if metrics == nil {
		t.Error("GetMetrics should not return nil")
	}
}

func TestCacheCollector(t *testing.T) {
	stats := CacheStats{
		Hits:          100,
		Misses:        20,
		Sets:          80,
		Deletes:       10,
		Errors:        5,
		HitRate:       0.83,
		TotalRequests: 120,
	}

	mockCache := &MockCache{stats: stats}
	collector := NewCacheCollector("test_cache", mockCache)

	// 测试名称
	if collector.Name() != "test_cache" {
		t.Errorf("Expected collector name 'test_cache', got '%s'", collector.Name())
	}

	// 测试收集
	metrics, err := collector.Collect()
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	if metrics == nil {
		t.Fatal("Collected metrics should not be nil")
	}

	// 验证指标转换
	if metrics.Hits != stats.Hits {
		t.Errorf("Expected hits %d, got %d", stats.Hits, metrics.Hits)
	}

	if metrics.Misses != stats.Misses {
		t.Errorf("Expected misses %d, got %d", stats.Misses, metrics.Misses)
	}

	if metrics.Sets != stats.Sets {
		t.Errorf("Expected sets %d, got %d", stats.Sets, metrics.Sets)
	}

	if metrics.HitRate != stats.HitRate {
		t.Errorf("Expected hit rate %f, got %f", stats.HitRate, metrics.HitRate)
	}

	if metrics.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
}

func TestMonitor_UpdateResponseTime(t *testing.T) {
	config := MonitoringConfig{
		EnableMetrics: true,
	}

	monitor := NewMonitor(config, zap.NewNop())

	// 测试第一个响应时间
	monitor.RecordRequest("get", 100*time.Millisecond, nil)
	metrics := monitor.GetMetrics()

	if metrics.MinResponseTime != 100*time.Millisecond {
		t.Errorf("Expected min response time 100ms, got %v", metrics.MinResponseTime)
	}

	if metrics.MaxResponseTime != 100*time.Millisecond {
		t.Errorf("Expected max response time 100ms, got %v", metrics.MaxResponseTime)
	}

	if metrics.AvgResponseTime != 100*time.Millisecond {
		t.Errorf("Expected avg response time 100ms, got %v", metrics.AvgResponseTime)
	}

	// 测试更短的响应时间
	monitor.RecordRequest("get", 50*time.Millisecond, nil)
	metrics = monitor.GetMetrics()

	if metrics.MinResponseTime != 50*time.Millisecond {
		t.Errorf("Expected min response time 50ms, got %v", metrics.MinResponseTime)
	}

	if metrics.MaxResponseTime != 100*time.Millisecond {
		t.Errorf("Expected max response time 100ms, got %v", metrics.MaxResponseTime)
	}

	// 测试更长的响应时间
	monitor.RecordRequest("get", 200*time.Millisecond, nil)
	metrics = monitor.GetMetrics()

	if metrics.MinResponseTime != 50*time.Millisecond {
		t.Errorf("Expected min response time 50ms, got %v", metrics.MinResponseTime)
	}

	if metrics.MaxResponseTime != 200*time.Millisecond {
		t.Errorf("Expected max response time 200ms, got %v", metrics.MaxResponseTime)
	}

	// 平均值应该在合理范围内
	if metrics.AvgResponseTime < 100*time.Millisecond || metrics.AvgResponseTime > 120*time.Millisecond {
		t.Errorf("Average response time %v is out of expected range", metrics.AvgResponseTime)
	}
}

// 性能测试
func BenchmarkMonitor_RecordRequest(b *testing.B) {
	config := MonitoringConfig{
		EnableMetrics: true,
		SlowThreshold: 100 * time.Millisecond,
		EnableLogging: false,
	}

	monitor := NewMonitor(config, zap.NewNop())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		monitor.RecordRequest("get", 50*time.Millisecond, nil)
	}
}

func BenchmarkMonitor_GetMetrics(b *testing.B) {
	config := MonitoringConfig{
		EnableMetrics: true,
	}

	monitor := NewMonitor(config, zap.NewNop())

	// 添加一些数据
	for i := 0; i < 1000; i++ {
		monitor.RecordRequest("get", 50*time.Millisecond, nil)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		monitor.GetMetrics()
	}
}

func BenchmarkMonitor_ConcurrentRecordRequest(b *testing.B) {
	config := MonitoringConfig{
		EnableMetrics: true,
		EnableLogging: false,
	}

	monitor := NewMonitor(config, zap.NewNop())

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			monitor.RecordRequest("get", 50*time.Millisecond, nil)
		}
	})
}
