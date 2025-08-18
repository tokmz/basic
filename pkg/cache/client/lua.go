package client

import (
	"context"
	"crypto/sha1"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// LuaOperations Lua脚本操作接口
type LuaOperations interface {
	// 基础操作
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error)
	EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) (interface{}, error)
	ScriptExists(ctx context.Context, hashes ...string) ([]bool, error)
	ScriptFlush(ctx context.Context) (string, error)
	ScriptKill(ctx context.Context) (string, error)
	ScriptLoad(ctx context.Context, script string) (string, error)

	// 高级操作
	RegisterScript(name, script string) error
	ExecuteScript(ctx context.Context, name string, keys []string, args ...interface{}) (interface{}, error)
	GetScript(name string) (*LuaScript, bool)
	ListScripts() []string
}

// LuaScript Lua脚本结构
type LuaScript struct {
	Name   string
	Script string
	SHA1   string
	Loaded bool
	mutex  sync.RWMutex
}

// NewLuaScript 创建新的Lua脚本
func NewLuaScript(name, script string) *LuaScript {
	return &LuaScript{
		Name:   name,
		Script: script,
		SHA1:   generateSHA1(script),
		Loaded: false,
	}
}

// generateSHA1 生成脚本的SHA1哈希
func generateSHA1(script string) string {
	h := sha1.New()
	h.Write([]byte(script))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// LuaScriptManager Lua脚本管理器
type LuaScriptManager struct {
	client  *Client
	scripts map[string]*LuaScript
	mutex   sync.RWMutex
}

// NewLuaScriptManager 创建新的Lua脚本管理器
func (c *Client) NewLuaScriptManager() *LuaScriptManager {
	return &LuaScriptManager{
		client:  c,
		scripts: make(map[string]*LuaScript),
	}
}

// Eval 执行Lua脚本
func (c *Client) Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	start := time.Now()
	var result interface{}
	var err error

	resultInterface, err := c.tracer.TraceLuaScript(ctx, script, keys, []interface{}{args}, func(ctx context.Context) (interface{}, error) {
		return c.rdb.Eval(ctx, script, keys, args...).Result()
	})
	if err == nil {
		result = resultInterface
	}

	c.recordOperation(ctx, "eval", "", start, err, script, keys, args)
	return result, err
}

// EvalSha 通过SHA1执行Lua脚本
func (c *Client) EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) (interface{}, error) {
	start := time.Now()
	var result interface{}
	var err error

	resultInterface, err := c.tracer.TraceLuaScript(ctx, sha1, keys, []interface{}{args}, func(ctx context.Context) (interface{}, error) {
		return c.rdb.EvalSha(ctx, sha1, keys, args...).Result()
	})
	if err == nil {
		result = resultInterface
	}

	c.recordOperation(ctx, "evalsha", "", start, err, sha1, keys, args)
	return result, err
}

// ScriptExists 检查脚本是否存在
func (c *Client) ScriptExists(ctx context.Context, hashes ...string) ([]bool, error) {
	start := time.Now()
	var result []bool
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "script_exists", "", func(ctx context.Context) (interface{}, error) {
		return c.rdb.ScriptExists(ctx, hashes...).Result()
	}, hashes)
	if err == nil {
		result = resultInterface.([]bool)
	}

	c.recordOperation(ctx, "script_exists", "", start, err, hashes)
	return result, err
}

// ScriptFlush 清空所有脚本缓存
func (c *Client) ScriptFlush(ctx context.Context) (string, error) {
	start := time.Now()
	var result string
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "script_flush", "", func(ctx context.Context) (interface{}, error) {
		return c.rdb.ScriptFlush(ctx).Result()
	})
	if err == nil {
		result = resultInterface.(string)
	}

	c.recordOperation(ctx, "script_flush", "", start, err)
	return result, err
}

// ScriptKill 终止正在执行的脚本
func (c *Client) ScriptKill(ctx context.Context) (string, error) {
	start := time.Now()
	var result string
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "script_kill", "", func(ctx context.Context) (interface{}, error) {
		return c.rdb.ScriptKill(ctx).Result()
	})
	if err == nil {
		result = resultInterface.(string)
	}

	c.recordOperation(ctx, "script_kill", "", start, err)
	return result, err
}

// ScriptLoad 加载脚本到Redis
func (c *Client) ScriptLoad(ctx context.Context, script string) (string, error) {
	start := time.Now()
	var result string
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "script_load", "", func(ctx context.Context) (interface{}, error) {
		return c.rdb.ScriptLoad(ctx, script).Result()
	}, script)
	if err == nil {
		result = resultInterface.(string)
	}

	c.recordOperation(ctx, "script_load", "", start, err, script)
	return result, err
}

