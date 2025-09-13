package cache

import (
	"context"
	"strings"
	"sync"
	"time"
)

// MemoryCache 内存缓存实现
type MemoryCache struct {
	config MemoryConfig
	data   sync.Map      // 缓存数据存储
	lru    *LRUCache     // LRU淘汰策略
	stats  *CacheStats   // 统计信息
	mu     sync.RWMutex  // 读写锁
	closed bool          // 是否已关闭
	ticker *time.Ticker  // 定时清理器
	stopCh chan struct{} // 停止信号
}

// NewMemoryCache 创建内存缓存
func NewMemoryCache(config MemoryConfig) *MemoryCache {
	mc := &MemoryCache{
		config: config,
		stats:  &CacheStats{},
		stopCh: make(chan struct{}),
	}

	// 初始化LRU缓存
	if config.EvictPolicy == EvictLRU {
		mc.lru = NewLRUCache(config.MaxSize)
	}

	// 启动定时清理
	if config.CleanupInterval > 0 {
		mc.ticker = time.NewTicker(config.CleanupInterval)
		go mc.cleanup()
	}

	return mc
}

// Get 获取缓存值
func (mc *MemoryCache) Get(ctx context.Context, key string) (interface{}, error) {
	if mc.closed {
		return nil, ErrCacheClosed
	}

	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// 更新统计
	mc.stats.TotalRequests++

	value, exists := mc.data.Load(key)
	if !exists {
		mc.stats.Misses++
		return nil, ErrKeyNotFound
	}

	item, ok := value.(*CacheItem)
	if !ok {
		mc.stats.Errors++
		return nil, ErrInvalidValue
	}

	// 检查是否过期
	if item.IsExpired() {
		mc.data.Delete(key)
		if mc.lru != nil {
			mc.lru.Remove(key)
		}
		mc.stats.Misses++
		return nil, ErrKeyNotFound
	}

	// 更新访问信息
	item.Touch()
	if mc.lru != nil {
		mc.lru.Get(key) // 更新LRU顺序
	}

	mc.stats.Hits++
	mc.updateHitRate()

	return item.Value, nil
}

// Set 设置缓存值
func (mc *MemoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if mc.closed {
		return ErrCacheClosed
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	// 检查内存限制
	if err := mc.checkMemoryLimit(); err != nil {
		return err
	}

	// 创建缓存项
	item := &CacheItem{
		Key:         key,
		Value:       value,
		TTL:         ttl,
		CreatedAt:   time.Now(),
		AccessAt:    time.Now(),
		AccessCount: 1,
	}

	// 如果需要淘汰
	if mc.needEvict() {
		if err := mc.evict(); err != nil {
			return err
		}
	}

	// 存储到缓存
	mc.data.Store(key, item)

	// 更新LRU
	if mc.lru != nil {
		mc.lru.Set(key, item)
	}

	mc.stats.Sets++
	mc.updateSize()

	return nil
}

// Delete 删除缓存键
func (mc *MemoryCache) Delete(ctx context.Context, key string) error {
	if mc.closed {
		return ErrCacheClosed
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	_, exists := mc.data.Load(key)
	if !exists {
		return ErrKeyNotFound
	}

	mc.data.Delete(key)
	if mc.lru != nil {
		mc.lru.Remove(key)
	}

	mc.stats.Deletes++
	mc.updateSize()

	return nil
}

// Exists 检查键是否存在
func (mc *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	if mc.closed {
		return false, ErrCacheClosed
	}

	mc.mu.RLock()
	defer mc.mu.RUnlock()

	value, exists := mc.data.Load(key)
	if !exists {
		return false, nil
	}

	item, ok := value.(*CacheItem)
	if !ok {
		return false, nil
	}

	// 检查是否过期
	if item.IsExpired() {
		mc.data.Delete(key)
		if mc.lru != nil {
			mc.lru.Remove(key)
		}
		return false, nil
	}

	return true, nil
}

// Clear 清空所有缓存
func (mc *MemoryCache) Clear(ctx context.Context) error {
	if mc.closed {
		return ErrCacheClosed
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.data.Range(func(key, value interface{}) bool {
		mc.data.Delete(key)
		return true
	})

	if mc.lru != nil {
		mc.lru.Clear()
	}

	mc.updateSize()
	return nil
}

// Keys 获取匹配模式的所有键
func (mc *MemoryCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	if mc.closed {
		return nil, ErrCacheClosed
	}

	mc.mu.RLock()
	defer mc.mu.RUnlock()

	var keys []string
	mc.data.Range(func(key, value interface{}) bool {
		keyStr := key.(string)
		item := value.(*CacheItem)

		// 检查是否过期
		if !item.IsExpired() && mc.matchPattern(keyStr, pattern) {
			keys = append(keys, keyStr)
		}
		return true
	})

	return keys, nil
}

// Stats 获取缓存统计信息
func (mc *MemoryCache) Stats() CacheStats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	stats := *mc.stats
	mc.updateHitRate()
	stats.HitRate = mc.stats.HitRate

	return stats
}

// Close 关闭缓存
func (mc *MemoryCache) Close() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.closed {
		return nil
	}

	mc.closed = true

	if mc.ticker != nil {
		mc.ticker.Stop()
	}

	close(mc.stopCh)
	return nil
}

// 私有方法

// cleanup 定时清理过期缓存
func (mc *MemoryCache) cleanup() {
	for {
		select {
		case <-mc.ticker.C:
			mc.cleanupExpired()
		case <-mc.stopCh:
			return
		}
	}
}

// cleanupExpired 清理过期缓存
func (mc *MemoryCache) cleanupExpired() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	var expiredKeys []interface{}

	mc.data.Range(func(key, value interface{}) bool {
		item := value.(*CacheItem)
		if item.IsExpired() {
			expiredKeys = append(expiredKeys, key)
		}
		return true
	})

	// 删除过期键
	for _, key := range expiredKeys {
		mc.data.Delete(key)
		if mc.lru != nil {
			mc.lru.Remove(key.(string))
		}
	}

	if len(expiredKeys) > 0 {
		mc.updateSize()
	}
}

