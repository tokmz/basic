package client

import (
	"context"
	"time"
)

// ListOperations 列表操作接口
type ListOperations interface {
	// 基础操作
	LPush(ctx context.Context, key string, values ...interface{}) (int64, error)
	RPush(ctx context.Context, key string, values ...interface{}) (int64, error)
	LPop(ctx context.Context, key string) (string, error)
	RPop(ctx context.Context, key string) (string, error)
	LLen(ctx context.Context, key string) (int64, error)
	LIndex(ctx context.Context, key string, index int64) (string, error)
	LSet(ctx context.Context, key string, index int64, value interface{}) (string, error)
	LRange(ctx context.Context, key string, start, stop int64) ([]string, error)
	LTrim(ctx context.Context, key string, start, stop int64) (string, error)
	LRem(ctx context.Context, key string, count int64, value interface{}) (int64, error)

	// 阻塞操作
	BLPop(ctx context.Context, timeout time.Duration, keys ...string) ([]string, error)
	BRPop(ctx context.Context, timeout time.Duration, keys ...string) ([]string, error)
	BRPopLPush(ctx context.Context, source, destination string, timeout time.Duration) (string, error)

	// 原子操作
	RPopLPush(ctx context.Context, source, destination string) (string, error)

	// 插入操作
	LInsert(ctx context.Context, key, op string, pivot, value interface{}) (int64, error)
	LInsertBefore(ctx context.Context, key string, pivot, value interface{}) (int64, error)
	LInsertAfter(ctx context.Context, key string, pivot, value interface{}) (int64, error)

	// 条件操作
	LPushX(ctx context.Context, key string, values ...interface{}) (int64, error)
	RPushX(ctx context.Context, key string, values ...interface{}) (int64, error)
}

// LPush 从列表左侧推入元素
func (c *Client) LPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "lpush", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.LPush(ctx, key, values...).Result()
	}, values)
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "lpush", key, start, err, values)
	return result, err
}

// RPush 从列表右侧推入元素
func (c *Client) RPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "rpush", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.RPush(ctx, key, values...).Result()
	}, values)
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "rpush", key, start, err, values)
	return result, err
}

// LPop 从列表左侧弹出元素
func (c *Client) LPop(ctx context.Context, key string) (string, error) {
	start := time.Now()
	var result string
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "lpop", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.LPop(ctx, key).Result()
	})
	if err == nil {
		result = resultInterface.(string)
	}

	c.recordOperation(ctx, "lpop", key, start, err)
	return result, err
}

// RPop 从列表右侧弹出元素
func (c *Client) RPop(ctx context.Context, key string) (string, error) {
	start := time.Now()
	var result string
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "rpop", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.RPop(ctx, key).Result()
	})
	if err == nil {
		result = resultInterface.(string)
	}

	c.recordOperation(ctx, "rpop", key, start, err)
	return result, err
}

// LLen 获取列表长度
func (c *Client) LLen(ctx context.Context, key string) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "llen", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.LLen(ctx, key).Result()
	})
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "llen", key, start, err)
	return result, err
}

// LIndex 获取列表指定位置的元素
func (c *Client) LIndex(ctx context.Context, key string, index int64) (string, error) {
	start := time.Now()
	var result string
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "lindex", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.LIndex(ctx, key, index).Result()
	}, index)
	if err == nil {
		result = resultInterface.(string)
	}

	c.recordOperation(ctx, "lindex", key, start, err, index)
	return result, err
}

// LSet 设置列表指定位置的元素值
func (c *Client) LSet(ctx context.Context, key string, index int64, value interface{}) (string, error) {
	start := time.Now()
	var result string
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "lset", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.LSet(ctx, key, index, value).Result()
	}, index, value)
	if err == nil {
		result = resultInterface.(string)
	}

	c.recordOperation(ctx, "lset", key, start, err, index, value)
	return result, err
}

// LRange 获取列表指定范围的元素
func (c *Client) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	startTime := time.Now()
	var result []string
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "lrange", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.LRange(ctx, key, start, stop).Result()
	}, start, stop)
	if err == nil {
		result = resultInterface.([]string)
	}

	c.recordOperation(ctx, "lrange", key, startTime, err, start, stop)
	return result, err
}

