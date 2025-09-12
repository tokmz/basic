package database

import (
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

func TestNewClient_InvalidConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *DatabaseConfig
		wantErr bool
	}{
		{
			name:    "nil config uses default",
			config:  nil,
			wantErr: true, // Will fail due to missing DSN
		},
		{
			name: "invalid config",
			config: &DatabaseConfig{
				Master: MasterConfig{
					Driver: "", // Missing driver
				},
			},
			wantErr: true,
		},
		{
			name: "unsupported driver",
			config: &DatabaseConfig{
				Master: MasterConfig{
					Driver: "unsupported",
					DSN:    "some-dsn",
				},
				Pool: PoolConfig{
					MaxOpenConns: 10,
					MaxIdleConns: 5,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got nil")
					if client != nil {
						client.Close()
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if client != nil {
					client.Close()
				}
			}
		})
	}
}

func TestClient_ConfigValidation(t *testing.T) {
	// Test with nil config (should use default)
	_, err := NewClient(nil)
	if err == nil {
		t.Error("Expected error with nil config due to missing DSN")
	}

	// Test config validation is called
	invalidConfig := &DatabaseConfig{
		Master: MasterConfig{
			Driver: "", // Invalid
		},
	}
	_, err = NewClient(invalidConfig)
	if err == nil {
		t.Error("Expected error with invalid config")
	}
}

func TestClient_OpenDatabase(t *testing.T) {
	client := &Client{}

	tests := []struct {
		name    string
		driver  string
		dsn     string
		wantErr bool
	}{
		{
			name:    "mysql driver",
			driver:  "mysql",
			dsn:     "user:pass@tcp(localhost:3306)/db",
			wantErr: false,
		},
		{
			name:    "postgres driver",
			driver:  "postgres",
			dsn:     "host=localhost user=test dbname=test sslmode=disable",
			wantErr: false,
		},
		{
			name:    "unsupported driver",
			driver:  "unsupported",
			dsn:     "some-dsn",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dialector, err := client.openDatabase(tt.driver, tt.dsn)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if dialector == nil {
					t.Error("Expected dialector but got nil")
				}
			}
		})
	}
}

func TestClient_IsClosed(t *testing.T) {
	client := &Client{closed: false}
	
	if client.IsClosed() {
		t.Error("Expected client to not be closed initially")
	}

	client.closed = true
	if !client.IsClosed() {
		t.Error("Expected client to be closed after setting closed=true")
	}
}

func TestClient_GetHealthStatus(t *testing.T) {
	client := &Client{
		healthStatus: &HealthStatus{},
	}

	// Initialize with some test data
	client.healthStatus.Master.Healthy = true
	client.healthStatus.Master.FailureCount = 0
	client.healthStatus.Slaves = []struct {
		Index         int       `json:"index"`
		Healthy       bool      `json:"healthy"`
		LastCheck     time.Time `json:"last_check"`
		FailureCount  int       `json:"failure_count"`
		LastError     string    `json:"last_error"`
	}{
		{Index: 0, Healthy: true, FailureCount: 0},
		{Index: 1, Healthy: false, FailureCount: 2},
	}

	status := client.GetHealthStatus()
	
	if status == nil {
		t.Fatal("Expected health status but got nil")
	}
	if !status.Master.Healthy {
		t.Error("Expected master to be healthy")
	}
	if len(status.Slaves) != 2 {
		t.Errorf("Expected 2 slaves, got %d", len(status.Slaves))
	}
	if status.Slaves[0].Index != 0 {
		t.Errorf("Expected slave 0 index to be 0, got %d", status.Slaves[0].Index)
	}
	if !status.Slaves[0].Healthy {
		t.Error("Expected slave 0 to be healthy")
	}
	if status.Slaves[1].Healthy {
		t.Error("Expected slave 1 to be unhealthy")
	}
}

