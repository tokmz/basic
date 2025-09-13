package logger

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

// TestHook 测试钩子
type TestHook struct {
	levels   []LogLevel
	entries  []*Entry
	fireFunc func(*Entry) error
}

func NewTestHook(levels ...LogLevel) *TestHook {
	return &TestHook{
		levels:  levels,
		entries: make([]*Entry, 0),
	}
}

func (h *TestHook) Levels() []LogLevel {
	return h.levels
}

func (h *TestHook) Fire(entry *Entry) error {
	h.entries = append(h.entries, entry)
	if h.fireFunc != nil {
		return h.fireFunc(entry)
	}
	return nil
}

func (h *TestHook) GetEntries() []*Entry {
	return h.entries
}

func (h *TestHook) Clear() {
	h.entries = h.entries[:0]
}

func TestHookLogger(t *testing.T) {
	config := DevelopmentConfig()
	hook := NewTestHook(InfoLevel, ErrorLevel)
	
	hookLogger, err := NewWithHooks(config, hook)
	if err != nil {
		t.Fatalf("创建钩子日志器失败: %v", err)
	}
	defer hookLogger.Close()

	// 测试Info级别日志（应该触发钩子）
	hookLogger.Info("test info message", zap.String("key", "value"))
	
	// 测试Debug级别日志（不应该触发钩子）
	hookLogger.Debug("test debug message", zap.String("key", "value"))
	
	// 测试Error级别日志（应该触发钩子）
	hookLogger.Error("test error message", zap.String("key", "value"))

	// 等待钩子执行（异步）
	time.Sleep(100 * time.Millisecond)

	entries := hook.GetEntries()
	expectedCount := 2 // Info和Error
	if len(entries) != expectedCount {
		t.Errorf("期望 %d 个钩子条目，实际 %d 个", expectedCount, len(entries))
	}

	// 验证条目内容
	if len(entries) >= 1 {
		if entries[0].Level != InfoLevel {
			t.Errorf("第一个条目级别应该为 %s, 实际为 %s", InfoLevel, entries[0].Level)
		}
		if entries[0].Message != "test info message" {
			t.Errorf("第一个条目消息不匹配: %s", entries[0].Message)
		}
	}
}

func TestAddRemoveHook(t *testing.T) {
	config := DevelopmentConfig()
	hook1 := NewTestHook(InfoLevel)
	hook2 := NewTestHook(ErrorLevel)
	
	hookLogger, err := NewWithHooks(config, hook1)
	if err != nil {
		t.Fatalf("创建钩子日志器失败: %v", err)
	}
	defer hookLogger.Close()

	// 添加第二个钩子
	hookLogger.AddHook(hook2)

	// 记录日志
	hookLogger.Info("test message")
	hookLogger.Error("error message")

	time.Sleep(100 * time.Millisecond)

	// hook1应该只有Info消息
	entries1 := hook1.GetEntries()
	if len(entries1) != 1 {
		t.Errorf("hook1期望1个条目，实际%d个", len(entries1))
	}

	// hook2应该只有Error消息
	entries2 := hook2.GetEntries()
	if len(entries2) != 1 {
		t.Errorf("hook2期望1个条目，实际%d个", len(entries2))
	}

	// 移除hook1
	hookLogger.RemoveHook(hook1)
	hook1.Clear()
	hook2.Clear()

	// 再次记录日志
	hookLogger.Info("test message 2")
	hookLogger.Error("error message 2")

	time.Sleep(100 * time.Millisecond)

	// hook1不应该有新条目
	entries1 = hook1.GetEntries()
	if len(entries1) != 0 {
		t.Errorf("hook1移除后不应该有条目，实际%d个", len(entries1))
	}

	// hook2应该有新的Error消息
	entries2 = hook2.GetEntries()
	if len(entries2) != 1 {
		t.Errorf("hook2期望1个新条目，实际%d个", len(entries2))
	}
}

func TestEmailHook(t *testing.T) {
	hook := NewEmailHook("smtp.example.com", 587, "user@example.com", "password", []string{"admin@example.com"})
	
	levels := hook.Levels()
	expectedLevels := []LogLevel{ErrorLevel, PanicLevel, FatalLevel}
	
	if len(levels) != len(expectedLevels) {
		t.Errorf("期望 %d 个级别，实际 %d 个", len(expectedLevels), len(levels))
	}
	
	for i, level := range expectedLevels {
		if i < len(levels) && levels[i] != level {
			t.Errorf("级别 %d 应该为 %s, 实际为 %s", i, level, levels[i])
		}
	}
	
	// 测试触发钩子
	entry := &Entry{
		Level:   ErrorLevel,
		Time:    time.Now(),
		Message: "test error",
		Context: context.Background(),
	}
	
	if err := hook.Fire(entry); err != nil {
		t.Errorf("触发邮件钩子失败: %v", err)
	}
}

