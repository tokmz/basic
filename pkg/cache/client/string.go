package client

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// StringOperations 字符串操作接口
type StringOperations interface {
	// 基础操作
	Set(ctx context.Context, key string, value interface{}, expiration *time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	GetSet(ctx context.Context, key string, value interface{}) (string, error)
	Del(ctx context.Context, keys ...string) (int64, error)
	Exists(ctx context.Context, keys ...string) (int64, error)
	Expire(ctx context.Context, key string, expiration time.Duration) (bool, error)
	TTL(ctx context.Context, key string) (time.Duration, error)

	// 批量操作
	MSet(ctx context.Context, pairs map[string]interface{}, expiration *time.Duration) error
	MGet(ctx context.Context, keys ...string) ([]interface{}, error)
	MSetNX(ctx context.Context, pairs map[string]interface{}) (bool, error)

	// 原子操作
	Incr(ctx context.Context, key string) (int64, error)
	IncrBy(ctx context.Context, key string, value int64) (int64, error)
	IncrByFloat(ctx context.Context, key string, value float64) (float64, error)
	Decr(ctx context.Context, key string) (int64, error)
	DecrBy(ctx context.Context, key string, value int64) (int64, error)

	// 条件操作
	SetNX(ctx context.Context, key string, value interface{}, expiration *time.Duration) (bool, error)
	SetEX(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	SetXX(ctx context.Context, key string, value interface{}, expiration *time.Duration) (bool, error)

	// 字符串操作
	Append(ctx context.Context, key, value string) (int64, error)
	StrLen(ctx context.Context, key string) (int64, error)
	GetRange(ctx context.Context, key string, start, end int64) (string, error)
	SetRange(ctx context.Context, key string, offset int64, value string) (int64, error)
}

// Set 设置键值对
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration *time.Duration) error {
	start := time.Now()
	var err error

	exp := c.getExpiration(expiration)
	_, err = c.tracer.TraceOperation(ctx, "set", key, func(ctx context.Context) (interface{}, error) {
		return nil, c.rdb.Set(ctx, key, value, exp).Err()
	}, value, exp)

	c.recordOperation(ctx, "set", key, start, err, value, exp)
	return err
}

// Get 获取键值
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	start := time.Now()
	var result string
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "get", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.Get(ctx, key).Result()
	})
	if err == nil {
		result = resultInterface.(string)
	}

	c.recordOperation(ctx, "get", key, start, err)
	return result, err
}

// GetSet 设置新值并返回旧值
func (c *Client) GetSet(ctx context.Context, key string, value interface{}) (string, error) {
	start := time.Now()
	var result string
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "getset", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.GetSet(ctx, key, value).Result()
	}, value)
	if err == nil {
		result = resultInterface.(string)
	}

	c.recordOperation(ctx, "getset", key, start, err, value)
	return result, err
}

// Del 删除键
func (c *Client) Del(ctx context.Context, keys ...string) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceBatchOperation(ctx, "del", keys, func(ctx context.Context) (interface{}, error) {
		return c.rdb.Del(ctx, keys...).Result()
	})
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "del", "", start, err, keys)
	return result, err
}

// Exists 检查键是否存在
func (c *Client) Exists(ctx context.Context, keys ...string) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceBatchOperation(ctx, "exists", keys, func(ctx context.Context) (interface{}, error) {
		return c.rdb.Exists(ctx, keys...).Result()
	})
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "exists", "", start, err, keys)
	return result, err
}

// Expire 设置键的过期时间
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	start := time.Now()
	var result bool
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "expire", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.Expire(ctx, key, expiration).Result()
	}, expiration)
	if err == nil {
		result = resultInterface.(bool)
	}

	c.recordOperation(ctx, "expire", key, start, err, expiration)
	return result, err
}

// TTL 获取键的剩余生存时间
func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	start := time.Now()
	var result time.Duration
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "ttl", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.TTL(ctx, key).Result()
	})
	if err == nil {
		result = resultInterface.(time.Duration)
	}

	c.recordOperation(ctx, "ttl", key, start, err)
	return result, err
}