// LTrim 修剪列表，只保留指定范围的元素
func (c *Client) LTrim(ctx context.Context, key string, start, stop int64) (string, error) {
	startTime := time.Now()
	var result string
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "ltrim", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.LTrim(ctx, key, start, stop).Result()
	}, start, stop)
	if err == nil {
		result = resultInterface.(string)
	}

	c.recordOperation(ctx, "ltrim", key, startTime, err, start, stop)
	return result, err
}

// LRem 从列表中删除元素
func (c *Client) LRem(ctx context.Context, key string, count int64, value interface{}) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "lrem", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.LRem(ctx, key, count, value).Result()
	}, count, value)
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "lrem", key, start, err, count, value)
	return result, err
}

// BLPop 阻塞式从列表左侧弹出元素
func (c *Client) BLPop(ctx context.Context, timeout time.Duration, keys ...string) ([]string, error) {
	start := time.Now()
	var result []string
	var err error

	resultInterface, err := c.tracer.TraceBatchOperation(ctx, "blpop", keys, func(ctx context.Context) (interface{}, error) {
		return c.rdb.BLPop(ctx, timeout, keys...).Result()
	}, timeout)
	if err == nil {
		result = resultInterface.([]string)
	}

	c.recordOperation(ctx, "blpop", "", start, err, timeout, keys)
	return result, err
}

// BRPop 阻塞式从列表右侧弹出元素
func (c *Client) BRPop(ctx context.Context, timeout time.Duration, keys ...string) ([]string, error) {
	start := time.Now()
	var result []string
	var err error

	resultInterface, err := c.tracer.TraceBatchOperation(ctx, "brpop", keys, func(ctx context.Context) (interface{}, error) {
		return c.rdb.BRPop(ctx, timeout, keys...).Result()
	}, timeout)
	if err == nil {
		result = resultInterface.([]string)
	}

	c.recordOperation(ctx, "brpop", "", start, err, timeout, keys)
	return result, err
}

// BRPopLPush 阻塞式从源列表右侧弹出元素并推入目标列表左侧
func (c *Client) BRPopLPush(ctx context.Context, source, destination string, timeout time.Duration) (string, error) {
	start := time.Now()
	var result string
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "brpoplpush", source, func(ctx context.Context) (interface{}, error) {
		return c.rdb.BRPopLPush(ctx, source, destination, timeout).Result()
	}, destination, timeout)
	if err == nil {
		result = resultInterface.(string)
	}

	c.recordOperation(ctx, "brpoplpush", source, start, err, destination, timeout)
	return result, err
}

// RPopLPush 从源列表右侧弹出元素并推入目标列表左侧
func (c *Client) RPopLPush(ctx context.Context, source, destination string) (string, error) {
	start := time.Now()
	var result string
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "rpoplpush", source, func(ctx context.Context) (interface{}, error) {
		return c.rdb.RPopLPush(ctx, source, destination).Result()
	}, destination)
	if err == nil {
		result = resultInterface.(string)
	}

	c.recordOperation(ctx, "rpoplpush", source, start, err, destination)
	return result, err
}

// LInsert 在列表中插入元素
func (c *Client) LInsert(ctx context.Context, key, op string, pivot, value interface{}) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "linsert", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.LInsert(ctx, key, op, pivot, value).Result()
	}, op, pivot, value)
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "linsert", key, start, err, op, pivot, value)
	return result, err
}

// LInsertBefore 在指定元素前插入新元素
func (c *Client) LInsertBefore(ctx context.Context, key string, pivot, value interface{}) (int64, error) {
	return c.LInsert(ctx, key, "BEFORE", pivot, value)
}

// LInsertAfter 在指定元素后插入新元素
func (c *Client) LInsertAfter(ctx context.Context, key string, pivot, value interface{}) (int64, error) {
	return c.LInsert(ctx, key, "AFTER", pivot, value)
}

// LPushX 仅当列表存在时从左侧推入元素
func (c *Client) LPushX(ctx context.Context, key string, values ...interface{}) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "lpushx", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.LPushX(ctx, key, values...).Result()
	}, values)
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "lpushx", key, start, err, values)
	return result, err
}

