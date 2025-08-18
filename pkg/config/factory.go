package config

import (
	"fmt"
	"sync"
)

// Factory 配置工厂接口
type Factory interface {
	// CreateConfig 创建指定环境的配置实例
	CreateConfig(env Environment, opts *Options) (Config, error)
	// GetConfig 获取已创建的配置实例
	GetConfig(env Environment) (Config, bool)
	// ListEnvironments 列出所有已创建的环境
	ListEnvironments() []Environment
	// CloseAll 关闭所有配置实例
	CloseAll() error
}

// ConfigFactory 配置工厂实现
type ConfigFactory struct {
	mu          sync.RWMutex
	configs     map[Environment]Config
	defaultOpts *Options
}

// NewFactory 创建新的配置工厂
func NewFactory(defaultOpts *Options) Factory {
	if defaultOpts == nil {
		defaultOpts = DefaultOptions()
	}

	return &ConfigFactory{
		configs:     make(map[Environment]Config),
		defaultOpts: defaultOpts,
	}
}

// CreateConfig 创建指定环境的配置实例
func (f *ConfigFactory) CreateConfig(env Environment, opts *Options) (Config, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// 检查是否已存在
	if existing, exists := f.configs[env]; exists {
		return existing, nil
	}

	// 合并选项
	finalOpts := f.mergeOptions(env, opts)

	// 创建配置实例
	config, err := NewManager(finalOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create config for environment %s: %w", env, err)
	}

	// 尝试读取配置文件
	if err := config.ReadInConfig(); err != nil {
		if finalOpts.Debug {
			fmt.Printf("[FACTORY] Warning: failed to read config file for %s: %v\n", env, err)
		}
	}

	// 尝试读取远程配置
	if len(finalOpts.RemoteProviders) > 0 {
		if err := config.ReadRemoteConfig(); err != nil {
			if finalOpts.Debug {
				fmt.Printf("[FACTORY] Warning: failed to read remote config for %s: %v\n", env, err)
			}
		}
	}

	f.configs[env] = config

	if finalOpts.Debug {
		fmt.Printf("[FACTORY] Created config instance for environment: %s\n", env)
	}

	return config, nil
}

// GetConfig 获取已创建的配置实例
func (f *ConfigFactory) GetConfig(env Environment) (Config, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	config, exists := f.configs[env]
	return config, exists
}

// ListEnvironments 列出所有已创建的环境
func (f *ConfigFactory) ListEnvironments() []Environment {
	f.mu.RLock()
	defer f.mu.RUnlock()

	envs := make([]Environment, 0, len(f.configs))
	for env := range f.configs {
		envs = append(envs, env)
	}

	return envs
}

// CloseAll 关闭所有配置实例
func (f *ConfigFactory) CloseAll() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	var lastErr error
	for env, config := range f.configs {
		if err := config.Close(); err != nil {
			lastErr = err
			if f.defaultOpts.Debug {
				fmt.Printf("[FACTORY] Failed to close config for %s: %v\n", env, err)
			}
		}
	}

	// 清空配置映射
	f.configs = make(map[Environment]Config)

	return lastErr
}

// mergeOptions 合并配置选项
func (f *ConfigFactory) mergeOptions(env Environment, opts *Options) *Options {
	// 从默认选项开始
	finalOpts := &Options{
		Environment:     env,
		ConfigName:      f.defaultOpts.ConfigName,
		ConfigType:      f.defaultOpts.ConfigType,
		ConfigPaths:     make([]string, len(f.defaultOpts.ConfigPaths)),
		RemoteProviders: make([]RemoteProvider, len(f.defaultOpts.RemoteProviders)),
		Debug:           f.defaultOpts.Debug,
		Defaults:        make(map[string]interface{}),
		EnvPrefix:       f.defaultOpts.EnvPrefix,
		EnvKeyReplacer:  f.defaultOpts.EnvKeyReplacer,
	}

	// 复制默认值
	copy(finalOpts.ConfigPaths, f.defaultOpts.ConfigPaths)
	copy(finalOpts.RemoteProviders, f.defaultOpts.RemoteProviders)
	for k, v := range f.defaultOpts.Defaults {
		finalOpts.Defaults[k] = v
	}

	// 如果提供了自定义选项，则覆盖默认值
	if opts != nil {
		if opts.ConfigName != "" {
			finalOpts.ConfigName = opts.ConfigName
		}
		if opts.ConfigType != "" {
			finalOpts.ConfigType = opts.ConfigType
		}
		if len(opts.ConfigPaths) > 0 {
			finalOpts.ConfigPaths = opts.ConfigPaths
		}
		if len(opts.RemoteProviders) > 0 {
			finalOpts.RemoteProviders = opts.RemoteProviders
		}
		if opts.EnvPrefix != "" {
			finalOpts.EnvPrefix = opts.EnvPrefix
		}
		if opts.EnvKeyReplacer != nil {
			finalOpts.EnvKeyReplacer = opts.EnvKeyReplacer
		}
		// 合并默认值
		for k, v := range opts.Defaults {
			finalOpts.Defaults[k] = v
		}
		// 调试模式可以被覆盖
		finalOpts.Debug = opts.Debug
	}

	return finalOpts
}

// 全局工厂实例
var (
	globalFactory Factory
	factoryOnce   sync.Once
)

// GetGlobalFactory 获取全局配置工厂实例
func GetGlobalFactory() Factory {
	factoryOnce.Do(func() {
		globalFactory = NewFactory(DefaultOptions())
	})
	return globalFactory
}

// CreateGlobalConfig 使用全局工厂创建配置实例
func CreateGlobalConfig(env Environment, opts *Options) (Config, error) {
	return GetGlobalFactory().CreateConfig(env, opts)
}

// GetGlobalConfig 获取全局配置实例
func GetGlobalConfig(env Environment) (Config, bool) {
	return GetGlobalFactory().GetConfig(env)
}

// CloseGlobalConfigs 关闭所有全局配置实例
func CloseGlobalConfigs() error {
	return GetGlobalFactory().CloseAll()
}
