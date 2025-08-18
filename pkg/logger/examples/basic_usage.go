package main

import (
	"context"
	"errors"
	"time"

	"basic/pkg/logger"
)

func main() {
	// 示例1: 使用预设配置
	examplePresetConfigs()

	// 示例2: 自定义配置
	exampleCustomConfig()

	// 示例3: 零内存分配日志
	exampleZeroAllocLogging()

	// 示例4: 异步缓冲日志
	exampleAsyncBuffering()

	// 示例5: 性能监控
	examplePerformanceMonitoring()

	// 示例6: 中间件使用
	exampleMiddleware()

	// 示例7: 全局日志器
	exampleGlobalLogger()

	// 示例8: 批量日志
	exampleBatchLogging()
}

// 示例1: 使用预设配置
func examplePresetConfigs() {
	println("=== 预设配置示例 ===")

	// 开发环境配置
	logger.InitDevelopment()
	logger.Info("开发环境日志", logger.String("env", "development"))

	// 生产环境配置
	logger.InitProduction()
	logger.Info("生产环境日志", logger.String("env", "production"))

	// 测试环境配置
	logger.InitTesting()
	logger.Info("测试环境日志", logger.String("env", "testing"))

	logger.Sync()
}

// 示例2: 自定义配置
func exampleCustomConfig() {
	println("\n=== 自定义配置示例 ===")

	// 创建自定义配置
	config := &logger.Config{
		Level:            logger.InfoLevel,
		Format:           logger.JSONFormat,
		Environment:      logger.Production,
		EnableCaller:     true,
		EnableStacktrace: true,
		EnableColor:      false,
		Output: logger.OutputConfig{
			Mode:     logger.MixedMode, // 同时输出到控制台和文件
			Filename: "app.log",
			Path:     "./logs",
		},
		Rotation: logger.RotationConfig{
			MaxSize:    100, // 100MB
			MaxAge:     30,  // 30天
			MaxBackups: 10,  // 10个备份
			Compress:   true,
		},
		Buffer: logger.BufferConfig{
			Size:        8192,
			FlushTime:   100 * time.Millisecond,
			AsyncBuffer: true,
		},
	}

	// 创建日志器
	customLogger, err := logger.New(config)
	if err != nil {
		panic(err)
	}
	defer customLogger.Close()

	// 使用自定义日志器
	customLogger.Info("自定义配置日志",
		logger.String("service", "example"),
		logger.Int("version", 1),
		logger.Bool("production", true),
	)

	customLogger.Warn("警告消息",
		logger.String("component", "database"),
		logger.Duration("latency", 150*time.Millisecond),
	)

	customLogger.Error("错误消息",
		logger.Error(errors.New("数据库连接失败")),
		logger.String("operation", "connect"),
	)

	customLogger.Sync()
}

// 示例3: 零内存分配日志
func exampleZeroAllocLogging() {
	println("\n=== 零内存分配日志示例 ===")

	// 初始化日志器
	logger.InitDevelopment()
	baseLogger := logger.GetGlobalLogger()

	// 创建零分配日志器
	zeroLogger := logger.NewZeroAllocLogger(baseLogger)

	// 链式调用添加字段
	zeroLogger.AddString("service", "api").
		AddInt("user_id", 12345).
		AddBool("authenticated", true).
		AddTime("timestamp", time.Now()).
		AddDuration("response_time", 50*time.Millisecond).
		Info("API请求处理完成")

	// 处理错误
	errorLogger := logger.NewZeroAllocLogger(baseLogger)
	errorLogger.AddString("operation", "database_query").
		AddError(errors.New("连接超时")).
		AddInt("retry_count", 3).
		Error("数据库操作失败")

	logger.Sync()
}

// 示例4: 异步缓冲日志
func exampleAsyncBuffering() {
	println("\n=== 异步缓冲日志示例 ===")

	// 配置异步缓冲
	config := &logger.Config{
		Level:  logger.InfoLevel,
		Format: logger.JSONFormat,
		Output: logger.OutputConfig{
			Mode:     logger.FileMode,
			Filename: "async.log",
			Path:     "./logs",
		},
		Buffer: logger.BufferConfig{
			Size:        4096,                  // 4KB缓冲
			FlushTime:   50 * time.Millisecond, // 50ms刷新
			AsyncBuffer: true,                  // 启用异步
		},
	}

	asyncLogger, err := logger.New(config)
	if err != nil {
		panic(err)
	}
	defer asyncLogger.Close()

	// 高频日志写入
	start := time.Now()
	for i := 0; i < 1000; i++ {
		asyncLogger.Info("高频日志消息",
			logger.Int("sequence", i),
			logger.String("type", "benchmark"),
			logger.Time("created_at", time.Now()),
		)
	}
	duration := time.Since(start)

	asyncLogger.Info("异步日志性能测试完成",
		logger.Int("total_logs", 1000),
		logger.Duration("total_time", duration),
		logger.Float64("logs_per_second", 1000.0/duration.Seconds()),
	)

	asyncLogger.Sync()
}