// RPushX 仅当列表存在时从右侧推入元素
func (c *Client) RPushX(ctx context.Context, key string, values ...interface{}) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "rpushx", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.RPushX(ctx, key, values...).Result()
	}, values)
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "rpushx", key, start, err, values)
	return result, err
}

// ListQueue 列表队列实现
type ListQueue struct {
	client *Client
	key    string
}

// NewListQueue 创建新的列表队列
func (c *Client) NewListQueue(key string) *ListQueue {
	return &ListQueue{
		client: c,
		key:    key,
	}
}

// Enqueue 入队（从右侧推入）
func (lq *ListQueue) Enqueue(ctx context.Context, values ...interface{}) (int64, error) {
	return lq.client.RPush(ctx, lq.key, values...)
}

// Dequeue 出队（从左侧弹出）
func (lq *ListQueue) Dequeue(ctx context.Context) (string, error) {
	return lq.client.LPop(ctx, lq.key)
}

// DequeueBlocking 阻塞式出队
func (lq *ListQueue) DequeueBlocking(ctx context.Context, timeout time.Duration) ([]string, error) {
	return lq.client.BLPop(ctx, timeout, lq.key)
}

// Size 获取队列大小
func (lq *ListQueue) Size(ctx context.Context) (int64, error) {
	return lq.client.LLen(ctx, lq.key)
}

// Peek 查看队列头部元素（不弹出）
func (lq *ListQueue) Peek(ctx context.Context) (string, error) {
	return lq.client.LIndex(ctx, lq.key, 0)
}

// Clear 清空队列
func (lq *ListQueue) Clear(ctx context.Context) error {
	_, err := lq.client.rdb.Del(ctx, lq.key).Result()
	return err
}

// ListStack 列表栈实现
type ListStack struct {
	client *Client
	key    string
}

// NewListStack 创建新的列表栈
func (c *Client) NewListStack(key string) *ListStack {
	return &ListStack{
		client: c,
		key:    key,
	}
}

// Push 入栈（从左侧推入）
func (ls *ListStack) Push(ctx context.Context, values ...interface{}) (int64, error) {
	return ls.client.LPush(ctx, ls.key, values...)
}

// Pop 出栈（从左侧弹出）
func (ls *ListStack) Pop(ctx context.Context) (string, error) {
	return ls.client.LPop(ctx, ls.key)
}

// PopBlocking 阻塞式出栈
func (ls *ListStack) PopBlocking(ctx context.Context, timeout time.Duration) ([]string, error) {
	return ls.client.BLPop(ctx, timeout, ls.key)
}

// Size 获取栈大小
func (ls *ListStack) Size(ctx context.Context) (int64, error) {
	return ls.client.LLen(ctx, ls.key)
}

// Peek 查看栈顶元素（不弹出）
func (ls *ListStack) Peek(ctx context.Context) (string, error) {
	return ls.client.LIndex(ctx, ls.key, 0)
}

// Clear 清空栈
func (ls *ListStack) Clear(ctx context.Context) error {
	_, err := ls.client.rdb.Del(ctx, ls.key).Result()
	return err
}

// ListBatch 列表批量操作
type ListBatch struct {
	client *Client
	key    string
	values []interface{}
	left   bool // true表示从左侧操作，false表示从右侧操作
}

// NewListBatch 创建新的列表批量操作
func (c *Client) NewListBatch(key string, left bool) *ListBatch {
	return &ListBatch{
		client: c,
		key:    key,
		values: make([]interface{}, 0),
		left:   left,
	}
}

// Add 添加值到批量操作
func (lb *ListBatch) Add(values ...interface{}) *ListBatch {
	lb.values = append(lb.values, values...)
	return lb
}

// Execute 执行批量操作
func (lb *ListBatch) Execute(ctx context.Context) (int64, error) {
	if len(lb.values) == 0 {
		return 0, nil
	}

	if lb.left {
		return lb.client.LPush(ctx, lb.key, lb.values...)
	}
	return lb.client.RPush(ctx, lb.key, lb.values...)
}

// Clear 清空批量操作
func (lb *ListBatch) Clear() {
	lb.values = make([]interface{}, 0)
}

// Size 获取批量操作值数量
func (lb *ListBatch) Size() int {
	return len(lb.values)
}
