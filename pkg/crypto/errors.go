package crypto

import (
	"errors"
	"fmt"
)

// 预定义错误
var (
	ErrInvalidKey           = errors.New("invalid key")
	ErrInvalidKeySize       = errors.New("invalid key size")
	ErrInvalidNonce         = errors.New("invalid nonce")
	ErrInvalidSignature     = errors.New("invalid signature")
	ErrInvalidAlgorithm     = errors.New("invalid algorithm")
	ErrInvalidInput         = errors.New("invalid input")
	ErrInvalidConfiguration = errors.New("invalid configuration")

	ErrKeyNotFound      = errors.New("key not found")
	ErrKeyExpired       = errors.New("key expired")
	ErrKeyAlreadyExists = errors.New("key already exists")

	ErrSignatureExpired   = errors.New("signature expired")
	ErrSignatureReplay    = errors.New("signature replay attack detected")
	ErrSignatureMismatch  = errors.New("signature mismatch")

	ErrEncryptionFailed   = errors.New("encryption failed")
	ErrDecryptionFailed   = errors.New("decryption failed")
	ErrAuthenticationFailed = errors.New("authentication failed")

	ErrContextCanceled = errors.New("context canceled")
	ErrTimeout         = errors.New("operation timeout")

	ErrObfuscationFailed = errors.New("obfuscation failed")
	ErrDeobfuscationFailed = errors.New("deobfuscation failed")
)

// CryptoError 加密错误类型
type CryptoError struct {
	Op        string // 操作名称
	Algorithm string // 算法名称
	Err       error  // 原始错误
	Code      int    // 错误代码
	Details   map[string]any // 额外详情
}

func (e *CryptoError) Error() string {
	if e.Algorithm != "" {
		return fmt.Sprintf("crypto %s (%s): %v", e.Op, e.Algorithm, e.Err)
	}
	return fmt.Sprintf("crypto %s: %v", e.Op, e.Err)
}

func (e *CryptoError) Unwrap() error {
	return e.Err
}

// NewCryptoError 创建加密错误
func NewCryptoError(op, algorithm string, err error) *CryptoError {
	return &CryptoError{
		Op:        op,
		Algorithm: algorithm,
		Err:       err,
		Details:   make(map[string]any),
	}
}

// WithCode 设置错误代码
func (e *CryptoError) WithCode(code int) *CryptoError {
	e.Code = code
	return e
}

// WithDetails 添加错误详情
func (e *CryptoError) WithDetails(key string, value any) *CryptoError {
	if e.Details == nil {
		e.Details = make(map[string]any)
	}
	e.Details[key] = value
	return e
}

// IsTemporary 判断是否为临时错误
func (e *CryptoError) IsTemporary() bool {
	return e.Code >= 500 && e.Code < 600
}

// IsRetryable 判断是否可重试
func (e *CryptoError) IsRetryable() bool {
	switch e.Err {
	case ErrTimeout, ErrContextCanceled:
		return true
	default:
		return e.IsTemporary()
	}
}

// ValidationError 验证错误
type ValidationError struct {
	Field   string
	Value   any
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s': %s", e.Field, e.Message)
}

// NewValidationError 创建验证错误
func NewValidationError(field string, value any, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// SecurityError 安全错误
type SecurityError struct {
	Type    string // 安全错误类型
	Message string
	Details map[string]any
}

func (e *SecurityError) Error() string {
	return fmt.Sprintf("security error (%s): %s", e.Type, e.Message)
}

// NewSecurityError 创建安全错误
func NewSecurityError(errorType, message string) *SecurityError {
	return &SecurityError{
		Type:    errorType,
		Message: message,
		Details: make(map[string]any),
	}
}

// ObfuscationError 混淆错误
type ObfuscationError struct {
	Stage   string // 混淆阶段
	Reason  string // 失败原因
	Details map[string]any
}

func (e *ObfuscationError) Error() string {
	return fmt.Sprintf("obfuscation error at stage '%s': %s", e.Stage, e.Reason)
}

// NewObfuscationError 创建混淆错误
func NewObfuscationError(stage, reason string) *ObfuscationError {
	return &ObfuscationError{
		Stage:   stage,
		Reason:  reason,
		Details: make(map[string]any),
	}
}

// 错误代码常量
const (
	// 通用错误代码
	CodeSuccess        = 0
	CodeInvalidInput   = 400
	CodeUnauthorized   = 401
	CodeForbidden      = 403
	CodeNotFound       = 404
	CodeConflict       = 409
	CodeInternalError  = 500
	CodeTimeout        = 504

	// 加密相关错误代码
	CodeEncryptionFailed  = 1001
	CodeDecryptionFailed  = 1002
	CodeInvalidKey        = 1003
	CodeInvalidSignature  = 1004
	CodeSignatureExpired  = 1005
	CodeReplayAttack      = 1006

	// 混淆相关错误代码
	CodeObfuscationFailed   = 2001
	CodeDeobfuscationFailed = 2002
	CodeTamperDetected      = 2003
)

// IsKeyError 检查是否为密钥相关错误
func IsKeyError(err error) bool {
	switch err {
	case ErrInvalidKey, ErrInvalidKeySize, ErrKeyNotFound, ErrKeyExpired:
		return true
	default:
		var cryptoErr *CryptoError
		if errors.As(err, &cryptoErr) {
			return cryptoErr.Code == CodeInvalidKey
		}
		return false
	}
}

// IsSignatureError 检查是否为签名相关错误
func IsSignatureError(err error) bool {
	switch err {
	case ErrInvalidSignature, ErrSignatureExpired, ErrSignatureReplay, ErrSignatureMismatch:
		return true
	default:
		var cryptoErr *CryptoError
		if errors.As(err, &cryptoErr) {
			return cryptoErr.Code >= CodeInvalidSignature && cryptoErr.Code <= CodeReplayAttack
		}
		return false
	}
}

// IsSecurityError 检查是否为安全相关错误
func IsSecurityError(err error) bool {
	var secErr *SecurityError
	return errors.As(err, &secErr)
}