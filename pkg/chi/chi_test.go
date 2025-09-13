package chi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func TestNewApp(t *testing.T) {
	// 测试默认配置
	app := New()
	if app == nil {
		t.Fatal("Expected app to be created")
	}
	
	config := app.Config()
	if config.Name != "chi-app" {
		t.Errorf("Expected default name 'chi-app', got '%s'", config.Name)
	}
	
	// 测试自定义配置
	app = New(
		WithName("test-app"),
		WithVersion("2.0.0"),
		WithEnvironment("test"),
	)
	
	config = app.Config()
	if config.Name != "test-app" {
		t.Errorf("Expected name 'test-app', got '%s'", config.Name)
	}
	if config.Version != "2.0.0" {
		t.Errorf("Expected version '2.0.0', got '%s'", config.Version)
	}
	if config.Environment != "test" {
		t.Errorf("Expected environment 'test', got '%s'", config.Environment)
	}
}

func TestAppRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := New(
		WithMode(gin.TestMode),
		DisableCORS(),
		DisableSecurity(),
		DisableMetrics(),
		DisableRequestID(),
	)
	
	// 添加测试路由
	app.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "test"})
	})
	
	app.POST("/test", func(c *gin.Context) {
		c.JSON(201, gin.H{"message": "created"})
	})
	
	app.PUT("/test/:id", func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(200, gin.H{"id": id, "message": "updated"})
	})
	
	app.DELETE("/test/:id", func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(200, gin.H{"id": id, "message": "deleted"})
	})
	
	// 测试GET请求
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	app.Engine().ServeHTTP(w, req)
	
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["message"] != "test" {
		t.Errorf("Expected message 'test', got '%v'", response["message"])
	}
	
	// 测试POST请求
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/test", nil)
	app.Engine().ServeHTTP(w, req)
	
	if w.Code != 201 {
		t.Errorf("Expected status 201, got %d", w.Code)
	}
	
	// 测试PUT请求
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/test/123", nil)
	app.Engine().ServeHTTP(w, req)
	
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["id"] != "123" {
		t.Errorf("Expected id '123', got '%v'", response["id"])
	}
	
	// 测试DELETE请求
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/test/456", nil)
	app.Engine().ServeHTTP(w, req)
	
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAppGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := New(
		WithMode(gin.TestMode),
		DisableCORS(),
		DisableSecurity(),
		DisableMetrics(),
		DisableRequestID(),
	)
	
	// 测试路由组
	v1 := app.Group("/api/v1")
	v1.GET("/users", func(c *gin.Context) {
		c.JSON(200, gin.H{"version": "v1", "users": []string{}})
	})
	
	v2 := app.Group("/api/v2")
	v2.GET("/users", func(c *gin.Context) {
		c.JSON(200, gin.H{"version": "v2", "users": []string{}})
	})
	
	// 测试v1路由
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/users", nil)
	app.Engine().ServeHTTP(w, req)
	
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["version"] != "v1" {
		t.Errorf("Expected version 'v1', got '%v'", response["version"])
	}
	
	// 测试v2路由
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v2/users", nil)
	app.Engine().ServeHTTP(w, req)
	
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["version"] != "v2" {
		t.Errorf("Expected version 'v2', got '%v'", response["version"])
	}
}

func TestDefaultRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := New(
		WithMode(gin.TestMode),
		DisableCORS(),
		DisableSecurity(),
		DisableRequestID(),
	)
	
	// 测试健康检查
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	app.Engine().ServeHTTP(w, req)
	
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var health HealthStatus
	json.Unmarshal(w.Body.Bytes(), &health)
	if health.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", health.Status)
	}
	
	// 测试就绪检查
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/ready", nil)
	app.Engine().ServeHTTP(w, req)
	
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	// 测试应用信息
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/info", nil)
	app.Engine().ServeHTTP(w, req)
	
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var info map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &info)
	if info["name"] != "chi-app" {
		t.Errorf("Expected name 'chi-app', got '%v'", info["name"])
	}
	
	// 测试指标端点
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/metrics", nil)
	app.Engine().ServeHTTP(w, req)
	
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var metrics MetricsData
	json.Unmarshal(w.Body.Bytes(), &metrics)
	if metrics.RequestCount < 0 {
		t.Errorf("Expected non-negative request count")
	}
}

func TestMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := New(
		WithMode(gin.TestMode),
		WithCORS(DefaultCORSConfig()),
		WithSecurity(DefaultSecurityConfig()),
	)
	
	app.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "test"})
	})
	
	// 测试CORS中间件
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	app.Engine().ServeHTTP(w, req)
	
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("Expected CORS headers to be set")
	}
	
	// 测试安全中间件
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("Expected X-Content-Type-Options header to be set")
	}
	
	// 测试OPTIONS预检请求
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	app.Engine().ServeHTTP(w, req)
	
	if w.Code != 204 {
		t.Errorf("Expected status 204 for OPTIONS, got %d", w.Code)
	}
}

func TestRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := New(
		WithMode(gin.TestMode),
		DisableCORS(),
		DisableSecurity(),
		DisableMetrics(),
	)
	
	app.GET("/test", func(c *gin.Context) {
		requestID := c.GetString("RequestID")
		if requestID == "" {
			t.Error("Expected RequestID to be set in context")
		}
		c.JSON(200, gin.H{"request_id": requestID})
	})
	
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	app.Engine().ServeHTTP(w, req)
	
	if w.Header().Get("X-Request-ID") == "" {
		t.Error("Expected X-Request-ID header to be set")
	}
	
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["request_id"] == "" {
		t.Error("Expected request_id in response")
	}
}

func TestMetrics(t *testing.T) {
	metrics := NewMetrics()
	
	// 测试初始状态
	data := metrics.GetMetrics()
	if data.RequestCount != 0 {
		t.Errorf("Expected RequestCount 0, got %d", data.RequestCount)
	}
	
	// 测试记录指标
	metrics.IncCurrentRequests()
	metrics.Record("GET", "/test", 200, time.Millisecond*100)
	data = metrics.GetMetrics()
	
	if data.RequestCount != 1 {
		t.Errorf("Expected RequestCount 1, got %d", data.RequestCount)
	}
	
	if data.StatusCounts[200] != 1 {
		t.Errorf("Expected status 200 count 1, got %d", data.StatusCounts[200])
	}
	
	if data.MethodCounts["GET"] != 1 {
		t.Errorf("Expected GET count 1, got %d", data.MethodCounts["GET"])
	}
	
	if data.PathCounts["/test"] != 1 {
		t.Errorf("Expected /test count 1, got %d", data.PathCounts["/test"])
	}
	
	// 测试错误计数
	metrics.IncCurrentRequests()
	metrics.Record("POST", "/test", 500, time.Millisecond*200)
	data = metrics.GetMetrics()
	
	if data.ErrorCount != 1 {
		t.Errorf("Expected ErrorCount 1, got %d", data.ErrorCount)
	}
	
	if data.ErrorRate <= 0 {
		t.Errorf("Expected positive error rate, got %f", data.ErrorRate)
	}
	
	// 测试重置
	metrics.Reset()
	data = metrics.GetMetrics()
	
	if data.RequestCount != 0 {
		t.Errorf("Expected RequestCount 0 after reset, got %d", data.RequestCount)
	}
}

