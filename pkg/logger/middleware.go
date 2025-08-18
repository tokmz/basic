package logger

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ContextKey 上下文键类型
type ContextKey string

const (
	// LoggerContextKey 日志器上下文键
	LoggerContextKey ContextKey = "logger"
	// RequestIDKey 请求ID上下文键
	RequestIDKey ContextKey = "request_id"
	// TraceIDKey 链路追踪ID上下文键
	TraceIDKey ContextKey = "trace_id"
)

// WithLogger 将日志器添加到上下文
func WithLogger(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, LoggerContextKey, logger)
}

// FromContext 从上下文获取日志器
func FromContext(ctx context.Context) *Logger {
	if logger, ok := ctx.Value(LoggerContextKey).(*Logger); ok {
		return logger
	}
	return GetGlobalLogger()
}

// WithRequestID 将请求ID添加到上下文
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// GetRequestID 从上下文获取请求ID
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// WithTraceID 将链路追踪ID添加到上下文
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// GetTraceID 从上下文获取链路追踪ID
func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// RequestLogger HTTP请求日志记录器
type RequestLogger struct {
	logger *Logger
	config RequestLoggerConfig
}

// RequestLoggerConfig 请求日志配置
type RequestLoggerConfig struct {
	// SkipPaths 跳过记录的路径
	SkipPaths []string
	// LogRequestBody 是否记录请求体
	LogRequestBody bool
	// LogResponseBody 是否记录响应体
	LogResponseBody bool
	// MaxBodySize 最大记录的请求/响应体大小
	MaxBodySize int
	// SlowThreshold 慢请求阈值
	SlowThreshold time.Duration
}

// DefaultRequestLoggerConfig 默认请求日志配置
func DefaultRequestLoggerConfig() RequestLoggerConfig {
	return RequestLoggerConfig{
		SkipPaths:       []string{"/health", "/metrics", "/favicon.ico"},
		LogRequestBody:  false,
		LogResponseBody: false,
		MaxBodySize:     1024,
		SlowThreshold:   time.Second,
	}
}

// NewRequestLogger 创建请求日志记录器
func NewRequestLogger(logger *Logger, config RequestLoggerConfig) *RequestLogger {
	return &RequestLogger{
		logger: logger,
		config: config,
	}
}

// shouldSkip 检查是否应该跳过记录
func (rl *RequestLogger) shouldSkip(path string) bool {
	for _, skipPath := range rl.config.SkipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	return false
}

// LogRequest 记录HTTP请求
func (rl *RequestLogger) LogRequest(ctx context.Context, method, path, userAgent, clientIP string, statusCode int, latency time.Duration, requestSize, responseSize int64) {
	if rl.shouldSkip(path) {
		return
	}

	fields := []zap.Field{
		zap.String("method", method),
		zap.String("path", path),
		zap.String("user_agent", userAgent),
		zap.String("client_ip", clientIP),
		zap.Int("status_code", statusCode),
		zap.Duration("latency", latency),
		zap.Int64("request_size", requestSize),
		zap.Int64("response_size", responseSize),
	}

	// 添加请求ID和链路追踪ID
	if requestID := GetRequestID(ctx); requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}
	if traceID := GetTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}

	// 根据状态码和延迟选择日志级别
	var level LogLevel
	var msg string

	switch {
	case statusCode >= 500:
		level = ErrorLevel
		msg = "HTTP request failed"
	case statusCode >= 400:
		level = WarnLevel
		msg = "HTTP request error"
	case latency > rl.config.SlowThreshold:
		level = WarnLevel
		msg = "HTTP slow request"
	default:
		level = InfoLevel
		msg = "HTTP request"
	}

	switch level {
	case DebugLevel:
		rl.logger.Debug(msg, fields...)
	case InfoLevel:
		rl.logger.Info(msg, fields...)
	case WarnLevel:
		rl.logger.Warn(msg, fields...)
	case ErrorLevel:
		rl.logger.Error(msg, fields...)
	case FatalLevel:
		rl.logger.Fatal(msg, fields...)
	}
}

// ErrorHandler 错误处理器
type ErrorHandler struct {
	logger *Logger
	config ErrorHandlerConfig
}

// ErrorHandlerConfig 错误处理配置
type ErrorHandlerConfig struct {
	// LogStackTrace 是否记录堆栈跟踪
	LogStackTrace bool
	// StackTraceLevel 记录堆栈跟踪的最小级别
	StackTraceLevel LogLevel
	// MaxStackDepth 最大堆栈深度
	MaxStackDepth int
}

// DefaultErrorHandlerConfig 默认错误处理配置
func DefaultErrorHandlerConfig() ErrorHandlerConfig {
	return ErrorHandlerConfig{
		LogStackTrace:   true,
		StackTraceLevel: ErrorLevel,
		MaxStackDepth:   32,
	}
}

// NewErrorHandler 创建错误处理器
func NewErrorHandler(logger *Logger, config ErrorHandlerConfig) *ErrorHandler {
	return &ErrorHandler{
		logger: logger,
		config: config,
	}
}

// HandleError 处理错误
func (eh *ErrorHandler) HandleError(ctx context.Context, err error, level LogLevel, msg string, fields ...zap.Field) {
	if err == nil {
		return
	}

	// 添加错误字段
	fields = append(fields, zap.Error(err))

	// 添加上下文信息
	if requestID := GetRequestID(ctx); requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}
	if traceID := GetTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}

	// 添加堆栈跟踪
	if eh.config.LogStackTrace && level >= eh.config.StackTraceLevel {
		stack := eh.getStackTrace()
		fields = append(fields, zap.String("stack_trace", stack))
	}

	// 记录日志
	switch level {
	case DebugLevel:
		eh.logger.Debug(msg, fields...)
	case InfoLevel:
		eh.logger.Info(msg, fields...)
	case WarnLevel:
		eh.logger.Warn(msg, fields...)
	case ErrorLevel:
		eh.logger.Error(msg, fields...)
	case FatalLevel:
		eh.logger.Fatal(msg, fields...)
	}
}

