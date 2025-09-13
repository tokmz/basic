package cache

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// MultiLevelCache 多级缓存管理器
type MultiLevelCache struct {
	config MultiLevelConfig
	l1     Cache // L1缓存(内存)
	l2     Cache // L2缓存(Redis)
	stats  *multiLevelStats
	closed int32
	mu     sync.RWMutex
}

// multiLevelStats 多级缓存统计
type multiLevelStats struct {
	l1Stats CacheStats
	l2Stats CacheStats
	l1Hits  int64 // L1命中次数
	l2Hits  int64 // L2命中次数
	l1Sets  int64 // L1设置次数
	l2Sets  int64 // L2设置次数
	syncs   int64 // 同步次数
	errors  int64 // 错误次数
}

// NewMultiLevelCache 创建多级缓存
func NewMultiLevelCache(config Config) (*MultiLevelCache, error) {
	ml := &MultiLevelCache{
		config: config.MultiLevel,
		stats:  &multiLevelStats{},
	}

	// 初始化L1缓存(内存)
	if config.MultiLevel.EnableL1 {
		ml.l1 = NewMemoryCache(config.Memory)
	}

	// 初始化L2缓存(Redis)
	if config.MultiLevel.EnableL2 {
		redisCache, err := NewRedisCache(config.Redis)
		if err != nil {
			return nil, err
		}
		ml.l2 = redisCache
	}

	return ml, nil
}

// Get 获取缓存值(优先从L1获取，未命中则从L2获取)
func (ml *MultiLevelCache) Get(ctx context.Context, key string) (interface{}, error) {
	if atomic.LoadInt32(&ml.closed) == 1 {
		return nil, ErrCacheClosed
	}

	// 先尝试从L1缓存获取
	if ml.l1 != nil {
		if value, err := ml.l1.Get(ctx, key); err == nil {
			atomic.AddInt64(&ml.stats.l1Hits, 1)
			return value, nil
		}
	}

	// L1未命中，尝试从L2缓存获取
	if ml.l2 != nil {
		value, err := ml.l2.Get(ctx, key)
		if err == nil {
			atomic.AddInt64(&ml.stats.l2Hits, 1)

			// 同步到L1缓存
			if ml.l1 != nil && ml.config.SyncStrategy != SyncWriteAround {
				go ml.syncToL1(ctx, key, value)
			}

			return value, nil
		}
	}

	return nil, ErrKeyNotFound
}

// Set 设置缓存值(根据同步策略设置到不同层级)
func (ml *MultiLevelCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if atomic.LoadInt32(&ml.closed) == 1 {
		return ErrCacheClosed
	}

	switch ml.config.SyncStrategy {
	case SyncWriteThrough:
		return ml.setWriteThrough(ctx, key, value, ttl)
	case SyncWriteBack:
		return ml.setWriteBack(ctx, key, value, ttl)
	case SyncWriteAround:
		return ml.setWriteAround(ctx, key, value, ttl)
	default:
		return ml.setWriteThrough(ctx, key, value, ttl)
	}
}

// Delete 删除缓存键
func (ml *MultiLevelCache) Delete(ctx context.Context, key string) error {
	if atomic.LoadInt32(&ml.closed) == 1 {
		return ErrCacheClosed
	}

	var errs []error

	// 从L1删除
	if ml.l1 != nil {
		if err := ml.l1.Delete(ctx, key); err != nil && err != ErrKeyNotFound {
			errs = append(errs, err)
		}
	}

	// 从L2删除
	if ml.l2 != nil {
		if err := ml.l2.Delete(ctx, key); err != nil && err != ErrKeyNotFound {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0] // 返回第一个错误
	}

	return nil
}

// Exists 检查键是否存在
func (ml *MultiLevelCache) Exists(ctx context.Context, key string) (bool, error) {
	if atomic.LoadInt32(&ml.closed) == 1 {
		return false, ErrCacheClosed
	}

	// 先检查L1
	if ml.l1 != nil {
		if exists, err := ml.l1.Exists(ctx, key); err == nil && exists {
			return true, nil
		}
	}

	// 再检查L2
	if ml.l2 != nil {
		return ml.l2.Exists(ctx, key)
	}

	return false, nil
}

