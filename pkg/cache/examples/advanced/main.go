package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	cache "github.com/tokmz/basic/pkg/cache"
)

func main() {
	fmt.Println("=== 缓存包高级特性示例 ===\n")

	// 示例1: 多级缓存使用
	multiLevelExample()

	// 示例2: 高并发场景测试
	concurrentExample()

	// 示例3: TTL和过期处理
	ttlExample()

	// 示例4: 缓存淘汰策略
	evictionExample()

	// 示例5: 自定义监控收集器
	monitoringExample()

	// 示例6: 批量操作(Redis)
	// batchOperationExample() // 需要Redis连接，注释掉避免错误
}

func multiLevelExample() {
	fmt.Println("1. 多级缓存使用:")

	config := cache.DefaultConfig()
	config.Type = cache.TypeMultiLevel
	config.MultiLevel.EnableL1 = true  // 启用L1缓存(内存)
	config.MultiLevel.EnableL2 = false // 禁用L2缓存(Redis)以避免Redis依赖
	config.MultiLevel.L1TTL = 30 * time.Second
	config.MultiLevel.SyncStrategy = cache.SyncWriteThrough

	// 调整内存缓存配置
	config.Memory.MaxSize = 100
	config.Memory.EvictPolicy = cache.EvictLRU
	config.Memory.EnableStats = true

	mlCache, err := cache.CreateWithConfig(config)
	if err != nil {
		log.Fatalf("创建多级缓存失败: %v", err)
	}
	defer mlCache.Close()

	ctx := context.Background()

	// 设置一些数据
	testData := map[string]string{
		"user:1":    "张三",
		"user:2":    "李四",
		"user:3":    "王五",
		"product:1": "笔记本电脑",
		"product:2": "无线鼠标",
	}

	fmt.Println("  设置缓存数据...")
	for key, value := range testData {
		err := mlCache.Set(ctx, key, value, time.Hour)
		if err != nil {
			log.Printf("设置 %s 失败: %v", key, err)
		}
	}

	// 模拟查询操作
	fmt.Println("  执行查询操作...")
	for i := 0; i < 3; i++ {
		for key := range testData {
			value, err := mlCache.Get(ctx, key)
			if err != nil {
				log.Printf("获取 %s 失败: %v", key, err)
			} else {
				fmt.Printf("    获取 %s: %s\n", key, value)
			}
		}
	}

	stats := mlCache.Stats()
	fmt.Printf("  多级缓存统计: 命中率=%.2f%%, 总请求=%d\n",
		stats.HitRate*100, stats.TotalRequests)

	fmt.Println()
}

func concurrentExample() {
	fmt.Println("2. 高并发场景测试:")

	concurrentCache, err := cache.Create(cache.TypeMemoryLRU)
	if err != nil {
		log.Fatalf("创建缓存失败: %v", err)
	}
	defer concurrentCache.Close()

	ctx := context.Background()
	const goroutines = 10
	const operationsPerGoroutine = 100

	var wg sync.WaitGroup
	var totalOperations int64

	start := time.Now()

	fmt.Printf("  启动 %d 个协程，每个执行 %d 次操作...\n", goroutines, operationsPerGoroutine)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("goroutine:%d:key:%d", goroutineID, j)
				value := fmt.Sprintf("value-%d-%d", goroutineID, j)

				// 混合操作：设置、获取、删除
				switch j % 4 {
				case 0, 1: // 50% 设置操作
					concurrentCache.Set(ctx, key, value, time.Minute)
				case 2: // 25% 获取操作
					concurrentCache.Get(ctx, key)
				case 3: // 25% 删除操作
					concurrentCache.Delete(ctx, key)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	totalOperations = int64(goroutines * operationsPerGoroutine)
	stats := concurrentCache.Stats()

	fmt.Printf("  并发测试完成:\n")
	fmt.Printf("    执行时间: %v\n", duration)
	fmt.Printf("    总操作数: %d\n", totalOperations)
	fmt.Printf("    操作吞吐量: %.2f ops/sec\n", float64(totalOperations)/duration.Seconds())
	fmt.Printf("    缓存统计: 命中率=%.2f%%, 总请求=%d, 当前大小=%d\n",
		stats.HitRate*100, stats.TotalRequests, stats.Size)

	fmt.Println()
}

func ttlExample() {
	fmt.Println("3. TTL和过期处理:")

	config := cache.DefaultConfig()
	config.Type = cache.TypeMemoryLRU
	config.Memory.CleanupInterval = 100 * time.Millisecond // 快速清理间隔
	ttlCache, err := cache.CreateWithConfig(config)
	if err != nil {
		log.Fatalf("创建缓存失败: %v", err)
	}
	defer ttlCache.Close()

	ctx := context.Background()

	// 设置不同TTL的缓存项
	ttlCache.Set(ctx, "short_ttl", "1秒后过期", 1*time.Second)
	ttlCache.Set(ctx, "medium_ttl", "3秒后过期", 3*time.Second)
	ttlCache.Set(ctx, "long_ttl", "10秒后过期", 10*time.Second)
	ttlCache.Set(ctx, "no_ttl", "永不过期", 0) // TTL=0表示永不过期

	fmt.Println("  设置了不同TTL的缓存项")

	// 监控过期过程
	for i := 0; i < 12; i++ {
		fmt.Printf("  第 %d 秒:\n", i)

		keys := []string{"short_ttl", "medium_ttl", "long_ttl", "no_ttl"}
		for _, key := range keys {
			value, err := ttlCache.Get(ctx, key)
			if err == cache.ErrKeyNotFound {
				fmt.Printf("    %s: 已过期\n", key)
			} else if err != nil {
				fmt.Printf("    %s: 错误 - %v\n", key, err)
			} else {
				fmt.Printf("    %s: %s\n", key, value)
			}
		}

		stats := ttlCache.Stats()
		fmt.Printf("    当前缓存大小: %d\n", stats.Size)

		time.Sleep(1 * time.Second)
	}

	fmt.Println()
}

func evictionExample() {
	fmt.Println("4. 缓存淘汰策略:")

	// 创建小容量缓存测试淘汰
	config := cache.DefaultConfig()
	config.Type = cache.TypeMemoryLRU
	config.Memory.MaxSize = 5 // 限制为5个项目
	config.Memory.EvictPolicy = cache.EvictLRU
	config.Memory.EnableStats = true

	evictCache, err := cache.CreateWithConfig(config)
	if err != nil {
		log.Fatalf("创建缓存失败: %v", err)
	}
	defer evictCache.Close()

	ctx := context.Background()

	fmt.Println("  测试LRU淘汰策略 (最大容量: 5):")

	// 填充缓存到容量上限
	for i := 1; i <= 5; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d", i)
		evictCache.Set(ctx, key, value, time.Hour)
		fmt.Printf("    添加 %s\n", key)
	}

	stats := evictCache.Stats()
	fmt.Printf("    当前缓存大小: %d/%d\n", stats.Size, config.Memory.MaxSize)

	// 访问某些键改变LRU顺序
	evictCache.Get(ctx, "key1") // key1变为最新
	evictCache.Get(ctx, "key3") // key3变为最新

	// 添加新项目，应该淘汰最久未使用的
	fmt.Println("  添加新项目，触发淘汰:")
	for i := 6; i <= 8; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d", i)
		evictCache.Set(ctx, key, value, time.Hour)
		fmt.Printf("    添加 %s\n", key)

		// 检查哪些键被淘汰了
		fmt.Printf("    当前存在的键: ")
		keys, _ := evictCache.Keys(ctx, "*")
		fmt.Printf("%v\n", keys)
	}

	stats = evictCache.Stats()
	fmt.Printf("  最终缓存大小: %d, 总设置次数: %d\n", stats.Size, stats.Sets)

	fmt.Println()
}

