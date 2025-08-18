package client

import (
	"context"
	"time"
)

// SetOperations 集合操作接口
type SetOperations interface {
	// 基础操作
	SAdd(ctx context.Context, key string, members ...interface{}) (int64, error)
	SRem(ctx context.Context, key string, members ...interface{}) (int64, error)
	SMembers(ctx context.Context, key string) ([]string, error)
	SCard(ctx context.Context, key string) (int64, error)
	SIsMember(ctx context.Context, key string, member interface{}) (bool, error)
	SPop(ctx context.Context, key string) (string, error)
	SPopN(ctx context.Context, key string, count int64) ([]string, error)
	SRandMember(ctx context.Context, key string) (string, error)
	SRandMemberN(ctx context.Context, key string, count int64) ([]string, error)

	// 集合运算
	SUnion(ctx context.Context, keys ...string) ([]string, error)
	SUnionStore(ctx context.Context, destination string, keys ...string) (int64, error)
	SInter(ctx context.Context, keys ...string) ([]string, error)
	SInterStore(ctx context.Context, destination string, keys ...string) (int64, error)
	SDiff(ctx context.Context, keys ...string) ([]string, error)
	SDiffStore(ctx context.Context, destination string, keys ...string) (int64, error)

	// 移动操作
	SMove(ctx context.Context, source, destination string, member interface{}) (bool, error)

	// 扫描操作
	SScan(ctx context.Context, key string, cursor uint64, match string, count int64) ([]string, uint64, error)
}

// SAdd 向集合添加成员
func (c *Client) SAdd(ctx context.Context, key string, members ...interface{}) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "sadd", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.SAdd(ctx, key, members...).Result()
	}, members)
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "sadd", key, start, err, members)
	return result, err
}

// SRem 从集合删除成员
func (c *Client) SRem(ctx context.Context, key string, members ...interface{}) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "srem", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.SRem(ctx, key, members...).Result()
	}, members)
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "srem", key, start, err, members)
	return result, err
}

// SMembers 获取集合所有成员
func (c *Client) SMembers(ctx context.Context, key string) ([]string, error) {
	start := time.Now()
	var result []string
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "smembers", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.SMembers(ctx, key).Result()
	})
	if err == nil {
		result = resultInterface.([]string)
	}

	c.recordOperation(ctx, "smembers", key, start, err)
	return result, err
}

// SCard 获取集合成员数量
func (c *Client) SCard(ctx context.Context, key string) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "scard", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.SCard(ctx, key).Result()
	})
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "scard", key, start, err)
	return result, err
}

// SIsMember 检查成员是否在集合中
func (c *Client) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	start := time.Now()
	var result bool
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "sismember", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.SIsMember(ctx, key, member).Result()
	}, member)
	if err == nil {
		result = resultInterface.(bool)
	}

	c.recordOperation(ctx, "sismember", key, start, err, member)
	return result, err
}

// SPop 随机弹出集合中的一个成员
func (c *Client) SPop(ctx context.Context, key string) (string, error) {
	start := time.Now()
	var result string
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "spop", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.SPop(ctx, key).Result()
	})
	if err == nil {
		result = resultInterface.(string)
	}

	c.recordOperation(ctx, "spop", key, start, err)
	return result, err
}

// SPopN 随机弹出集合中的多个成员
func (c *Client) SPopN(ctx context.Context, key string, count int64) ([]string, error) {
	start := time.Now()
	var result []string
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "spopn", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.SPopN(ctx, key, count).Result()
	}, count)
	if err == nil {
		result = resultInterface.([]string)
	}

	c.recordOperation(ctx, "spopn", key, start, err, count)
	return result, err
}

// SRandMember 随机获取集合中的一个成员（不删除）
func (c *Client) SRandMember(ctx context.Context, key string) (string, error) {
	start := time.Now()
	var result string
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "srandmember", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.SRandMember(ctx, key).Result()
	})
	if err == nil {
		result = resultInterface.(string)
	}

	c.recordOperation(ctx, "srandmember", key, start, err)
	return result, err
}

// SRandMemberN 随机获取集合中的多个成员（不删除）
func (c *Client) SRandMemberN(ctx context.Context, key string, count int64) ([]string, error) {
	start := time.Now()
	var result []string
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "srandmembern", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.SRandMemberN(ctx, key, count).Result()
	}, count)
	if err == nil {
		result = resultInterface.([]string)
	}

	c.recordOperation(ctx, "srandmembern", key, start, err, count)
	return result, err
}