func TestWebhookHook(t *testing.T) {
	hook := NewWebhookHook("https://webhook.example.com")
	
	// 测试默认级别
	levels := hook.Levels()
	expectedLevels := []LogLevel{ErrorLevel, PanicLevel, FatalLevel}
	
	if len(levels) != len(expectedLevels) {
		t.Errorf("期望 %d 个级别，实际 %d 个", len(expectedLevels), len(levels))
	}
	
	// 测试自定义级别
	customHook := NewWebhookHook("https://webhook.example.com", InfoLevel, WarnLevel)
	customLevels := customHook.Levels()
	expectedCustomLevels := []LogLevel{InfoLevel, WarnLevel}
	
	if len(customLevels) != len(expectedCustomLevels) {
		t.Errorf("期望 %d 个自定义级别，实际 %d 个", len(expectedCustomLevels), len(customLevels))
	}
	
	// 测试触发钩子
	entry := &Entry{
		Level:   ErrorLevel,
		Time:    time.Now(),
		Message: "test error",
		Context: context.Background(),
	}
	
	if err := hook.Fire(entry); err != nil {
		t.Errorf("触发Webhook钩子失败: %v", err)
	}
}

func TestDatabaseHook(t *testing.T) {
	hook := NewDatabaseHook()
	
	// 测试默认级别
	levels := hook.Levels()
	expectedLevels := []LogLevel{InfoLevel, WarnLevel, ErrorLevel, PanicLevel, FatalLevel}
	
	if len(levels) != len(expectedLevels) {
		t.Errorf("期望 %d 个级别，实际 %d 个", len(expectedLevels), len(levels))
	}
	
	// 测试自定义级别
	customHook := NewDatabaseHook(ErrorLevel, PanicLevel)
	customLevels := customHook.Levels()
	expectedCustomLevels := []LogLevel{ErrorLevel, PanicLevel}
	
	if len(customLevels) != len(expectedCustomLevels) {
		t.Errorf("期望 %d 个自定义级别，实际 %d 个", len(expectedCustomLevels), len(customLevels))
	}
	
	// 测试触发钩子
	entry := &Entry{
		Level:   InfoLevel,
		Time:    time.Now(),
		Message: "test info",
		Context: context.Background(),
	}
	
	if err := hook.Fire(entry); err != nil {
		t.Errorf("触发数据库钩子失败: %v", err)
	}
}

func TestShouldFireHook(t *testing.T) {
	config := DevelopmentConfig()
	hook := NewTestHook(InfoLevel, ErrorLevel)
	
	hookLogger, err := NewWithHooks(config, hook)
	if err != nil {
		t.Fatalf("创建钩子日志器失败: %v", err)
	}
	defer hookLogger.Close()

	tests := []struct {
		level    LogLevel
		expected bool
	}{
		{DebugLevel, false},
		{InfoLevel, true},
		{WarnLevel, false},
		{ErrorLevel, true},
		{PanicLevel, false},
	}

	for _, tt := range tests {
		result := hookLogger.shouldFireHook(hook, tt.level)
		if result != tt.expected {
			t.Errorf("级别 %s 应该 %v, 实际 %v", tt.level, tt.expected, result)
		}
	}
	
	// 测试空级别钩子（应该对所有级别触发）
	emptyLevelsHook := NewTestHook() // 无级别参数
	for _, tt := range tests {
		result := hookLogger.shouldFireHook(emptyLevelsHook, tt.level)
		if !result {
			t.Errorf("空级别钩子对级别 %s 应该返回 true", tt.level)
		}
	}
}

func TestZapFieldsToFields(t *testing.T) {
	zapFields := []zap.Field{
		zap.String("string_key", "string_value"),
		zap.Int64("int64_key", 123),
		zap.Float64("float64_key", 3.14),
		zap.Bool("bool_key", true),
	}
	
	fields := zapFieldsToFields(zapFields)
	
	if len(fields) != len(zapFields) {
		t.Errorf("期望 %d 个字段，实际 %d 个", len(zapFields), len(fields))
	}
	
	if fields["string_key"] != "string_value" {
		t.Errorf("string_key 值不匹配: %v", fields["string_key"])
	}
	
	if fields["int64_key"] != int64(123) {
		t.Errorf("int64_key 值不匹配: %v", fields["int64_key"])
	}
	
	if fields["bool_key"] != true {
		t.Errorf("bool_key 值不匹配: %v", fields["bool_key"])
	}
}

// 基准测试
func BenchmarkHookLogger_Info(b *testing.B) {
	config := DefaultConfig()
	config.Console.Enabled = true // 保持控制台输出以满足配置要求
	config.Level = ErrorLevel // 设置高级别以减少输出
	
	hook := NewTestHook(InfoLevel)
	hookLogger, err := NewWithHooks(config, hook)
	if err != nil {
		b.Fatalf("创建钩子日志器失败: %v", err)
	}
	defer hookLogger.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hookLogger.Info("benchmark test message")
	}
}

func BenchmarkHookLogger_InfoWithoutHook(b *testing.B) {
	config := DefaultConfig()
	config.Console.Enabled = true // 保持控制台输出以满足配置要求
	config.Level = ErrorLevel // 设置高级别以减少输出
	
	hookLogger, err := NewWithHooks(config) // 无钩子
	if err != nil {
		b.Fatalf("创建钩子日志器失败: %v", err)
	}
	defer hookLogger.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hookLogger.Info("benchmark test message")
	}
}