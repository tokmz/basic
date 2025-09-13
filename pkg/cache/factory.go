package cache

import (
	"fmt"
)

// CacheFactory 缓存工厂
type CacheFactory struct {
	configs map[CacheType]*Config
}

// NewCacheFactory 创建缓存工厂
func NewCacheFactory() *CacheFactory {
	return &CacheFactory{
		configs: make(map[CacheType]*Config),
	}
}

// RegisterConfig 注册缓存配置
func (f *CacheFactory) RegisterConfig(cacheType CacheType, config *Config) {
	f.configs[cacheType] = config
}

// CreateCache 创建缓存实例
func (f *CacheFactory) CreateCache(cacheType CacheType) (Cache, error) {
	config, exists := f.configs[cacheType]
	if !exists {
		// 使用默认配置
		config = DefaultConfig()
		config.Type = cacheType
	}

	return f.createCacheWithConfig(cacheType, config)
}

// CreateCacheWithConfig 使用指定配置创建缓存实例
func (f *CacheFactory) CreateCacheWithConfig(config *Config) (Cache, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return f.createCacheWithConfig(config.Type, config)
}

// createCacheWithConfig 内部创建缓存实例的方法
func (f *CacheFactory) createCacheWithConfig(cacheType CacheType, config *Config) (Cache, error) {
	switch cacheType {
	case TypeMemoryLRU:
		return f.createMemoryCache(config)
	case TypeMemoryLFU:
		return f.createMemoryCache(config)
	case TypeRedisSingle:
		return f.createRedisCache(config)
	case TypeRedisCluster:
		return f.createRedisCache(config)
	case TypeMultiLevel:
		return f.createMultiLevelCache(config)
	default:
		return nil, fmt.Errorf("unsupported cache type: %s", cacheType)
	}
}

// createMemoryCache 创建内存缓存
func (f *CacheFactory) createMemoryCache(config *Config) (Cache, error) {
	memoryConfig := config.Memory

	// 根据类型设置淘汰策略
	switch config.Type {
	case TypeMemoryLRU:
		memoryConfig.EvictPolicy = EvictLRU
	case TypeMemoryLFU:
		memoryConfig.EvictPolicy = EvictLFU
	}

	return NewMemoryCache(memoryConfig), nil
}

// createRedisCache 创建Redis缓存
func (f *CacheFactory) createRedisCache(config *Config) (Cache, error) {
	redisConfig := config.Redis

	// 根据类型设置集群模式
	switch config.Type {
	case TypeRedisSingle:
		redisConfig.ClusterMode = false
	case TypeRedisCluster:
		redisConfig.ClusterMode = true
	}

	return NewRedisCache(redisConfig)
}

// createMultiLevelCache 创建多级缓存
func (f *CacheFactory) createMultiLevelCache(config *Config) (Cache, error) {
	return NewMultiLevelCache(*config)
}

// CacheManager 缓存管理器
type CacheManager struct {
	factory *CacheFactory
	caches  map[string]Cache
	monitor *Monitor
	logger  *Logger
}

// NewCacheManager 创建缓存管理器
func NewCacheManager() (*CacheManager, error) {
	factory := NewCacheFactory()

	// 创建默认的监控和日志
	config := DefaultConfig()
	monitor := NewMonitor(config.Monitoring, nil)
	logger, err := NewLogger(config.Monitoring)
	if err != nil {
		return nil, err
	}

	return &CacheManager{
		factory: factory,
		caches:  make(map[string]Cache),
		monitor: monitor,
		logger:  logger,
	}, nil
}

// RegisterCache 注册缓存实例
func (m *CacheManager) RegisterCache(name string, cache Cache) {
	m.caches[name] = cache

	// 添加到监控
	collector := NewCacheCollector(name, cache)
	m.monitor.AddCollector(collector)
}

// GetCache 获取缓存实例
func (m *CacheManager) GetCache(name string) (Cache, bool) {
	cache, exists := m.caches[name]
	return cache, exists
}

// CreateAndRegisterCache 创建并注册缓存
func (m *CacheManager) CreateAndRegisterCache(name string, config *Config) (Cache, error) {
	cache, err := m.factory.CreateCacheWithConfig(config)
	if err != nil {
		return nil, err
	}

	m.RegisterCache(name, cache)
	return cache, nil
}

// GetOrCreateCache 获取或创建缓存
func (m *CacheManager) GetOrCreateCache(name string, config *Config) (Cache, error) {
	if cache, exists := m.GetCache(name); exists {
		return cache, nil
	}

	return m.CreateAndRegisterCache(name, config)
}

