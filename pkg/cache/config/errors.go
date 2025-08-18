package config

import "errors"

// 配置相关错误定义
var (
	// ErrInvalidAddr 无效的Redis地址
	ErrInvalidAddr = errors.New("invalid redis address")

	// ErrInvalidPoolSize 无效的连接池大小
	ErrInvalidPoolSize = errors.New("invalid pool size, must be greater than 0")

	// ErrInvalidMinIdleConns 无效的最小空闲连接数
	ErrInvalidMinIdleConns = errors.New("invalid min idle connections, must be greater than or equal to 0")

	// ErrInvalidMaxIdleConns 无效的最大空闲连接数
	ErrInvalidMaxIdleConns = errors.New("invalid max idle connections, must be greater than or equal to min idle connections")

	// ErrConnectionFailed Redis连接失败
	ErrConnectionFailed = errors.New("failed to connect to redis")

	// ErrConfigValidation 配置验证失败
	ErrConfigValidation = errors.New("configuration validation failed")

	// ErrInvalidExpiration 无效的过期时间
	ErrInvalidExpiration = errors.New("invalid expiration time")

	// ErrInvalidTimeout 无效的超时时间
	ErrInvalidTimeout = errors.New("invalid timeout configuration")
)
