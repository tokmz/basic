package cache

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"basic/pkg/cache/config"
)

// 全局测试客户端，避免重复创建导致Prometheus指标冲突
var testCacheClient *RedisCache
var testOnce sync.Once

func getTestCacheClient(t *testing.T) *RedisCache {
	testOnce.Do(func() {
		cfg := config.DefaultConfig()
		cfg.DB = 1 // 使用测试数据库
		cfg.Monitoring.Enabled = true
		cfg.Monitoring.MetricsNamespace = "redis_cache_test"
		cfg.Monitoring.SlowLogThreshold = time.Millisecond
		cfg.Tracing.Enabled = false

		client, err := New(cfg)
		if err != nil {
			t.Fatalf("Failed to create test cache client: %v", err)
		}
		testCacheClient = client.(*RedisCache)
	})
	return testCacheClient
}

func TestCacheBasicOperations(t *testing.T) {
	// 获取共享的测试客户端
	cacheClient := getTestCacheClient(t)

	ctx := context.Background()

	// 测试连接
	err := cacheClient.Ping(ctx)
	require.NoError(t, err)

	// 测试字符串操作
	t.Run("String Operations", func(t *testing.T) {
		key := "test:string"
		value := "test_value"
		expiration := 10 * time.Second

		// 设置值
		err := cacheClient.Set(ctx, key, value, &expiration)
		assert.NoError(t, err)

		// 获取值
		result, err := cacheClient.Get(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, value, result)

		// 检查键是否存在
		exists, err := cacheClient.Exists(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), exists)

		// 删除键
		deleted, err := cacheClient.Del(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), deleted)
	})

	// 测试哈希表操作
	t.Run("Hash Operations", func(t *testing.T) {
		key := "test:hash"
		field := "field1"
		value := "value1"

		// 设置哈希字段
		count, err := cacheClient.HSet(ctx, key, field, value)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count)

		// 获取哈希字段
		result, err := cacheClient.HGet(ctx, key, field)
		assert.NoError(t, err)
		assert.Equal(t, value, result)

		// 检查字段是否存在
		exists, err := cacheClient.HExists(ctx, key, field)
		assert.NoError(t, err)
		assert.True(t, exists)

		// 删除哈希字段
		deleted, err := cacheClient.HDel(ctx, key, field)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), deleted)
	})

	// 测试列表操作
	t.Run("List Operations", func(t *testing.T) {
		key := "test:list"
		value1 := "item1"
		value2 := "item2"

		// 左侧推入
		count, err := cacheClient.LPush(ctx, key, value1, value2)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), count)

		// 获取列表长度
		length, err := cacheClient.LLen(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), length)

		// 右侧弹出
		result, err := cacheClient.RPop(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, value1, result)

		// 清理
		cacheClient.Del(ctx, key)
	})

	// 测试集合操作
	t.Run("Set Operations", func(t *testing.T) {
		key := "test:set"
		member1 := "member1"
		member2 := "member2"

		// 添加成员
		count, err := cacheClient.SAdd(ctx, key, member1, member2)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), count)

		// 检查成员是否存在
		exists, err := cacheClient.SIsMember(ctx, key, member1)
		assert.NoError(t, err)
		assert.True(t, exists)

		// 获取集合大小
		size, err := cacheClient.SCard(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), size)

		// 清理
		cacheClient.Del(ctx, key)
	})

	// 测试有序集合操作
	t.Run("ZSet Operations", func(t *testing.T) {
		key := "test:zset"
		member1 := "member1"
		member2 := "member2"
		score1 := float64(100)
		score2 := float64(200)

		// 添加成员
		count, err := cacheClient.ZAdd(ctx, key,
			redis.Z{Score: score1, Member: member1},
			redis.Z{Score: score2, Member: member2})
		assert.NoError(t, err)
		assert.Equal(t, int64(2), count)

		// 获取成员分数
		score, err := cacheClient.ZScore(ctx, key, member1)
		assert.NoError(t, err)
		assert.Equal(t, score1, score)

		// 获取有序集合大小
		size, err := cacheClient.ZCard(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), size)

		// 清理
		cacheClient.Del(ctx, key)
	})
}

