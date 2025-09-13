package config

import (
	"fmt"
	"sync"
	"time"
)

// Manager 配置管理器接口
type Manager interface {
	// Load 加载配置
	Load() error
	
	// Reload 重新加载配置
	Reload() error
	
	// Get 获取配置值
	Get(key string) interface{}
	
	// GetString 获取字符串配置
	GetString(key string) string
	
	// GetInt 获取整数配置
	GetInt(key string) int
	
	// GetBool 获取布尔配置
	GetBool(key string) bool
	
	// GetFloat64 获取浮点数配置
	GetFloat64(key string) float64
	
	// GetDuration 获取时间间隔配置
	GetDuration(key string) time.Duration
	
	// GetStringSlice 获取字符串数组配置
	GetStringSlice(key string) []string
	
	// Set 设置配置值
	Set(key string, value interface{})
	
	// IsSet 检查配置是否存在
	IsSet(key string) bool
	
	// Unmarshal 将配置解析到结构体
	Unmarshal(rawVal interface{}) error
	
	// UnmarshalKey 将指定键的配置解析到结构体
	UnmarshalKey(key string, rawVal interface{}) error
	
	// Watch 监听配置变化
	Watch(callback func(event Event)) error
	
	// StopWatch 停止监听配置变化
	StopWatch()
	
	// Validate 验证配置
	Validate(validator Validator) error
	
	// GetAllSettings 获取所有配置
	GetAllSettings() map[string]interface{}
	
	// Close 关闭配置管理器
	Close() error
}

// Config 配置管理器配置
type Config struct {
	// 配置文件相关
	ConfigFile   string   `json:"config_file" yaml:"config_file"`       // 配置文件路径
	ConfigPaths  []string `json:"config_paths" yaml:"config_paths"`     // 配置文件搜索路径
	ConfigName   string   `json:"config_name" yaml:"config_name"`       // 配置文件名（不含扩展名）
	ConfigType   string   `json:"config_type" yaml:"config_type"`       // 配置文件类型
	
	// 环境变量相关
	EnvPrefix    string `json:"env_prefix" yaml:"env_prefix"`           // 环境变量前缀
	EnvKeyReplacer map[string]string `json:"env_key_replacer" yaml:"env_key_replacer"` // 环境变量键替换规则
	
	// 远程配置相关
	RemoteConfig RemoteConfig `json:"remote_config" yaml:"remote_config"`
	
	// 监听配置
	WatchConfig  bool `json:"watch_config" yaml:"watch_config"`        // 是否监听配置文件变化
	
	// 验证相关
	ValidateOnLoad bool `json:"validate_on_load" yaml:"validate_on_load"` // 加载时是否验证
	
	// 加密相关
	EncryptionKey string `json:"encryption_key" yaml:"encryption_key"`  // 加密密钥
	
	// 默认值
	Defaults     map[string]interface{} `json:"defaults" yaml:"defaults"`
	
	// 环境相关
	Environment  string `json:"environment" yaml:"environment"`         // 当前环境
	
	// 其他选项
	AllowEmptyEnv     bool `json:"allow_empty_env" yaml:"allow_empty_env"`         // 是否允许空环境变量
	AutomaticEnv      bool `json:"automatic_env" yaml:"automatic_env"`             // 是否自动读取环境变量
	CaseSensitive     bool `json:"case_sensitive" yaml:"case_sensitive"`           // 是否区分大小写
}

// RemoteConfig 远程配置
type RemoteConfig struct {
	Provider string `json:"provider" yaml:"provider"` // etcd, consul, firestore
	Endpoint string `json:"endpoint" yaml:"endpoint"`
	Path     string `json:"path" yaml:"path"`
	SecretKey string `json:"secret_key" yaml:"secret_key"`
}

// Event 配置变化事件
type Event struct {
	Type      EventType   `json:"type"`
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	OldValue  interface{} `json:"old_value"`
	Timestamp time.Time   `json:"timestamp"`
}

// EventType 事件类型
type EventType int

const (
	EventTypeSet EventType = iota
	EventTypeDelete
	EventTypeReload
)

func (e EventType) String() string {
	switch e {
	case EventTypeSet:
		return "SET"
	case EventTypeDelete:
		return "DELETE"
	case EventTypeReload:
		return "RELOAD"
	default:
		return "UNKNOWN"
	}
}

// Validator 配置验证器接口
type Validator interface {
	Validate(config map[string]interface{}) error
}

