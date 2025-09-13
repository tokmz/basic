package cache

import (
	"reflect"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig should not return nil")
	}

	// 测试基础配置
	if config.Type != TypeMemoryLRU {
		t.Errorf("Expected Type to be TypeMemoryLRU, got %v", config.Type)
	}

	if config.DefaultTTL != time.Hour {
		t.Errorf("Expected DefaultTTL to be 1 hour, got %v", config.DefaultTTL)
	}

	// 测试内存配置
	if config.Memory.MaxSize != 1000 {
		t.Errorf("Expected Memory.MaxSize to be 1000, got %d", config.Memory.MaxSize)
	}

	if config.Memory.MaxMemory != 100*1024*1024 {
		t.Errorf("Expected Memory.MaxMemory to be 100MB, got %d", config.Memory.MaxMemory)
	}

	if config.Memory.EvictPolicy != EvictLRU {
		t.Errorf("Expected Memory.EvictPolicy to be EvictLRU, got %v", config.Memory.EvictPolicy)
	}

	if config.Memory.CleanupInterval != 10*time.Minute {
		t.Errorf("Expected Memory.CleanupInterval to be 10 minutes, got %v", config.Memory.CleanupInterval)
	}

	if !config.Memory.EnableStats {
		t.Error("Expected Memory.EnableStats to be true")
	}

	// 测试Redis配置
	expectedAddrs := []string{"localhost:6379"}
	if !reflect.DeepEqual(config.Redis.Addrs, expectedAddrs) {
		t.Errorf("Expected Redis.Addrs to be %v, got %v", expectedAddrs, config.Redis.Addrs)
	}

	if config.Redis.PoolSize != 10 {
		t.Errorf("Expected Redis.PoolSize to be 10, got %d", config.Redis.PoolSize)
	}

	if config.Redis.MinIdleConns != 5 {
		t.Errorf("Expected Redis.MinIdleConns to be 5, got %d", config.Redis.MinIdleConns)
	}

	if config.Redis.MaxRetries != 3 {
		t.Errorf("Expected Redis.MaxRetries to be 3, got %d", config.Redis.MaxRetries)
	}

	if config.Redis.DialTimeout != 5*time.Second {
		t.Errorf("Expected Redis.DialTimeout to be 5 seconds, got %v", config.Redis.DialTimeout)
	}

	if config.Redis.ReadTimeout != 3*time.Second {
		t.Errorf("Expected Redis.ReadTimeout to be 3 seconds, got %v", config.Redis.ReadTimeout)
	}

	if config.Redis.WriteTimeout != 3*time.Second {
		t.Errorf("Expected Redis.WriteTimeout to be 3 seconds, got %v", config.Redis.WriteTimeout)
	}

	if config.Redis.KeyPrefix != "cache:" {
		t.Errorf("Expected Redis.KeyPrefix to be 'cache:', got '%s'", config.Redis.KeyPrefix)
	}

	// 测试多级缓存配置
	if !config.MultiLevel.EnableL1 {
		t.Error("Expected MultiLevel.EnableL1 to be true")
	}

	if config.MultiLevel.EnableL2 {
		t.Error("Expected MultiLevel.EnableL2 to be false")
	}

	if config.MultiLevel.L1TTL != 30*time.Minute {
		t.Errorf("Expected MultiLevel.L1TTL to be 30 minutes, got %v", config.MultiLevel.L1TTL)
	}

	if config.MultiLevel.L2TTL != 2*time.Hour {
		t.Errorf("Expected MultiLevel.L2TTL to be 2 hours, got %v", config.MultiLevel.L2TTL)
	}

	if config.MultiLevel.SyncStrategy != SyncWriteThrough {
		t.Errorf("Expected MultiLevel.SyncStrategy to be SyncWriteThrough, got %v", config.MultiLevel.SyncStrategy)
	}

	// 测试监控配置
	if !config.Monitoring.EnableMetrics {
		t.Error("Expected Monitoring.EnableMetrics to be true")
	}

	if config.Monitoring.MetricsInterval != 30*time.Second {
		t.Errorf("Expected Monitoring.MetricsInterval to be 30 seconds, got %v", config.Monitoring.MetricsInterval)
	}

	if !config.Monitoring.EnableLogging {
		t.Error("Expected Monitoring.EnableLogging to be true")
	}

	if config.Monitoring.LogLevel != "info" {
		t.Errorf("Expected Monitoring.LogLevel to be 'info', got '%s'", config.Monitoring.LogLevel)
	}

	if config.Monitoring.SlowThreshold != 100*time.Millisecond {
		t.Errorf("Expected Monitoring.SlowThreshold to be 100ms, got %v", config.Monitoring.SlowThreshold)
	}

	// 测试分布式配置
	if config.Distributed.EnableSync {
		t.Error("Expected Distributed.EnableSync to be false")
	}

	if config.Distributed.SyncInterval != 5*time.Second {
		t.Errorf("Expected Distributed.SyncInterval to be 5 seconds, got %v", config.Distributed.SyncInterval)
	}

	if config.Distributed.ConsistencyLevel != ConsistencyEventual {
		t.Errorf("Expected Distributed.ConsistencyLevel to be ConsistencyEventual, got %v", config.Distributed.ConsistencyLevel)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr error
	}{
		{
			name:    "Valid default config",
			config:  DefaultConfig(),
			wantErr: nil,
		},
		{
			name: "Invalid negative DefaultTTL",
			config: &Config{
				DefaultTTL: -time.Hour,
				Memory: MemoryConfig{
					MaxSize:   100,
					MaxMemory: 1024 * 1024,
				},
				Redis: RedisConfig{
					Addrs: []string{"localhost:6379"},
				},
			},
			wantErr: ErrInvalidTTL,
		},
		{
			name: "Invalid zero MaxSize",
			config: &Config{
				DefaultTTL: time.Hour,
				Memory: MemoryConfig{
					MaxSize:   0,
					MaxMemory: 1024 * 1024,
				},
				Redis: RedisConfig{
					Addrs: []string{"localhost:6379"},
				},
			},
			wantErr: ErrInvalidMaxSize,
		},
		{
			name: "Invalid negative MaxSize",
			config: &Config{
				DefaultTTL: time.Hour,
				Memory: MemoryConfig{
					MaxSize:   -1,
					MaxMemory: 1024 * 1024,
				},
				Redis: RedisConfig{
					Addrs: []string{"localhost:6379"},
				},
			},
			wantErr: ErrInvalidMaxSize,
		},
		{
			name: "Invalid zero MaxMemory",
			config: &Config{
				DefaultTTL: time.Hour,
				Memory: MemoryConfig{
					MaxSize:   100,
					MaxMemory: 0,
				},
				Redis: RedisConfig{
					Addrs: []string{"localhost:6379"},
				},
			},
			wantErr: ErrInvalidMaxMemory,
		},
		{
			name: "Invalid negative MaxMemory",
			config: &Config{
				DefaultTTL: time.Hour,
				Memory: MemoryConfig{
					MaxSize:   100,
					MaxMemory: -1,
				},
				Redis: RedisConfig{
					Addrs: []string{"localhost:6379"},
				},
			},
			wantErr: ErrInvalidMaxMemory,
		},
		{
			name: "Empty Redis addresses",
			config: &Config{
				DefaultTTL: time.Hour,
				Memory: MemoryConfig{
					MaxSize:   100,
					MaxMemory: 1024 * 1024,
				},
				Redis: RedisConfig{
					Addrs: []string{},
				},
			},
			wantErr: ErrEmptyRedisAddrs,
		},
		{
			name: "Nil Redis addresses",
			config: &Config{
				DefaultTTL: time.Hour,
				Memory: MemoryConfig{
					MaxSize:   100,
					MaxMemory: 1024 * 1024,
				},
				Redis: RedisConfig{
					Addrs: nil,
				},
			},
			wantErr: ErrEmptyRedisAddrs,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if err != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSyncStrategy_Constants(t *testing.T) {
	tests := []struct {
		strategy SyncStrategy
		expected string
	}{
		{SyncWriteThrough, "write_through"},
		{SyncWriteBack, "write_back"},
		{SyncWriteAround, "write_around"},
	}

	for _, tt := range tests {
		t.Run(string(tt.strategy), func(t *testing.T) {
			if string(tt.strategy) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.strategy))
			}
		})
	}
}