// checkMemoryLimit 检查内存限制
func (mc *MemoryCache) checkMemoryLimit() error {
	if mc.config.MaxMemory <= 0 {
		return nil
	}

	// 简单的内存使用估算
	currentSize := mc.estimateMemoryUsage()
	if currentSize > mc.config.MaxMemory {
		return ErrMemoryLimitExceeded
	}

	return nil
}

// estimateMemoryUsage 估算内存使用量
func (mc *MemoryCache) estimateMemoryUsage() int64 {
	var totalSize int64
	mc.data.Range(func(key, value interface{}) bool {
		keySize := int64(len(key.(string)))
		valueSize := mc.estimateValueSize(value)
		totalSize += keySize + valueSize
		return true
	})
	return totalSize
}

// estimateValueSize 估算值的大小
func (mc *MemoryCache) estimateValueSize(value interface{}) int64 {
	switch v := value.(type) {
	case string:
		return int64(len(v))
	case []byte:
		return int64(len(v))
	case *CacheItem:
		return mc.estimateValueSize(v.Value) + 64 // 加上CacheItem结构的大小
	default:
		return 64 // 默认估算
	}
}

// needEvict 检查是否需要淘汰
func (mc *MemoryCache) needEvict() bool {
	if mc.config.MaxSize <= 0 {
		return false
	}

	currentSize := int64(0)
	mc.data.Range(func(key, value interface{}) bool {
		currentSize++
		return true
	})

	return currentSize >= int64(mc.config.MaxSize)
}

// evict 执行淘汰策略
func (mc *MemoryCache) evict() error {
	switch mc.config.EvictPolicy {
	case EvictLRU:
		return mc.evictLRU()
	case EvictRandom:
		return mc.evictRandom()
	case EvictTTL:
		return mc.evictTTL()
	default:
		return mc.evictLRU()
	}
}

// evictLRU LRU淘汰
func (mc *MemoryCache) evictLRU() error {
	if mc.lru == nil {
		return ErrEvictionFailed
	}

	key := mc.lru.RemoveOldest()
	if key != "" {
		mc.data.Delete(key)
	}

	return nil
}

// evictRandom 随机淘汰
func (mc *MemoryCache) evictRandom() error {
	var keyToEvict interface{}
	count := 0

	mc.data.Range(func(key, value interface{}) bool {
		count++
		if count == 1 { // 简单的随机选择第一个
			keyToEvict = key
			return false
		}
		return true
	})

	if keyToEvict != nil {
		mc.data.Delete(keyToEvict)
		if mc.lru != nil {
			mc.lru.Remove(keyToEvict.(string))
		}
	}

	return nil
}

// evictTTL 基于TTL淘汰最快过期的
func (mc *MemoryCache) evictTTL() error {
	var oldestKey interface{}
	var oldestTime time.Time

	mc.data.Range(func(key, value interface{}) bool {
		item := value.(*CacheItem)
		if oldestKey == nil || item.CreatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.CreatedAt
		}
		return true
	})

	if oldestKey != nil {
		mc.data.Delete(oldestKey)
		if mc.lru != nil {
			mc.lru.Remove(oldestKey.(string))
		}
	}

	return nil
}

// updateHitRate 更新命中率
func (mc *MemoryCache) updateHitRate() {
	if mc.stats.TotalRequests > 0 {
		mc.stats.HitRate = float64(mc.stats.Hits) / float64(mc.stats.TotalRequests)
	}
}

// updateSize 更新缓存大小
func (mc *MemoryCache) updateSize() {
	size := int64(0)
	mc.data.Range(func(key, value interface{}) bool {
		size++
		return true
	})
	mc.stats.Size = size
}

// matchPattern 模式匹配
func (mc *MemoryCache) matchPattern(key, pattern string) bool {
	if pattern == "*" {
		return true
	}

	// 简单的通配符匹配
	if strings.Contains(pattern, "*") {
		prefix := strings.Split(pattern, "*")[0]
		return strings.HasPrefix(key, prefix)
	}

	return key == pattern
}
