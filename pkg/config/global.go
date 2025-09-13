package config

import (
	"fmt"
	"sync"
	"time"
)

// 全局配置管理器实例
var (
	globalManager Manager
	globalOnce    sync.Once
	globalMu      sync.RWMutex
)

// Init 初始化全局配置管理器
func Init(config *Config) error {
	var err error
	globalOnce.Do(func() {
		globalManager = NewViperManager(config)
		err = globalManager.Load()
	})
	return err
}

// InitWithEncryption 使用加密功能初始化全局配置管理器
func InitWithEncryption(config *Config, encryptedKeys []string) error {
	var err error
	globalOnce.Do(func() {
		var mgr *EncryptedViperManager
		mgr, err = NewEncryptedViperManager(config, encryptedKeys)
		if err != nil {
			return
		}
		globalManager = mgr
		err = globalManager.Load()
	})
	return err
}

// InitWithEnvironment 使用环境管理器初始化全局配置
func InitWithEnvironment(env Environment, baseConfig *Config) error {
	var err error
	globalOnce.Do(func() {
		envManager := NewEnvironmentManager(env)
		globalManager, err = envManager.LoadWithEnvironment(env, baseConfig)
	})
	return err
}

// Global 获取全局配置管理器
func Global() Manager {
	globalMu.RLock()
	defer globalMu.RUnlock()
	
	if globalManager == nil {
		// 如果没有初始化，使用默认配置
		_ = Init(DefaultConfig())
	}
	return globalManager
}

// SetGlobal 设置全局配置管理器
func SetGlobal(manager Manager) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalManager = manager
}

// Reload 重新加载全局配置
func Reload() error {
	return Global().Reload()
}

// Get 获取配置值
func Get(key string) interface{} {
	return Global().Get(key)
}

// GetString 获取字符串配置
func GetString(key string) string {
	return Global().GetString(key)
}

// GetInt 获取整数配置
func GetInt(key string) int {
	return Global().GetInt(key)
}

// GetBool 获取布尔配置
func GetBool(key string) bool {
	return Global().GetBool(key)
}

// GetFloat64 获取浮点数配置
func GetFloat64(key string) float64 {
	return Global().GetFloat64(key)
}

// GetDuration 获取时间间隔配置
func GetDuration(key string) time.Duration {
	return Global().GetDuration(key)
}

// GetStringSlice 获取字符串数组配置
func GetStringSlice(key string) []string {
	return Global().GetStringSlice(key)
}

// Set 设置配置值
func Set(key string, value interface{}) {
	Global().Set(key, value)
}

// IsSet 检查配置是否存在
func IsSet(key string) bool {
	return Global().IsSet(key)
}

// Unmarshal 将配置解析到结构体
func Unmarshal(rawVal interface{}) error {
	return Global().Unmarshal(rawVal)
}

// UnmarshalKey 将指定键的配置解析到结构体
func UnmarshalKey(key string, rawVal interface{}) error {
	return Global().UnmarshalKey(key, rawVal)
}

// Watch 监听配置变化
func Watch(callback func(event Event)) error {
	return Global().Watch(callback)
}

// StopWatch 停止监听配置变化
func StopWatch() {
	Global().StopWatch()
}

// Validate 验证配置
func Validate(validator Validator) error {
	return Global().Validate(validator)
}

// GetAllSettings 获取所有配置
func GetAllSettings() map[string]interface{} {
	return Global().GetAllSettings()
}

// Close 关闭全局配置管理器
func Close() error {
	if globalManager != nil {
		return globalManager.Close()
	}
	return nil
}

// ConfigBuilder 配置构建器
type ConfigBuilder struct {
	config         *Config
	validator      Validator
	encryptedKeys  []string
	environment    Environment
	customSettings map[string]interface{}
}

// NewConfigBuilder 创建配置构建器
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config:         DefaultConfig(),
		customSettings: make(map[string]interface{}),
	}
}

// WithConfig 设置基础配置
func (b *ConfigBuilder) WithConfig(config *Config) *ConfigBuilder {
	b.config = config
	return b
}

// WithConfigFile 设置配置文件
func (b *ConfigBuilder) WithConfigFile(file string) *ConfigBuilder {
	b.config.ConfigFile = file
	return b
}

// WithConfigName 设置配置文件名
func (b *ConfigBuilder) WithConfigName(name string) *ConfigBuilder {
	b.config.ConfigName = name
	return b
}

// WithConfigType 设置配置文件类型
func (b *ConfigBuilder) WithConfigType(configType string) *ConfigBuilder {
	b.config.ConfigType = configType
	return b
}

// WithConfigPaths 设置配置文件搜索路径
func (b *ConfigBuilder) WithConfigPaths(paths ...string) *ConfigBuilder {
	b.config.ConfigPaths = append(b.config.ConfigPaths, paths...)
	return b
}

// WithEnvironment 设置环境
func (b *ConfigBuilder) WithEnvironment(env Environment) *ConfigBuilder {
	b.environment = env
	b.config.Environment = env.String()
	return b
}

