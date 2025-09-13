package chi

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ExampleBasicUsage 基本使用示例
func ExampleBasicUsage() {
	fmt.Println("=== Chi Framework Basic Usage Example ===")
	
	// 创建应用
	app := New(
		WithName("my-api"),
		WithVersion("1.0.0"),
		WithEnvironment("development"),
	)
	
	// 添加基本路由
	app.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Welcome to Chi Framework",
			"version": app.Config().Version,
		})
	})
	
	app.GET("/users/:id", func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(200, gin.H{
			"user_id": id,
			"name":    "User " + id,
		})
	})
	
	app.POST("/users", func(c *gin.Context) {
		var user struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}
		
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		
		c.JSON(201, gin.H{
			"message": "User created",
			"user":    user,
		})
	})
	
	// 启动服务器（这里只是示例，实际使用时需要在单独的goroutine中运行）
	fmt.Println("Server would start on :8080")
	fmt.Println("Routes available:")
	fmt.Println("  GET  /")
	fmt.Println("  GET  /users/:id")
	fmt.Println("  POST /users")
}

// ExampleWithMiddleware 中间件使用示例
func ExampleWithMiddleware() {
	fmt.Println("\n=== Chi Framework with Middleware Example ===")
	
	// 创建应用
	app := New(
		WithName("middleware-api"),
		WithVersion("1.0.0"),
		WithCORS(&CORSConfig{
			AllowOrigins: []string{"*"},
			AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
		}),
		WithSecurity(DefaultSecurityConfig()),
		WithRateLimit(&RateLimitConfig{
			Rate:   10,
			Burst:  5,
			Window: time.Minute,
		}),
	)
	
	// 添加自定义中间件
	app.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		
		fmt.Printf("Request processed: %s %s in %v\n",
			c.Request.Method, c.Request.URL.Path, duration)
	})
	
	// 添加认证中间件的路由组
	authGroup := app.Group("/api/v1")
	authGroup.Use(AuthMiddleware(func(token string) (interface{}, error) {
		if token == "valid-token" {
			return gin.H{"user_id": 123, "username": "john"}, nil
		}
		return nil, fmt.Errorf("invalid token")
	}))
	
	authGroup.GET("/profile", func(c *gin.Context) {
		user := c.MustGet("user")
		c.JSON(200, gin.H{
			"profile": user,
		})
	})
	
	// 公开路由
	app.GET("/public", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "This is a public endpoint",
		})
	})
	
	fmt.Println("Server configured with middleware:")
	fmt.Println("  - CORS enabled")
	fmt.Println("  - Security headers enabled")
	fmt.Println("  - Rate limiting enabled")
	fmt.Println("  - Custom logging middleware")
	fmt.Println("  - Authentication middleware for /api/v1/*")
}

