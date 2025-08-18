package logger

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggerError(t *testing.T) {
	t.Run("基本错误创建", func(t *testing.T) {
		err := NewLoggerError(SystemError, "SYS_001", "系统错误")
		assert.Equal(t, SystemError, err.Type)
		assert.Equal(t, "SYS_001", err.Code)
		assert.Equal(t, "系统错误", err.Message)
		assert.Equal(t, MediumSeverity, err.Severity)
		assert.False(t, err.Retryable)
		assert.NotZero(t, err.Timestamp)
	})

	t.Run("错误链式构建", func(t *testing.T) {
		causeErr := errors.New("原始错误")
		err := NewLoggerError(NetworkError, "NET_001", "网络错误").
			WithCause(causeErr).
			WithSeverity(HighSeverity).
			WithRetryable(true).
			WithOperation("connect").
			WithContext("host", "localhost").
			WithStackTrace()

		assert.Equal(t, NetworkError, err.Type)
		assert.Equal(t, "NET_001", err.Code)
		assert.Equal(t, causeErr, err.Cause)
		assert.Equal(t, HighSeverity, err.Severity)
		assert.True(t, err.Retryable)
		assert.Equal(t, "connect", err.Operation)
		assert.Equal(t, "localhost", err.Context["host"])
		assert.NotEmpty(t, err.StackTrace)
	})

	t.Run("错误接口实现", func(t *testing.T) {
		causeErr := errors.New("原始错误")
		err := NewLoggerError(DatabaseError, "DB_001", "数据库错误").WithCause(causeErr)

		// 测试Error()方法
		errorStr := err.Error()
		assert.Contains(t, errorStr, "DB_001")
		assert.Contains(t, errorStr, "数据库错误")
		assert.Contains(t, errorStr, "原始错误")

		// 测试Unwrap()方法
		assert.Equal(t, causeErr, err.Unwrap())

		// 测试Is()方法
		targetErr := NewLoggerError(DatabaseError, "DB_001", "其他消息")
		assert.True(t, err.Is(targetErr))

		differentErr := NewLoggerError(NetworkError, "NET_001", "网络错误")
		assert.False(t, err.Is(differentErr))
	})
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name     string
		creator  func(string, string) *LoggerError
		expType  ErrorType
		expSeverity ErrorSeverity
		expRetryable bool
	}{
		{"系统错误", NewSystemError, SystemError, HighSeverity, false},
		{"网络错误", NewNetworkError, NetworkError, MediumSeverity, true},
		{"数据库错误", NewDatabaseError, DatabaseError, HighSeverity, false},
		{"认证错误", NewAuthError, AuthError, MediumSeverity, false},
		{"验证错误", NewValidationError, ValidationError, LowSeverity, false},
		{"业务错误", NewBusinessError, BusinessError, MediumSeverity, false},
		{"配置错误", NewConfigError, ConfigError, CriticalSeverity, false},
		{"外部服务错误", NewExternalError, ExternalError, MediumSeverity, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.creator("TEST_001", "测试错误")
			assert.Equal(t, tt.expType, err.Type)
			assert.Equal(t, tt.expSeverity, err.Severity)
			assert.Equal(t, tt.expRetryable, err.Retryable)
		})
	}
}