// Clear 清空所有缓存
func (ml *MultiLevelCache) Clear(ctx context.Context) error {
	if atomic.LoadInt32(&ml.closed) == 1 {
		return ErrCacheClosed
	}

	var errs []error

	// 清空L1
	if ml.l1 != nil {
		if err := ml.l1.Clear(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	// 清空L2
	if ml.l2 != nil {
		if err := ml.l2.Clear(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

// Keys 获取匹配模式的所有键
func (ml *MultiLevelCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	if atomic.LoadInt32(&ml.closed) == 1 {
		return nil, ErrCacheClosed
	}

	keyMap := make(map[string]bool)

	// 从L1获取键
	if ml.l1 != nil {
		if keys, err := ml.l1.Keys(ctx, pattern); err == nil {
			for _, key := range keys {
				keyMap[key] = true
			}
		}
	}

	// 从L2获取键
	if ml.l2 != nil {
		if keys, err := ml.l2.Keys(ctx, pattern); err == nil {
			for _, key := range keys {
				keyMap[key] = true
			}
		}
	}

	// 合并结果
	result := make([]string, 0, len(keyMap))
	for key := range keyMap {
		result = append(result, key)
	}

	return result, nil
}

// Stats 获取缓存统计信息
func (ml *MultiLevelCache) Stats() CacheStats {
	ml.mu.RLock()
	defer ml.mu.RUnlock()

	var totalStats CacheStats

	// L1统计
	if ml.l1 != nil {
		ml.stats.l1Stats = ml.l1.Stats()
		totalStats.Hits += ml.stats.l1Stats.Hits
		totalStats.Misses += ml.stats.l1Stats.Misses
		totalStats.Sets += ml.stats.l1Stats.Sets
		totalStats.Deletes += ml.stats.l1Stats.Deletes
		totalStats.Errors += ml.stats.l1Stats.Errors
		totalStats.TotalRequests += ml.stats.l1Stats.TotalRequests
		totalStats.Size += ml.stats.l1Stats.Size
	}

	// L2统计
	if ml.l2 != nil {
		ml.stats.l2Stats = ml.l2.Stats()
		totalStats.Hits += ml.stats.l2Stats.Hits
		totalStats.Misses += ml.stats.l2Stats.Misses
		totalStats.Sets += ml.stats.l2Stats.Sets
		totalStats.Deletes += ml.stats.l2Stats.Deletes
		totalStats.Errors += ml.stats.l2Stats.Errors
		totalStats.TotalRequests += ml.stats.l2Stats.TotalRequests
	}

	// 计算总体命中率
	if totalStats.TotalRequests > 0 {
		totalStats.HitRate = float64(totalStats.Hits) / float64(totalStats.TotalRequests)
	}

	return totalStats
}

// Close 关闭缓存
func (ml *MultiLevelCache) Close() error {
	if !atomic.CompareAndSwapInt32(&ml.closed, 0, 1) {
		return nil
	}

	var errs []error

	// 关闭L1
	if ml.l1 != nil {
		if err := ml.l1.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	// 关闭L2
	if ml.l2 != nil {
		if err := ml.l2.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

// GetL1Cache 获取L1缓存实例
func (ml *MultiLevelCache) GetL1Cache() Cache {
	return ml.l1
}

// GetL2Cache 获取L2缓存实例
func (ml *MultiLevelCache) GetL2Cache() Cache {
	return ml.l2
}

// GetMultiLevelStats 获取多级缓存详细统计
func (ml *MultiLevelCache) GetMultiLevelStats() *multiLevelStats {
	ml.mu.RLock()
	defer ml.mu.RUnlock()

	stats := *ml.stats

	// 更新当前统计
	if ml.l1 != nil {
		stats.l1Stats = ml.l1.Stats()
	}
	if ml.l2 != nil {
		stats.l2Stats = ml.l2.Stats()
	}

	stats.l1Hits = atomic.LoadInt64(&ml.stats.l1Hits)
	stats.l2Hits = atomic.LoadInt64(&ml.stats.l2Hits)
	stats.l1Sets = atomic.LoadInt64(&ml.stats.l1Sets)
	stats.l2Sets = atomic.LoadInt64(&ml.stats.l2Sets)
	stats.syncs = atomic.LoadInt64(&ml.stats.syncs)
	stats.errors = atomic.LoadInt64(&ml.stats.errors)

	return &stats
}

// Invalidate 使指定键在所有层级失效
func (ml *MultiLevelCache) Invalidate(ctx context.Context, key string) error {
	return ml.Delete(ctx, key)
}

// Sync 同步L1和L2缓存
func (ml *MultiLevelCache) Sync(ctx context.Context) error {
	if atomic.LoadInt32(&ml.closed) == 1 {
		return ErrCacheClosed
	}

	if ml.l1 == nil || ml.l2 == nil {
		return nil
	}

	// 获取L2中的所有键
	l2Keys, err := ml.l2.Keys(ctx, "*")
	if err != nil {
		atomic.AddInt64(&ml.stats.errors, 1)
		return err
	}

	// 同步到L1
	for _, key := range l2Keys {
		value, err := ml.l2.Get(ctx, key)
		if err != nil {
			continue
		}

		// 使用L1的TTL设置
		ttl := ml.config.L1TTL
		if ttl <= 0 {
			ttl = time.Hour // 默认1小时
		}

		ml.l1.Set(ctx, key, value, ttl)
	}

	atomic.AddInt64(&ml.stats.syncs, 1)
	return nil
}

// 私有方法

// setWriteThrough 写穿透模式
func (ml *MultiLevelCache) setWriteThrough(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	var errs []error

	// 先写L2
	if ml.l2 != nil {
		l2TTL := ml.config.L2TTL
		if ttl > 0 && (l2TTL <= 0 || ttl < l2TTL) {
			l2TTL = ttl
		}
		if err := ml.l2.Set(ctx, key, value, l2TTL); err != nil {
			errs = append(errs, err)
		} else {
			atomic.AddInt64(&ml.stats.l2Sets, 1)
		}
	}

	// 再写L1
	if ml.l1 != nil {
		l1TTL := ml.config.L1TTL
		if ttl > 0 && (l1TTL <= 0 || ttl < l1TTL) {
			l1TTL = ttl
		}
		if err := ml.l1.Set(ctx, key, value, l1TTL); err != nil {
			errs = append(errs, err)
		} else {
			atomic.AddInt64(&ml.stats.l1Sets, 1)
		}
	}

	if len(errs) > 0 {
		atomic.AddInt64(&ml.stats.errors, 1)
		return errs[0]
	}

	return nil
}

// setWriteBack 写回模式(先写L1，异步写L2)
func (ml *MultiLevelCache) setWriteBack(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// 先写L1
	if ml.l1 != nil {
		l1TTL := ml.config.L1TTL
		if ttl > 0 && (l1TTL <= 0 || ttl < l1TTL) {
			l1TTL = ttl
		}
		if err := ml.l1.Set(ctx, key, value, l1TTL); err != nil {
			atomic.AddInt64(&ml.stats.errors, 1)
			return err
		}
		atomic.AddInt64(&ml.stats.l1Sets, 1)
	}

	// 异步写L2
	if ml.l2 != nil {
		go func() {
			l2TTL := ml.config.L2TTL
			if ttl > 0 && (l2TTL <= 0 || ttl < l2TTL) {
				l2TTL = ttl
			}
			if err := ml.l2.Set(context.Background(), key, value, l2TTL); err == nil {
				atomic.AddInt64(&ml.stats.l2Sets, 1)
			} else {
				atomic.AddInt64(&ml.stats.errors, 1)
			}
		}()
	}

	return nil
}

// setWriteAround 写绕过模式(只写L2，不写L1)
func (ml *MultiLevelCache) setWriteAround(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// 只写L2
	if ml.l2 != nil {
		l2TTL := ml.config.L2TTL
		if ttl > 0 && (l2TTL <= 0 || ttl < l2TTL) {
			l2TTL = ttl
		}
		if err := ml.l2.Set(ctx, key, value, l2TTL); err != nil {
			atomic.AddInt64(&ml.stats.errors, 1)
			return err
		}
		atomic.AddInt64(&ml.stats.l2Sets, 1)
	}

	return nil
}

// syncToL1 同步数据到L1缓存
func (ml *MultiLevelCache) syncToL1(ctx context.Context, key string, value interface{}) {
	if ml.l1 == nil {
		return
	}

	l1TTL := ml.config.L1TTL
	if l1TTL <= 0 {
		l1TTL = 30 * time.Minute // 默认30分钟
	}

	if err := ml.l1.Set(ctx, key, value, l1TTL); err == nil {
		atomic.AddInt64(&ml.stats.l1Sets, 1)
		atomic.AddInt64(&ml.stats.syncs, 1)
	} else {
		atomic.AddInt64(&ml.stats.errors, 1)
	}
}
