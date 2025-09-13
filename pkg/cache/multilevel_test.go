package cache

import (
	"context"
	"testing"
	"time"
)

// MockCache2 简单的模拟缓存实现
type MockCache2 struct {
	data      map[string]interface{}
	stats     CacheStats
	closed    bool
	setErr    error
	getErr    error
	delErr    error
	clrErr    error
	existsErr error
	keysErr   error
}

func NewMockCache2() *MockCache2 {
	return &MockCache2{
		data: make(map[string]interface{}),
	}
}

func (m *MockCache2) Get(ctx context.Context, key string) (interface{}, error) {
	if m.closed {
		return nil, ErrCacheClosed
	}
	if m.getErr != nil {
		return nil, m.getErr
	}

	m.stats.TotalRequests++
	if value, exists := m.data[key]; exists {
		m.stats.Hits++
		return value, nil
	}
	m.stats.Misses++
	return nil, ErrKeyNotFound
}

func (m *MockCache2) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if m.closed {
		return ErrCacheClosed
	}
	if m.setErr != nil {
		m.stats.Errors++
		return m.setErr
	}

	m.data[key] = value
	m.stats.Sets++
	m.stats.Size = int64(len(m.data))
	return nil
}

func (m *MockCache2) Delete(ctx context.Context, key string) error {
	if m.closed {
		return ErrCacheClosed
	}
	if m.delErr != nil {
		return m.delErr
	}

	if _, exists := m.data[key]; exists {
		delete(m.data, key)
		m.stats.Deletes++
		m.stats.Size = int64(len(m.data))
		return nil
	}
	return ErrKeyNotFound
}

func (m *MockCache2) Exists(ctx context.Context, key string) (bool, error) {
	if m.closed {
		return false, ErrCacheClosed
	}
	if m.existsErr != nil {
		return false, m.existsErr
	}

	_, exists := m.data[key]
	return exists, nil
}

