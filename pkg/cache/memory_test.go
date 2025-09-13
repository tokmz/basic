package cache

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestMemoryCache_SetGet(t *testing.T) {
	config := MemoryConfig{
		MaxSize:         100,
		MaxMemory:       1024 * 1024, // 1MB
		EvictPolicy:     EvictLRU,
		CleanupInterval: time.Minute,
		EnableStats:     true,
	}

	cache := NewMemoryCache(config)
	defer cache.Close()

	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	// 测试设置
	err := cache.Set(ctx, key, value, time.Hour)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// 测试获取
	result, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get cache: %v", err)
	}

	if result != value {
		t.Errorf("Expected %v, got %v", value, result)
	}
}

func TestMemoryCache_TTL(t *testing.T) {
	config := MemoryConfig{
		MaxSize:     100,
		EvictPolicy: EvictLRU,
		EnableStats: true,
	}

	cache := NewMemoryCache(config)
	defer cache.Close()

	ctx := context.Background()
	key := "ttl_key"
	value := "ttl_value"
	ttl := 100 * time.Millisecond

	// 设置短TTL的缓存
	err := cache.Set(ctx, key, value, ttl)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// 立即获取应该成功
	result, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get cache immediately: %v", err)
	}
	if result != value {
		t.Errorf("Expected %v, got %v", value, result)
	}

	// 等待过期
	time.Sleep(ttl + 50*time.Millisecond)

	// 获取过期的缓存应该失败
	_, err = cache.Get(ctx, key)
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound, got %v", err)
	}
}

func TestMemoryCache_Delete(t *testing.T) {
	config := MemoryConfig{
		MaxSize:     100,
		EvictPolicy: EvictLRU,
		EnableStats: true,
	}

	cache := NewMemoryCache(config)
	defer cache.Close()

	ctx := context.Background()
	key := "delete_key"
	value := "delete_value"

	// 设置缓存
	err := cache.Set(ctx, key, value, time.Hour)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// 验证存在
	exists, err := cache.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if !exists {
		t.Error("Key should exist")
	}

	// 删除
	err = cache.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Failed to delete cache: %v", err)
	}

	// 验证不存在
	exists, err = cache.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Failed to check existence after delete: %v", err)
	}
	if exists {
		t.Error("Key should not exist after delete")
	}
}

func TestMemoryCache_Eviction(t *testing.T) {
	config := MemoryConfig{
		MaxSize:     3, // 限制为3个项目
		EvictPolicy: EvictLRU,
		EnableStats: true,
	}

	cache := NewMemoryCache(config)
	defer cache.Close()

	ctx := context.Background()

	// 设置4个缓存项
	keys := []string{"key1", "key2", "key3", "key4"}
	for _, key := range keys {
		err := cache.Set(ctx, key, "value_"+key, time.Hour)
		if err != nil {
			t.Fatalf("Failed to set cache for %s: %v", key, err)
		}
	}

	// 第一个键应该被淘汰
	_, err := cache.Get(ctx, "key1")
	if err != ErrKeyNotFound {
		t.Error("key1 should have been evicted")
	}

	// 其他键应该存在
	for _, key := range keys[1:] {
		_, err := cache.Get(ctx, key)
		if err != nil {
			t.Errorf("Key %s should exist: %v", key, err)
		}
	}
}

func TestMemoryCache_Stats(t *testing.T) {
	config := MemoryConfig{
		MaxSize:     100,
		EvictPolicy: EvictLRU,
		EnableStats: true,
	}

	cache := NewMemoryCache(config)
	defer cache.Close()

	ctx := context.Background()

	// 执行一些操作
	cache.Set(ctx, "key1", "value1", time.Hour)
	cache.Set(ctx, "key2", "value2", time.Hour)
	cache.Get(ctx, "key1") // hit
	cache.Get(ctx, "key3") // miss
	cache.Delete(ctx, "key1")

	stats := cache.Stats()

	if stats.Sets != 2 {
		t.Errorf("Expected 2 sets, got %d", stats.Sets)
	}

	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}

	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}

	if stats.Deletes != 1 {
		t.Errorf("Expected 1 delete, got %d", stats.Deletes)
	}

	if stats.TotalRequests != 2 { // Get操作的总数
		t.Errorf("Expected 2 total requests, got %d", stats.TotalRequests)
	}
}

