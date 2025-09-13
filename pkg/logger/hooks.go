package logger

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Hook 日志钩子接口
type Hook interface {
	Levels() []LogLevel
	Fire(entry *Entry) error
}

// Entry 日志条目
type Entry struct {
	Level   LogLevel
	Time    time.Time
	Message string
	Fields  Fields
	Context context.Context
}

// HookLogger 支持钩子的日志器
type HookLogger struct {
	*Logger
	hooks []Hook
}

// NewWithHooks 创建支持钩子的日志器
func NewWithHooks(config *Config, hooks ...Hook) (*HookLogger, error) {
	logger, err := New(config)
	if err != nil {
		return nil, err
	}
	
	return &HookLogger{
		Logger: logger,
		hooks:  hooks,
	}, nil
}

// AddHook 添加钩子
func (hl *HookLogger) AddHook(hook Hook) {
	hl.hooks = append(hl.hooks, hook)
}

// RemoveHook 移除钩子
func (hl *HookLogger) RemoveHook(hook Hook) {
	for i, h := range hl.hooks {
		if h == hook {
			hl.hooks = append(hl.hooks[:i], hl.hooks[i+1:]...)
			break
		}
	}
}

// fireHooks 触发钩子
func (hl *HookLogger) fireHooks(level LogLevel, msg string, fields Fields, ctx context.Context) {
	if len(hl.hooks) == 0 {
		return
	}
	
	entry := &Entry{
		Level:   level,
		Time:    time.Now(),
		Message: msg,
		Fields:  fields,
		Context: ctx,
	}
	
	for _, hook := range hl.hooks {
		if hl.shouldFireHook(hook, level) {
			go func(h Hook) {
				if err := h.Fire(entry); err != nil {
					// 钩子错误不应该影响正常日志记录
					fmt.Printf("Hook error: %v\n", err)
				}
			}(hook)
		}
	}
}

// shouldFireHook 检查是否应该触发钩子
func (hl *HookLogger) shouldFireHook(hook Hook, level LogLevel) bool {
	levels := hook.Levels()
	if len(levels) == 0 {
		return true
	}
	
	for _, l := range levels {
		if l == level {
			return true
		}
	}
	return false
}

// 重写日志方法以支持钩子

// Debug 记录Debug级别日志
func (hl *HookLogger) Debug(msg string, fields ...zap.Field) {
	hl.fireHooks(DebugLevel, msg, zapFieldsToFields(fields), context.Background())
	hl.Logger.Debug(msg, fields...)
}

// Info 记录Info级别日志
func (hl *HookLogger) Info(msg string, fields ...zap.Field) {
	hl.fireHooks(InfoLevel, msg, zapFieldsToFields(fields), context.Background())
	hl.Logger.Info(msg, fields...)
}

// Warn 记录Warn级别日志
func (hl *HookLogger) Warn(msg string, fields ...zap.Field) {
	hl.fireHooks(WarnLevel, msg, zapFieldsToFields(fields), context.Background())
	hl.Logger.Warn(msg, fields...)
}

// Error 记录Error级别日志
func (hl *HookLogger) Error(msg string, fields ...zap.Field) {
	hl.fireHooks(ErrorLevel, msg, zapFieldsToFields(fields), context.Background())
	hl.Logger.Error(msg, fields...)
}

// zapFieldsToFields 转换zap字段到Fields
func zapFieldsToFields(fields []zap.Field) Fields {
	result := make(Fields)
	for _, field := range fields {
		switch field.Type {
		case zapcore.StringType:
			result[field.Key] = field.String
		case zapcore.Int64Type:
			result[field.Key] = field.Integer
		case zapcore.Float64Type:
			result[field.Key] = field.Interface
		case zapcore.BoolType:
			result[field.Key] = field.Integer == 1
		default:
			result[field.Key] = field.Interface
		}
	}
	return result
}

// EmailHook 邮件告警钩子示例
type EmailHook struct {
	levels   []LogLevel
	smtpHost string
	smtpPort int
	username string
	password string
	to       []string
}

// NewEmailHook 创建邮件钩子
func NewEmailHook(smtpHost string, smtpPort int, username, password string, to []string) *EmailHook {
	return &EmailHook{
		levels:   []LogLevel{ErrorLevel, PanicLevel, FatalLevel},
		smtpHost: smtpHost,
		smtpPort: smtpPort,
		username: username,
		password: password,
		to:       to,
	}
}

// Levels 返回钩子关注的日志级别
func (h *EmailHook) Levels() []LogLevel {
	return h.levels
}

// Fire 触发钩子
func (h *EmailHook) Fire(entry *Entry) error {
	// 这里实现邮件发送逻辑
	// 为了简化，这里只是打印
	fmt.Printf("Email Alert: [%s] %s at %s\n", entry.Level, entry.Message, entry.Time.Format(time.RFC3339))
	return nil
}

// WebhookHook Webhook钩子示例
type WebhookHook struct {
	levels []LogLevel
	url    string
}

// NewWebhookHook 创建Webhook钩子
func NewWebhookHook(url string, levels ...LogLevel) *WebhookHook {
	if len(levels) == 0 {
		levels = []LogLevel{ErrorLevel, PanicLevel, FatalLevel}
	}
	return &WebhookHook{
		levels: levels,
		url:    url,
	}
}

// Levels 返回钩子关注的日志级别
func (h *WebhookHook) Levels() []LogLevel {
	return h.levels
}

// Fire 触发钩子
func (h *WebhookHook) Fire(entry *Entry) error {
	// 这里实现HTTP POST到webhook URL的逻辑
	// 为了简化，这里只是打印
	fmt.Printf("Webhook Alert to %s: [%s] %s at %s\n", h.url, entry.Level, entry.Message, entry.Time.Format(time.RFC3339))
	return nil
}

// DatabaseHook 数据库钩子示例
type DatabaseHook struct {
	levels []LogLevel
	// 数据库连接信息
}

// NewDatabaseHook 创建数据库钩子
func NewDatabaseHook(levels ...LogLevel) *DatabaseHook {
	if len(levels) == 0 {
		levels = []LogLevel{InfoLevel, WarnLevel, ErrorLevel, PanicLevel, FatalLevel}
	}
	return &DatabaseHook{
		levels: levels,
	}
}

// Levels 返回钩子关注的日志级别
func (h *DatabaseHook) Levels() []LogLevel {
	return h.levels
}

// Fire 触发钩子
func (h *DatabaseHook) Fire(entry *Entry) error {
	// 这里实现写入数据库的逻辑
	// 为了简化，这里只是打印
	fmt.Printf("Database Log: [%s] %s at %s\n", entry.Level, entry.Message, entry.Time.Format(time.RFC3339))
	return nil
}