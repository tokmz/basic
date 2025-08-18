package config

import (
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisConfig Redis配置结构
type RedisConfig struct {
	// 基础连接配置
	Addr     string `json:"addr" yaml:"addr"`         // Redis地址
	Password string `json:"password" yaml:"password"` // 密码
	DB       int    `json:"db" yaml:"db"`             // 数据库编号

	// 连接池配置
	PoolSize        int           `json:"pool_size" yaml:"pool_size"`                   // 连接池大小
	MinIdleConns    int           `json:"min_idle_conns" yaml:"min_idle_conns"`         // 最小空闲连接数
	MaxIdleConns    int           `json:"max_idle_conns" yaml:"max_idle_conns"`         // 最大空闲连接数
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time" yaml:"conn_max_idle_time"` // 连接最大空闲时间
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime" yaml:"conn_max_lifetime"`   // 连接最大生存时间

	// 超时配置
	DialTimeout  time.Duration `json:"dial_timeout" yaml:"dial_timeout"`   // 连接超时
	ReadTimeout  time.Duration `json:"read_timeout" yaml:"read_timeout"`   // 读取超时
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout"` // 写入超时

	// 重试配置
	MaxRetries      int           `json:"max_retries" yaml:"max_retries"`             // 最大重试次数
	MinRetryBackoff time.Duration `json:"min_retry_backoff" yaml:"min_retry_backoff"` // 最小重试间隔
	MaxRetryBackoff time.Duration `json:"max_retry_backoff" yaml:"max_retry_backoff"` // 最大重试间隔

	// 默认配置
	DefaultExpiration time.Duration `json:"default_expiration" yaml:"default_expiration"` // 默认过期时间

	// 链路追踪配置
	Tracing TracingConfig `json:"tracing" yaml:"tracing"`

	// 监控配置
	Monitoring MonitoringConfig `json:"monitoring" yaml:"monitoring"`
}

// TracingConfig 链路追踪配置
type TracingConfig struct {
	Enabled     bool   `json:"enabled" yaml:"enabled"`           // 是否启用追踪
	ServiceName string `json:"service_name" yaml:"service_name"` // 服务名称
	Version     string `json:"version" yaml:"version"`           // 版本号
}

// MonitoringConfig 监控配置
type MonitoringConfig struct {
	Enabled          bool          `json:"enabled" yaml:"enabled"`                       // 是否启用监控
	MetricsNamespace string        `json:"metrics_namespace" yaml:"metrics_namespace"`   // 指标命名空间
	SlowLogThreshold time.Duration `json:"slow_log_threshold" yaml:"slow_log_threshold"` // 慢查询阈值
	CollectInterval  time.Duration `json:"collect_interval" yaml:"collect_interval"`     // 收集间隔
}

// DefaultConfig 返回默认配置
func DefaultConfig() *RedisConfig {
	return &RedisConfig{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,

		// 连接池配置
		PoolSize:        10,
		MinIdleConns:    2,
		MaxIdleConns:    5,
		ConnMaxIdleTime: 30 * time.Minute,
		ConnMaxLifetime: time.Hour,

		// 超时配置
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,

		// 重试配置
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,

		// 默认过期时间
		DefaultExpiration: 24 * time.Hour,

		// 链路追踪配置
		Tracing: TracingConfig{
			Enabled:     true,
			ServiceName: "redis-cache",
			Version:     "1.0.0",
		},

		// 监控配置
		Monitoring: MonitoringConfig{
			Enabled:          true,
			MetricsNamespace: "redis_cache",
			SlowLogThreshold: 100 * time.Millisecond,
			CollectInterval:  10 * time.Second,
		},
	}
}

// ToRedisOptions 转换为 go-redis 配置
func (c *RedisConfig) ToRedisOptions() *redis.Options {
	return &redis.Options{
		Addr:     c.Addr,
		Password: c.Password,
		DB:       c.DB,

		// 连接池配置
		PoolSize:        c.PoolSize,
		MinIdleConns:    c.MinIdleConns,
		MaxIdleConns:    c.MaxIdleConns,
		ConnMaxIdleTime: c.ConnMaxIdleTime,
		ConnMaxLifetime: c.ConnMaxLifetime,

		// 超时配置
		DialTimeout:  c.DialTimeout,
		ReadTimeout:  c.ReadTimeout,
		WriteTimeout: c.WriteTimeout,

		// 重试配置
		MaxRetries:      c.MaxRetries,
		MinRetryBackoff: c.MinRetryBackoff,
		MaxRetryBackoff: c.MaxRetryBackoff,
	}
}

// Validate 验证配置
func (c *RedisConfig) Validate() error {
	if c.Addr == "" {
		return ErrInvalidAddr
	}
	if c.PoolSize <= 0 {
		return ErrInvalidPoolSize
	}
	if c.MinIdleConns < 0 {
		return ErrInvalidMinIdleConns
	}
	if c.MaxIdleConns < c.MinIdleConns {
		return ErrInvalidMaxIdleConns
	}
	return nil
}

// UpdatePoolConfig 动态更新连接池配置
func (c *RedisConfig) UpdatePoolConfig(poolSize, minIdle, maxIdle int) {
	if poolSize > 0 {
		c.PoolSize = poolSize
	}
	if minIdle >= 0 {
		c.MinIdleConns = minIdle
	}
	if maxIdle >= minIdle {
		c.MaxIdleConns = maxIdle
	}
}

// UpdateTimeouts 动态更新超时配置
func (c *RedisConfig) UpdateTimeouts(dial, read, write time.Duration) {
	if dial > 0 {
		c.DialTimeout = dial
	}
	if read > 0 {
		c.ReadTimeout = read
	}
	if write > 0 {
		c.WriteTimeout = write
	}
}