func TestMemoryCache_Keys(t *testing.T) {
	config := MemoryConfig{
		MaxSize:     100,
		EvictPolicy: EvictLRU,
		EnableStats: true,
	}

	cache := NewMemoryCache(config)
	defer cache.Close()

	ctx := context.Background()

	// 设置一些缓存项
	testKeys := []string{"user:1", "user:2", "order:1", "order:2"}
	for _, key := range testKeys {
		cache.Set(ctx, key, "value", time.Hour)
	}

	// 获取所有键
	allKeys, err := cache.Keys(ctx, "*")
	if err != nil {
		t.Fatalf("Failed to get all keys: %v", err)
	}

	if len(allKeys) != len(testKeys) {
		t.Errorf("Expected %d keys, got %d", len(testKeys), len(allKeys))
	}

	// 获取用户相关的键
	userKeys, err := cache.Keys(ctx, "user:*")
	if err != nil {
		t.Fatalf("Failed to get user keys: %v", err)
	}

	if len(userKeys) != 2 {
		t.Errorf("Expected 2 user keys, got %d", len(userKeys))
	}
}

func TestMemoryCache_Clear(t *testing.T) {
	config := MemoryConfig{
		MaxSize:     100,
		EvictPolicy: EvictLRU,
		EnableStats: true,
	}

	cache := NewMemoryCache(config)
	defer cache.Close()

	ctx := context.Background()

	// 设置一些缓存项
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key%d", i)
		cache.Set(ctx, key, "value", time.Hour)
	}

	// 清空缓存
	err := cache.Clear(ctx)
	if err != nil {
		t.Fatalf("Failed to clear cache: %v", err)
	}

	// 验证缓存为空
	keys, err := cache.Keys(ctx, "*")
	if err != nil {
		t.Fatalf("Failed to get keys after clear: %v", err)
	}

	if len(keys) != 0 {
		t.Errorf("Expected 0 keys after clear, got %d", len(keys))
	}
}

func TestMemoryCache_Concurrent(t *testing.T) {
	config := MemoryConfig{
		MaxSize:     1000,
		EvictPolicy: EvictLRU,
		EnableStats: true,
	}

	cache := NewMemoryCache(config)
	defer cache.Close()

	ctx := context.Background()
	const goroutines = 10
	const operations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	// 并发设置和获取
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < operations; i++ {
				key := fmt.Sprintf("key_%d_%d", id, i)
				value := fmt.Sprintf("value_%d_%d", id, i)

				// 设置
				err := cache.Set(ctx, key, value, time.Hour)
				if err != nil {
					t.Errorf("Failed to set in goroutine %d: %v", id, err)
					return
				}

				// 获取
				result, err := cache.Get(ctx, key)
				if err != nil {
					t.Errorf("Failed to get in goroutine %d: %v", id, err)
					return
				}

				if result != value {
					t.Errorf("Value mismatch in goroutine %d: expected %v, got %v", id, value, result)
					return
				}
			}
		}(g)
	}

	wg.Wait()
}

func TestMemoryCache_CleanupExpired(t *testing.T) {
	config := MemoryConfig{
		MaxSize:         100,
		EvictPolicy:     EvictLRU,
		CleanupInterval: 50 * time.Millisecond,
		EnableStats:     true,
	}

	cache := NewMemoryCache(config)
	defer cache.Close()

	ctx := context.Background()

	// 设置短TTL的缓存项
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("expire_key_%d", i)
		cache.Set(ctx, key, "value", 30*time.Millisecond)
	}

	// 设置长TTL的缓存项
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("keep_key_%d", i)
		cache.Set(ctx, key, "value", time.Hour)
	}

	// 等待清理
	time.Sleep(100 * time.Millisecond)

	// 验证过期的键被清理
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("expire_key_%d", i)
		_, err := cache.Get(ctx, key)
		if err != ErrKeyNotFound {
			t.Errorf("Expired key %s should have been cleaned up", key)
		}
	}

	// 验证未过期的键仍然存在
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("keep_key_%d", i)
		_, err := cache.Get(ctx, key)
		if err != nil {
			t.Errorf("Key %s should still exist: %v", key, err)
		}
	}
}

// 性能测试
func BenchmarkMemoryCache_Set(b *testing.B) {
	config := MemoryConfig{
		MaxSize:     10000,
		EvictPolicy: EvictLRU,
		EnableStats: false,
	}

	cache := NewMemoryCache(config)
	defer cache.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key_%d", i)
		cache.Set(ctx, key, "test_value", time.Hour)
	}
}

func BenchmarkMemoryCache_Get(b *testing.B) {
	config := MemoryConfig{
		MaxSize:     10000,
		EvictPolicy: EvictLRU,
		EnableStats: false,
	}

	cache := NewMemoryCache(config)
	defer cache.Close()

	ctx := context.Background()

	// 预填充数据
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

func BenchmarkMemoryCache_ConcurrentReadWrite(b *testing.B) {
	config := MemoryConfig{
		MaxSize:     10000,
		EvictPolicy: EvictLRU,
		EnableStats: false,
	}

	cache := NewMemoryCache(config)
	defer cache.Close()

	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key_%d", i%1000)
			if i%2 == 0 {
				cache.Set(ctx, key, "test_value", time.Hour)
			} else {
				cache.Get(ctx, key)
			}
			i++
		}
	})
}
