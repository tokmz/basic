package config

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// ViperManager 基于Viper的配置管理器
type ViperManager struct {
	viper     *viper.Viper
	config    *Config
	validator Validator
	watchers  []func(event Event)
	mu        sync.RWMutex
	closed    bool
}

// NewViperManager 创建新的Viper配置管理器
func NewViperManager(config *Config) *ViperManager {
	if config == nil {
		config = DefaultConfig()
	}

	v := viper.New()
	
	manager := &ViperManager{
		viper:    v,
		config:   config,
		watchers: make([]func(event Event), 0),
	}

	// 应用配置
	manager.applyConfig()

	return manager
}

// applyConfig 应用配置到viper实例
func (m *ViperManager) applyConfig() {
	// 设置配置文件
	if m.config.ConfigFile != "" {
		m.viper.SetConfigFile(m.config.ConfigFile)
	} else {
		m.viper.SetConfigName(m.config.ConfigName)
		m.viper.SetConfigType(m.config.ConfigType)
	}

	// 添加配置文件搜索路径
	for _, path := range m.config.ConfigPaths {
		m.viper.AddConfigPath(path)
	}

	// 设置环境变量
	if m.config.AutomaticEnv {
		m.viper.AutomaticEnv()
	}

	if m.config.EnvPrefix != "" {
		m.viper.SetEnvPrefix(m.config.EnvPrefix)
	}

	// 设置环境变量键替换
	if len(m.config.EnvKeyReplacer) > 0 {
		var oldnew []string
		for old, new := range m.config.EnvKeyReplacer {
			oldnew = append(oldnew, old, new)
		}
		m.viper.SetEnvKeyReplacer(strings.NewReplacer(oldnew...))
	}

	// 设置默认值
	for key, value := range m.config.Defaults {
		m.viper.SetDefault(key, value)
	}

	// 设置大小写敏感
	if !m.config.CaseSensitive {
		// Viper默认不区分大小写
	}

	// 允许空环境变量
	m.viper.AllowEmptyEnv(m.config.AllowEmptyEnv)
}

// Load 加载配置
func (m *ViperManager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return fmt.Errorf("config manager is closed")
	}

	// 读取配置文件
	if err := m.viper.ReadInConfig(); err != nil {
		// 如果配置文件不存在，检查是否有默认值或环境变量
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 配置文件未找到，但可能有环境变量或默认值
		} else {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// 读取远程配置
	if err := m.loadRemoteConfig(); err != nil {
		return fmt.Errorf("failed to load remote config: %w", err)
	}

	// 验证配置
	if m.config.ValidateOnLoad && m.validator != nil {
		if err := m.validator.Validate(m.viper.AllSettings()); err != nil {
			return fmt.Errorf("config validation failed: %w", err)
		}
	}

	// 启动配置监听
	if m.config.WatchConfig {
		m.startWatching()
	}

	return nil
}

// loadRemoteConfig 加载远程配置
func (m *ViperManager) loadRemoteConfig() error {
	if m.config.RemoteConfig.Provider == "" {
		return nil // 没有配置远程配置源
	}

	// TODO: 实现远程配置加载
	// 这里需要根据不同的provider实现相应的逻辑
	switch m.config.RemoteConfig.Provider {
	case "etcd":
		return m.loadFromEtcd()
	case "consul":
		return m.loadFromConsul()
	case "firestore":
		return m.loadFromFirestore()
	default:
		return fmt.Errorf("unsupported remote config provider: %s", m.config.RemoteConfig.Provider)
	}
}

// loadFromEtcd 从etcd加载配置
func (m *ViperManager) loadFromEtcd() error {
	// TODO: 实现etcd配置加载
	return nil
}

// loadFromConsul 从consul加载配置
func (m *ViperManager) loadFromConsul() error {
	// TODO: 实现consul配置加载
	return nil
}

// loadFromFirestore 从firestore加载配置
func (m *ViperManager) loadFromFirestore() error {
	// TODO: 实现firestore配置加载
	return nil
}

// Reload 重新加载配置
func (m *ViperManager) Reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return fmt.Errorf("config manager is closed")
	}

	oldSettings := m.viper.AllSettings()

	// 重新读取配置
	if err := m.viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}

	// 验证配置
	if m.validator != nil {
		if err := m.validator.Validate(m.viper.AllSettings()); err != nil {
			return fmt.Errorf("config validation failed after reload: %w", err)
		}
	}

	// 触发重载事件
	event := Event{
		Type:      EventTypeReload,
		Timestamp: time.Now(),
	}

	// 比较配置变化
	newSettings := m.viper.AllSettings()
	m.compareAndNotify(oldSettings, newSettings)

	// 发送重载事件
	m.notifyWatchers(event)

	return nil
}

