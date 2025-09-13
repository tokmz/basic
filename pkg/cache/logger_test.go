package cache

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// 创建测试用的logger，输出到缓冲区便于验证
func createTestLogger(config MonitoringConfig) (*Logger, *bytes.Buffer, error) {
	if !config.EnableLogging {
		return &Logger{
			Logger: zap.NewNop(),
			config: config,
		}, nil, nil
	}

	var buffer bytes.Buffer
	level := parseLogLevel(config.LogLevel)

	// 创建encoder配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 创建core
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(&buffer),
		level,
	)

	logger := zap.New(core).Named("cache")

	return &Logger{
		Logger: logger,
		config: config,
	}, &buffer, nil
}

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name    string
		config  MonitoringConfig
		wantNil bool
		wantErr bool
	}{
		{
			name: "Enabled logging with info level",
			config: MonitoringConfig{
				EnableLogging: true,
				LogLevel:      "info",
				SlowThreshold: 100 * time.Millisecond,
			},
			wantNil: false,
			wantErr: false,
		},
		{
			name: "Disabled logging",
			config: MonitoringConfig{
				EnableLogging: false,
				LogLevel:      "info",
				SlowThreshold: 100 * time.Millisecond,
			},
			wantNil: false,
			wantErr: false,
		},
		{
			name: "Debug level logging",
			config: MonitoringConfig{
				EnableLogging: true,
				LogLevel:      "debug",
				SlowThreshold: 50 * time.Millisecond,
			},
			wantNil: false,
			wantErr: false,
		},
		{
			name: "Error level logging",
			config: MonitoringConfig{
				EnableLogging: true,
				LogLevel:      "error",
				SlowThreshold: 200 * time.Millisecond,
			},
			wantNil: false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLogger(tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewLogger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if (logger == nil) != tt.wantNil {
				t.Errorf("NewLogger() = %v, wantNil %v", logger == nil, tt.wantNil)
			}

			if logger != nil && logger.config.EnableLogging != tt.config.EnableLogging {
				t.Errorf("Logger config mismatch: expected EnableLogging %v, got %v",
					tt.config.EnableLogging, logger.config.EnableLogging)
			}
		})
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		level    string
		expected zapcore.Level
	}{
		{"debug", zapcore.DebugLevel},
		{"info", zapcore.InfoLevel},
		{"warn", zapcore.WarnLevel},
		{"error", zapcore.ErrorLevel},
		{"fatal", zapcore.FatalLevel},
		{"unknown", zapcore.InfoLevel}, // 默认级别
		{"", zapcore.InfoLevel},        // 空字符串默认级别
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			result := parseLogLevel(tt.level)
			if result != tt.expected {
				t.Errorf("parseLogLevel(%s) = %v, expected %v", tt.level, result, tt.expected)
			}
		})
	}
}

func TestLogger_LogOperation(t *testing.T) {
	tests := []struct {
		name      string
		config    MonitoringConfig
		operation string
		key       string
		duration  time.Duration
		err       error
		expectLog bool
		logLevel  string
	}{
		{
			name: "Successful operation under threshold",
			config: MonitoringConfig{
				EnableLogging: true,
				LogLevel:      "debug",
				SlowThreshold: 100 * time.Millisecond,
			},
			operation: "GET",
			key:       "test_key",
			duration:  50 * time.Millisecond,
			err:       nil,
			expectLog: true,
			logLevel:  "debug",
		},
		{
			name: "Slow operation",
			config: MonitoringConfig{
				EnableLogging: true,
				LogLevel:      "info",
				SlowThreshold: 100 * time.Millisecond,
			},
			operation: "SET",
			key:       "slow_key",
			duration:  200 * time.Millisecond,
			err:       nil,
			expectLog: true,
			logLevel:  "warn",
		},
		{
			name: "Failed operation",
			config: MonitoringConfig{
				EnableLogging: true,
				LogLevel:      "info",
				SlowThreshold: 100 * time.Millisecond,
			},
			operation: "DELETE",
			key:       "error_key",
			duration:  50 * time.Millisecond,
			err:       errors.New("operation failed"),
			expectLog: true,
			logLevel:  "error",
		},
		{
			name: "Logging disabled",
			config: MonitoringConfig{
				EnableLogging: false,
				LogLevel:      "info",
				SlowThreshold: 100 * time.Millisecond,
			},
			operation: "GET",
			key:       "test_key",
			duration:  50 * time.Millisecond,
			err:       nil,
			expectLog: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, buffer, err := createTestLogger(tt.config)
			if err != nil {
				t.Fatalf("Failed to create test logger: %v", err)
			}

			logger.LogOperation(tt.operation, tt.key, tt.duration, tt.err)

			if !tt.expectLog {
				if buffer != nil && buffer.Len() > 0 {
					t.Error("Expected no log output when logging is disabled")
				}
				return
			}

			if buffer == nil || buffer.Len() == 0 {
				t.Error("Expected log output but got none")
				return
			}

			logOutput := buffer.String()

			// 验证日志包含期望的字段
			if !strings.Contains(logOutput, tt.operation) {
				t.Errorf("Log should contain operation '%s'", tt.operation)
			}

			if !strings.Contains(logOutput, tt.key) {
				t.Errorf("Log should contain key '%s'", tt.key)
			}

			if !strings.Contains(logOutput, tt.logLevel) {
				t.Errorf("Log should contain level '%s'", tt.logLevel)
			}
		})
	}
}

