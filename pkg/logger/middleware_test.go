package logger

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestWithLogger(t *testing.T) {
	ctx := context.Background()
	logger := GetGlobalLogger()

	newCtx := WithLogger(ctx, logger)
	retrievedLogger := FromContext(newCtx)

	assert.Equal(t, logger, retrievedLogger)
}

func TestFromContext(t *testing.T) {
	// 测试上下文中没有日志器的情况
	ctx := context.Background()
	logger := FromContext(ctx)
	assert.NotNil(t, logger)
	assert.Equal(t, GetGlobalLogger(), logger)

	// 测试上下文中有日志器的情况
	customLogger := GetGlobalLogger()
	ctxWithLogger := WithLogger(ctx, customLogger)
	retrievedLogger := FromContext(ctxWithLogger)
	assert.Equal(t, customLogger, retrievedLogger)
}

func TestWithRequestID(t *testing.T) {
	ctx := context.Background()
	requestID := "test-request-123"

	newCtx := WithRequestID(ctx, requestID)
	retrievedID := GetRequestID(newCtx)

	assert.Equal(t, requestID, retrievedID)
}

func TestGetRequestID(t *testing.T) {
	// 测试上下文中没有请求ID的情况
	ctx := context.Background()
	requestID := GetRequestID(ctx)
	assert.Empty(t, requestID)

	// 测试上下文中有请求ID的情况
	expectedID := "test-request-456"
	ctxWithID := WithRequestID(ctx, expectedID)
	retrievedID := GetRequestID(ctxWithID)
	assert.Equal(t, expectedID, retrievedID)
}

func TestWithTraceID(t *testing.T) {
	ctx := context.Background()
	traceID := "trace-123-456"

	newCtx := WithTraceID(ctx, traceID)
	retrievedID := GetTraceID(newCtx)

	assert.Equal(t, traceID, retrievedID)
}

func TestGetTraceID(t *testing.T) {
	// 测试上下文中没有追踪ID的情况
	ctx := context.Background()
	traceID := GetTraceID(ctx)
	assert.Empty(t, traceID)

	// 测试上下文中有追踪ID的情况
	expectedID := "trace-789-012"
	ctxWithID := WithTraceID(ctx, expectedID)
	retrievedID := GetTraceID(ctxWithID)
	assert.Equal(t, expectedID, retrievedID)
}

func TestDefaultRequestLoggerConfig(t *testing.T) {
	config := DefaultRequestLoggerConfig()

	assert.Contains(t, config.SkipPaths, "/health")
	assert.Contains(t, config.SkipPaths, "/metrics")
	assert.Contains(t, config.SkipPaths, "/favicon.ico")
	assert.False(t, config.LogRequestBody)
	assert.False(t, config.LogResponseBody)
	assert.Equal(t, 1024, config.MaxBodySize)
	assert.Equal(t, time.Second, config.SlowThreshold)
}

func TestNewRequestLogger(t *testing.T) {
	logger := GetGlobalLogger()
	config := DefaultRequestLoggerConfig()

	rl := NewRequestLogger(logger, config)

	assert.NotNil(t, rl)
	assert.Equal(t, logger, rl.logger)
	assert.Equal(t, config, rl.config)
}

func TestRequestLogger_shouldSkip(t *testing.T) {
	logger := GetGlobalLogger()
	config := RequestLoggerConfig{
		SkipPaths: []string{"/health", "/metrics"},
	}
	rl := NewRequestLogger(logger, config)

	assert.True(t, rl.shouldSkip("/health"))
	assert.True(t, rl.shouldSkip("/metrics"))
	assert.False(t, rl.shouldSkip("/api/users"))
	assert.False(t, rl.shouldSkip("/"))
}

func TestRequestLogger_LogRequest(t *testing.T) {
	logger := GetGlobalLogger()
	config := DefaultRequestLoggerConfig()
	rl := NewRequestLogger(logger, config)

	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-123")
	ctx = WithTraceID(ctx, "trace-456")

	// 测试正常请求
	rl.LogRequest(ctx, "GET", "/api/users", "test-agent", "127.0.0.1", 200, 100*time.Millisecond, 1024, 2048)

	// 测试慢请求
	rl.LogRequest(ctx, "POST", "/api/users", "test-agent", "127.0.0.1", 201, 2*time.Second, 2048, 1024)

	// 测试错误请求
	rl.LogRequest(ctx, "GET", "/api/users/999", "test-agent", "127.0.0.1", 404, 50*time.Millisecond, 512, 256)

	// 测试跳过的路径
	rl.LogRequest(ctx, "GET", "/health", "test-agent", "127.0.0.1", 200, 10*time.Millisecond, 100, 50)
}

func TestDefaultErrorHandlerConfig(t *testing.T) {
	config := DefaultErrorHandlerConfig()

	assert.True(t, config.LogStackTrace)
	assert.Equal(t, ErrorLevel, config.StackTraceLevel)
	assert.Equal(t, 32, config.MaxStackDepth)
}

func TestNewErrorHandler(t *testing.T) {
	logger := GetGlobalLogger()
	config := DefaultErrorHandlerConfig()

	eh := NewErrorHandler(logger, config)

	assert.NotNil(t, eh)
	assert.Equal(t, logger, eh.logger)
	assert.Equal(t, config, eh.config)
}