func TestErrorClassifier(t *testing.T) {
	logger := GetGlobalLogger()
	classifier := NewErrorClassifier(logger)

	tests := []struct {
		name        string
		inputError  error
		expectedType ErrorType
		expectedCode string
	}{
		{
			"数据库连接错误",
			errors.New("database connection failed"),
			DatabaseError,
			ErrCodeDatabaseConnection,
		},
		{
			"网络连接错误",
			errors.New("connection refused"),
			NetworkError,
			ErrCodeConnectionRefused,
		},
		{
			"网络超时错误",
			errors.New("network timeout occurred"),
			NetworkError,
			ErrCodeNetworkTimeout,
		},
		{
			"数据库查询超时",
			errors.New("database query timeout"),
			DatabaseError,
			ErrCodeQueryTimeout,
		},
		{
			"权限拒绝错误",
			errors.New("permission denied"),
			SystemError,
			ErrCodePermissionDenied,
		},
		{
			"资源未找到错误",
			errors.New("resource not found"),
			BusinessError,
			ErrCodeResourceNotFound,
		},
		{
			"无效输入错误",
			errors.New("invalid input format"),
			ValidationError,
			ErrCodeInvalidInput,
		},
		{
			"认证失败错误",
			errors.New("authentication failed"),
			AuthError,
			ErrCodeInvalidCredentials,
		},
		{
			"配置错误",
			errors.New("config file invalid"),
			ConfigError,
			ErrCodeConfigInvalid,
		},
		{
			"未知错误",
			errors.New("some unknown error"),
			UnknownError,
			"UNKNOWN_001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.ClassifyError(tt.inputError)
			require.NotNil(t, result)
			assert.Equal(t, tt.expectedType, result.Type)
			assert.Equal(t, tt.expectedCode, result.Code)
			assert.Equal(t, tt.inputError, result.Cause)
		})
	}

	t.Run("nil错误", func(t *testing.T) {
		result := classifier.ClassifyError(nil)
		assert.Nil(t, result)
	})

	t.Run("已分类的LoggerError", func(t *testing.T) {
		originalErr := NewNetworkError("NET_001", "网络错误")
		result := classifier.ClassifyError(originalErr)
		assert.Equal(t, originalErr, result)
	})
}