// SUnion 计算多个集合的并集
func (c *Client) SUnion(ctx context.Context, keys ...string) ([]string, error) {
	start := time.Now()
	var result []string
	var err error

	resultInterface, err := c.tracer.TraceBatchOperation(ctx, "sunion", keys, func(ctx context.Context) (interface{}, error) {
		return c.rdb.SUnion(ctx, keys...).Result()
	})
	if err == nil {
		result = resultInterface.([]string)
	}

	c.recordOperation(ctx, "sunion", "", start, err, keys)
	return result, err
}

// SUnionStore 计算多个集合的并集并存储到目标集合
func (c *Client) SUnionStore(ctx context.Context, destination string, keys ...string) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceBatchOperation(ctx, "sunionstore", keys, func(ctx context.Context) (interface{}, error) {
		return c.rdb.SUnionStore(ctx, destination, keys...).Result()
	}, destination)
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "sunionstore", destination, start, err, keys)
	return result, err
}

// SInter 计算多个集合的交集
func (c *Client) SInter(ctx context.Context, keys ...string) ([]string, error) {
	start := time.Now()
	var result []string
	var err error

	resultInterface, err := c.tracer.TraceBatchOperation(ctx, "sinter", keys, func(ctx context.Context) (interface{}, error) {
		return c.rdb.SInter(ctx, keys...).Result()
	})
	if err == nil {
		result = resultInterface.([]string)
	}

	c.recordOperation(ctx, "sinter", "", start, err, keys)
	return result, err
}

// SInterStore 计算多个集合的交集并存储到目标集合
func (c *Client) SInterStore(ctx context.Context, destination string, keys ...string) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceBatchOperation(ctx, "sinterstore", keys, func(ctx context.Context) (interface{}, error) {
		return c.rdb.SInterStore(ctx, destination, keys...).Result()
	}, destination)
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "sinterstore", destination, start, err, keys)
	return result, err
}

// SDiff 计算多个集合的差集
func (c *Client) SDiff(ctx context.Context, keys ...string) ([]string, error) {
	start := time.Now()
	var result []string
	var err error

	resultInterface, err := c.tracer.TraceBatchOperation(ctx, "sdiff", keys, func(ctx context.Context) (interface{}, error) {
		return c.rdb.SDiff(ctx, keys...).Result()
	})
	if err == nil {
		result = resultInterface.([]string)
	}

	c.recordOperation(ctx, "sdiff", "", start, err, keys)
	return result, err
}

// SDiffStore 计算多个集合的差集并存储到目标集合
func (c *Client) SDiffStore(ctx context.Context, destination string, keys ...string) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceBatchOperation(ctx, "sdiffstore", keys, func(ctx context.Context) (interface{}, error) {
		return c.rdb.SDiffStore(ctx, destination, keys...).Result()
	}, destination)
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "sdiffstore", destination, start, err, keys)
	return result, err
}

// SMove 将成员从源集合移动到目标集合
func (c *Client) SMove(ctx context.Context, source, destination string, member interface{}) (bool, error) {
	start := time.Now()
	var result bool
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "smove", source, func(ctx context.Context) (interface{}, error) {
		return c.rdb.SMove(ctx, source, destination, member).Result()
	}, destination, member)
	if err == nil {
		result = resultInterface.(bool)
	}

	c.recordOperation(ctx, "smove", source, start, err, destination, member)
	return result, err
}

// SScan 扫描集合
func (c *Client) SScan(ctx context.Context, key string, cursor uint64, match string, count int64) ([]string, uint64, error) {
	start := time.Now()
	var keys []string
	var newCursor uint64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "sscan", key, func(ctx context.Context) (interface{}, error) {
		keys, cursor, err := c.rdb.SScan(ctx, key, cursor, match, count).Result()
		return []interface{}{keys, cursor}, err
	}, cursor, match, count)
	if err == nil {
		result := resultInterface.([]interface{})
		keys = result[0].([]string)
		newCursor = result[1].(uint64)
	}

	c.recordOperation(ctx, "sscan", key, start, err, cursor, match, count)
	return keys, newCursor, err
}

// SetFilter 集合过滤器
type SetFilter struct {
	client *Client
	key    string
}

// NewSetFilter 创建新的集合过滤器
func (c *Client) NewSetFilter(key string) *SetFilter {
	return &SetFilter{
		client: c,
		key:    key,
	}
}

// Add 添加元素到过滤器
func (sf *SetFilter) Add(ctx context.Context, members ...interface{}) (int64, error) {
	return sf.client.SAdd(ctx, sf.key, members...)
}

