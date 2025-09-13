package logger

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// Example_basicUsage 基本使用示例
func Example_basicUsage() {
	// 使用默认配置创建日志器
	logger, err := New(DefaultConfig())
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	// 基本日志记录
	logger.Info("应用启动")
	logger.Infof("应用启动在端口: %d", 8080)

	// 带结构化字段的日志
	logger.Info("用户登录",
		zap.String("user_id", "12345"),
		zap.String("username", "john_doe"),
		zap.Duration("login_time", 100),
	)

	// 使用字段映射
	logger.WithFields(Fields{
		"module": "auth",
		"action": "login",
		"result": "success",
	}).Info("用户认证成功")


}

// Example_developmentConfig 开发环境配置示例
func Example_developmentConfig() {
	// 使用开发环境配置
	logger, err := New(DevelopmentConfig())
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	logger.Debug("这是调试信息")
	logger.Info("这是信息日志")
	logger.Warn("这是警告日志")
	logger.Error("这是错误日志")


}

// Example_productionConfig 生产环境配置示例
func Example_productionConfig() {
	// 使用生产环境配置
	config := ProductionConfig("my-service", "production")
	logger, err := New(config)
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	logger.Info("服务启动",
		zap.String("version", "1.0.0"),
		zap.Int("port", 8080),
	)

}


// Example_fileLogging 文件日志示例
func Example_fileLogging() {
	// 配置文件输出
	config := DefaultConfig()
	config.Console.Enabled = false
	config.File.Enabled = true
	config.File.Filename = "./logs/app.log"
	config.File.Rotation.MaxSize = 100   // 100MB
	config.File.Rotation.MaxAge = 30     // 30天
	config.File.Rotation.MaxBackups = 5  // 5个备份
	config.File.Rotation.Compress = true // 压缩旧文件

	logger, err := New(config)
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	logger.Info("日志将写入文件")


}

// Example_contextLogging 上下文日志示例
func Example_contextLogging() {
	logger, err := New(DevelopmentConfig())
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	// 创建带有追踪信息的上下文
	ctx := context.Background()
	ctx = WithTraceID(ctx, "trace-123")
	ctx = WithRequestID(ctx, "req-456")
	ctx = WithUserID(ctx, "user-789")

	// 使用上下文日志
	logger.WithContext(ctx).Info("处理用户请求")


}

// Example_globalLogger 全局日志器示例
func Example_globalLogger() {
	// 初始化全局日志器
	err := InitGlobal(DevelopmentConfig())
	if err != nil {
		panic(err)
	}

	// 使用全局函数记录日志
	Info("这是全局info日志")
	Infof("格式化日志: %s", "value")

	// 带字段的全局日志
	WithFields(Fields{
		"component": "main",
		"action":    "startup",
	}).Info("应用启动")

	// 带上下文的全局日志
	ctx := WithTraceID(context.Background(), "global-trace-123")
	WithContext(ctx).Info("全局上下文日志")


}

// Example_hooks 钩子示例
func Example_hooks() {
	// 创建邮件告警钩子
	emailHook := NewEmailHook("smtp.example.com", 587, "app@example.com", "password", []string{"admin@example.com"})

	// 创建Webhook钩子
	webhookHook := NewWebhookHook("https://webhook.example.com/alert")

	// 创建带钩子的日志器
	logger, err := NewWithHooks(DevelopmentConfig(), emailHook, webhookHook)
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	// 这些日志会触发钩子
	logger.Error("数据库连接失败")
	logger.Error("外部API调用超时")


}

// Example_customConfig 自定义配置示例
func Example_customConfig() {
	config := &Config{
		Level:        WarnLevel, // 只记录警告及以上级别
		Format:       JSONFormat,
		EnableCaller: true,
		EnableStack:  true,
		TimeFormat:   "2006-01-02 15:04:05",
		Console: ConsoleConfig{
			Enabled:    true,
			ColoredOut: false,
		},
		File: FileConfig{
			Enabled:  true,
			Filename: "./logs/custom.log",
			Rotation: RotationConfig{
				MaxSize:    50, // 50MB
				MaxAge:     7,  // 7天
				MaxBackups: 3,  // 3个备份
				Compress:   true,
			},
		},
		ServiceName: "my-custom-service",
		ServiceEnv:  "staging",
	}

	logger, err := New(config)
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	// 这些日志不会被记录（级别太低）
	logger.Debug("调试信息")
	logger.Info("信息日志")

	// 这些日志会被记录
	logger.Warn("警告信息")
	logger.Error("错误信息")


}

// Example_performanceLogging 性能日志示例
func Example_performanceLogging() {
	logger, err := New(ProductionConfig("perf-service", "production"))
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	// 记录API调用性能
	logger.Infof("开始API调用: %s", "/api/users")

	// ... 执行API调用 ...

	logger.Info("API调用完成",
		zap.String("endpoint", "/api/users"),
		zap.Duration("duration", 150),
		zap.Int("status_code", 200),
		zap.Int("response_size", 1024),
	)

	// 记录数据库查询性能
	logger.Info("数据库查询",
		zap.String("query", "SELECT * FROM users WHERE active = ?"),
		zap.Duration("duration", 25),
		zap.Int("rows_affected", 10),
	)


}

// Example_errorHandling 错误处理示例
func Example_errorHandling() {
	logger, err := New(DevelopmentConfig())
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	// 模拟一个业务错误
	err = fmt.Errorf("用户不存在")
	if err != nil {
		logger.Error("业务逻辑错误",
			zap.Error(err),
			zap.String("user_id", "12345"),
			zap.String("operation", "get_user"),
		)
	}

	// 模拟系统错误
	err = fmt.Errorf("数据库连接失败: connection timeout")
	if err != nil {
		logger.Error("系统错误",
			zap.Error(err),
			zap.String("component", "database"),
			zap.String("action", "connect"),
		)
	}


}

// Example_structuredLogging 结构化日志示例
func Example_structuredLogging() {
	logger, err := New(ProductionConfig("structured-service", "production"))
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	// 用户操作日志
	logger.Info("用户操作",
		zap.String("user_id", "u123456"),
		zap.String("action", "create_order"),
		zap.String("resource", "order"),
		zap.String("resource_id", "o789012"),
		zap.Float64("amount", 99.99),
		zap.String("currency", "USD"),
	)

	// 系统性能指标
	logger.Info("系统指标",
		zap.String("metric_type", "performance"),
		zap.Float64("cpu_usage", 45.6),
		zap.Float64("memory_usage", 78.2),
		zap.Int("active_connections", 150),
		zap.Duration("avg_response_time", 120),
	)

	// 安全审计日志
	logger.Info("安全审计",
		zap.String("event_type", "login_attempt"),
		zap.String("user_id", "u123456"),
		zap.String("ip_address", "192.168.1.100"),
		zap.String("user_agent", "Mozilla/5.0..."),
		zap.Bool("success", true),
		zap.String("reason", "valid_credentials"),
	)

}
