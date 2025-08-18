package client

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// ZSetOperations 有序集合操作接口
type ZSetOperations interface {
	// 基础操作
	ZAdd(ctx context.Context, key string, members ...redis.Z) (int64, error)
	ZRem(ctx context.Context, key string, members ...interface{}) (int64, error)
	ZScore(ctx context.Context, key, member string) (float64, error)
	ZRank(ctx context.Context, key, member string) (int64, error)
	ZRevRank(ctx context.Context, key, member string) (int64, error)
	ZCard(ctx context.Context, key string) (int64, error)
	ZCount(ctx context.Context, key, min, max string) (int64, error)

	// 范围查询
	ZRange(ctx context.Context, key string, start, stop int64) ([]string, error)
	ZRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error)
	ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error)
	ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error)
	ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) ([]string, error)
	ZRangeByScoreWithScores(ctx context.Context, key string, opt *redis.ZRangeBy) ([]redis.Z, error)
	ZRevRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) ([]string, error)
	ZRevRangeByScoreWithScores(ctx context.Context, key string, opt *redis.ZRangeBy) ([]redis.Z, error)

	// 原子操作
	ZIncrBy(ctx context.Context, key string, increment float64, member string) (float64, error)

	// 删除操作
	ZRemRangeByRank(ctx context.Context, key string, start, stop int64) (int64, error)
	ZRemRangeByScore(ctx context.Context, key, min, max string) (int64, error)

	// 集合运算
	ZUnionStore(ctx context.Context, dest string, store *redis.ZStore) (int64, error)
	ZInterStore(ctx context.Context, dest string, store *redis.ZStore) (int64, error)

	// 扫描操作
	ZScan(ctx context.Context, key string, cursor uint64, match string, count int64) ([]string, uint64, error)

	// 弹出操作
	ZPopMax(ctx context.Context, key string, count ...int64) ([]redis.Z, error)
	ZPopMin(ctx context.Context, key string, count ...int64) ([]redis.Z, error)
	BZPopMax(ctx context.Context, timeout time.Duration, keys ...string) ([]redis.Z, error)
	BZPopMin(ctx context.Context, timeout time.Duration, keys ...string) ([]redis.Z, error)
}

// ZAdd 向有序集合添加成员
func (c *Client) ZAdd(ctx context.Context, key string, members ...redis.Z) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "zadd", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.ZAdd(ctx, key, members...).Result()
	}, members)
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "zadd", key, start, err, members)
	return result, err
}

// ZRem 从有序集合删除成员
func (c *Client) ZRem(ctx context.Context, key string, members ...interface{}) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "zrem", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.ZRem(ctx, key, members...).Result()
	}, members)
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "zrem", key, start, err, members)
	return result, err
}

// ZScore 获取有序集合成员的分数
func (c *Client) ZScore(ctx context.Context, key, member string) (float64, error) {
	start := time.Now()
	var result float64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "zscore", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.ZScore(ctx, key, member).Result()
	}, member)
	if err == nil {
		result = resultInterface.(float64)
	}

	c.recordOperation(ctx, "zscore", key, start, err, member)
	return result, err
}

// ZRank 获取有序集合成员的排名（从小到大）
func (c *Client) ZRank(ctx context.Context, key, member string) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "zrank", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.ZRank(ctx, key, member).Result()
	}, member)
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "zrank", key, start, err, member)
	return result, err
}

// ZRevRank 获取有序集合成员的排名（从大到小）
func (c *Client) ZRevRank(ctx context.Context, key, member string) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "zrevrank", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.ZRevRank(ctx, key, member).Result()
	}, member)
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "zrevrank", key, start, err, member)
	return result, err
}

// ZCard 获取有序集合成员数量
func (c *Client) ZCard(ctx context.Context, key string) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "zcard", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.ZCard(ctx, key).Result()
	})
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "zcard", key, start, err)
	return result, err
}

// ZCount 计算有序集合指定分数范围的成员数量
func (c *Client) ZCount(ctx context.Context, key, min, max string) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "zcount", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.ZCount(ctx, key, min, max).Result()
	}, min, max)
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "zcount", key, start, err, min, max)
	return result, err
}

// ZRange 获取有序集合指定排名范围的成员（从小到大）
func (c *Client) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	startTime := time.Now()
	var result []string
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "zrange", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.ZRange(ctx, key, start, stop).Result()
	}, start, stop)
	if err == nil {
		result = resultInterface.([]string)
	}

	c.recordOperation(ctx, "zrange", key, startTime, err, start, stop)
	return result, err
}

