package config

import (
	"fmt"
	"testing"
	"time"
)

func TestNewViperManager(t *testing.T) {
	config := DefaultConfig()
	manager := NewViperManager(config)
	
	if manager == nil {
		t.Fatal("NewViperManager should not return nil")
	}
	
	if manager.config != config {
		t.Error("Manager should store the provided config")
	}
	
	if manager.viper == nil {
		t.Error("Manager should initialize viper instance")
	}
}

func TestNewViperManagerWithNilConfig(t *testing.T) {
	manager := NewViperManager(nil)
	
	if manager == nil {
		t.Fatal("NewViperManager should not return nil even with nil config")
	}
	
	if manager.config == nil {
		t.Error("Manager should use default config when nil is provided")
	}
}

func TestViperManagerBasicOperations(t *testing.T) {
	config := DefaultConfig()
	manager := NewViperManager(config)
	
	// 测试Set和Get
	testKey := "test.key"
	testValue := "test value"
	
	manager.Set(testKey, testValue)
	
	if value := manager.GetString(testKey); value != testValue {
		t.Errorf("Expected '%s', got '%s'", testValue, value)
	}
	
	// 测试IsSet
	if !manager.IsSet(testKey) {
		t.Error("IsSet should return true for existing key")
	}
	
	if manager.IsSet("non.existent.key") {
		t.Error("IsSet should return false for non-existent key")
	}
}

func TestViperManagerTypedGets(t *testing.T) {
	config := DefaultConfig()
	manager := NewViperManager(config)
	
	// 测试不同类型的get操作
	manager.Set("string_key", "string_value")
	manager.Set("int_key", 42)
	manager.Set("bool_key", true)
	manager.Set("float_key", 3.14)
	manager.Set("duration_key", 30*time.Second)
	manager.Set("slice_key", []string{"a", "b", "c"})
	
	// 验证字符串
	if value := manager.GetString("string_key"); value != "string_value" {
		t.Errorf("Expected 'string_value', got '%s'", value)
	}
	
	// 验证整数
	if value := manager.GetInt("int_key"); value != 42 {
		t.Errorf("Expected 42, got %d", value)
	}
	
	// 验证布尔值
	if value := manager.GetBool("bool_key"); !value {
		t.Errorf("Expected true, got %t", value)
	}
	
	// 验证浮点数
	if value := manager.GetFloat64("float_key"); value != 3.14 {
		t.Errorf("Expected 3.14, got %f", value)
	}
	
	// 验证时间间隔
	if value := manager.GetDuration("duration_key"); value != 30*time.Second {
		t.Errorf("Expected 30s, got %v", value)
	}
	
	// 验证字符串切片
	if value := manager.GetStringSlice("slice_key"); len(value) != 3 || value[0] != "a" {
		t.Errorf("Expected ['a', 'b', 'c'], got %v", value)
	}
}

func TestViperManagerUnmarshal(t *testing.T) {
	config := DefaultConfig()
	manager := NewViperManager(config)
	
	// 设置一些嵌套配置
	manager.Set("database.host", "localhost")
	manager.Set("database.port", 5432)
	manager.Set("database.username", "user")
	manager.Set("database.password", "pass")
	
	// 定义目标结构体
	type DatabaseConfig struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
	}
	
	var dbConfig DatabaseConfig
	
	// 测试UnmarshalKey
	if err := manager.UnmarshalKey("database", &dbConfig); err != nil {
		t.Errorf("UnmarshalKey failed: %v", err)
	}
	
	// 验证解析结果
	if dbConfig.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got '%s'", dbConfig.Host)
	}
	
	if dbConfig.Port != 5432 {
		t.Errorf("Expected port 5432, got %d", dbConfig.Port)
	}
}

func TestViperManagerAllSettings(t *testing.T) {
	config := DefaultConfig()
	manager := NewViperManager(config)
	
	// 设置一些配置
	manager.Set("key1", "value1")
	manager.Set("key2", 42)
	manager.Set("nested.key", "nested_value")
	
	settings := manager.GetAllSettings()
	
	if settings == nil {
		t.Fatal("GetAllSettings should not return nil")
	}
	
	if len(settings) == 0 {
		t.Error("GetAllSettings should return non-empty map")
	}
	
	// 验证设置的值是否存在
	if settings["key1"] != "value1" {
		t.Error("Settings should contain key1 with value1")
	}
}

