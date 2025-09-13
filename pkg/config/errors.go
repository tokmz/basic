package config

import (
	"fmt"
)

// 配置相关错误类型
var (
	// ErrConfigNotFound 配置文件未找到错误
	ErrConfigNotFound = NewConfigError("CONFIG_NOT_FOUND", "configuration file not found")
	
	// ErrConfigInvalid 配置无效错误
	ErrConfigInvalid = NewConfigError("CONFIG_INVALID", "configuration is invalid")
	
	// ErrConfigValidationFailed 配置验证失败错误
	ErrConfigValidationFailed = NewConfigError("CONFIG_VALIDATION_FAILED", "configuration validation failed")
	
	// ErrEncryptionKeyMissing 加密密钥缺失错误
	ErrEncryptionKeyMissing = NewConfigError("ENCRYPTION_KEY_MISSING", "encryption key is missing")
	
	// ErrDecryptionFailed 解密失败错误
	ErrDecryptionFailed = NewConfigError("DECRYPTION_FAILED", "failed to decrypt value")
	
	// ErrEnvironmentInvalid 环境无效错误
	ErrEnvironmentInvalid = NewConfigError("ENVIRONMENT_INVALID", "invalid environment")
	
	// ErrManagerClosed 管理器已关闭错误
	ErrManagerClosed = NewConfigError("MANAGER_CLOSED", "config manager is closed")
	
	// ErrRemoteConfigFailed 远程配置失败错误
	ErrRemoteConfigFailed = NewConfigError("REMOTE_CONFIG_FAILED", "failed to load remote configuration")
)

// ConfigError 配置错误类型
type ConfigError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
	Cause   error  `json:"-"`
}

// NewConfigError 创建新的配置错误
func NewConfigError(code, message string) *ConfigError {
	return &ConfigError{
		Code:    code,
		Message: message,
	}
}

// Error 实现error接口
func (e *ConfigError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// WithDetails 添加错误详情
func (e *ConfigError) WithDetails(details string) *ConfigError {
	return &ConfigError{
		Code:    e.Code,
		Message: e.Message,
		Details: details,
		Cause:   e.Cause,
	}
}

// WithDetailsf 使用格式化添加错误详情
func (e *ConfigError) WithDetailsf(format string, args ...interface{}) *ConfigError {
	return e.WithDetails(fmt.Sprintf(format, args...))
}

// WithCause 添加原因错误
func (e *ConfigError) WithCause(cause error) *ConfigError {
	return &ConfigError{
		Code:    e.Code,
		Message: e.Message,
		Details: e.Details,
		Cause:   cause,
	}
}

// Unwrap 返回原因错误
func (e *ConfigError) Unwrap() error {
	return e.Cause
}

// IsConfigError 检查是否为配置错误
func IsConfigError(err error) bool {
	_, ok := err.(*ConfigError)
	return ok
}

// IsConfigErrorWithCode 检查是否为指定代码的配置错误
func IsConfigErrorWithCode(err error, code string) bool {
	if configErr, ok := err.(*ConfigError); ok {
		return configErr.Code == code
	}
	return false
}

// ValidationError 验证错误
type ValidationError struct {
	Field   string      `json:"field"`
	Value   interface{} `json:"value"`
	Rule    string      `json:"rule"`
	Message string      `json:"message"`
}

// Error 实现error接口
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s': %s (rule: %s, value: %v)", 
		e.Field, e.Message, e.Rule, e.Value)
}

// NewValidationError 创建验证错误
func NewValidationError(field, rule, message string, value interface{}) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Rule:    rule,
		Message: message,
	}
}

// ValidationErrors 验证错误集合
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

// Error 实现error接口
func (e *ValidationErrors) Error() string {
	if len(e.Errors) == 0 {
		return "no validation errors"
	}
	
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	
	return fmt.Sprintf("%d validation errors: %s", len(e.Errors), e.Errors[0].Error())
}

// Add 添加验证错误
func (e *ValidationErrors) Add(field, rule, message string, value interface{}) {
	e.Errors = append(e.Errors, ValidationError{
		Field:   field,
		Value:   value,
		Rule:    rule,
		Message: message,
	})
}

// HasErrors 检查是否有错误
func (e *ValidationErrors) HasErrors() bool {
	return len(e.Errors) > 0
}

// Count 返回错误数量
func (e *ValidationErrors) Count() int {
	return len(e.Errors)
}

// GetByField 根据字段获取错误
func (e *ValidationErrors) GetByField(field string) []ValidationError {
	var result []ValidationError
	for _, err := range e.Errors {
		if err.Field == field {
			result = append(result, err)
		}
	}
	return result
}

// ErrorHandler 错误处理器接口
type ErrorHandler interface {
	HandleError(err error)
}

// DefaultErrorHandler 默认错误处理器
type DefaultErrorHandler struct {
	logger func(format string, args ...interface{})
}

// NewDefaultErrorHandler 创建默认错误处理器
func NewDefaultErrorHandler(logger func(format string, args ...interface{})) *DefaultErrorHandler {
	if logger == nil {
		logger = func(format string, args ...interface{}) {
			fmt.Printf("[CONFIG ERROR] "+format+"\n", args...)
		}
	}
	
	return &DefaultErrorHandler{
		logger: logger,
	}
}

// HandleError 处理错误
func (h *DefaultErrorHandler) HandleError(err error) {
	if err == nil {
		return
	}
	
	switch e := err.(type) {
	case *ConfigError:
		h.logger("Config Error [%s]: %s", e.Code, e.Message)
		if e.Details != "" {
			h.logger("Details: %s", e.Details)
		}
		if e.Cause != nil {
			h.logger("Caused by: %v", e.Cause)
		}
		
	case *ValidationError:
		h.logger("Validation Error: %s", e.Error())
		
	case *ValidationErrors:
		h.logger("Multiple Validation Errors (%d):", e.Count())
		for i, valErr := range e.Errors {
			h.logger("  %d. %s", i+1, valErr.Error())
		}
		
	default:
		h.logger("Unknown Error: %v", err)
	}
}

// RecoverableError 可恢复错误接口
type RecoverableError interface {
	error
	CanRecover() bool
	Recover() error
}

// ConfigLoadError 配置加载错误
type ConfigLoadError struct {
	*ConfigError
	FilePath    string
	Recoverable bool
}

// NewConfigLoadError 创建配置加载错误
func NewConfigLoadError(filePath string, cause error, recoverable bool) *ConfigLoadError {
	return &ConfigLoadError{
		ConfigError: ErrConfigNotFound.WithCause(cause).WithDetailsf("file: %s", filePath),
		FilePath:    filePath,
		Recoverable: recoverable,
	}
}

// CanRecover 检查是否可以恢复
func (e *ConfigLoadError) CanRecover() bool {
	return e.Recoverable
}

// Recover 尝试恢复
func (e *ConfigLoadError) Recover() error {
	// 这里可以实现具体的恢复逻辑
	// 比如创建默认配置文件、从备份恢复等
	return fmt.Errorf("recovery not implemented for config load error")
}

// WrapError 包装错误为配置错误
func WrapError(err error, code, message string) *ConfigError {
	return &ConfigError{
		Code:    code,
		Message: message,
		Cause:   err,
	}
}

// WrapErrorf 使用格式化消息包装错误
func WrapErrorf(err error, code, format string, args ...interface{}) *ConfigError {
	return WrapError(err, code, fmt.Sprintf(format, args...))
}