func TestLogger_LogError(t *testing.T) {
	config := MonitoringConfig{
		EnableLogging: true,
		LogLevel:      "info",
		SlowThreshold: 100 * time.Millisecond,
	}

	logger, buffer, err := createTestLogger(config)
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}

	testErr := errors.New("test error")
	logger.LogError("Test error message", testErr, zap.String("extra", "field"))

	if buffer.Len() == 0 {
		t.Error("Expected error log output")
		return
	}

	logOutput := buffer.String()
	if !strings.Contains(logOutput, "error") {
		t.Error("Log should contain error level")
	}

	if !strings.Contains(logOutput, "Test error message") {
		t.Error("Log should contain error message")
	}

	if !strings.Contains(logOutput, "test error") {
		t.Error("Log should contain error details")
	}
}

func TestLogger_LogInfo(t *testing.T) {
	config := MonitoringConfig{
		EnableLogging: true,
		LogLevel:      "info",
		SlowThreshold: 100 * time.Millisecond,
	}

	logger, buffer, err := createTestLogger(config)
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}

	logger.LogInfo("Test info message", zap.String("key", "value"))

	if buffer.Len() == 0 {
		t.Error("Expected info log output")
		return
	}

	logOutput := buffer.String()
	if !strings.Contains(logOutput, "info") {
		t.Error("Log should contain info level")
	}

	if !strings.Contains(logOutput, "Test info message") {
		t.Error("Log should contain info message")
	}
}

func TestLogger_LogStats(t *testing.T) {
	config := MonitoringConfig{
		EnableLogging: true,
		LogLevel:      "info",
		SlowThreshold: 100 * time.Millisecond,
	}

	logger, buffer, err := createTestLogger(config)
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}

	stats := CacheStats{
		Hits:          100,
		Misses:        20,
		Sets:          50,
		Deletes:       10,
		Errors:        5,
		HitRate:       0.83,
		TotalRequests: 120,
		Size:          75,
	}

	logger.LogStats(stats)

	if buffer.Len() == 0 {
		t.Error("Expected stats log output")
		return
	}

	// 解析JSON日志验证字段
	var logData map[string]interface{}
	if err := json.Unmarshal(buffer.Bytes(), &logData); err != nil {
		t.Fatalf("Failed to parse log JSON: %v", err)
	}

	if logData["hits"] != float64(100) {
		t.Errorf("Expected hits=100, got %v", logData["hits"])
	}

	if logData["misses"] != float64(20) {
		t.Errorf("Expected misses=20, got %v", logData["misses"])
	}

	if logData["hit_rate"] != 0.83 {
		t.Errorf("Expected hit_rate=0.83, got %v", logData["hit_rate"])
	}
}

func TestLogger_LogCacheEvent(t *testing.T) {
	config := MonitoringConfig{
		EnableLogging: true,
		LogLevel:      "debug",
		SlowThreshold: 100 * time.Millisecond,
	}

	logger, buffer, err := createTestLogger(config)
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}

	tests := []struct {
		event   string
		level   string
		message string
	}{
		{"cache_start", "info", "Cache started"},
		{"cache_error", "error", "Cache error occurred"},
		{"cache_warning", "warn", "Cache warning"},
		{"cache_debug", "debug", "Cache debug info"},
		{"cache_unknown", "unknown", "Unknown level defaults to info"},
	}

	for _, tt := range tests {
		buffer.Reset()
		logger.LogCacheEvent(tt.event, tt.level, tt.message, zap.String("extra", "data"))

		if buffer.Len() == 0 {
			t.Errorf("Expected log output for event %s", tt.event)
			continue
		}

		logOutput := buffer.String()
		if !strings.Contains(logOutput, tt.event) {
			t.Errorf("Log should contain event '%s'", tt.event)
		}

		if !strings.Contains(logOutput, tt.message) {
			t.Errorf("Log should contain message '%s'", tt.message)
		}
	}
}