// RegisterScript 注册Lua脚本
func (lsm *LuaScriptManager) RegisterScript(name, script string) error {
	lsm.mutex.Lock()
	defer lsm.mutex.Unlock()

	luaScript := NewLuaScript(name, script)
	lsm.scripts[name] = luaScript
	return nil
}

// ExecuteScript 执行已注册的脚本
func (lsm *LuaScriptManager) ExecuteScript(ctx context.Context, name string, keys []string, args ...interface{}) (interface{}, error) {
	lsm.mutex.RLock()
	script, exists := lsm.scripts[name]
	lsm.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("script '%s' not found", name)
	}

	// 尝试使用SHA1执行
	if script.Loaded {
		result, err := lsm.client.EvalSha(ctx, script.SHA1, keys, args...)
		if err == nil {
			return result, nil
		}
		// 如果SHA1执行失败，标记为未加载并重新加载
		script.mutex.Lock()
		script.Loaded = false
		script.mutex.Unlock()
	}

	// 加载脚本
	sha1, err := lsm.client.ScriptLoad(ctx, script.Script)
	if err != nil {
		return nil, fmt.Errorf("failed to load script '%s': %w", name, err)
	}

	script.mutex.Lock()
	script.SHA1 = sha1
	script.Loaded = true
	script.mutex.Unlock()

	// 使用SHA1执行脚本
	return lsm.client.EvalSha(ctx, sha1, keys, args...)
}

// GetScript 获取已注册的脚本
func (lsm *LuaScriptManager) GetScript(name string) (*LuaScript, bool) {
	lsm.mutex.RLock()
	defer lsm.mutex.RUnlock()

	script, exists := lsm.scripts[name]
	return script, exists
}

// ListScripts 列出所有已注册的脚本名称
func (lsm *LuaScriptManager) ListScripts() []string {
	lsm.mutex.RLock()
	defer lsm.mutex.RUnlock()

	names := make([]string, 0, len(lsm.scripts))
	for name := range lsm.scripts {
		names = append(names, name)
	}
	return names
}

// LoadScript 加载脚本到Redis
func (ls *LuaScript) LoadScript(ctx context.Context, client *Client) error {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()

	sha1, err := client.ScriptLoad(ctx, ls.Script)
	if err != nil {
		return err
	}

	ls.SHA1 = sha1
	ls.Loaded = true
	return nil
}

// Execute 执行脚本
func (ls *LuaScript) Execute(ctx context.Context, client *Client, keys []string, args ...interface{}) (interface{}, error) {
	ls.mutex.RLock()
	loaded := ls.Loaded
	sha1 := ls.SHA1
	ls.mutex.RUnlock()

	// 尝试使用SHA1执行
	if loaded {
		result, err := client.EvalSha(ctx, sha1, keys, args...)
		if err == nil {
			return result, nil
		}
		// 如果SHA1执行失败，重新加载脚本
	}

	// 重新加载脚本
	err := ls.LoadScript(ctx, client)
	if err != nil {
		return nil, err
	}

	// 使用SHA1执行脚本
	return client.EvalSha(ctx, ls.SHA1, keys, args...)
}

// 预定义的常用Lua脚本
var (
	// 原子递增并设置过期时间
	IncrWithExpireScript = `
		local key = KEYS[1]
		local increment = tonumber(ARGV[1])
		local expire = tonumber(ARGV[2])
		
		local current = redis.call('INCRBY', key, increment)
		if current == increment then
			redis.call('EXPIRE', key, expire)
		end
		return current
	`

	// 分布式锁脚本
	DistributedLockScript = `
		local key = KEYS[1]
		local value = ARGV[1]
		local expire = tonumber(ARGV[2])
		
		local result = redis.call('SET', key, value, 'PX', expire, 'NX')
		if result then
			return 1
		else
			return 0
		end
	`

	// 释放分布式锁脚本
	ReleaseLockScript = `
		local key = KEYS[1]
		local value = ARGV[1]
		
		local current = redis.call('GET', key)
		if current == value then
			return redis.call('DEL', key)
		else
			return 0
		end
	`

	// 限流脚本（滑动窗口）
	RateLimitScript = `
		local key = KEYS[1]
		local window = tonumber(ARGV[1])
		local limit = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		
		-- 清理过期的记录
		redis.call('ZREMRANGEBYSCORE', key, 0, now - window)
		
		-- 获取当前窗口内的请求数
		local current = redis.call('ZCARD', key)
		
		if current < limit then
			-- 添加当前请求
			redis.call('ZADD', key, now, now)
			redis.call('EXPIRE', key, math.ceil(window / 1000))
			return {1, limit - current - 1}
		else
			return {0, 0}
		end
	`

	// 批量设置带过期时间的键值对
	BatchSetWithExpireScript = `
		local expire = tonumber(ARGV[1])
		local len = #KEYS
		
		for i = 1, len do
			redis.call('SET', KEYS[i], ARGV[i + 1], 'EX', expire)
		end
		
		return len
	`

	// 获取多个哈希表的指定字段
	MultiHGetScript = `
		local result = {}
		local field = ARGV[1]
		
		for i = 1, #KEYS do
			local value = redis.call('HGET', KEYS[i], field)
			table.insert(result, value)
		end
		
		return result
	`
)

