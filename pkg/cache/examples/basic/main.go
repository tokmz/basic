package main

import (
	"context"
	"fmt"
	"log"
	"time"

	cache "github.com/tokmz/basic/pkg/cache"
)

func main() {
	fmt.Println("=== 缓存包基本使用示例 ===\n")

	// 示例1: 基本的内存缓存使用
	basicMemoryExample()

	// 示例2: 使用Builder模式创建缓存
	builderExample()

	// 示例3: 缓存管理器使用
	managerExample()

	// 示例4: 统计信息和监控
	statsExample()

	// 示例5: 错误处理
	errorHandlingExample()
}

func basicMemoryExample() {
	fmt.Println("1. 基本内存缓存使用:")

	// 创建内存缓存
	memCache, err := cache.Create(cache.TypeMemoryLRU)
	if err != nil {
		log.Fatalf("创建缓存失败: %v", err)
	}
	defer memCache.Close()

	ctx := context.Background()

	// 设置缓存
	err = memCache.Set(ctx, "user:123", "Alice Smith", time.Hour)
	if err != nil {
		log.Fatalf("设置缓存失败: %v", err)
	}

	err = memCache.Set(ctx, "user:456", "Bob Johnson", time.Hour)
	if err != nil {
		log.Fatalf("设置缓存失败: %v", err)
	}

	// 获取缓存
	value, err := memCache.Get(ctx, "user:123")
	if err != nil {
		log.Printf("获取缓存失败: %v", err)
	} else {
		fmt.Printf("  获取用户: %s\n", value)
	}

	// 检查键是否存在
	exists, _ := memCache.Exists(ctx, "user:456")
	fmt.Printf("  用户456是否存在: %t\n", exists)

	// 获取所有键
	keys, _ := memCache.Keys(ctx, "*")
	fmt.Printf("  所有键: %v\n", keys)

	// 删除缓存
	err = memCache.Delete(ctx, "user:456")
	if err != nil {
		log.Printf("删除缓存失败: %v", err)
	} else {
		fmt.Println("  已删除用户456")
	}

	fmt.Println()
}

func builderExample() {
	fmt.Println("2. 使用Builder模式创建缓存:")

	// 使用Builder模式创建自定义配置的缓存
	builderCache, err := cache.NewCacheBuilder().
		WithType(cache.TypeMemoryLRU).
		WithMemoryConfig(cache.MemoryConfig{
			MaxSize:         100,
			MaxMemory:       10 * 1024 * 1024, // 10MB
			EvictPolicy:     cache.EvictLRU,
			CleanupInterval: time.Minute,
			EnableStats:     true,
		}).
		WithMonitoring(cache.MonitoringConfig{
			EnableMetrics: true,
			EnableLogging: false, // 禁用日志以简化输出
			SlowThreshold: 50 * time.Millisecond,
		}).
		Build()

	if err != nil {
		log.Fatalf("构建缓存失败: %v", err)
	}
	defer builderCache.Close()

	ctx := context.Background()

	// 填充一些测试数据
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("item:%d", i)
		value := fmt.Sprintf("数据项 %d", i)
		builderCache.Set(ctx, key, value, 30*time.Minute)
	}

	fmt.Printf("  已创建自定义配置的缓存，填充了10个数据项\n")

	// 模拟一些获取操作
	for i := 0; i < 15; i++ {
		key := fmt.Sprintf("item:%d", i%10)
		builderCache.Get(ctx, key)
	}

	stats := builderCache.Stats()
	fmt.Printf("  缓存统计: 命中率=%.2f%%, 总请求=%d, 缓存大小=%d\n",
		stats.HitRate*100, stats.TotalRequests, stats.Size)

	fmt.Println()
}