// RemoveCache 移除缓存实例
func (m *CacheManager) RemoveCache(name string) error {
	cache, exists := m.caches[name]
	if !exists {
		return fmt.Errorf("cache %s not found", name)
	}

	// 关闭缓存
	if err := cache.Close(); err != nil {
		return err
	}

	delete(m.caches, name)
	return nil
}

// CloseAll 关闭所有缓存
func (m *CacheManager) CloseAll() error {
	var errs []error

	for name, cache := range m.caches {
		if err := cache.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close cache %s: %w", name, err))
		}
	}

	// 停止监控
	if err := m.monitor.Stop(); err != nil {
		errs = append(errs, fmt.Errorf("failed to stop monitor: %w", err))
	}

	if len(errs) > 0 {
		return errs[0] // 返回第一个错误
	}

	return nil
}

// GetMonitor 获取监控器
func (m *CacheManager) GetMonitor() *Monitor {
	return m.monitor
}

// GetLogger 获取日志器
func (m *CacheManager) GetLogger() *Logger {
	return m.logger
}

// ListCaches 列出所有注册的缓存
func (m *CacheManager) ListCaches() []string {
	names := make([]string, 0, len(m.caches))
	for name := range m.caches {
		names = append(names, name)
	}
	return names
}

// GetAllStats 获取所有缓存的统计信息
func (m *CacheManager) GetAllStats() map[string]CacheStats {
	stats := make(map[string]CacheStats)
	for name, cache := range m.caches {
		stats[name] = cache.Stats()
	}
	return stats
}

// 全局工厂实例和便捷函数

var (
	defaultFactory *CacheFactory
	defaultManager *CacheManager
)

// init 初始化默认实例
func init() {
	defaultFactory = NewCacheFactory()

	// 注册默认配置
	defaultFactory.RegisterConfig(TypeMemoryLRU, DefaultConfig())

	manager, _ := NewCacheManager()
	defaultManager = manager
}

// GetDefaultFactory 获取默认工厂
func GetDefaultFactory() *CacheFactory {
	return defaultFactory
}

// GetDefaultManager 获取默认管理器
func GetDefaultManager() *CacheManager {
	return defaultManager
}

// Create 使用默认工厂创建缓存
func Create(cacheType CacheType) (Cache, error) {
	return defaultFactory.CreateCache(cacheType)
}

// CreateWithConfig 使用配置创建缓存
func CreateWithConfig(config *Config) (Cache, error) {
	return defaultFactory.CreateCacheWithConfig(config)
}

// Register 注册到默认管理器
func Register(name string, cache Cache) {
	defaultManager.RegisterCache(name, cache)
}

// Get 从默认管理器获取缓存
func Get(name string) (Cache, bool) {
	return defaultManager.GetCache(name)
}

// Remove 从默认管理器移除缓存
func Remove(name string) error {
	return defaultManager.RemoveCache(name)
}

// CloseAllCaches 关闭所有缓存
func CloseAllCaches() error {
	return defaultManager.CloseAll()
}

// CacheBuilder 缓存构建器(Builder模式)
type CacheBuilder struct {
	config *Config
}

// NewCacheBuilder 创建缓存构建器
func NewCacheBuilder() *CacheBuilder {
	return &CacheBuilder{
		config: DefaultConfig(),
	}
}

// WithType 设置缓存类型
func (b *CacheBuilder) WithType(cacheType CacheType) *CacheBuilder {
	b.config.Type = cacheType
	return b
}

// WithMemoryConfig 设置内存配置
func (b *CacheBuilder) WithMemoryConfig(config MemoryConfig) *CacheBuilder {
	b.config.Memory = config
	return b
}

// WithRedisConfig 设置Redis配置
func (b *CacheBuilder) WithRedisConfig(config RedisConfig) *CacheBuilder {
	b.config.Redis = config
	return b
}

// WithMultiLevelConfig 设置多级缓存配置
func (b *CacheBuilder) WithMultiLevelConfig(config MultiLevelConfig) *CacheBuilder {
	b.config.MultiLevel = config
	return b
}

// WithMonitoring 设置监控配置
func (b *CacheBuilder) WithMonitoring(config MonitoringConfig) *CacheBuilder {
	b.config.Monitoring = config
	return b
}

// WithDistributed 设置分布式配置
func (b *CacheBuilder) WithDistributed(config DistributedConfig) *CacheBuilder {
	b.config.Distributed = config
	return b
}

// Build 构建缓存实例
func (b *CacheBuilder) Build() (Cache, error) {
	return CreateWithConfig(b.config)
}

// BuildAndRegister 构建并注册缓存实例
func (b *CacheBuilder) BuildAndRegister(name string) (Cache, error) {
	cache, err := b.Build()
	if err != nil {
		return nil, err
	}

	Register(name, cache)
	return cache, nil
}
