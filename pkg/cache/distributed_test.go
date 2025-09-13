package cache

import (
	"context"
	"testing"
	"time"
)

// MockConsistencyManager 模拟一致性管理器
type MockConsistencyManager struct {
	ensureConsistencyFunc func(ctx context.Context, key string, operation string) error
	checkConsistencyFunc  func(ctx context.Context, keys []string) ([]string, error)
	invalidateFunc        func(ctx context.Context, key string) error
}

func (m *MockConsistencyManager) EnsureConsistency(ctx context.Context, key string, operation string) error {
	if m.ensureConsistencyFunc != nil {
		return m.ensureConsistencyFunc(ctx, key, operation)
	}
	return nil
}

func (m *MockConsistencyManager) CheckConsistency(ctx context.Context, keys []string) ([]string, error) {
	if m.checkConsistencyFunc != nil {
		return m.checkConsistencyFunc(ctx, keys)
	}
	return []string{}, nil
}

func (m *MockConsistencyManager) Invalidate(ctx context.Context, key string) error {
	if m.invalidateFunc != nil {
		return m.invalidateFunc(ctx, key)
	}
	return nil
}

// createTestDistributedConfig 创建测试用的分布式配置
func createTestDistributedConfig() Config {
	return Config{
		Type:       TypeMultiLevel,
		DefaultTTL: time.Hour,
		Memory: MemoryConfig{
			MaxSize:         100,
			MaxMemory:       1024 * 1024,
			EvictPolicy:     EvictLRU,
			CleanupInterval: time.Minute,
			EnableStats:     true,
		},
		MultiLevel: MultiLevelConfig{
			EnableL1:     true,
			EnableL2:     false, // 禁用Redis以简化测试
			L1TTL:        30 * time.Minute,
			L2TTL:        2 * time.Hour,
			SyncStrategy: SyncWriteThrough,
		},
		Distributed: DistributedConfig{
			EnableSync:       true,
			SyncInterval:     5 * time.Second,
			ConsistencyLevel: ConsistencyEventual,
			NodeID:           "test-node-1",
			BroadcastAddr:    "localhost:8080",
		},
	}
}

func TestNewDistributedCache(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantError bool
	}{
		{
			name:      "有效配置",
			config:    createTestDistributedConfig(),
			wantError: false,
		},
		{
			name: "禁用分布式同步",
			config: func() Config {
				config := createTestDistributedConfig()
				config.Distributed.EnableSync = false
				return config
			}(),
			wantError: true,
		},
		{
			name: "强一致性配置",
			config: func() Config {
				config := createTestDistributedConfig()
				config.Distributed.ConsistencyLevel = ConsistencyStrong
				return config
			}(),
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc, err := NewDistributedCache(tt.config)
			if tt.wantError {
				if err == nil {
					t.Error("期望错误，但没有返回错误")
				}
				return
			}

			if err != nil {
				t.Errorf("NewDistributedCache() error = %v", err)
				return
			}

			if dc == nil {
				t.Error("返回的分布式缓存实例为nil")
				return
			}

			// 验证组件是否正确初始化
			if dc.localCache == nil {
				t.Error("本地缓存未初始化")
			}

			if dc.nodeManager == nil {
				t.Error("节点管理器未初始化")
			}

			if dc.eventBus == nil {
				t.Error("事件总线未初始化")
			}

			if dc.consistency == nil {
				t.Error("一致性管理器未初始化")
			}

			if dc.synchronizer == nil {
				t.Error("同步器未初始化")
			}

			// 清理
			dc.Close()
		})
	}
}

