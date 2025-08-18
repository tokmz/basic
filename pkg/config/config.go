package config

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Environment 环境类型
type Environment string

const (
	Development Environment = "dev"
	Testing     Environment = "test"
	Production  Environment = "prod"
)

// Config 配置管理器接口
type Config interface {
	// 基础配置操作
	Get(key string) interface{}
	GetString(key string) string
	GetInt(key string) int
	GetInt64(key string) int64
	GetFloat64(key string) float64
	GetBool(key string) bool
	GetStringSlice(key string) []string
	GetStringMap(key string) map[string]interface{}
	GetStringMapString(key string) map[string]string
	GetTime(key string) time.Time
	GetDuration(key string) time.Duration

	// 设置配置
	Set(key string, value interface{})
	SetDefault(key string, value interface{})

	// 结构体绑定
	Unmarshal(rawVal interface{}) error
	UnmarshalKey(key string, rawVal interface{}) error

	// 配置文件操作
	ReadInConfig() error
	WatchConfig()
	OnConfigChange(run func())

	// 远程配置
	AddRemoteProvider(provider, endpoint, path string) error
	ReadRemoteConfig() error
	WatchRemoteConfig() error

	// 调试和状态
	Debug() bool
	SetDebug(debug bool)
	GetConfigFile() string
	AllSettings() map[string]interface{}

	// 生命周期
	Close() error
}

// Manager 配置管理器实现
type Manager struct {
	viper           *viper.Viper
	mu              sync.RWMutex
	debug           bool
	env             Environment
	configPaths     []string
	configName      string
	configType      string
	remoteProviders []RemoteProvider
	watchers        []func()
	ctx             context.Context
	cancel          context.CancelFunc
}

// RemoteProvider 远程配置提供者
type RemoteProvider struct {
	Provider string
	Endpoint string
	Path     string
}

// Options 配置选项
type Options struct {
	Environment     Environment
	ConfigName      string
	ConfigType      string
	ConfigPaths     []string
	RemoteProviders []RemoteProvider
	Debug           bool
	Defaults        map[string]interface{}
	EnvPrefix       string
	EnvKeyReplacer  *strings.Replacer
}

// DefaultOptions 默认配置选项
func DefaultOptions() *Options {
	return &Options{
		Environment: Development,
		ConfigName:  "config",
		ConfigType:  "yaml",
		ConfigPaths: []string{
			".",
			"./config",
			"./configs",
			"$HOME/.config",
			"/etc",
		},
		Debug:          false,
		Defaults:       make(map[string]interface{}),
		EnvKeyReplacer: strings.NewReplacer(".", "_"),
	}
}

// NewManager 创建新的配置管理器
func NewManager(opts *Options) (Config, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	v := viper.New()
	ctx, cancel := context.WithCancel(context.Background())

	m := &Manager{
		viper:           v,
		debug:           opts.Debug,
		env:             opts.Environment,
		configPaths:     opts.ConfigPaths,
		configName:      opts.ConfigName,
		configType:      opts.ConfigType,
		remoteProviders: opts.RemoteProviders,
		watchers:        make([]func(), 0),
		ctx:             ctx,
		cancel:          cancel,
	}

	// 设置默认值
	for key, value := range opts.Defaults {
		m.viper.SetDefault(key, value)
	}

	// 配置环境变量
	m.setupEnv(opts)

	// 配置文件路径和类型
	m.setupConfig()

	// 添加远程配置提供者
	for _, provider := range opts.RemoteProviders {
		if err := m.AddRemoteProvider(provider.Provider, provider.Endpoint, provider.Path); err != nil {
			if m.debug {
				fmt.Printf("[CONFIG] Failed to add remote provider %s: %v\n", provider.Provider, err)
			}
		}
	}

	return m, nil
}

// setupEnv 设置环境变量配置
func (m *Manager) setupEnv(opts *Options) {
	m.viper.AutomaticEnv()

	if opts.EnvPrefix != "" {
		m.viper.SetEnvPrefix(opts.EnvPrefix)
	}

	if opts.EnvKeyReplacer != nil {
		m.viper.SetEnvKeyReplacer(opts.EnvKeyReplacer)
	}
}

// setupConfig 设置配置文件
func (m *Manager) setupConfig() {
	m.viper.SetConfigName(m.configName)
	m.viper.SetConfigType(m.configType)

	// 添加配置文件搜索路径
	for _, path := range m.configPaths {
		expandedPath := os.ExpandEnv(path)
		m.viper.AddConfigPath(expandedPath)
		if m.debug {
			fmt.Printf("[CONFIG] Added config path: %s\n", expandedPath)
		}
	}

	// 根据环境添加特定配置文件
	envConfigName := fmt.Sprintf("%s.%s", m.configName, m.env)
	m.viper.SetConfigName(envConfigName)
	if m.debug {
		fmt.Printf("[CONFIG] Looking for config file: %s.%s\n", envConfigName, m.configType)
	}
}

// Get 获取配置值
func (m *Manager) Get(key string) interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.Get(key)
}

// GetString 获取字符串配置
func (m *Manager) GetString(key string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetString(key)
}