// getStackTrace 获取堆栈跟踪
func (eh *ErrorHandler) getStackTrace() string {
	var stack strings.Builder
	for i := 2; i < eh.config.MaxStackDepth+2; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		stack.WriteString(fmt.Sprintf("%s:%d %s\n", file, line, fn.Name()))
	}
	return stack.String()
}

// RecoveryHandler 恢复处理器
type RecoveryHandler struct {
	logger *Logger
	config RecoveryHandlerConfig
}

// RecoveryHandlerConfig 恢复处理配置
type RecoveryHandlerConfig struct {
	// LogStackTrace 是否记录堆栈跟踪
	LogStackTrace bool
	// MaxStackDepth 最大堆栈深度
	MaxStackDepth int
	// PanicLevel panic日志级别
	PanicLevel LogLevel
}

// DefaultRecoveryHandlerConfig 默认恢复处理配置
func DefaultRecoveryHandlerConfig() RecoveryHandlerConfig {
	return RecoveryHandlerConfig{
		LogStackTrace: true,
		MaxStackDepth: 32,
		PanicLevel:    ErrorLevel,
	}
}

// NewRecoveryHandler 创建恢复处理器
func NewRecoveryHandler(logger *Logger, config RecoveryHandlerConfig) *RecoveryHandler {
	return &RecoveryHandler{
		logger: logger,
		config: config,
	}
}

// HandlePanic 处理panic
func (rh *RecoveryHandler) HandlePanic(ctx context.Context, recovered interface{}) {
	fields := []zap.Field{
		zap.Any("panic", recovered),
	}

	// 添加上下文信息
	if requestID := GetRequestID(ctx); requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}
	if traceID := GetTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}

	// 添加堆栈跟踪
	if rh.config.LogStackTrace {
		stack := rh.getStackTrace()
		fields = append(fields, zap.String("stack_trace", stack))
	}

	// 记录panic日志
	switch rh.config.PanicLevel {
	case DebugLevel:
		rh.logger.Debug("Panic recovered", fields...)
	case InfoLevel:
		rh.logger.Info("Panic recovered", fields...)
	case WarnLevel:
		rh.logger.Warn("Panic recovered", fields...)
	case ErrorLevel:
		rh.logger.Error("Panic recovered", fields...)
	case FatalLevel:
		rh.logger.Fatal("Panic recovered", fields...)
	}
}

// getStackTrace 获取堆栈跟踪
func (rh *RecoveryHandler) getStackTrace() string {
	var stack strings.Builder
	for i := 3; i < rh.config.MaxStackDepth+3; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		stack.WriteString(fmt.Sprintf("%s:%d %s\n", file, line, fn.Name()))
	}
	return stack.String()
}

// PerformanceTracker 性能追踪器
type PerformanceTracker struct {
	logger *Logger
	config PerformanceTrackerConfig
}

// PerformanceTrackerConfig 性能追踪配置
type PerformanceTrackerConfig struct {
	// SlowThreshold 慢操作阈值
	SlowThreshold time.Duration
	// LogLevel 日志级别
	LogLevel LogLevel
	// LogDetails 是否记录详细信息
	LogDetails bool
}

// DefaultPerformanceTrackerConfig 默认性能追踪配置
func DefaultPerformanceTrackerConfig() PerformanceTrackerConfig {
	return PerformanceTrackerConfig{
		SlowThreshold: 100 * time.Millisecond,
		LogLevel:      WarnLevel,
		LogDetails:    true,
	}
}

// NewPerformanceTracker 创建性能追踪器
func NewPerformanceTracker(logger *Logger, config PerformanceTrackerConfig) *PerformanceTracker {
	return &PerformanceTracker{
		logger: logger,
		config: config,
	}
}

// TrackOperation 追踪操作性能
func (pt *PerformanceTracker) TrackOperation(ctx context.Context, operation string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	fields := []zap.Field{
		zap.String("operation", operation),
		zap.Duration("duration", duration),
	}

	// 添加上下文信息
	if requestID := GetRequestID(ctx); requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}
	if traceID := GetTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}

	// 添加错误信息
	if err != nil {
		fields = append(fields, zap.Error(err))
	}

	// 添加详细信息
	if pt.config.LogDetails {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		fields = append(fields,
			zap.Int("goroutines", runtime.NumGoroutine()),
			zap.Uint64("memory_alloc", m.Alloc),
		)
	}

	// 根据性能和错误状态选择日志级别和消息
	var level LogLevel
	var msg string

	switch {
	case err != nil:
		level = ErrorLevel
		msg = "Operation failed"
	case duration > pt.config.SlowThreshold:
		level = pt.config.LogLevel
		msg = "Slow operation detected"
	default:
		level = DebugLevel
		msg = "Operation completed"
	}

	// 记录日志
	switch level {
	case DebugLevel:
		pt.logger.Debug(msg, fields...)
	case InfoLevel:
		pt.logger.Info(msg, fields...)
	case WarnLevel:
		pt.logger.Warn(msg, fields...)
	case ErrorLevel:
		pt.logger.Error(msg, fields...)
	case FatalLevel:
		pt.logger.Fatal(msg, fields...)
	}

	return err
}
