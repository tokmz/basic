package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"basic/pkg/cache/client"
	"basic/pkg/cache/config"
	"basic/pkg/cache/monitoring"
)

// Cache Redis缓存客户端接口
type Cache interface {
	// 基础操作
	Close() error
	Ping(ctx context.Context) error
	GetClient() *redis.Client
	GetConfig() *config.RedisConfig

	// 字符串操作
	client.StringOperations

	// 哈希表操作
	client.HashOperations

	// 列表操作
	client.ListOperations

	// 集合操作
	client.SetOperations

	// 有序集合操作
	client.ZSetOperations

	// Lua脚本操作
	client.LuaOperations

	// 监控统计
	GetStats() *monitoring.Stats
	GetSlowQueries(limit int) []monitoring.SlowQuery
	ResetStats()
	ClearSlowQueries()
	SetSlowLogThreshold(threshold time.Duration)
	EnableMonitoring()
	DisableMonitoring()

	// 配置管理
	UpdatePoolConfig(poolSize, minIdleConns, maxIdleConns int)
	UpdateTimeouts(dialTimeout, readTimeout, writeTimeout time.Duration)

	// 工厂方法
	NewCounter(key string) *client.Counter
	NewHashCounter(key, field string) *client.HashCounter
	NewListQueue(key string) *client.ListQueue
	NewListStack(key string) *client.ListStack
	NewSetFilter(key string) *client.SetFilter
	NewLeaderboard(key string) *client.Leaderboard
	NewDistributedLock(key, value string, expire time.Duration) *client.DistributedLock
	NewRateLimiter(key string, window time.Duration, limit int64) *client.RateLimiter
	NewAtomicCounter(key string, expire time.Duration) *client.AtomicCounter
	NewLuaScriptManager() *client.LuaScriptManager
	NewBatchOperator() *client.BatchOperator
	NewSetOperator() *client.SetOperator

	// 批量操作工厂
	NewHashBatch(key string) *client.HashBatch
	NewListBatch(key string, left bool) *client.ListBatch
	NewSetBatch(key string, opType string) *client.SetBatch
	NewZSetBatch(key string) *client.ZSetBatch
}

// RedisCache Redis缓存实现
type RedisCache struct {
	*client.Client
}

// New 创建新的Redis缓存客户端
func New(cfg *config.RedisConfig) (Cache, error) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	client, err := client.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return &RedisCache{
		Client: client,
	}, nil
}

// NewWithOptions 使用Redis选项创建缓存客户端
func NewWithOptions(opt *redis.Options) (Cache, error) {
	cfg := &config.RedisConfig{
		Addr:              opt.Addr,
		Password:          opt.Password,
		DB:                opt.DB,
		PoolSize:          opt.PoolSize,
		MinIdleConns:      opt.MinIdleConns,
		MaxIdleConns:      opt.MaxIdleConns,
		DialTimeout:       opt.DialTimeout,
		ReadTimeout:       opt.ReadTimeout,
		WriteTimeout:      opt.WriteTimeout,
		ConnMaxIdleTime:   30 * time.Minute,
		ConnMaxLifetime:   time.Hour,
		MaxRetries:        opt.MaxRetries,
		MinRetryBackoff:   opt.MinRetryBackoff,
		MaxRetryBackoff:   opt.MaxRetryBackoff,
		DefaultExpiration: 24 * time.Hour,
		Tracing: config.TracingConfig{
			Enabled:     true,
			ServiceName: "redis-cache",
			Version:     "1.0.0",
		},
		Monitoring: config.MonitoringConfig{
			Enabled:          true,
			SlowLogThreshold: 100 * time.Millisecond,
			CollectInterval:  10 * time.Second,
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	client, err := client.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return &RedisCache{
		Client: client,
	}, nil
}

// MustNew 创建新的Redis缓存客户端，失败时panic
func MustNew(cfg *config.RedisConfig) Cache {
	cache, err := New(cfg)
	if err != nil {
		panic(err)
	}
	return cache
}

// MustNewWithOptions 使用Redis选项创建缓存客户端，失败时panic
func MustNewWithOptions(opt *redis.Options) Cache {
	cache, err := NewWithOptions(opt)
	if err != nil {
		panic(err)
	}
	return cache
}

// DefaultCache 默认缓存实例
var DefaultCache Cache

// Init 初始化默认缓存实例
func Init(cfg *config.RedisConfig) error {
	cache, err := New(cfg)
	if err != nil {
		return err
	}
	DefaultCache = cache
	return nil
}

// InitWithOptions 使用Redis选项初始化默认缓存实例
func InitWithOptions(opt *redis.Options) error {
	cache, err := NewWithOptions(opt)
	if err != nil {
		return err
	}
	DefaultCache = cache
	return nil
}

// MustInit 初始化默认缓存实例，失败时panic
func MustInit(cfg *config.RedisConfig) {
	if err := Init(cfg); err != nil {
		panic(err)
	}
}

// MustInitWithOptions 使用Redis选项初始化默认缓存实例，失败时panic
func MustInitWithOptions(opt *redis.Options) {
	if err := InitWithOptions(opt); err != nil {
		panic(err)
	}
}

// GetDefaultCache 获取默认缓存实例
func GetDefaultCache() Cache {
	return DefaultCache
}

// 便捷函数，使用默认缓存实例

// Set 设置键值对
func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return DefaultCache.Set(ctx, key, value, &expiration)
}

// Get 获取键值
func Get(ctx context.Context, key string) (string, error) {
	return DefaultCache.Get(ctx, key)
}

// Del 删除键
func Del(ctx context.Context, keys ...string) (int64, error) {
	return DefaultCache.Del(ctx, keys...)
}

// Exists 检查键是否存在
func Exists(ctx context.Context, keys ...string) (int64, error) {
	return DefaultCache.Exists(ctx, keys...)
}

// Expire 设置键过期时间
func Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	return DefaultCache.Expire(ctx, key, expiration)
}

