package database

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

// TestConcurrentHealthStatus 测试GetHealthStatus的竞态条件
func TestConcurrentHealthStatus(t *testing.T) {
	client := &Client{
		healthStatus: &HealthStatus{},
		mu:           sync.RWMutex{},
	}

	// 初始化健康状态
	client.healthStatus.Master.Healthy = true
	client.healthStatus.Slaves = []struct {
		Index        int       `json:"index"`
		Healthy      bool      `json:"healthy"`
		LastCheck    time.Time `json:"last_check"`
		FailureCount int       `json:"failure_count"`
		LastError    string    `json:"last_error"`
	}{
		{Index: 0, Healthy: true, LastCheck: time.Now()},
		{Index: 1, Healthy: false, LastCheck: time.Now()},
	}

	const numGoroutines = 100
	const numOperations = 1000

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // readers and writers

	// 启动读取器
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				status := client.GetHealthStatus()
				if status == nil {
					t.Errorf("GetHealthStatus returned nil")
					return
				}
				// 验证返回的数据是一致的
				if len(status.Slaves) != 2 {
					t.Errorf("Expected 2 slaves, got %d", len(status.Slaves))
					return
				}
			}
		}()
	}

	// 启动写入器（模拟健康检查更新）
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				client.mu.Lock()
				client.healthStatus.Master.LastCheck = time.Now()
				client.healthStatus.Master.FailureCount = j % 5
				for k := range client.healthStatus.Slaves {
					client.healthStatus.Slaves[k].LastCheck = time.Now()
					client.healthStatus.Slaves[k].FailureCount = (j + k) % 3
					client.healthStatus.Slaves[k].Healthy = (j+k)%2 == 0
				}
				client.mu.Unlock()
				runtime.Gosched() // 让出给其他协程
			}
		}(i)
	}

	wg.Wait()
}

// TestConcurrentSlowQueryMonitor 测试SlowQueryMonitor的竞态条件
func TestConcurrentSlowQueryMonitor(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := SlowQueryConfig{
		Enabled:   true,
		Threshold: time.Millisecond * 10,
		Logger:    logger,
	}

	monitor := NewSlowQueryMonitor(config)

	const numGoroutines = 50
	const numOperations = 500

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3) // recorders, readers, and resetters

	queries := []string{
		"SELECT * FROM users WHERE id = 1",
		"INSERT INTO users (name) VALUES ('test')",
		"UPDATE users SET name = 'updated' WHERE id = 1",
		"DELETE FROM users WHERE id = 1",
		"SELECT COUNT(*) FROM products",
	}

	// 启动查询记录器
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				query := queries[j%len(queries)]
				duration := time.Duration(j%100+50) * time.Millisecond // 50-150ms
				monitor.RecordSlowQuery(query, duration, int64(j%10), nil)
				runtime.Gosched()
			}
		}(i)
	}

	// 启动统计读取器
	for range numGoroutines {
		go func() {
			defer wg.Done()
			for range numOperations {
				stats := monitor.GetStats()
				// 验证一致性
				if stats.TotalSlowQueries < 0 {
					t.Errorf("Negative total slow queries: %d", stats.TotalSlowQueries)
					return
				}
				// len()永远不会返回负值，所以这个检查是不必要的
				runtime.Gosched()
			}
		}()
	}

	// 启动周期性重置器
	for range numGoroutines {
		go func() {
			defer wg.Done()
			for range numOperations / 10 { // 较少频率地重置
				time.Sleep(time.Millisecond * 5)
				monitor.ResetStats()
				runtime.Gosched()
			}
		}()
	}

	wg.Wait()

	// 最终验证
	finalStats := monitor.GetStats()
	if finalStats.QueryTypeStats == nil {
		t.Error("QueryTypeStats should not be nil")
	}
	if finalStats.RecentSlowQueries == nil {
		t.Error("RecentSlowQueries should not be nil")
	}
}

// TestDeepCopyFunction 测试深拷贝工具函数
func TestDeepCopyFunction(t *testing.T) {
	original := &SlowQueryStats{
		TotalSlowQueries: 10,
		SlowestQuery: &SlowQueryRecord{
			SQL:      "SELECT * FROM test",
			Duration: time.Second,
		},
		RecentSlowQueries: []*SlowQueryRecord{
			{SQL: "SELECT 1", Duration: time.Millisecond * 100},
			{SQL: "SELECT 2", Duration: time.Millisecond * 200},
		},
		QueryTypeStats: map[string]*QueryTypeStats{
			"SELECT": {
				Count:       5,
				TotalTime:   time.Second * 2,
				AverageTime: time.Millisecond * 400,
				MaxTime:     time.Second,
				MinTime:     time.Millisecond * 100,
			},
		},
	}

	var copied SlowQueryStats
	if !safeDeepCopy(original, &copied) {
		t.Fatal("Deep copy failed")
	}

	// 验证拷贝是独立的
	if copied.TotalSlowQueries != original.TotalSlowQueries {
		t.Error("TotalSlowQueries not copied correctly")
	}

	if copied.SlowestQuery.SQL != original.SlowestQuery.SQL {
		t.Error("SlowestQuery not copied correctly")
	}

	if len(copied.RecentSlowQueries) != len(original.RecentSlowQueries) {
		t.Error("RecentSlowQueries not copied correctly")
	}

	// 修改原始数据以确保独立性
	original.TotalSlowQueries = 999
	original.SlowestQuery.SQL = "MODIFIED"

	if copied.TotalSlowQueries == 999 {
		t.Error("Deep copy is not independent - TotalSlowQueries")
	}

	if copied.SlowestQuery.SQL == "MODIFIED" {
		t.Error("Deep copy is not independent - SlowestQuery.SQL")
	}
}

// BenchmarkConcurrentHealthStatus 对健康状态的并发访问进行基准测试
func BenchmarkConcurrentHealthStatus(b *testing.B) {
	client := &Client{
		healthStatus: &HealthStatus{},
		mu:           sync.RWMutex{},
	}

	client.healthStatus.Master.Healthy = true
	client.healthStatus.Slaves = []struct {
		Index        int       `json:"index"`
		Healthy      bool      `json:"healthy"`
		LastCheck    time.Time `json:"last_check"`
		FailureCount int       `json:"failure_count"`
		LastError    string    `json:"last_error"`
	}{
		{Index: 0, Healthy: true},
		{Index: 1, Healthy: false},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			status := client.GetHealthStatus()
			if status == nil {
				b.Error("GetHealthStatus returned nil")
			}
		}
	})
}

// BenchmarkConcurrentSlowQueryStats 对慢查询统计的并发访问进行基准测试
func BenchmarkConcurrentSlowQueryStats(b *testing.B) {
	logger := zaptest.NewLogger(b)
	config := SlowQueryConfig{
		Enabled:   true,
		Threshold: time.Millisecond * 10,
		Logger:    logger,
	}

	monitor := NewSlowQueryMonitor(config)

	// 预先填充一些数据
	for i := 0; i < 100; i++ {
		monitor.RecordSlowQuery("SELECT * FROM test", time.Millisecond*100, 1, nil)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			stats := monitor.GetStats()
			if stats.QueryTypeStats == nil {
				b.Error("QueryTypeStats is nil")
			}
		}
	})
}
