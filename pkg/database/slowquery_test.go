package database

import (
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

func TestNewSlowQueryMonitor(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := SlowQueryConfig{
		Enabled:   true,
		Threshold: time.Millisecond * 100,
		Logger:    logger,
	}

	monitor := NewSlowQueryMonitor(config)

	if monitor == nil {
		t.Fatal("NewSlowQueryMonitor returned nil")
	}
	if monitor.config.Threshold != time.Millisecond*100 {
		t.Errorf("Expected threshold 100ms, got %v", monitor.config.Threshold)
	}
	if monitor.logger != logger {
		t.Error("Logger not set correctly")
	}
	if len(monitor.stats.RecentSlowQueries) != 0 {
		t.Error("Expected empty recent queries slice")
	}
	if len(monitor.stats.QueryTypeStats) != 0 {
		t.Error("Expected empty query type stats map")
	}
}

func TestNewSlowQueryMonitor_NilLogger(t *testing.T) {
	config := SlowQueryConfig{
		Enabled:   true,
		Threshold: time.Millisecond * 100,
		Logger:    nil,
	}

	monitor := NewSlowQueryMonitor(config)

	if monitor == nil {
		t.Fatal("NewSlowQueryMonitor returned nil")
	}
	// Should not panic with nil logger (uses zap.NewNop())
}

func TestSlowQueryMonitor_RecordSlowQuery(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := SlowQueryConfig{
		Enabled:   true,
		Threshold: time.Millisecond * 100,
		Logger:    logger,
	}

	monitor := NewSlowQueryMonitor(config)

	// Test query that meets threshold
	sql := "SELECT * FROM users WHERE complex_condition = true"
	duration := time.Millisecond * 150
	rowsAffected := int64(5)

	monitor.RecordSlowQuery(sql, duration, rowsAffected, nil)

	stats := monitor.GetStats()
	if stats.TotalSlowQueries != 1 {
		t.Errorf("Expected 1 slow query, got %d", stats.TotalSlowQueries)
	}
	if stats.SlowestQuery == nil {
		t.Fatal("Expected slowest query to be set")
	}
	if stats.SlowestQuery.SQL != sql {
		t.Errorf("Expected SQL %s, got %s", sql, stats.SlowestQuery.SQL)
	}
	if stats.SlowestQuery.Duration != duration {
		t.Errorf("Expected duration %v, got %v", duration, stats.SlowestQuery.Duration)
	}
	if stats.SlowestQuery.RowsAffected != rowsAffected {
		t.Errorf("Expected rows affected %d, got %d", rowsAffected, stats.SlowestQuery.RowsAffected)
	}
	if stats.SlowestQuery.QueryType != "SELECT" {
		t.Errorf("Expected query type SELECT, got %s", stats.SlowestQuery.QueryType)
	}
	if len(stats.RecentSlowQueries) != 1 {
		t.Errorf("Expected 1 recent slow query, got %d", len(stats.RecentSlowQueries))
	}

	// Check query type stats
	selectStats, exists := stats.QueryTypeStats["SELECT"]
	if !exists {
		t.Fatal("Expected SELECT query type stats")
	}
	if selectStats.Count != 1 {
		t.Errorf("Expected count 1, got %d", selectStats.Count)
	}
	if selectStats.TotalTime != duration {
		t.Errorf("Expected total time %v, got %v", duration, selectStats.TotalTime)
	}
	if selectStats.AverageTime != duration {
		t.Errorf("Expected average time %v, got %v", duration, selectStats.AverageTime)
	}
	if selectStats.MaxTime != duration {
		t.Errorf("Expected max time %v, got %v", duration, selectStats.MaxTime)
	}
	if selectStats.MinTime != duration {
		t.Errorf("Expected min time %v, got %v", duration, selectStats.MinTime)
	}
}

func TestSlowQueryMonitor_RecordSlowQuery_BelowThreshold(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := SlowQueryConfig{
		Enabled:   true,
		Threshold: time.Millisecond * 100,
		Logger:    logger,
	}

	monitor := NewSlowQueryMonitor(config)

	// Test query that doesn't meet threshold
	monitor.RecordSlowQuery("SELECT 1", time.Millisecond*50, 1, nil)

	stats := monitor.GetStats()
	if stats.TotalSlowQueries != 0 {
		t.Errorf("Expected 0 slow queries, got %d", stats.TotalSlowQueries)
	}
}

func TestSlowQueryMonitor_ExtractQueryType(t *testing.T) {
	monitor := &SlowQueryMonitor{}

	tests := []struct {
		sql      string
		expected string
	}{
		{"SELECT * FROM users", "SELECT"},
		{"select * from users", "SELECT"},
		{"INSERT INTO users VALUES (1, 'test')", "INSERT"},
		{"insert into users values (1, 'test')", "INSERT"},
		{"UPDATE users SET name = 'test'", "UPDATE"},
		{"update users set name = 'test'", "UPDATE"},
		{"DELETE FROM users WHERE id = 1", "DELETE"},
		{"delete from users where id = 1", "DELETE"},
		{"CREATE TABLE test (id INT)", "CREATE"},
		{"create table test (id int)", "CREATE"},
		{"DROP TABLE test", "DROP"},
		{"drop table test", "DROP"},
		{"ALTER TABLE users ADD COLUMN age INT", "ALTER"},
		{"alter table users add column age int", "ALTER"},
		{"EXPLAIN SELECT * FROM users", "OTHER"},
		{"", "UNKNOWN"},
		{"   ", "UNKNOWN"},
		{"INVALID", "OTHER"},
	}

	for _, tt := range tests {
		t.Run(tt.sql, func(t *testing.T) {
			result := monitor.extractQueryType(tt.sql)
			if result != tt.expected {
				t.Errorf("extractQueryType(%q) = %q, want %q", tt.sql, result, tt.expected)
			}
		})
	}
}

func TestSlowQueryMonitor_MultipleQueries(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := SlowQueryConfig{
		Enabled:   true,
		Threshold: time.Millisecond * 100,
		Logger:    logger,
	}

	monitor := NewSlowQueryMonitor(config)

	// Record multiple slow queries
	queries := []struct {
		sql      string
		duration time.Duration
		rows     int64
	}{
		{"SELECT * FROM users", time.Millisecond * 150, 5},
		{"INSERT INTO users VALUES (1, 'test')", time.Millisecond * 120, 1},
		{"SELECT * FROM products", time.Millisecond * 300, 10},
		{"UPDATE users SET name = 'test'", time.Millisecond * 200, 1},
		{"SELECT * FROM orders", time.Millisecond * 180, 8},
	}

	for _, q := range queries {
		monitor.RecordSlowQuery(q.sql, q.duration, q.rows, nil)
	}

	stats := monitor.GetStats()
	if stats.TotalSlowQueries != int64(len(queries)) {
		t.Errorf("Expected %d slow queries, got %d", len(queries), stats.TotalSlowQueries)
	}

	// Check slowest query (should be the products query with 300ms)
	if stats.SlowestQuery.Duration != time.Millisecond*300 {
		t.Errorf("Expected slowest query duration 300ms, got %v", stats.SlowestQuery.Duration)
	}

	// Check query type stats
	if selectStats, exists := stats.QueryTypeStats["SELECT"]; exists {
		if selectStats.Count != 3 {
			t.Errorf("Expected 3 SELECT queries, got %d", selectStats.Count)
		}
		expectedTotalTime := time.Millisecond * (150 + 300 + 180)
		if selectStats.TotalTime != expectedTotalTime {
			t.Errorf("Expected total SELECT time %v, got %v", expectedTotalTime, selectStats.TotalTime)
		}
	} else {
		t.Error("Expected SELECT query type stats")
	}

	if insertStats, exists := stats.QueryTypeStats["INSERT"]; exists {
		if insertStats.Count != 1 {
			t.Errorf("Expected 1 INSERT query, got %d", insertStats.Count)
		}
	} else {
		t.Error("Expected INSERT query type stats")
	}

	if updateStats, exists := stats.QueryTypeStats["UPDATE"]; exists {
		if updateStats.Count != 1 {
			t.Errorf("Expected 1 UPDATE query, got %d", updateStats.Count)
		}
	} else {
		t.Error("Expected UPDATE query type stats")
	}
}

func TestSlowQueryMonitor_RecentQueriesLimit(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := SlowQueryConfig{
		Enabled:   true,
		Threshold: time.Millisecond * 100,
		Logger:    logger,
	}

	monitor := NewSlowQueryMonitor(config)

	// Record 105 slow queries (more than the limit of 100)
	for i := 0; i < 105; i++ {
		sql := "SELECT * FROM test_table"
		duration := time.Millisecond * 150
		monitor.RecordSlowQuery(sql, duration, 1, nil)
	}

	stats := monitor.GetStats()
	if stats.TotalSlowQueries != 105 {
		t.Errorf("Expected total count 105, got %d", stats.TotalSlowQueries)
	}

	// Recent queries should be limited to 100
	if len(stats.RecentSlowQueries) != 100 {
		t.Errorf("Expected 100 recent queries, got %d", len(stats.RecentSlowQueries))
	}
}

func TestSlowQueryMonitor_ResetStats(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := SlowQueryConfig{
		Enabled:   true,
		Threshold: time.Millisecond * 100,
		Logger:    logger,
	}

	monitor := NewSlowQueryMonitor(config)

	// Record some slow queries
	monitor.RecordSlowQuery("SELECT * FROM users", time.Millisecond*150, 5, nil)
	monitor.RecordSlowQuery("INSERT INTO users VALUES (1, 'test')", time.Millisecond*120, 1, nil)

	stats := monitor.GetStats()
	if stats.TotalSlowQueries != 2 {
		t.Errorf("Expected 2 slow queries before reset, got %d", stats.TotalSlowQueries)
	}

	// Reset stats
	monitor.ResetStats()

	stats = monitor.GetStats()
	if stats.TotalSlowQueries != 0 {
		t.Errorf("Expected 0 slow queries after reset, got %d", stats.TotalSlowQueries)
	}
	if stats.SlowestQuery != nil {
		t.Error("Expected slowest query to be nil after reset")
	}
	if len(stats.RecentSlowQueries) != 0 {
		t.Errorf("Expected 0 recent queries after reset, got %d", len(stats.RecentSlowQueries))
	}
	if len(stats.QueryTypeStats) != 0 {
		t.Errorf("Expected 0 query type stats after reset, got %d", len(stats.QueryTypeStats))
	}
}

func TestSlowQueryMonitor_QueryTypeStatsUpdate(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := SlowQueryConfig{
		Enabled:   true,
		Threshold: time.Millisecond * 100,
		Logger:    logger,
	}

	monitor := NewSlowQueryMonitor(config)

	// Record multiple SELECT queries with different durations
	durations := []time.Duration{
		time.Millisecond * 150,
		time.Millisecond * 200,
		time.Millisecond * 120,
	}

	for _, duration := range durations {
		monitor.RecordSlowQuery("SELECT * FROM test", duration, 1, nil)
	}

	stats := monitor.GetStats()
	selectStats := stats.QueryTypeStats["SELECT"]

	if selectStats.Count != 3 {
		t.Errorf("Expected count 3, got %d", selectStats.Count)
	}

	expectedTotalTime := time.Millisecond * (150 + 200 + 120)
	if selectStats.TotalTime != expectedTotalTime {
		t.Errorf("Expected total time %v, got %v", expectedTotalTime, selectStats.TotalTime)
	}

	expectedAverageTime := expectedTotalTime / 3
	if selectStats.AverageTime != expectedAverageTime {
		t.Errorf("Expected average time %v, got %v", expectedAverageTime, selectStats.AverageTime)
	}

	if selectStats.MaxTime != time.Millisecond*200 {
		t.Errorf("Expected max time 200ms, got %v", selectStats.MaxTime)
	}

	if selectStats.MinTime != time.Millisecond*120 {
		t.Errorf("Expected min time 120ms, got %v", selectStats.MinTime)
	}
}