func TestHealthChecker(t *testing.T) {
	checker := NewHealthChecker("test-app", "1.0.0", "test")
	
	// 添加健康检查
	checker.AddCheck("database", func() CheckResult {
		return CheckResult{
			Status:  "healthy",
			Message: "Database connection successful",
		}
	})
	
	checker.AddCheck("cache", func() CheckResult {
		return CheckResult{
			Status:  "unhealthy",
			Message: "Cache connection failed",
		}
	})
	
	// 执行检查
	health := checker.Check()
	
	if health.Status != "unhealthy" {
		t.Errorf("Expected status 'unhealthy', got '%s'", health.Status)
	}
	
	if health.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", health.Version)
	}
	
	if len(health.Checks) != 2 {
		t.Errorf("Expected 2 checks, got %d", len(health.Checks))
	}
	
	if health.Checks["database"].Status != "healthy" {
		t.Errorf("Expected database check to be healthy")
	}
	
	if health.Checks["cache"].Status != "unhealthy" {
		t.Errorf("Expected cache check to be unhealthy")
	}
	
	// 移除不健康的检查
	checker.RemoveCheck("cache")
	health = checker.Check()
	
	if health.Status != "healthy" {
		t.Errorf("Expected status 'healthy' after removing unhealthy check, got '%s'", health.Status)
	}
	
	if len(health.Checks) != 1 {
		t.Errorf("Expected 1 check after removal, got %d", len(health.Checks))
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(2, time.Second)
	
	// 测试允许请求
	if !limiter.Allow("test") {
		t.Error("Expected first request to be allowed")
	}
	
	if !limiter.Allow("test") {
		t.Error("Expected second request to be allowed")
	}
	
	// 测试限流
	if limiter.Allow("test") {
		t.Error("Expected third request to be blocked")
	}
	
	// 测试不同key
	if !limiter.Allow("test2") {
		t.Error("Expected request with different key to be allowed")
	}
	
	// 等待时间窗口过期
	time.Sleep(time.Second + time.Millisecond*100)
	
	// 应该可以再次请求
	if !limiter.Allow("test") {
		t.Error("Expected request to be allowed after window expired")
	}
}

func TestWebSocket(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := New(
		WithMode(gin.TestMode),
		DisableCORS(),
		DisableSecurity(),
		DisableMetrics(),
		DisableRequestID(),
	)
	
	// 创建WebSocket处理器
	app.WebSocket("/ws", func(conn *WebSocketConn) {
		defer conn.Close()
		
		// 发送欢迎消息
		conn.SendJSON(gin.H{
			"type":    "welcome",
			"message": "Connected",
		})
		
		// Echo消息
		for {
			var message gin.H
			if err := conn.ReadJSON(&message); err != nil {
				break
			}
			
			response := gin.H{
				"type":    "echo",
				"message": message["message"],
			}
			if err := conn.SendJSON(response); err != nil {
				break
			}
		}
	})
	
	// 启动测试服务器
	testServer := httptest.NewServer(app.Engine())
	defer testServer.Close()
	
	// 连接WebSocket
	url := "ws" + testServer.URL[4:] + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()
	
	// 读取欢迎消息
	var welcome gin.H
	err = conn.ReadJSON(&welcome)
	if err != nil {
		t.Fatalf("Failed to read welcome message: %v", err)
	}
	
	if welcome["type"] != "welcome" {
		t.Errorf("Expected welcome type, got %v", welcome["type"])
	}
	
	// 发送测试消息
	testMessage := gin.H{"message": "Hello WebSocket"}
	err = conn.WriteJSON(testMessage)
	if err != nil {
		t.Fatalf("Failed to write message: %v", err)
	}
	
	// 读取回声响应
	var response gin.H
	err = conn.ReadJSON(&response)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}
	
	if response["type"] != "echo" {
		t.Errorf("Expected echo type, got %v", response["type"])
	}
	
	if response["message"] != "Hello WebSocket" {
		t.Errorf("Expected 'Hello WebSocket', got %v", response["message"])
	}
}