func TestErrorRecovery(t *testing.T) {
	logger := GetGlobalLogger()
	config := DefaultErrorRecoveryConfig()
	config.MaxRetries = 2
	config.RetryInterval = 10 * time.Millisecond
	recovery := NewErrorRecovery(logger, config)

	t.Run("成功执行无需重试", func(t *testing.T) {
		ctx := context.Background()
		executeCount := 0

		err := recovery.ExecuteWithRecovery(ctx, "test_operation", func() error {
			executeCount++
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, executeCount)
	})

	t.Run("可重试错误最终成功", func(t *testing.T) {
		ctx := context.Background()
		executeCount := 0

		err := recovery.ExecuteWithRecovery(ctx, "test_operation", func() error {
			executeCount++
			if executeCount < 3 {
				return NewNetworkError("NET_001", "网络超时").WithRetryable(true)
			}
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 3, executeCount)
	})

	t.Run("不可重试错误立即返回", func(t *testing.T) {
		ctx := context.Background()
		executeCount := 0

		err := recovery.ExecuteWithRecovery(ctx, "test_operation", func() error {
			executeCount++
			return NewValidationError("VAL_001", "验证失败").WithRetryable(false)
		})

		assert.Error(t, err)
		assert.Equal(t, 1, executeCount)
		assert.IsType(t, &LoggerError{}, err)
	})

	t.Run("重试次数达到上限", func(t *testing.T) {
		ctx := context.Background()
		executeCount := 0

		err := recovery.ExecuteWithRecovery(ctx, "test_operation", func() error {
			executeCount++
			return NewNetworkError("NET_001", "持续网络错误").WithRetryable(true)
		})

		assert.Error(t, err)
		assert.Equal(t, 3, executeCount) // 1次初始尝试 + 2次重试
		assert.IsType(t, &LoggerError{}, err)
	})

	t.Run("上下文取消", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		executeCount := 0

		err := recovery.ExecuteWithRecovery(ctx, "test_operation", func() error {
			executeCount++
			if executeCount == 1 {
				// 第一次执行后取消上下文
				go func() {
					time.Sleep(5 * time.Millisecond)
					cancel()
				}()
				return NewNetworkError("NET_001", "网络错误").WithRetryable(true)
			}
			return nil
		})

		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
		assert.Equal(t, 1, executeCount)
	})
}

func TestLoggerLogError(t *testing.T) {
	// 创建临时日志器用于测试
	tempDir := t.TempDir()
	config := Config{
		Level:  DebugLevel,
		Format: JSONFormat,
		Output: OutputConfig{
			Mode:     FileMode,
			Filename: "test.log",
			Path:     tempDir,
		},
	}

	logger, err := New(&config)
	require.NoError(t, err)
	defer logger.Close()

	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-123")
	ctx = WithTraceID(ctx, "trace-456")

	// 测试记录不同类型的错误
	testErr := NewDatabaseError("DB_001", "数据库连接失败").
		WithContext("host", "localhost").
		WithContext("port", 5432)

	logger.LogError(ctx, testErr, "database_connect", String("extra", "info"))

	// 测试记录普通错误
	standardErr := errors.New("standard error")
	logger.LogError(ctx, standardErr, "standard_operation")

	// 测试记录nil错误（应该不记录任何内容）
	logger.LogError(ctx, nil, "nil_operation")

	logger.Sync()
}

func TestUtilityFunctions(t *testing.T) {
	t.Run("IsRetryableError", func(t *testing.T) {
		// 测试LoggerError
		retryableErr := NewNetworkError("NET_001", "网络错误").WithRetryable(true)
		nonRetryableErr := NewValidationError("VAL_001", "验证错误").WithRetryable(false)

		assert.True(t, IsRetryableError(retryableErr))
		assert.False(t, IsRetryableError(nonRetryableErr))

		// 测试标准错误
		timeoutErr := errors.New("connection timeout")
		validationErr := errors.New("invalid format")

		assert.True(t, IsRetryableError(timeoutErr))
		assert.False(t, IsRetryableError(validationErr))

		// 测试nil错误
		assert.False(t, IsRetryableError(nil))
	})

	t.Run("GetErrorSeverity", func(t *testing.T) {
		// 测试LoggerError
		criticalErr := NewConfigError("CFG_001", "配置错误")
		assert.Equal(t, CriticalSeverity, GetErrorSeverity(criticalErr))

		// 测试标准错误
		standardErr := errors.New("standard error")
		assert.Equal(t, MediumSeverity, GetErrorSeverity(standardErr))

		// 测试nil错误
		assert.Equal(t, LowSeverity, GetErrorSeverity(nil))
	})

	t.Run("WrapError", func(t *testing.T) {
		originalErr := errors.New("original error")
		wrappedErr := WrapError(originalErr, NetworkError, "NET_001", "网络错误")

		require.NotNil(t, wrappedErr)
		assert.Equal(t, NetworkError, wrappedErr.Type)
		assert.Equal(t, "NET_001", wrappedErr.Code)
		assert.Equal(t, "网络错误", wrappedErr.Message)
		assert.Equal(t, originalErr, wrappedErr.Cause)
		assert.NotEmpty(t, wrappedErr.StackTrace)

		// 测试nil错误
		nilWrapped := WrapError(nil, SystemError, "SYS_001", "系统错误")
		assert.Nil(t, nilWrapped)
	})
}

func TestStackTrace(t *testing.T) {
	stack := getStackTrace(5)
	assert.NotEmpty(t, stack)
	assert.Contains(t, stack, "TestStackTrace")
	assert.True(t, strings.Count(stack, "\n") <= 5)
}

func TestErrorRecoveryConfig(t *testing.T) {
	config := DefaultErrorRecoveryConfig()
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, time.Second, config.RetryInterval)
	assert.Equal(t, 2.0, config.BackoffFactor)
	assert.Equal(t, 30*time.Second, config.MaxRetryInterval)
	assert.True(t, config.EnableCircuitBreaker)
	assert.Equal(t, 5, config.CircuitBreakerThreshold)
	assert.Equal(t, 60*time.Second, config.CircuitBreakerTimeout)
}