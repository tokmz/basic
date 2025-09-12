package database

import (
	"time"

	"go.uber.org/zap"
)

// DatabaseConfig 表示完整的数据库配置
type DatabaseConfig struct {
	// 主数据库配置
	Master MasterConfig `json:"master" yaml:"master"`

	// 从数据库配置，用于读操作
	Slaves []SlaveConfig `json:"slaves" yaml:"slaves"`

	// 连接池设置
	Pool PoolConfig `json:"pool" yaml:"pool"`

	// 日志配置
	Logger LoggerConfig `json:"logger" yaml:"logger"`

	// 慢查询监控配置
	SlowQuery SlowQueryConfig `json:"slow_query" yaml:"slow_query"`

	// 健康检查配置
	HealthCheck HealthCheckConfig `json:"health_check" yaml:"health_check"`
}

// MasterConfig 表示主数据库配置
type MasterConfig struct {
	// 数据库驱动 (mysql, postgres 等)
	Driver string `json:"driver" yaml:"driver"`

	// 数据库连接 DSN
	DSN string `json:"dsn" yaml:"dsn"`
}

// SlaveConfig 表示从数据库配置
type SlaveConfig struct {
	// 数据库驱动 (mysql, postgres 等)
	Driver string `json:"driver" yaml:"driver"`

	// 数据库连接 DSN
	DSN string `json:"dsn" yaml:"dsn"`

	// 负载均衡权重 (数值越高 = 流量越多)
	Weight int `json:"weight" yaml:"weight"`
}

// PoolConfig 表示连接池配置
type PoolConfig struct {
	// 最大打开连接数
	MaxOpenConns int `json:"max_open_conns" yaml:"max_open_conns"`

	// 最大空闲连接数
	MaxIdleConns int `json:"max_idle_conns" yaml:"max_idle_conns"`

	// 连接的最大生存时间
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime" yaml:"conn_max_lifetime"`

	// 连接的最大空闲时间
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time" yaml:"conn_max_idle_time"`
}

// LoggerConfig 表示日志配置
type LoggerConfig struct {
	// 外部 zap 日志实例 (可选)
	ZapLogger *zap.Logger `json:"-" yaml:"-"`

	// 日志级别 (silent, error, warn, info)
	Level LogLevel `json:"level" yaml:"level"`

	// 启用彩色日志
	Colorful bool `json:"colorful" yaml:"colorful"`

	// 慢查询日志阈值
	SlowThreshold time.Duration `json:"slow_threshold" yaml:"slow_threshold"`
}

// LogLevel 表示日志级别
type LogLevel int

const (
	Silent LogLevel = iota + 1
	Error
	Warn
	Info
)

// String 返回 LogLevel 的字符串表示
func (l LogLevel) String() string {
	switch l {
	case Silent:
		return "silent"
	case Error:
		return "error"
	case Warn:
		return "warn"
	case Info:
		return "info"
	default:
		return "unknown"
	}
}

// SlowQueryConfig 表示慢查询监控配置
type SlowQueryConfig struct {
	// 启用慢查询监控
	Enabled bool `json:"enabled" yaml:"enabled"`

	// 判断查询为慢查询的阈值
	Threshold time.Duration `json:"threshold" yaml:"threshold"`

	// 慢查询自定义日志器 (可选)
	Logger *zap.Logger `json:"-" yaml:"-"`
}

// HealthCheckConfig 表示健康检查配置
type HealthCheckConfig struct {
	// 启用健康检查
	Enabled bool `json:"enabled" yaml:"enabled"`

	// 健康检查间隔
	Interval time.Duration `json:"interval" yaml:"interval"`

	// 健康检查操作超时时间
	Timeout time.Duration `json:"timeout" yaml:"timeout"`

	// 标记为不健康前的最大连续失败次数
	MaxFailures int `json:"max_failures" yaml:"max_failures"`

	// 健康检查事件自定义日志器 (可选)
	Logger *zap.Logger `json:"-" yaml:"-"`
}

// DefaultConfig 返回默认的数据库配置
func DefaultConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Pool: PoolConfig{
			MaxOpenConns:    100,
			MaxIdleConns:    10,
			ConnMaxLifetime: time.Hour,
			ConnMaxIdleTime: time.Minute * 30,
		},
		Logger: LoggerConfig{
			Level:         Info,
			Colorful:      true,
			SlowThreshold: time.Millisecond * 200,
		},
		SlowQuery: SlowQueryConfig{
			Enabled:   true,
			Threshold: time.Millisecond * 200,
		},
		HealthCheck: HealthCheckConfig{
			Enabled:     true,
			Interval:    time.Minute * 5,
			Timeout:     time.Second * 30,
			MaxFailures: 3,
		},
	}
}