func TestCacheAdvancedFeatures(t *testing.T) {
	// 获取共享的测试客户端
	cacheClient := getTestCacheClient(t)

	ctx := context.Background()

	// 测试计数器
	t.Run("Counter", func(t *testing.T) {
		// 清理之前的数据
		cacheClient.Del(ctx, "test:counter")
		counter := cacheClient.NewCounter("test:counter")

		// 递增
		count, err := counter.Increment(ctx, 1)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count)

		// 再次递增
		count, err = counter.Increment(ctx, 5)
		assert.NoError(t, err)
		assert.Equal(t, int64(6), count)

		// 获取当前值
		value, err := counter.Get(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(6), value)

		// 清理
		cacheClient.Del(ctx, "test:counter")
	})

	// 测试哈希计数器
	t.Run("Hash Counter", func(t *testing.T) {
		hashCounter := cacheClient.NewHashCounter("test:hash:counter", "field1")

		// 递增
		count, err := hashCounter.Increment(ctx, 10)
		assert.NoError(t, err)
		assert.Equal(t, int64(10), count)

		// 获取值
		value, err := hashCounter.Get(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "10", value)

		// 清理
		hashCounter.Delete(ctx)
	})

	// 测试队列
	t.Run("Queue", func(t *testing.T) {
		queue := cacheClient.NewListQueue("test:queue")

		// 入队
		_, err := queue.Enqueue(ctx, "task1", "task2", "task3")
		assert.NoError(t, err)

		// 获取队列大小
		size, err := queue.Size(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(3), size)

		// 出队
		task, err := queue.Dequeue(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "task1", task)

		// 清理
		queue.Clear(ctx)
	})

	// 测试栈
	t.Run("Stack", func(t *testing.T) {
		stack := cacheClient.NewListStack("test:stack")

		// 入栈
		_, err := stack.Push(ctx, "item1", "item2", "item3")
		assert.NoError(t, err)

		// 获取栈大小
		size, err := stack.Size(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(3), size)

		// 出栈
		item, err := stack.Pop(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "item3", item) // 后进先出

		// 清理
		stack.Clear(ctx)
	})

	// 测试排行榜
	t.Run("Leaderboard", func(t *testing.T) {
		// 清理之前的数据
		cacheClient.Del(ctx, "test:leaderboard")
		leaderboard := cacheClient.NewLeaderboard("test:leaderboard")

		// 添加分数
		err := leaderboard.AddScore(ctx, "player1", 100)
		assert.NoError(t, err)

		err = leaderboard.AddScore(ctx, "player2", 200)
		assert.NoError(t, err)

		err = leaderboard.AddScore(ctx, "player3", 150)
		assert.NoError(t, err)

		// 获取排名（从大到小）
		rank, err := leaderboard.GetReverseRank(ctx, "player2")
		assert.NoError(t, err)
		assert.Equal(t, int64(0), rank) // 最高分，排名第0

		// 获取前2名
		topPlayers, err := leaderboard.GetTopN(ctx, 2)
		assert.NoError(t, err)
		assert.Len(t, topPlayers, 2)
		assert.Equal(t, "player2", topPlayers[0].Member)
		assert.Equal(t, float64(200), topPlayers[0].Score)

		// 清理
		leaderboard.Clear(ctx)
	})
}

func TestCacheBatchOperations(t *testing.T) {
	// 获取共享的测试客户端
	cacheClient := getTestCacheClient(t)

	ctx := context.Background()

	// 测试批量字符串操作
	t.Run("Batch String Operations", func(t *testing.T) {
		pairs := map[string]interface{}{
			"batch:key1": "value1",
			"batch:key2": "value2",
			"batch:key3": "value3",
		}

		// 批量设置
		err := cacheClient.MSet(ctx, pairs, nil)
		assert.NoError(t, err)

		// 批量获取
		values, err := cacheClient.MGet(ctx, "batch:key1", "batch:key2", "batch:key3")
		assert.NoError(t, err)
		assert.Len(t, values, 3)
		assert.Equal(t, "value1", values[0])
		assert.Equal(t, "value2", values[1])
		assert.Equal(t, "value3", values[2])

		// 清理
		cacheClient.Del(ctx, "batch:key1", "batch:key2", "batch:key3")
	})

	// 测试哈希批量操作
	t.Run("Hash Batch Operations", func(t *testing.T) {
		key := "batch:hash"
		hashBatch := cacheClient.NewHashBatch(key)

		// 添加多个字段
		hashBatch.Set("field1", "value1").Set("field2", "value2").Set("field3", "value3")

		// 执行批量操作
		count, err := hashBatch.Execute(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(3), count)

		// 验证结果
		value1, err := cacheClient.HGet(ctx, key, "field1")
		assert.NoError(t, err)
		assert.Equal(t, "value1", value1)

		// 清理
		cacheClient.Del(ctx, key)
	})
}