func (m *MockCache2) Keys(ctx context.Context, pattern string) ([]string, error) {
	if m.closed {
		return nil, ErrCacheClosed
	}
	if m.keysErr != nil {
		return nil, m.keysErr
	}

	keys := make([]string, 0, len(m.data))
	for key := range m.data {
		// 简化的模式匹配，只支持"*"
		if pattern == "*" || pattern == key {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

func (m *MockCache2) Clear(ctx context.Context) error {
	if m.closed {
		return ErrCacheClosed
	}
	if m.clrErr != nil {
		return m.clrErr
	}

	m.data = make(map[string]interface{})
	m.stats.Size = 0
	return nil
}

func (m *MockCache2) Stats() CacheStats {
	if m.stats.TotalRequests > 0 {
		m.stats.HitRate = float64(m.stats.Hits) / float64(m.stats.TotalRequests)
	}
	return m.stats
}

func (m *MockCache2) Close() error {
	m.closed = true
	return nil
}

func TestNewMultiLevelCache(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "L1 only cache",
			config: Config{
				MultiLevel: MultiLevelConfig{
					EnableL1: true,
					EnableL2: false,
				},
				Memory: MemoryConfig{
					MaxSize:     100,
					EvictPolicy: EvictLRU,
				},
			},
			wantErr: false,
		},
		{
			name: "L1 and L2 cache",
			config: Config{
				MultiLevel: MultiLevelConfig{
					EnableL1: true,
					EnableL2: false, // 避免Redis依赖
				},
				Memory: MemoryConfig{
					MaxSize:     100,
					EvictPolicy: EvictLRU,
				},
			},
			wantErr: false,
		},
		{
			name: "No cache enabled",
			config: Config{
				MultiLevel: MultiLevelConfig{
					EnableL1: false,
					EnableL2: false,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache, err := NewMultiLevelCache(tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewMultiLevelCache() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if cache == nil {
					t.Error("NewMultiLevelCache() should not return nil")
					return
				}

				defer cache.Close()

				// 验证配置
				if cache.config.EnableL1 != tt.config.MultiLevel.EnableL1 {
					t.Errorf("Expected EnableL1 %v, got %v", tt.config.MultiLevel.EnableL1, cache.config.EnableL1)
				}

				if cache.config.EnableL2 != tt.config.MultiLevel.EnableL2 {
					t.Errorf("Expected EnableL2 %v, got %v", tt.config.MultiLevel.EnableL2, cache.config.EnableL2)
				}

				// 验证缓存实例
				if tt.config.MultiLevel.EnableL1 && cache.l1 == nil {
					t.Error("L1 cache should be initialized when enabled")
				}

				if !tt.config.MultiLevel.EnableL1 && cache.l1 != nil {
					t.Error("L1 cache should not be initialized when disabled")
				}
			}
		})
	}
}

func TestMultiLevelCache_Get(t *testing.T) {
	config := Config{
		MultiLevel: MultiLevelConfig{
			EnableL1:     true,
			EnableL2:     false, // 使用mock代替Redis
			SyncStrategy: SyncWriteThrough,
		},
		Memory: MemoryConfig{
			MaxSize:     100,
			EvictPolicy: EvictLRU,
		},
	}

	// 创建带模拟L2缓存的多级缓存
	ml, err := NewMultiLevelCache(config)
	if err != nil {
		t.Fatalf("Failed to create multi-level cache: %v", err)
	}
	defer ml.Close()

	// 手动设置L2为模拟缓存
	ml.l2 = NewMockCache2()

	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	tests := []struct {
		name      string
		setup     func()
		expectVal interface{}
		expectErr error
		checkFunc func(*testing.T, *MultiLevelCache)
	}{
		{
			name: "Get from L1 cache",
			setup: func() {
				ml.l1.Set(ctx, key, value, time.Hour)
			},
			expectVal: value,
			expectErr: nil,
			checkFunc: func(t *testing.T, cache *MultiLevelCache) {
				stats := cache.GetMultiLevelStats()
				if stats.l1Hits != 1 {
					t.Errorf("Expected 1 L1 hit, got %d", stats.l1Hits)
				}
			},
		},
		{
			name: "Get from L2 cache (L1 miss)",
			setup: func() {
				ml.l1.Clear(ctx)
				ml.l2.Set(ctx, key, value, time.Hour)
			},
			expectVal: value,
			expectErr: nil,
			checkFunc: func(t *testing.T, cache *MultiLevelCache) {
				stats := cache.GetMultiLevelStats()
				if stats.l2Hits != 1 {
					t.Errorf("Expected 1 L2 hit, got %d", stats.l2Hits)
				}
			},
		},
		{
			name: "Key not found in any cache",
			setup: func() {
				ml.l1.Clear(ctx)
				ml.l2.Clear(ctx)
			},
			expectVal: nil,
			expectErr: ErrKeyNotFound,
			checkFunc: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 重置统计
			ml.stats.l1Hits = 0
			ml.stats.l2Hits = 0

			if tt.setup != nil {
				tt.setup()
			}

			result, err := ml.Get(ctx, key)

			if err != tt.expectErr {
				t.Errorf("Expected error %v, got %v", tt.expectErr, err)
			}

			if result != tt.expectVal {
				t.Errorf("Expected value %v, got %v", tt.expectVal, result)
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, ml)
			}
		})
	}
}

func TestMultiLevelCache_Set_WriteThrough(t *testing.T) {
	config := Config{
		MultiLevel: MultiLevelConfig{
			EnableL1:     true,
			EnableL2:     false,
			SyncStrategy: SyncWriteThrough,
			L1TTL:        30 * time.Minute,
			L2TTL:        2 * time.Hour,
		},
		Memory: MemoryConfig{
			MaxSize:     100,
			EvictPolicy: EvictLRU,
		},
	}

	ml, err := NewMultiLevelCache(config)
	if err != nil {
		t.Fatalf("Failed to create multi-level cache: %v", err)
	}
	defer ml.Close()

	// 手动设置L2为模拟缓存
	ml.l2 = NewMockCache2()

	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	err = ml.Set(ctx, key, value, time.Hour)
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	// 验证L1中有数据
	l1Val, err := ml.l1.Get(ctx, key)
	if err != nil {
		t.Errorf("Expected value in L1, got error: %v", err)
	}
	if l1Val != value {
		t.Errorf("Expected L1 value %v, got %v", value, l1Val)
	}

	// 验证L2中有数据
	l2Val, err := ml.l2.Get(ctx, key)
	if err != nil {
		t.Errorf("Expected value in L2, got error: %v", err)
	}
	if l2Val != value {
		t.Errorf("Expected L2 value %v, got %v", value, l2Val)
	}

	// 验证统计
	stats := ml.GetMultiLevelStats()
	if stats.l1Sets != 1 {
		t.Errorf("Expected 1 L1 set, got %d", stats.l1Sets)
	}
	if stats.l2Sets != 1 {
		t.Errorf("Expected 1 L2 set, got %d", stats.l2Sets)
	}
}