// GetInt 获取整数配置
func (m *Manager) GetInt(key string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetInt(key)
}

// GetInt64 获取64位整数配置
func (m *Manager) GetInt64(key string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetInt64(key)
}

// GetFloat64 获取浮点数配置
func (m *Manager) GetFloat64(key string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetFloat64(key)
}

// GetBool 获取布尔配置
func (m *Manager) GetBool(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetBool(key)
}

// GetStringSlice 获取字符串切片配置
func (m *Manager) GetStringSlice(key string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetStringSlice(key)
}

// GetStringMap 获取字符串映射配置
func (m *Manager) GetStringMap(key string) map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetStringMap(key)
}

// GetStringMapString 获取字符串到字符串映射配置
func (m *Manager) GetStringMapString(key string) map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetStringMapString(key)
}

// GetTime 获取时间配置
func (m *Manager) GetTime(key string) time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetTime(key)
}

// GetDuration 获取时间间隔配置
func (m *Manager) GetDuration(key string) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetDuration(key)
}

// Set 设置配置值
func (m *Manager) Set(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.viper.Set(key, value)
}

// SetDefault 设置默认配置值
func (m *Manager) SetDefault(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.viper.SetDefault(key, value)
}

// Unmarshal 将配置绑定到结构体
func (m *Manager) Unmarshal(rawVal interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.Unmarshal(rawVal)
}

// UnmarshalKey 将指定键的配置绑定到结构体
func (m *Manager) UnmarshalKey(key string, rawVal interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.UnmarshalKey(key, rawVal)
}

// ReadInConfig 读取配置文件
func (m *Manager) ReadInConfig() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.debug {
		fmt.Printf("[CONFIG] Reading config file...\n")
	}

	err := m.viper.ReadInConfig()
	if err != nil {
		if m.debug {
			fmt.Printf("[CONFIG] Failed to read config file: %v\n", err)
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if m.debug {
		fmt.Printf("[CONFIG] Successfully loaded config file: %s\n", m.viper.ConfigFileUsed())
	}

	return nil
}

// WatchConfig 监听配置文件变化
func (m *Manager) WatchConfig() {
	m.viper.WatchConfig()
	m.viper.OnConfigChange(func(e fsnotify.Event) {
		if m.debug {
			fmt.Printf("[CONFIG] Config file changed: %s\n", e.Name)
		}

		// 执行所有注册的回调函数
		m.mu.RLock()
		watchers := make([]func(), len(m.watchers))
		copy(watchers, m.watchers)
		m.mu.RUnlock()

		for _, watcher := range watchers {
			go func(w func()) {
				defer func() {
					if r := recover(); r != nil {
						if m.debug {
							fmt.Printf("[CONFIG] Watcher panic: %v\n", r)
						}
					}
				}()
				w()
			}(watcher)
		}
	})
}

// OnConfigChange 注册配置变化回调
func (m *Manager) OnConfigChange(run func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.watchers = append(m.watchers, run)
}

// AddRemoteProvider 添加远程配置提供者
func (m *Manager) AddRemoteProvider(provider, endpoint, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.debug {
		fmt.Printf("[CONFIG] Adding remote provider: %s at %s\n", provider, endpoint)
	}

	err := m.viper.AddRemoteProvider(provider, endpoint, path)
	if err != nil {
		return fmt.Errorf("failed to add remote provider: %w", err)
	}

	m.remoteProviders = append(m.remoteProviders, RemoteProvider{
		Provider: provider,
		Endpoint: endpoint,
		Path:     path,
	})

	return nil
}

// ReadRemoteConfig 读取远程配置
func (m *Manager) ReadRemoteConfig() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.debug {
		fmt.Printf("[CONFIG] Reading remote config...\n")
	}

	err := m.viper.ReadRemoteConfig()
	if err != nil {
		if m.debug {
			fmt.Printf("[CONFIG] Failed to read remote config: %v\n", err)
		}
		return fmt.Errorf("failed to read remote config: %w", err)
	}

	if m.debug {
		fmt.Printf("[CONFIG] Successfully loaded remote config\n")
	}

	return nil
}

// WatchRemoteConfig 监听远程配置变化
func (m *Manager) WatchRemoteConfig() error {
	if m.debug {
		fmt.Printf("[CONFIG] Starting remote config watcher...\n")
	}

	err := m.viper.WatchRemoteConfig()
	if err != nil {
		return fmt.Errorf("failed to watch remote config: %w", err)
	}

	return nil
}

// Debug 获取调试模式状态
func (m *Manager) Debug() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.debug
}

// SetDebug 设置调试模式
func (m *Manager) SetDebug(debug bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.debug = debug
}

// GetConfigFile 获取当前使用的配置文件路径
func (m *Manager) GetConfigFile() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.ConfigFileUsed()
}

// AllSettings 获取所有配置设置
func (m *Manager) AllSettings() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.AllSettings()
}

// Close 关闭配置管理器
func (m *Manager) Close() error {
	if m.cancel != nil {
		m.cancel()
	}
	return nil
}
