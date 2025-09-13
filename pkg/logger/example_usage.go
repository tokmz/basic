package logger

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"
)

func ExampleUsage() {
	// 示例1: 基本使用
	basicUsageExample()

	// 示例2: 生产环境配置
	productionExample()

	// 示例3: 文件日志和轮转
	fileLoggingExample()

	// 示例4: 上下文日志
	contextLoggingExample()

	// 示例5: 钩子机制
	hooksExample()

	// 示例6: HTTP中间件
	httpMiddlewareExample()
}

// basicUsageExample 基本使用示例
func basicUsageExample() {
	// 初始化全局日志器
	err := InitGlobal(DevelopmentConfig())
	if err != nil {
		panic(err)
	}
	defer Sync()

	// 基本日志记录
	Info("应用启动")
	Infof("应用启动在端口: %d", 8080)

	// 带结构化字段的日志
	Info("用户登录",
		zap.String("user_id", "12345"),
		zap.String("username", "john_doe"),
		zap.Duration("login_time", 100*time.Millisecond),
	)

	// 使用字段映射
	WithFields(Fields{
		"module": "auth",
		"action": "login",
		"result": "success",
	}).Info("用户认证成功")
}

// productionExample 生产环境示例
func productionExample() {
	// 创建生产环境配置
	config := ProductionConfig("my-service", "production")
	
	// 自定义配置
	config.File.Filename = "./logs/production.log"
	config.File.Rotation.MaxSize = 50 // 50MB

	productionLogger, err := New(config)
	if err != nil {
		panic(err)
	}
	defer productionLogger.Close()

	productionLogger.Info("生产环境服务启动",
		zap.String("version", "1.0.0"),
		zap.Int("port", 8080),
		zap.String("environment", "production"),
	)

	// 模拟一些业务日志
	productionLogger.Info("API调用",
		zap.String("endpoint", "/api/users"),
		zap.String("method", "GET"),
		zap.Int("status_code", 200),
		zap.Duration("response_time", 150*time.Millisecond),
	)
}

// fileLoggingExample 文件日志示例
func fileLoggingExample() {
	config := &Config{
		Level:        InfoLevel,
		Format:       JSONFormat,
		EnableCaller: true,
		EnableStack:  true,
		TimeFormat:   time.RFC3339,
		Console: ConsoleConfig{
			Enabled: false, // 只写文件，不输出到控制台
		},
		File: FileConfig{
			Enabled:  true,
			Filename: "./logs/app.log",
			Rotation: RotationConfig{
				MaxSize:    10,   // 10MB轮转
				MaxAge:     7,    // 保留7天
				MaxBackups: 5,    // 最多5个备份
				Compress:   true, // 压缩旧文件
			},
		},
		ServiceName: "file-logger-demo",
		ServiceEnv:  "development",
	}

	fileLogger, err := New(config)
	if err != nil {
		panic(err)
	}
	defer fileLogger.Close()

	// 写入大量日志测试轮转
	for i := 0; i < 100; i++ {
		fileLogger.Info("日志轮转测试",
			zap.Int("iteration", i),
			zap.String("data", "这是一些测试数据用于填充日志文件"),
			zap.Time("timestamp", time.Now()),
		)
	}

	fileLogger.Info("文件日志测试完成")
}

// contextLoggingExample 上下文日志示例
func contextLoggingExample() {
	contextLogger, err := New(DevelopmentConfig())
	if err != nil {
		panic(err)
	}
	defer contextLogger.Close()

	// 创建带有追踪信息的上下文
	ctx := context.Background()
	ctx = WithTraceID(ctx, "trace-123456")
	ctx = WithRequestID(ctx, "req-789012")
	ctx = WithUserID(ctx, "user-345678")

	// 使用上下文日志
	contextLogger.WithContext(ctx).Info("开始处理用户请求")

	// 模拟业务处理
	processUserRequest(ctx, contextLogger)

	contextLogger.WithContext(ctx).Info("用户请求处理完成")
}

func processUserRequest(ctx context.Context, logger *Logger) {
	// 在子函数中也可以使用相同的上下文
	logger.WithContext(ctx).Debug("验证用户权限")
	logger.WithContext(ctx).Info("查询用户数据")
	logger.WithContext(ctx).Info("生成响应数据")
}

// hooksExample 钩子机制示例
func hooksExample() {
	// 创建钩子
	emailHook := NewEmailHook("smtp.example.com", 587, "app@example.com", "password", []string{"admin@example.com"})
	webhookHook := NewWebhookHook("https://webhook.example.com/alert")
	databaseHook := NewDatabaseHook(ErrorLevel, PanicLevel)

	// 创建带钩子的日志器
	hookLogger, err := NewWithHooks(DevelopmentConfig(), emailHook, webhookHook, databaseHook)
	if err != nil {
		panic(err)
	}
	defer hookLogger.Close()

	// 这些日志会触发相应的钩子
	hookLogger.Info("普通信息日志") // 只有数据库钩子会触发
	hookLogger.Error("数据库连接失败", zap.String("database", "users_db")) // 所有钩子都会触发
}

// httpMiddlewareExample HTTP中间件示例
func httpMiddlewareExample() {
	httpLogger, err := New(DevelopmentConfig())
	if err != nil {
		panic(err)
	}
	defer httpLogger.Close()

	mux := http.NewServeMux()

	// 添加一些路由
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 从上下文获取请求ID
		requestID := GetRequestID(r.Context())
		
		httpLogger.WithContext(r.Context()).Info("处理首页请求",
			zap.String("request_id", requestID),
		)
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	})

	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		httpLogger.WithContext(r.Context()).Info("获取用户列表")
		
		// 模拟数据库查询
		time.Sleep(50 * time.Millisecond)
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"users": []}`))
	})

	mux.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		httpLogger.WithContext(r.Context()).Error("模拟错误")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	})

	// 应用中间件
	handler := HTTPMiddleware(httpLogger)(mux)
	handler = RecoveryMiddleware(httpLogger)(handler)
	handler = RequestLoggingMiddleware(httpLogger, false)(handler)

	httpLogger.Info("HTTP服务器启动", zap.String("address", ":8080"))
	
	// 在实际应用中，这里会调用 http.ListenAndServe(":8080", handler)
	// 为了示例，我们只是打印日志并模拟使用handler
	if handler != nil {
		httpLogger.Info("HTTP服务器配置完成，可以开始处理请求")
	}
}