// ExampleWithWebSocket WebSocket使用示例
func ExampleWithWebSocket() {
	fmt.Println("\n=== Chi Framework with WebSocket Example ===")
	
	// 创建应用
	app := New(
		WithName("websocket-api"),
		WithVersion("1.0.0"),
		WithWebSocket(&WebSocketConfig{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			WriteWait:       10 * time.Second,
			PongWait:        60 * time.Second,
			PingPeriod:      54 * time.Second,
		}),
	)
	
	// WebSocket聊天室示例
	hub := app.WSHub()
	
	app.WebSocket("/ws", func(conn *WebSocketConn) {
		// 注册连接
		hub.Register(conn)
		defer hub.Unregister(conn)
		
		// 发送欢迎消息
		conn.SendJSON(gin.H{
			"type":    "welcome",
			"message": "Connected to chat room",
			"clients": hub.GetClientCount(),
		})
		
		// 广播新用户加入
		hub.BroadcastJSON(gin.H{
			"type":    "user_joined",
			"clients": hub.GetClientCount(),
		})
		
		// 处理消息
		for {
			var message gin.H
			if err := conn.ReadJSON(&message); err != nil {
				break
			}
			
			// 广播消息
			response := gin.H{
				"type":      "message",
				"client_id": conn.ClientID(),
				"message":   message["message"],
				"timestamp": time.Now().Format(time.RFC3339),
			}
			hub.BroadcastJSON(response)
		}
		
		// 广播用户离开
		hub.BroadcastJSON(gin.H{
			"type":    "user_left",
			"clients": hub.GetClientCount() - 1,
		})
	})
	
	// WebSocket房间示例
	app.WebSocket("/ws/room/:room", func(conn *WebSocketConn) {
		// 从URL获取房间名
		// 这里需要额外的处理来获取房间参数
		room := "default" // 简化示例
		
		// 加入房间
		hub.JoinRoom(conn, room)
		defer hub.LeaveRoom(conn, room)
		
		conn.SendJSON(gin.H{
			"type":    "joined_room",
			"room":    room,
			"clients": hub.GetRoomClients(room),
		})
		
		// 向房间广播
		hub.BroadcastToRoom(room, []byte(fmt.Sprintf(
			`{"type":"user_joined_room","room":"%s","clients":%d}`,
			room, hub.GetRoomClients(room))))
		
		for {
			var message gin.H
			if err := conn.ReadJSON(&message); err != nil {
				break
			}
			
			hub.BroadcastToRoom(room, []byte(fmt.Sprintf(
				`{"type":"room_message","room":"%s","message":"%v","client_id":"%s"}`,
				room, message["message"], conn.ClientID())))
		}
	})
	
	// WebSocket状态端点
	app.GET("/ws/status", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"total_clients": hub.GetClientCount(),
			"total_rooms":   hub.GetRoomCount(),
		})
	})
	
	fmt.Println("WebSocket endpoints configured:")
	fmt.Println("  - /ws (chat room)")
	fmt.Println("  - /ws/room/:room (room-based chat)")
	fmt.Println("  - GET /ws/status (WebSocket status)")
}

// ExampleWithHealthChecks 健康检查示例
func ExampleWithHealthChecks() {
	fmt.Println("\n=== Chi Framework with Health Checks Example ===")
	
	// 创建应用
	app := New(
		WithName("health-check-api"),
		WithVersion("1.0.0"),
	)
	
	// 添加健康检查
	health := app.Health()
	
	// 数据库健康检查
	health.AddDatabaseCheck("database", func() error {
		// 模拟数据库连接检查
		time.Sleep(10 * time.Millisecond)
		return nil // 假设连接成功
	})
	
	// Redis健康检查
	health.AddRedisCheck("redis", func() error {
		// 模拟Redis连接检查
		time.Sleep(5 * time.Millisecond)
		return nil // 假设连接成功
	})
	
	// 外部服务健康检查
	health.AddExternalServiceCheck("payment-service", "https://api.payment.com", func(url string) error {
		// 模拟外部服务检查
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != 200 {
			return fmt.Errorf("service returned status %d", resp.StatusCode)
		}
		return nil
	})
	
	// 自定义健康检查
	health.AddCheck("custom", func() CheckResult {
		// 执行自定义检查逻辑
		return CheckResult{
			Status:  "healthy",
			Message: "Custom check passed",
			Details: map[string]interface{}{
				"version": "1.0.0",
				"feature_flags": map[string]bool{
					"new_feature": true,
				},
			},
		}
	})
	
	// 指标端点
	app.GET("/metrics/detailed", func(c *gin.Context) {
		metrics := app.Metrics().GetMetrics()
		c.JSON(200, metrics)
	})
	
	fmt.Println("Health checks configured:")
	fmt.Println("  - Database connectivity")
	fmt.Println("  - Redis connectivity") 
	fmt.Println("  - External payment service")
	fmt.Println("  - Custom business logic check")
	fmt.Println("Endpoints:")
	fmt.Println("  - GET /health (health status)")
	fmt.Println("  - GET /ready (readiness check)")
	fmt.Println("  - GET /metrics (basic metrics)")
	fmt.Println("  - GET /metrics/detailed (detailed metrics)")
}

