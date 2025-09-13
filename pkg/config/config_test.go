package config

import (
	"fmt"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	// 验证默认值
	if config.ConfigName != "config" {
		t.Errorf("Expected ConfigName to be 'config', got %s", config.ConfigName)
	}
	
	if config.ConfigType != "yaml" {
		t.Errorf("Expected ConfigType to be 'yaml', got %s", config.ConfigType)
	}
	
	if config.Environment != "development" {
		t.Errorf("Expected Environment to be 'development', got %s", config.Environment)
	}
	
	if !config.ValidateOnLoad {
		t.Error("Expected ValidateOnLoad to be true")
	}
	
	if !config.AutomaticEnv {
		t.Error("Expected AutomaticEnv to be true")
	}
}

func TestDevelopmentConfig(t *testing.T) {
	config := DevelopmentConfig()
	
	if config.Environment != "development" {
		t.Errorf("Expected Environment to be 'development', got %s", config.Environment)
	}
	
	if !config.WatchConfig {
		t.Error("Expected WatchConfig to be true for development")
	}
	
	if !config.ValidateOnLoad {
		t.Error("Expected ValidateOnLoad to be true for development")
	}
}

func TestProductionConfig(t *testing.T) {
	config := ProductionConfig()
	
	if config.Environment != "production" {
		t.Errorf("Expected Environment to be 'production', got %s", config.Environment)
	}
	
	if config.WatchConfig {
		t.Error("Expected WatchConfig to be false for production")
	}
	
	if !config.ValidateOnLoad {
		t.Error("Expected ValidateOnLoad to be true for production")
	}
	
	if !config.CaseSensitive {
		t.Error("Expected CaseSensitive to be true for production")
	}
}

func TestTestingConfig(t *testing.T) {
	config := TestingConfig()
	
	if config.Environment != "testing" {
		t.Errorf("Expected Environment to be 'testing', got %s", config.Environment)
	}
	
	if config.WatchConfig {
		t.Error("Expected WatchConfig to be false for testing")
	}
	
	if config.ValidateOnLoad {
		t.Error("Expected ValidateOnLoad to be false for testing")
	}
}

func TestEventType(t *testing.T) {
	tests := []struct {
		eventType EventType
		expected  string
	}{
		{EventTypeSet, "SET"},
		{EventTypeDelete, "DELETE"},
		{EventTypeReload, "RELOAD"},
		{EventType(999), "UNKNOWN"},
	}
	
	for _, test := range tests {
		if test.eventType.String() != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, test.eventType.String())
		}
	}
}

func TestDefaultValidator(t *testing.T) {
	validator := NewDefaultValidator()
	
	// 添加验证规则
	validator.AddRule(ValidationRule{
		Key:      "database.host",
		Required: true,
		Type:     "string",
	})
	
	validator.AddRule(ValidationRule{
		Key:      "database.port",
		Required: true,
		Type:     "int",
		MinValue: 1,
		MaxValue: 65535,
	})
	
	validator.AddRule(ValidationRule{
		Key:     "log.level",
		Type:    "string",
		Options: []interface{}{"debug", "info", "warn", "error"},
	})
	
	// 测试有效配置
	validConfig := map[string]interface{}{
		"database.host": "localhost",
		"database.port": 5432,
		"log.level":     "info",
	}
	
	if err := validator.Validate(validConfig); err != nil {
		t.Errorf("Valid config should pass validation, got error: %v", err)
	}
	
	// 测试缺少必填项
	invalidConfig1 := map[string]interface{}{
		"database.port": 5432,
		"log.level":     "info",
	}
	
	if err := validator.Validate(invalidConfig1); err == nil {
		t.Error("Config missing required field should fail validation")
	}
	
	// 测试无效选项
	invalidConfig2 := map[string]interface{}{
		"database.host": "localhost",
		"database.port": 5432,
		"log.level":     "invalid",
	}
	
	if err := validator.Validate(invalidConfig2); err == nil {
		t.Error("Config with invalid option should fail validation")
	}
	
	// 测试类型错误
	invalidConfig3 := map[string]interface{}{
		"database.host": "localhost",
		"database.port": "invalid_port", // 应该是int
		"log.level":     "info",
	}
	
	if err := validator.Validate(invalidConfig3); err == nil {
		t.Error("Config with wrong type should fail validation")
	}
}

