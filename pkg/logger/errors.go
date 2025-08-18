package logger

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"
)

// ErrorType 错误类型
type ErrorType string

const (
	// 系统错误
	SystemError ErrorType = "system"
	// 网络错误
	NetworkError ErrorType = "network"
	// 数据库错误
	DatabaseError ErrorType = "database"
	// 认证错误
	AuthError ErrorType = "auth"
	// 权限错误
	PermissionError ErrorType = "permission"
	// 验证错误
	ValidationError ErrorType = "validation"
	// 业务逻辑错误
	BusinessError ErrorType = "business"
	// 配置错误
	ConfigError ErrorType = "config"
	// 外部服务错误
	ExternalError ErrorType = "external"
	// 未知错误
	UnknownError ErrorType = "unknown"
)

// ErrorSeverity 错误严重程度
type ErrorSeverity string

const (
	// 低严重程度
	LowSeverity ErrorSeverity = "low"
	// 中等严重程度
	MediumSeverity ErrorSeverity = "medium"
	// 高严重程度
	HighSeverity ErrorSeverity = "high"
	// 严重
	CriticalSeverity ErrorSeverity = "critical"
)

// LoggerError 日志器自定义错误
type LoggerError struct {
	// 错误类型
	Type ErrorType `json:"type"`
	// 错误代码
	Code string `json:"code"`
	// 错误消息
	Message string `json:"message"`
	// 原始错误
	Cause error `json:"-"`
	// 错误严重程度
	Severity ErrorSeverity `json:"severity"`
	// 错误发生时间
	Timestamp time.Time `json:"timestamp"`
	// 错误上下文
	Context map[string]interface{} `json:"context,omitempty"`
	// 堆栈信息
	StackTrace string `json:"stack_trace,omitempty"`
	// 是否可重试
	Retryable bool `json:"retryable"`
	// 操作名称
	Operation string `json:"operation,omitempty"`
}

