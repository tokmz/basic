package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"
)

// TestLoggerIntegration 测试日志器的完整集成功能
func TestLoggerIntegration(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "integration.log")

	// 创建配置
	config := &Config{
		Level:       InfoLevel,
		Format:      JSONFormat,
		Environment: Testing,
		Output: OutputConfig{
			Mode:     FileMode,
			Path:     tempDir,
			Filename: "integration.log",
		},
		Rotation: RotationConfig{
			MaxSize:    1, // 1MB for testing
			MaxAge:     1,
			MaxBackups: 3,
			Compress:   true,
		},
		Buffer: BufferConfig{
			Size:        1024,
			FlushTime:   100 * time.Millisecond,
			AsyncBuffer: true,
		},
		EnableCaller:     true,
		EnableStacktrace: true,
		EnableColor:      false,
	}

	// 验证配置
	err := config.Validate()
	if err != nil {
		t.Fatalf("Config validation failed: %v", err)
	}

	// 创建日志器
	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// 测试基本日志记录
	logger.Info("Integration test started", zap.String("test_id", "integration_001"))
	logger.Debug("Debug message", zap.Int("debug_value", 42))
	logger.Warn("Warning message", zap.Bool("warning_flag", true))

	// 测试错误日志记录
	testErr := fmt.Errorf("test error for integration")
	loggerErr := NewLoggerError(SystemError, "TEST_001", "Integration test error")
	loggerErr.WithCause(testErr).WithSeverity(MediumSeverity).WithStackTrace()

	ctx := context.Background()
	logger.LogError(ctx, loggerErr, "integration_test", zap.String("operation", "test_error_logging"))

	// 测试性能监控
	perfMonitor := NewPerformanceMonitor(logger)
	start := time.Now()
	time.Sleep(10 * time.Millisecond) // 模拟操作
	perfMonitor.RecordLog(time.Since(start))
	perfMonitor.LogStats()

	// 强制刷新缓冲区
	err = logger.Sync()
	if err != nil {
		t.Errorf("Failed to sync logger: %v", err)
	}

	// 验证日志文件是否创建
	if _, err = os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("Log file was not created: %s", logFile)
	}

	// 测试日志分析
	analyzer := NewLogAnalyzer(logger)
	stats, err := analyzer.AnalyzeLogFile(logFile)
	if err != nil {
		t.Errorf("Failed to analyze log file: %v", err)
	} else {
		if stats.TotalLines == 0 {
			t.Error("No log lines found in analysis")
		}
		t.Logf("Log analysis: %d total lines, error rate: %.2f%%", stats.TotalLines, stats.ErrorRate*100)
	}
}

// TestLoggerWithErrorRecovery 测试日志器与错误恢复的集成
func TestLoggerWithErrorRecovery(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建配置
	config := &Config{
		Level:       DebugLevel,
		Format:      JSONFormat,
		Environment: Testing,
		Output: OutputConfig{
			Mode:     FileMode,
			Path:     tempDir,
			Filename: "recovery.log",
		},
		Buffer: BufferConfig{
			Size:        512,
			FlushTime:   50 * time.Millisecond,
			AsyncBuffer: false, // 同步模式便于测试
		},
	}

	// 创建日志器
	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// 创建错误恢复器
	recoveryConfig := DefaultErrorRecoveryConfig()
	recoveryConfig.MaxRetries = 3
	recoveryConfig.RetryInterval = 10 * time.Millisecond
	errorRecovery := NewErrorRecovery(logger, recoveryConfig)

	// 测试成功的操作
	ctx := context.Background()
	err = errorRecovery.ExecuteWithRecovery(ctx, "successful_operation", func() error {
		logger.Info("Operation executed successfully")
		return nil
	})
	if err != nil {
		t.Errorf("Successful operation should not return error: %v", err)
	}

	// 测试失败后恢复的操作
	attempts := 0
	err = errorRecovery.ExecuteWithRecovery(ctx, "retry_operation", func() error {
		attempts++
		if attempts < 3 {
			return fmt.Errorf("temporary failure %d", attempts)
		}
		logger.Info("Operation succeeded after retries", zap.Int("attempts", attempts))
		return nil
	})
	if err != nil {
		t.Errorf("Retry operation should succeed: %v", err)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}

	// 测试最终失败的操作
	err = errorRecovery.ExecuteWithRecovery(ctx, "failing_operation", func() error {
		return fmt.Errorf("persistent failure")
	})
	if err == nil {
		t.Error("Failing operation should return error")
	}
}

