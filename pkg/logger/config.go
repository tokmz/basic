package logger

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogLevel 日志级别类型
// 定义了日志的严重程度级别，级别越高表示越重要
// 日志器只会输出等于或高于设置级别的日志
type LogLevel string

const (
	DebugLevel LogLevel = "debug" // 调试级别：详细的调试信息，通常只在开发环境启用
	InfoLevel  LogLevel = "info"  // 信息级别：一般的信息性消息，表示程序正常运行
	WarnLevel  LogLevel = "warn"  // 警告级别：潜在问题或异常情况，但不影响程序运行
	ErrorLevel LogLevel = "error" // 错误级别：错误信息，程序遇到问题但仍可继续运行
	FatalLevel LogLevel = "fatal" // 致命级别：严重错误，程序无法继续运行
)

// OutputMode 输出模式
// 定义日志的输出目标，支持多种输出方式以适应不同的使用场景
type OutputMode string

const (
	ConsoleMode OutputMode = "console" // 控制台输出：日志输出到标准输出/错误流
	FileMode    OutputMode = "file"    // 文件输出：日志输出到指定文件
	MixedMode   OutputMode = "mixed"   // 混合输出：同时输出到控制台和文件
	MultiMode   OutputMode = "multi"   // 多目标输出：支持多个文件目标
)

// Format 日志格式
// 定义日志的序列化格式，影响日志的可读性和解析方式
type Format string

const (
	JSONFormat Format = "json" // JSON格式：结构化格式，便于程序解析和日志分析
	TextFormat Format = "text" // 文本格式：人类可读格式，便于直接查看
)

// Environment 环境类型
// 定义运行环境，不同环境会应用不同的默认配置策略
type Environment string

const (
	Development Environment = "development" // 开发环境：详细日志，彩色输出，调用者信息
	Testing     Environment = "testing"     // 测试环境：适中的日志级别，便于测试调试
	Production  Environment = "production"  // 生产环境：高效配置，JSON格式，性能优化
)

// RotationConfig 日志轮转配置
// 控制日志文件的轮转策略，防止单个日志文件过大
// 当满足轮转条件时，会创建新的日志文件并可选择性地压缩旧文件
type RotationConfig struct {
	MaxSize    int  `json:"max_size" yaml:"max_size"`       // 单个文件最大大小(MB)，超过此大小会触发轮转
	MaxAge     int  `json:"max_age" yaml:"max_age"`         // 文件保留天数，超过此时间的文件会被删除
	MaxBackups int  `json:"max_backups" yaml:"max_backups"` // 最大备份文件数，超过此数量的旧文件会被删除
	Compress   bool `json:"compress" yaml:"compress"`       // 是否压缩旧的日志文件以节省空间
}

// BufferConfig 缓冲配置
// 控制日志的缓冲行为，可以提高写入性能并减少I/O操作
// 异步缓冲可以显著提升高并发场景下的日志性能
type BufferConfig struct {
	Size        int           `json:"size" yaml:"size"`                 // 缓冲区大小(字节)，影响内存使用和批量写入效率
	FlushTime   time.Duration `json:"flush_time" yaml:"flush_time"`     // 刷新间隔，控制缓冲数据写入磁盘的频率
	AsyncBuffer bool          `json:"async_buffer" yaml:"async_buffer"` // 是否启用异步缓冲，可提升性能但可能影响实时性
}

// OutputConfig 输出配置
// 定义日志的输出目标和文件相关设置
type OutputConfig struct {
	Mode     OutputMode `json:"mode" yaml:"mode"`         // 输出模式：控制台、文件、混合或多目标
	Filename string     `json:"filename" yaml:"filename"` // 日志文件名，仅在文件模式下有效
	Path     string     `json:"path" yaml:"path"`         // 日志文件存储路径，如果不存在会自动创建
}

// Config 日志配置
// 包含日志器的完整配置选项，支持灵活的定制化设置
// 可以通过预设配置函数快速获取适合不同环境的配置
type Config struct {
	// 基础配置
	Level       LogLevel    `json:"level" yaml:"level"`             // 日志级别：控制输出的最低日志级别
	Format      Format      `json:"format" yaml:"format"`           // 输出格式：JSON或文本格式
	Environment Environment `json:"environment" yaml:"environment"` // 运行环境：影响默认配置策略

	// 功能开关
	EnableCaller     bool `json:"enable_caller" yaml:"enable_caller"`         // 启用调用者信息：在日志中包含文件名和行号
	EnableStacktrace bool `json:"enable_stacktrace" yaml:"enable_stacktrace"` // 启用堆栈追踪：在错误日志中包含调用栈
	EnableColor      bool `json:"enable_color" yaml:"enable_color"`           // 启用颜色输出：在控制台输出中使用颜色

	// 输出配置
	Output   OutputConfig   `json:"output" yaml:"output"`     // 输出配置：定义日志输出目标
	Rotation RotationConfig `json:"rotation" yaml:"rotation"` // 轮转配置：控制日志文件轮转策略
	Buffer   BufferConfig   `json:"buffer" yaml:"buffer"`     // 缓冲配置：控制日志缓冲行为

	// 高级配置
	Sampling     *zap.SamplingConfig    `json:"sampling,omitempty" yaml:"sampling,omitempty"`           // 采样配置：在高频日志场景下进行采样以提升性能
	CustomFields map[string]interface{} `json:"custom_fields,omitempty" yaml:"custom_fields,omitempty"` // 自定义字段：在所有日志中包含的全局字段
}

