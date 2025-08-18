package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"basic/pkg/cache"
	"basic/pkg/cache/config"
)

func main() {
	// 示例1: 使用默认配置创建缓存客户端
	example1()

	// 示例2: 使用自定义配置创建缓存客户端
	example2()

	// 示例3: 使用Redis选项创建缓存客户端
	example3()

	// 示例4: 字符串操作示例
	example4()

	// 示例5: 哈希表操作示例
	example5()

	// 示例6: 列表操作示例
	example6()

	// 示例7: 集合操作示例
	example7()

	// 示例8: 有序集合操作示例
	example8()

	// 示例9: Lua脚本操作示例
	example9()

	// 示例10: 批量操作示例
	example10()

	// 示例11: 高级功能示例
	example11()

	// 示例12: 监控和统计示例
	example12()
}

// 示例1: 使用默认配置
func example1() {
	fmt.Println("=== 示例1: 使用默认配置 ===")

	// 使用默认配置创建缓存客户端
	cfg := config.DefaultConfig()
	cacheClient, err := cache.New(cfg)
	if err != nil {
		log.Fatalf("创建缓存客户端失败: %v", err)
	}
	defer cacheClient.Close()

	ctx := context.Background()

	// 测试连接
	if err := cacheClient.Ping(ctx); err != nil {
		log.Fatalf("连接Redis失败: %v", err)
	}

	fmt.Println("✓ 成功连接到Redis")
}

// 示例2: 使用自定义配置
func example2() {
	fmt.Println("\n=== 示例2: 使用自定义配置 ===")

	// 创建自定义配置
	cfg := &config.RedisConfig{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1, // 使用数据库1

		// 连接池配置
		PoolSize:        20,
		MinIdleConns:    5,
		MaxIdleConns:    10,
		ConnMaxIdleTime: 30 * time.Minute,
		ConnMaxLifetime: 2 * time.Hour,

		// 超时配置
		DialTimeout:  10 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,

		// 重试配置
		MaxRetries:      5,
		MinRetryBackoff: 10 * time.Millisecond,
		MaxRetryBackoff: 1 * time.Second,

		// 默认过期时间
		DefaultExpiration: 12 * time.Hour,

		// 链路追踪配置
		Tracing: config.TracingConfig{
			Enabled:     true,
			ServiceName: "my-redis-cache",
			Version:     "2.0.0",
		},

		// 监控配置
		Monitoring: config.MonitoringConfig{
			Enabled:          true,
			MetricsNamespace: "my_app_redis",
			SlowLogThreshold: 50 * time.Millisecond,
			CollectInterval:  5 * time.Second,
		},
	}

	cacheClient, err := cache.New(cfg)
	if err != nil {
		log.Fatalf("创建缓存客户端失败: %v", err)
	}
	defer cacheClient.Close()

	fmt.Println("✓ 成功创建自定义配置的缓存客户端")
}

// 示例3: 使用Redis选项
func example3() {
	fmt.Println("\n=== 示例3: 使用Redis选项 ===")

	// 使用Redis原生选项
	opt := &redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       2,
		PoolSize: 15,
	}

	cacheClient, err := cache.NewWithOptions(opt)
	if err != nil {
		log.Fatalf("创建缓存客户端失败: %v", err)
	}
	defer cacheClient.Close()

	fmt.Println("✓ 成功使用Redis选项创建缓存客户端")
}

// 示例4: 字符串操作
func example4() {
	fmt.Println("\n=== 示例4: 字符串操作 ===")

	cfg := config.DefaultConfig()
	cacheClient, err := cache.New(cfg)
	if err != nil {
		log.Fatalf("创建缓存客户端失败: %v", err)
	}
	defer cacheClient.Close()

	ctx := context.Background()

	// 基础操作
	expiration := 10 * time.Minute
	err = cacheClient.Set(ctx, "user:1001", "张三", &expiration)
	if err != nil {
		log.Printf("设置键值失败: %v", err)
		return
	}

	value, err := cacheClient.Get(ctx, "user:1001")
	if err != nil {
		log.Printf("获取键值失败: %v", err)
		return
	}
	fmt.Printf("获取到的值: %s\n", value)

	// 批量操作
	pairs := map[string]interface{}{
		"user:1002": "李四",
		"user:1003": "王五",
		"user:1004": "赵六",
	}
	err = cacheClient.MSet(ctx, pairs, &expiration)
	if err != nil {
		log.Printf("批量设置失败: %v", err)
		return
	}

	values, err := cacheClient.MGet(ctx, "user:1002", "user:1003", "user:1004")
	if err != nil {
		log.Printf("批量获取失败: %v", err)
		return
	}
	fmt.Printf("批量获取的值: %v\n", values)

	// 原子操作
	counter := cacheClient.NewCounter("page_views")
	count, err := counter.Increment(ctx, 1)
	if err != nil {
		log.Printf("计数器递增失败: %v", err)
		return
	}
	fmt.Printf("页面访问次数: %d\n", count)

	fmt.Println("✓ 字符串操作完成")
}

