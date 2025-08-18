package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger 高性能日志器
// 提供结构化日志记录功能，支持多种输出模式、异步缓冲、日志轮转等特性
// 线程安全，支持并发使用
type Logger struct {
	zapLogger *zap.Logger      // 底层zap日志器
	config    *Config          // 日志配置
	level     *zap.AtomicLevel // 原子级别控制，支持动态调整
	buffer    *AsyncBuffer     // 异步缓冲器，可选
	mu        sync.RWMutex     // 读写锁，保护并发访问
	closed    int32            // 关闭状态标志，使用原子操作
}

// Field 日志字段
type Field = zap.Field

// 预定义字段构造函数，避免运行时分配
var (
	String   = zap.String
	Int      = zap.Int
	Int64    = zap.Int64
	Float64  = zap.Float64
	Bool     = zap.Bool
	Time     = zap.Time
	Duration = zap.Duration
	Error    = zap.Error
	Any      = zap.Any
	Binary   = zap.Binary
	Reflect  = zap.Reflect
)

// New 创建新的日志器实例
// 根据提供的配置创建一个完全配置的日志器
// 如果config为nil，将使用默认配置
// 返回的日志器是线程安全的，可以在多个goroutine中并发使用
func New(config *Config) (*Logger, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	level := zap.NewAtomicLevelAt(config.Level.ToZapLevel())

	// 创建核心编码器
	encoder, err := createEncoder(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create encoder: %w", err)
	}

	// 创建写入器
	writeSyncer, err := createWriteSyncer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create write syncer: %w", err)
	}

	// 创建核心
	core := zapcore.NewCore(encoder, writeSyncer, level)

	// 创建选项
	opts := createOptions(config)

	// 创建zap日志器
	zapLogger := zap.New(core, opts...)

	logger := &Logger{
		zapLogger: zapLogger,
		config:    config,
		level:     &level,
	}

	// 初始化异步缓冲
	if config.Buffer.AsyncBuffer {
		buffer, err := NewAsyncBuffer(config.Buffer, writeSyncer)
		if err != nil {
			return nil, fmt.Errorf("failed to create async buffer: %w", err)
		}
		logger.buffer = buffer
	}

	return logger, nil
}

// createEncoder 创建编码器
// 根据配置的格式类型创建相应的编码器（JSON或文本格式）
// 编码器负责将日志条目序列化为指定格式
func createEncoder(config *Config) (zapcore.Encoder, error) {
	encoderConfig := createEncoderConfig(config)

	switch config.Format {
	case JSONFormat:
		return zapcore.NewJSONEncoder(encoderConfig), nil
	case TextFormat:
		return zapcore.NewConsoleEncoder(encoderConfig), nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", config.Format)
	}
}

// createEncoderConfig 创建编码器配置
func createEncoderConfig(config *Config) zapcore.EncoderConfig {
	encoderConfig := zap.NewProductionEncoderConfig()

	if config.Environment == Development {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	}

	// 时间格式
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// 级别格式
	encoderConfig.LevelKey = "level"
	if config.EnableColor && config.Output.Mode == ConsoleMode {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	}

	// 调用者信息
	if config.EnableCaller {
		encoderConfig.CallerKey = "caller"
		encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	} else {
		encoderConfig.CallerKey = zapcore.OmitKey
	}

	// 堆栈追踪
	if config.EnableStacktrace {
		encoderConfig.StacktraceKey = "stacktrace"
	} else {
		encoderConfig.StacktraceKey = zapcore.OmitKey
	}

	return encoderConfig
}

// createWriteSyncer 创建写入器
// 根据输出模式创建相应的写入器（控制台、文件、混合或多目标）
// 写入器负责将格式化后的日志数据写入到指定的目标
func createWriteSyncer(config *Config) (zapcore.WriteSyncer, error) {
	switch config.Output.Mode {
	case ConsoleMode:
		return zapcore.AddSync(os.Stdout), nil
	case FileMode:
		return createFileWriteSyncer(config)
	case MixedMode:
		return createMixedWriteSyncer(config)
	case MultiMode:
		return createMultiWriteSyncer(config)
	default:
		return nil, fmt.Errorf("unsupported output mode: %s", config.Output.Mode)
	}
}