func TestClient_IsHealthy(t *testing.T) {
	client := &Client{
		healthStatus: &HealthStatus{},
	}

	// Test all healthy
	client.healthStatus.Master.Healthy = true
	client.healthStatus.Slaves = []struct {
		Index         int       `json:"index"`
		Healthy       bool      `json:"healthy"`
		LastCheck     time.Time `json:"last_check"`
		FailureCount  int       `json:"failure_count"`
		LastError     string    `json:"last_error"`
	}{
		{Healthy: true},
		{Healthy: true},
	}

	if !client.IsHealthy() {
		t.Error("Expected client to be healthy when all connections are healthy")
	}

	// Test master unhealthy
	client.healthStatus.Master.Healthy = false
	if client.IsHealthy() {
		t.Error("Expected client to be unhealthy when master is unhealthy")
	}

	// Test slave unhealthy
	client.healthStatus.Master.Healthy = true
	client.healthStatus.Slaves[0].Healthy = false
	if client.IsHealthy() {
		t.Error("Expected client to be unhealthy when any slave is unhealthy")
	}
}

func TestClient_IsMasterHealthy(t *testing.T) {
	client := &Client{
		healthStatus: &HealthStatus{},
	}

	client.healthStatus.Master.Healthy = true
	if !client.IsMasterHealthy() {
		t.Error("Expected master to be healthy")
	}

	client.healthStatus.Master.Healthy = false
	if client.IsMasterHealthy() {
		t.Error("Expected master to be unhealthy")
	}
}

func TestClient_GetHealthySlavesCount(t *testing.T) {
	client := &Client{
		healthStatus: &HealthStatus{},
	}

	client.healthStatus.Slaves = []struct {
		Index         int       `json:"index"`
		Healthy       bool      `json:"healthy"`
		LastCheck     time.Time `json:"last_check"`
		FailureCount  int       `json:"failure_count"`
		LastError     string    `json:"last_error"`
	}{
		{Healthy: true},
		{Healthy: false},
		{Healthy: true},
	}

	count := client.GetHealthySlavesCount()
	if count != 2 {
		t.Errorf("Expected 2 healthy slaves, got %d", count)
	}
}

func TestClient_GetTotalSlavesCount(t *testing.T) {
	client := &Client{
		healthStatus: &HealthStatus{},
	}

	client.healthStatus.Slaves = []struct {
		Index         int       `json:"index"`
		Healthy       bool      `json:"healthy"`
		LastCheck     time.Time `json:"last_check"`
		FailureCount  int       `json:"failure_count"`
		LastError     string    `json:"last_error"`
	}{
		{Healthy: true},
		{Healthy: false},
		{Healthy: true},
	}

	count := client.GetTotalSlavesCount()
	if count != 3 {
		t.Errorf("Expected 3 total slaves, got %d", count)
	}
}

func TestClient_SlowQueryMethods(t *testing.T) {
	// Test with nil slow query monitor
	client := &Client{slowQuery: nil}
	
	stats := client.GetSlowQueryStats()
	if stats != nil {
		t.Error("Expected nil slow query stats when monitor is nil")
	}

	// Should not panic
	client.ResetSlowQueryStats()

	// Test with slow query monitor
	logger := zaptest.NewLogger(t)
	config := SlowQueryConfig{
		Enabled:   true,
		Threshold: time.Millisecond * 100,
		Logger:    logger,
	}
	monitor := NewSlowQueryMonitor(config)
	
	// Record a slow query
	monitor.RecordSlowQuery("SELECT * FROM users", time.Millisecond*150, 5, nil)
	
	client.slowQuery = monitor
	stats = client.GetSlowQueryStats()
	if stats == nil {
		t.Fatal("Expected slow query stats but got nil")
	}
	if stats.TotalSlowQueries != 1 {
		t.Errorf("Expected 1 slow query, got %d", stats.TotalSlowQueries)
	}

	// Reset stats
	client.ResetSlowQueryStats()
	stats = client.GetSlowQueryStats()
	if stats.TotalSlowQueries != 0 {
		t.Errorf("Expected 0 slow queries after reset, got %d", stats.TotalSlowQueries)
	}
}

func TestClient_SetZapLogger_Closed(t *testing.T) {
	client := &Client{closed: true}
	logger := zaptest.NewLogger(t)

	err := client.SetZapLogger(logger)
	if err == nil {
		t.Error("Expected error when setting zap logger on closed client")
	}
}

func TestClient_Close(t *testing.T) {
	client := &Client{closed: false}

	// First close should work
	err := client.Close()
	if err != nil {
		t.Errorf("Expected no error on first close, got: %v", err)
	}

	if !client.IsClosed() {
		t.Error("Expected client to be closed after Close()")
	}

	// Second close should not error
	err = client.Close()
	if err != nil {
		t.Errorf("Expected no error on second close, got: %v", err)
	}
}