func TestConsistencyLevel_Constants(t *testing.T) {
	tests := []struct {
		level    ConsistencyLevel
		expected string
	}{
		{ConsistencyEventual, "eventual"},
		{ConsistencyStrong, "strong"},
		{ConsistencyWeak, "weak"},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			if string(tt.level) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.level))
			}
		})
	}
}

func TestConfig_DeepCopy(t *testing.T) {
	original := DefaultConfig()

	// 修改原始配置
	original.DefaultTTL = 2 * time.Hour
	original.Memory.MaxSize = 2000
	original.Redis.Addrs = []string{"localhost:6380"}

	// 创建新的默认配置来验证独立性
	newConfig := DefaultConfig()

	// 验证新配置不受原始配置修改的影响
	if newConfig.DefaultTTL != time.Hour {
		t.Errorf("Expected DefaultTTL to be 1 hour, got %v", newConfig.DefaultTTL)
	}

	if newConfig.Memory.MaxSize != 1000 {
		t.Errorf("Expected MaxSize to be 1000, got %d", newConfig.Memory.MaxSize)
	}

	expectedAddrs := []string{"localhost:6379"}
	if !reflect.DeepEqual(newConfig.Redis.Addrs, expectedAddrs) {
		t.Errorf("Expected Redis.Addrs to be %v, got %v", expectedAddrs, newConfig.Redis.Addrs)
	}
}