// createFileWriteSyncer 创建文件写入同步器
func createFileWriteSyncer(config *Config) (zapcore.WriteSyncer, error) {
	// 确保目录存在
	if err := os.MkdirAll(config.Output.Path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	filePath := filepath.Join(config.Output.Path, config.Output.Filename)

	// 使用lumberjack进行日志轮转
	lumberjackLogger := &lumberjack.Logger{
		Filename:   filePath,
		MaxSize:    config.Rotation.MaxSize,
		MaxAge:     config.Rotation.MaxAge,
		MaxBackups: config.Rotation.MaxBackups,
		Compress:   config.Rotation.Compress,
	}

	return zapcore.AddSync(lumberjackLogger), nil
}

// createMixedWriteSyncer 创建混合写入同步器
func createMixedWriteSyncer(config *Config) (zapcore.WriteSyncer, error) {
	fileWriter, err := createFileWriteSyncer(config)
	if err != nil {
		return nil, err
	}

	consoleWriter := zapcore.AddSync(os.Stdout)
	return zapcore.NewMultiWriteSyncer(consoleWriter, fileWriter), nil
}

// createMultiWriteSyncer 创建多目标写入同步器
func createMultiWriteSyncer(config *Config) (zapcore.WriteSyncer, error) {
	// 这里可以扩展支持多个文件、网络等目标
	return createMixedWriteSyncer(config)
}

// createOptions 创建zap选项
// 根据配置生成zap日志器的选项，包括调用者信息、堆栈追踪、采样等
// 这些选项控制日志器的行为和输出内容
func createOptions(config *Config) []zap.Option {
	opts := make([]zap.Option, 0, 4)

	if config.EnableCaller {
		opts = append(opts, zap.AddCaller())
	}

	if config.EnableStacktrace {
		opts = append(opts, zap.AddStacktrace(zapcore.ErrorLevel))
	}

	if config.Sampling != nil {
		opts = append(opts, zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewSamplerWithOptions(
				core,
				time.Second,
				config.Sampling.Initial,
				config.Sampling.Thereafter,
			)
		}))
	}

	// 添加自定义字段
	if len(config.CustomFields) > 0 {
		fields := make([]zap.Field, 0, len(config.CustomFields))
		for k, v := range config.CustomFields {
			fields = append(fields, zap.Any(k, v))
		}
		opts = append(opts, zap.Fields(fields...))
	}

	return opts
}

// Debug 记录调试级别日志
// 用于记录详细的调试信息，通常只在开发环境中启用
// 参数:
//   - msg: 日志消息
//   - fields: 结构化字段，提供额外的上下文信息
func (l *Logger) Debug(msg string, fields ...Field) {
	if l.isClosed() {
		return
	}
	l.zapLogger.Debug(msg, fields...)
}

// Info 记录信息级别日志
// 用于记录一般的信息性消息，表示程序正常运行
// 参数:
//   - msg: 日志消息
//   - fields: 结构化字段，提供额外的上下文信息
func (l *Logger) Info(msg string, fields ...Field) {
	if l.isClosed() {
		return
	}
	l.zapLogger.Info(msg, fields...)
}

// Warn 记录警告级别日志
// 用于记录潜在的问题或异常情况，但不影响程序继续运行
// 参数:
//   - msg: 日志消息
//   - fields: 结构化字段，提供额外的上下文信息
func (l *Logger) Warn(msg string, fields ...Field) {
	if l.isClosed() {
		return
	}
	l.zapLogger.Warn(msg, fields...)
}

// Error 记录错误级别日志
// 用于记录错误信息，表示程序遇到了问题但仍可继续运行
// 参数:
//   - msg: 日志消息
//   - fields: 结构化字段，提供额外的上下文信息
func (l *Logger) Error(msg string, fields ...Field) {
	if l.isClosed() {
		return
	}
	l.zapLogger.Error(msg, fields...)
}

// Fatal 记录致命级别日志
// 用于记录致命错误，记录后程序将退出
// 注意：此方法会调用os.Exit(1)，请谨慎使用
// 参数:
//   - msg: 日志消息
//   - fields: 结构化字段，提供额外的上下文信息
func (l *Logger) Fatal(msg string, fields ...Field) {
	if l.isClosed() {
		return
	}
	l.zapLogger.Fatal(msg, fields...)
}

// Debugf 格式化调试级别日志
func (l *Logger) Debugf(template string, args ...interface{}) {
	if l.isClosed() {
		return
	}
	l.zapLogger.Sugar().Debugf(template, args...)
}

// Infof 格式化信息级别日志
func (l *Logger) Infof(template string, args ...interface{}) {
	if l.isClosed() {
		return
	}
	l.zapLogger.Sugar().Infof(template, args...)
}

// Warnf 格式化警告级别日志
func (l *Logger) Warnf(template string, args ...interface{}) {
	if l.isClosed() {
		return
	}
	l.zapLogger.Sugar().Warnf(template, args...)
}

// Errorf 格式化错误级别日志
func (l *Logger) Errorf(template string, args ...interface{}) {
	if l.isClosed() {
		return
	}
	l.zapLogger.Sugar().Errorf(template, args...)
}

// Fatalf 格式化致命级别日志
func (l *Logger) Fatalf(template string, args ...interface{}) {
	if l.isClosed() {
		return
	}
	l.zapLogger.Sugar().Fatalf(template, args...)
}