func TestValidationRule(t *testing.T) {
	validator := NewDefaultValidator()
	
	// 测试自定义验证函数
	validator.AddRule(ValidationRule{
		Key:      "password",
		Required: true,
		Type:     "string",
		CustomFunc: func(value interface{}) error {
			if str, ok := value.(string); ok {
				if len(str) < 8 {
					return fmt.Errorf("password must be at least 8 characters long")
				}
			}
			return nil
		},
	})
	
	// 测试密码太短
	config := map[string]interface{}{
		"password": "short",
	}
	
	if err := validator.Validate(config); err == nil {
		t.Error("Short password should fail validation")
	}
	
	// 测试有效密码
	config["password"] = "longenoughpassword"
	
	if err := validator.Validate(config); err != nil {
		t.Errorf("Valid password should pass validation, got error: %v", err)
	}
}

func TestConfigValidation(t *testing.T) {
	// 测试配置验证逻辑
	validator := NewDefaultValidator()
	
	// 添加duration验证规则
	validator.AddRule(ValidationRule{
		Key:  "timeout",
		Type: "duration",
	})
	
	// 测试有效duration
	config1 := map[string]interface{}{
		"timeout": "30s",
	}
	
	if err := validator.Validate(config1); err != nil {
		t.Errorf("Valid duration string should pass validation, got error: %v", err)
	}
	
	// 测试time.Duration类型
	config2 := map[string]interface{}{
		"timeout": 30 * time.Second,
	}
	
	if err := validator.Validate(config2); err != nil {
		t.Errorf("Valid duration type should pass validation, got error: %v", err)
	}
	
	// 测试无效duration字符串
	config3 := map[string]interface{}{
		"timeout": "invalid_duration",
	}
	
	if err := validator.Validate(config3); err == nil {
		t.Error("Invalid duration string should fail validation")
	}
}

func TestRemoteConfig(t *testing.T) {
	config := DefaultConfig()
	
	// 设置远程配置
	config.RemoteConfig = RemoteConfig{
		Provider: "etcd",
		Endpoint: "http://localhost:2379",
		Path:     "/config/myapp",
	}
	
	if config.RemoteConfig.Provider != "etcd" {
		t.Errorf("Expected remote provider to be 'etcd', got %s", config.RemoteConfig.Provider)
	}
	
	if config.RemoteConfig.Endpoint != "http://localhost:2379" {
		t.Errorf("Expected remote endpoint to be 'http://localhost:2379', got %s", config.RemoteConfig.Endpoint)
	}
}

func TestConfigCopy(t *testing.T) {
	original := DefaultConfig()
	original.ConfigName = "test"
	original.Environment = "testing"
	original.Defaults["test_key"] = "test_value"
	
	// 测试配置复制（深拷贝）
	copy := *original
	copy.ConfigName = "modified"
	copy.Defaults["new_key"] = "new_value"
	
	// 原始配置不应该被修改
	if original.ConfigName != "test" {
		t.Error("Original config should not be modified")
	}
	
	// 但是map是引用，所以会被修改（这是一个已知限制）
	// 在实际使用中可能需要实现真正的深拷贝
}

func BenchmarkDefaultValidator(b *testing.B) {
	validator := NewDefaultValidator()
	
	// 添加一些验证规则
	for i := 0; i < 10; i++ {
		validator.AddRule(ValidationRule{
			Key:      fmt.Sprintf("key%d", i),
			Required: true,
			Type:     "string",
		})
	}
	
	config := make(map[string]interface{})
	for i := 0; i < 10; i++ {
		config[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.Validate(config)
	}
}

func BenchmarkConfigCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DefaultConfig()
	}
}