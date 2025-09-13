package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger 企业级日志器
type Logger struct {
	config *Config
	zap    *zap.Logger
	sugar  *zap.SugaredLogger
	mu     sync.RWMutex
}

// Fields 日志字段类型
type Fields map[string]interface{}

// globalLogger 全局日志器实例
var (
	globalLogger *Logger
	globalOnce   sync.Once
)

// New 创建新的日志器实例
func New(config *Config) (*Logger, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	// 创建核心配置
	zapConfig := zap.NewProductionConfig()
	if config.Format == ConsoleFormat {
		zapConfig = zap.NewDevelopmentConfig()
	}

	// 设置日志级别
	level, err := zapcore.ParseLevel(string(config.Level))
	if err != nil {
		return nil, fmt.Errorf("无效的日志级别 %s: %w", config.Level, err)
	}
	zapConfig.Level = zap.NewAtomicLevelAt(level)

	// 设置时间格式
	zapConfig.EncoderConfig.TimeKey = "timestamp"
	zapConfig.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(config.TimeFormat)

	// 设置调用者信息
	if config.EnableCaller {
		zapConfig.EncoderConfig.CallerKey = "caller"
		zapConfig.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	} else {
		zapConfig.EncoderConfig.CallerKey = ""
	}

	// 设置堆栈信息
	if config.EnableStack {
		zapConfig.EncoderConfig.StacktraceKey = "stacktrace"
	} else {
		zapConfig.EncoderConfig.StacktraceKey = ""
	}

	// 添加服务信息到初始字段
	zapConfig.InitialFields = map[string]interface{}{
		"service_name": config.ServiceName,
		"service_env":  config.ServiceEnv,
	}

	// 创建编码器
	encoder := getEncoder(config)

	// 创建输出核心
	cores := []zapcore.Core{}

	// 控制台输出
	if config.Console.Enabled {
		consoleEncoder := encoder
		if config.Format == ConsoleFormat && config.Console.ColoredOut {
			consoleEncoder = zapcore.NewConsoleEncoder(zapConfig.EncoderConfig)
		}
		cores = append(cores, zapcore.NewCore(
			consoleEncoder,
			zapcore.AddSync(os.Stdout),
			level,
		))
	}

	// 文件输出
	if config.File.Enabled {
		fileWriter, err := getFileWriter(config.File)
		if err != nil {
			return nil, fmt.Errorf("创建文件写入器失败: %w", err)
		}

		cores = append(cores, zapcore.NewCore(
			encoder,
			zapcore.AddSync(fileWriter),
			level,
		))
	}

	if len(cores) == 0 {
		return nil, fmt.Errorf("至少需要启用一种输出方式")
	}

	// 创建核心
	core := zapcore.NewTee(cores...)

	// 创建zap日志器
	options := []zap.Option{}
	if config.EnableCaller {
		options = append(options, zap.AddCaller())
	}
	if config.EnableStack {
		options = append(options, zap.AddStacktrace(zapcore.ErrorLevel))
	}

	// 添加采样器
	if config.EnableSampler {
		core = zapcore.NewSamplerWithOptions(
			core,
			time.Second,
			100, // 第一秒内最多100条
			10,  // 之后每秒最多10条
		)
	}

	zapLogger := zap.New(core, options...)

	return &Logger{
		config: config,
		zap:    zapLogger,
		sugar:  zapLogger.Sugar(),
	}, nil
}

// InitGlobal 初始化全局日志器
func InitGlobal(config *Config) error {
	var err error
	globalOnce.Do(func() {
		globalLogger, err = New(config)
	})
	return err
}

// Global 获取全局日志器
func Global() *Logger {
	if globalLogger == nil {
		// 如果没有初始化，使用默认配置
		_ = InitGlobal(DefaultConfig())
	}
	return globalLogger
}

// getEncoder 获取编码器
func getEncoder(config *Config) zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	if config.Format == ConsoleFormat {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	}

	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(config.TimeFormat)
	encoderConfig.LevelKey = "level"
	encoderConfig.NameKey = "logger"
	encoderConfig.MessageKey = "message"

	if config.EnableCaller {
		encoderConfig.CallerKey = "caller"
		encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	}

	if config.EnableStack {
		encoderConfig.StacktraceKey = "stacktrace"
	}

	switch config.Format {
	case JSONFormat:
		return zapcore.NewJSONEncoder(encoderConfig)
	case ConsoleFormat:
		if config.Console.ColoredOut {
			encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}
		return zapcore.NewConsoleEncoder(encoderConfig)
	default:
		return zapcore.NewJSONEncoder(encoderConfig)
	}
}

