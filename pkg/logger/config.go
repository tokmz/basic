package logger

import (
	"time"
)

// LogLevel 日志级别
type LogLevel string

const (
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
	PanicLevel LogLevel = "panic"
	FatalLevel LogLevel = "fatal"
)

// OutputFormat 输出格式
type OutputFormat string

const (
	JSONFormat    OutputFormat = "json"
	ConsoleFormat OutputFormat = "console"
)

// RotationConfig 日志轮转配置
type RotationConfig struct {
	MaxSize    int  `yaml:"max_size" json:"max_size"`       // 单个日志文件最大大小(MB)
	MaxAge     int  `yaml:"max_age" json:"max_age"`         // 保留旧日志文件的最大天数
	MaxBackups int  `yaml:"max_backups" json:"max_backups"` // 保留旧日志文件的最大个数
	Compress   bool `yaml:"compress" json:"compress"`       // 是否压缩旧日志文件
}

// FileConfig 文件输出配置
type FileConfig struct {
	Enabled  bool           `yaml:"enabled" json:"enabled"`   // 是否启用文件输出
	Filename string         `yaml:"filename" json:"filename"` // 日志文件路径
	Rotation RotationConfig `yaml:"rotation" json:"rotation"` // 日志轮转配置
}

// ConsoleConfig 控制台输出配置
type ConsoleConfig struct {
	Enabled    bool `yaml:"enabled" json:"enabled"`         // 是否启用控制台输出
	ColoredOut bool `yaml:"colored_out" json:"colored_out"` // 是否启用彩色输出
}

// Config 日志配置
type Config struct {
	Level         LogLevel      `yaml:"level" json:"level"`                   // 日志级别
	Format        OutputFormat  `yaml:"format" json:"format"`                 // 输出格式
	EnableCaller  bool          `yaml:"enable_caller" json:"enable_caller"`   // 是否显示调用者信息
	EnableStack   bool          `yaml:"enable_stack" json:"enable_stack"`     // 是否显示堆栈信息(Error级别以上)
	TimeFormat    string        `yaml:"time_format" json:"time_format"`       // 时间格式
	Console       ConsoleConfig `yaml:"console" json:"console"`               // 控制台配置
	File          FileConfig    `yaml:"file" json:"file"`                     // 文件配置
	ServiceName   string        `yaml:"service_name" json:"service_name"`     // 服务名称
	ServiceEnv    string        `yaml:"service_env" json:"service_env"`       // 服务环境
	EnableSampler bool          `yaml:"enable_sampler" json:"enable_sampler"` // 是否启用采样
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Level:        InfoLevel,
		Format:       JSONFormat,
		EnableCaller: true,
		EnableStack:  true,
		TimeFormat:   time.RFC3339,
		Console: ConsoleConfig{
			Enabled:    true,
			ColoredOut: true,
		},
		File: FileConfig{
			Enabled:  false,
			Filename: "./logs/app.log",
			Rotation: RotationConfig{
				MaxSize:    100, // 100MB
				MaxAge:     30,  // 30天
				MaxBackups: 10,  // 10个备份文件
				Compress:   true,
			},
		},
		ServiceName:   "unknown",
		ServiceEnv:    "development",
		EnableSampler: false,
	}
}

// DevelopmentConfig 返回开发环境配置
func DevelopmentConfig() *Config {
	config := DefaultConfig()
	config.Level = DebugLevel
	config.Format = ConsoleFormat
	config.Console.ColoredOut = true
	config.File.Enabled = false
	return config
}

// ProductionConfig 返回生产环境配置
func ProductionConfig(serviceName, serviceEnv string) *Config {
	config := DefaultConfig()
	config.Level = InfoLevel
	config.Format = JSONFormat
	config.Console.Enabled = false
	config.File.Enabled = true
	config.File.Filename = "./logs/" + serviceName + ".log"
	config.ServiceName = serviceName
	config.ServiceEnv = serviceEnv
	config.EnableSampler = true
	return config
}

// Validate 验证配置有效性
func (c *Config) Validate() error {
	if c.ServiceName == "" {
		c.ServiceName = "unknown"
	}
	
	if c.ServiceEnv == "" {
		c.ServiceEnv = "development"
	}
	
	if c.TimeFormat == "" {
		c.TimeFormat = time.RFC3339
	}
	
	if c.File.Enabled && c.File.Filename == "" {
		c.File.Filename = "./logs/app.log"
	}
	
	if c.File.Rotation.MaxSize <= 0 {
		c.File.Rotation.MaxSize = 100
	}
	
	if c.File.Rotation.MaxAge <= 0 {
		c.File.Rotation.MaxAge = 30
	}
	
	if c.File.Rotation.MaxBackups < 0 {
		c.File.Rotation.MaxBackups = 0
	}
	
	return nil
}