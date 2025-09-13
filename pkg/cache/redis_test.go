package cache

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

// setupTestRedis 设置测试用的Redis服务器
func setupTestRedis(t *testing.T) (*miniredis.Miniredis, RedisConfig) {
	t.Helper()

	// 创建一个内存中的Redis实例用于测试
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("无法启动测试Redis服务器: %v", err)
	}

	config := RedisConfig{
		Addrs:        []string{mr.Addr()},
		PoolSize:     10,
		MinIdleConns: 5,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		KeyPrefix:    "test:",
	}

	return mr, config
}

func TestNewRedisCache(t *testing.T) {
	mr, config := setupTestRedis(t)
	defer mr.Close()

	tests := []struct {
		name      string
		config    RedisConfig
		wantError bool
	}{
		{
			name:      "有效配置",
			config:    config,
			wantError: false,
		},
		{
			name: "集群模式配置",
			config: RedisConfig{
				Addrs:       []string{mr.Addr()},
				ClusterMode: true,
				PoolSize:    10,
				KeyPrefix:   "cluster:",
			},
			wantError: false,
		},
		{
			name: "无效地址",
			config: RedisConfig{
				Addrs:     []string{"invalid:6379"},
				PoolSize:  10,
				KeyPrefix: "test:",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache, err := NewRedisCache(tt.config)
			if tt.wantError {
				if err == nil {
					t.Error("期望错误，但没有返回错误")
				}
				return
			}

			if err != nil {
				t.Errorf("NewRedisCache() error = %v", err)
				return
			}

			if cache == nil {
				t.Error("返回的缓存实例为nil")
				return
			}

			// 清理
			cache.Close()
		})
	}
}

func TestRedisCache_Set_Get(t *testing.T) {
	mr, config := setupTestRedis(t)
	defer mr.Close()

	cache, err := NewRedisCache(config)
	if err != nil {
		t.Fatalf("创建Redis缓存失败: %v", err)
	}
	defer cache.Close()

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
		{
			name:  "布尔值",
			key:   "bool_key",
			value: true,
			ttl:   time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置值
			err := cache.Set(ctx, tt.key, tt.value, tt.ttl)
			if err != nil {
				t.Errorf("Set() error = %v", err)
				return
			}

			// 获取值
			got, err := cache.Get(ctx, tt.key)
			if err != nil {
				t.Errorf("Get() error = %v", err)
				return
			}

			// 验证值（由于JSON序列化的特性，需要特殊处理）
			switch expected := tt.value.(type) {
			case int:
				// JSON会将int转换为float64
				if gotFloat, ok := got.(float64); ok {
					if int(gotFloat) != expected {
						t.Errorf("Get() = %v, want %v", got, tt.value)
					}
				} else {
					t.Errorf("Get() = %v (type %T), expected float64", got, got)
				}
			default:
				// 对于其他类型，进行简单的字符串比较
				if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", tt.value) {
					t.Errorf("Get() = %v, want %v", got, tt.value)
				}
			}
		})
	}
}