// DefaultConfig 返回默认配置
// 提供一个平衡的配置，适用于大多数应用场景
// 使用JSON格式、信息级别、混合输出模式，并启用基本功能
func DefaultConfig() *Config {
	return &Config{
		Level:            InfoLevel,
		Format:           JSONFormat,
		Environment:      Development,
		EnableCaller:     true,
		EnableStacktrace: true,
		EnableColor:      true,
		Output: OutputConfig{
			Mode:     ConsoleMode,
			Filename: "app.log",
			Path:     "./logs",
		},
		Rotation: RotationConfig{
			MaxSize:    100,
			MaxAge:     30,
			MaxBackups: 10,
			Compress:   true,
		},
		Buffer: BufferConfig{
			Size:        1024,
			FlushTime:   time.Second,
			AsyncBuffer: true,
		},
	}
}

// DevelopmentConfig 返回开发环境配置
// 针对开发环境优化，提供详细的调试信息和人类可读的输出
// 启用调用者信息、堆栈追踪和彩色输出，使用文本格式便于阅读
func DevelopmentConfig() *Config {
	config := DefaultConfig()
	config.Environment = Development
	config.Level = DebugLevel
	config.Format = TextFormat
	config.Output.Mode = ConsoleMode
	config.EnableColor = true
	return config
}

// TestingConfig 返回测试环境配置
// 针对测试环境优化，平衡详细程度和性能
// 使用JSON格式便于程序解析，适中的日志级别，禁用彩色输出
func TestingConfig() *Config {
	config := DefaultConfig()
	config.Environment = Testing
	config.Level = InfoLevel
	config.Format = JSONFormat
	config.Output.Mode = FileMode
	config.EnableColor = false
	return config
}

// ProductionConfig 返回生产环境配置
// 针对生产环境优化，注重性能和存储效率
// 使用JSON格式、较高的日志级别、文件输出、启用压缩和异步缓冲
func ProductionConfig() *Config {
	config := DefaultConfig()
	config.Environment = Production
	config.Level = InfoLevel
	config.Format = JSONFormat
	config.Output.Mode = FileMode
	config.EnableColor = false
	config.EnableStacktrace = false
	config.Buffer.AsyncBuffer = true
	config.Sampling = &zap.SamplingConfig{
		Initial:    100,
		Thereafter: 100,
	}
	return config
}

// Validate 验证配置的有效性
// 检查配置参数是否合理，并为某些参数设置默认值
// 确保配置在创建日志器时不会导致错误
// 返回:
//   - error: 如果配置无效则返回描述性错误，否则返回nil
func (c *Config) Validate() error {
	// 验证必需的基础配置
	if c.Level == "" {
		return fmt.Errorf("log level cannot be empty")
	}

	if c.Format == "" {
		return fmt.Errorf("log format cannot be empty")
	}

	// 验证日志级别的有效性
	validLevels := map[LogLevel]bool{
		DebugLevel: true, InfoLevel: true, WarnLevel: true,
		ErrorLevel: true, FatalLevel: true,
	}
	if !validLevels[c.Level] {
		return fmt.Errorf("invalid log level: %s", c.Level)
	}

	// 验证输出格式的有效性
	validFormats := map[Format]bool{JSONFormat: true, TextFormat: true}
	if !validFormats[c.Format] {
		return fmt.Errorf("invalid log format: %s", c.Format)
	}

	// 验证输出模式的有效性
	validModes := map[OutputMode]bool{
		ConsoleMode: true, FileMode: true, MixedMode: true, MultiMode: true,
	}
	if !validModes[c.Output.Mode] {
		return fmt.Errorf("invalid output mode: %s", c.Output.Mode)
	}

	// 验证文件输出相关配置
	if c.Output.Mode == FileMode || c.Output.Mode == MixedMode {
		if c.Output.Filename == "" {
			return fmt.Errorf("filename cannot be empty when using file output")
		}
		if c.Output.Path == "" {
			return fmt.Errorf("path cannot be empty when using file output")
		}
	}

	// 设置合理的默认值
	if c.Rotation.MaxSize <= 0 {
		c.Rotation.MaxSize = 100 // 默认100MB
	}
	if c.Rotation.MaxAge <= 0 {
		c.Rotation.MaxAge = 30 // 默认30天
	}
	if c.Rotation.MaxBackups <= 0 {
		c.Rotation.MaxBackups = 10 // 默认10个备份
	}

	if c.Buffer.Size <= 0 {
		c.Buffer.Size = 8192 // 默认8KB
	}
	if c.Buffer.FlushTime <= 0 {
		c.Buffer.FlushTime = 100 * time.Millisecond // 默认100ms
	}

	return nil
}

// ToZapLevel 转换为Zap日志级别
func (l LogLevel) ToZapLevel() zapcore.Level {
	switch l {
	case DebugLevel:
		return zapcore.DebugLevel
	case InfoLevel:
		return zapcore.InfoLevel
	case WarnLevel:
		return zapcore.WarnLevel
	case ErrorLevel:
		return zapcore.ErrorLevel
	case FatalLevel:
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// FromZapLevel 从Zap日志级别转换
func FromZapLevel(level zapcore.Level) LogLevel {
	switch level {
	case zapcore.DebugLevel:
		return DebugLevel
	case zapcore.InfoLevel:
		return InfoLevel
	case zapcore.WarnLevel:
		return WarnLevel
	case zapcore.ErrorLevel:
		return ErrorLevel
	case zapcore.FatalLevel:
		return FatalLevel
	default:
		return InfoLevel
	}
}
