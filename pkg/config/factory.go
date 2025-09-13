package config

import (
	"fmt"
	"strings"
)

// ManagerType 管理器类型
type ManagerType string

const (
	// ViperManagerType Viper管理器类型
	ViperManagerType ManagerType = "viper"
	// EncryptedManagerType 加密管理器类型
	EncryptedManagerType ManagerType = "encrypted"
	// EnvironmentManagerType 环境管理器类型
	EnvironmentManagerType ManagerType = "environment"
	// MultiEnvironmentManagerType 多环境管理器类型
	MultiEnvironmentManagerType ManagerType = "multi-environment"
)

// Factory 配置管理器工厂接口
type Factory interface {
	Create(managerType ManagerType, config *Config) (Manager, error)
	Register(managerType ManagerType, creator ManagerCreator)
	List() []ManagerType
}

// ManagerCreator 管理器创建函数类型
type ManagerCreator func(config *Config, options ...interface{}) (Manager, error)

// DefaultFactory 默认配置管理器工厂
type DefaultFactory struct {
	creators map[ManagerType]ManagerCreator
}

// NewDefaultFactory 创建默认工厂
func NewDefaultFactory() *DefaultFactory {
	factory := &DefaultFactory{
		creators: make(map[ManagerType]ManagerCreator),
	}
	
	// 注册默认管理器创建器
	factory.registerDefaultCreators()
	
	return factory
}

// registerDefaultCreators 注册默认管理器创建器
func (f *DefaultFactory) registerDefaultCreators() {
	// Viper管理器
	f.Register(ViperManagerType, func(config *Config, options ...interface{}) (Manager, error) {
		return NewViperManager(config), nil
	})
	
	// 加密管理器
	f.Register(EncryptedManagerType, func(config *Config, options ...interface{}) (Manager, error) {
		var encryptedKeys []string
		if len(options) > 0 {
			if keys, ok := options[0].([]string); ok {
				encryptedKeys = keys
			}
		}
		return NewEncryptedViperManager(config, encryptedKeys)
	})
	
	// 环境管理器
	f.Register(EnvironmentManagerType, func(config *Config, options ...interface{}) (Manager, error) {
		env := Development
		if config != nil && config.Environment != "" {
			env = Environment(config.Environment)
		}
		if len(options) > 0 {
			if envOption, ok := options[0].(Environment); ok {
				env = envOption
			}
		}
		
		envManager := NewEnvironmentManager(env)
		return envManager.LoadWithEnvironment(env, config)
	})
}

// Create 创建配置管理器
func (f *DefaultFactory) Create(managerType ManagerType, config *Config) (Manager, error) {
	return f.CreateWithOptions(managerType, config)
}

// CreateWithOptions 使用选项创建配置管理器
func (f *DefaultFactory) CreateWithOptions(managerType ManagerType, config *Config, options ...interface{}) (Manager, error) {
	creator, exists := f.creators[managerType]
	if !exists {
		return nil, fmt.Errorf("unknown manager type: %s", managerType)
	}
	
	if config == nil {
		config = DefaultConfig()
	}
	
	manager, err := creator(config, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create manager of type %s: %w", managerType, err)
	}
	
	// 加载配置
	if err := manager.Load(); err != nil {
		return nil, fmt.Errorf("failed to load config for manager type %s: %w", managerType, err)
	}
	
	return manager, nil
}

// Register 注册管理器创建器
func (f *DefaultFactory) Register(managerType ManagerType, creator ManagerCreator) {
	f.creators[managerType] = creator
}

// List 列出所有可用的管理器类型
func (f *DefaultFactory) List() []ManagerType {
	types := make([]ManagerType, 0, len(f.creators))
	for managerType := range f.creators {
		types = append(types, managerType)
	}
	return types
}

// Unregister 注销管理器创建器
func (f *DefaultFactory) Unregister(managerType ManagerType) {
	delete(f.creators, managerType)
}

// HasType 检查是否支持指定类型
func (f *DefaultFactory) HasType(managerType ManagerType) bool {
	_, exists := f.creators[managerType]
	return exists
}

// 全局工厂实例
var defaultFactory = NewDefaultFactory()

// CreateManager 使用默认工厂创建管理器
func CreateManager(managerType ManagerType, config *Config) (Manager, error) {
	return defaultFactory.Create(managerType, config)
}

// CreateManagerWithOptions 使用默认工厂和选项创建管理器
func CreateManagerWithOptions(managerType ManagerType, config *Config, options ...interface{}) (Manager, error) {
	return defaultFactory.CreateWithOptions(managerType, config, options...)
}

// RegisterManagerCreator 注册管理器创建器到默认工厂
func RegisterManagerCreator(managerType ManagerType, creator ManagerCreator) {
	defaultFactory.Register(managerType, creator)
}

// ListManagerTypes 列出默认工厂支持的管理器类型
func ListManagerTypes() []ManagerType {
	return defaultFactory.List()
}

// ManagerRegistry 管理器注册表
type ManagerRegistry struct {
	managers map[string]Manager
	factory  Factory
}

// NewManagerRegistry 创建管理器注册表
func NewManagerRegistry(factory Factory) *ManagerRegistry {
	if factory == nil {
		factory = NewDefaultFactory()
	}
	
	return &ManagerRegistry{
		managers: make(map[string]Manager),
		factory:  factory,
	}
}

// Register 注册管理器实例
func (r *ManagerRegistry) Register(name string, manager Manager) {
	r.managers[name] = manager
}

