package cache

import (
	"errors"
	"testing"
)

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		// 通用错误
		{"ErrKeyNotFound", ErrKeyNotFound, "cache key not found"},
		{"ErrKeyExists", ErrKeyExists, "cache key already exists"},
		{"ErrInvalidKey", ErrInvalidKey, "invalid cache key"},
		{"ErrInvalidValue", ErrInvalidValue, "invalid cache value"},
		{"ErrInvalidTTL", ErrInvalidTTL, "invalid TTL value"},
		{"ErrCacheClosed", ErrCacheClosed, "cache is closed"},
		{"ErrOperationFailed", ErrOperationFailed, "cache operation failed"},

		// 配置相关错误
		{"ErrInvalidConfig", ErrInvalidConfig, "invalid cache config"},
		{"ErrInvalidMaxSize", ErrInvalidMaxSize, "invalid max size"},
		{"ErrInvalidMaxMemory", ErrInvalidMaxMemory, "invalid max memory"},
		{"ErrEmptyRedisAddrs", ErrEmptyRedisAddrs, "redis addresses cannot be empty"},

		// 连接相关错误
		{"ErrConnectionFailed", ErrConnectionFailed, "failed to connect to cache server"},
		{"ErrConnectionTimeout", ErrConnectionTimeout, "cache connection timeout"},
		{"ErrConnectionClosed", ErrConnectionClosed, "cache connection closed"},

		// 内存缓存相关错误
		{"ErrMemoryLimitExceeded", ErrMemoryLimitExceeded, "memory limit exceeded"},
		{"ErrEvictionFailed", ErrEvictionFailed, "cache eviction failed"},

		// Redis相关错误
		{"ErrRedisConnection", ErrRedisConnection, "redis connection error"},
		{"ErrRedisTimeout", ErrRedisTimeout, "redis operation timeout"},
		{"ErrRedisCluster", ErrRedisCluster, "redis cluster error"},
		{"ErrSerializationFailed", ErrSerializationFailed, "serialization failed"},
		{"ErrDeserializationFailed", ErrDeserializationFailed, "deserialization failed"},

		// 分布式相关错误
		{"ErrSyncFailed", ErrSyncFailed, "cache sync failed"},
		{"ErrConsistencyFailed", ErrConsistencyFailed, "cache consistency check failed"},
		{"ErrNodeNotFound", ErrNodeNotFound, "cache node not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("Expected error message '%s', got '%s'", tt.expected, tt.err.Error())
			}
		})
	}
}

func TestCacheError_Error(t *testing.T) {
	tests := []struct {
		name     string
		cacheErr *CacheError
		expected string
	}{
		{
			name: "Error without cause",
			cacheErr: &CacheError{
				Code:    "TEST_ERROR",
				Message: "test error message",
				Cause:   nil,
			},
			expected: "test error message",
		},
		{
			name: "Error with cause",
			cacheErr: &CacheError{
				Code:    "TEST_ERROR",
				Message: "test error message",
				Cause:   errors.New("underlying error"),
			},
			expected: "test error message: underlying error",
		},
		{
			name: "Empty message without cause",
			cacheErr: &CacheError{
				Code:    "TEST_ERROR",
				Message: "",
				Cause:   nil,
			},
			expected: "",
		},
		{
			name: "Empty message with cause",
			cacheErr: &CacheError{
				Code:    "TEST_ERROR",
				Message: "",
				Cause:   errors.New("underlying error"),
			},
			expected: ": underlying error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cacheErr.Error()
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestCacheError_Unwrap(t *testing.T) {
	tests := []struct {
		name     string
		cacheErr *CacheError
		expected error
	}{
		{
			name: "Unwrap with cause",
			cacheErr: &CacheError{
				Code:    "TEST_ERROR",
				Message: "test error",
				Cause:   ErrKeyNotFound,
			},
			expected: ErrKeyNotFound,
		},
		{
			name: "Unwrap without cause",
			cacheErr: &CacheError{
				Code:    "TEST_ERROR",
				Message: "test error",
				Cause:   nil,
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cacheErr.Unwrap()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestNewCacheError(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		message  string
		cause    error
		expected *CacheError
	}{
		{
			name:    "Create error without cause",
			code:    "TEST_CODE",
			message: "test message",
			cause:   nil,
			expected: &CacheError{
				Code:    "TEST_CODE",
				Message: "test message",
				Cause:   nil,
			},
		},
		{
			name:    "Create error with cause",
			code:    "TEST_CODE",
			message: "test message",
			cause:   ErrKeyNotFound,
			expected: &CacheError{
				Code:    "TEST_CODE",
				Message: "test message",
				Cause:   ErrKeyNotFound,
			},
		},
		{
			name:    "Create error with empty code and message",
			code:    "",
			message: "",
			cause:   nil,
			expected: &CacheError{
				Code:    "",
				Message: "",
				Cause:   nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewCacheError(tt.code, tt.message, tt.cause)

			if result.Code != tt.expected.Code {
				t.Errorf("Expected Code '%s', got '%s'", tt.expected.Code, result.Code)
			}

			if result.Message != tt.expected.Message {
				t.Errorf("Expected Message '%s', got '%s'", tt.expected.Message, result.Message)
			}

			if result.Cause != tt.expected.Cause {
				t.Errorf("Expected Cause %v, got %v", tt.expected.Cause, result.Cause)
			}
		})
	}
}

func TestErrorConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"ErrCodeKeyNotFound", ErrCodeKeyNotFound, "KEY_NOT_FOUND"},
		{"ErrCodeInvalidKey", ErrCodeInvalidKey, "INVALID_KEY"},
		{"ErrCodeInvalidValue", ErrCodeInvalidValue, "INVALID_VALUE"},
		{"ErrCodeConnectionFailed", ErrCodeConnectionFailed, "CONNECTION_FAILED"},
		{"ErrCodeTimeout", ErrCodeTimeout, "TIMEOUT"},
		{"ErrCodeMemoryLimit", ErrCodeMemoryLimit, "MEMORY_LIMIT_EXCEEDED"},
		{"ErrCodeSerialization", ErrCodeSerialization, "SERIALIZATION_FAILED"},
		{"ErrCodeSyncFailed", ErrCodeSyncFailed, "SYNC_FAILED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Expected constant '%s', got '%s'", tt.expected, tt.constant)
			}
		})
	}
}