func TestDistributedCache_Set_Get(t *testing.T) {
	config := createTestDistributedConfig()
	dc, err := NewDistributedCache(config)
	if err != nil {
		t.Fatalf("创建分布式缓存失败: %v", err)
	}
	defer dc.Close()

	ctx := context.Background()

	tests := []struct {
		name  string
		key   string
		value interface{}
		ttl   time.Duration
	}{
		{
			name:  "字符串值",
			key:   "string_key",
			value: "test_value",
			ttl:   time.Hour,
		},
		{
			name:  "整数值",
			key:   "int_key",
			value: 123,
			ttl:   30 * time.Minute,
		},
		{
			name:  "复杂对象",
			key:   "object_key",
			value: map[string]interface{}{"name": "test", "age": 25},
			ttl:   0, // 永不过期
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置值
			err := dc.Set(ctx, tt.key, tt.value, tt.ttl)
			if err != nil {
				t.Errorf("Set() error = %v", err)
				return
			}

			// 获取值
			got, err := dc.Get(ctx, tt.key)
			if err != nil {
				t.Errorf("Get() error = %v", err)
				return
			}

			// 验证值（特殊处理不可比较的类型）
			switch expectedValue := tt.value.(type) {
			case map[string]interface{}:
				if gotMap, ok := got.(map[string]interface{}); ok {
					if len(gotMap) != len(expectedValue) {
						t.Errorf("Get() map length = %d, want %d", len(gotMap), len(expectedValue))
					}
					for k, v := range expectedValue {
						if gotMap[k] != v {
							t.Errorf("Get() map[%s] = %v, want %v", k, gotMap[k], v)
						}
					}
				} else {
					t.Errorf("Get() = %v (type %T), expected map[string]interface{}", got, got)
				}
			default:
				if got != tt.value {
					t.Errorf("Get() = %v, want %v", got, tt.value)
				}
			}
		})
	}
}

func TestDistributedCache_Get_KeyNotFound(t *testing.T) {
	config := createTestDistributedConfig()
	dc, err := NewDistributedCache(config)
	if err != nil {
		t.Fatalf("创建分布式缓存失败: %v", err)
	}
	defer dc.Close()

	ctx := context.Background()

	_, err = dc.Get(ctx, "non_existent_key")
	if err != ErrKeyNotFound {
		t.Errorf("Get() error = %v, want %v", err, ErrKeyNotFound)
	}
}

func TestDistributedCache_Delete(t *testing.T) {
	config := createTestDistributedConfig()
	dc, err := NewDistributedCache(config)
	if err != nil {
		t.Fatalf("创建分布式缓存失败: %v", err)
	}
	defer dc.Close()

	ctx := context.Background()

	// 首先设置一个值
	key := "delete_test_key"
	value := "delete_test_value"
	err = dc.Set(ctx, key, value, time.Hour)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// 删除键
	err = dc.Delete(ctx, key)
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	// 验证键不存在
	_, err = dc.Get(ctx, key)
	if err != ErrKeyNotFound {
		t.Errorf("Get() after delete error = %v, want %v", err, ErrKeyNotFound)
	}
}

func TestDistributedCache_Exists(t *testing.T) {
	config := createTestDistributedConfig()
	dc, err := NewDistributedCache(config)
	if err != nil {
		t.Fatalf("创建分布式缓存失败: %v", err)
	}
	defer dc.Close()

	ctx := context.Background()

	// 检查不存在的键
	exists, err := dc.Exists(ctx, "non_existent_key")
	if err != nil {
		t.Errorf("Exists() error = %v", err)
	}
	if exists {
		t.Error("Exists() = true, want false for non-existent key")
	}

	// 设置一个键并检查存在性
	key := "exists_test_key"
	err = dc.Set(ctx, key, "test_value", time.Hour)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	exists, err = dc.Exists(ctx, key)
	if err != nil {
		t.Errorf("Exists() error = %v", err)
	}
	if !exists {
		t.Error("Exists() = false, want true for existing key")
	}
}

func TestDistributedCache_Clear(t *testing.T) {
	config := createTestDistributedConfig()
	dc, err := NewDistributedCache(config)
	if err != nil {
		t.Fatalf("创建分布式缓存失败: %v", err)
	}
	defer dc.Close()

	ctx := context.Background()

	// 设置多个键
	testData := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for key, value := range testData {
		err = dc.Set(ctx, key, value, time.Hour)
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}
	}

	// 清空缓存
	err = dc.Clear(ctx)
	if err != nil {
		t.Errorf("Clear() error = %v", err)
	}

	// 验证所有键都被删除
	for key := range testData {
		exists, err := dc.Exists(ctx, key)
		if err != nil {
			t.Errorf("Exists() error = %v", err)
		}
		if exists {
			t.Errorf("Key %s still exists after Clear()", key)
		}
	}
}

