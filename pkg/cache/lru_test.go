package cache

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
)

func TestLRUCache_SetGet(t *testing.T) {
	lru := NewLRUCache(3)

	// 测试设置和获取
	lru.Set("key1", "value1")
	lru.Set("key2", "value2")
	lru.Set("key3", "value3")

	if lru.Len() != 3 {
		t.Errorf("Expected length 3, got %d", lru.Len())
	}

	// 测试获取存在的键
	value := lru.Get("key1")
	if value == nil {
		t.Error("Expected to find key1")
		return
	}

	if value != "value1" {
		t.Errorf("Expected value1, got %v", value)
	}

	// 测试获取不存在的键
	value = lru.Get("nonexistent")
	if value != nil {
		t.Error("Expected nil for nonexistent key")
	}
}

func TestLRUCache_Eviction(t *testing.T) {
	lru := NewLRUCache(2)

	// 添加2个项目
	lru.Set("key1", "value1")
	lru.Set("key2", "value2")

	// 添加第3个项目，应该淘汰key1
	lru.Set("key3", "value3")

	if lru.Len() != 2 {
		t.Errorf("Expected length 2, got %d", lru.Len())
	}

	// key1应该被淘汰
	if lru.Get("key1") != nil {
		t.Error("key1 should have been evicted")
	}

	// key2和key3应该存在
	if lru.Get("key2") == nil {
		t.Error("key2 should exist")
	}

	if lru.Get("key3") == nil {
		t.Error("key3 should exist")
	}
}

func TestLRUCache_UpdateExisting(t *testing.T) {
	lru := NewLRUCache(2)

	lru.Set("key1", "value1")
	lru.Set("key2", "value2")

	// 更新现有键的值
	lru.Set("key1", "new_value1")

	if lru.Len() != 2 {
		t.Errorf("Expected length 2, got %d", lru.Len())
	}

	// 获取更新的值
	value := lru.Get("key1")
	if value == nil {
		t.Error("key1 should exist")
	}

	if value != "new_value1" {
		t.Errorf("Expected new_value1, got %v", value)
	}
}

func TestLRUCache_AccessOrder(t *testing.T) {
	lru := NewLRUCache(3)

	lru.Set("key1", "value1")
	lru.Set("key2", "value2")
	lru.Set("key3", "value3")

	// 访问key1，使其成为最新的
	lru.Get("key1")

	// 添加key4，应该淘汰key2（最久未使用的）
	lru.Set("key4", "value4")

	if lru.Get("key2") != nil {
		t.Error("key2 should have been evicted")
	}

	if lru.Get("key1") == nil {
		t.Error("key1 should still exist (was accessed recently)")
	}

	if lru.Get("key3") == nil {
		t.Error("key3 should still exist")
	}

	if lru.Get("key4") == nil {
		t.Error("key4 should exist")
	}
}

func TestLRUCache_Remove(t *testing.T) {
	lru := NewLRUCache(3)

	lru.Set("key1", "value1")
	lru.Set("key2", "value2")
	lru.Set("key3", "value3")

	// 删除存在的键
	removed := lru.Remove("key2")
	if !removed {
		t.Error("Remove should return true for existing key")
	}

	if lru.Len() != 2 {
		t.Errorf("Expected length 2 after remove, got %d", lru.Len())
	}

	if lru.Get("key2") != nil {
		t.Error("key2 should not exist after remove")
	}

	// 删除不存在的键
	removed = lru.Remove("nonexistent")
	if removed {
		t.Error("Remove should return false for nonexistent key")
	}
}

func TestLRUCache_RemoveOldest(t *testing.T) {
	lru := NewLRUCache(3)

	lru.Set("key1", "value1")
	lru.Set("key2", "value2")
	lru.Set("key3", "value3")

	// 访问key1使其变新
	lru.Get("key1")

	// 删除最旧的元素（应该是key2）
	oldestKey := lru.RemoveOldest()
	if oldestKey != "key2" {
		t.Errorf("Expected to remove key2, got %s", oldestKey)
	}

	if lru.Len() != 2 {
		t.Errorf("Expected length 2 after RemoveOldest, got %d", lru.Len())
	}

	// 空缓存测试
	lru.Clear()
	oldestKey = lru.RemoveOldest()
	if oldestKey != "" {
		t.Errorf("Expected empty string for empty cache, got %s", oldestKey)
	}
}

func TestLRUCache_Clear(t *testing.T) {
	lru := NewLRUCache(3)

	lru.Set("key1", "value1")
	lru.Set("key2", "value2")
	lru.Set("key3", "value3")

	if lru.Len() != 3 {
		t.Errorf("Expected length 3 before clear, got %d", lru.Len())
	}

	lru.Clear()

	if lru.Len() != 0 {
		t.Errorf("Expected length 0 after clear, got %d", lru.Len())
	}

	if lru.Get("key1") != nil {
		t.Error("key1 should not exist after clear")
	}
}

func TestLRUCache_Keys(t *testing.T) {
	lru := NewLRUCache(3)

	lru.Set("key3", "value3")
	lru.Set("key1", "value1")
	lru.Set("key2", "value2")

	keys := lru.Keys()

	// 键应该按照从最新到最旧的顺序返回
	expectedKeys := []string{"key2", "key1", "key3"}

	if !reflect.DeepEqual(keys, expectedKeys) {
		t.Errorf("Expected keys %v, got %v", expectedKeys, keys)
	}
}

func TestLRUCache_ZeroCapacity(t *testing.T) {
	lru := NewLRUCache(0)

	lru.Set("key1", "value1")

	if lru.Len() != 0 {
		t.Errorf("Expected length 0 for zero capacity cache, got %d", lru.Len())
	}

	if lru.Get("key1") != nil {
		t.Error("Should not store anything with zero capacity")
	}
}

func TestLRUCache_SingleCapacity(t *testing.T) {
	lru := NewLRUCache(1)

	lru.Set("key1", "value1")
	lru.Set("key2", "value2")

	if lru.Len() != 1 {
		t.Errorf("Expected length 1, got %d", lru.Len())
	}

	// 只有最新的键应该存在
	if lru.Get("key1") != nil {
		t.Error("key1 should have been evicted")
	}

	if lru.Get("key2") == nil {
		t.Error("key2 should exist")
	}
}

// 并发测试
func TestLRUCache_Concurrent(t *testing.T) {
	lru := NewLRUCache(100)

	const goroutines = 10
	const operations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < operations; i++ {
				key := fmt.Sprintf("key_%d_%d", id, i)
				value := fmt.Sprintf("value_%d_%d", id, i)

				lru.Set(key, value)
				lru.Get(key)

				if i%10 == 0 {
					lru.Remove(key)
				}
			}
		}(g)
	}

	wg.Wait()
}

// 性能测试
func BenchmarkLRUCache_Set(b *testing.B) {
	lru := NewLRUCache(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key_%d", i)
		lru.Set(key, "test_value")
	}
}

func BenchmarkLRUCache_Get(b *testing.B) {
	lru := NewLRUCache(1000)

	// 预填充数据
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key_%d", i)
		lru.Set(key, "test_value")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key_%d", i%1000)
		lru.Get(key)
	}
}

func BenchmarkLRUCache_Remove(b *testing.B) {
	lru := NewLRUCache(1000)

	// 预填充数据
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key_%d", i)
		lru.Set(key, "test_value")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key_%d", i%1000)
		lru.Remove(key)
		// 重新添加以保持测试的连续性
		lru.Set(key, "test_value")
	}
}
