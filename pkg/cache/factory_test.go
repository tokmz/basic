package cache

import (
	"context"
	"testing"
	"time"
)

func TestCacheFactory_CreateMemoryCache(t *testing.T) {
	factory := NewCacheFactory()

	config := DefaultConfig()
	config.Type = TypeMemoryLRU

	cache, err := factory.CreateCacheWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create memory cache: %v", err)
	}

	if cache == nil {
		t.Fatal("Cache should not be nil")
	}

	defer cache.Close()

	// 验证缓存类型
	if _, ok := cache.(*MemoryCache); !ok {
		t.Error("Expected MemoryCache type")
	}
}

func TestCacheFactory_CreateRedisCache(t *testing.T) {
	factory := NewCacheFactory()

	config := DefaultConfig()
	config.Type = TypeRedisSingle
	config.Redis.Addrs = []string{"localhost:6379"}

	_, err := factory.CreateCacheWithConfig(config)
	// Redis可能未运行，这里只测试工厂方法不会panic
	if err != nil {
		t.Logf("Redis cache creation failed (expected if Redis not running): %v", err)
	}
}

func TestCacheFactory_CreateMultiLevelCache(t *testing.T) {
	factory := NewCacheFactory()

	config := DefaultConfig()
	config.Type = TypeMultiLevel
	config.MultiLevel.EnableL1 = true
	config.MultiLevel.EnableL2 = false // 避免Redis依赖

	cache, err := factory.CreateCacheWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create multi-level cache: %v", err)
	}

	if cache == nil {
		t.Fatal("Cache should not be nil")
	}

	defer cache.Close()

	// 验证缓存类型
	if _, ok := cache.(*MultiLevelCache); !ok {
		t.Error("Expected MultiLevelCache type")
	}
}

func TestCacheFactory_UnsupportedType(t *testing.T) {
	factory := NewCacheFactory()

	config := DefaultConfig()
	config.Type = CacheType("unsupported")

	_, err := factory.CreateCacheWithConfig(config)
	if err == nil {
		t.Error("Expected error for unsupported cache type")
	}
}

func TestCacheFactory_RegisterConfig(t *testing.T) {
	factory := NewCacheFactory()

	customConfig := &Config{
		Type:       TypeMemoryLRU,
		DefaultTTL: 30 * time.Minute,
		Memory: MemoryConfig{
			MaxSize:     500,
			EvictPolicy: EvictLRU,
		},
	}

	factory.RegisterConfig(TypeMemoryLRU, customConfig)

	cache, err := factory.CreateCache(TypeMemoryLRU)
	if err != nil {
		t.Fatalf("Failed to create cache with registered config: %v", err)
	}

	defer cache.Close()

	if cache == nil {
		t.Fatal("Cache should not be nil")
	}
}

func TestCacheManager_RegisterAndGet(t *testing.T) {
	manager, err := NewCacheManager()
	if err != nil {
		t.Fatalf("Failed to create cache manager: %v", err)
	}
	defer manager.CloseAll()

	config := DefaultConfig()
	config.Type = TypeMemoryLRU

	cache, err := manager.CreateAndRegisterCache("test_cache", config)
	if err != nil {
		t.Fatalf("Failed to create and register cache: %v", err)
	}

	// 获取缓存
	retrievedCache, exists := manager.GetCache("test_cache")
	if !exists {
		t.Fatal("Cache should exist")
	}

	if retrievedCache != cache {
		t.Error("Retrieved cache should be the same instance")
	}
}

func TestCacheManager_GetOrCreateCache(t *testing.T) {
	manager, err := NewCacheManager()
	if err != nil {
		t.Fatalf("Failed to create cache manager: %v", err)
	}
	defer manager.CloseAll()

	config := DefaultConfig()
	config.Type = TypeMemoryLRU

	// 第一次调用应该创建缓存
	cache1, err := manager.GetOrCreateCache("test_cache", config)
	if err != nil {
		t.Fatalf("Failed to get or create cache: %v", err)
	}

	// 第二次调用应该返回相同的缓存
	cache2, err := manager.GetOrCreateCache("test_cache", config)
	if err != nil {
		t.Fatalf("Failed to get or create cache: %v", err)
	}

	if cache1 != cache2 {
		t.Error("Should return the same cache instance")
	}
}

func TestCacheManager_RemoveCache(t *testing.T) {
	manager, err := NewCacheManager()
	if err != nil {
		t.Fatalf("Failed to create cache manager: %v", err)
	}
	defer manager.CloseAll()

	config := DefaultConfig()
	config.Type = TypeMemoryLRU

	// 创建并注册缓存
	_, err = manager.CreateAndRegisterCache("test_cache", config)
	if err != nil {
		t.Fatalf("Failed to create and register cache: %v", err)
	}

	// 验证缓存存在
	_, exists := manager.GetCache("test_cache")
	if !exists {
		t.Fatal("Cache should exist before removal")
	}

	// 移除缓存
	err = manager.RemoveCache("test_cache")
	if err != nil {
		t.Fatalf("Failed to remove cache: %v", err)
	}

	// 验证缓存不存在
	_, exists = manager.GetCache("test_cache")
	if exists {
		t.Error("Cache should not exist after removal")
	}
}