// MSet 批量设置键值对
func (c *Client) MSet(ctx context.Context, pairs map[string]interface{}, expiration *time.Duration) error {
	start := time.Now()
	var err error

	keys := make([]string, 0, len(pairs))
	for k := range pairs {
		keys = append(keys, k)
	}

	_, err = c.tracer.TraceBatchOperation(ctx, "mset", keys, func(ctx context.Context) (interface{}, error) {
		if expiration != nil {
			// 使用Pipeline批量设置带过期时间的键值对
			pipe := c.rdb.Pipeline()
			exp := c.getExpiration(expiration)
			for k, v := range pairs {
				pipe.Set(ctx, k, v, exp)
			}
			_, err := pipe.Exec(ctx)
			return nil, err
		} else {
			// 使用MSet批量设置
			args := make([]interface{}, 0, len(pairs)*2)
			for k, v := range pairs {
				args = append(args, k, v)
			}
			return nil, c.rdb.MSet(ctx, args...).Err()
		}
	}, pairs)

	c.recordOperation(ctx, "mset", "", start, err, pairs)
	return err
}

// MGet 批量获取键值
func (c *Client) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	start := time.Now()
	var result []interface{}
	var err error

	resultInterface, err := c.tracer.TraceBatchOperation(ctx, "mget", keys, func(ctx context.Context) (interface{}, error) {
		return c.rdb.MGet(ctx, keys...).Result()
	})
	if err == nil {
		result = resultInterface.([]interface{})
	}

	c.recordOperation(ctx, "mget", "", start, err, keys)
	return result, err
}

// MSetNX 批量设置键值对（仅当所有键都不存在时）
func (c *Client) MSetNX(ctx context.Context, pairs map[string]interface{}) (bool, error) {
	start := time.Now()
	var result bool
	var err error

	keys := make([]string, 0, len(pairs))
	for k := range pairs {
		keys = append(keys, k)
	}

	resultInterface, err := c.tracer.TraceBatchOperation(ctx, "msetnx", keys, func(ctx context.Context) (interface{}, error) {
		args := make([]interface{}, 0, len(pairs)*2)
		for k, v := range pairs {
			args = append(args, k, v)
		}
		return c.rdb.MSetNX(ctx, args...).Result()
	}, pairs)
	if err == nil {
		result = resultInterface.(bool)
	}

	c.recordOperation(ctx, "msetnx", "", start, err, pairs)
	return result, err
}

// Incr 原子递增
func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "incr", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.Incr(ctx, key).Result()
	})
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "incr", key, start, err)
	return result, err
}

// IncrBy 原子递增指定值
func (c *Client) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "incrby", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.IncrBy(ctx, key, value).Result()
	}, value)
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "incrby", key, start, err, value)
	return result, err
}

// IncrByFloat 原子递增浮点数
func (c *Client) IncrByFloat(ctx context.Context, key string, value float64) (float64, error) {
	start := time.Now()
	var result float64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "incrbyfloat", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.IncrByFloat(ctx, key, value).Result()
	}, value)
	if err == nil {
		result = resultInterface.(float64)
	}

	c.recordOperation(ctx, "incrbyfloat", key, start, err, value)
	return result, err
}

// Decr 原子递减
func (c *Client) Decr(ctx context.Context, key string) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "decr", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.Decr(ctx, key).Result()
	})
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "decr", key, start, err)
	return result, err
}

// DecrBy 原子递减指定值
func (c *Client) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "decrby", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.DecrBy(ctx, key, value).Result()
	}, value)
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "decrby", key, start, err, value)
	return result, err
}