func managerExample() {
	fmt.Println("3. 缓存管理器使用:")

	// 获取默认管理器
	manager := cache.GetDefaultManager()

	// 创建用户缓存
	userConfig := cache.DefaultConfig()
	userConfig.Type = cache.TypeMemoryLRU
	userConfig.Memory.MaxSize = 1000

	userCache, err := manager.CreateAndRegisterCache("users", userConfig)
	if err != nil {
		log.Fatalf("创建用户缓存失败: %v", err)
	}

	// 创建产品缓存
	productConfig := cache.DefaultConfig()
	productConfig.Type = cache.TypeMemoryLRU
	productConfig.Memory.MaxSize = 500

	productCache, err := manager.CreateAndRegisterCache("products", productConfig)
	if err != nil {
		log.Fatalf("创建产品缓存失败: %v", err)
	}

	ctx := context.Background()

	// 使用不同的缓存
	userCache.Set(ctx, "user:100", "管理员", time.Hour)
	userCache.Set(ctx, "user:101", "普通用户", time.Hour)

	productCache.Set(ctx, "product:200", "笔记本电脑", time.Hour)
	productCache.Set(ctx, "product:201", "无线鼠标", time.Hour)

	// 模拟一些查询操作
	userCache.Get(ctx, "user:100")
	userCache.Get(ctx, "user:999") // 不存在的键
	productCache.Get(ctx, "product:200")

	// 列出所有缓存
	cacheList := manager.ListCaches()
	fmt.Printf("  注册的缓存: %v\n", cacheList)

	// 获取所有缓存的统计信息
	allStats := manager.GetAllStats()
	for name, stats := range allStats {
		fmt.Printf("  缓存 %s: 命中率=%.2f%%, 总请求=%d\n",
			name, stats.HitRate*100, stats.TotalRequests)
	}

	fmt.Println()
}

func statsExample() {
	fmt.Println("4. 统计信息和监控:")

	statsCache, err := cache.Create(cache.TypeMemoryLRU)
	if err != nil {
		log.Fatalf("创建缓存失败: %v", err)
	}
	defer statsCache.Close()

	ctx := context.Background()

	// 执行各种操作以产生统计数据
	fmt.Println("  执行缓存操作...")

	// 设置操作
	for i := 0; i < 20; i++ {
		key := fmt.Sprintf("key:%d", i)
		value := fmt.Sprintf("值 %d", i)
		statsCache.Set(ctx, key, value, time.Hour)
	}

	// 获取操作 (一些命中，一些未命中)
	hitCount := 0
	missCount := 0
	for i := 0; i < 30; i++ {
		key := fmt.Sprintf("key:%d", i) // 包含一些不存在的键
		_, err := statsCache.Get(ctx, key)
		if err == nil {
			hitCount++
		} else {
			missCount++
		}
	}

	// 删除操作
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key:%d", i)
		statsCache.Delete(ctx, key)
	}

	// 获取并显示统计信息
	stats := statsCache.Stats()
	fmt.Printf("  统计信息:\n")
	fmt.Printf("    总请求数: %d\n", stats.TotalRequests)
	fmt.Printf("    命中次数: %d\n", stats.Hits)
	fmt.Printf("    未命中次数: %d\n", stats.Misses)
	fmt.Printf("    设置次数: %d\n", stats.Sets)
	fmt.Printf("    删除次数: %d\n", stats.Deletes)
	fmt.Printf("    错误次数: %d\n", stats.Errors)
	fmt.Printf("    命中率: %.2f%%\n", stats.HitRate*100)
	fmt.Printf("    当前缓存大小: %d\n", stats.Size)

	fmt.Println()
}

func errorHandlingExample() {
	fmt.Println("5. 错误处理示例:")

	errorCache, err := cache.Create(cache.TypeMemoryLRU)
	if err != nil {
		log.Fatalf("创建缓存失败: %v", err)
	}
	defer errorCache.Close()

	ctx := context.Background()

	// 尝试获取不存在的键
	_, err = errorCache.Get(ctx, "不存在的键")
	if err != nil {
		switch err {
		case cache.ErrKeyNotFound:
			fmt.Println("  处理键不存在的情况")
		case cache.ErrCacheClosed:
			fmt.Println("  处理缓存已关闭的情况")
		default:
			fmt.Printf("  处理其他错误: %v\n", err)
		}
	}

	// 关闭缓存后尝试操作
	errorCache.Close()
	_, err = errorCache.Get(ctx, "any_key")
	if err == cache.ErrCacheClosed {
		fmt.Println("  缓存已关闭，操作被拒绝")
	}

	// 尝试删除不存在的键
	cache2, _ := cache.Create(cache.TypeMemoryLRU)
	defer cache2.Close()

	err = cache2.Delete(ctx, "不存在的键")
	if err == cache.ErrKeyNotFound {
		fmt.Println("  尝试删除不存在的键")
	}

	fmt.Println()
}
