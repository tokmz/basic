package cache

import (
	"context"
	"time"
)

// Cache 定义缓存的核心接口
type Cache interface {
	// Get 获取缓存值
	Get(ctx context.Context, key string) (interface{}, error)

	// Set 设置缓存值，支持TTL
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Delete 删除缓存键
	Delete(ctx context.Context, key string) error

	// Exists 检查键是否存在
	Exists(ctx context.Context, key string) (bool, error)

	// Clear 清空所有缓存
	Clear(ctx context.Context) error

	// Keys 获取匹配模式的所有键
	Keys(ctx context.Context, pattern string) ([]string, error)

	// Stats 获取缓存统计信息
	Stats() CacheStats

	// Close 关闭缓存连接
	Close() error
}

// CacheStats 缓存统计信息
type CacheStats struct {
	Hits          int64   `json:"hits"`           // 命中次数
	Misses        int64   `json:"misses"`         // 未命中次数
	Sets          int64   `json:"sets"`           // 设置次数
	Deletes       int64   `json:"deletes"`        // 删除次数
	Errors        int64   `json:"errors"`         // 错误次数
	Size          int64   `json:"size"`           // 缓存大小
	HitRate       float64 `json:"hit_rate"`       // 命中率
	TotalRequests int64   `json:"total_requests"` // 总请求次数
}

// CacheLevel 缓存级别枚举
type CacheLevel int

const (
	LevelMemory CacheLevel = iota // 内存级缓存
	LevelRedis                    // Redis级缓存
)

// String 返回缓存级别的字符串表示
func (l CacheLevel) String() string {
	switch l {
	case LevelMemory:
		return "memory"
	case LevelRedis:
		return "redis"
	default:
		return "unknown"
	}
}

// CacheType 缓存类型
type CacheType string

const (
	TypeMemoryLRU    CacheType = "memory_lru"    // 内存LRU缓存
	TypeMemoryLFU    CacheType = "memory_lfu"    // 内存LFU缓存
	TypeRedisCluster CacheType = "redis_cluster" // Redis集群
	TypeRedisSingle  CacheType = "redis_single"  // Redis单点
	TypeMultiLevel   CacheType = "multi_level"   // 多级缓存
)

// EvictPolicy 淘汰策略
type EvictPolicy string

const (
	EvictLRU    EvictPolicy = "lru"    // 最近最少使用
	EvictLFU    EvictPolicy = "lfu"    // 最不经常使用
	EvictRandom EvictPolicy = "random" // 随机淘汰
	EvictTTL    EvictPolicy = "ttl"    // 基于TTL淘汰
)

// CacheItem 缓存项结构
type CacheItem struct {
	Key         string        `json:"key"`          // 键
	Value       interface{}   `json:"value"`        // 值
	TTL         time.Duration `json:"ttl"`          // 过期时间
	CreatedAt   time.Time     `json:"created_at"`   // 创建时间
	AccessAt    time.Time     `json:"access_at"`    // 最后访问时间
	AccessCount int64         `json:"access_count"` // 访问次数
}

// IsExpired 检查缓存项是否已过期
func (item *CacheItem) IsExpired() bool {
	if item.TTL <= 0 {
		return false // 永不过期
	}
	return time.Since(item.CreatedAt) > item.TTL
}

// Touch 更新访问时间和计数
func (item *CacheItem) Touch() {
	item.AccessAt = time.Now()
	item.AccessCount++
}