// Error 实现error接口
// 返回格式化的错误字符串，包含错误代码、消息和原因（如果有）
// 返回:
//   - string: 格式化的错误描述
func (e *LoggerError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap 实现errors.Unwrap接口
// 用于错误链的展开，返回包装的原始错误
// 返回:
//   - error: 原始错误，如果没有则返回nil
func (e *LoggerError) Unwrap() error {
	return e.Cause
}

// Is 实现errors.Is接口
// 判断当前错误是否与目标错误相同，基于错误类型和代码进行比较
// 参数:
//   - target: 要比较的目标错误
// 返回:
//   - bool: 如果错误匹配则返回true，否则返回false
func (e *LoggerError) Is(target error) bool {
	if target == nil {
		return false
	}
	if loggerErr, ok := target.(*LoggerError); ok {
		return e.Type == loggerErr.Type && e.Code == loggerErr.Code
	}
	return false
}

// NewLoggerError 创建新的日志器错误
// 参数:
//   - errorType: 错误类型，用于分类错误
//   - code: 错误代码，用于唯一标识错误
//   - message: 错误消息，描述错误详情
// 返回:
//   - *LoggerError: 新创建的日志器错误实例
func NewLoggerError(errorType ErrorType, code, message string) *LoggerError {
	return &LoggerError{
		Type:      errorType,
		Code:      code,
		Message:   message,
		Severity:  MediumSeverity,
		Timestamp: time.Now(),
		Context:   make(map[string]interface{}),
		Retryable: false,
	}
}

// WithCause 设置原始错误
// 用于包装原始错误，支持错误链
// 参数:
//   - cause: 要包装的原始错误
// 返回:
//   - *LoggerError: 返回自身以支持链式调用
func (e *LoggerError) WithCause(cause error) *LoggerError {
	e.Cause = cause
	return e
}

// WithSeverity 设置错误严重程度
// 用于标识错误的严重程度，影响日志级别和处理策略
// 参数:
//   - severity: 错误严重程度
// 返回:
//   - *LoggerError: 返回自身以支持链式调用
func (e *LoggerError) WithSeverity(severity ErrorSeverity) *LoggerError {
	e.Severity = severity
	return e
}

// WithContext 添加上下文信息
// 用于添加额外的上下文数据，便于错误分析和调试
// 参数:
//   - key: 上下文键
//   - value: 上下文值
// 返回:
//   - *LoggerError: 返回自身以支持链式调用
func (e *LoggerError) WithContext(key string, value interface{}) *LoggerError {
	e.Context[key] = value
	return e
}

// WithStackTrace 添加堆栈跟踪
// 捕获当前调用栈信息，便于定位错误发生位置
// 返回:
//   - *LoggerError: 返回自身以支持链式调用
func (e *LoggerError) WithStackTrace() *LoggerError {
	e.StackTrace = getStackTrace(32)
	return e
}

// WithRetryable 设置是否可重试
// 标识错误是否可以通过重试解决，用于自动恢复机制
// 参数:
//   - retryable: 是否可重试
// 返回:
//   - *LoggerError: 返回自身以支持链式调用
func (e *LoggerError) WithRetryable(retryable bool) *LoggerError {
	e.Retryable = retryable
	return e
}

// WithOperation 设置操作名称
// 标识发生错误的操作，便于错误分类和统计
// 参数:
//   - operation: 操作名称
// 返回:
//   - *LoggerError: 返回自身以支持链式调用
func (e *LoggerError) WithOperation(operation string) *LoggerError {
	e.Operation = operation
	return e
}

// getStackTrace 获取堆栈跟踪
func getStackTrace(maxDepth int) string {
	var stack strings.Builder
	for i := 1; i < maxDepth+1; i++ {
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

// 预定义的错误代码
const (
	// 系统错误代码
	ErrCodeSystemFailure     = "SYS_001"
	ErrCodeMemoryExhausted   = "SYS_002"
	ErrCodeDiskFull          = "SYS_003"
	ErrCodePermissionDenied  = "SYS_004"

	// 网络错误代码
	ErrCodeNetworkTimeout    = "NET_001"
	ErrCodeConnectionRefused = "NET_002"
	ErrCodeDNSResolution     = "NET_003"
	ErrCodeNetworkUnreachable = "NET_004"

	// 数据库错误代码
	ErrCodeDatabaseConnection = "DB_001"
	ErrCodeQueryTimeout      = "DB_002"
	ErrCodeConstraintViolation = "DB_003"
	ErrCodeDeadlock          = "DB_004"

	// 认证错误代码
	ErrCodeInvalidCredentials = "AUTH_001"
	ErrCodeTokenExpired      = "AUTH_002"
	ErrCodeTokenInvalid      = "AUTH_003"
	ErrCodeAccountLocked     = "AUTH_004"

	// 验证错误代码
	ErrCodeInvalidInput      = "VAL_001"
	ErrCodeMissingRequired   = "VAL_002"
	ErrCodeFormatError       = "VAL_003"
	ErrCodeRangeError        = "VAL_004"

	// 业务逻辑错误代码
	ErrCodeBusinessRule      = "BIZ_001"
	ErrCodeResourceNotFound  = "BIZ_002"
	ErrCodeResourceConflict  = "BIZ_003"
	ErrCodeOperationNotAllowed = "BIZ_004"

	// 配置错误代码
	ErrCodeConfigMissing     = "CFG_001"
	ErrCodeConfigInvalid     = "CFG_002"
	ErrCodeConfigParseError  = "CFG_003"

	// 外部服务错误代码
	ErrCodeExternalTimeout   = "EXT_001"
	ErrCodeExternalUnavailable = "EXT_002"
	ErrCodeExternalRateLimit = "EXT_003"
)

// 预定义错误构造函数

// NewSystemError 创建系统错误
func NewSystemError(code, message string) *LoggerError {
	return NewLoggerError(SystemError, code, message).WithSeverity(HighSeverity)
}

// NewNetworkError 创建网络错误
func NewNetworkError(code, message string) *LoggerError {
	return NewLoggerError(NetworkError, code, message).WithSeverity(MediumSeverity).WithRetryable(true)
}

// NewDatabaseError 创建数据库错误
func NewDatabaseError(code, message string) *LoggerError {
	return NewLoggerError(DatabaseError, code, message).WithSeverity(HighSeverity)
}

// NewAuthError 创建认证错误
func NewAuthError(code, message string) *LoggerError {
	return NewLoggerError(AuthError, code, message).WithSeverity(MediumSeverity)
}

// NewValidationError 创建验证错误
func NewValidationError(code, message string) *LoggerError {
	return NewLoggerError(ValidationError, code, message).WithSeverity(LowSeverity)
}

// NewBusinessError 创建业务错误
func NewBusinessError(code, message string) *LoggerError {
	return NewLoggerError(BusinessError, code, message).WithSeverity(MediumSeverity)
}

// NewConfigError 创建配置错误
func NewConfigError(code, message string) *LoggerError {
	return NewLoggerError(ConfigError, code, message).WithSeverity(CriticalSeverity)
}

// NewExternalError 创建外部服务错误
func NewExternalError(code, message string) *LoggerError {
	return NewLoggerError(ExternalError, code, message).WithSeverity(MediumSeverity).WithRetryable(true)
}

// ErrorClassifier 错误分类器
// 用于自动分析和分类错误，将通用错误转换为结构化的LoggerError
// 支持基于错误消息、类型等信息进行智能分类
type ErrorClassifier struct {
	// logger 用于记录分类过程中的日志
	logger *Logger
}

// NewErrorClassifier 创建错误分类器
// 参数:
//   - logger: 日志器实例，用于记录分类过程
// 返回:
//   - *ErrorClassifier: 新创建的错误分类器
func NewErrorClassifier(logger *Logger) *ErrorClassifier {
	return &ErrorClassifier{
		logger: logger,
	}
}

// ClassifyError 分类错误
// 根据错误的类型、消息等信息自动分类错误
// 将普通error转换为结构化的LoggerError，便于统一处理
// 参数:
//   - err: 要分类的错误
// 返回:
//   - *LoggerError: 分类后的结构化错误
func (ec *ErrorClassifier) ClassifyError(err error) *LoggerError {
	if err == nil {
		return nil
	}

	// 如果已经是LoggerError，直接返回
	if loggerErr, ok := err.(*LoggerError); ok {
		return loggerErr
	}

	// 根据错误消息进行分类
	errorMsg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errorMsg, "connection"):
		if strings.Contains(errorMsg, "database") {
			return NewDatabaseError(ErrCodeDatabaseConnection, err.Error()).WithCause(err)
		}
		return NewNetworkError(ErrCodeConnectionRefused, err.Error()).WithCause(err)

	case strings.Contains(errorMsg, "timeout"):
		if strings.Contains(errorMsg, "database") || strings.Contains(errorMsg, "sql") {
			return NewDatabaseError(ErrCodeQueryTimeout, err.Error()).WithCause(err)
		}
		return NewNetworkError(ErrCodeNetworkTimeout, err.Error()).WithCause(err)

	case strings.Contains(errorMsg, "permission") || strings.Contains(errorMsg, "access denied"):
		return NewSystemError(ErrCodePermissionDenied, err.Error()).WithCause(err)

	case strings.Contains(errorMsg, "not found"):
		return NewBusinessError(ErrCodeResourceNotFound, err.Error()).WithCause(err)

	case strings.Contains(errorMsg, "unauthorized") || strings.Contains(errorMsg, "authentication"):
		return NewAuthError(ErrCodeInvalidCredentials, err.Error()).WithCause(err)

	case strings.Contains(errorMsg, "config"):
		return NewConfigError(ErrCodeConfigInvalid, err.Error()).WithCause(err)

	case strings.Contains(errorMsg, "invalid") || strings.Contains(errorMsg, "malformed"):
		return NewValidationError(ErrCodeInvalidInput, err.Error()).WithCause(err)

	default:
		return NewLoggerError(UnknownError, "UNKNOWN_001", err.Error()).WithCause(err).WithSeverity(MediumSeverity).WithRetryable(true)
	}
}

// ErrorRecovery 错误恢复器
// 提供自动错误恢复机制，包括重试、退避、熔断等策略
// 用于提高系统的容错能力和稳定性
type ErrorRecovery struct {
	// logger 用于记录恢复过程的日志
	logger     *Logger
	// classifier 用于分类和分析错误
	classifier *ErrorClassifier
	// config 恢复策略配置
	config     ErrorRecoveryConfig
}

// ErrorRecoveryConfig 错误恢复配置
// 定义错误恢复的各种策略参数
type ErrorRecoveryConfig struct {
	// 最大重试次数
	MaxRetries int
	// 重试间隔
	RetryInterval time.Duration
	// 指数退避因子
	BackoffFactor float64
	// 最大重试间隔
	MaxRetryInterval time.Duration
	// 启用断路器
	EnableCircuitBreaker bool
	// 断路器失败阈值
	CircuitBreakerThreshold int
	// 断路器恢复时间
	CircuitBreakerTimeout time.Duration
}

// DefaultErrorRecoveryConfig 默认错误恢复配置
func DefaultErrorRecoveryConfig() ErrorRecoveryConfig {
	return ErrorRecoveryConfig{
		MaxRetries:              3,
		RetryInterval:           time.Second,
		BackoffFactor:           2.0,
		MaxRetryInterval:        30 * time.Second,
		EnableCircuitBreaker:    true,
		CircuitBreakerThreshold: 5,
		CircuitBreakerTimeout:   60 * time.Second,
	}
}

// NewErrorRecovery 创建错误恢复器
func NewErrorRecovery(logger *Logger, config ErrorRecoveryConfig) *ErrorRecovery {
	return &ErrorRecovery{
		logger:     logger,
		classifier: NewErrorClassifier(logger),
		config:     config,
	}
}

// RecoveryFunc 恢复函数类型
type RecoveryFunc func() error

// ExecuteWithRecovery 执行带恢复的操作
func (er *ErrorRecovery) ExecuteWithRecovery(ctx context.Context, operation string, fn RecoveryFunc) error {
	var lastErr error
	retryInterval := er.config.RetryInterval

	for attempt := 0; attempt <= er.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// 检查上下文是否已取消
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// 等待重试间隔
			timer := time.NewTimer(retryInterval)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}

			// 指数退避
			retryInterval = time.Duration(float64(retryInterval) * er.config.BackoffFactor)
			if retryInterval > er.config.MaxRetryInterval {
				retryInterval = er.config.MaxRetryInterval
			}
		}

		err := fn()
		if err == nil {
			if attempt > 0 {
				er.logger.Info("操作重试成功",
					String("operation", operation),
					Int("attempt", attempt+1),
				)
			}
			return nil
		}

		lastErr = err
		loggerErr := er.classifier.ClassifyError(err)

		// 记录错误
		er.logger.Error("操作执行失败",
			String("operation", operation),
			Int("attempt", attempt+1),
			String("error_type", string(loggerErr.Type)),
			String("error_code", loggerErr.Code),
			String("error_severity", string(loggerErr.Severity)),
			Bool("retryable", loggerErr.Retryable),
			Error(err),
		)

		// 如果错误不可重试，直接返回
		if !loggerErr.Retryable {
			er.logger.Warn("错误不可重试，停止重试",
				String("operation", operation),
				String("error_code", loggerErr.Code),
			)
			return loggerErr
		}

		// 如果是最后一次尝试，返回错误
		if attempt == er.config.MaxRetries {
			er.logger.Error("操作重试次数已达上限",
				String("operation", operation),
				Int("max_retries", er.config.MaxRetries),
				String("final_error", loggerErr.Error()),
			)
			return loggerErr
		}
	}

	return lastErr
}