func TestDistributedCache_Keys(t *testing.T) {
	config := createTestDistributedConfig()
	dc, err := NewDistributedCache(config)
	if err != nil {
		t.Fatalf("创建分布式缓存失败: %v", err)
	}
	defer dc.Close()

	ctx := context.Background()

	// 设置测试数据
	testKeys := []string{
		"user:1", "user:2", "user:3",
		"product:1", "product:2",
	}

	for _, key := range testKeys {
		err = dc.Set(ctx, key, "test_value", time.Hour)
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}
	}

	// 获取所有键
	keys, err := dc.Keys(ctx, "*")
	if err != nil {
		t.Errorf("Keys() error = %v", err)
		return
	}

	if len(keys) == 0 {
		t.Error("Keys() returned empty result")
		return
	}

	// 验证返回的键
	keyMap := make(map[string]bool)
	for _, key := range keys {
		keyMap[key] = true
	}

	for _, expectedKey := range testKeys {
		if !keyMap[expectedKey] {
			t.Errorf("Expected key %s not found in result", expectedKey)
		}
	}
}

func TestDistributedCache_Stats(t *testing.T) {
	config := createTestDistributedConfig()
	dc, err := NewDistributedCache(config)
	if err != nil {
		t.Fatalf("创建分布式缓存失败: %v", err)
	}
	defer dc.Close()

	ctx := context.Background()

	// 初始统计应该为零
	stats := dc.Stats()
	if stats.Hits != 0 || stats.Misses != 0 || stats.Sets != 0 {
		t.Error("初始统计不为零")
	}

	// 执行一些操作
	dc.Set(ctx, "key1", "value1", time.Hour)
	dc.Get(ctx, "key1")    // 命中
	dc.Get(ctx, "key2")    // 未命中
	dc.Delete(ctx, "key1") // 删除

	// 检查统计
	stats = dc.Stats()
	if stats.Sets == 0 {
		t.Error("Sets统计应该大于0")
	}
	if stats.TotalRequests == 0 {
		t.Error("TotalRequests统计应该大于0")
	}
}

func TestDistributedCache_Start_Stop(t *testing.T) {
	config := createTestDistributedConfig()
	dc, err := NewDistributedCache(config)
	if err != nil {
		t.Fatalf("创建分布式缓存失败: %v", err)
	}

	ctx := context.Background()

	// 启动分布式缓存
	err = dc.Start(ctx)
	if err != nil {
		t.Errorf("Start() error = %v", err)
	}

	// 停止分布式缓存
	err = dc.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// 多次关闭应该不报错
	err = dc.Close()
	if err != nil {
		t.Errorf("Second Close() error = %v", err)
	}
}

func TestDistributedCache_ClosedOperations(t *testing.T) {
	config := createTestDistributedConfig()
	dc, err := NewDistributedCache(config)
	if err != nil {
		t.Fatalf("创建分布式缓存失败: %v", err)
	}

	// 关闭缓存
	dc.Close()

	ctx := context.Background()

	// 关闭后的操作应该返回错误
	_, err = dc.Get(ctx, "test_key")
	if err != ErrCacheClosed {
		t.Errorf("Get() after close error = %v, want %v", err, ErrCacheClosed)
	}

	err = dc.Set(ctx, "test_key", "test_value", time.Hour)
	if err != ErrCacheClosed {
		t.Errorf("Set() after close error = %v, want %v", err, ErrCacheClosed)
	}

	err = dc.Delete(ctx, "test_key")
	if err != ErrCacheClosed {
		t.Errorf("Delete() after close error = %v, want %v", err, ErrCacheClosed)
	}

	_, err = dc.Exists(ctx, "test_key")
	if err != ErrCacheClosed {
		t.Errorf("Exists() after close error = %v, want %v", err, ErrCacheClosed)
	}

	err = dc.Clear(ctx)
	if err != ErrCacheClosed {
		t.Errorf("Clear() after close error = %v, want %v", err, ErrCacheClosed)
	}

	_, err = dc.Keys(ctx, "*")
	if err != ErrCacheClosed {
		t.Errorf("Keys() after close error = %v, want %v", err, ErrCacheClosed)
	}
}