// 示例5: 哈希表操作
func example5() {
	fmt.Println("\n=== 示例5: 哈希表操作 ===")

	cfg := config.DefaultConfig()
	cacheClient, err := cache.New(cfg)
	if err != nil {
		log.Fatalf("创建缓存客户端失败: %v", err)
	}
	defer cacheClient.Close()

	ctx := context.Background()

	// 设置哈希表字段
	userKey := "user:profile:1001"
	_, err = cacheClient.HSet(ctx, userKey, "name", "张三", "age", "25", "city", "北京")
	if err != nil {
		log.Printf("设置哈希表失败: %v", err)
		return
	}

	// 获取单个字段
	name, err := cacheClient.HGet(ctx, userKey, "name")
	if err != nil {
		log.Printf("获取哈希表字段失败: %v", err)
		return
	}
	fmt.Printf("用户姓名: %s\n", name)

	// 获取所有字段
	profile, err := cacheClient.HGetAll(ctx, userKey)
	if err != nil {
		log.Printf("获取所有字段失败: %v", err)
		return
	}
	fmt.Printf("用户资料: %v\n", profile)

	// 批量操作
	hashBatch := cacheClient.NewHashBatch(userKey)
	hashBatch.Set("email", "zhangsan@example.com").Set("phone", "13800138000")
	_, err = hashBatch.Execute(ctx)
	if err != nil {
		log.Printf("批量设置哈希表失败: %v", err)
		return
	}

	// 哈希表计数器
	hashCounter := cacheClient.NewHashCounter("stats:daily", "login_count")
	count, err := hashCounter.Increment(ctx, 1)
	if err != nil {
		log.Printf("哈希表计数器递增失败: %v", err)
		return
	}
	fmt.Printf("今日登录次数: %d\n", count)

	fmt.Println("✓ 哈希表操作完成")
}

// 示例6: 列表操作
func example6() {
	fmt.Println("\n=== 示例6: 列表操作 ===")

	cfg := config.DefaultConfig()
	cacheClient, err := cache.New(cfg)
	if err != nil {
		log.Fatalf("创建缓存客户端失败: %v", err)
	}
	defer cacheClient.Close()

	ctx := context.Background()

	// 基础列表操作
	listKey := "message_queue"
	_, err = cacheClient.LPush(ctx, listKey, "消息1", "消息2", "消息3")
	if err != nil {
		log.Printf("推入列表失败: %v", err)
		return
	}

	message, err := cacheClient.RPop(ctx, listKey)
	if err != nil {
		log.Printf("弹出列表元素失败: %v", err)
		return
	}
	fmt.Printf("弹出的消息: %s\n", message)

	// 队列操作
	queue := cacheClient.NewListQueue("task_queue")
	_, err = queue.Enqueue(ctx, "任务1", "任务2", "任务3")
	if err != nil {
		log.Printf("入队失败: %v", err)
		return
	}

	task, err := queue.Dequeue(ctx)
	if err != nil {
		log.Printf("出队失败: %v", err)
		return
	}
	fmt.Printf("出队的任务: %s\n", task)

	// 栈操作
	stack := cacheClient.NewListStack("operation_stack")
	_, err = stack.Push(ctx, "操作1", "操作2", "操作3")
	if err != nil {
		log.Printf("入栈失败: %v", err)
		return
	}

	operation, err := stack.Pop(ctx)
	if err != nil {
		log.Printf("出栈失败: %v", err)
		return
	}
	fmt.Printf("出栈的操作: %s\n", operation)

	fmt.Println("✓ 列表操作完成")
}