// WithEnvPrefix 设置环境变量前缀
func (b *ConfigBuilder) WithEnvPrefix(prefix string) *ConfigBuilder {
	b.config.EnvPrefix = prefix
	return b
}

// WithEncryption 设置加密配置
func (b *ConfigBuilder) WithEncryption(key string, encryptedKeys ...string) *ConfigBuilder {
	b.config.EncryptionKey = key
	b.encryptedKeys = encryptedKeys
	return b
}

// WithValidator 设置验证器
func (b *ConfigBuilder) WithValidator(validator Validator) *ConfigBuilder {
	b.validator = validator
	return b
}

// WithWatchConfig 设置配置文件监听
func (b *ConfigBuilder) WithWatchConfig(watch bool) *ConfigBuilder {
	b.config.WatchConfig = watch
	return b
}

// WithValidateOnLoad 设置加载时验证
func (b *ConfigBuilder) WithValidateOnLoad(validate bool) *ConfigBuilder {
	b.config.ValidateOnLoad = validate
	return b
}

// WithDefault 设置默认值
func (b *ConfigBuilder) WithDefault(key string, value interface{}) *ConfigBuilder {
	b.config.Defaults[key] = value
	return b
}

// WithDefaults 批量设置默认值
func (b *ConfigBuilder) WithDefaults(defaults map[string]interface{}) *ConfigBuilder {
	for key, value := range defaults {
		b.config.Defaults[key] = value
	}
	return b
}

// WithCustomSetting 设置自定义配置
func (b *ConfigBuilder) WithCustomSetting(key string, value interface{}) *ConfigBuilder {
	b.customSettings[key] = value
	return b
}

// WithRemoteConfig 设置远程配置
func (b *ConfigBuilder) WithRemoteConfig(provider, endpoint, path string) *ConfigBuilder {
	b.config.RemoteConfig = RemoteConfig{
		Provider: provider,
		Endpoint: endpoint,
		Path:     path,
	}
	return b
}

// Build 构建配置管理器
func (b *ConfigBuilder) Build() (Manager, error) {
	var manager Manager
	var err error
	
	// 根据是否有加密需求选择管理器类型
	if len(b.encryptedKeys) > 0 {
		var encryptedManager *EncryptedViperManager
		encryptedManager, err = NewEncryptedViperManager(b.config, b.encryptedKeys)
		if err != nil {
			return nil, fmt.Errorf("failed to create encrypted manager: %w", err)
		}
		manager = encryptedManager
	} else if b.environment != "" {
		envManager := NewEnvironmentManager(b.environment)
		manager, err = envManager.LoadWithEnvironment(b.environment, b.config)
		if err != nil {
			return nil, fmt.Errorf("failed to create environment manager: %w", err)
		}
	} else {
		manager = NewViperManager(b.config)
	}
	
	// 加载配置
	if err := manager.Load(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	
	// 应用自定义设置
	for key, value := range b.customSettings {
		manager.Set(key, value)
	}
	
	// 应用验证器
	if b.validator != nil {
		if err := manager.Validate(b.validator); err != nil {
			return nil, fmt.Errorf("config validation failed: %w", err)
		}
	}
	
	return manager, nil
}

// BuildAndSetGlobal 构建并设置为全局配置管理器
func (b *ConfigBuilder) BuildAndSetGlobal() error {
	manager, err := b.Build()
	if err != nil {
		return err
	}
	
	SetGlobal(manager)
	return nil
}

// QuickStart 快速启动配置
// 这是一个便捷函数，用于快速创建和初始化配置管理器
func QuickStart(options ...func(*ConfigBuilder)) (Manager, error) {
	builder := NewConfigBuilder()
	
	for _, option := range options {
		option(builder)
	}
	
	return builder.Build()
}

// 便捷的配置选项函数

// WithYAMLFile 使用YAML配置文件
func WithYAMLFile(filename string) func(*ConfigBuilder) {
	return func(b *ConfigBuilder) {
		b.WithConfigFile(filename).WithConfigType("yaml")
	}
}

// WithJSONFile 使用JSON配置文件
func WithJSONFile(filename string) func(*ConfigBuilder) {
	return func(b *ConfigBuilder) {
		b.WithConfigFile(filename).WithConfigType("json")
	}
}

// WithTOMLFile 使用TOML配置文件
func WithTOMLFile(filename string) func(*ConfigBuilder) {
	return func(b *ConfigBuilder) {
		b.WithConfigFile(filename).WithConfigType("toml")
	}
}

// WithDevelopment 开发环境配置
func WithDevelopment() func(*ConfigBuilder) {
	return func(b *ConfigBuilder) {
		b.WithEnvironment(Development).WithWatchConfig(true)
	}
}

// WithProduction 生产环境配置
func WithProduction() func(*ConfigBuilder) {
	return func(b *ConfigBuilder) {
		b.WithEnvironment(Production).WithWatchConfig(false).WithValidateOnLoad(true)
	}
}

// WithTesting 测试环境配置
func WithTesting() func(*ConfigBuilder) {
	return func(b *ConfigBuilder) {
		b.WithEnvironment(Testing).WithWatchConfig(false).WithValidateOnLoad(false)
	}
}