func TestLogger_LogConnection(t *testing.T) {
	config := MonitoringConfig{
		EnableLogging: true,
		LogLevel:      "info",
		SlowThreshold: 100 * time.Millisecond,
	}

	logger, buffer, err := createTestLogger(config)
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}

	tests := []struct {
		name   string
		action string
		addr   string
		err    error
	}{
		{
			name:   "Successful connection",
			action: "connect",
			addr:   "localhost:6379",
			err:    nil,
		},
		{
			name:   "Failed connection",
			action: "connect",
			addr:   "invalid:6379",
			err:    errors.New("connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buffer.Reset()
			logger.LogConnection(tt.action, tt.addr, tt.err)

			if buffer.Len() == 0 {
				t.Error("Expected connection log output")
				return
			}

			logOutput := buffer.String()
			if !strings.Contains(logOutput, tt.action) {
				t.Errorf("Log should contain action '%s'", tt.action)
			}

			if !strings.Contains(logOutput, tt.addr) {
				t.Errorf("Log should contain address '%s'", tt.addr)
			}

			if tt.err != nil {
				if !strings.Contains(logOutput, "error") {
					t.Error("Log should contain error level for failed connection")
				}
			} else {
				if !strings.Contains(logOutput, "info") {
					t.Error("Log should contain info level for successful connection")
				}
			}
		})
	}
}

func TestLogger_LogConfiguration(t *testing.T) {
	config := MonitoringConfig{
		EnableLogging: true,
		LogLevel:      "info",
		SlowThreshold: 100 * time.Millisecond,
	}

	logger, buffer, err := createTestLogger(config)
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}

	testConfig := map[string]interface{}{
		"type":         "memory",
		"max_size":     1000,
		"enable_stats": true,
	}

	logger.LogConfiguration(testConfig)

	if buffer.Len() == 0 {
		t.Error("Expected configuration log output")
		return
	}

	logOutput := buffer.String()
	if !strings.Contains(logOutput, "Cache configuration loaded") {
		t.Error("Log should contain configuration message")
	}
}

func TestLogger_DisabledLogging(t *testing.T) {
	config := MonitoringConfig{
		EnableLogging: false,
		LogLevel:      "info",
		SlowThreshold: 100 * time.Millisecond,
	}

	logger, buffer, err := createTestLogger(config)
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}

	// 测试所有日志方法在禁用时不产生输出
	logger.LogOperation("GET", "key", time.Millisecond, nil)
	logger.LogError("error", errors.New("test"), zap.String("field", "value"))
	logger.LogInfo("info", zap.String("field", "value"))
	logger.LogWarn("warn", zap.String("field", "value"))
	logger.LogDebug("debug", zap.String("field", "value"))
	logger.LogStats(CacheStats{})
	logger.LogCacheEvent("event", "info", "message")
	logger.LogConnection("connect", "localhost", nil)
	logger.LogConfiguration(map[string]string{"key": "value"})

	if buffer != nil && buffer.Len() > 0 {
		t.Error("Expected no log output when logging is disabled")
	}
}

func TestLogger_LogPerformance(t *testing.T) {
	config := MonitoringConfig{
		EnableLogging: true,
		LogLevel:      "info",
		SlowThreshold: 100 * time.Millisecond,
	}

	logger, buffer, err := createTestLogger(config)
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}

	metrics := &Metrics{
		AvgResponseTime: 50 * time.Millisecond,
		MaxResponseTime: 200 * time.Millisecond,
		MinResponseTime: 10 * time.Millisecond,
		SlowQueries:     5,
		Throughput:      1000.5,
		MemoryUsage:     1024 * 1024,
		CPUUsage:        0.75,
		ConnectionCount: 10,
	}

	logger.LogPerformance(metrics)

	if buffer.Len() == 0 {
		t.Error("Expected performance log output")
		return
	}

	logOutput := buffer.String()
	if !strings.Contains(logOutput, "Cache performance metrics") {
		t.Error("Log should contain performance metrics message")
	}

	// 验证包含性能指标
	if !strings.Contains(logOutput, "avg_response_time") {
		t.Error("Log should contain avg_response_time")
	}

	if !strings.Contains(logOutput, "throughput") {
		t.Error("Log should contain throughput")
	}
}

func TestStructuredLogger_Interface(t *testing.T) {
	// 验证Logger实现了StructuredLogger接口
	config := MonitoringConfig{
		EnableLogging: true,
		LogLevel:      "info",
		SlowThreshold: 100 * time.Millisecond,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	var _ StructuredLogger = logger
}

// 性能测试
func BenchmarkLogger_LogOperation(b *testing.B) {
	config := MonitoringConfig{
		EnableLogging: true,
		LogLevel:      "info",
		SlowThreshold: 100 * time.Millisecond,
	}

	logger, _, err := createTestLogger(config)
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.LogOperation("GET", "test_key", 50*time.Millisecond, nil)
	}
}

func BenchmarkLogger_LogStats(b *testing.B) {
	config := MonitoringConfig{
		EnableLogging: true,
		LogLevel:      "info",
		SlowThreshold: 100 * time.Millisecond,
	}

	logger, _, err := createTestLogger(config)
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}

	stats := CacheStats{
		Hits:          100,
		Misses:        20,
		Sets:          50,
		HitRate:       0.83,
		TotalRequests: 120,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.LogStats(stats)
	}
}

func BenchmarkParseLogLevel(b *testing.B) {
	levels := []string{"debug", "info", "warn", "error", "fatal", "unknown"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		level := levels[i%len(levels)]
		parseLogLevel(level)
	}
}