func TestCacheError_ImplementsError(t *testing.T) {
	// 验证CacheError实现了error接口
	var _ error = &CacheError{}

	cacheErr := NewCacheError("TEST", "test error", nil)
	var err error = cacheErr

	if err.Error() != "test error" {
		t.Errorf("CacheError should implement error interface correctly")
	}
}

func TestErrorWrapping(t *testing.T) {
	// 测试error wrapping功能
	originalErr := errors.New("original error")
	cacheErr := NewCacheError("WRAP_TEST", "wrapped error", originalErr)

	// 使用errors.Is测试
	if !errors.Is(cacheErr, originalErr) {
		t.Error("errors.Is should work with wrapped CacheError")
	}

	// 使用errors.Unwrap测试
	unwrapped := errors.Unwrap(cacheErr)
	if unwrapped != originalErr {
		t.Errorf("Expected unwrapped error to be %v, got %v", originalErr, unwrapped)
	}
}

func TestCacheError_ChainedWrapping(t *testing.T) {
	// 测试链式包装
	baseErr := errors.New("base error")
	firstWrap := NewCacheError("FIRST", "first wrap", baseErr)
	secondWrap := NewCacheError("SECOND", "second wrap", firstWrap)

	// 验证可以通过链式unwrap找到原始错误
	if !errors.Is(secondWrap, baseErr) {
		t.Error("Should be able to find base error through chain")
	}

	if !errors.Is(secondWrap, firstWrap) {
		t.Error("Should be able to find first wrapped error")
	}
}

func TestCacheError_ErrorString(t *testing.T) {
	// 测试复杂的错误字符串构建
	baseErr := errors.New("connection refused")
	cacheErr := NewCacheError("CONNECTION_ERROR", "Failed to connect to Redis", baseErr)

	expected := "Failed to connect to Redis: connection refused"
	if cacheErr.Error() != expected {
		t.Errorf("Expected error string '%s', got '%s'", expected, cacheErr.Error())
	}
}

func TestCacheError_NilSafety(t *testing.T) {
	// 测试nil安全性
	var cacheErr *CacheError

	// 这不应该panic
	defer func() {
		if r := recover(); r != nil {
			t.Error("CacheError methods should be nil-safe")
		}
	}()

	// 注意：nil的Error()调用会panic，这是Go的标准行为
	// 我们只测试Unwrap的nil安全性
	if cacheErr.Unwrap() != nil {
		t.Error("Nil CacheError Unwrap should return nil")
	}
}

// 性能测试
func BenchmarkCacheError_Error(b *testing.B) {
	cacheErr := NewCacheError("BENCH_TEST", "benchmark error message", ErrKeyNotFound)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cacheErr.Error()
	}
}

func BenchmarkCacheError_Unwrap(b *testing.B) {
	cacheErr := NewCacheError("BENCH_TEST", "benchmark error", ErrKeyNotFound)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cacheErr.Unwrap()
	}
}

func BenchmarkNewCacheError(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewCacheError("BENCH_CODE", "benchmark message", ErrKeyNotFound)
	}
}