// compareAndNotify 比较配置变化并通知
func (m *ViperManager) compareAndNotify(oldSettings, newSettings map[string]interface{}) {
	// 检查修改和新增的键
	for key, newValue := range newSettings {
		if oldValue, exists := oldSettings[key]; exists {
			if !m.deepEqual(oldValue, newValue) {
				event := Event{
					Type:      EventTypeSet,
					Key:       key,
					Value:     newValue,
					OldValue:  oldValue,
					Timestamp: time.Now(),
				}
				m.notifyWatchers(event)
			}
		} else {
			event := Event{
				Type:      EventTypeSet,
				Key:       key,
				Value:     newValue,
				Timestamp: time.Now(),
			}
			m.notifyWatchers(event)
		}
	}

	// 检查删除的键
	for key, oldValue := range oldSettings {
		if _, exists := newSettings[key]; !exists {
			event := Event{
				Type:      EventTypeDelete,
				Key:       key,
				OldValue:  oldValue,
				Timestamp: time.Now(),
			}
			m.notifyWatchers(event)
		}
	}
}

// deepEqual 深度比较两个值是否相等
func (m *ViperManager) deepEqual(a, b interface{}) bool {
	// 简单实现，实际项目中可能需要更复杂的比较逻辑
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// Get 获取配置值
func (m *ViperManager) Get(key string) interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.Get(key)
}

// GetString 获取字符串配置
func (m *ViperManager) GetString(key string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetString(key)
}

// GetInt 获取整数配置
func (m *ViperManager) GetInt(key string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetInt(key)
}

// GetBool 获取布尔配置
func (m *ViperManager) GetBool(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetBool(key)
}

// GetFloat64 获取浮点数配置
func (m *ViperManager) GetFloat64(key string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetFloat64(key)
}

// GetDuration 获取时间间隔配置
func (m *ViperManager) GetDuration(key string) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetDuration(key)
}

// GetStringSlice 获取字符串数组配置
func (m *ViperManager) GetStringSlice(key string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.GetStringSlice(key)
}

// Set 设置配置值
func (m *ViperManager) Set(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	oldValue := m.viper.Get(key)
	m.viper.Set(key, value)
	
	// 触发变化事件
	event := Event{
		Type:      EventTypeSet,
		Key:       key,
		Value:     value,
		OldValue:  oldValue,
		Timestamp: time.Now(),
	}
	m.notifyWatchers(event)
}

// IsSet 检查配置是否存在
func (m *ViperManager) IsSet(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.IsSet(key)
}

// Unmarshal 将配置解析到结构体
func (m *ViperManager) Unmarshal(rawVal interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.Unmarshal(rawVal)
}

// UnmarshalKey 将指定键的配置解析到结构体
func (m *ViperManager) UnmarshalKey(key string, rawVal interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.UnmarshalKey(key, rawVal)
}

// Watch 监听配置变化
func (m *ViperManager) Watch(callback func(event Event)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return fmt.Errorf("config manager is closed")
	}

	m.watchers = append(m.watchers, callback)
	return nil
}

// StopWatch 停止监听配置变化
func (m *ViperManager) StopWatch() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.watchers = make([]func(event Event), 0)
}

// startWatching 启动配置文件监听
func (m *ViperManager) startWatching() {
	m.viper.WatchConfig()
	m.viper.OnConfigChange(func(e fsnotify.Event) {
		event := Event{
			Type:      EventTypeReload,
			Timestamp: time.Now(),
		}
		m.notifyWatchers(event)
		
		// 重新加载配置
		go func() {
			if err := m.Reload(); err != nil {
				// 这里应该有更好的错误处理，比如日志记录
				fmt.Printf("Failed to reload config: %v\n", err)
			}
		}()
	})
}

// notifyWatchers 通知所有监听器
func (m *ViperManager) notifyWatchers(event Event) {
	for _, watcher := range m.watchers {
		go watcher(event)
	}
}

// Validate 验证配置
func (m *ViperManager) Validate(validator Validator) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.validator = validator
	return validator.Validate(m.viper.AllSettings())
}

// GetAllSettings 获取所有配置
func (m *ViperManager) GetAllSettings() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.viper.AllSettings()
}

// Close 关闭配置管理器
func (m *ViperManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true
	m.watchers = nil
	return nil
}