// SetNX 仅当键不存在时设置
func (c *Client) SetNX(ctx context.Context, key string, value interface{}, expiration *time.Duration) (bool, error) {
	start := time.Now()
	var result bool
	var err error

	exp := c.getExpiration(expiration)
	resultInterface, err := c.tracer.TraceOperation(ctx, "setnx", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.SetNX(ctx, key, value, exp).Result()
	}, value, exp)
	if err == nil {
		result = resultInterface.(bool)
	}

	c.recordOperation(ctx, "setnx", key, start, err, value, exp)
	return result, err
}

// SetEX 设置键值对并指定过期时间
func (c *Client) SetEX(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	start := time.Now()
	var err error

	_, err = c.tracer.TraceOperation(ctx, "setex", key, func(ctx context.Context) (interface{}, error) {
		return nil, c.rdb.SetEx(ctx, key, value, expiration).Err()
	}, value, expiration)

	c.recordOperation(ctx, "setex", key, start, err, value, expiration)
	return err
}

// SetXX 仅当键存在时设置
func (c *Client) SetXX(ctx context.Context, key string, value interface{}, expiration *time.Duration) (bool, error) {
	start := time.Now()
	var result bool
	var err error

	exp := c.getExpiration(expiration)
	resultInterface, err := c.tracer.TraceOperation(ctx, "setxx", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.SetXX(ctx, key, value, exp).Result()
	}, value, exp)
	if err == nil {
		result = resultInterface.(bool)
	}

	c.recordOperation(ctx, "setxx", key, start, err, value, exp)
	return result, err
}

// Append 追加字符串
func (c *Client) Append(ctx context.Context, key, value string) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "append", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.Append(ctx, key, value).Result()
	}, value)
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "append", key, start, err, value)
	return result, err
}

// StrLen 获取字符串长度
func (c *Client) StrLen(ctx context.Context, key string) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "strlen", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.StrLen(ctx, key).Result()
	})
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "strlen", key, start, err)
	return result, err
}

// GetRange 获取字符串子串
func (c *Client) GetRange(ctx context.Context, key string, start, end int64) (string, error) {
	startTime := time.Now()
	var result string
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "getrange", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.GetRange(ctx, key, start, end).Result()
	}, start, end)
	if err == nil {
		result = resultInterface.(string)
	}

	c.recordOperation(ctx, "getrange", key, startTime, err, start, end)
	return result, err
}

// SetRange 设置字符串子串
func (c *Client) SetRange(ctx context.Context, key string, offset int64, value string) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "setrange", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.SetRange(ctx, key, offset, value).Result()
	}, offset, value)
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "setrange", key, start, err, offset, value)
	return result, err
}

// Counter 原子计数器实现
type Counter struct {
	client *Client
	key    string
}

// NewCounter 创建新的计数器
func (c *Client) NewCounter(key string) *Counter {
	return &Counter{
		client: c,
		key:    key,
	}
}

// Increment 递增计数器
func (counter *Counter) Increment(ctx context.Context, delta int64) (int64, error) {
	if delta == 1 {
		return counter.client.Incr(ctx, counter.key)
	}
	return counter.client.IncrBy(ctx, counter.key, delta)
}

// Decrement 递减计数器
func (counter *Counter) Decrement(ctx context.Context, delta int64) (int64, error) {
	if delta == 1 {
		return counter.client.Decr(ctx, counter.key)
	}
	return counter.client.DecrBy(ctx, counter.key, delta)
}

// Get 获取计数器值
func (counter *Counter) Get(ctx context.Context) (int64, error) {
	val, err := counter.client.Get(ctx, counter.key)
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}

// Set 设置计数器值
func (counter *Counter) Set(ctx context.Context, value int64, expiration *time.Duration) error {
	return counter.client.Set(ctx, counter.key, value, expiration)
}

// Reset 重置计数器
func (counter *Counter) Reset(ctx context.Context) error {
	_, err := counter.client.Del(ctx, counter.key)
	return err
}

// SetExpire 设置计数器过期时间
func (counter *Counter) SetExpire(ctx context.Context, expiration time.Duration) (bool, error) {
	return counter.client.Expire(ctx, counter.key, expiration)
}