// Get 获取管理器实例
func (r *ManagerRegistry) Get(name string) (Manager, bool) {
	manager, exists := r.managers[name]
	return manager, exists
}

// GetOrCreate 获取或创建管理器
func (r *ManagerRegistry) GetOrCreate(name string, managerType ManagerType, config *Config, options ...interface{}) (Manager, error) {
	if manager, exists := r.managers[name]; exists {
		return manager, nil
	}
	
	// 检查工厂类型并调用相应方法
	if defaultFactory, ok := r.factory.(*DefaultFactory); ok {
		manager, err := defaultFactory.CreateWithOptions(managerType, config, options...)
		if err != nil {
			return nil, err
		}
		r.managers[name] = manager
		return manager, nil
	}
	
	// 如果不是DefaultFactory，使用基本Create方法
	manager, err := r.factory.Create(managerType, config)
	if err != nil {
		return nil, err
	}
	
	r.managers[name] = manager
	return manager, nil
}

// Unregister 注销管理器
func (r *ManagerRegistry) Unregister(name string) {
	if manager, exists := r.managers[name]; exists {
		manager.Close() // 关闭管理器
		delete(r.managers, name)
	}
}

// List 列出所有注册的管理器名称
func (r *ManagerRegistry) List() []string {
	names := make([]string, 0, len(r.managers))
	for name := range r.managers {
		names = append(names, name)
	}
	return names
}

// Close 关闭所有管理器
func (r *ManagerRegistry) Close() error {
	var errs []string
	
	for name, manager := range r.managers {
		if err := manager.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, err))
		}
	}
	
	// 清空注册表
	r.managers = make(map[string]Manager)
	
	if len(errs) > 0 {
		return fmt.Errorf("failed to close some managers: %s", strings.Join(errs, "; "))
	}
	
	return nil
}

// Count 返回注册的管理器数量
func (r *ManagerRegistry) Count() int {
	return len(r.managers)
}

// 全局管理器注册表
var globalRegistry = NewManagerRegistry(defaultFactory)

// GetGlobalRegistry 获取全局管理器注册表
func GetGlobalRegistry() *ManagerRegistry {
	return globalRegistry
}

// RegisterGlobalManager 注册全局管理器
func RegisterGlobalManager(name string, manager Manager) {
	globalRegistry.Register(name, manager)
}

// GetGlobalManager 获取全局管理器
func GetGlobalManager(name string) (Manager, bool) {
	return globalRegistry.Get(name)
}

// UnregisterGlobalManager 注销全局管理器
func UnregisterGlobalManager(name string) {
	globalRegistry.Unregister(name)
}

// CloseAllGlobalManagers 关闭所有全局管理器
func CloseAllGlobalManagers() error {
	return globalRegistry.Close()
}

// AutoDetectManagerType 自动检测管理器类型
func AutoDetectManagerType(config *Config) ManagerType {
	if config == nil {
		return ViperManagerType
	}
	
	// 如果有加密密钥，使用加密管理器
	if config.EncryptionKey != "" {
		return EncryptedManagerType
	}
	
	// 如果有环境配置，使用环境管理器
	if config.Environment != "" && Environment(config.Environment).IsValid() {
		return EnvironmentManagerType
	}
	
	// 如果有远程配置，可以扩展支持远程管理器
	if config.RemoteConfig.Provider != "" {
		// 这里可以返回远程管理器类型
		// return RemoteManagerType
	}
	
	// 默认使用Viper管理器
	return ViperManagerType
}

// CreateAuto 自动检测并创建合适的管理器
func CreateAuto(config *Config, options ...interface{}) (Manager, error) {
	managerType := AutoDetectManagerType(config)
	return CreateManagerWithOptions(managerType, config, options...)
}

// PresetFactory 预设工厂
type PresetFactory struct {
	*DefaultFactory
}

// NewPresetFactory 创建预设工厂
func NewPresetFactory() *PresetFactory {
	return &PresetFactory{
		DefaultFactory: NewDefaultFactory(),
	}
}

// CreateDevelopment 创建开发环境管理器
func (f *PresetFactory) CreateDevelopment(configPaths ...string) (Manager, error) {
	config := DevelopmentConfig()
	if len(configPaths) > 0 {
		config.ConfigPaths = configPaths
	}
	return f.Create(ViperManagerType, config)
}

// CreateProduction 创建生产环境管理器
func (f *PresetFactory) CreateProduction(configPaths ...string) (Manager, error) {
	config := ProductionConfig()
	if len(configPaths) > 0 {
		config.ConfigPaths = configPaths
	}
	return f.Create(ViperManagerType, config)
}

// CreateTesting 创建测试环境管理器
func (f *PresetFactory) CreateTesting(configPaths ...string) (Manager, error) {
	config := TestingConfig()
	if len(configPaths) > 0 {
		config.ConfigPaths = configPaths
	}
	return f.Create(ViperManagerType, config)
}

// CreateEncrypted 创建加密管理器
func (f *PresetFactory) CreateEncrypted(encryptionKey string, encryptedKeys []string, config *Config) (Manager, error) {
	if config == nil {
		config = DefaultConfig()
	}
	config.EncryptionKey = encryptionKey
	return f.CreateWithOptions(EncryptedManagerType, config, encryptedKeys)
}

// 全局预设工厂
var presetFactory = NewPresetFactory()

// GetPresetFactory 获取预设工厂
func GetPresetFactory() *PresetFactory {
	return presetFactory
}