func TestWebSocketHub(t *testing.T) {
	hub := NewWebSocketHub()
	
	// 测试初始状态
	if hub.GetClientCount() != 0 {
		t.Errorf("Expected 0 clients, got %d", hub.GetClientCount())
	}
	
	if hub.GetRoomCount() != 0 {
		t.Errorf("Expected 0 rooms, got %d", hub.GetRoomCount())
	}
	
	// 这里可以添加更多WebSocket Hub的测试
	// 但需要模拟WebSocket连接，比较复杂
}

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := New(
		WithMode(gin.TestMode),
		DisableCORS(),
		DisableSecurity(),
		DisableMetrics(),
		DisableRequestID(),
	)
	
	// 添加认证中间件
	authMiddleware := AuthMiddleware(func(token string) (interface{}, error) {
		if token == "valid-token" {
			return gin.H{"user_id": 123}, nil
		}
		return nil, fmt.Errorf("invalid token")
	})
	
	app.GET("/protected", authMiddleware, func(c *gin.Context) {
		user := c.MustGet("user")
		c.JSON(200, gin.H{"user": user})
	})
	
	// 测试无token请求
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	app.Engine().ServeHTTP(w, req)
	
	if w.Code != 401 {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
	
	// 测试无效token
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	app.Engine().ServeHTTP(w, req)
	
	if w.Code != 401 {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
	
	// 测试有效token
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	app.Engine().ServeHTTP(w, req)
	
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["user"] == nil {
		t.Error("Expected user in response")
		return
	}
	user := response["user"].(map[string]interface{})
	if user["user_id"] != float64(123) {
		t.Errorf("Expected user_id 123, got %v", user["user_id"])
	}
}

func TestAppLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := New(
		WithMode(gin.TestMode),
		DisableCORS(),
		DisableSecurity(),
		DisableMetrics(),
		DisableRequestID(),
	)
	
	beforeStartCalled := false
	afterStartCalled := false
	beforeStopCalled := false
	afterStopCalled := false
	
	// 添加生命周期钩子
	app.BeforeStart(func() error {
		beforeStartCalled = true
		return nil
	})
	
	app.AfterStart(func() error {
		afterStartCalled = true
		return nil
	})
	
	app.BeforeStop(func() error {
		beforeStopCalled = true
		return nil
	})
	
	app.AfterStop(func() error {
		afterStopCalled = true
		return nil
	})
	
	// 启动服务器
	go func() {
		app.Start(":0") // 使用随机端口
	}()
	
	// 等待启动
	time.Sleep(100 * time.Millisecond)
	
	// 检查启动钩子
	if !beforeStartCalled {
		t.Error("Expected beforeStart hook to be called")
	}
	
	if !afterStartCalled {
		t.Error("Expected afterStart hook to be called")
	}
	
	// 关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	app.Shutdown(ctx)
	
	// 检查停止钩子
	if !beforeStopCalled {
		t.Error("Expected beforeStop hook to be called")
	}
	
	if !afterStopCalled {
		t.Error("Expected afterStop hook to be called")
	}
}

func TestJSONBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := New(
		WithMode(gin.TestMode),
		DisableCORS(),
		DisableSecurity(),
		DisableMetrics(),
		DisableRequestID(),
	)
	
	app.POST("/users", func(c *gin.Context) {
		var user struct {
			Name  string `json:"name" binding:"required"`
			Email string `json:"email" binding:"required,email"`
			Age   int    `json:"age" binding:"min=0,max=120"`
		}
		
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		
		c.JSON(201, gin.H{"user": user})
	})
	
	// 测试有效数据
	userData := gin.H{
		"name":  "John Doe",
		"email": "john@example.com",
		"age":   30,
	}
	
	body, _ := json.Marshal(userData)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	app.Engine().ServeHTTP(w, req)
	
	if w.Code != 201 {
		t.Errorf("Expected status 201, got %d", w.Code)
	}
	
	// 测试无效数据
	invalidData := gin.H{
		"name":  "", // required field empty
		"email": "invalid-email",
		"age":   -1, // below minimum
	}
	
	body, _ = json.Marshal(invalidData)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	app.Engine().ServeHTTP(w, req)
	
	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
	
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["error"] == nil {
		t.Error("Expected error in response")
	}
}

func BenchmarkApp(b *testing.B) {
	gin.SetMode(gin.TestMode)
	app := New(
		WithMode(gin.TestMode),
		DisableCORS(),
		DisableSecurity(),
		DisableMetrics(),
		DisableRequestID(),
	)
	
	app.GET("/benchmark", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "benchmark"})
	})
	
	req, _ := http.NewRequest("GET", "/benchmark", nil)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w := httptest.NewRecorder()
			app.Engine().ServeHTTP(w, req)
		}
	})
}