func TestErrorHandler_HandleError(t *testing.T) {
	logger := GetGlobalLogger()
	config := DefaultErrorHandlerConfig()
	eh := NewErrorHandler(logger, config)

	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-123")

	testErr := errors.New("test error")

	// 测试不同级别的错误处理
	eh.HandleError(ctx, testErr, ErrorLevel, "Test error occurred")
	eh.HandleError(ctx, testErr, WarnLevel, "Test warning", zap.String("extra", "info"))
	eh.HandleError(ctx, testErr, InfoLevel, "Test info")

	// 测试不记录堆栈的配置
	config.LogStackTrace = false
	eh = NewErrorHandler(logger, config)
	eh.HandleError(ctx, testErr, ErrorLevel, "Error without stack trace")
}

func TestErrorHandler_getStackTrace(t *testing.T) {
	logger := GetGlobalLogger()
	config := DefaultErrorHandlerConfig()
	eh := NewErrorHandler(logger, config)

	stackTrace := eh.getStackTrace()
	assert.NotEmpty(t, stackTrace)
	// 验证堆栈跟踪包含运行时信息
	assert.Contains(t, stackTrace, "runtime")
}

func TestDefaultRecoveryHandlerConfig(t *testing.T) {
	config := DefaultRecoveryHandlerConfig()

	assert.True(t, config.LogStackTrace)
	assert.Equal(t, 32, config.MaxStackDepth)
	assert.Equal(t, ErrorLevel, config.PanicLevel)
}

func TestNewRecoveryHandler(t *testing.T) {
	logger := GetGlobalLogger()
	config := DefaultRecoveryHandlerConfig()

	rh := NewRecoveryHandler(logger, config)

	assert.NotNil(t, rh)
	assert.Equal(t, logger, rh.logger)
	assert.Equal(t, config, rh.config)
}

func TestRecoveryHandler_HandlePanic(t *testing.T) {
	// 创建一个静默的logger配置
	loggerConfig := DefaultConfig()
	loggerConfig.Level = ErrorLevel // 设置为Error级别，避免Fatal导致程序退出
	logger, _ := New(loggerConfig)
	defer logger.Close()
	
	config := DefaultRecoveryHandlerConfig()
	config.PanicLevel = ErrorLevel // 设置panic日志级别为Error
	rh := NewRecoveryHandler(logger, config)

	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-123")

	// 测试字符串panic
	rh.HandlePanic(ctx, "string panic")

	// 测试错误panic
	rh.HandlePanic(ctx, errors.New("error panic"))

	// 测试其他类型panic
	rh.HandlePanic(ctx, 42)
	rh.HandlePanic(ctx, map[string]interface{}{"key": "value"})

	// 测试不记录堆栈的配置
	config.LogStackTrace = false
	rh = NewRecoveryHandler(logger, config)
	rh.HandlePanic(ctx, "panic without stack trace")
}

func TestRecoveryHandler_getStackTrace(t *testing.T) {
	logger := GetGlobalLogger()
	config := DefaultRecoveryHandlerConfig()
	rh := NewRecoveryHandler(logger, config)

	stackTrace := rh.getStackTrace()
	assert.NotEmpty(t, stackTrace)
	// 验证堆栈跟踪包含运行时信息
	assert.Contains(t, stackTrace, "runtime")
}

func TestDefaultPerformanceTrackerConfig(t *testing.T) {
	config := DefaultPerformanceTrackerConfig()
// 验证默认值
	assert.Equal(t, 100*time.Millisecond, config.SlowThreshold)
	assert.Equal(t, WarnLevel, config.LogLevel)
	assert.True(t, config.LogDetails)
}

func TestNewPerformanceTracker(t *testing.T) {
	logger := GetGlobalLogger()
	config := DefaultPerformanceTrackerConfig()

	pt := NewPerformanceTracker(logger, config)

	assert.NotNil(t, pt)
	assert.Equal(t, logger, pt.logger)
	assert.Equal(t, config, pt.config)
}

func TestPerformanceTracker_TrackOperation(t *testing.T) {
	// 创建一个静默的logger配置
	loggerConfig := DefaultConfig()
	loggerConfig.Level = ErrorLevel // 只记录错误级别以上的日志
	logger, _ := New(loggerConfig)
	
	config := DefaultPerformanceTrackerConfig()
	config.SlowThreshold = 1 * time.Second // 设置较高的阈值避免慢操作日志
	config.LogLevel = ErrorLevel // 只在错误级别记录
	pt := NewPerformanceTracker(logger, config)

	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-123")

	// 测试快速操作
	err := pt.TrackOperation(ctx, "fast_operation", func() error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})
	assert.NoError(t, err)

	// 测试正常操作（不触发慢操作警告）
	err = pt.TrackOperation(ctx, "normal_operation", func() error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})
	assert.NoError(t, err)

	// 测试操作成功（避免错误日志）
	err = pt.TrackOperation(ctx, "success_operation", func() error {
		return nil
	})
	assert.NoError(t, err)
}