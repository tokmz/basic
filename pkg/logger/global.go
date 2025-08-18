package logger

import (
	"context"
	"sync"
)

var (
	// 全局日志器实例
	globalLogger *Logger
	globalMu     sync.RWMutex
	once         sync.Once
)

// Init 初始化全局日志器
func Init(config *Config) error {
	logger, err := New(config)
	if err != nil {
		return err
	}

	globalMu.Lock()
	if globalLogger != nil {
		globalLogger.Close()
	}
	globalLogger = logger
	globalMu.Unlock()

	return nil
}

// InitDefault 使用默认配置初始化全局日志器
func InitDefault() error {
	return Init(DefaultConfig())
}

// InitDevelopment 使用开发环境配置初始化全局日志器
func InitDevelopment() error {
	return Init(DevelopmentConfig())
}

// InitTesting 使用测试环境配置初始化全局日志器
func InitTesting() error {
	return Init(TestingConfig())
}

// InitProduction 使用生产环境配置初始化全局日志器
func InitProduction() error {
	return Init(ProductionConfig())
}

// getGlobalLogger 获取全局日志器
func getGlobalLogger() *Logger {
	globalMu.RLock()
	logger := globalLogger
	globalMu.RUnlock()

	if logger == nil {
		// 懒加载默认日志器
		once.Do(func() {
			if err := InitDefault(); err != nil {
				// 如果初始化失败，创建一个最基本的日志器
				logger, _ = New(DefaultConfig())
				globalMu.Lock()
				globalLogger = logger
				globalMu.Unlock()
			}
		})
		globalMu.RLock()
		logger = globalLogger
		globalMu.RUnlock()
	}

	return logger
}

// SetGlobalLogger 设置全局日志器
func SetGlobalLogger(logger *Logger) {
	globalMu.Lock()
	if globalLogger != nil {
		globalLogger.Close()
	}
	globalLogger = logger
	globalMu.Unlock()
}

// GetGlobalLogger 获取全局日志器副本
func GetGlobalLogger() *Logger {
	return getGlobalLogger()
}

// 全局便捷函数 - 结构化日志

// Debug 全局调试日志
func Debug(msg string, fields ...Field) {
	getGlobalLogger().Debug(msg, fields...)
}

// Info 全局信息日志
func Info(msg string, fields ...Field) {
	getGlobalLogger().Info(msg, fields...)
}

// Warn 全局警告日志
func Warn(msg string, fields ...Field) {
	getGlobalLogger().Warn(msg, fields...)
}

// ErrorLog 全局错误日志
func ErrorLog(msg string, fields ...Field) {
	getGlobalLogger().Error(msg, fields...)
}

// Fatal 全局致命日志
func Fatal(msg string, fields ...Field) {
	getGlobalLogger().Fatal(msg, fields...)
}

// 全局便捷函数 - 格式化日志

// Debugf 全局格式化调试日志
func Debugf(template string, args ...interface{}) {
	getGlobalLogger().Debugf(template, args...)
}

// Infof 全局格式化信息日志
func Infof(template string, args ...interface{}) {
	getGlobalLogger().Infof(template, args...)
}

// Warnf 全局格式化警告日志
func Warnf(template string, args ...interface{}) {
	getGlobalLogger().Warnf(template, args...)
}

// Errorf 全局格式化错误日志
func Errorf(template string, args ...interface{}) {
	getGlobalLogger().Errorf(template, args...)
}

// Fatalf 全局格式化致命日志
func Fatalf(template string, args ...interface{}) {
	getGlobalLogger().Fatalf(template, args...)
}

// 全局便捷函数 - 键值对日志

// Debugw 全局键值对调试日志
func Debugw(msg string, keysAndValues ...interface{}) {
	getGlobalLogger().Debugw(msg, keysAndValues...)
}

// Infow 全局键值对信息日志
func Infow(msg string, keysAndValues ...interface{}) {
	getGlobalLogger().Infow(msg, keysAndValues...)
}

// Warnw 全局键值对警告日志
func Warnw(msg string, keysAndValues ...interface{}) {
	getGlobalLogger().Warnw(msg, keysAndValues...)
}

// Errorw 全局键值对错误日志
func Errorw(msg string, keysAndValues ...interface{}) {
	getGlobalLogger().Errorw(msg, keysAndValues...)
}

// Fatalw 全局键值对致命日志
func Fatalw(msg string, keysAndValues ...interface{}) {
	getGlobalLogger().Fatalw(msg, keysAndValues...)
}

// 全局工具函数

// With 添加字段到全局日志器
func With(fields ...Field) *Logger {
	return getGlobalLogger().With(fields...)
}

// WithContext 添加上下文到全局日志器
func WithContext(ctx context.Context) *Logger {
	return getGlobalLogger().WithContext(ctx)
}

// SetLevel 设置全局日志级别
func SetLevel(level LogLevel) {
	getGlobalLogger().SetLevel(level)
}

// GetLevel 获取全局日志级别
func GetLevel() LogLevel {
	return getGlobalLogger().GetLevel()
}

// Sync 同步全局日志
func Sync() error {
	return getGlobalLogger().Sync()
}

// Close 关闭全局日志器
func Close() error {
	globalMu.Lock()
	defer globalMu.Unlock()

	if globalLogger != nil {
		err := globalLogger.Close()
		globalLogger = nil
		return err
	}
	return nil
}