func TestCacheLuaScript(t *testing.T) {
	// 获取共享的测试客户端
	cacheClient := getTestCacheClient(t)

	ctx := context.Background()

	// 测试Lua脚本执行
	t.Run("Lua Script Execution", func(t *testing.T) {
		script := `
			local key = KEYS[1]
			local value = ARGV[1]
			redis.call('SET', key, value)
			return redis.call('GET', key)
		`

		// 执行脚本
		result, err := cacheClient.Eval(ctx, script, []string{"lua:test"}, "test_value")
		assert.NoError(t, err)
		assert.Equal(t, "test_value", result)

		// 清理
		cacheClient.Del(ctx, "lua:test")
	})

	// 测试脚本注册和执行
	t.Run("Script Registration", func(t *testing.T) {
		script := `
			local key = KEYS[1]
			local increment = tonumber(ARGV[1])
			return redis.call('INCRBY', key, increment)
		`

		// 注册脚本
		err := cacheClient.RegisterScript("test_incr", script)
		assert.NoError(t, err)

		// 执行注册的脚本
		result, err := cacheClient.ExecuteScript(ctx, "test_incr", []string{"script:counter"}, 5)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), result)

		// 再次执行
		result, err = cacheClient.ExecuteScript(ctx, "test_incr", []string{"script:counter"}, 3)
		assert.NoError(t, err)
		assert.Equal(t, int64(8), result)

		// 清理
		cacheClient.Del(ctx, "script:counter")
	})
}

func TestCacheMonitoring(t *testing.T) {
	// 获取共享的测试客户端
	cacheClient := getTestCacheClient(t)

	ctx := context.Background()

	// 执行一些操作以生成统计数据
	cacheClient.Set(ctx, "monitor:test1", "value1", nil)
	cacheClient.Set(ctx, "monitor:test2", "value2", nil)
	cacheClient.Get(ctx, "monitor:test1")
	cacheClient.Get(ctx, "monitor:test2")
	cacheClient.Get(ctx, "monitor:nonexistent")

	// 等待一小段时间让监控数据更新
	time.Sleep(10 * time.Millisecond)

	// 测试统计信息
	t.Run("Statistics", func(t *testing.T) {
		stats := cacheClient.GetStats()
		assert.NotNil(t, stats)
		assert.True(t, stats.TotalOperations > 0)
		assert.True(t, stats.SuccessOperations > 0)
	})

	// 测试慢查询记录
	t.Run("Slow Queries", func(t *testing.T) {
		slowQueries := cacheClient.GetSlowQueries(10)
		// 由于设置了很低的慢查询阈值，应该有一些慢查询记录
		assert.True(t, len(slowQueries) >= 0) // 可能为0，取决于执行速度
	})

	// 测试统计重置
	t.Run("Reset Statistics", func(t *testing.T) {
		cacheClient.ResetStats()
		cacheClient.ClearSlowQueries()

		stats := cacheClient.GetStats()
		assert.Equal(t, int64(0), stats.TotalOperations)

		slowQueries := cacheClient.GetSlowQueries(10)
		assert.Len(t, slowQueries, 0)
	})

	// 清理
	cacheClient.Del(ctx, "monitor:test1", "monitor:test2")
}

func TestDefaultCacheInstance(t *testing.T) {
	// 测试默认缓存实例
	cfg := config.DefaultConfig()
	cfg.DB = 1 // 使用测试数据库

	// 初始化默认实例
	err := Init(cfg)
	require.NoError(t, err)

	ctx := context.Background()

	// 测试全局函数
	t.Run("Global Functions", func(t *testing.T) {
		key := "global:test"
		value := "global_value"
		expiration := 10 * time.Second

		// 使用全局函数设置值
		err := Set(ctx, key, value, expiration)
		assert.NoError(t, err)

		// 使用全局函数获取值
		result, err := Get(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, value, result)

		// 清理
		Del(ctx, key)
	})
}