// 示例7: 集合操作
func example7() {
	fmt.Println("\n=== 示例7: 集合操作 ===")

	cfg := config.DefaultConfig()
	cacheClient, err := cache.New(cfg)
	if err != nil {
		log.Fatalf("创建缓存客户端失败: %v", err)
	}
	defer cacheClient.Close()

	ctx := context.Background()

	// 基础集合操作
	setKey := "user_tags"
	_, err = cacheClient.SAdd(ctx, setKey, "技术", "编程", "Go语言", "Redis")
	if err != nil {
		log.Printf("添加集合成员失败: %v", err)
		return
	}

	members, err := cacheClient.SMembers(ctx, setKey)
	if err != nil {
		log.Printf("获取集合成员失败: %v", err)
		return
	}
	fmt.Printf("用户标签: %v\n", members)

	// 集合运算
	set1 := "skills:backend"
	set2 := "skills:frontend"
	cacheClient.SAdd(ctx, set1, "Go", "Python", "Java", "Redis")
	cacheClient.SAdd(ctx, set2, "JavaScript", "React", "Vue", "Redis")

	// 交集
	intersection, err := cacheClient.SInter(ctx, set1, set2)
	if err != nil {
		log.Printf("计算交集失败: %v", err)
		return
	}
	fmt.Printf("技能交集: %v\n", intersection)

	// 集合过滤器
	filter := cacheClient.NewSetFilter("active_users")
	filter.Add(ctx, "user1", "user2", "user3")
	exists, err := filter.Contains(ctx, "user2")
	if err != nil {
		log.Printf("检查成员存在失败: %v", err)
		return
	}
	fmt.Printf("用户user2是否活跃: %v\n", exists)

	fmt.Println("✓ 集合操作完成")
}

// 示例8: 有序集合操作
func example8() {
	fmt.Println("\n=== 示例8: 有序集合操作 ===")

	cfg := config.DefaultConfig()
	cacheClient, err := cache.New(cfg)
	if err != nil {
		log.Fatalf("创建缓存客户端失败: %v", err)
	}
	defer cacheClient.Close()

	ctx := context.Background()

	// 基础有序集合操作
	leaderboardKey := "game_scores"
	members := []redis.Z{
		{Score: 1000, Member: "玩家A"},
		{Score: 950, Member: "玩家B"},
		{Score: 900, Member: "玩家C"},
		{Score: 850, Member: "玩家D"},
	}
	_, err = cacheClient.ZAdd(ctx, leaderboardKey, members...)
	if err != nil {
		log.Printf("添加有序集合成员失败: %v", err)
		return
	}

	// 获取排行榜前3名
	top3, err := cacheClient.ZRevRange(ctx, leaderboardKey, 0, 2)
	if err != nil {
		log.Printf("获取排行榜失败: %v", err)
		return
	}
	fmt.Printf("排行榜前3名: %v\n", top3)

	// 排行榜功能
	leaderboard := cacheClient.NewLeaderboard("weekly_scores")
	leaderboard.AddScore(ctx, "玩家X", 1200)
	leaderboard.AddScore(ctx, "玩家Y", 1100)
	leaderboard.AddScore(ctx, "玩家Z", 1050)

	// 获取玩家排名
	rank, err := leaderboard.GetRank(ctx, "玩家X")
	if err != nil {
		log.Printf("获取排名失败: %v", err)
		return
	}
	fmt.Printf("玩家X的排名: %d\n", rank+1) // Redis排名从0开始，显示时+1

	// 获取前5名
	topPlayers, err := leaderboard.GetTopN(ctx, 5)
	if err != nil {
		log.Printf("获取前5名失败: %v", err)
		return
	}
	fmt.Printf("前5名玩家: %v\n", topPlayers)

	fmt.Println("✓ 有序集合操作完成")
}