// TTL 获取键剩余过期时间
func TTL(ctx context.Context, key string) (time.Duration, error) {
	return DefaultCache.TTL(ctx, key)
}

// HSet 设置哈希表字段值
func HSet(ctx context.Context, key string, values ...interface{}) (int64, error) {
	return DefaultCache.HSet(ctx, key, values...)
}

// HGet 获取哈希表字段值
func HGet(ctx context.Context, key, field string) (string, error) {
	return DefaultCache.HGet(ctx, key, field)
}

// HGetAll 获取哈希表所有字段和值
func HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return DefaultCache.HGetAll(ctx, key)
}

// LPush 从列表左侧推入元素
func LPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	return DefaultCache.LPush(ctx, key, values...)
}

// RPush 从列表右侧推入元素
func RPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	return DefaultCache.RPush(ctx, key, values...)
}

// LPop 从列表左侧弹出元素
func LPop(ctx context.Context, key string) (string, error) {
	return DefaultCache.LPop(ctx, key)
}

// RPop 从列表右侧弹出元素
func RPop(ctx context.Context, key string) (string, error) {
	return DefaultCache.RPop(ctx, key)
}

// SAdd 向集合添加成员
func SAdd(ctx context.Context, key string, members ...interface{}) (int64, error) {
	return DefaultCache.SAdd(ctx, key, members...)
}

// SMembers 获取集合所有成员
func SMembers(ctx context.Context, key string) ([]string, error) {
	return DefaultCache.SMembers(ctx, key)
}

// ZAdd 向有序集合添加成员
func ZAdd(ctx context.Context, key string, members ...redis.Z) (int64, error) {
	return DefaultCache.ZAdd(ctx, key, members...)
}

// ZRange 获取有序集合指定排名范围的成员
func ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return DefaultCache.ZRange(ctx, key, start, stop)
}

// ZRevRange 获取有序集合指定排名范围的成员（从大到小）
func ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return DefaultCache.ZRevRange(ctx, key, start, stop)
}

// Eval 执行Lua脚本
func Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	return DefaultCache.Eval(ctx, script, keys, args...)
}

// EvalSha 通过SHA1执行Lua脚本
func EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) (interface{}, error) {
	return DefaultCache.EvalSha(ctx, sha1, keys, args...)
}

// Ping 检查连接
func Ping(ctx context.Context) error {
	return DefaultCache.Ping(ctx)
}

// Close 关闭连接
func Close() error {
	if DefaultCache != nil {
		return DefaultCache.Close()
	}
	return nil
}

// GetStats 获取统计信息
func GetStats() *monitoring.Stats {
	if DefaultCache != nil {
		return DefaultCache.GetStats()
	}
	return nil
}

// GetSlowQueries 获取慢查询记录
func GetSlowQueries(limit int) []monitoring.SlowQuery {
	if DefaultCache != nil {
		return DefaultCache.GetSlowQueries(limit)
	}
	return nil
}

// ResetStats 重置统计信息
func ResetStats() {
	if DefaultCache != nil {
		DefaultCache.ResetStats()
	}
}

// ClearSlowQueries 清空慢查询记录
func ClearSlowQueries() {
	if DefaultCache != nil {
		DefaultCache.ClearSlowQueries()
	}
}

// SetSlowLogThreshold 设置慢查询阈值
func SetSlowLogThreshold(threshold time.Duration) {
	if DefaultCache != nil {
		DefaultCache.SetSlowLogThreshold(threshold)
	}
}

// EnableMonitoring 启用监控
func EnableMonitoring() {
	if DefaultCache != nil {
		DefaultCache.EnableMonitoring()
	}
}

// DisableMonitoring 禁用监控
func DisableMonitoring() {
	if DefaultCache != nil {
		DefaultCache.DisableMonitoring()
	}
}
