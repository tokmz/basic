package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

// ZapLogger 使用zap实现GORM的日志接口
type ZapLogger struct {
	ZapLogger                 *zap.Logger
	LogLevel                  logger.LogLevel
	SlowThreshold             time.Duration
	Colorful                  bool
	IgnoreRecordNotFoundError bool
}

// NewZapLogger 创建新的基于zap的GORM日志器
func NewZapLogger(zapLogger *zap.Logger, config LoggerConfig) logger.Interface {
	return &ZapLogger{
		ZapLogger:                 zapLogger,
		LogLevel:                  convertLogLevel(config.Level),
		SlowThreshold:             config.SlowThreshold,
		Colorful:                  config.Colorful,
		IgnoreRecordNotFoundError: true,
	}
}

// LogMode 设置日志器的日志模式
func (l *ZapLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

// Info 记录信息级别的消息
func (l *ZapLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Info {
		l.ZapLogger.Sugar().Infof(msg, data...)
	}
}

// Warn 记录警告级别的消息
func (l *ZapLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Warn {
		l.ZapLogger.Sugar().Warnf(msg, data...)
	}
}

// Error 记录错误级别的消息
func (l *ZapLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Error {
		l.ZapLogger.Sugar().Errorf(msg, data...)
	}
}

// Trace 记录SQL执行跟踪信息
func (l *ZapLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	fields := []zap.Field{
		zap.String("sql", sql),
		zap.Duration("elapsed", elapsed),
		zap.Int64("rows", rows),
		zap.String("file", utils.FileWithLineNum()),
	}

	switch {
	case err != nil && l.LogLevel >= logger.Error && (!errors.Is(err, logger.ErrRecordNotFound) || !l.IgnoreRecordNotFoundError):
		fields = append(fields, zap.Error(err))
		l.ZapLogger.Error("SQL Error", fields...)
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= logger.Warn:
		fields = append(fields, zap.Bool("slow_query", true))
		l.ZapLogger.Warn("Slow SQL Query", fields...)
	case l.LogLevel == logger.Info:
		l.ZapLogger.Info("SQL Query", fields...)
	}
}

// createGormLogger 基于配置创建GORM日志器
func (c *Client) createGormLogger() logger.Interface {
	// 如果提供了外部zap日志器则使用它
	if c.config.Logger.ZapLogger != nil {
		return NewZapLogger(c.config.Logger.ZapLogger, c.config.Logger)
	}

	// 创建默认的GORM日志器
	logLevel := convertLogLevel(c.config.Logger.Level)
	return logger.New(
		&GormLogWriter{},
		logger.Config{
			SlowThreshold:             c.config.Logger.SlowThreshold,
			LogLevel:                  logLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  c.config.Logger.Colorful,
		},
	)
}

// GormLogWriter 为GORM的默认日志器实现logger.Writer接口
type GormLogWriter struct{}

// Write 实现io.Writer接口
func (w *GormLogWriter) Write(p []byte) (n int, err error) {
	fmt.Print(string(p))
	return len(p), nil
}

// Printf 实现logger.Writer接口
func (w *GormLogWriter) Printf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

// convertLogLevel 将我们的LogLevel转换为GORM的LogLevel
func convertLogLevel(level LogLevel) logger.LogLevel {
	switch level {
	case Silent:
		return logger.Silent
	case Error:
		return logger.Error
	case Warn:
		return logger.Warn
	case Info:
		return logger.Info
	default:
		return logger.Info
	}
}

// SetZapLogger 更新客户端的zap日志器
func (c *Client) SetZapLogger(zapLogger *zap.Logger) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("client is closed")
	}

	c.config.Logger.ZapLogger = zapLogger

	// 更新GORM日志器
	newLogger := NewZapLogger(zapLogger, c.config.Logger)
	c.db.Logger = newLogger

	return nil
}