// LogError 记录错误的便捷方法
func (l *Logger) LogError(ctx context.Context, err error, operation string, fields ...Field) {
	if err == nil {
		return
	}

	classifier := NewErrorClassifier(l)
	loggerErr := classifier.ClassifyError(err)

	// 构建字段
	allFields := []Field{
		String("operation", operation),
		String("error_type", string(loggerErr.Type)),
		String("error_code", loggerErr.Code),
		String("error_severity", string(loggerErr.Severity)),
		Bool("retryable", loggerErr.Retryable),
		Time("error_timestamp", loggerErr.Timestamp),
		Error(err),
	}

	// 添加上下文信息
	if requestID := GetRequestID(ctx); requestID != "" {
		allFields = append(allFields, String("request_id", requestID))
	}
	if traceID := GetTraceID(ctx); traceID != "" {
		allFields = append(allFields, String("trace_id", traceID))
	}

	// 添加错误上下文
	for key, value := range loggerErr.Context {
		allFields = append(allFields, Any(fmt.Sprintf("ctx_%s", key), value))
	}

	// 添加用户提供的字段
	allFields = append(allFields, fields...)

	// 添加堆栈跟踪（如果有）
	if loggerErr.StackTrace != "" {
		allFields = append(allFields, String("stack_trace", loggerErr.StackTrace))
	}

	// 根据严重程度选择日志级别
	switch loggerErr.Severity {
	case LowSeverity:
		l.Warn(loggerErr.Message, allFields...)
	case MediumSeverity:
		l.Error(loggerErr.Message, allFields...)
	case HighSeverity, CriticalSeverity:
		l.Error(loggerErr.Message, allFields...)
		// 对于严重错误，可以考虑发送告警
	default:
		l.Error(loggerErr.Message, allFields...)
	}
}

// IsRetryableError 检查错误是否可重试
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	if loggerErr, ok := err.(*LoggerError); ok {
		return loggerErr.Retryable
	}

	// 对于非LoggerError，根据错误消息判断
	errorMsg := strings.ToLower(err.Error())
	return strings.Contains(errorMsg, "timeout") ||
		strings.Contains(errorMsg, "connection") ||
		strings.Contains(errorMsg, "network") ||
		strings.Contains(errorMsg, "temporary")
}

// GetErrorSeverity 获取错误严重程度
func GetErrorSeverity(err error) ErrorSeverity {
	if err == nil {
		return LowSeverity
	}

	if loggerErr, ok := err.(*LoggerError); ok {
		return loggerErr.Severity
	}

	return MediumSeverity
}

// WrapError 包装错误为LoggerError
func WrapError(err error, errorType ErrorType, code, message string) *LoggerError {
	if err == nil {
		return nil
	}

	return NewLoggerError(errorType, code, message).WithCause(err).WithStackTrace()
}