// ZRangeWithScores 获取有序集合指定排名范围的成员和分数（从小到大）
func (c *Client) ZRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error) {
	startTime := time.Now()
	var result []redis.Z
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "zrangewithscores", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.ZRangeWithScores(ctx, key, start, stop).Result()
	}, start, stop)
	if err == nil {
		result = resultInterface.([]redis.Z)
	}

	c.recordOperation(ctx, "zrangewithscores", key, startTime, err, start, stop)
	return result, err
}

// ZRevRange 获取有序集合指定排名范围的成员（从大到小）
func (c *Client) ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	startTime := time.Now()
	var result []string
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "zrevrange", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.ZRevRange(ctx, key, start, stop).Result()
	}, start, stop)
	if err == nil {
		result = resultInterface.([]string)
	}

	c.recordOperation(ctx, "zrevrange", key, startTime, err, start, stop)
	return result, err
}

// ZRevRangeWithScores 获取有序集合指定排名范围的成员和分数（从大到小）
func (c *Client) ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error) {
	startTime := time.Now()
	var result []redis.Z
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "zrevrangewithscores", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.ZRevRangeWithScores(ctx, key, start, stop).Result()
	}, start, stop)
	if err == nil {
		result = resultInterface.([]redis.Z)
	}

	c.recordOperation(ctx, "zrevrangewithscores", key, startTime, err, start, stop)
	return result, err
}

// ZRangeByScore 根据分数范围获取有序集合成员
func (c *Client) ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) ([]string, error) {
	start := time.Now()
	var result []string
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "zrangebyscore", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.ZRangeByScore(ctx, key, opt).Result()
	}, opt)
	if err == nil {
		result = resultInterface.([]string)
	}

	c.recordOperation(ctx, "zrangebyscore", key, start, err, opt)
	return result, err
}

// ZRangeByScoreWithScores 根据分数范围获取有序集合成员和分数
func (c *Client) ZRangeByScoreWithScores(ctx context.Context, key string, opt *redis.ZRangeBy) ([]redis.Z, error) {
	start := time.Now()
	var result []redis.Z
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "zrangebyscorewithscores", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.ZRangeByScoreWithScores(ctx, key, opt).Result()
	}, opt)
	if err == nil {
		result = resultInterface.([]redis.Z)
	}

	c.recordOperation(ctx, "zrangebyscorewithscores", key, start, err, opt)
	return result, err
}

// ZRevRangeByScore 根据分数范围获取有序集合成员（从大到小）
func (c *Client) ZRevRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) ([]string, error) {
	start := time.Now()
	var result []string
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "zrevrangebyscore", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.ZRevRangeByScore(ctx, key, opt).Result()
	}, opt)
	if err == nil {
		result = resultInterface.([]string)
	}

	c.recordOperation(ctx, "zrevrangebyscore", key, start, err, opt)
	return result, err
}

// ZRevRangeByScoreWithScores 根据分数范围获取有序集合成员和分数（从大到小）
func (c *Client) ZRevRangeByScoreWithScores(ctx context.Context, key string, opt *redis.ZRangeBy) ([]redis.Z, error) {
	start := time.Now()
	var result []redis.Z
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "zrevrangebyscorewithscores", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.ZRevRangeByScoreWithScores(ctx, key, opt).Result()
	}, opt)
	if err == nil {
		result = resultInterface.([]redis.Z)
	}

	c.recordOperation(ctx, "zrevrangebyscorewithscores", key, start, err, opt)
	return result, err
}

// ZIncrBy 增加有序集合成员的分数
func (c *Client) ZIncrBy(ctx context.Context, key string, increment float64, member string) (float64, error) {
	start := time.Now()
	var result float64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "zincrby", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.ZIncrBy(ctx, key, increment, member).Result()
	}, increment, member)
	if err == nil {
		result = resultInterface.(float64)
	}

	c.recordOperation(ctx, "zincrby", key, start, err, increment, member)
	return result, err
}

// ZRemRangeByRank 根据排名范围删除有序集合成员
func (c *Client) ZRemRangeByRank(ctx context.Context, key string, start, stop int64) (int64, error) {
	startTime := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "zremrangebyrank", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.ZRemRangeByRank(ctx, key, start, stop).Result()
	}, start, stop)
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "zremrangebyrank", key, startTime, err, start, stop)
	return result, err
}

// ZRemRangeByScore 根据分数范围删除有序集合成员
func (c *Client) ZRemRangeByScore(ctx context.Context, key, min, max string) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "zremrangebyscore", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.ZRemRangeByScore(ctx, key, min, max).Result()
	}, min, max)
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "zremrangebyscore", key, start, err, min, max)
	return result, err
}

// ZUnionStore 计算有序集合的并集并存储
func (c *Client) ZUnionStore(ctx context.Context, dest string, store *redis.ZStore) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "zunionstore", dest, func(ctx context.Context) (interface{}, error) {
		return c.rdb.ZUnionStore(ctx, dest, store).Result()
	}, store)
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "zunionstore", dest, start, err, store)
	return result, err
}

