package client

import (
	"context"
	"time"
)

// HashOperations 哈希表操作接口
type HashOperations interface {
	// 基础操作
	HSet(ctx context.Context, key string, values ...interface{}) (int64, error)
	HGet(ctx context.Context, key, field string) (string, error)
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	HDel(ctx context.Context, key string, fields ...string) (int64, error)
	HExists(ctx context.Context, key, field string) (bool, error)
	HLen(ctx context.Context, key string) (int64, error)
	HKeys(ctx context.Context, key string) ([]string, error)
	HVals(ctx context.Context, key string) ([]string, error)

	// 批量操作
	HMSet(ctx context.Context, key string, values ...interface{}) (bool, error)
	HMGet(ctx context.Context, key string, fields ...string) ([]interface{}, error)

	// 原子操作
	HIncrBy(ctx context.Context, key, field string, incr int64) (int64, error)
	HIncrByFloat(ctx context.Context, key, field string, incr float64) (float64, error)

	// 条件操作
	HSetNX(ctx context.Context, key, field string, value interface{}) (bool, error)

	// 扫描操作
	HScan(ctx context.Context, key string, cursor uint64, match string, count int64) ([]string, uint64, error)
}

// HSet 设置哈希表字段值
func (c *Client) HSet(ctx context.Context, key string, values ...interface{}) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "hset", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.HSet(ctx, key, values...).Result()
	}, values)
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "hset", key, start, err, values)
	return result, err
}

// HGet 获取哈希表字段值
func (c *Client) HGet(ctx context.Context, key, field string) (string, error) {
	start := time.Now()
	var result string
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "hget", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.HGet(ctx, key, field).Result()
	}, field)
	if err == nil {
		result = resultInterface.(string)
	}

	c.recordOperation(ctx, "hget", key, start, err, field)
	return result, err
}

// HGetAll 获取哈希表所有字段和值
func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	start := time.Now()
	var result map[string]string
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "hgetall", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.HGetAll(ctx, key).Result()
	})
	if err == nil {
		result = resultInterface.(map[string]string)
	}

	c.recordOperation(ctx, "hgetall", key, start, err)
	return result, err
}

// HDel 删除哈希表字段
func (c *Client) HDel(ctx context.Context, key string, fields ...string) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "hdel", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.HDel(ctx, key, fields...).Result()
	}, fields)
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "hdel", key, start, err, fields)
	return result, err
}

// HExists 检查哈希表字段是否存在
func (c *Client) HExists(ctx context.Context, key, field string) (bool, error) {
	start := time.Now()
	var result bool
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "hexists", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.HExists(ctx, key, field).Result()
	}, field)
	if err == nil {
		result = resultInterface.(bool)
	}

	c.recordOperation(ctx, "hexists", key, start, err, field)
	return result, err
}

// HLen 获取哈希表字段数量
func (c *Client) HLen(ctx context.Context, key string) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "hlen", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.HLen(ctx, key).Result()
	})
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "hlen", key, start, err)
	return result, err
}

// HKeys 获取哈希表所有字段名
func (c *Client) HKeys(ctx context.Context, key string) ([]string, error) {
	start := time.Now()
	var result []string
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "hkeys", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.HKeys(ctx, key).Result()
	})
	if err == nil {
		result = resultInterface.([]string)
	}

	c.recordOperation(ctx, "hkeys", key, start, err)
	return result, err
}

// HVals 获取哈希表所有值
func (c *Client) HVals(ctx context.Context, key string) ([]string, error) {
	start := time.Now()
	var result []string
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "hvals", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.HVals(ctx, key).Result()
	})
	if err == nil {
		result = resultInterface.([]string)
	}

	c.recordOperation(ctx, "hvals", key, start, err)
	return result, err
}

// HMSet 批量设置哈希表字段值
func (c *Client) HMSet(ctx context.Context, key string, values ...interface{}) (bool, error) {
	start := time.Now()
	var result bool
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "hmset", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.HMSet(ctx, key, values...).Result()
	}, values)
	if err == nil {
		result = resultInterface.(bool)
	}

	c.recordOperation(ctx, "hmset", key, start, err, values)
	return result, err
}

// HMGet 批量获取哈希表字段值
func (c *Client) HMGet(ctx context.Context, key string, fields ...string) ([]interface{}, error) {
	start := time.Now()
	var result []interface{}
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "hmget", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.HMGet(ctx, key, fields...).Result()
	}, fields)
	if err == nil {
		result = resultInterface.([]interface{})
	}

	c.recordOperation(ctx, "hmget", key, start, err, fields)
	return result, err
}