// 示例9: Lua脚本操作
func example9() {
	fmt.Println("\n=== 示例9: Lua脚本操作 ===")

	cfg := config.DefaultConfig()
	cacheClient, err := cache.New(cfg)
	if err != nil {
		log.Fatalf("创建缓存客户端失败: %v", err)
	}
	defer cacheClient.Close()

	ctx := context.Background()

	// 直接执行Lua脚本
	script := `
		local key = KEYS[1]
		local value = ARGV[1]
		local expire = tonumber(ARGV[2])
		
		redis.call('SET', key, value)
		redis.call('EXPIRE', key, expire)
		return redis.call('GET', key)
	`
	result, err := cacheClient.Eval(ctx, script, []string{"test_key"}, "test_value", 60)
	if err != nil {
		log.Printf("执行Lua脚本失败: %v", err)
		return
	}
	fmt.Printf("Lua脚本执行结果: %v\n", result)

	// 注册和执行脚本
	err = cacheClient.RegisterScript("set_with_expire", script)
	if err != nil {
		log.Printf("注册脚本失败: %v", err)
		return
	}

	result2, err := cacheClient.ExecuteScript(ctx, "set_with_expire", []string{"test_key2"}, "test_value2", 120)
	if err != nil {
		log.Printf("执行注册脚本失败: %v", err)
		return
	}
	fmt.Printf("注册脚本执行结果: %v\n", result2)

	// 分布式锁
	lock := cacheClient.NewDistributedLock("resource_lock", "unique_value", 30*time.Second)
	locked, err := lock.TryLock(ctx)
	if err != nil {
		log.Printf("尝试获取锁失败: %v", err)
		return
	}
	if locked {
		fmt.Println("✓ 成功获取分布式锁")
		// 模拟业务处理
		time.Sleep(1 * time.Second)
		// 释放锁
		unlocked, err := lock.Unlock(ctx)
		if err != nil {
			log.Printf("释放锁失败: %v", err)
		} else if unlocked {
			fmt.Println("✓ 成功释放分布式锁")
		}
	} else {
		fmt.Println("✗ 获取分布式锁失败")
	}

	// 限流器
	rateLimiter := cacheClient.NewRateLimiter("api_rate_limit", 60*time.Second, 100)
	allowed, remaining, err := rateLimiter.Allow(ctx)
	if err != nil {
		log.Printf("检查限流失败: %v", err)
		return
	}
	if allowed {
		fmt.Printf("✓ 请求通过，剩余配额: %d\n", remaining)
	} else {
		fmt.Println("✗ 请求被限流")
	}

	fmt.Println("✓ Lua脚本操作完成")
}

// 示例10: 批量操作
func example10() {
	fmt.Println("\n=== 示例10: 批量操作 ===")

	cfg := config.DefaultConfig()
	cacheClient, err := cache.New(cfg)
	if err != nil {
		log.Fatalf("创建缓存客户端失败: %v", err)
	}
	defer cacheClient.Close()

	ctx := context.Background()

	// 批量操作器
	batchOp := cacheClient.NewBatchOperator()

	// 批量设置带过期时间的键值对
	kvs := map[string]interface{}{
		"session:1001": "user_data_1",
		"session:1002": "user_data_2",
		"session:1003": "user_data_3",
	}
	count, err := batchOp.BatchSetWithExpire(ctx, kvs, 30*time.Minute)
	if err != nil {
		log.Printf("批量设置失败: %v", err)
		return
	}
	fmt.Printf("批量设置了 %d 个键值对\n", count)

	// 多个哈希表的字段获取
	hashKeys := []string{"user:1001", "user:1002", "user:1003"}
	results, err := batchOp.MultiHGet(ctx, hashKeys, "name")
	if err != nil {
		log.Printf("批量获取哈希字段失败: %v", err)
		return
	}
	fmt.Printf("批量获取的用户名: %v\n", results)

	// 各种批量操作结构
	// 哈希表批量操作
	hashBatch := cacheClient.NewHashBatch("batch_hash")
	hashBatch.Set("field1", "value1").Set("field2", "value2").Set("field3", "value3")
	hashBatch.Execute(ctx)

	// 列表批量操作
	listBatch := cacheClient.NewListBatch("batch_list", true) // 从左侧操作
	listBatch.Add("item1", "item2", "item3")
	listBatch.Execute(ctx)

	// 集合批量操作
	setBatch := cacheClient.NewSetBatch("batch_set", "add")
	setBatch.Add("member1", "member2", "member3")
	setBatch.Execute(ctx)

	// 有序集合批量操作
	zsetBatch := cacheClient.NewZSetBatch("batch_zset")
	zsetBatch.Add(100, "player1").Add(200, "player2").Add(300, "player3")
	zsetBatch.Execute(ctx)

	fmt.Println("✓ 批量操作完成")
}