func TestMultiLevelCache_Set_WriteBack(t *testing.T) {
	config := Config{
		MultiLevel: MultiLevelConfig{
			EnableL1:     true,
			EnableL2:     false,
			SyncStrategy: SyncWriteBack,
		},
		Memory: MemoryConfig{
			MaxSize:     100,
			EvictPolicy: EvictLRU,
		},
	}

	ml, err := NewMultiLevelCache(config)
	if err != nil {
		t.Fatalf("Failed to create multi-level cache: %v", err)
	}
	defer ml.Close()

	ml.l2 = NewMockCache2()

	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	err = ml.Set(ctx, key, value, time.Hour)
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	// 验证L1中立即有数据
	l1Val, err := ml.l1.Get(ctx, key)
	if err != nil {
		t.Errorf("Expected value in L1, got error: %v", err)
	}
	if l1Val != value {
		t.Errorf("Expected L1 value %v, got %v", value, l1Val)
	}

	// 等待异步写入L2完成
	time.Sleep(10 * time.Millisecond)

	// 验证L2中也有数据
	l2Val, err := ml.l2.Get(ctx, key)
	if err != nil {
		t.Errorf("Expected value in L2 after async write, got error: %v", err)
	}
	if l2Val != value {
		t.Errorf("Expected L2 value %v, got %v", value, l2Val)
	}
}

func TestMultiLevelCache_Set_WriteAround(t *testing.T) {
	config := Config{
		MultiLevel: MultiLevelConfig{
			EnableL1:     true,
			EnableL2:     false,
			SyncStrategy: SyncWriteAround,
		},
		Memory: MemoryConfig{
			MaxSize:     100,
			EvictPolicy: EvictLRU,
		},
	}

	ml, err := NewMultiLevelCache(config)
	if err != nil {
		t.Fatalf("Failed to create multi-level cache: %v", err)
	}
	defer ml.Close()

	ml.l2 = NewMockCache2()

	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	err = ml.Set(ctx, key, value, time.Hour)
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	// 验证L1中没有数据（写绕过模式）
	_, err = ml.l1.Get(ctx, key)
	if err != ErrKeyNotFound {
		t.Errorf("Expected L1 miss in write-around mode, got error: %v", err)
	}

	// 验证L2中有数据
	l2Val, err := ml.l2.Get(ctx, key)
	if err != nil {
		t.Errorf("Expected value in L2, got error: %v", err)
	}
	if l2Val != value {
		t.Errorf("Expected L2 value %v, got %v", value, l2Val)
	}
}

func TestMultiLevelCache_Delete(t *testing.T) {
	config := Config{
		MultiLevel: MultiLevelConfig{
			EnableL1: true,
			EnableL2: false,
		},
		Memory: MemoryConfig{
			MaxSize:     100,
			EvictPolicy: EvictLRU,
		},
	}

	ml, err := NewMultiLevelCache(config)
	if err != nil {
		t.Fatalf("Failed to create multi-level cache: %v", err)
	}
	defer ml.Close()

	ml.l2 = NewMockCache2()

	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	// 在两个缓存中都设置值
	ml.l1.Set(ctx, key, value, time.Hour)
	ml.l2.Set(ctx, key, value, time.Hour)

	// 删除键
	err = ml.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Failed to delete key: %v", err)
	}

	// 验证L1中已删除
	_, err = ml.l1.Get(ctx, key)
	if err != ErrKeyNotFound {
		t.Error("Expected key to be deleted from L1")
	}

	// 验证L2中已删除
	_, err = ml.l2.Get(ctx, key)
	if err != ErrKeyNotFound {
		t.Error("Expected key to be deleted from L2")
	}
}