func TestEventBus(t *testing.T) {
	eventBus := NewEventBus()

	var receivedEvent *CacheEvent
	handler := func(event *CacheEvent) error {
		receivedEvent = event
		return nil
	}

	// 订阅事件
	eventBus.Subscribe("test.event", handler)

	// 发布事件
	testEvent := &CacheEvent{
		Type:      "test",
		Key:       "test_key",
		Value:     "test_value",
		NodeID:    "test_node",
		Timestamp: time.Now(),
	}

	eventBus.Publish("test.event", testEvent)

	// 等待事件处理
	time.Sleep(100 * time.Millisecond)

	// 验证事件被接收
	if receivedEvent == nil {
		t.Error("事件未被接收")
		return
	}

	if receivedEvent.Type != testEvent.Type {
		t.Errorf("事件类型不匹配: got %s, want %s", receivedEvent.Type, testEvent.Type)
	}

	if receivedEvent.Key != testEvent.Key {
		t.Errorf("事件键不匹配: got %s, want %s", receivedEvent.Key, testEvent.Key)
	}
}

func TestNodeManager(t *testing.T) {
	nodeManager := NewNodeManager("test-node")

	if nodeManager.nodeID != "test-node" {
		t.Errorf("节点ID不匹配: got %s, want test-node", nodeManager.nodeID)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动节点管理器
	err := nodeManager.Start(ctx)
	if err != nil {
		t.Errorf("Start() error = %v", err)
	}

	// 停止节点管理器
	nodeManager.Stop()
}

func TestSynchronizer(t *testing.T) {
	config := DistributedConfig{
		SyncInterval: 100 * time.Millisecond,
	}

	eventBus := NewEventBus()
	nodeManager := NewNodeManager("test-node")
	synchronizer := NewSynchronizer(config, eventBus, nodeManager)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动同步器
	err := synchronizer.Start(ctx)
	if err != nil {
		t.Errorf("Start() error = %v", err)
	}

	// 等待一段时间
	time.Sleep(200 * time.Millisecond)

	// 停止同步器
	err = synchronizer.Stop()
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

func TestEventualConsistency(t *testing.T) {
	config := DistributedConfig{
		NodeID: "test-node",
	}

	nodeManager := NewNodeManager("test-node")
	eventBus := NewEventBus()
	ec := NewEventualConsistency(config, nodeManager, eventBus)

	ctx := context.Background()

	// 测试确保一致性
	err := ec.EnsureConsistency(ctx, "test_key", "set")
	if err != nil {
		t.Errorf("EnsureConsistency() error = %v", err)
	}

	// 测试检查一致性
	inconsistentKeys, err := ec.CheckConsistency(ctx, []string{"key1", "key2"})
	if err != nil {
		t.Errorf("CheckConsistency() error = %v", err)
	}
	if len(inconsistentKeys) != 0 {
		t.Errorf("CheckConsistency() returned %d inconsistent keys, want 0", len(inconsistentKeys))
	}

	// 测试失效处理
	err = ec.Invalidate(ctx, "test_key")
	if err != nil {
		t.Errorf("Invalidate() error = %v", err)
	}
}

func TestStrongConsistency(t *testing.T) {
	config := DistributedConfig{
		NodeID: "test-node",
	}

	nodeManager := NewNodeManager("test-node")
	eventBus := NewEventBus()
	sc := NewStrongConsistency(config, nodeManager, eventBus)

	ctx := context.Background()

	// 测试确保一致性
	err := sc.EnsureConsistency(ctx, "test_key", "set")
	if err != nil {
		t.Errorf("EnsureConsistency() error = %v", err)
	}

	// 测试并发锁 - 手动设置锁状态来模拟并发场景
	lockKey := "lock:concurrent_test_key"
	sc.locks.Store(lockKey, time.Now())

	// 尝试获取已被锁定的键
	err = sc.EnsureConsistency(ctx, "concurrent_test_key", "set")
	if err == nil {
		t.Error("并发操作应该被锁阻止")
	}

	// 清理锁
	sc.locks.Delete(lockKey)

	// 测试检查一致性
	inconsistentKeys, err := sc.CheckConsistency(ctx, []string{"key1", "key2"})
	if err != nil {
		t.Errorf("CheckConsistency() error = %v", err)
	}
	if len(inconsistentKeys) != 0 {
		t.Errorf("CheckConsistency() returned %d inconsistent keys, want 0", len(inconsistentKeys))
	}

	// 测试失效处理
	err = sc.Invalidate(ctx, "test_key")
	if err != nil {
		t.Errorf("Invalidate() error = %v", err)
	}
}

func TestDistributedCache_CalculateChecksum(t *testing.T) {
	config := createTestDistributedConfig()
	dc, err := NewDistributedCache(config)
	if err != nil {
		t.Fatalf("创建分布式缓存失败: %v", err)
	}
	defer dc.Close()

	// 测试校验和计算
	checksum1 := dc.calculateChecksum("key1", "value1")
	checksum2 := dc.calculateChecksum("key1", "value1")
	checksum3 := dc.calculateChecksum("key1", "value2")

	// 相同的键值应该产生相同的校验和
	if checksum1 != checksum2 {
		t.Error("相同键值的校验和应该相同")
	}

	// 不同的值应该产生不同的校验和
	if checksum1 == checksum3 {
		t.Error("不同值的校验和应该不同")
	}

	// 校验和应该是64字符的十六进制字符串
	if len(checksum1) != 64 {
		t.Errorf("校验和长度应该是64，实际是%d", len(checksum1))
	}
}

func TestDistributedCache_EventHandlers(t *testing.T) {
	config := createTestDistributedConfig()
	dc, err := NewDistributedCache(config)
	if err != nil {
		t.Fatalf("创建分布式缓存失败: %v", err)
	}
	defer dc.Close()

	// 测试设置事件处理
	setEvent := &CacheEvent{
		Type:      "set",
		Key:       "test_key",
		Value:     "test_value",
		TTL:       time.Hour,
		NodeID:    "other-node", // 不同的节点ID
		Timestamp: time.Now(),
	}

	err = dc.handleSetEvent(setEvent)
	if err != nil {
		t.Errorf("handleSetEvent() error = %v", err)
	}

	// 测试删除事件处理
	deleteEvent := &CacheEvent{
		Type:      "delete",
		Key:       "test_key",
		NodeID:    "other-node",
		Timestamp: time.Now(),
	}

	err = dc.handleDeleteEvent(deleteEvent)
	if err != nil {
		t.Errorf("handleDeleteEvent() error = %v", err)
	}

	// 测试清空事件处理
	clearEvent := &CacheEvent{
		Type:      "clear",
		NodeID:    "other-node",
		Timestamp: time.Now(),
	}

	err = dc.handleClearEvent(clearEvent)
	if err != nil {
		t.Errorf("handleClearEvent() error = %v", err)
	}

	// 测试忽略自己的事件
	ownEvent := &CacheEvent{
		Type:      "set",
		Key:       "test_key",
		Value:     "test_value",
		NodeID:    config.Distributed.NodeID, // 相同的节点ID
		Timestamp: time.Now(),
	}

	err = dc.handleSetEvent(ownEvent)
	if err != nil {
		t.Errorf("handleSetEvent() with own event error = %v", err)
	}
}

// 性能测试
func BenchmarkDistributedCache_Set(b *testing.B) {
	config := createTestDistributedConfig()
	dc, err := NewDistributedCache(config)
	if err != nil {
		b.Fatalf("创建分布式缓存失败: %v", err)
	}
	defer dc.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "benchmark_key"
		dc.Set(ctx, key, "benchmark_value", time.Hour)
	}
}

func BenchmarkDistributedCache_Get(b *testing.B) {
	config := createTestDistributedConfig()
	dc, err := NewDistributedCache(config)
	if err != nil {
		b.Fatalf("创建分布式缓存失败: %v", err)
	}
	defer dc.Close()

	ctx := context.Background()

	// 预设数据
	dc.Set(ctx, "benchmark_key", "benchmark_value", time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dc.Get(ctx, "benchmark_key")
	}
}

func BenchmarkDistributedCache_CalculateChecksum(b *testing.B) {
	config := createTestDistributedConfig()
	dc, err := NewDistributedCache(config)
	if err != nil {
		b.Fatalf("创建分布式缓存失败: %v", err)
	}
	defer dc.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dc.calculateChecksum("benchmark_key", "benchmark_value")
	}
}