// ZInterStore 计算有序集合的交集并存储
func (c *Client) ZInterStore(ctx context.Context, dest string, store *redis.ZStore) (int64, error) {
	start := time.Now()
	var result int64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "zinterstore", dest, func(ctx context.Context) (interface{}, error) {
		return c.rdb.ZInterStore(ctx, dest, store).Result()
	}, store)
	if err == nil {
		result = resultInterface.(int64)
	}

	c.recordOperation(ctx, "zinterstore", dest, start, err, store)
	return result, err
}

// ZScan 扫描有序集合
func (c *Client) ZScan(ctx context.Context, key string, cursor uint64, match string, count int64) ([]string, uint64, error) {
	start := time.Now()
	var keys []string
	var newCursor uint64
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "zscan", key, func(ctx context.Context) (interface{}, error) {
		keys, cursor, err := c.rdb.ZScan(ctx, key, cursor, match, count).Result()
		return []interface{}{keys, cursor}, err
	}, cursor, match, count)
	if err == nil {
		result := resultInterface.([]interface{})
		keys = result[0].([]string)
		newCursor = result[1].(uint64)
	}

	c.recordOperation(ctx, "zscan", key, start, err, cursor, match, count)
	return keys, newCursor, err
}

// ZPopMax 弹出有序集合中分数最高的成员
func (c *Client) ZPopMax(ctx context.Context, key string, count ...int64) ([]redis.Z, error) {
	start := time.Now()
	var result []redis.Z
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "zpopmax", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.ZPopMax(ctx, key, count...).Result()
	}, count)
	if err == nil {
		result = resultInterface.([]redis.Z)
	}

	c.recordOperation(ctx, "zpopmax", key, start, err, count)
	return result, err
}

// ZPopMin 弹出有序集合中分数最低的成员
func (c *Client) ZPopMin(ctx context.Context, key string, count ...int64) ([]redis.Z, error) {
	start := time.Now()
	var result []redis.Z
	var err error

	resultInterface, err := c.tracer.TraceOperation(ctx, "zpopmin", key, func(ctx context.Context) (interface{}, error) {
		return c.rdb.ZPopMin(ctx, key, count...).Result()
	}, count)
	if err == nil {
		result = resultInterface.([]redis.Z)
	}

	c.recordOperation(ctx, "zpopmin", key, start, err, count)
	return result, err
}

// BZPopMax 阻塞式弹出有序集合中分数最高的成员
func (c *Client) BZPopMax(ctx context.Context, timeout time.Duration, keys ...string) ([]redis.Z, error) {
	start := time.Now()
	var result []redis.Z
	var err error

	resultInterface, err := c.tracer.TraceBatchOperation(ctx, "bzpopmax", keys, func(ctx context.Context) (interface{}, error) {
		return c.rdb.BZPopMax(ctx, timeout, keys...).Result()
	}, timeout)
	if err == nil {
		result = resultInterface.([]redis.Z)
	}

	c.recordOperation(ctx, "bzpopmax", "", start, err, timeout, keys)
	return result, err
}

// BZPopMin 阻塞式弹出有序集合中分数最低的成员
func (c *Client) BZPopMin(ctx context.Context, timeout time.Duration, keys ...string) ([]redis.Z, error) {
	start := time.Now()
	var result []redis.Z
	var err error

	resultInterface, err := c.tracer.TraceBatchOperation(ctx, "bzpopmin", keys, func(ctx context.Context) (interface{}, error) {
		return c.rdb.BZPopMin(ctx, timeout, keys...).Result()
	}, timeout)
	if err == nil {
		result = resultInterface.([]redis.Z)
	}

	c.recordOperation(ctx, "bzpopmin", "", start, err, timeout, keys)
	return result, err
}

// Leaderboard 排行榜实现
type Leaderboard struct {
	client *Client
	key    string
}

// NewLeaderboard 创建新的排行榜
func (c *Client) NewLeaderboard(key string) *Leaderboard {
	return &Leaderboard{
		client: c,
		key:    key,
	}
}

// AddScore 添加或更新成员分数
func (lb *Leaderboard) AddScore(ctx context.Context, member string, score float64) error {
	_, err := lb.client.ZAdd(ctx, lb.key, redis.Z{Score: score, Member: member})
	return err
}

// IncrementScore 增加成员分数
func (lb *Leaderboard) IncrementScore(ctx context.Context, member string, increment float64) (float64, error) {
	return lb.client.ZIncrBy(ctx, lb.key, increment, member)
}