func TestMultiLevelCache_Exists(t *testing.T) {
	config := Config{
		MultiLevel: MultiLevelConfig{
			EnableL1: true,
			EnableL2: false,
		},
		Memory: MemoryConfig{
			MaxSize:     100,
			EvictPolicy: EvictLRU,
		},
	}

	ml, err := NewMultiLevelCache(config)
	if err != nil {
		t.Fatalf("Failed to create multi-level cache: %v", err)
	}
	defer ml.Close()

	ml.l2 = NewMockCache2()

	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	// 测试不存在的键
	exists, err := ml.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if exists {
		t.Error("Expected key to not exist")
	}

	// 在L1中设置键
	ml.l1.Set(ctx, key, value, time.Hour)

	exists, err = ml.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if !exists {
		t.Error("Expected key to exist in L1")
	}

	// 从L1删除，在L2中设置
	ml.l1.Delete(ctx, key)
	ml.l2.Set(ctx, key, value, time.Hour)

	exists, err = ml.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if !exists {
		t.Error("Expected key to exist in L2")
	}
}

func TestMultiLevelCache_Clear(t *testing.T) {
	config := Config{
		MultiLevel: MultiLevelConfig{
			EnableL1: true,
			EnableL2: false,
		},
		Memory: MemoryConfig{
			MaxSize:     100,
			EvictPolicy: EvictLRU,
		},
	}

	ml, err := NewMultiLevelCache(config)
	if err != nil {
		t.Fatalf("Failed to create multi-level cache: %v", err)
	}
	defer ml.Close()

	ml.l2 = NewMockCache2()

	ctx := context.Background()

	// 在两个缓存中设置一些值
	for i := 0; i < 5; i++ {
		key := "key" + string(rune('0'+i))
		value := "value" + string(rune('0'+i))
		ml.l1.Set(ctx, key, value, time.Hour)
		ml.l2.Set(ctx, key, value, time.Hour)
	}

	// 清空缓存
	err = ml.Clear(ctx)
	if err != nil {
		t.Fatalf("Failed to clear cache: %v", err)
	}

	// 验证L1为空
	l1Keys, err := ml.l1.Keys(ctx, "*")
	if err != nil {
		t.Fatalf("Failed to get L1 keys: %v", err)
	}
	if len(l1Keys) != 0 {
		t.Errorf("Expected L1 to be empty, got %d keys", len(l1Keys))
	}

	// 验证L2为空
	l2Keys, err := ml.l2.Keys(ctx, "*")
	if err != nil {
		t.Fatalf("Failed to get L2 keys: %v", err)
	}
	if len(l2Keys) != 0 {
		t.Errorf("Expected L2 to be empty, got %d keys", len(l2Keys))
	}
}

func TestMultiLevelCache_Keys(t *testing.T) {
	config := Config{
		MultiLevel: MultiLevelConfig{
			EnableL1: true,
			EnableL2: false,
		},
		Memory: MemoryConfig{
			MaxSize:     100,
			EvictPolicy: EvictLRU,
		},
	}

	ml, err := NewMultiLevelCache(config)
	if err != nil {
		t.Fatalf("Failed to create multi-level cache: %v", err)
	}
	defer ml.Close()

	ml.l2 = NewMockCache2()

	ctx := context.Background()

	// 在L1中设置一些键
	ml.l1.Set(ctx, "key1", "value1", time.Hour)
	ml.l1.Set(ctx, "key2", "value2", time.Hour)

	// 在L2中设置一些键（包括重复的）
	ml.l2.Set(ctx, "key2", "value2", time.Hour)
	ml.l2.Set(ctx, "key3", "value3", time.Hour)

	keys, err := ml.Keys(ctx, "*")
	if err != nil {
		t.Fatalf("Failed to get keys: %v", err)
	}

	// 验证去重后的键
	expectedKeys := map[string]bool{
		"key1": true,
		"key2": true,
		"key3": true,
	}

	if len(keys) != len(expectedKeys) {
		t.Errorf("Expected %d unique keys, got %d", len(expectedKeys), len(keys))
	}

	for _, key := range keys {
		if !expectedKeys[key] {
			t.Errorf("Unexpected key: %s", key)
		}
	}
}

