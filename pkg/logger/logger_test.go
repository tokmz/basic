package logger

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "默认配置",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name:    "开发环境配置",
			config:  DevelopmentConfig(),
			wantErr: false,
		},
		{
			name:    "生产环境配置",
			config:  ProductionConfig("test-service", "production"),
			wantErr: false,
		},
		{
			name:    "nil配置",
			config:  nil,
			wantErr: false,
		},
		{
			name: "无效日志级别",
			config: &Config{
				Level:  "invalid",
				Format: JSONFormat,
				Console: ConsoleConfig{
					Enabled: true,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Error("期望错误，但没有返回错误")
				}
				return
			}

			if err != nil {
				t.Errorf("New() error = %v", err)
				return
			}

			if logger == nil {
				t.Error("返回的日志器为nil")
				return
			}

			// 测试基本日志功能
			logger.Info("test message")
			logger.Infof("test formatted message: %s", "value")

			// 清理
			logger.Close()
		})
	}
}

func TestLogger_WithFields(t *testing.T) {
	logger, err := New(DevelopmentConfig())
	if err != nil {
		t.Fatalf("创建日志器失败: %v", err)
	}
	defer logger.Close()

	fields := Fields{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	fieldLogger := logger.WithFields(fields)
	fieldLogger.Info("test message with fields")

	// 测试空字段
	emptyFieldLogger := logger.WithFields(Fields{})
	if emptyFieldLogger == nil {
		t.Error("WithFields返回nil")
	}
}

func TestLogger_WithContext(t *testing.T) {
	logger, err := New(DevelopmentConfig())
	if err != nil {
		t.Fatalf("创建日志器失败: %v", err)
	}
	defer logger.Close()

	// 创建带有追踪信息的上下文
	ctx := context.Background()
	ctx = context.WithValue(ctx, "trace_id", "test-trace-123")
	ctx = context.WithValue(ctx, "request_id", "test-request-456")
	ctx = context.WithValue(ctx, "user_id", "test-user-789")

	contextLogger := logger.WithContext(ctx)
	contextLogger.Info("test message with context")

	// 测试nil上下文
	nilContextLogger := logger.WithContext(nil)
	if nilContextLogger == nil {
		t.Error("WithContext返回nil")
	}
}

func TestLogger_LogLevels(t *testing.T) {
	logger, err := New(DevelopmentConfig())
	if err != nil {
		t.Fatalf("创建日志器失败: %v", err)
	}
	defer logger.Close()

	// 测试各种日志级别
	logger.Debug("debug message", zap.String("level", "debug"))
	logger.Info("info message", zap.String("level", "info"))
	logger.Warn("warn message", zap.String("level", "warn"))
	logger.Error("error message", zap.String("level", "error"))

	// 测试格式化日志
	logger.Debugf("debug formatted: %s", "value")
	logger.Infof("info formatted: %s", "value")
	logger.Warnf("warn formatted: %s", "value")
	logger.Errorf("error formatted: %s", "value")
}

func TestLogger_FileOutput(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	config := DefaultConfig()
	config.Console.Enabled = false
	config.File.Enabled = true
	config.File.Filename = logFile

	logger, err := New(config)
	if err != nil {
		t.Fatalf("创建日志器失败: %v", err)
	}
	defer logger.Close()

	// 写入一些日志
	logger.Info("test file log message")
	logger.Error("test file error message")

	// 同步日志
	if err := logger.Sync(); err != nil {
		t.Errorf("同步日志失败: %v", err)
	}

	// 检查文件是否存在
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("日志文件不存在: %s", logFile)
	}
}

func TestLogger_Rotation(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "rotation.log")

	config := DefaultConfig()
	config.Console.Enabled = false
	config.File.Enabled = true
	config.File.Filename = logFile
	config.File.Rotation.MaxSize = 1  // 1MB
	config.File.Rotation.MaxAge = 1   // 1天
	config.File.Rotation.MaxBackups = 3
	config.File.Rotation.Compress = false

	logger, err := New(config)
	if err != nil {
		t.Fatalf("创建日志器失败: %v", err)
	}
	defer logger.Close()

	// 写入日志
	for i := 0; i < 10; i++ {
		logger.Infof("rotation test message %d", i)
	}

	// 同步日志
	if err := logger.Sync(); err != nil {
		t.Errorf("同步日志失败: %v", err)
	}

	// 检查文件是否存在
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("日志文件不存在: %s", logFile)
	}
}