// ValidationRule 验证规则
type ValidationRule struct {
	Key        string      `json:"key"`
	Required   bool        `json:"required"`
	Type       string      `json:"type"`       // string, int, bool, float, duration
	MinValue   interface{} `json:"min_value"`
	MaxValue   interface{} `json:"max_value"`
	Pattern    string      `json:"pattern"`    // 正则表达式
	Options    []interface{} `json:"options"`  // 可选值列表
	CustomFunc func(interface{}) error `json:"-"` // 自定义验证函数
}

// DefaultValidator 默认验证器
type DefaultValidator struct {
	Rules []ValidationRule `json:"rules"`
	mu    sync.RWMutex
}

// NewDefaultValidator 创建默认验证器
func NewDefaultValidator() *DefaultValidator {
	return &DefaultValidator{
		Rules: make([]ValidationRule, 0),
	}
}

// AddRule 添加验证规则
func (v *DefaultValidator) AddRule(rule ValidationRule) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.Rules = append(v.Rules, rule)
}

// Validate 验证配置
func (v *DefaultValidator) Validate(config map[string]interface{}) error {
	v.mu.RLock()
	defer v.mu.RUnlock()
	
	for _, rule := range v.Rules {
		if err := v.validateRule(rule, config); err != nil {
			return fmt.Errorf("validation failed for key '%s': %w", rule.Key, err)
		}
	}
	
	return nil
}

func (v *DefaultValidator) validateRule(rule ValidationRule, config map[string]interface{}) error {
	value, exists := config[rule.Key]
	
	// 检查必填项
	if rule.Required && !exists {
		return fmt.Errorf("required key missing")
	}
	
	if !exists {
		return nil // 非必填项不存在时跳过
	}
	
	// 类型检查
	if err := v.validateType(rule.Type, value); err != nil {
		return err
	}
	
	// 范围检查
	if err := v.validateRange(rule, value); err != nil {
		return err
	}
	
	// 选项检查
	if err := v.validateOptions(rule, value); err != nil {
		return err
	}
	
	// 自定义验证
	if rule.CustomFunc != nil {
		if err := rule.CustomFunc(value); err != nil {
			return err
		}
	}
	
	return nil
}

func (v *DefaultValidator) validateType(expectedType string, value interface{}) error {
	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "int":
		if _, ok := value.(int); !ok {
			return fmt.Errorf("expected int, got %T", value)
		}
	case "bool":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected bool, got %T", value)
		}
	case "float":
		if _, ok := value.(float64); !ok {
			return fmt.Errorf("expected float64, got %T", value)
		}
	case "duration":
		if _, ok := value.(time.Duration); !ok {
			// 尝试解析字符串形式的duration
			if strVal, ok := value.(string); ok {
				if _, err := time.ParseDuration(strVal); err != nil {
					return fmt.Errorf("expected duration, got invalid string: %v", err)
				}
			} else {
				return fmt.Errorf("expected duration, got %T", value)
			}
		}
	}
	return nil
}

func (v *DefaultValidator) validateRange(rule ValidationRule, value interface{}) error {
	// TODO: 实现范围验证逻辑
	return nil
}

func (v *DefaultValidator) validateOptions(rule ValidationRule, value interface{}) error {
	if len(rule.Options) == 0 {
		return nil
	}
	
	for _, option := range rule.Options {
		if value == option {
			return nil
		}
	}
	
	return fmt.Errorf("value %v not in allowed options %v", value, rule.Options)
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		ConfigName:     "config",
		ConfigType:     "yaml",
		ConfigPaths:    []string{".", "./config", "./configs"},
		EnvPrefix:      "",
		WatchConfig:    false,
		ValidateOnLoad: true,
		AutomaticEnv:   true,
		CaseSensitive:  false,
		AllowEmptyEnv:  false,
		Environment:    "development",
		Defaults:       make(map[string]interface{}),
	}
}

// DevelopmentConfig 返回开发环境配置
func DevelopmentConfig() *Config {
	config := DefaultConfig()
	config.Environment = "development"
	config.WatchConfig = true
	config.ValidateOnLoad = true
	return config
}

// ProductionConfig 返回生产环境配置
func ProductionConfig() *Config {
	config := DefaultConfig()
	config.Environment = "production"
	config.WatchConfig = false
	config.ValidateOnLoad = true
	config.CaseSensitive = true
	return config
}

// TestingConfig 返回测试环境配置
func TestingConfig() *Config {
	config := DefaultConfig()
	config.Environment = "testing"
	config.WatchConfig = false
	config.ValidateOnLoad = false
	return config
}