// TestLoggerFileRotation 测试日志文件轮转集成
func TestLoggerFileRotation(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建配置，设置很小的文件大小以触发轮转
	config := &Config{
		Level:       InfoLevel,
		Format:      TextFormat,
		Environment: Testing,
		Output: OutputConfig{
			Mode:     FileMode,
			Path:     tempDir,
			Filename: "rotation.log",
		},
		Rotation: RotationConfig{
			MaxSize:    1, // 1KB for testing
			MaxAge:     1,
			MaxBackups: 2,
			Compress:   false, // 不压缩便于测试
		},
		Buffer: BufferConfig{
			Size:        256,
			FlushTime:   10 * time.Millisecond,
			AsyncBuffer: false,
		},
	}

	// 创建日志器
	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// 创建文件管理器
	fileManagerConfig := DefaultFileManagerConfig()
	fileManagerConfig.LogDir = tempDir
	fileManagerConfig.MaxBackups = 2
	fileManager := NewFileManager(fileManagerConfig, logger)

	// 确保日志目录存在
	err = fileManager.EnsureLogDir()
	if err != nil {
		t.Fatalf("Failed to ensure log directory: %v", err)
	}

	// 写入大量日志以触发轮转
	for i := 0; i < 100; i++ {
		logger.Info("Log message for rotation test",
			zap.Int("iteration", i),
			zap.String("data", fmt.Sprintf("This is a longer message to increase file size - iteration %d", i)),
		)
		// 强制刷新
		logger.Sync()
	}

	// 等待一段时间确保所有操作完成
	time.Sleep(100 * time.Millisecond)

	// 检查是否有多个日志文件（表示发生了轮转）
	files, err := filepath.Glob(filepath.Join(tempDir, "*.log*"))
	if err != nil {
		t.Fatalf("Failed to list log files: %v", err)
	}

	t.Logf("Found %d log files after rotation test", len(files))
	for _, file := range files {
		info, _ := os.Stat(file)
		t.Logf("Log file: %s, size: %d bytes", filepath.Base(file), info.Size())
	}

	// 应该至少有主日志文件
	if len(files) == 0 {
		t.Error("No log files found")
	}
}

// TestLoggerConcurrency 测试日志器的并发安全性
func TestLoggerConcurrency(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建配置
	config := &Config{
		Level:       InfoLevel,
		Format:      JSONFormat,
		Environment: Testing,
		Output: OutputConfig{
			Mode:     FileMode,
			Path:     tempDir,
			Filename: "concurrent.log",
		},
		Buffer: BufferConfig{
			Size:        2048,
			FlushTime:   50 * time.Millisecond,
			AsyncBuffer: true,
		},
	}

	// 创建日志器
	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// 并发写入日志
	const numGoroutines = 10
	const messagesPerGoroutine = 50

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer func() { done <- true }()
			for j := 0; j < messagesPerGoroutine; j++ {
				logger.Info("Concurrent log message",
					zap.Int("goroutine_id", goroutineID),
					zap.Int("message_id", j),
					zap.String("timestamp", time.Now().Format(time.RFC3339)),
				)
				// 随机延迟
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// 强制刷新所有缓冲区
	err = logger.Sync()
	if err != nil {
		t.Errorf("Failed to sync logger: %v", err)
	}

	// 验证日志文件
	logFile := filepath.Join(tempDir, "concurrent.log")
	if _, err = os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("Log file was not created: %s", logFile)
	}

	// 分析日志文件
	analyzer := NewLogAnalyzer(logger)
	stats, err := analyzer.AnalyzeLogFile(logFile)
	if err != nil {
		t.Errorf("Failed to analyze log file: %v", err)
	} else {
		expectedLines := int64(numGoroutines * messagesPerGoroutine)
		t.Logf("Concurrent test: expected %d lines, found %d lines", expectedLines, stats.TotalLines)

		// 允许一些误差，因为可能有其他日志消息
		if stats.TotalLines < expectedLines {
			t.Errorf("Expected at least %d log lines, got %d", expectedLines, stats.TotalLines)
		}
	}
}

// TestZeroAllocLoggerIntegration 测试零分配日志器的集成
func TestZeroAllocLoggerIntegration(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建配置
	config := &Config{
		Level:       InfoLevel,
		Format:      JSONFormat,
		Environment: Production,
		Output: OutputConfig{
			Mode:     FileMode,
			Path:     tempDir,
			Filename: "zero_alloc.log",
		},
		Buffer: BufferConfig{
			Size:        1024,
			FlushTime:   100 * time.Millisecond,
			AsyncBuffer: false,
		},
	}

	// 创建日志器
	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// 创建零分配日志器
	zeroAllocLogger := NewZeroAllocLogger(logger)

	// 创建性能监控器
	perfMonitor := NewPerformanceMonitor(logger)

	// 测试零分配日志记录
	start := time.Now()
	for i := 0; i < 1000; i++ {
		zeroAllocLogger.Info("Zero allocation log message")
	}
	duration := time.Since(start)
	perfMonitor.RecordLog(duration)

	// 测试带字段的零分配日志记录
	start = time.Now()
	for i := 0; i < 1000; i++ {
		zeroAllocLogger.AddInt("iteration", i).AddString("type", "performance_test").Info("Zero allocation with fields")
	}
	duration = time.Since(start)
	perfMonitor.RecordLog(duration)

	// 记录性能统计
	perfMonitor.LogStats()

	// 强制刷新
	err = logger.Sync()
	if err != nil {
		t.Errorf("Failed to sync logger: %v", err)
	}

	// 验证日志文件
	logFile := filepath.Join(tempDir, "zero_alloc.log")
	if _, err = os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("Log file was not created: %s", logFile)
	}

	// 分析日志文件
	analyzer := NewLogAnalyzer(logger)
	stats, err := analyzer.AnalyzeLogFile(logFile)
	if err != nil {
		t.Errorf("Failed to analyze log file: %v", err)
	} else {
		t.Logf("Zero allocation test: %d total lines, avg line size: %.2f bytes",
			stats.TotalLines, stats.AvgLineSize)

		// 应该有大量的日志行
		if stats.TotalLines < 2000 {
			t.Errorf("Expected at least 2000 log lines, got %d", stats.TotalLines)
		}
	}
}
