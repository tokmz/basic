package cache

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisCache Redis缓存实现
type RedisCache struct {
	client    redis.UniversalClient
	config    RedisConfig
	stats     *atomicStats
	closed    int32
	keyPrefix string
}

// atomicStats 原子统计
type atomicStats struct {
	hits          int64
	misses        int64
	sets          int64
	deletes       int64
	errors        int64
	totalRequests int64
}

// NewRedisCache 创建Redis缓存
func NewRedisCache(config RedisConfig) (*RedisCache, error) {
	var client redis.UniversalClient

	if config.ClusterMode {
		// 集群模式
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        config.Addrs,
			Username:     config.Username,
			Password:     config.Password,
			PoolSize:     config.PoolSize,
			MinIdleConns: config.MinIdleConns,
			MaxRetries:   config.MaxRetries,
			DialTimeout:  config.DialTimeout,
			ReadTimeout:  config.ReadTimeout,
			WriteTimeout: config.WriteTimeout,
		})
	} else {
		// 单点模式
		client = redis.NewClient(&redis.Options{
			Addr:         config.Addrs[0],
			Username:     config.Username,
			Password:     config.Password,
			DB:           config.DB,
			PoolSize:     config.PoolSize,
			MinIdleConns: config.MinIdleConns,
			MaxRetries:   config.MaxRetries,
			DialTimeout:  config.DialTimeout,
			ReadTimeout:  config.ReadTimeout,
			WriteTimeout: config.WriteTimeout,
		})
	}

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, NewCacheError(ErrCodeConnectionFailed, "Failed to connect to Redis", err)
	}

	return &RedisCache{
		client:    client,
		config:    config,
		stats:     &atomicStats{},
		keyPrefix: config.KeyPrefix,
	}, nil
}

// Get 获取缓存值
func (rc *RedisCache) Get(ctx context.Context, key string) (interface{}, error) {
	if atomic.LoadInt32(&rc.closed) == 1 {
		return nil, ErrCacheClosed
	}

	atomic.AddInt64(&rc.stats.totalRequests, 1)

	fullKey := rc.buildKey(key)
	result, err := rc.client.Get(ctx, fullKey).Result()
	if err != nil {
		if err == redis.Nil {
			atomic.AddInt64(&rc.stats.misses, 1)
			return nil, ErrKeyNotFound
		}
		atomic.AddInt64(&rc.stats.errors, 1)
		return nil, NewCacheError(ErrCodeTimeout, "Redis get operation failed", err)
	}

	// 反序列化
	var item CacheItem
	if err := json.Unmarshal([]byte(result), &item); err != nil {
		atomic.AddInt64(&rc.stats.errors, 1)
		return nil, NewCacheError(ErrCodeSerialization, "Failed to deserialize cache item", err)
	}

	atomic.AddInt64(&rc.stats.hits, 1)
	return item.Value, nil
}

// Set 设置缓存值
func (rc *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if atomic.LoadInt32(&rc.closed) == 1 {
		return ErrCacheClosed
	}

	fullKey := rc.buildKey(key)

	// 创建缓存项
	item := &CacheItem{
		Key:         key,
		Value:       value,
		TTL:         ttl,
		CreatedAt:   time.Now(),
		AccessAt:    time.Now(),
		AccessCount: 1,
	}

	// 序列化
	data, err := json.Marshal(item)
	if err != nil {
		atomic.AddInt64(&rc.stats.errors, 1)
		return NewCacheError(ErrCodeSerialization, "Failed to serialize cache item", err)
	}

	// 设置到Redis
	var result *redis.StatusCmd
	if ttl > 0 {
		result = rc.client.Set(ctx, fullKey, data, ttl)
	} else {
		result = rc.client.Set(ctx, fullKey, data, 0)
	}

	if err := result.Err(); err != nil {
		atomic.AddInt64(&rc.stats.errors, 1)
		return NewCacheError(ErrCodeTimeout, "Redis set operation failed", err)
	}

	atomic.AddInt64(&rc.stats.sets, 1)
	return nil
}

// Delete 删除缓存键
func (rc *RedisCache) Delete(ctx context.Context, key string) error {
	if atomic.LoadInt32(&rc.closed) == 1 {
		return ErrCacheClosed
	}

	fullKey := rc.buildKey(key)
	result := rc.client.Del(ctx, fullKey)
	if err := result.Err(); err != nil {
		atomic.AddInt64(&rc.stats.errors, 1)
		return NewCacheError(ErrCodeTimeout, "Redis delete operation failed", err)
	}

	if result.Val() == 0 {
		return ErrKeyNotFound
	}

	atomic.AddInt64(&rc.stats.deletes, 1)
	return nil
}

// Exists 检查键是否存在
func (rc *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	if atomic.LoadInt32(&rc.closed) == 1 {
		return false, ErrCacheClosed
	}

	fullKey := rc.buildKey(key)
	result := rc.client.Exists(ctx, fullKey)
	if err := result.Err(); err != nil {
		atomic.AddInt64(&rc.stats.errors, 1)
		return false, NewCacheError(ErrCodeTimeout, "Redis exists operation failed", err)
	}

	return result.Val() > 0, nil
}