func TestMultiLevelCache_Stats(t *testing.T) {
	config := Config{
		MultiLevel: MultiLevelConfig{
			EnableL1: true,
			EnableL2: false,
		},
		Memory: MemoryConfig{
			MaxSize:     100,
			EvictPolicy: EvictLRU,
		},
	}

	ml, err := NewMultiLevelCache(config)
	if err != nil {
		t.Fatalf("Failed to create multi-level cache: %v", err)
	}
	defer ml.Close()

	ml.l2 = NewMockCache2()

	ctx := context.Background()

	// 执行一些操作产生统计数据
	ml.Set(ctx, "key1", "value1", time.Hour)
	ml.Set(ctx, "key2", "value2", time.Hour)
	ml.Get(ctx, "key1") // hit
	ml.Get(ctx, "key3") // miss

	stats := ml.Stats()

	// 验证统计数据聚合
	if stats.Sets == 0 {
		t.Error("Expected some set operations")
	}

	if stats.Hits == 0 {
		t.Error("Expected some hits")
	}

	if stats.Misses == 0 {
		t.Error("Expected some misses")
	}

	if stats.TotalRequests == 0 {
		t.Error("Expected some total requests")
	}

	if stats.HitRate == 0 {
		t.Error("Expected hit rate to be calculated")
	}
}

func TestMultiLevelCache_GetMultiLevelStats(t *testing.T) {
	config := Config{
		MultiLevel: MultiLevelConfig{
			EnableL1: true,
			EnableL2: false,
		},
		Memory: MemoryConfig{
			MaxSize:     100,
			EvictPolicy: EvictLRU,
		},
	}

	ml, err := NewMultiLevelCache(config)
	if err != nil {
		t.Fatalf("Failed to create multi-level cache: %v", err)
	}
	defer ml.Close()

	ml.l2 = NewMockCache2()

	ctx := context.Background()

	// 执行一些操作
	ml.Set(ctx, "key1", "value1", time.Hour) // L1和L2设置
	ml.Get(ctx, "key1")                      // L1命中

	// 清除L1，从L2获取
	ml.l1.Clear(ctx)
	ml.Get(ctx, "key1") // L2命中

	stats := ml.GetMultiLevelStats()

	if stats.l1Hits != 1 {
		t.Errorf("Expected 1 L1 hit, got %d", stats.l1Hits)
	}

	if stats.l2Hits != 1 {
		t.Errorf("Expected 1 L2 hit, got %d", stats.l2Hits)
	}

	if stats.l1Sets == 0 {
		t.Error("Expected some L1 sets")
	}

	if stats.l2Sets == 0 {
		t.Error("Expected some L2 sets")
	}
}

func TestMultiLevelCache_Sync(t *testing.T) {
	config := Config{
		MultiLevel: MultiLevelConfig{
			EnableL1: true,
			EnableL2: false,
			L1TTL:    30 * time.Minute,
		},
		Memory: MemoryConfig{
			MaxSize:     100,
			EvictPolicy: EvictLRU,
		},
	}

	ml, err := NewMultiLevelCache(config)
	if err != nil {
		t.Fatalf("Failed to create multi-level cache: %v", err)
	}
	defer ml.Close()

	ml.l2 = NewMockCache2()

	ctx := context.Background()

	// 在L2中设置一些值
	ml.l2.Set(ctx, "key1", "value1", time.Hour)
	ml.l2.Set(ctx, "key2", "value2", time.Hour)

	// 同步到L1
	err = ml.Sync(ctx)
	if err != nil {
		t.Fatalf("Failed to sync: %v", err)
	}

	// 验证L1中有数据
	l1Val1, err := ml.l1.Get(ctx, "key1")
	if err != nil {
		t.Errorf("Expected key1 in L1 after sync, got error: %v", err)
	}
	if l1Val1 != "value1" {
		t.Errorf("Expected L1 value1, got %v", l1Val1)
	}

	l1Val2, err := ml.l1.Get(ctx, "key2")
	if err != nil {
		t.Errorf("Expected key2 in L1 after sync, got error: %v", err)
	}
	if l1Val2 != "value2" {
		t.Errorf("Expected L1 value2, got %v", l1Val2)
	}

	// 验证同步统计
	stats := ml.GetMultiLevelStats()
	if stats.syncs == 0 {
		t.Error("Expected sync count > 0")
	}
}