// HIncrBy 哈希表字段原子递增
func (c *Client) HIncrBy(ctx context.Context, key, field string, incr int64) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "hincrby", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.HIncrBy(ctx, key, field, incr).Result()
	}, field, incr)
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "hincrby", key, start, err, field, incr)
	return result, err
}

// HIncrByFloat 哈希表字段原子递增浮点数
func (c *Client) HIncrByFloat(ctx context.Context, key, field string, incr float64) (float64, error) {
	start := time.Now()
	var result float64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "hincrbyfloat", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.HIncrByFloat(ctx, key, field, incr).Result()
	}, field, incr)
	if err == nil {
		result = resultInterface.(float64)
	}

	c.recordOperation(ctx, "hincrbyfloat", key, start, err, field, incr)
	return result, err
}

// HSetNX 仅当哈希表字段不存在时设置
func (c *Client) HSetNX(ctx context.Context, key, field string, value interface{}) (bool, error) {
	start := time.Now()
	var result bool
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "hsetnx", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.HSetNX(ctx, key, field, value).Result()
	}, field, value)
	if err == nil {
		result = resultInterface.(bool)
	}

	c.recordOperation(ctx, "hsetnx", key, start, err, field, value)
	return result, err
}

// HScan 扫描哈希表
func (c *Client) HScan(ctx context.Context, key string, cursor uint64, match string, count int64) ([]string, uint64, error) {
	start := time.Now()
	var keys []string
	var newCursor uint64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "hscan", key, func(ctx context.Context) (interface{}, error) {
		keys, cursor, err := c.rdb.HScan(ctx, key, cursor, match, count).Result()
		return []interface{}{keys, cursor}, err
	}, cursor, match, count)
	if err == nil {
		result := resultInterface.([]interface{})
		keys = result[0].([]string)
		newCursor = result[1].(uint64)
	}

	c.recordOperation(ctx, "hscan", key, start, err, cursor, match, count)
	return keys, newCursor, err
}

// HashCounter 哈希表计数器实现
type HashCounter struct {
	client *Client
	key    string
	field  string
}

// NewHashCounter 创建新的哈希表计数器
func (c *Client) NewHashCounter(key, field string) *HashCounter {
	return &HashCounter{
		client: c,
		key:    key,
		field:  field,
	}
}

// Increment 递增哈希表计数器
func (hc *HashCounter) Increment(ctx context.Context, delta int64) (int64, error) {
	return hc.client.HIncrBy(ctx, hc.key, hc.field, delta)
}

// IncrementFloat 递增哈希表浮点计数器
func (hc *HashCounter) IncrementFloat(ctx context.Context, delta float64) (float64, error) {
	return hc.client.HIncrByFloat(ctx, hc.key, hc.field, delta)
}

// Get 获取哈希表计数器值
func (hc *HashCounter) Get(ctx context.Context) (string, error) {
	return hc.client.HGet(ctx, hc.key, hc.field)
}

// Set 设置哈希表计数器值
func (hc *HashCounter) Set(ctx context.Context, value interface{}) (int64, error) {
	return hc.client.HSet(ctx, hc.key, hc.field, value)
}

// Delete 删除哈希表计数器
func (hc *HashCounter) Delete(ctx context.Context) (int64, error) {
	return hc.client.HDel(ctx, hc.key, hc.field)
}

// Exists 检查哈希表计数器是否存在
func (hc *HashCounter) Exists(ctx context.Context) (bool, error) {
	return hc.client.HExists(ctx, hc.key, hc.field)
}

// HashBatch 哈希表批量操作
type HashBatch struct {
	client *Client
	key    string
	fields map[string]interface{}
}

// NewHashBatch 创建新的哈希表批量操作
func (c *Client) NewHashBatch(key string) *HashBatch {
	return &HashBatch{
		client: c,
		key:    key,
		fields: make(map[string]interface{}),
	}
}

// Set 添加字段到批量操作
func (hb *HashBatch) Set(field string, value interface{}) *HashBatch {
	hb.fields[field] = value
	return hb
}

// Execute 执行批量操作
func (hb *HashBatch) Execute(ctx context.Context) (int64, error) {
	if len(hb.fields) == 0 {
		return 0, nil
	}

	args := make([]interface{}, 0, len(hb.fields)*2)
	for field, value := range hb.fields {
		args = append(args, field, value)
	}

	return hb.client.HSet(ctx, hb.key, args...)
}

// Clear 清空批量操作
func (hb *HashBatch) Clear() {
	hb.fields = make(map[string]interface{})
}

// Size 获取批量操作字段数量
func (hb *HashBatch) Size() int {
	return len(hb.fields)
}