// DistributedLock 分布式锁实现
type DistributedLock struct {
	client *Client
	key    string
	value  string
	expire time.Duration
}

// NewDistributedLock 创建新的分布式锁
func (c *Client) NewDistributedLock(key, value string, expire time.Duration) *DistributedLock {
	return &DistributedLock{
		client: c,
		key:    key,
		value:  value,
		expire: expire,
	}
}

// TryLock 尝试获取锁
func (dl *DistributedLock) TryLock(ctx context.Context) (bool, error) {
	result, err := dl.client.Eval(ctx, DistributedLockScript, []string{dl.key}, dl.value, dl.expire.Milliseconds())
	if err != nil {
		return false, err
	}
	return result.(int64) == 1, nil
}

// Unlock 释放锁
func (dl *DistributedLock) Unlock(ctx context.Context) (bool, error) {
	result, err := dl.client.Eval(ctx, ReleaseLockScript, []string{dl.key}, dl.value)
	if err != nil {
		return false, err
	}
	return result.(int64) == 1, nil
}

// RateLimiter 限流器实现
type RateLimiter struct {
	client *Client
	key    string
	window time.Duration
	limit  int64
}

// NewRateLimiter 创建新的限流器
func (c *Client) NewRateLimiter(key string, window time.Duration, limit int64) *RateLimiter {
	return &RateLimiter{
		client: c,
		key:    key,
		window: window,
		limit:  limit,
	}
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow(ctx context.Context) (bool, int64, error) {
	now := time.Now().UnixMilli()
	result, err := rl.client.Eval(ctx, RateLimitScript, []string{rl.key}, rl.window.Milliseconds(), rl.limit, now)
	if err != nil {
		return false, 0, err
	}

	results := result.([]interface{})
	allowed := results[0].(int64) == 1
	remaining := results[1].(int64)

	return allowed, remaining, nil
}

// AtomicCounter 原子计数器（带过期时间）
type AtomicCounter struct {
	client *Client
	key    string
	expire time.Duration
}

// NewAtomicCounter 创建新的原子计数器
func (c *Client) NewAtomicCounter(key string, expire time.Duration) *AtomicCounter {
	return &AtomicCounter{
		client: c,
		key:    key,
		expire: expire,
	}
}

// IncrBy 原子递增并设置过期时间
func (ac *AtomicCounter) IncrBy(ctx context.Context, increment int64) (int64, error) {
	result, err := ac.client.Eval(ctx, IncrWithExpireScript, []string{ac.key}, increment, ac.expire.Seconds())
	if err != nil {
		return 0, err
	}
	return result.(int64), nil
}

// Incr 原子递增1
func (ac *AtomicCounter) Incr(ctx context.Context) (int64, error) {
	return ac.IncrBy(ctx, 1)
}

// Get 获取当前值
func (ac *AtomicCounter) Get(ctx context.Context) (int64, error) {
	result, err := ac.client.Get(ctx, ac.key)
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, err
	}

	var value int64
	_, err = fmt.Sscanf(result, "%d", &value)
	return value, err
}

// BatchOperator 批量操作器
type BatchOperator struct {
	client *Client
}

// NewBatchOperator 创建新的批量操作器
func (c *Client) NewBatchOperator() *BatchOperator {
	return &BatchOperator{
		client: c,
	}
}

// BatchSetWithExpire 批量设置带过期时间的键值对
func (bo *BatchOperator) BatchSetWithExpire(ctx context.Context, kvs map[string]interface{}, expire time.Duration) (int64, error) {
	if len(kvs) == 0 {
		return 0, nil
	}

	keys := make([]string, 0, len(kvs))
	args := make([]interface{}, 0, len(kvs)+1)
	args = append(args, expire.Seconds())

	for key, value := range kvs {
		keys = append(keys, key)
		args = append(args, value)
	}

	result, err := bo.client.Eval(ctx, BatchSetWithExpireScript, keys, args...)
	if err != nil {
		return 0, err
	}
	return result.(int64), nil
}

// MultiHGet 获取多个哈希表的指定字段
func (bo *BatchOperator) MultiHGet(ctx context.Context, keys []string, field string) ([]interface{}, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	result, err := bo.client.Eval(ctx, MultiHGetScript, keys, field)
	if err != nil {
		return nil, err
	}
	return result.([]interface{}), nil
}
