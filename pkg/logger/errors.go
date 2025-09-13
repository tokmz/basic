package logger

import (
	"fmt"
)

// LoggerError 日志器错误类型
type LoggerError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Cause   error  `json:"cause,omitempty"`
}

// Error 实现error接口
func (e *LoggerError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap 展开底层错误
func (e *LoggerError) Unwrap() error {
	return e.Cause
}

// NewLoggerError 创建日志器错误
func NewLoggerError(code, message string, cause error) *LoggerError {
	return &LoggerError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// 预定义错误
var (
	ErrInvalidConfig    = NewLoggerError("INVALID_CONFIG", "无效的配置", nil)
	ErrInvalidLevel     = NewLoggerError("INVALID_LEVEL", "无效的日志级别", nil)
	ErrCreateFile       = NewLoggerError("CREATE_FILE", "创建日志文件失败", nil)
	ErrCreateDir        = NewLoggerError("CREATE_DIR", "创建日志目录失败", nil)
	ErrWriteLog         = NewLoggerError("WRITE_LOG", "写入日志失败", nil)
	ErrRotateLog        = NewLoggerError("ROTATE_LOG", "日志轮转失败", nil)
	ErrSyncLog          = NewLoggerError("SYNC_LOG", "同步日志失败", nil)
	ErrHookFailed       = NewLoggerError("HOOK_FAILED", "日志钩子执行失败", nil)
	ErrLoggerClosed     = NewLoggerError("LOGGER_CLOSED", "日志器已关闭", nil)
	ErrEncoderConfig    = NewLoggerError("ENCODER_CONFIG", "编码器配置错误", nil)
	ErrOutputConfig     = NewLoggerError("OUTPUT_CONFIG", "输出配置错误", nil)
)

// ErrorCode 错误码常量
const (
	ErrCodeInvalidConfig = "INVALID_CONFIG"
	ErrCodeInvalidLevel  = "INVALID_LEVEL"
	ErrCodeCreateFile    = "CREATE_FILE"
	ErrCodeCreateDir     = "CREATE_DIR"
	ErrCodeWriteLog      = "WRITE_LOG"
	ErrCodeRotateLog     = "ROTATE_LOG"
	ErrCodeSyncLog       = "SYNC_LOG"
	ErrCodeHookFailed    = "HOOK_FAILED"
	ErrCodeLoggerClosed  = "LOGGER_CLOSED"
	ErrCodeEncoderConfig = "ENCODER_CONFIG"
	ErrCodeOutputConfig  = "OUTPUT_CONFIG"
)