// Debugw 键值对调试级别日志
func (l *Logger) Debugw(msg string, keysAndValues ...interface{}) {
	if l.isClosed() {
		return
	}
	l.zapLogger.Sugar().Debugw(msg, keysAndValues...)
}

// Infow 键值对信息级别日志
func (l *Logger) Infow(msg string, keysAndValues ...interface{}) {
	if l.isClosed() {
		return
	}
	l.zapLogger.Sugar().Infow(msg, keysAndValues...)
}

// Warnw 键值对警告级别日志
func (l *Logger) Warnw(msg string, keysAndValues ...interface{}) {
	if l.isClosed() {
		return
	}
	l.zapLogger.Sugar().Warnw(msg, keysAndValues...)
}

// Errorw 键值对错误级别日志
func (l *Logger) Errorw(msg string, keysAndValues ...interface{}) {
	if l.isClosed() {
		return
	}
	l.zapLogger.Sugar().Errorw(msg, keysAndValues...)
}

// Fatalw 键值对致命级别日志
func (l *Logger) Fatalw(msg string, keysAndValues ...interface{}) {
	if l.isClosed() {
		return
	}
	l.zapLogger.Sugar().Fatalw(msg, keysAndValues...)
}

// With 创建带有预设字段的新日志器
// 返回一个新的日志器实例，该实例会在所有日志消息中包含指定的字段
// 这对于在特定上下文中添加通用字段（如请求ID、用户ID等）非常有用
// 参数:
//   - fields: 要预设的结构化字段
// 返回:
//   - *Logger: 新的日志器实例，包含预设字段
func (l *Logger) With(fields ...Field) *Logger {
	if l.isClosed() {
		return l
	}
	return &Logger{
		zapLogger: l.zapLogger.With(fields...),
		config:    l.config,
		level:     l.level,
		buffer:    l.buffer,
	}
}

// WithContext 创建带有上下文信息的新日志器
// 从上下文中提取相关信息（如trace_id、request_id、user_id等）并添加到日志器中
// 这对于在请求处理过程中自动添加追踪信息非常有用
// 参数:
//   - ctx: 包含上下文信息的Context对象
// 返回:
//   - *Logger: 新的日志器实例，包含从上下文提取的字段
func (l *Logger) WithContext(ctx context.Context) *Logger {
	// 可以从context中提取trace id等信息
	fields := extractFieldsFromContext(ctx)
	return l.With(fields...)
}

// SetLevel 动态设置日志级别
// 允许在运行时调整日志级别，只有等于或高于设置级别的日志才会被输出
// 这对于在生产环境中动态调整日志详细程度非常有用
// 参数:
//   - level: 要设置的日志级别
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level.SetLevel(level.ToZapLevel())
}

// GetLevel 获取当前日志级别
// 返回当前设置的日志级别，可用于条件性日志记录或级别检查
// 返回:
//   - LogLevel: 当前的日志级别
func (l *Logger) GetLevel() LogLevel {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return FromZapLevel(l.level.Level())
}

// Sync 同步日志缓冲区
// 强制将缓冲区中的所有日志数据写入到目标输出
// 在程序退出前或需要确保日志完整性时应调用此方法
// 返回:
//   - error: 同步过程中的错误，如果有的话
func (l *Logger) Sync() error {
	if l.isClosed() {
		return nil
	}

	if l.buffer != nil {
		l.buffer.Flush()
	}

	return l.zapLogger.Sync()
}

// Close 关闭日志器并释放资源
// 关闭异步缓冲器（如果有），同步所有待写入的日志，并释放相关资源
// 日志器关闭后不应再使用，所有后续的日志调用都会被忽略
// 返回:
//   - error: 关闭过程中的错误，如果有的话
func (l *Logger) Close() error {
	if !atomic.CompareAndSwapInt32(&l.closed, 0, 1) {
		return nil
	}

	if l.buffer != nil {
		l.buffer.Close()
	}

	return l.zapLogger.Sync()
}

// isClosed 检查是否已关闭
func (l *Logger) isClosed() bool {
	return atomic.LoadInt32(&l.closed) == 1
}

// extractFieldsFromContext 从上下文中提取字段
func extractFieldsFromContext(ctx context.Context) []Field {
	var fields []Field

	// 提取trace id
	if traceID := ctx.Value("trace_id"); traceID != nil {
		if id, ok := traceID.(string); ok {
			fields = append(fields, String("trace_id", id))
		}
	}

	// 提取request id
	if requestID := ctx.Value("request_id"); requestID != nil {
		if id, ok := requestID.(string); ok {
			fields = append(fields, String("request_id", id))
		}
	}

	// 提取user id
	if userID := ctx.Value("user_id"); userID != nil {
		if id, ok := userID.(string); ok {
			fields = append(fields, String("user_id", id))
		}
	}

	return fields
}