func monitoringExample() {
	fmt.Println("5. 缓存统计监控:")

	// 创建缓存
	monitorCache, err := cache.Create(cache.TypeMemoryLRU)
	if err != nil {
		log.Fatalf("创建缓存失败: %v", err)
	}
	defer monitorCache.Close()

	fmt.Println("  执行缓存操作并监控统计信息...")

	// 执行一些操作
	go func() {
		cacheCtx := context.Background()
		for i := 0; i < 50; i++ {
			key := fmt.Sprintf("monitor_key_%d", i)
			value := fmt.Sprintf("monitor_value_%d", i)

			monitorCache.Set(cacheCtx, key, value, time.Minute)
			monitorCache.Get(cacheCtx, key)

			if i%10 == 0 {
				monitorCache.Delete(cacheCtx, key)
			}

			time.Sleep(50 * time.Millisecond)
		}
	}()

	// 定期获取统计信息
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for i := 0; i < 3; i++ {
		select {
		case <-ticker.C:
			stats := monitorCache.Stats()
			fmt.Printf("  第 %d 秒统计信息:\n", i+1)
			fmt.Printf("    命中: %d, 未命中: %d\n", stats.Hits, stats.Misses)
			totalRequests := stats.Hits + stats.Misses
			if totalRequests > 0 {
				hitRate := float64(stats.Hits) / float64(totalRequests) * 100
				fmt.Printf("    命中率: %.2f%%\n", hitRate)
			}
			fmt.Printf("    当前大小: %d, 设置次数: %d\n", stats.Size, stats.Sets)
		case <-ctx.Done():
			return
		}
	}

	fmt.Println()
}

func batchOperationExample() {
	fmt.Println("6. 批量操作(Redis):")

	// 注意: 这个示例需要Redis连接，在实际环境中使用
	config := cache.DefaultConfig()
	config.Type = cache.TypeRedisSingle
	config.Redis.Addrs = []string{"localhost:6379"}

	redisCache, err := cache.CreateWithConfig(config)
	if err != nil {
		log.Printf("无法连接Redis，跳过批量操作示例: %v", err)
		return
	}
	defer redisCache.Close()

	// 类型断言以访问Redis特有的方法
	if rc, ok := redisCache.(*cache.RedisCache); ok {
		ctx := context.Background()

		// 批量设置
		batchData := map[string]interface{}{
			"batch:user:1": "批量用户1",
			"batch:user:2": "批量用户2",
			"batch:user:3": "批量用户3",
			"batch:user:4": "批量用户4",
			"batch:user:5": "批量用户5",
		}

		fmt.Println("  执行批量设置...")
		err = rc.MSet(ctx, batchData, time.Hour)
		if err != nil {
			log.Printf("批量设置失败: %v", err)
			return
		}

		// 批量获取
		keys := make([]string, 0, len(batchData))
		for key := range batchData {
			keys = append(keys, key)
		}

		fmt.Println("  执行批量获取...")
		results, err := rc.MGet(ctx, keys...)
		if err != nil {
			log.Printf("批量获取失败: %v", err)
			return
		}

		fmt.Println("  批量获取结果:")
		for key, value := range results {
			fmt.Printf("    %s: %v\n", key, value)
		}

		stats := rc.Stats()
		fmt.Printf("  Redis缓存统计: 设置=%d, 命中=%d\n", stats.Sets, stats.Hits)
	}

	fmt.Println()
}