// GetScore 获取成员分数
func (lb *Leaderboard) GetScore(ctx context.Context, member string) (float64, error) {
	return lb.client.ZScore(ctx, lb.key, member)
}

// GetRank 获取成员排名（从小到大，0开始）
func (lb *Leaderboard) GetRank(ctx context.Context, member string) (int64, error) {
	return lb.client.ZRank(ctx, lb.key, member)
}

// GetReverseRank 获取成员排名（从大到小，0开始）
func (lb *Leaderboard) GetReverseRank(ctx context.Context, member string) (int64, error) {
	return lb.client.ZRevRank(ctx, lb.key, member)
}

// GetTopN 获取前N名成员（从大到小）
func (lb *Leaderboard) GetTopN(ctx context.Context, n int64) ([]redis.Z, error) {
	return lb.client.ZRevRangeWithScores(ctx, lb.key, 0, n-1)
}

// GetBottomN 获取后N名成员（从小到大）
func (lb *Leaderboard) GetBottomN(ctx context.Context, n int64) ([]redis.Z, error) {
	return lb.client.ZRangeWithScores(ctx, lb.key, 0, n-1)
}

// GetRangeByRank 获取指定排名范围的成员（从大到小）
func (lb *Leaderboard) GetRangeByRank(ctx context.Context, start, stop int64) ([]redis.Z, error) {
	return lb.client.ZRevRangeWithScores(ctx, lb.key, start, stop)
}

// GetRangeByScore 获取指定分数范围的成员
func (lb *Leaderboard) GetRangeByScore(ctx context.Context, min, max string, offset, count int64) ([]redis.Z, error) {
	opt := &redis.ZRangeBy{
		Min:    min,
		Max:    max,
		Offset: offset,
		Count:  count,
	}
	return lb.client.ZRangeByScoreWithScores(ctx, lb.key, opt)
}

// GetAroundMember 获取指定成员周围的排名
func (lb *Leaderboard) GetAroundMember(ctx context.Context, member string, count int64) ([]redis.Z, error) {
	rank, err := lb.client.ZRevRank(ctx, lb.key, member)
	if err != nil {
		return nil, err
	}

	start := rank - count/2
	stop := rank + count/2
	if start < 0 {
		start = 0
	}

	return lb.client.ZRevRangeWithScores(ctx, lb.key, start, stop)
}

// RemoveMember 删除成员
func (lb *Leaderboard) RemoveMember(ctx context.Context, members ...string) (int64, error) {
	interfaces := make([]interface{}, len(members))
	for i, member := range members {
		interfaces[i] = member
	}
	return lb.client.ZRem(ctx, lb.key, interfaces...)
}

// GetTotalMembers 获取总成员数
func (lb *Leaderboard) GetTotalMembers(ctx context.Context) (int64, error) {
	return lb.client.ZCard(ctx, lb.key)
}

// Clear 清空排行榜
func (lb *Leaderboard) Clear(ctx context.Context) error {
	_, err := lb.client.rdb.Del(ctx, lb.key).Result()
	return err
}

// GetPercentileRank 获取成员的百分位排名
func (lb *Leaderboard) GetPercentileRank(ctx context.Context, member string) (float64, error) {
	rank, err := lb.GetReverseRank(ctx, member)
	if err != nil {
		return 0, err
	}

	total, err := lb.GetTotalMembers(ctx)
	if err != nil {
		return 0, err
	}

	if total == 0 {
		return 0, nil
	}

	return float64(rank) / float64(total) * 100, nil
}

// ZSetBatch 有序集合批量操作
type ZSetBatch struct {
	client  *Client
	key     string
	members []redis.Z
}

// NewZSetBatch 创建新的有序集合批量操作
func (c *Client) NewZSetBatch(key string) *ZSetBatch {
	return &ZSetBatch{
		client:  c,
		key:     key,
		members: make([]redis.Z, 0),
	}
}

// Add 添加成员到批量操作
func (zb *ZSetBatch) Add(score float64, member string) *ZSetBatch {
	zb.members = append(zb.members, redis.Z{Score: score, Member: member})
	return zb
}

// AddZ 添加Z结构到批量操作
func (zb *ZSetBatch) AddZ(members ...redis.Z) *ZSetBatch {
	zb.members = append(zb.members, members...)
	return zb
}

// Execute 执行批量操作
func (zb *ZSetBatch) Execute(ctx context.Context) (int64, error) {
	if len(zb.members) == 0 {
		return 0, nil
	}
	return zb.client.ZAdd(ctx, zb.key, zb.members...)
}

// Clear 清空批量操作
func (zb *ZSetBatch) Clear() {
	zb.members = make([]redis.Z, 0)
}

// Size 获取批量操作成员数量
func (zb *ZSetBatch) Size() int {
	return len(zb.members)
}
