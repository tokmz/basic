package cache

import "errors"

// 预定义错误
var (
	// 通用错误
	ErrKeyNotFound     = errors.New("cache key not found")
	ErrKeyExists       = errors.New("cache key already exists")
	ErrInvalidKey      = errors.New("invalid cache key")
	ErrInvalidValue    = errors.New("invalid cache value")
	ErrInvalidTTL      = errors.New("invalid TTL value")
	ErrCacheClosed     = errors.New("cache is closed")
	ErrOperationFailed = errors.New("cache operation failed")

	// 配置相关错误
	ErrInvalidConfig    = errors.New("invalid cache config")
	ErrInvalidMaxSize   = errors.New("invalid max size")
	ErrInvalidMaxMemory = errors.New("invalid max memory")
	ErrEmptyRedisAddrs  = errors.New("redis addresses cannot be empty")

	// 连接相关错误
	ErrConnectionFailed  = errors.New("failed to connect to cache server")
	ErrConnectionTimeout = errors.New("cache connection timeout")
	ErrConnectionClosed  = errors.New("cache connection closed")

	// 内存缓存相关错误
	ErrMemoryLimitExceeded = errors.New("memory limit exceeded")
	ErrEvictionFailed      = errors.New("cache eviction failed")

	// Redis相关错误
	ErrRedisConnection       = errors.New("redis connection error")
	ErrRedisTimeout          = errors.New("redis operation timeout")
	ErrRedisCluster          = errors.New("redis cluster error")
	ErrSerializationFailed   = errors.New("serialization failed")
	ErrDeserializationFailed = errors.New("deserialization failed")

	// 分布式相关错误
	ErrSyncFailed        = errors.New("cache sync failed")
	ErrConsistencyFailed = errors.New("cache consistency check failed")
	ErrNodeNotFound      = errors.New("cache node not found")
)

// CacheError 缓存错误结构
type CacheError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Cause   error  `json:"cause,omitempty"`
}

// Error 实现error接口
func (e *CacheError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// Unwrap 展开底层错误
func (e *CacheError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// NewCacheError 创建缓存错误
func NewCacheError(code, message string, cause error) *CacheError {
	return &CacheError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// 错误码常量
const (
	ErrCodeKeyNotFound      = "KEY_NOT_FOUND"
	ErrCodeInvalidKey       = "INVALID_KEY"
	ErrCodeInvalidValue     = "INVALID_VALUE"
	ErrCodeConnectionFailed = "CONNECTION_FAILED"
	ErrCodeTimeout          = "TIMEOUT"
	ErrCodeMemoryLimit      = "MEMORY_LIMIT_EXCEEDED"
	ErrCodeSerialization    = "SERIALIZATION_FAILED"
	ErrCodeSyncFailed       = "SYNC_FAILED"
)
