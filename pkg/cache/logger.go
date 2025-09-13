package cache

import (
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger 缓存日志器
type Logger struct {
	*zap.Logger
	config MonitoringConfig
}

// NewLogger 创建日志器
func NewLogger(config MonitoringConfig) (*Logger, error) {
	if !config.EnableLogging {
		return &Logger{
			Logger: zap.NewNop(),
			config: config,
		}, nil
	}

	// 解析日志级别
	level := parseLogLevel(config.LogLevel)

	// 创建配置
	zapConfig := zap.Config{
		Level:       zap.NewAtomicLevelAt(level),
		Development: false,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding: "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	// 构建日志器
	logger, err := zapConfig.Build(zap.AddCallerSkip(1))
	if err != nil {
		return nil, err
	}

	return &Logger{
		Logger: logger.Named("cache"),
		config: config,
	}, nil
}

// LogOperation 记录操作日志
func (l *Logger) LogOperation(operation, key string, duration time.Duration, err error) {
	if !l.config.EnableLogging {
		return
	}

	fields := []zap.Field{
		zap.String("operation", operation),
		zap.String("key", key),
		zap.Duration("duration", duration),
	}

	if err != nil {
		fields = append(fields, zap.Error(err))
		l.Error("Cache operation failed", fields...)
	} else {
		// 检查是否为慢查询
		if duration > l.config.SlowThreshold {
			l.Warn("Slow cache operation", fields...)
		} else {
			l.Debug("Cache operation completed", fields...)
		}
	}
}

// LogError 记录错误日志
func (l *Logger) LogError(message string, err error, fields ...zap.Field) {
	if !l.config.EnableLogging {
		return
	}

	allFields := append(fields, zap.Error(err))
	l.Error(message, allFields...)
}

// LogInfo 记录信息日志
func (l *Logger) LogInfo(message string, fields ...zap.Field) {
	if !l.config.EnableLogging {
		return
	}

	l.Info(message, fields...)
}

// LogWarn 记录警告日志
func (l *Logger) LogWarn(message string, fields ...zap.Field) {
	if !l.config.EnableLogging {
		return
	}

	l.Warn(message, fields...)
}

// LogDebug 记录调试日志
func (l *Logger) LogDebug(message string, fields ...zap.Field) {
	if !l.config.EnableLogging {
		return
	}

	l.Debug(message, fields...)
}

// LogStats 记录统计日志
func (l *Logger) LogStats(stats CacheStats) {
	if !l.config.EnableLogging {
		return
	}

	l.Info("Cache statistics",
		zap.Int64("hits", stats.Hits),
		zap.Int64("misses", stats.Misses),
		zap.Int64("sets", stats.Sets),
		zap.Int64("deletes", stats.Deletes),
		zap.Int64("errors", stats.Errors),
		zap.Float64("hit_rate", stats.HitRate),
		zap.Int64("total_requests", stats.TotalRequests),
		zap.Int64("size", stats.Size),
	)
}

// LogPerformance 记录性能日志
func (l *Logger) LogPerformance(metrics *Metrics) {
	if !l.config.EnableLogging {
		return
	}

	l.Info("Cache performance metrics",
		zap.Duration("avg_response_time", metrics.AvgResponseTime),
		zap.Duration("max_response_time", metrics.MaxResponseTime),
		zap.Duration("min_response_time", metrics.MinResponseTime),
		zap.Int64("slow_queries", metrics.SlowQueries),
		zap.Float64("throughput", metrics.Throughput),
		zap.Int64("memory_usage", metrics.MemoryUsage),
		zap.Float64("cpu_usage", metrics.CPUUsage),
		zap.Int64("connection_count", metrics.ConnectionCount),
	)
}

// LogCacheEvent 记录缓存事件
func (l *Logger) LogCacheEvent(event, level string, message string, fields ...zap.Field) {
	if !l.config.EnableLogging {
		return
	}

	allFields := append([]zap.Field{
		zap.String("event", event),
		zap.String("level", level),
	}, fields...)

	switch level {
	case "error":
		l.Error(message, allFields...)
	case "warn":
		l.Warn(message, allFields...)
	case "info":
		l.Info(message, allFields...)
	case "debug":
		l.Debug(message, allFields...)
	default:
		l.Info(message, allFields...)
	}
}

// LogConnection 记录连接日志
func (l *Logger) LogConnection(action string, addr string, err error) {
	if !l.config.EnableLogging {
		return
	}

	fields := []zap.Field{
		zap.String("action", action),
		zap.String("address", addr),
	}

	if err != nil {
		fields = append(fields, zap.Error(err))
		l.Error("Cache connection event failed", fields...)
	} else {
		l.Info("Cache connection event", fields...)
	}
}

// LogConfiguration 记录配置日志
func (l *Logger) LogConfiguration(config interface{}) {
	if !l.config.EnableLogging {
		return
	}

	l.Info("Cache configuration loaded",
		zap.Any("config", config),
	)
}

// parseLogLevel 解析日志级别
func parseLogLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// StructuredLogger 结构化日志接口
type StructuredLogger interface {
	LogOperation(operation, key string, duration time.Duration, err error)
	LogError(message string, err error, fields ...zap.Field)
	LogInfo(message string, fields ...zap.Field)
	LogWarn(message string, fields ...zap.Field)
	LogDebug(message string, fields ...zap.Field)
	LogStats(stats CacheStats)
	LogPerformance(metrics *Metrics)
	LogCacheEvent(event, level string, message string, fields ...zap.Field)
	LogConnection(action string, addr string, err error)
	LogConfiguration(config interface{})
}