func TestCacheManager_ListCaches(t *testing.T) {
	manager, err := NewCacheManager()
	if err != nil {
		t.Fatalf("Failed to create cache manager: %v", err)
	}
	defer manager.CloseAll()

	config := DefaultConfig()
	config.Type = TypeMemoryLRU

	// 创建多个缓存
	cacheNames := []string{"cache1", "cache2", "cache3"}
	for _, name := range cacheNames {
		_, err := manager.CreateAndRegisterCache(name, config)
		if err != nil {
			t.Fatalf("Failed to create cache %s: %v", name, err)
		}
	}

	// 获取缓存列表
	listedNames := manager.ListCaches()

	if len(listedNames) != len(cacheNames) {
		t.Errorf("Expected %d caches, got %d", len(cacheNames), len(listedNames))
	}

	// 验证所有缓存都在列表中
	for _, name := range cacheNames {
		found := false
		for _, listedName := range listedNames {
			if name == listedName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Cache %s not found in list", name)
		}
	}
}

func TestCacheManager_GetAllStats(t *testing.T) {
	manager, err := NewCacheManager()
	if err != nil {
		t.Fatalf("Failed to create cache manager: %v", err)
	}
	defer manager.CloseAll()

	config := DefaultConfig()
	config.Type = TypeMemoryLRU

	// 创建缓存
	cache, err := manager.CreateAndRegisterCache("test_cache", config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// 执行一些操作以产生统计数据
	ctx := context.Background()
	cache.Set(ctx, "key1", "value1", time.Hour)
	cache.Get(ctx, "key1")

	// 获取统计信息
	allStats := manager.GetAllStats()

	if len(allStats) != 1 {
		t.Errorf("Expected 1 cache stats, got %d", len(allStats))
	}

	stats, exists := allStats["test_cache"]
	if !exists {
		t.Fatal("Stats for test_cache should exist")
	}

	if stats.Sets != 1 {
		t.Errorf("Expected 1 set operation, got %d", stats.Sets)
	}

	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}
}

func TestCacheBuilder_Build(t *testing.T) {
	builder := NewCacheBuilder()

	cache, err := builder.
		WithType(TypeMemoryLRU).
		WithMemoryConfig(MemoryConfig{
			MaxSize:     100,
			MaxMemory:   1024 * 1024,
			EvictPolicy: EvictLRU,
		}).
		Build()

	if err != nil {
		t.Fatalf("Failed to build cache: %v", err)
	}

	if cache == nil {
		t.Fatal("Cache should not be nil")
	}

	defer cache.Close()

	// 验证缓存类型
	if _, ok := cache.(*MemoryCache); !ok {
		t.Error("Expected MemoryCache type")
	}
}

func TestCacheBuilder_BuildAndRegister(t *testing.T) {
	builder := NewCacheBuilder()

	cache, err := builder.
		WithType(TypeMemoryLRU).
		BuildAndRegister("builder_cache")

	if err != nil {
		t.Fatalf("Failed to build and register cache: %v", err)
	}

	if cache == nil {
		t.Fatal("Cache should not be nil")
	}

	// 验证缓存已注册到默认管理器
	retrievedCache, exists := Get("builder_cache")
	if !exists {
		t.Fatal("Cache should be registered in default manager")
	}

	if retrievedCache != cache {
		t.Error("Retrieved cache should be the same instance")
	}

	// 清理
	Remove("builder_cache")
}

func TestDefaultFunctions(t *testing.T) {
	// 测试默认工厂和管理器函数
	cache, err := Create(TypeMemoryLRU)
	if err != nil {
		t.Fatalf("Failed to create cache with default factory: %v", err)
	}
	defer cache.Close()

	if cache == nil {
		t.Fatal("Cache should not be nil")
	}

	// 注册到默认管理器
	Register("default_test", cache)

	// 从默认管理器获取
	retrievedCache, exists := Get("default_test")
	if !exists {
		t.Fatal("Cache should exist in default manager")
	}

	if retrievedCache != cache {
		t.Error("Retrieved cache should be the same instance")
	}

	// 移除
	err = Remove("default_test")
	if err != nil {
		t.Fatalf("Failed to remove from default manager: %v", err)
	}

	// 验证已移除
	_, exists = Get("default_test")
	if exists {
		t.Error("Cache should not exist after removal")
	}
}