func TestMemoryConfig_Defaults(t *testing.T) {
	config := DefaultConfig()
	memory := config.Memory

	// 验证内存配置的所有字段
	if memory.MaxSize != 1000 {
		t.Errorf("Expected MaxSize 1000, got %d", memory.MaxSize)
	}

	if memory.MaxMemory != 100*1024*1024 {
		t.Errorf("Expected MaxMemory 100MB, got %d", memory.MaxMemory)
	}

	if memory.EvictPolicy != EvictLRU {
		t.Errorf("Expected EvictLRU policy, got %v", memory.EvictPolicy)
	}

	if memory.CleanupInterval != 10*time.Minute {
		t.Errorf("Expected CleanupInterval 10 minutes, got %v", memory.CleanupInterval)
	}

	if !memory.EnableStats {
		t.Error("Expected EnableStats to be true")
	}
}

func TestRedisConfig_Defaults(t *testing.T) {
	config := DefaultConfig()
	redis := config.Redis

	// 验证Redis配置的所有字段
	expectedAddrs := []string{"localhost:6379"}
	if !reflect.DeepEqual(redis.Addrs, expectedAddrs) {
		t.Errorf("Expected Addrs %v, got %v", expectedAddrs, redis.Addrs)
	}

	if redis.Username != "" {
		t.Errorf("Expected empty Username, got '%s'", redis.Username)
	}

	if redis.Password != "" {
		t.Errorf("Expected empty Password, got '%s'", redis.Password)
	}

	if redis.DB != 0 {
		t.Errorf("Expected DB 0, got %d", redis.DB)
	}

	if redis.PoolSize != 10 {
		t.Errorf("Expected PoolSize 10, got %d", redis.PoolSize)
	}

	if redis.MinIdleConns != 5 {
		t.Errorf("Expected MinIdleConns 5, got %d", redis.MinIdleConns)
	}

	if redis.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries 3, got %d", redis.MaxRetries)
	}

	if redis.DialTimeout != 5*time.Second {
		t.Errorf("Expected DialTimeout 5s, got %v", redis.DialTimeout)
	}

	if redis.ReadTimeout != 3*time.Second {
		t.Errorf("Expected ReadTimeout 3s, got %v", redis.ReadTimeout)
	}

	if redis.WriteTimeout != 3*time.Second {
		t.Errorf("Expected WriteTimeout 3s, got %v", redis.WriteTimeout)
	}

	if redis.ClusterMode {
		t.Error("Expected ClusterMode to be false")
	}

	if redis.KeyPrefix != "cache:" {
		t.Errorf("Expected KeyPrefix 'cache:', got '%s'", redis.KeyPrefix)
	}
}

func TestMultiLevelConfig_Defaults(t *testing.T) {
	config := DefaultConfig()
	multiLevel := config.MultiLevel

	if !multiLevel.EnableL1 {
		t.Error("Expected EnableL1 to be true")
	}

	if multiLevel.EnableL2 {
		t.Error("Expected EnableL2 to be false")
	}

	if multiLevel.L1TTL != 30*time.Minute {
		t.Errorf("Expected L1TTL 30 minutes, got %v", multiLevel.L1TTL)
	}

	if multiLevel.L2TTL != 2*time.Hour {
		t.Errorf("Expected L2TTL 2 hours, got %v", multiLevel.L2TTL)
	}

	if multiLevel.SyncStrategy != SyncWriteThrough {
		t.Errorf("Expected SyncWriteThrough, got %v", multiLevel.SyncStrategy)
	}
}

func TestMonitoringConfig_Defaults(t *testing.T) {
	config := DefaultConfig()
	monitoring := config.Monitoring

	if !monitoring.EnableMetrics {
		t.Error("Expected EnableMetrics to be true")
	}

	if monitoring.MetricsInterval != 30*time.Second {
		t.Errorf("Expected MetricsInterval 30s, got %v", monitoring.MetricsInterval)
	}

	if !monitoring.EnableLogging {
		t.Error("Expected EnableLogging to be true")
	}

	if monitoring.LogLevel != "info" {
		t.Errorf("Expected LogLevel 'info', got '%s'", monitoring.LogLevel)
	}

	if monitoring.SlowThreshold != 100*time.Millisecond {
		t.Errorf("Expected SlowThreshold 100ms, got %v", monitoring.SlowThreshold)
	}
}

func TestDistributedConfig_Defaults(t *testing.T) {
	config := DefaultConfig()
	distributed := config.Distributed

	if distributed.EnableSync {
		t.Error("Expected EnableSync to be false")
	}

	if distributed.SyncInterval != 5*time.Second {
		t.Errorf("Expected SyncInterval 5s, got %v", distributed.SyncInterval)
	}

	if distributed.ConsistencyLevel != ConsistencyEventual {
		t.Errorf("Expected ConsistencyEventual, got %v", distributed.ConsistencyLevel)
	}

	if distributed.NodeID != "" {
		t.Errorf("Expected empty NodeID, got '%s'", distributed.NodeID)
	}

	if distributed.BroadcastAddr != "" {
		t.Errorf("Expected empty BroadcastAddr, got '%s'", distributed.BroadcastAddr)
	}
}

// 性能测试
func BenchmarkDefaultConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		DefaultConfig()
	}
}

func BenchmarkConfig_Validate(b *testing.B) {
	config := DefaultConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config.Validate()
	}
}