// ExampleProduction 生产环境配置示例
func ExampleProduction() {
	fmt.Println("\n=== Chi Framework Production Example ===")
	
	// 生产环境配置
	app := New(
		WithName("production-api"),
		WithVersion("2.1.0"),
		WithEnvironment("production"),
		WithMode(gin.ReleaseMode),
		WithBuildInfo("2024-01-15T10:30:00Z", "abc123"),
		WithTimeouts(5*time.Second, 5*time.Second, 30*time.Second),
		WithCORS(&CORSConfig{
			AllowOrigins:     []string{"https://myapp.com", "https://admin.myapp.com"},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
			AllowHeaders:     []string{"Authorization", "Content-Type"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}),
		WithSecurity(&SecurityConfig{
			ContentTypeNosniff:    true,
			XFrameOptions:         "DENY",
			XSSProtection:         "1; mode=block",
			ContentSecurityPolicy: "default-src 'self'",
			HSTSMaxAge:            31536000,
			HSTSIncludeSubdomains: true,
		}),
		WithRateLimit(&RateLimitConfig{
			Rate:   100,
			Burst:  50,
			Window: time.Minute,
		}),
	)
	
	// 添加生产环境中间件
	logger := &DefaultLogger{}
	app.Use(LoggingMiddleware(logger))
	app.Use(RecoveryMiddleware(logger))
	app.Use(ValidationMiddleware())
	app.Use(CompressMiddleware())
	
	// API版本控制
	v1 := app.Group("/api/v1")
	v2 := app.Group("/api/v2")
	
	v1.GET("/users", func(c *gin.Context) {
		c.JSON(200, gin.H{"version": "v1", "users": []string{}})
	})
	
	v2.GET("/users", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"version": "v2",
			"users":   []gin.H{},
			"meta": gin.H{
				"total": 0,
				"page":  1,
			},
		})
	})
	
	// 管理接口（需要特殊认证）
	admin := app.Group("/admin")
	admin.Use(AuthMiddleware(func(token string) (interface{}, error) {
		if token == "admin-secret-token" {
			return gin.H{"role": "admin"}, nil
		}
		return nil, fmt.Errorf("unauthorized")
	}))
	
	admin.GET("/stats", func(c *gin.Context) {
		c.JSON(200, app.Metrics().GetMetrics())
	})
	
	admin.POST("/health/reset", func(c *gin.Context) {
		app.Metrics().Reset()
		c.JSON(200, gin.H{"message": "Metrics reset"})
	})
	
	fmt.Println("Production configuration:")
	fmt.Println("  - Release mode enabled")
	fmt.Println("  - Strict CORS policy")
	fmt.Println("  - Security headers enforced")
	fmt.Println("  - Rate limiting active")
	fmt.Println("  - Comprehensive logging")
	fmt.Println("  - Response compression")
	fmt.Println("  - API versioning")
	fmt.Println("  - Admin endpoints protected")
}

// RunExample 运行示例
func RunExample() {
	fmt.Println("🚀 Chi Framework Examples")
	fmt.Println("========================")
	
	ExampleBasicUsage()
	ExampleWithMiddleware()
	ExampleWithWebSocket()
	ExampleWithHealthChecks()
	ExampleProduction()
	
	fmt.Println("\n✅ All examples completed!")
	fmt.Println("To run a real server, create an app and call app.Run(\":8080\")")
}

// 示例：完整的Web应用
func ExampleFullWebApp() {
	// 创建应用
	app := New(
		WithName("my-web-app"),
		WithVersion("1.0.0"),
		WithEnvironment("development"),
	)
	
	// 静态文件服务
	app.Static("/static", "./static")
	app.StaticFile("/favicon.ico", "./static/favicon.ico")
	
	// 模板渲染（如果使用HTML模板）
	app.Engine().LoadHTMLGlob("templates/*")
	
	// 首页
	app.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title":   "My Web App",
			"version": app.Config().Version,
		})
	})
	
	// API路由
	api := app.Group("/api")
	{
		api.GET("/status", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":  "ok",
				"version": app.Config().Version,
			})
		})
		
		api.POST("/contact", func(c *gin.Context) {
			var form struct {
				Name    string `json:"name" binding:"required"`
				Email   string `json:"email" binding:"required,email"`
				Message string `json:"message" binding:"required"`
			}
			
			if err := c.ShouldBindJSON(&form); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}
			
			// 处理联系表单
			log.Printf("Contact form: %+v", form)
			
			c.JSON(200, gin.H{
				"message": "Thank you for your message!",
			})
		})
	}
	
	// 启动服务器
	log.Println("Starting web application...")
	if err := app.Run(":8080"); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}