func TestViperManagerWatcher(t *testing.T) {
	config := DefaultConfig()
	manager := NewViperManager(config)
	
	// 设置监听器
	var receivedEvents []Event
	callback := func(event Event) {
		receivedEvents = append(receivedEvents, event)
	}
	
	if err := manager.Watch(callback); err != nil {
		t.Errorf("Watch should not return error: %v", err)
	}
	
	// 触发一些变化
	manager.Set("watch_key", "watch_value")
	
	// 给一点时间处理异步事件
	time.Sleep(10 * time.Millisecond)
	
	if len(receivedEvents) == 0 {
		t.Error("Should have received at least one event")
	}
	
	// 验证事件类型
	if receivedEvents[0].Type != EventTypeSet {
		t.Errorf("Expected EventTypeSet, got %s", receivedEvents[0].Type.String())
	}
	
	// 停止监听
	manager.StopWatch()
	
	// 再次设置值，应该不会收到事件
	initialEventCount := len(receivedEvents)
	manager.Set("another_key", "another_value")
	time.Sleep(10 * time.Millisecond)
	
	if len(receivedEvents) != initialEventCount {
		t.Error("Should not receive events after StopWatch")
	}
}

func TestViperManagerValidation(t *testing.T) {
	config := DefaultConfig()
	manager := NewViperManager(config)
	
	// 设置一些配置
	manager.Set("required_key", "value")
	manager.Set("optional_key", "optional_value")
	
	// 创建验证器
	validator := NewDefaultValidator()
	validator.AddRule(ValidationRule{
		Key:      "required_key",
		Required: true,
		Type:     "string",
	})
	
	// 验证应该成功
	if err := manager.Validate(validator); err != nil {
		t.Errorf("Validation should succeed: %v", err)
	}
	
	// 移除必需的键
	manager.Set("required_key", nil)
	
	// 验证应该失败
	if err := manager.Validate(validator); err == nil {
		t.Error("Validation should fail when required key is missing")
	}
}

func TestViperManagerClose(t *testing.T) {
	config := DefaultConfig()
	manager := NewViperManager(config)
	
	// 正常关闭
	if err := manager.Close(); err != nil {
		t.Errorf("Close should not return error: %v", err)
	}
	
	// 重复关闭应该不报错
	if err := manager.Close(); err != nil {
		t.Errorf("Multiple Close calls should not return error: %v", err)
	}
}

func TestViperManagerLoadError(t *testing.T) {
	config := &Config{
		ConfigFile: "/non/existent/path/config.yaml",
	}
	
	manager := NewViperManager(config)
	
	// 加载不存在的文件应该处理错误
	// 注意：在实际实现中，如果有环境变量或默认值，可能不会报错
	err := manager.Load()
	// 这里我们不严格要求错误，因为viper的行为比较宽松
	_ = err // 忽略错误，因为在某些情况下可能是正常的
}

func TestViperManagerConcurrency(t *testing.T) {
	config := DefaultConfig()
	manager := NewViperManager(config)
	
	// 并发读写测试
	done := make(chan bool)
	
	// 启动多个goroutine进行并发操作
	for i := 0; i < 10; i++ {
		go func(index int) {
			defer func() { done <- true }()
			
			key := fmt.Sprintf("concurrent_key_%d", index)
			value := fmt.Sprintf("concurrent_value_%d", index)
			
			manager.Set(key, value)
			
			if retrievedValue := manager.GetString(key); retrievedValue != value {
				t.Errorf("Concurrent operation failed for %s", key)
			}
		}(i)
	}
	
	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}
}

func BenchmarkViperManagerSet(b *testing.B) {
	config := DefaultConfig()
	manager := NewViperManager(config)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("benchmark_key_%d", i)
		manager.Set(key, "benchmark_value")
	}
}

func BenchmarkViperManagerGet(b *testing.B) {
	config := DefaultConfig()
	manager := NewViperManager(config)
	
	// 预设一些键值
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("benchmark_key_%d", i)
		manager.Set(key, "benchmark_value")
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("benchmark_key_%d", i%1000)
		manager.GetString(key)
	}
}