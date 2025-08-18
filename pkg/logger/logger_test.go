package logger

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLogger(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	logFile := "test.log"

	// 测试配置
	config := Config{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: OutputConfig{
			Mode:     FileMode,
			Filename: logFile,
			Path:     tempDir,
		},
		Rotation: RotationConfig{
			MaxSize:    1,
			MaxAge:     1,
			MaxBackups: 1,
			Compress:   false,
		},
	}

	// 创建日志器
	logger, err := New(&config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// 测试基本日志记录
	logger.Info("Test info message", String("key", "value"))
	logger.Warn("Test warn message", Int("number", 42))
	logger.Error("Test error message", Bool("flag", true))

	// 同步确保写入
	logger.Sync()

	// 等待文件系统操作完成
	time.Sleep(100 * time.Millisecond)

	// 检查文件是否存在
	fullLogPath := filepath.Join(tempDir, logFile)
	if _, err := os.Stat(fullLogPath); os.IsNotExist(err) {
		// 如果文件不存在，检查目录是否存在
		if _, dirErr := os.Stat(tempDir); os.IsNotExist(dirErr) {
			t.Errorf("Temp directory was not created: %s", tempDir)
		} else {
			// 列出目录内容进行调试
			files, _ := os.ReadDir(tempDir)
			t.Logf("Directory contents: %v", files)
			t.Errorf("Log file was not created: %s", fullLogPath)
		}
	}
}

func TestLoggerLevels(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "levels.log")

	config := Config{
		Level:  DebugLevel,
		Format: TextFormat,
		Output: OutputConfig{
			Mode:     FileMode,
			Filename: logFile,
			Path:     tempDir,
		},
	}

	logger, err := New(&config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// 测试所有日志级别
	logger.Debug("Debug message")
	logger.Info("Info message")
	logger.Warn("Warn message")
	logger.Error("Error message")

	// 测试级别设置
	logger.SetLevel(WarnLevel)
	if logger.GetLevel() != WarnLevel {
		t.Errorf("Expected level %v, got %v", WarnLevel, logger.GetLevel())
	}

	logger.Sync()
}

func TestLoggerWithContext(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "context.log")

	config := Config{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: OutputConfig{
			Mode:     FileMode,
			Filename: logFile,
			Path:     tempDir,
		},
	}

	logger, err := New(&config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// 测试上下文日志
	ctx := context.Background()
	ctx = WithRequestID(ctx, "test-request-123")
	ctx = WithTraceID(ctx, "test-trace-456")

	logger.WithContext(ctx).Info("Context message", String("action", "test"))
	logger.Sync()
}

func TestZeroAllocLogger(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "zero_alloc.log")

	config := Config{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: OutputConfig{
			Mode:     FileMode,
			Filename: logFile,
			Path:     tempDir,
		},
	}

	logger, err := New(&config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// 测试零分配日志器
	zeroLogger := NewZeroAllocLogger(logger)
	zeroLogger.AddString("key1", "value1").
		AddInt("key2", 42).
		AddBool("key3", true).
		AddTime("key4", time.Now()).
		Info("Zero allocation message")

	logger.Sync()
}

func TestAsyncBuffer(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "async.log")

	config := Config{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: OutputConfig{
			Mode:     FileMode,
			Filename: logFile,
			Path:     tempDir,
		},
		Buffer: BufferConfig{
			Size:      1024,
			FlushTime: 100 * time.Millisecond,
		},
	}

	logger, err := New(&config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// 写入多条日志
	for i := 0; i < 10; i++ {
		logger.Info("Async message", Int("index", i))
	}

	// 等待缓冲区刷新
	time.Sleep(200 * time.Millisecond)
	logger.Sync()
}

func TestGlobalLogger(t *testing.T) {
	// 测试全局日志器初始化
	err := InitDevelopment()
	if err != nil {
		t.Fatalf("Failed to init development logger: %v", err)
	}

	// 测试全局日志函数
	Debug("Global debug message", String("test", "debug"))
	Info("Global info message", String("test", "info"))
	Warn("Global warn message", String("test", "warn"))
	ErrorLog("Global error message", String("test", "error"))

	// 测试格式化日志
	Debugf("Debug: %s = %d", "count", 42)
	Infof("Info: %s = %d", "count", 42)
	Warnf("Warn: %s = %d", "count", 42)
	Errorf("Error: %s = %d", "count", 42)

	// 测试键值对日志
	Debugw("Debug message", "key1", "value1", "key2", 42)
	Infow("Info message", "key1", "value1", "key2", 42)
	Warnw("Warn message", "key1", "value1", "key2", 42)
	Errorw("Error message", "key1", "value1", "key2", 42)

	Sync()
}

func TestPerformanceMonitor(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "perf.log")

	config := Config{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: OutputConfig{
			Mode:     FileMode,
			Filename: logFile,
			Path:     tempDir,
		},
	}

	logger, err := New(&config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// 创建性能监控器
	monitor := NewPerformanceMonitor(logger)

	// 模拟一些日志记录
	for i := 0; i < 100; i++ {
		start := time.Now()
		logger.Info("Performance test", Int("iteration", i))
		monitor.RecordLog(time.Since(start))
	}

	// 获取统计信息
	stats := monitor.GetStats()
	if stats.TotalLogs != 100 {
		t.Errorf("Expected 100 logs, got %d", stats.TotalLogs)
	}

	// 记录统计日志
	monitor.LogStats()
	logger.Sync()
}

func TestBatchLogger(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "batch.log")

	config := Config{
		Level:  DebugLevel, // 设置为Debug级别以测试所有日志级别
		Format: JSONFormat,
		Output: OutputConfig{
			Mode:     FileMode,
			Filename: logFile,
			Path:     tempDir,
		},
	}

	logger, err := New(&config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// 创建批量日志器
	batchLogger := NewBatchLogger(logger, 3)

	// 测试不同级别的日志
	batchLogger.Add(DebugLevel, "Debug message", String("type", "debug"))
	batchLogger.Add(InfoLevel, "Info message", String("type", "info"))
	batchLogger.Add(WarnLevel, "Warn message", String("type", "warn"))
	// 这里应该触发自动刷新，因为批次大小为3

	// 添加更多日志
	batchLogger.Add(ErrorLevel, "Error message", String("type", "error"))
	// 注意：不测试FatalLevel，因为它会导致程序退出

	// 手动刷新剩余的日志
	batchLogger.Flush()
	
	// 测试空批次刷新
	batchLogger.Flush()
	
	logger.Sync()
}

func TestMiddleware(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "middleware.log")

	config := Config{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: OutputConfig{
			Mode:     FileMode,
			Filename: logFile,
			Path:     tempDir,
		},
	}

	logger, err := New(&config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// 测试请求日志记录器
	requestLogger := NewRequestLogger(logger, DefaultRequestLoggerConfig())
	ctx := WithRequestID(context.Background(), "req-123")
	requestLogger.LogRequest(ctx, "GET", "/api/test", "test-agent", "127.0.0.1", 200, 50*time.Millisecond, 1024, 2048)

	// 测试错误处理器
	errorHandler := NewErrorHandler(logger, DefaultErrorHandlerConfig())
	errorHandler.HandleError(ctx, err, ErrorLevel, "Test error handling")

	// 测试性能追踪器
	perfTracker := NewPerformanceTracker(logger, DefaultPerformanceTrackerConfig())
	err = perfTracker.TrackOperation(ctx, "test_operation", func() error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Errorf("Unexpected error from tracked operation: %v", err)
	}

	logger.Sync()
}

func BenchmarkLogger(b *testing.B) {
	tempDir := b.TempDir()
	logFile := "bench.log"

	config := Config{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: OutputConfig{
			Mode:     FileMode,
			Filename: logFile,
			Path:     tempDir,
		},
		Buffer: BufferConfig{
			Size:      8192,
			FlushTime: 100 * time.Millisecond,
		},
	}

	logger, err := New(&config)
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("Benchmark message",
				String("key1", "value1"),
				Int("key2", 42),
				Bool("key3", true),
				Time("key4", time.Now()),
			)
		}
	})
}

func BenchmarkZeroAllocLogger(b *testing.B) {
	tempDir := b.TempDir()
	logFile := "bench_zero.log"

	config := Config{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: OutputConfig{
			Mode:     FileMode,
			Filename: logFile,
			Path:     tempDir,
		},
		Buffer: BufferConfig{
			Size:      8192,
			FlushTime: 100 * time.Millisecond,
		},
	}

	logger, err := New(&config)
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			zeroLogger := NewZeroAllocLogger(logger)
			zeroLogger.AddString("key1", "value1").
				AddInt("key2", 42).
				AddBool("key3", true).
				AddTime("key4", time.Now()).
				Info("Benchmark zero alloc message")
		}
	})
}