// Clear 清空所有缓存
func (rc *RedisCache) Clear(ctx context.Context) error {
	if atomic.LoadInt32(&rc.closed) == 1 {
		return ErrCacheClosed
	}

	// 获取所有匹配的键
	pattern := rc.buildKey("*")
	keys, err := rc.client.Keys(ctx, pattern).Result()
	if err != nil {
		atomic.AddInt64(&rc.stats.errors, 1)
		return NewCacheError(ErrCodeTimeout, "Redis keys operation failed", err)
	}

	if len(keys) == 0 {
		return nil
	}

	// 批量删除
	result := rc.client.Del(ctx, keys...)
	if err := result.Err(); err != nil {
		atomic.AddInt64(&rc.stats.errors, 1)
		return NewCacheError(ErrCodeTimeout, "Redis batch delete operation failed", err)
	}

	return nil
}

// Keys 获取匹配模式的所有键
func (rc *RedisCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	if atomic.LoadInt32(&rc.closed) == 1 {
		return nil, ErrCacheClosed
	}

	fullPattern := rc.buildKey(pattern)
	keys, err := rc.client.Keys(ctx, fullPattern).Result()
	if err != nil {
		atomic.AddInt64(&rc.stats.errors, 1)
		return nil, NewCacheError(ErrCodeTimeout, "Redis keys operation failed", err)
	}

	// 移除键前缀
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		if strings.HasPrefix(key, rc.keyPrefix) {
			result = append(result, key[len(rc.keyPrefix):])
		}
	}

	return result, nil
}

// Stats 获取缓存统计信息
func (rc *RedisCache) Stats() CacheStats {
	hits := atomic.LoadInt64(&rc.stats.hits)
	misses := atomic.LoadInt64(&rc.stats.misses)
	sets := atomic.LoadInt64(&rc.stats.sets)
	deletes := atomic.LoadInt64(&rc.stats.deletes)
	errors := atomic.LoadInt64(&rc.stats.errors)
	totalRequests := atomic.LoadInt64(&rc.stats.totalRequests)

	var hitRate float64
	if totalRequests > 0 {
		hitRate = float64(hits) / float64(totalRequests)
	}

	return CacheStats{
		Hits:          hits,
		Misses:        misses,
		Sets:          sets,
		Deletes:       deletes,
		Errors:        errors,
		TotalRequests: totalRequests,
		HitRate:       hitRate,
		Size:          rc.getSize(),
	}
}

// Close 关闭缓存连接
func (rc *RedisCache) Close() error {
	if !atomic.CompareAndSwapInt32(&rc.closed, 0, 1) {
		return nil
	}

	return rc.client.Close()
}

// GetClient 获取Redis客户端（用于扩展功能）
func (rc *RedisCache) GetClient() redis.UniversalClient {
	return rc.client
}

// Pipeline 创建管道操作
func (rc *RedisCache) Pipeline(ctx context.Context, fn func(pipe redis.Pipeliner) error) error {
	if atomic.LoadInt32(&rc.closed) == 1 {
		return ErrCacheClosed
	}

	pipe := rc.client.Pipeline()
	defer pipe.Close()

	if err := fn(pipe); err != nil {
		return err
	}

	_, err := pipe.Exec(ctx)
	return err
}

// MGet 批量获取
func (rc *RedisCache) MGet(ctx context.Context, keys ...string) (map[string]interface{}, error) {
	if atomic.LoadInt32(&rc.closed) == 1 {
		return nil, ErrCacheClosed
	}

	if len(keys) == 0 {
		return make(map[string]interface{}), nil
	}

	// 构建完整键名
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = rc.buildKey(key)
	}

	// 批量获取
	results, err := rc.client.MGet(ctx, fullKeys...).Result()
	if err != nil {
		atomic.AddInt64(&rc.stats.errors, 1)
		return nil, NewCacheError(ErrCodeTimeout, "Redis mget operation failed", err)
	}

	// 处理结果
	resultMap := make(map[string]interface{})
	for i, result := range results {
		if result != nil {
			var item CacheItem
			if err := json.Unmarshal([]byte(result.(string)), &item); err == nil {
				resultMap[keys[i]] = item.Value
				atomic.AddInt64(&rc.stats.hits, 1)
			} else {
				atomic.AddInt64(&rc.stats.errors, 1)
			}
		} else {
			atomic.AddInt64(&rc.stats.misses, 1)
		}
		atomic.AddInt64(&rc.stats.totalRequests, 1)
	}

	return resultMap, nil
}

// MSet 批量设置
func (rc *RedisCache) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if atomic.LoadInt32(&rc.closed) == 1 {
		return ErrCacheClosed
	}

	if len(items) == 0 {
		return nil
	}

	// 使用管道进行批量设置
	return rc.Pipeline(ctx, func(pipe redis.Pipeliner) error {
		for key, value := range items {
			fullKey := rc.buildKey(key)

			item := &CacheItem{
				Key:         key,
				Value:       value,
				TTL:         ttl,
				CreatedAt:   time.Now(),
				AccessAt:    time.Now(),
				AccessCount: 1,
			}

			data, err := json.Marshal(item)
			if err != nil {
				return err
			}

			if ttl > 0 {
				pipe.Set(ctx, fullKey, data, ttl)
			} else {
				pipe.Set(ctx, fullKey, data, 0)
			}

			atomic.AddInt64(&rc.stats.sets, 1)
		}
		return nil
	})
}

// 私有方法

// buildKey 构建完整键名
func (rc *RedisCache) buildKey(key string) string {
	return rc.keyPrefix + key
}

// getSize 获取缓存大小（估算）
func (rc *RedisCache) getSize() int64 {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pattern := rc.buildKey("*")
	keys, err := rc.client.Keys(ctx, pattern).Result()
	if err != nil {
		return 0
	}

	return int64(len(keys))
}