// getFileWriter 获取文件写入器
func getFileWriter(config FileConfig) (io.Writer, error) {
	// 确保日志目录存在
	dir := filepath.Dir(config.Filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %w", err)
	}

	// 使用lumberjack进行日志轮转
	return &lumberjack.Logger{
		Filename:   config.Filename,
		MaxSize:    config.Rotation.MaxSize,
		MaxAge:     config.Rotation.MaxAge,
		MaxBackups: config.Rotation.MaxBackups,
		Compress:   config.Rotation.Compress,
		LocalTime:  true,
	}, nil
}

// WithFields 添加字段
func (l *Logger) WithFields(fields Fields) *Logger {
	if len(fields) == 0 {
		return l
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	zapFields := make([]zap.Field, 0, len(fields))
	for key, value := range fields {
		zapFields = append(zapFields, zap.Any(key, value))
	}

	return &Logger{
		config: l.config,
		zap:    l.zap.With(zapFields...),
		sugar:  l.zap.With(zapFields...).Sugar(),
	}
}

// WithContext 从上下文中添加字段
func (l *Logger) WithContext(ctx context.Context) *Logger {
	if ctx == nil {
		return l
	}

	fields := Fields{}

	// 从上下文中提取常见字段
	if traceID := ctx.Value("trace_id"); traceID != nil {
		fields["trace_id"] = traceID
	}

	if requestID := ctx.Value("request_id"); requestID != nil {
		fields["request_id"] = requestID
	}

	if userID := ctx.Value("user_id"); userID != nil {
		fields["user_id"] = userID
	}

	return l.WithFields(fields)
}

// Debug 记录Debug级别日志
func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.zap.Debug(msg, fields...)
}

// Debugf 记录Debug级别日志(格式化)
func (l *Logger) Debugf(template string, args ...interface{}) {
	l.sugar.Debugf(template, args...)
}

// Info 记录Info级别日志
func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.zap.Info(msg, fields...)
}

// Infof 记录Info级别日志(格式化)
func (l *Logger) Infof(template string, args ...interface{}) {
	l.sugar.Infof(template, args...)
}

// Warn 记录Warn级别日志
func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.zap.Warn(msg, fields...)
}

// Warnf 记录Warn级别日志(格式化)
func (l *Logger) Warnf(template string, args ...interface{}) {
	l.sugar.Warnf(template, args...)
}

// Error 记录Error级别日志
func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.zap.Error(msg, fields...)
}

// Errorf 记录Error级别日志(格式化)
func (l *Logger) Errorf(template string, args ...interface{}) {
	l.sugar.Errorf(template, args...)
}

// Panic 记录Panic级别日志
func (l *Logger) Panic(msg string, fields ...zap.Field) {
	l.zap.Panic(msg, fields...)
}

// Panicf 记录Panic级别日志(格式化)
func (l *Logger) Panicf(template string, args ...interface{}) {
	l.sugar.Panicf(template, args...)
}

// Fatal 记录Fatal级别日志
func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	l.zap.Fatal(msg, fields...)
}

// Fatalf 记录Fatal级别日志(格式化)
func (l *Logger) Fatalf(template string, args ...interface{}) {
	l.sugar.Fatalf(template, args...)
}

// Sync 同步日志输出
func (l *Logger) Sync() error {
	if l.zap != nil {
		return l.zap.Sync()
	}
	return nil
}

// Close 关闭日志器
func (l *Logger) Close() error {
	return l.Sync()
}

// GetLevel 获取当前日志级别
func (l *Logger) GetLevel() LogLevel {
	return l.config.Level
}

// SetLevel 动态设置日志级别
func (l *Logger) SetLevel(level LogLevel) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	_, err := zapcore.ParseLevel(string(level))
	if err != nil {
		return fmt.Errorf("无效的日志级别 %s: %w", level, err)
	}

	// 注意：zap不支持动态修改level，这里只更新配置
	l.config.Level = level
	return nil
}

// GetConfig 获取配置副本
func (l *Logger) GetConfig() *Config {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// 返回配置的深拷贝
	config := *l.config
	return &config
}
