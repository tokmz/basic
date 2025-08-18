package client

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"basic/pkg/cache/config"
	"basic/pkg/cache/monitoring"
	"basic/pkg/cache/tracing"
)

// Client Redis客户端封装
type Client struct {
	rdb           *redis.Client
	config        *config.RedisConfig
	tracer        *tracing.Tracer
	monitor       *monitoring.Monitor
	scriptManager *LuaScriptManager
}

// NewClient 创建新的Redis客户端
func NewClient(cfg *config.RedisConfig) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// 创建Redis客户端
	rdb := redis.NewClient(cfg.ToRedisOptions())

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, config.ErrConnectionFailed
	}

	// 创建追踪器
	tracer := tracing.NewTracer(
		cfg.Tracing.ServiceName,
		cfg.Tracing.Version,
		cfg.Tracing.Enabled,
	)

	// 创建监控器
	monitor := monitoring.NewMonitor(
		cfg.Monitoring.MetricsNamespace,
		cfg.Monitoring.Enabled,
		cfg.Monitoring.SlowLogThreshold,
	)

	client := &Client{
		rdb:     rdb,
		config:  cfg,
		tracer:  tracer,
		monitor: monitor,
	}

	// 初始化脚本管理器
	client.scriptManager = &LuaScriptManager{
		client:  client,
		scripts: make(map[string]*LuaScript),
	}

	return client, nil
}

// Close 关闭客户端连接
func (c *Client) Close() error {
	return c.rdb.Close()
}

// Ping 测试连接
func (c *Client) Ping(ctx context.Context) error {
	start := time.Now()
	var err error

	_, err = c.tracer.TraceOperation(ctx, "ping", "", func(ctx context.Context) (interface{}, error) {
		return c.rdb.Ping(ctx).Result()
	})

	c.monitor.RecordOperation(ctx, "ping", "", time.Since(start), err)
	return err
}

// GetClient 获取原始Redis客户端（用于高级操作）
func (c *Client) GetClient() *redis.Client {
	return c.rdb
}

// GetConfig 获取配置
func (c *Client) GetConfig() *config.RedisConfig {
	return c.config
}

// GetStats 获取监控统计
func (c *Client) GetStats() *monitoring.Stats {
	return c.monitor.GetStats()
}

// GetSlowQueries 获取慢查询记录
func (c *Client) GetSlowQueries(limit int) []monitoring.SlowQuery {
	return c.monitor.GetSlowQueries(limit)
}

// UpdatePoolConfig 动态更新连接池配置
func (c *Client) UpdatePoolConfig(poolSize, minIdle, maxIdle int) {
	c.config.UpdatePoolConfig(poolSize, minIdle, maxIdle)
	// 注意：go-redis不支持动态更新连接池配置，需要重新创建客户端
	// 这里只更新配置，实际应用中可能需要重新创建连接
}

// UpdateTimeouts 动态更新超时配置
func (c *Client) UpdateTimeouts(dial, read, write time.Duration) {
	c.config.UpdateTimeouts(dial, read, write)
	// 注意：go-redis不支持动态更新超时配置，需要重新创建客户端
}

// SetSlowLogThreshold 设置慢查询阈值
func (c *Client) SetSlowLogThreshold(threshold time.Duration) {
	c.monitor.SetSlowLogThreshold(threshold)
}

// EnableMonitoring 启用监控
func (c *Client) EnableMonitoring() {
	c.monitor.Enable()
}

// DisableMonitoring 禁用监控
func (c *Client) DisableMonitoring() {
	c.monitor.Disable()
}

// ResetStats 重置统计数据
func (c *Client) ResetStats() {
	c.monitor.ResetStats()
}

// ClearSlowQueries 清空慢查询记录
func (c *Client) ClearSlowQueries() {
	c.monitor.ClearSlowQueries()
}

// getExpiration 获取过期时间，如果未指定则使用默认值
func (c *Client) getExpiration(expiration *time.Duration) time.Duration {
	if expiration != nil {
		return *expiration
	}
	return c.config.DefaultExpiration
}

// recordOperation 记录操作指标的辅助方法
func (c *Client) recordOperation(ctx context.Context, operation, key string, start time.Time, err error, args ...interface{}) {
	duration := time.Since(start)
	c.monitor.RecordOperation(ctx, operation, key, duration, err, args...)
}

// RegisterScript 注册Lua脚本
func (c *Client) RegisterScript(name, script string) error {
	return c.scriptManager.RegisterScript(name, script)
}

// ExecuteScript 执行已注册的脚本
func (c *Client) ExecuteScript(ctx context.Context, name string, keys []string, args ...interface{}) (interface{}, error) {
	return c.scriptManager.ExecuteScript(ctx, name, keys, args...)
}

// GetScript 获取已注册的脚本
func (c *Client) GetScript(name string) (*LuaScript, bool) {
	return c.scriptManager.GetScript(name)
}

// ListScripts 列出所有已注册的脚本名称
func (c *Client) ListScripts() []string {
	return c.scriptManager.ListScripts()
}
