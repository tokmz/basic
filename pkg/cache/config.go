package cache

import (
	"time"
)

// Config 缓存配置结构
type Config struct {
	// 基础配置
	Type       CacheType     `yaml:"type" json:"type"`               // 缓存类型
	DefaultTTL time.Duration `yaml:"default_ttl" json:"default_ttl"` // 默认TTL

	// 内存缓存配置
	Memory MemoryConfig `yaml:"memory" json:"memory"`

	// Redis缓存配置
	Redis RedisConfig `yaml:"redis" json:"redis"`

	// 多级缓存配置
	MultiLevel MultiLevelConfig `yaml:"multi_level" json:"multi_level"`

	// 监控配置
	Monitoring MonitoringConfig `yaml:"monitoring" json:"monitoring"`

	// 分布式配置
	Distributed DistributedConfig `yaml:"distributed" json:"distributed"`
}

// MemoryConfig 内存缓存配置
type MemoryConfig struct {
	MaxSize         int           `yaml:"max_size" json:"max_size"`                 // 最大缓存项数量
	MaxMemory       int64         `yaml:"max_memory" json:"max_memory"`             // 最大内存使用量(字节)
	EvictPolicy     EvictPolicy   `yaml:"evict_policy" json:"evict_policy"`         // 淘汰策略
	CleanupInterval time.Duration `yaml:"cleanup_interval" json:"cleanup_interval"` // 清理间隔
	EnableStats     bool          `yaml:"enable_stats" json:"enable_stats"`         // 是否启用统计
}

// RedisConfig Redis缓存配置
type RedisConfig struct {
	// 连接配置
	Addrs    []string `yaml:"addrs" json:"addrs"`       // Redis地址列表
	Username string   `yaml:"username" json:"username"` // 用户名
	Password string   `yaml:"password" json:"password"` // 密码
	DB       int      `yaml:"db" json:"db"`             // 数据库编号

	// 连接池配置
	PoolSize     int           `yaml:"pool_size" json:"pool_size"`           // 连接池大小
	MinIdleConns int           `yaml:"min_idle_conns" json:"min_idle_conns"` // 最小空闲连接
	MaxRetries   int           `yaml:"max_retries" json:"max_retries"`       // 最大重试次数
	DialTimeout  time.Duration `yaml:"dial_timeout" json:"dial_timeout"`     // 连接超时
	ReadTimeout  time.Duration `yaml:"read_timeout" json:"read_timeout"`     // 读超时
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`   // 写超时

	// 集群配置
	ClusterMode bool `yaml:"cluster_mode" json:"cluster_mode"` // 是否集群模式

	// 键前缀
	KeyPrefix string `yaml:"key_prefix" json:"key_prefix"`
}

// MultiLevelConfig 多级缓存配置
type MultiLevelConfig struct {
	EnableL1     bool          `yaml:"enable_l1" json:"enable_l1"`         // 启用L1缓存(内存)
	EnableL2     bool          `yaml:"enable_l2" json:"enable_l2"`         // 启用L2缓存(Redis)
	L1TTL        time.Duration `yaml:"l1_ttl" json:"l1_ttl"`               // L1缓存TTL
	L2TTL        time.Duration `yaml:"l2_ttl" json:"l2_ttl"`               // L2缓存TTL
	SyncStrategy SyncStrategy  `yaml:"sync_strategy" json:"sync_strategy"` // 同步策略
}

// MonitoringConfig 监控配置
type MonitoringConfig struct {
	EnableMetrics   bool          `yaml:"enable_metrics" json:"enable_metrics"`     // 启用指标收集
	MetricsInterval time.Duration `yaml:"metrics_interval" json:"metrics_interval"` // 指标收集间隔
	EnableLogging   bool          `yaml:"enable_logging" json:"enable_logging"`     // 启用日志
	LogLevel        string        `yaml:"log_level" json:"log_level"`               // 日志级别
	SlowThreshold   time.Duration `yaml:"slow_threshold" json:"slow_threshold"`     // 慢操作阈值
}

// DistributedConfig 分布式配置
type DistributedConfig struct {
	EnableSync       bool             `yaml:"enable_sync" json:"enable_sync"`             // 启用分布式同步
	SyncInterval     time.Duration    `yaml:"sync_interval" json:"sync_interval"`         // 同步间隔
	ConsistencyLevel ConsistencyLevel `yaml:"consistency_level" json:"consistency_level"` // 一致性级别
	NodeID           string           `yaml:"node_id" json:"node_id"`                     // 节点ID
	BroadcastAddr    string           `yaml:"broadcast_addr" json:"broadcast_addr"`       // 广播地址
}

// SyncStrategy 同步策略
type SyncStrategy string

const (
	SyncWriteThrough SyncStrategy = "write_through" // 写穿透
	SyncWriteBack    SyncStrategy = "write_back"    // 写回
	SyncWriteAround  SyncStrategy = "write_around"  // 写绕过
)

// ConsistencyLevel 一致性级别
type ConsistencyLevel string

const (
	ConsistencyEventual ConsistencyLevel = "eventual" // 最终一致性
	ConsistencyStrong   ConsistencyLevel = "strong"   // 强一致性
	ConsistencyWeak     ConsistencyLevel = "weak"     // 弱一致性
)

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Type:       TypeMemoryLRU,
		DefaultTTL: time.Hour,
		Memory: MemoryConfig{
			MaxSize:         1000,
			MaxMemory:       100 * 1024 * 1024, // 100MB
			EvictPolicy:     EvictLRU,
			CleanupInterval: 10 * time.Minute,
			EnableStats:     true,
		},
		Redis: RedisConfig{
			Addrs:        []string{"localhost:6379"},
			PoolSize:     10,
			MinIdleConns: 5,
			MaxRetries:   3,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
			KeyPrefix:    "cache:",
		},
		MultiLevel: MultiLevelConfig{
			EnableL1:     true,
			EnableL2:     false,
			L1TTL:        30 * time.Minute,
			L2TTL:        2 * time.Hour,
			SyncStrategy: SyncWriteThrough,
		},
		Monitoring: MonitoringConfig{
			EnableMetrics:   true,
			MetricsInterval: 30 * time.Second,
			EnableLogging:   true,
			LogLevel:        "info",
			SlowThreshold:   100 * time.Millisecond,
		},
		Distributed: DistributedConfig{
			EnableSync:       false,
			SyncInterval:     5 * time.Second,
			ConsistencyLevel: ConsistencyEventual,
		},
	}
}

// Validate 验证配置有效性
func (c *Config) Validate() error {
	if c.DefaultTTL < 0 {
		return ErrInvalidTTL
	}

	if c.Memory.MaxSize <= 0 {
		return ErrInvalidMaxSize
	}

	if c.Memory.MaxMemory <= 0 {
		return ErrInvalidMaxMemory
	}

	if len(c.Redis.Addrs) == 0 {
		return ErrEmptyRedisAddrs
	}

	return nil
}