// 示例11: 高级功能
func example11() {
	fmt.Println("\n=== 示例11: 高级功能 ===")

	cfg := config.DefaultConfig()
	cacheClient, err := cache.New(cfg)
	if err != nil {
		log.Fatalf("创建缓存客户端失败: %v", err)
	}
	defer cacheClient.Close()

	ctx := context.Background()

	// 原子计数器
	atomicCounter := cacheClient.NewAtomicCounter("daily_visits", 24*time.Hour)
	count, err := atomicCounter.IncrBy(ctx, 5)
	if err != nil {
		log.Printf("原子计数器操作失败: %v", err)
		return
	}
	fmt.Printf("今日访问量: %d\n", count)

	// Lua脚本管理器
	scriptManager := cacheClient.NewLuaScriptManager()

	// 注册自定义脚本
	customScript := `
		local key = KEYS[1]
		local field = ARGV[1]
		local increment = tonumber(ARGV[2])
		
		local current = redis.call('HGET', key, field)
		if current == false then
			current = 0
		else
			current = tonumber(current)
		end
		
		local new_value = current + increment
		redis.call('HSET', key, field, new_value)
		return new_value
	`
	err = scriptManager.RegisterScript("hash_incr", customScript)
	if err != nil {
		log.Printf("注册脚本失败: %v", err)
		return
	}

	// 执行自定义脚本
	result, err := scriptManager.ExecuteScript(ctx, "hash_incr", []string{"stats:counters"}, "page_views", 10)
	if err != nil {
		log.Printf("执行自定义脚本失败: %v", err)
		return
	}
	fmt.Printf("页面浏览量: %v\n", result)

	// 列出所有注册的脚本
	scripts := scriptManager.ListScripts()
	fmt.Printf("已注册的脚本: %v\n", scripts)

	// 集合运算器
	setOp := cacheClient.NewSetOperator()
	// 这里可以进行复杂的集合运算操作
	_ = setOp // 避免未使用变量警告

	fmt.Println("✓ 高级功能演示完成")
}

// 示例12: 监控和统计
func example12() {
	fmt.Println("\n=== 示例12: 监控和统计 ===")

	cfg := config.DefaultConfig()
	cacheClient, err := cache.New(cfg)
	if err != nil {
		log.Fatalf("创建缓存客户端失败: %v", err)
	}
	defer cacheClient.Close()

	ctx := context.Background()

	// 执行一些操作以生成统计数据
	cacheClient.Set(ctx, "test1", "value1", nil)
	cacheClient.Set(ctx, "test2", "value2", nil)
	cacheClient.Get(ctx, "test1")
	cacheClient.Get(ctx, "test2")
	cacheClient.Get(ctx, "nonexistent")

	// 获取统计信息
	stats := cacheClient.GetStats()
	if stats != nil {
		fmt.Printf("操作统计:\n")
		fmt.Printf("  总操作数: %d\n", stats.TotalOperations)
		fmt.Printf("  成功操作数: %d\n", stats.SuccessOperations)
		fmt.Printf("  失败操作数: %d\n", stats.FailedOperations)
		fmt.Printf("  平均响应时间: %v\n", stats.AverageDuration)
		// 获取最大和最小响应时间需要从操作统计中计算
		var maxTime, minTime time.Duration
		for _, opStat := range stats.OperationStats {
			if maxTime == 0 || opStat.MaxTime > maxTime {
				maxTime = opStat.MaxTime
			}
			if minTime == 0 || (opStat.MinTime > 0 && opStat.MinTime < minTime) {
				minTime = opStat.MinTime
			}
		}
		fmt.Printf("  最大响应时间: %v\n", maxTime)
		fmt.Printf("  最小响应时间: %v\n", minTime)
	}

	// 获取慢查询记录
	slowQueries := cacheClient.GetSlowQueries(10)
	if len(slowQueries) > 0 {
		fmt.Printf("\n慢查询记录 (%d条):\n", len(slowQueries))
		for i, query := range slowQueries {
			if i >= 5 { // 只显示前5条
				break
			}
			fmt.Printf("  %d. 操作: %s, 键: %s, 耗时: %v\n",
				i+1, query.Operation, query.Key, query.Duration)
		}
	} else {
		fmt.Println("\n暂无慢查询记录")
	}

	// 配置管理
	fmt.Println("\n配置管理:")
	cfg = cacheClient.GetConfig()
	fmt.Printf("  当前连接池大小: %d\n", cfg.PoolSize)
	fmt.Printf("  默认过期时间: %v\n", cfg.DefaultExpiration)
	fmt.Printf("  慢查询阈值: %v\n", cfg.Monitoring.SlowLogThreshold)

	// 动态调整配置
	cacheClient.UpdatePoolConfig(20, 5, 15)
	cacheClient.UpdateTimeouts(10*time.Second, 5*time.Second, 5*time.Second)
	cacheClient.SetSlowLogThreshold(50 * time.Millisecond)

	fmt.Println("✓ 配置已更新")

	// 重置统计信息
	cacheClient.ResetStats()
	cacheClient.ClearSlowQueries()
	fmt.Println("✓ 统计信息已重置")

	fmt.Println("✓ 监控和统计演示完成")
}