func TestLogger_GetSetLevel(t *testing.T) {
	logger, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("创建日志器失败: %v", err)
	}
	defer logger.Close()

	// 测试获取级别
	level := logger.GetLevel()
	if level != InfoLevel {
		t.Errorf("期望日志级别为 %s, 实际为 %s", InfoLevel, level)
	}

	// 测试设置级别
	if err := logger.SetLevel(DebugLevel); err != nil {
		t.Errorf("设置日志级别失败: %v", err)
	}

	newLevel := logger.GetLevel()
	if newLevel != DebugLevel {
		t.Errorf("期望日志级别为 %s, 实际为 %s", DebugLevel, newLevel)
	}

	// 测试无效级别
	if err := logger.SetLevel("invalid"); err == nil {
		t.Error("期望设置无效级别时返回错误")
	}
}

func TestGlobalLogger(t *testing.T) {
	// 测试初始化全局日志器
	config := DevelopmentConfig()
	if err := InitGlobal(config); err != nil {
		t.Fatalf("初始化全局日志器失败: %v", err)
	}

	// 测试全局日志函数
	Info("global info message")
	Infof("global info formatted: %s", "value")
	Debug("global debug message")
	Warn("global warn message")
	Error("global error message")

	// 测试全局带字段日志
	WithFields(Fields{"global": true}).Info("global field message")

	// 测试全局带上下文日志
	ctx := context.WithValue(context.Background(), "test_key", "test_value")
	WithContext(ctx).Info("global context message")

	// 测试同步 (忽略stdout同步错误，这在某些环境下是正常的)
	if err := Sync(); err != nil {
		// 忽略stdout同步错误
		errMsg := err.Error()
		if !strings.Contains(errMsg, "inappropriate ioctl for device") && 
		   !strings.Contains(errMsg, "bad file descriptor") {
			t.Errorf("同步全局日志器失败: %v", err)
		}
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		modify func(*Config)
	}{
		{
			name:   "空服务名",
			config: DefaultConfig(),
			modify: func(c *Config) { c.ServiceName = "" },
		},
		{
			name:   "空环境",
			config: DefaultConfig(),
			modify: func(c *Config) { c.ServiceEnv = "" },
		},
		{
			name:   "空时间格式",
			config: DefaultConfig(),
			modify: func(c *Config) { c.TimeFormat = "" },
		},
		{
			name:   "启用文件但无文件名",
			config: DefaultConfig(),
			modify: func(c *Config) {
				c.File.Enabled = true
				c.File.Filename = ""
			},
		},
		{
			name:   "无效轮转配置",
			config: DefaultConfig(),
			modify: func(c *Config) {
				c.File.Rotation.MaxSize = 0
				c.File.Rotation.MaxAge = 0
				c.File.Rotation.MaxBackups = -1
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.config
			if tt.modify != nil {
				tt.modify(config)
			}

			if err := config.Validate(); err != nil {
				t.Errorf("配置验证失败: %v", err)
			}

			// 验证默认值是否被正确设置
			if config.ServiceName == "" {
				t.Error("ServiceName应该有默认值")
			}
			if config.ServiceEnv == "" {
				t.Error("ServiceEnv应该有默认值")
			}
			if config.TimeFormat == "" {
				t.Error("TimeFormat应该有默认值")
			}
		})
	}
}

func TestPresetConfigs(t *testing.T) {
	// 测试默认配置
	defaultConfig := DefaultConfig()
	if defaultConfig.Level != InfoLevel {
		t.Errorf("默认配置级别应该为 %s", InfoLevel)
	}
	if !defaultConfig.Console.Enabled {
		t.Error("默认配置应该启用控制台输出")
	}

	// 测试开发配置
	devConfig := DevelopmentConfig()
	if devConfig.Level != DebugLevel {
		t.Errorf("开发配置级别应该为 %s", DebugLevel)
	}
	if devConfig.Format != ConsoleFormat {
		t.Errorf("开发配置格式应该为 %s", ConsoleFormat)
	}

	// 测试生产配置
	prodConfig := ProductionConfig("test-service", "production")
	if prodConfig.Level != InfoLevel {
		t.Errorf("生产配置级别应该为 %s", InfoLevel)
	}
	if prodConfig.Format != JSONFormat {
		t.Errorf("生产配置格式应该为 %s", JSONFormat)
	}
	if prodConfig.Console.Enabled {
		t.Error("生产配置不应该启用控制台输出")
	}
	if !prodConfig.File.Enabled {
		t.Error("生产配置应该启用文件输出")
	}
}

// 基准测试
func BenchmarkLogger_Info(b *testing.B) {
	logger, _ := New(DefaultConfig())
	defer logger.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark test message")
	}
}

func BenchmarkLogger_InfoWithFields(b *testing.B) {
	logger, _ := New(DefaultConfig())
	defer logger.Close()

	fields := Fields{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.WithFields(fields).Info("benchmark test message with fields")
	}
}

func BenchmarkGlobal_Info(b *testing.B) {
	InitGlobal(DefaultConfig())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("benchmark global test message")
	}
}

func BenchmarkLogger_Infof(b *testing.B) {
	logger, _ := New(DefaultConfig())
	defer logger.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Infof("benchmark test message: %d", i)
	}
}