// Contains 检查元素是否在过滤器中
func (sf *SetFilter) Contains(ctx context.Context, member interface{}) (bool, error) {
	return sf.client.SIsMember(ctx, sf.key, member)
}

// Remove 从过滤器中删除元素
func (sf *SetFilter) Remove(ctx context.Context, members ...interface{}) (int64, error) {
	return sf.client.SRem(ctx, sf.key, members...)
}

// Size 获取过滤器大小
func (sf *SetFilter) Size(ctx context.Context) (int64, error) {
	return sf.client.SCard(ctx, sf.key)
}

// Members 获取过滤器所有成员
func (sf *SetFilter) Members(ctx context.Context) ([]string, error) {
	return sf.client.SMembers(ctx, sf.key)
}

// Clear 清空过滤器
func (sf *SetFilter) Clear(ctx context.Context) error {
	_, err := sf.client.rdb.Del(ctx, sf.key).Result()
	return err
}

// Union 与其他集合求并集
func (sf *SetFilter) Union(ctx context.Context, otherKeys ...string) ([]string, error) {
	allKeys := append([]string{sf.key}, otherKeys...)
	return sf.client.SUnion(ctx, allKeys...)
}

// Intersect 与其他集合求交集
func (sf *SetFilter) Intersect(ctx context.Context, otherKeys ...string) ([]string, error) {
	allKeys := append([]string{sf.key}, otherKeys...)
	return sf.client.SInter(ctx, allKeys...)
}

// Difference 与其他集合求差集
func (sf *SetFilter) Difference(ctx context.Context, otherKeys ...string) ([]string, error) {
	allKeys := append([]string{sf.key}, otherKeys...)
	return sf.client.SDiff(ctx, allKeys...)
}

// SetBatch 集合批量操作
type SetBatch struct {
	client  *Client
	key     string
	members []interface{}
	opType  string // "add" or "remove"
}

// NewSetBatch 创建新的集合批量操作
func (c *Client) NewSetBatch(key string, opType string) *SetBatch {
	return &SetBatch{
		client:  c,
		key:     key,
		members: make([]interface{}, 0),
		opType:  opType,
	}
}

// Add 添加成员到批量操作
func (sb *SetBatch) Add(members ...interface{}) *SetBatch {
	sb.members = append(sb.members, members...)
	return sb
}

// Execute 执行批量操作
func (sb *SetBatch) Execute(ctx context.Context) (int64, error) {
	if len(sb.members) == 0 {
		return 0, nil
	}

	switch sb.opType {
	case "add":
		return sb.client.SAdd(ctx, sb.key, sb.members...)
	case "remove":
		return sb.client.SRem(ctx, sb.key, sb.members...)
	default:
		return 0, nil
	}
}

// Clear 清空批量操作
func (sb *SetBatch) Clear() {
	sb.members = make([]interface{}, 0)
}

// Size 获取批量操作成员数量
func (sb *SetBatch) Size() int {
	return len(sb.members)
}

// SetOperator 集合运算器
type SetOperator struct {
	client *Client
}

// NewSetOperator 创建新的集合运算器
func (c *Client) NewSetOperator() *SetOperator {
	return &SetOperator{
		client: c,
	}
}

// Union 计算多个集合的并集
func (so *SetOperator) Union(ctx context.Context, keys ...string) ([]string, error) {
	return so.client.SUnion(ctx, keys...)
}

// UnionAndStore 计算多个集合的并集并存储
func (so *SetOperator) UnionAndStore(ctx context.Context, destination string, keys ...string) (int64, error) {
	return so.client.SUnionStore(ctx, destination, keys...)
}

// Intersect 计算多个集合的交集
func (so *SetOperator) Intersect(ctx context.Context, keys ...string) ([]string, error) {
	return so.client.SInter(ctx, keys...)
}

// IntersectAndStore 计算多个集合的交集并存储
func (so *SetOperator) IntersectAndStore(ctx context.Context, destination string, keys ...string) (int64, error) {
	return so.client.SInterStore(ctx, destination, keys...)
}

// Difference 计算多个集合的差集
func (so *SetOperator) Difference(ctx context.Context, keys ...string) ([]string, error) {
	return so.client.SDiff(ctx, keys...)
}

// DifferenceAndStore 计算多个集合的差集并存储
func (so *SetOperator) DifferenceAndStore(ctx context.Context, destination string, keys ...string) (int64, error) {
	return so.client.SDiffStore(ctx, destination, keys...)
}