// 示例5: 性能监控
func examplePerformanceMonitoring() {
	println("\n=== 性能监控示例 ===")

	logger.InitDevelopment()
	baseLogger := logger.GetGlobalLogger()

	// 创建性能监控器
	monitor := logger.NewPerformanceMonitor(baseLogger)

	// 模拟一些日志操作并记录性能
	for i := 0; i < 100; i++ {
		start := time.Now()
		baseLogger.Info("性能测试日志",
			logger.Int("iteration", i),
			logger.String("operation", "test"),
		)
		monitor.RecordLog(time.Since(start))
	}

	// 获取并记录统计信息
	stats := monitor.GetStats()
	baseLogger.Info("性能统计",
		logger.Int64("total_logs", stats.TotalLogs),
		logger.Float64("log_rate", stats.LogRate),
		logger.Duration("avg_latency", stats.AvgLatency),
		logger.Int64("memory_usage", stats.MemoryUsage),
		logger.Int("goroutines", stats.Goroutines),
	)

	// 使用监控器记录统计日志
	monitor.LogStats()

	logger.Sync()
}

// 示例6: 中间件使用
func exampleMiddleware() {
	println("\n=== 中间件使用示例 ===")

	logger.InitDevelopment()
	baseLogger := logger.GetGlobalLogger()

	// 创建请求日志记录器
	requestLogger := logger.NewRequestLogger(baseLogger, logger.DefaultRequestLoggerConfig())

	// 模拟HTTP请求日志
	ctx := context.Background()
	ctx = logger.WithRequestID(ctx, "req-12345")
	ctx = logger.WithTraceID(ctx, "trace-67890")

	requestLogger.LogRequest(ctx, "GET", "/api/users", "Mozilla/5.0", "192.168.1.100", 200, 45*time.Millisecond, 1024, 2048)
	requestLogger.LogRequest(ctx, "POST", "/api/orders", "curl/7.68.0", "10.0.0.1", 201, 120*time.Millisecond, 512, 256)
	requestLogger.LogRequest(ctx, "GET", "/api/slow", "Chrome/91.0", "172.16.0.1", 200, 2*time.Second, 2048, 4096) // 慢请求

	// 创建错误处理器
	errorHandler := logger.NewErrorHandler(baseLogger, logger.DefaultErrorHandlerConfig())

	// 处理不同类型的错误
	errorHandler.HandleError(ctx, errors.New("数据库连接失败"), logger.ErrorLevel, "数据库错误")
	errorHandler.HandleError(ctx, errors.New("参数验证失败"), logger.WarnLevel, "请求参数错误")

	// 创建性能追踪器
	perfTracker := logger.NewPerformanceTracker(baseLogger, logger.DefaultPerformanceTrackerConfig())

	// 追踪操作性能
	err := perfTracker.TrackOperation(ctx, "database_query", func() error {
		time.Sleep(30 * time.Millisecond) // 模拟数据库查询
		return nil
	})
	if err != nil {
		baseLogger.Error("操作失败", logger.Error(err))
	}

	// 追踪慢操作
	err = perfTracker.TrackOperation(ctx, "slow_operation", func() error {
		time.Sleep(200 * time.Millisecond) // 模拟慢操作
		return nil
	})
	if err != nil {
		baseLogger.Error("慢操作失败", logger.Error(err))
	}

	logger.Sync()
}

// 示例7: 全局日志器
func exampleGlobalLogger() {
	println("\n=== 全局日志器示例 ===")

	// 初始化全局日志器
	logger.InitProduction()

	// 使用结构化日志
	logger.Debug("调试信息", logger.String("module", "auth"))
	logger.Info("用户登录", logger.String("username", "john"), logger.Int("user_id", 123))
	logger.Warn("磁盘空间不足", logger.Float64("usage_percent", 85.5))
	logger.ErrorLog("支付失败", logger.String("order_id", "ORD-001"), logger.Error(errors.New("余额不足")))

	// 使用格式化日志
	logger.Debugf("处理请求: %s %s", "GET", "/api/users")
	logger.Infof("服务启动完成，监听端口: %d", 8080)
	logger.Warnf("连接池使用率: %.2f%%", 78.5)
	logger.Errorf("文件读取失败: %s", "config.yaml")

	// 使用键值对日志
	logger.Debugw("缓存操作", "action", "set", "key", "user:123", "ttl", 3600)
	logger.Infow("订单创建", "order_id", "ORD-002", "amount", 99.99, "currency", "USD")
	logger.Warnw("API限流", "client_ip", "192.168.1.100", "requests_per_minute", 1000)
	logger.Errorw("数据同步失败", "table", "users", "error", "connection timeout")

	// 动态调整日志级别
	logger.SetLevel(logger.WarnLevel)
	logger.Debug("这条调试日志不会输出") // 不会输出
	logger.Warn("这条警告日志会输出")   // 会输出

	logger.Sync()
}

// 示例8: 批量日志
func exampleBatchLogging() {
	println("\n=== 批量日志示例 ===")

	logger.InitDevelopment()
	baseLogger := logger.GetGlobalLogger()

	// 创建批量日志器，批次大小为5
	batchLogger := logger.NewBatchLogger(baseLogger, 5)

	// 添加日志到批次
	for i := 0; i < 12; i++ {
		batchLogger.Add(logger.InfoLevel, "批量日志消息",
			logger.Int("batch_id", i),
			logger.String("type", "batch_test"),
			logger.Time("timestamp", time.Now()),
		)
		// 当达到批次大小(5)时，会自动刷新
	}

	// 手动刷新剩余的日志
	batchLogger.Flush()

	baseLogger.Info("批量日志示例完成")
	logger.Sync()
}

// 辅助函数
func println(msg string) {
	// 使用标准输出，避免与日志输出混淆
	logger.GetGlobalLogger().Info(msg)
}
