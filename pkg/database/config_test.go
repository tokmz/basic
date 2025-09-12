package database

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	// Test pool defaults
	if config.Pool.MaxOpenConns != 100 {
		t.Errorf("Expected MaxOpenConns=100, got %d", config.Pool.MaxOpenConns)
	}
	if config.Pool.MaxIdleConns != 10 {
		t.Errorf("Expected MaxIdleConns=10, got %d", config.Pool.MaxIdleConns)
	}
	if config.Pool.ConnMaxLifetime != time.Hour {
		t.Errorf("Expected ConnMaxLifetime=1h, got %v", config.Pool.ConnMaxLifetime)
	}
	if config.Pool.ConnMaxIdleTime != time.Minute*30 {
		t.Errorf("Expected ConnMaxIdleTime=30m, got %v", config.Pool.ConnMaxIdleTime)
	}

	// Test logger defaults
	if config.Logger.Level != Info {
		t.Errorf("Expected log level=Info, got %v", config.Logger.Level)
	}
	if !config.Logger.Colorful {
		t.Error("Expected colorful logging to be enabled by default")
	}
	if config.Logger.SlowThreshold != time.Millisecond*200 {
		t.Errorf("Expected slow threshold=200ms, got %v", config.Logger.SlowThreshold)
	}

	// Test slow query defaults
	if !config.SlowQuery.Enabled {
		t.Error("Expected slow query monitoring to be enabled by default")
	}
	if config.SlowQuery.Threshold != time.Millisecond*200 {
		t.Errorf("Expected slow query threshold=200ms, got %v", config.SlowQuery.Threshold)
	}

	// Test health check defaults
	if !config.HealthCheck.Enabled {
		t.Error("Expected health check to be enabled by default")
	}
	if config.HealthCheck.Interval != time.Minute*5 {
		t.Errorf("Expected health check interval=5m, got %v", config.HealthCheck.Interval)
	}
	if config.HealthCheck.Timeout != time.Second*30 {
		t.Errorf("Expected health check timeout=30s, got %v", config.HealthCheck.Timeout)
	}
	if config.HealthCheck.MaxFailures != 3 {
		t.Errorf("Expected max failures=3, got %d", config.HealthCheck.MaxFailures)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *DatabaseConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &DatabaseConfig{
				Master: MasterConfig{
					Driver: "mysql",
					DSN:    "user:pass@tcp(localhost:3306)/db",
				},
				Pool: PoolConfig{
					MaxOpenConns: 10,
					MaxIdleConns: 5,
				},
			},
			wantErr: false,
		},
		{
			name: "missing master driver",
			config: &DatabaseConfig{
				Master: MasterConfig{
					DSN: "user:pass@tcp(localhost:3306)/db",
				},
			},
			wantErr: true,
			errMsg:  "master database driver is required",
		},
		{
			name: "missing master DSN",
			config: &DatabaseConfig{
				Master: MasterConfig{
					Driver: "mysql",
				},
			},
			wantErr: true,
			errMsg:  "master database DSN is required",
		},
		{
			name: "invalid max open connections",
			config: &DatabaseConfig{
				Master: MasterConfig{
					Driver: "mysql",
					DSN:    "user:pass@tcp(localhost:3306)/db",
				},
				Pool: PoolConfig{
					MaxOpenConns: -1,
				},
			},
			wantErr: true,
			errMsg:  "max open connections must be positive",
		},
		{
			name: "invalid max idle connections",
			config: &DatabaseConfig{
				Master: MasterConfig{
					Driver: "mysql",
					DSN:    "user:pass@tcp(localhost:3306)/db",
				},
				Pool: PoolConfig{
					MaxOpenConns: 10,
					MaxIdleConns: -1,
				},
			},
			wantErr: true,
			errMsg:  "max idle connections must be non-negative",
		},
		{
			name: "idle connections exceed open connections",
			config: &DatabaseConfig{
				Master: MasterConfig{
					Driver: "mysql",
					DSN:    "user:pass@tcp(localhost:3306)/db",
				},
				Pool: PoolConfig{
					MaxOpenConns: 5,
					MaxIdleConns: 10,
				},
			},
			wantErr: true,
			errMsg:  "max idle connections cannot exceed max open connections",
		},
		{
			name: "slave missing driver",
			config: &DatabaseConfig{
				Master: MasterConfig{
					Driver: "mysql",
					DSN:    "user:pass@tcp(localhost:3306)/db",
				},
				Slaves: []SlaveConfig{
					{
						DSN: "user:pass@tcp(localhost:3307)/db",
					},
				},
				Pool: PoolConfig{
					MaxOpenConns: 10,
					MaxIdleConns: 5,
				},
			},
			wantErr: true,
			errMsg:  "slave 0: driver is required",
		},
		{
			name: "slave missing DSN",
			config: &DatabaseConfig{
				Master: MasterConfig{
					Driver: "mysql",
					DSN:    "user:pass@tcp(localhost:3306)/db",
				},
				Slaves: []SlaveConfig{
					{
						Driver: "mysql",
					},
				},
				Pool: PoolConfig{
					MaxOpenConns: 10,
					MaxIdleConns: 5,
				},
			},
			wantErr: true,
			errMsg:  "slave 0: DSN is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got nil")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{Silent, "silent"},
		{Error, "error"},
		{Warn, "warn"},
		{Info, "info"},
		{LogLevel(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("LogLevel.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConfigWithZapLogger(t *testing.T) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create zap logger: %v", err)
	}

	config := DefaultConfig()
	config.Logger.ZapLogger = logger
	config.SlowQuery.Logger = logger
	config.HealthCheck.Logger = logger

	if config.Logger.ZapLogger != logger {
		t.Error("Zap logger was not set correctly in logger config")
	}
	if config.SlowQuery.Logger != logger {
		t.Error("Zap logger was not set correctly in slow query config")
	}
	if config.HealthCheck.Logger != logger {
		t.Error("Zap logger was not set correctly in health check config")
	}
}