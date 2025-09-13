package logger

import (
	"context"

	"go.uber.org/zap"
)

// 全局便捷函数

// Debug 全局Debug日志
func Debug(msg string, fields ...zap.Field) {
	Global().Debug(msg, fields...)
}

// Debugf 全局Debug日志(格式化)
func Debugf(template string, args ...interface{}) {
	Global().Debugf(template, args...)
}

// Info 全局Info日志
func Info(msg string, fields ...zap.Field) {
	Global().Info(msg, fields...)
}

// Infof 全局Info日志(格式化)
func Infof(template string, args ...interface{}) {
	Global().Infof(template, args...)
}

// Warn 全局Warn日志
func Warn(msg string, fields ...zap.Field) {
	Global().Warn(msg, fields...)
}

// Warnf 全局Warn日志(格式化)
func Warnf(template string, args ...interface{}) {
	Global().Warnf(template, args...)
}

// Error 全局Error日志
func Error(msg string, fields ...zap.Field) {
	Global().Error(msg, fields...)
}

// Errorf 全局Error日志(格式化)
func Errorf(template string, args ...interface{}) {
	Global().Errorf(template, args...)
}

// Panic 全局Panic日志
func Panic(msg string, fields ...zap.Field) {
	Global().Panic(msg, fields...)
}

// Panicf 全局Panic日志(格式化)
func Panicf(template string, args ...interface{}) {
	Global().Panicf(template, args...)
}

// Fatal 全局Fatal日志
func Fatal(msg string, fields ...zap.Field) {
	Global().Fatal(msg, fields...)
}

// Fatalf 全局Fatal日志(格式化)
func Fatalf(template string, args ...interface{}) {
	Global().Fatalf(template, args...)
}

// WithFields 全局带字段日志
func WithFields(fields Fields) *Logger {
	return Global().WithFields(fields)
}

// WithContext 全局带上下文日志
func WithContext(ctx context.Context) *Logger {
	return Global().WithContext(ctx)
}

// Sync 同步全局日志
func Sync() error {
	return Global().Sync()
}

// SetLevel 设置全局日志级别
func SetLevel(level LogLevel) error {
	return Global().SetLevel(level)
}

// GetLevel 获取全局日志级别
func GetLevel() LogLevel {
	return Global().GetLevel()
}