func TestMultiLevelCache_Close(t *testing.T) {
	config := Config{
		MultiLevel: MultiLevelConfig{
			EnableL1: true,
			EnableL2: false,
		},
		Memory: MemoryConfig{
			MaxSize:     100,
			EvictPolicy: EvictLRU,
		},
	}

	ml, err := NewMultiLevelCache(config)
	if err != nil {
		t.Fatalf("Failed to create multi-level cache: %v", err)
	}

	ml.l2 = NewMockCache2()

	// 关闭缓存
	err = ml.Close()
	if err != nil {
		t.Fatalf("Failed to close cache: %v", err)
	}

	// 验证关闭状态
	ctx := context.Background()
	_, err = ml.Get(ctx, "test")
	if err != ErrCacheClosed {
		t.Errorf("Expected ErrCacheClosed, got %v", err)
	}

	err = ml.Set(ctx, "test", "value", time.Hour)
	if err != ErrCacheClosed {
		t.Errorf("Expected ErrCacheClosed, got %v", err)
	}

	// 重复关闭应该不报错
	err = ml.Close()
	if err != nil {
		t.Errorf("Repeated close should not error, got %v", err)
	}
}

func TestMultiLevelCache_AccessorMethods(t *testing.T) {
	config := Config{
		MultiLevel: MultiLevelConfig{
			EnableL1: true,
			EnableL2: false,
		},
		Memory: MemoryConfig{
			MaxSize:     100,
			EvictPolicy: EvictLRU,
		},
	}

	ml, err := NewMultiLevelCache(config)
	if err != nil {
		t.Fatalf("Failed to create multi-level cache: %v", err)
	}
	defer ml.Close()

	// 测试GetL1Cache
	l1 := ml.GetL1Cache()
	if l1 == nil {
		t.Error("GetL1Cache should return L1 cache instance")
	}
	if l1 != ml.l1 {
		t.Error("GetL1Cache should return the same instance as ml.l1")
	}

	// 测试GetL2Cache
	l2 := ml.GetL2Cache()
	if l2 != ml.l2 {
		t.Error("GetL2Cache should return the same instance as ml.l2")
	}

	// 测试Invalidate（应该等同于Delete）
	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	ml.l1.Set(ctx, key, value, time.Hour)

	err = ml.Invalidate(ctx, key)
	if err != nil {
		t.Fatalf("Failed to invalidate: %v", err)
	}

	_, err = ml.l1.Get(ctx, key)
	if err != ErrKeyNotFound {
		t.Error("Expected key to be invalidated")
	}
}

func TestMultiLevelCache_ErrorHandling(t *testing.T) {
	config := Config{
		MultiLevel: MultiLevelConfig{
			EnableL1: true,
			EnableL2: false,
		},
		Memory: MemoryConfig{
			MaxSize:     100,
			EvictPolicy: EvictLRU,
		},
	}

	ml, err := NewMultiLevelCache(config)
	if err != nil {
		t.Fatalf("Failed to create multi-level cache: %v", err)
	}
	defer ml.Close()

	// 创建错误的模拟缓存
	errorCache := NewMockCache2()
	errorCache.setErr = ErrOperationFailed
	ml.l2 = errorCache

	ctx := context.Background()

	// 测试Set错误处理
	err = ml.Set(ctx, "key", "value", time.Hour)
	if err == nil {
		t.Error("Expected error when L2 set fails")
	}

	// 验证错误统计
	stats := ml.GetMultiLevelStats()
	if stats.errors == 0 {
		t.Error("Expected error count > 0")
	}
}

// 性能测试
func BenchmarkMultiLevelCache_Get_L1Hit(b *testing.B) {
	config := Config{
		MultiLevel: MultiLevelConfig{
			EnableL1: true,
			EnableL2: false,
		},
		Memory: MemoryConfig{
			MaxSize:     10000,
			EvictPolicy: EvictLRU,
		},
	}

	ml, err := NewMultiLevelCache(config)
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer ml.Close()

	ctx := context.Background()
	key := "benchmark_key"
	value := "benchmark_value"

	// 预设值到L1
	ml.l1.Set(ctx, key, value, time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ml.Get(ctx, key)
	}
}

func BenchmarkMultiLevelCache_Set_WriteThrough(b *testing.B) {
	config := Config{
		MultiLevel: MultiLevelConfig{
			EnableL1:     true,
			EnableL2:     false,
			SyncStrategy: SyncWriteThrough,
		},
		Memory: MemoryConfig{
			MaxSize:     10000,
			EvictPolicy: EvictLRU,
		},
	}

	ml, err := NewMultiLevelCache(config)
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer ml.Close()

	ml.l2 = NewMockCache2()

	ctx := context.Background()
	value := "benchmark_value"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "key_" + string(rune(i%1000))
		ml.Set(ctx, key, value, time.Hour)
	}
}