func TestRedisCache_Get_KeyNotFound(t *testing.T) {
	mr, config := setupTestRedis(t)
	defer mr.Close()

	cache, err := NewRedisCache(config)
	if err != nil {
		t.Fatalf("创建Redis缓存失败: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	_, err = cache.Get(ctx, "non_existent_key")
	if err != ErrKeyNotFound {
		t.Errorf("Get() error = %v, want %v", err, ErrKeyNotFound)
	}
}

func TestRedisCache_Delete(t *testing.T) {
	mr, config := setupTestRedis(t)
	defer mr.Close()

	cache, err := NewRedisCache(config)
	if err != nil {
		t.Fatalf("创建Redis缓存失败: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// 首先设置一个值
	key := "delete_test_key"
	value := "delete_test_value"
	err = cache.Set(ctx, key, value, time.Hour)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// 删除键
	err = cache.Delete(ctx, key)
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	// 验证键不存在
	_, err = cache.Get(ctx, key)
	if err != ErrKeyNotFound {
		t.Errorf("Get() after delete error = %v, want %v", err, ErrKeyNotFound)
	}

	// 删除不存在的键
	err = cache.Delete(ctx, "non_existent_key")
	if err != ErrKeyNotFound {
		t.Errorf("Delete() non-existent key error = %v, want %v", err, ErrKeyNotFound)
	}
}

func TestRedisCache_Exists(t *testing.T) {
	mr, config := setupTestRedis(t)
	defer mr.Close()

	cache, err := NewRedisCache(config)
	if err != nil {
		t.Fatalf("创建Redis缓存失败: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// 检查不存在的键
	exists, err := cache.Exists(ctx, "non_existent_key")
	if err != nil {
		t.Errorf("Exists() error = %v", err)
	}
	if exists {
		t.Error("Exists() = true, want false for non-existent key")
	}

	// 设置一个键并检查存在性
	key := "exists_test_key"
	err = cache.Set(ctx, key, "test_value", time.Hour)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	exists, err = cache.Exists(ctx, key)
	if err != nil {
		t.Errorf("Exists() error = %v", err)
	}
	if !exists {
		t.Error("Exists() = false, want true for existing key")
	}
}

func TestRedisCache_Clear(t *testing.T) {
	mr, config := setupTestRedis(t)
	defer mr.Close()

	cache, err := NewRedisCache(config)
	if err != nil {
		t.Fatalf("创建Redis缓存失败: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// 设置多个键
	testData := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for key, value := range testData {
		err = cache.Set(ctx, key, value, time.Hour)
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}
	}

	// 清空缓存
	err = cache.Clear(ctx)
	if err != nil {
		t.Errorf("Clear() error = %v", err)
	}

	// 验证所有键都被删除
	for key := range testData {
		exists, err := cache.Exists(ctx, key)
		if err != nil {
			t.Errorf("Exists() error = %v", err)
		}
		if exists {
			t.Errorf("Key %s still exists after Clear()", key)
		}
	}
}

func TestRedisCache_Keys(t *testing.T) {
	mr, config := setupTestRedis(t)
	defer mr.Close()

	cache, err := NewRedisCache(config)
	if err != nil {
		t.Fatalf("创建Redis缓存失败: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// 设置测试数据
	testKeys := []string{
		"user:1", "user:2", "user:3",
		"product:1", "product:2",
	}

	for _, key := range testKeys {
		err = cache.Set(ctx, key, "test_value", time.Hour)
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}
	}

	tests := []struct {
		name         string
		pattern      string
		expectedKeys []string
	}{
		{
			name:         "所有键",
			pattern:      "*",
			expectedKeys: testKeys,
		},
		{
			name:         "用户键",
			pattern:      "user:*",
			expectedKeys: []string{"user:1", "user:2", "user:3"},
		},
		{
			name:         "产品键",
			pattern:      "product:*",
			expectedKeys: []string{"product:1", "product:2"},
		},
		{
			name:         "不匹配的模式",
			pattern:      "nonexistent:*",
			expectedKeys: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys, err := cache.Keys(ctx, tt.pattern)
			if err != nil {
				t.Errorf("Keys() error = %v", err)
				return
			}

			if len(keys) != len(tt.expectedKeys) {
				t.Errorf("Keys() returned %d keys, want %d", len(keys), len(tt.expectedKeys))
				return
			}

			// 验证返回的键
			keyMap := make(map[string]bool)
			for _, key := range keys {
				keyMap[key] = true
			}

			for _, expectedKey := range tt.expectedKeys {
				if !keyMap[expectedKey] {
					t.Errorf("Expected key %s not found in result", expectedKey)
				}
			}
		})
	}
}

func TestRedisCache_Stats(t *testing.T) {
	mr, config := setupTestRedis(t)
	defer mr.Close()

	cache, err := NewRedisCache(config)
	if err != nil {
		t.Fatalf("创建Redis缓存失败: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// 初始统计应该为零
	stats := cache.Stats()
	if stats.Hits != 0 || stats.Misses != 0 || stats.Sets != 0 {
		t.Error("初始统计不为零")
	}

	// 执行一些操作
	cache.Set(ctx, "key1", "value1", time.Hour)
	cache.Get(ctx, "key1")    // 命中
	cache.Get(ctx, "key2")    // 未命中
	cache.Delete(ctx, "key1") // 删除

	// 检查统计
	stats = cache.Stats()
	if stats.Sets != 1 {
		t.Errorf("Sets = %d, want 1", stats.Sets)
	}
	if stats.Hits != 1 {
		t.Errorf("Hits = %d, want 1", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Misses = %d, want 1", stats.Misses)
	}
	if stats.Deletes != 1 {
		t.Errorf("Deletes = %d, want 1", stats.Deletes)
	}
	if stats.TotalRequests != 2 {
		t.Errorf("TotalRequests = %d, want 2", stats.TotalRequests)
	}
	if stats.HitRate != 0.5 {
		t.Errorf("HitRate = %f, want 0.5", stats.HitRate)
	}
}

func TestRedisCache_MGet(t *testing.T) {
	mr, config := setupTestRedis(t)
	defer mr.Close()

	cache, err := NewRedisCache(config)
	if err != nil {
		t.Fatalf("创建Redis缓存失败: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// 设置测试数据
	testData := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for key, value := range testData {
		err = cache.Set(ctx, key, value, time.Hour)
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}
	}

	// 测试批量获取
	keys := []string{"key1", "key2", "key3", "key4"} // key4不存在
	results, err := cache.MGet(ctx, keys...)
	if err != nil {
		t.Errorf("MGet() error = %v", err)
		return
	}

	// 验证结果
	if len(results) != 3 { // 只有3个键存在
		t.Errorf("MGet() returned %d results, want 3", len(results))
	}

	for key, expectedValue := range testData {
		if value, exists := results[key]; !exists {
			t.Errorf("Key %s not found in MGet results", key)
		} else if fmt.Sprintf("%v", value) != fmt.Sprintf("%v", expectedValue) {
			t.Errorf("MGet() key %s = %v, want %v", key, value, expectedValue)
		}
	}

	// 测试空键列表
	emptyResults, err := cache.MGet(ctx)
	if err != nil {
		t.Errorf("MGet() with empty keys error = %v", err)
	}
	if len(emptyResults) != 0 {
		t.Error("MGet() with empty keys should return empty map")
	}
}

func TestRedisCache_MSet(t *testing.T) {
	mr, config := setupTestRedis(t)
	defer mr.Close()

	cache, err := NewRedisCache(config)
	if err != nil {
		t.Fatalf("创建Redis缓存失败: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// 测试数据
	testData := map[string]interface{}{
		"batch_key1": "batch_value1",
		"batch_key2": "batch_value2",
		"batch_key3": "batch_value3",
	}

	// 批量设置
	err = cache.MSet(ctx, testData, time.Hour)
	if err != nil {
		t.Errorf("MSet() error = %v", err)
		return
	}

	// 验证所有键都被设置
	for key, expectedValue := range testData {
		value, err := cache.Get(ctx, key)
		if err != nil {
			t.Errorf("Get() after MSet error = %v", err)
		} else if fmt.Sprintf("%v", value) != fmt.Sprintf("%v", expectedValue) {
			t.Errorf("Get() after MSet key %s = %v, want %v", key, value, expectedValue)
		}
	}

	// 测试空数据
	err = cache.MSet(ctx, map[string]interface{}{}, time.Hour)
	if err != nil {
		t.Errorf("MSet() with empty data error = %v", err)
	}
}

func TestRedisCache_Pipeline(t *testing.T) {
	mr, config := setupTestRedis(t)
	defer mr.Close()

	cache, err := NewRedisCache(config)
	if err != nil {
		t.Fatalf("创建Redis缓存失败: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// 使用管道设置多个键
	err = cache.Pipeline(ctx, func(pipe redis.Pipeliner) error {
		pipe.Set(ctx, cache.buildKey("pipe_key1"), "pipe_value1", time.Hour)
		pipe.Set(ctx, cache.buildKey("pipe_key2"), "pipe_value2", time.Hour)
		return nil
	})

	if err != nil {
		t.Errorf("Pipeline() error = %v", err)
	}

	// 验证键被设置（直接使用Redis客户端检查）
	client := cache.GetClient()
	result1, err := client.Get(ctx, cache.buildKey("pipe_key1")).Result()
	if err != nil {
		t.Errorf("Failed to get pipe_key1: %v", err)
	}
	if result1 != "pipe_value1" {
		t.Errorf("pipe_key1 = %s, want pipe_value1", result1)
	}
}

func TestRedisCache_Close(t *testing.T) {
	mr, config := setupTestRedis(t)
	defer mr.Close()

	cache, err := NewRedisCache(config)
	if err != nil {
		t.Fatalf("创建Redis缓存失败: %v", err)
	}

	// 关闭缓存
	err = cache.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// 多次关闭应该不报错
	err = cache.Close()
	if err != nil {
		t.Errorf("Second Close() error = %v", err)
	}

	ctx := context.Background()

	// 关闭后的操作应该返回错误
	_, err = cache.Get(ctx, "test_key")
	if err != ErrCacheClosed {
		t.Errorf("Get() after close error = %v, want %v", err, ErrCacheClosed)
	}

	err = cache.Set(ctx, "test_key", "test_value", time.Hour)
	if err != ErrCacheClosed {
		t.Errorf("Set() after close error = %v, want %v", err, ErrCacheClosed)
	}
}

func TestRedisCache_GetClient(t *testing.T) {
	mr, config := setupTestRedis(t)
	defer mr.Close()

	cache, err := NewRedisCache(config)
	if err != nil {
		t.Fatalf("创建Redis缓存失败: %v", err)
	}
	defer cache.Close()

	client := cache.GetClient()
	if client == nil {
		t.Error("GetClient() returned nil")
	}

	// 验证客户端可以使用
	ctx := context.Background()
	err = client.Set(ctx, "direct_key", "direct_value", time.Hour).Err()
	if err != nil {
		t.Errorf("Direct client Set() error = %v", err)
	}

	result, err := client.Get(ctx, "direct_key").Result()
	if err != nil {
		t.Errorf("Direct client Get() error = %v", err)
	}
	if result != "direct_value" {
		t.Errorf("Direct client Get() = %s, want direct_value", result)
	}
}

func TestRedisCache_BuildKey(t *testing.T) {
	config := RedisConfig{
		Addrs:     []string{"localhost:6379"},
		KeyPrefix: "test:",
	}

	cache := &RedisCache{
		config:    config,
		keyPrefix: config.KeyPrefix,
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"key1", "test:key1"},
		{"user:123", "test:user:123"},
		{"", "test:"},
	}

	for _, tt := range tests {
		result := cache.buildKey(tt.input)
		if result != tt.expected {
			t.Errorf("buildKey(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

// 性能测试
func BenchmarkRedisCache_Set(b *testing.B) {
	mr, err := miniredis.Run()
	if err != nil {
		b.Fatalf("无法启动测试Redis服务器: %v", err)
	}
	defer mr.Close()

	config := RedisConfig{
		Addrs:     []string{mr.Addr()},
		KeyPrefix: "bench:",
	}

	cache, err := NewRedisCache(config)
	if err != nil {
		b.Fatalf("创建Redis缓存失败: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key_%d", i)
		cache.Set(ctx, key, "test_value", time.Hour)
	}
}

func BenchmarkRedisCache_Get(b *testing.B) {
	mr, err := miniredis.Run()
	if err != nil {
		b.Fatalf("无法启动测试Redis服务器: %v", err)
	}
	defer mr.Close()

	config := RedisConfig{
		Addrs:     []string{mr.Addr()},
		KeyPrefix: "bench:",
	}

	cache, err := NewRedisCache(config)
	if err != nil {
		b.Fatalf("创建Redis缓存失败: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// 预设数据
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key_%d", i)
		cache.Set(ctx, key, "test_value", time.Hour)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key_%d", i%1000)
		cache.Get(ctx, key)
	}
}

func BenchmarkRedisCache_MGet(b *testing.B) {
	mr, err := miniredis.Run()
	if err != nil {
		b.Fatalf("无法启动测试Redis服务器: %v", err)
	}
	defer mr.Close()

	config := RedisConfig{
		Addrs:     []string{mr.Addr()},
		KeyPrefix: "bench:",
	}

	cache, err := NewRedisCache(config)
	if err != nil {
		b.Fatalf("创建Redis缓存失败: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// 预设数据
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key_%d", i)
		cache.Set(ctx, key, "test_value", time.Hour)
	}

	keys := make([]string, 10)
	for i := 0; i < 10; i++ {
		keys[i] = fmt.Sprintf("key_%d", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.MGet(ctx, keys...)
	}
}
