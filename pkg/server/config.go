package server

import (
	"fmt"
	"time"
)

// Config 服务器配置结构体
type Config struct {
	// 基础配置
	Name string `json:"name" yaml:"name" mapstructure:"name"` // 服务名称
	Port int    `json:"port" yaml:"port" mapstructure:"port"` // 监听端口
	Host string `json:"host" yaml:"host" mapstructure:"host"` // 监听地址

	// 超时配置
	ReadTimeout  time.Duration `json:"read_timeout" yaml:"read_timeout" mapstructure:"read_timeout"`   // 读取超时
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout" mapstructure:"write_timeout"` // 写入超时
	IdleTimeout  time.Duration `json:"idle_timeout" yaml:"idle_timeout" mapstructure:"idle_timeout"`   // 空闲超时

	// 功能开关
	EnableGracefulShutdown bool `json:"enable_graceful_shutdown" yaml:"enable_graceful_shutdown" mapstructure:"enable_graceful_shutdown"` // 启用优雅关闭
	EnableHotRestart       bool `json:"enable_hot_restart" yaml:"enable_hot_restart" mapstructure:"enable_hot_restart"`             // 启用热重启
	EnablePprof            bool `json:"enable_pprof" yaml:"enable_pprof" mapstructure:"enable_pprof"`                               // 启用性能分析
	EnableMetrics          bool `json:"enable_metrics" yaml:"enable_metrics" mapstructure:"enable_metrics"`                         // 启用监控指标
	EnableTracing          bool `json:"enable_tracing" yaml:"enable_tracing" mapstructure:"enable_tracing"`                         // 启用链路追踪

	// 静态文件配置
	StaticDir         string `json:"static_dir" yaml:"static_dir" mapstructure:"static_dir"`                   // 静态文件目录
	StaticURLPrefix   string `json:"static_url_prefix" yaml:"static_url_prefix" mapstructure:"static_url_prefix"` // 静态文件URL前缀
	EnableDirectoryList bool `json:"enable_directory_list" yaml:"enable_directory_list" mapstructure:"enable_directory_list"` // 启用目录列表
	EnableCompression   bool `json:"enable_compression" yaml:"enable_compression" mapstructure:"enable_compression"`       // 启用压缩

	// 安全配置
	EnableCORS bool     `json:"enable_cors" yaml:"enable_cors" mapstructure:"enable_cors"` // 启用CORS
	CORSOrigins []string `json:"cors_origins" yaml:"cors_origins" mapstructure:"cors_origins"` // CORS允许的源

	// 限流配置
	EnableRateLimit bool    `json:"enable_rate_limit" yaml:"enable_rate_limit" mapstructure:"enable_rate_limit"` // 启用限流
	RateLimit       float64 `json:"rate_limit" yaml:"rate_limit" mapstructure:"rate_limit"`                   // 限流速率(请求/秒)
	RateBurst       int     `json:"rate_burst" yaml:"rate_burst" mapstructure:"rate_burst"`                   // 限流突发数

	// 优雅关闭配置
	ShutdownTimeout time.Duration `json:"shutdown_timeout" yaml:"shutdown_timeout" mapstructure:"shutdown_timeout"` // 关闭超时时间

	// 热重启配置
	RestartSignal string `json:"restart_signal" yaml:"restart_signal" mapstructure:"restart_signal"` // 重启信号
	PidFile       string `json:"pid_file" yaml:"pid_file" mapstructure:"pid_file"`                   // PID文件路径
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Name: "gin-web-server",
		Port: 8080,
		Host: "0.0.0.0",

		// 默认超时配置
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,

		// 默认功能开关
		EnableGracefulShutdown: true,
		EnableHotRestart:       false,
		EnablePprof:            false,
		EnableMetrics:          true,
		EnableTracing:          false,

		// 默认静态文件配置
		StaticDir:           "./static",
		StaticURLPrefix:     "/static",
		EnableDirectoryList: false,
		EnableCompression:   true,

		// 默认安全配置
		EnableCORS:  true,
		CORSOrigins: []string{"*"},

		// 默认限流配置
		EnableRateLimit: false,
		RateLimit:       100.0,
		RateBurst:       200,

		// 默认优雅关闭配置
		ShutdownTimeout: 30 * time.Second,

		// 默认热重启配置
		RestartSignal: "SIGUSR2",
		PidFile:       "/tmp/gin-server.pid",
	}
}

// Validate 验证配置的有效性
func (c *Config) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return ErrInvalidPort
	}

	if c.ReadTimeout <= 0 {
		return ErrInvalidTimeout
	}

	if c.WriteTimeout <= 0 {
		return ErrInvalidTimeout
	}

	if c.ShutdownTimeout <= 0 {
		return ErrInvalidTimeout
	}

	if c.EnableRateLimit && c.RateLimit <= 0 {
		return ErrInvalidRateLimit
	}

	return nil
}

// GetAddr 获取监听地址
func (c *Config) GetAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}