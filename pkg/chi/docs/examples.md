# Chi 框架代码示例集

本文档提供丰富的代码示例，涵盖 Chi 框架的各种使用场景，从简单的 Hello World 到复杂的企业级微服务架构。

## 目录

- [基础应用示例](#基础应用示例)
- [微服务开发示例](#微服务开发示例)
- [API 网关示例](#api-网关示例)
- [服务治理示例](#服务治理示例)
- [插件开发示例](#插件开发示例)
- [中间件示例](#中间件示例)
- [数据库集成示例](#数据库集成示例)
- [WebSocket 示例](#websocket-示例)
- [部署配置示例](#部署配置示例)

---

## 基础应用示例

### Hello World

最简单的 Chi 应用示例：

```go
package main

import (
    "github.com/your-org/basic/pkg/chi"
    "github.com/gin-gonic/gin"
)

func main() {
    app := chi.New()

    app.GET("/", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "message": "Hello, Chi Framework!",
            "version": "1.0.0",
        })
    })

    app.Run(":8080")
}
```

### RESTful API

完整的 RESTful API 示例：

```go
package main

import (
    "net/http"
    "strconv"

    "github.com/your-org/basic/pkg/chi"
    "github.com/gin-gonic/gin"
)

type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
    Email string `json:"email"`
}

var users = []User{
    {1, "Alice", "alice@example.com"},
    {2, "Bob", "bob@example.com"},
}

func main() {
    app := chi.New()

    // 用户相关路由
    userGroup := app.Group("/api/v1/users")
    {
        userGroup.GET("", getUsers)
        userGroup.GET("/:id", getUser)
        userGroup.POST("", createUser)
        userGroup.PUT("/:id", updateUser)
        userGroup.DELETE("/:id", deleteUser)
    }

    app.Run(":8080")
}

func getUsers(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "users": users,
        "total": len(users),
    })
}

func getUser(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
        return
    }

    for _, user := range users {
        if user.ID == id {
            c.JSON(http.StatusOK, user)
            return
        }
    }

    c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
}

func createUser(c *gin.Context) {
    var newUser User
    if err := c.ShouldBindJSON(&newUser); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    newUser.ID = len(users) + 1
    users = append(users, newUser)

    c.JSON(http.StatusCreated, newUser)
}

func updateUser(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
        return
    }

    var updatedUser User
    if err := c.ShouldBindJSON(&updatedUser); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    for i, user := range users {
        if user.ID == id {
            updatedUser.ID = id
            users[i] = updatedUser
            c.JSON(http.StatusOK, updatedUser)
            return
        }
    }

    c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
}

func deleteUser(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
        return
    }

    for i, user := range users {
        if user.ID == id {
            users = append(users[:i], users[i+1:]...)
            c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
            return
        }
    }

    c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
}
```

---

## 微服务开发示例

### 用户服务

完整的用户微服务示例：

```go
package main

import (
    "context"
    "log"

    "github.com/your-org/basic/pkg/chi/microservice"
    "github.com/gin-gonic/gin"
    "google.golang.org/grpc"
    pb "path/to/your/proto"
)

type UserService struct {
    pb.UnimplementedUserServiceServer
}

func (s *UserService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
    // 实现 gRPC 方法
    return &pb.User{
        Id:    req.Id,
        Name:  "John Doe",
        Email: "john@example.com",
    }, nil
}

func main() {
    // 创建微服务实例
    service := microservice.New("user-service", &microservice.Config{
        Port: 8080,
        Registry: "consul://localhost:8500",
        ConfigCenter: "consul://localhost:8500",
        HealthCheck: &microservice.HealthCheckConfig{
            Enabled:  true,
            Interval: "10s",
            Timeout:  "5s",
        },
    })

    // 注册 gRPC 服务
    userService := &UserService{}
    service.RegisterGRPC(userService)

    // 注册 HTTP 路由
    service.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "healthy"})
    })

    service.GET("/users/:id", func(c *gin.Context) {
        // HTTP API 实现
        id := c.Param("id")
        c.JSON(200, gin.H{
            "id":    id,
            "name":  "John Doe",
            "email": "john@example.com",
        })
    })

    // 启用服务治理
    service.UseCircuitBreaker(&microservice.CircuitBreakerConfig{
        MaxRequests: 10,
        Interval:    "60s",
        Timeout:     "30s",
    })
    service.UseRateLimit(100) // 100 req/s
    service.UseTracing()
    service.UseMetrics()

    // 启动微服务
    if err := service.Start(); err != nil {
        log.Fatal("Failed to start service:", err)
    }
}
```

### 订单服务

与用户服务通信的订单服务：

```go
package main

import (
    "context"
    "log"

    "github.com/your-org/basic/pkg/chi/microservice"
    "github.com/your-org/basic/pkg/chi/discovery"
    "github.com/gin-gonic/gin"
    "google.golang.org/grpc"
    pb "path/to/your/proto"
)

type OrderService struct {
    userServiceClient pb.UserServiceClient
}

func (s *OrderService) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.Order, error) {
    // 调用用户服务验证用户
    user, err := s.userServiceClient.GetUser(ctx, &pb.GetUserRequest{
        Id: req.UserId,
    })
    if err != nil {
        return nil, err
    }

    // 创建订单逻辑
    return &pb.Order{
        Id:     "order-123",
        UserId: user.Id,
        Amount: req.Amount,
        Status: "created",
    }, nil
}

func main() {
    // 服务发现客户端
    discoveryClient, err := discovery.NewConsul("localhost:8500")
    if err != nil {
        log.Fatal("Failed to create discovery client:", err)
    }

    // 发现用户服务
    userServices, err := discoveryClient.Discover("user-service")
    if err != nil {
        log.Fatal("Failed to discover user service:", err)
    }

    // 连接用户服务
    userConn, err := grpc.Dial(userServices[0].Address, grpc.WithInsecure())
    if err != nil {
        log.Fatal("Failed to connect to user service:", err)
    }
    defer userConn.Close()

    userClient := pb.NewUserServiceClient(userConn)

    // 创建订单服务
    service := microservice.New("order-service", &microservice.Config{
        Port: 8081,
        Registry: "consul://localhost:8500",
    })

    orderService := &OrderService{
        userServiceClient: userClient,
    }
    service.RegisterGRPC(orderService)

    // 启动服务
    if err := service.Start(); err != nil {
        log.Fatal("Failed to start service:", err)
    }
}
```

---

## API 网关示例

### 完整网关配置

```go
package main

import (
    "log"

    "github.com/your-org/basic/pkg/chi/gateway"
)

func main() {
    // 创建 API 网关
    gw := gateway.New(&gateway.Config{
        Port: 8080,
        TLS: &gateway.TLSConfig{
            Enabled:  true,
            CertFile: "cert.pem",
            KeyFile:  "key.pem",
        },
        CORS: &gateway.CORSConfig{
            AllowOrigins: []string{"*"},
            AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
            AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
        },
    })

    // 配置路由
    routes := []*gateway.Route{
        {
            Path:         "/api/users/*",
            Service:      "user-service",
            Methods:      []string{"GET", "POST", "PUT", "DELETE"},
            StripPrefix:  "/api",
            LoadBalancer: "round_robin",
            Timeout:      "30s",
            Retries:      3,
        },
        {
            Path:         "/api/orders/*",
            Service:      "order-service",
            Methods:      []string{"GET", "POST", "PUT", "DELETE"},
            StripPrefix:  "/api",
            LoadBalancer: "weighted_round_robin",
            Timeout:      "30s",
            Retries:      3,
        },
        {
            Path:         "/api/payments/*",
            Service:      "payment-service",
            Methods:      []string{"POST"},
            StripPrefix:  "/api",
            LoadBalancer: "least_connections",
            Timeout:      "60s",
            Retries:      1,
        },
    }

    for _, route := range routes {
        if err := gw.AddRoute(route); err != nil {
            log.Fatal("Failed to add route:", err)
        }
    }

    // 启用中间件
    gw.UseAuth("jwt")
    gw.UseRateLimit(1000) // 1000 req/s
    gw.UseCircuitBreaker()
    gw.UseMetrics()
    gw.UseTracing()

    // 启动网关
    if err := gw.Start(); err != nil {
        log.Fatal("Failed to start gateway:", err)
    }
}
```

---

## 服务治理示例

### 熔断器使用

```go
package main

import (
    "errors"
    "log"
    "time"

    "github.com/your-org/basic/pkg/chi/governance"
)

func main() {
    // 配置熔断器
    cb := governance.NewCircuitBreaker(&governance.CircuitBreakerConfig{
        MaxRequests: 3,
        Interval:    time.Minute,
        Timeout:     30 * time.Second,
        ReadyToTrip: func(counts governance.Counts) bool {
            failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
            return counts.Requests >= 3 && failureRatio >= 0.6
        },
        OnStateChange: func(name string, from, to governance.State) {
            log.Printf("Circuit breaker state changed from %v to %v", from, to)
        },
    })

    // 使用熔断器
    for i := 0; i < 10; i++ {
        err := cb.Call(func() error {
            // 模拟可能失败的操作
            if i%2 == 0 {
                return errors.New("service error")
            }
            return nil
        })

        if err != nil {
            log.Printf("Request %d failed: %v", i, err)
        } else {
            log.Printf("Request %d succeeded", i)
        }

        time.Sleep(time.Second)
    }
}
```

### 限流器使用

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/your-org/basic/pkg/chi/governance"
)

func main() {
    // 创建令牌桶限流器
    limiter := governance.NewTokenBucket(10.0, 5) // 10 req/s, 桶容量 5

    // 模拟请求
    for i := 0; i < 20; i++ {
        if limiter.Allow() {
            log.Printf("Request %d allowed", i)
        } else {
            log.Printf("Request %d rate limited", i)
        }
        time.Sleep(100 * time.Millisecond)
    }

    // 使用等待模式
    ctx := context.Background()
    for i := 0; i < 5; i++ {
        err := limiter.Wait(ctx)
        if err != nil {
            log.Printf("Wait failed: %v", err)
            continue
        }
        log.Printf("Request %d processed after wait", i)
    }
}
```

---

## 插件开发示例

### 认证插件

```go
package auth

import (
    "errors"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v4"
    "github.com/your-org/basic/pkg/chi/plugins"
)

type AuthPlugin struct {
    secretKey []byte
    config    map[string]interface{}
}

func (p *AuthPlugin) Name() string {
    return "auth"
}

func (p *AuthPlugin) Version() string {
    return "1.0.0"
}

func (p *AuthPlugin) Init(config map[string]interface{}) error {
    p.config = config

    if secret, ok := config["secret_key"].(string); ok {
        p.secretKey = []byte(secret)
    } else {
        return errors.New("secret_key is required")
    }

    return nil
}

func (p *AuthPlugin) Start() error {
    // 注册中间件
    plugins.RegisterMiddleware("jwt_auth", p.JWTMiddleware())
    return nil
}

func (p *AuthPlugin) Stop() error {
    return nil
}

func (p *AuthPlugin) JWTMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(401, gin.H{"error": "Authorization header required"})
            c.Abort()
            return
        }

        tokenString := strings.TrimPrefix(authHeader, "Bearer ")

        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            return p.secretKey, nil
        })

        if err != nil || !token.Valid {
            c.JSON(401, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }

        if claims, ok := token.Claims.(jwt.MapClaims); ok {
            c.Set("user_id", claims["user_id"])
            c.Set("username", claims["username"])
        }

        c.Next()
    }
}

// 生成 JWT 令牌
func (p *AuthPlugin) GenerateToken(userID string, username string) (string, error) {
    claims := jwt.MapClaims{
        "user_id":  userID,
        "username": username,
        "exp":      time.Now().Add(24 * time.Hour).Unix(),
        "iat":      time.Now().Unix(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(p.secretKey)
}

// 注册插件
func init() {
    plugins.Register("auth", &AuthPlugin{})
}
```

### 日志插件

```go
package logging

import (
    "log"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/your-org/basic/pkg/chi/plugins"
)

type LoggingPlugin struct {
    config map[string]interface{}
}

func (p *LoggingPlugin) Name() string {
    return "logging"
}

func (p *LoggingPlugin) Version() string {
    return "1.0.0"
}

func (p *LoggingPlugin) Init(config map[string]interface{}) error {
    p.config = config
    return nil
}

func (p *LoggingPlugin) Start() error {
    plugins.RegisterMiddleware("request_logger", p.RequestLogger())
    return nil
}

func (p *LoggingPlugin) Stop() error {
    return nil
}

func (p *LoggingPlugin) RequestLogger() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        path := c.Request.URL.Path
        raw := c.Request.URL.RawQuery

        c.Next()

        end := time.Now()
        latency := end.Sub(start)

        if raw != "" {
            path = path + "?" + raw
        }

        log.Printf("[%s] %s %s %d %v %s %s",
            end.Format("2006/01/02 - 15:04:05"),
            c.Request.Method,
            path,
            c.Writer.Status(),
            latency,
            c.Request.UserAgent(),
            c.ClientIP(),
        )
    }
}

func init() {
    plugins.Register("logging", &LoggingPlugin{})
}
```

---

## 中间件示例

### CORS 中间件

```go
package middleware

import (
    "github.com/gin-gonic/gin"
)

type CORSConfig struct {
    AllowOrigins []string
    AllowMethods []string
    AllowHeaders []string
}

func CORS(config *CORSConfig) gin.HandlerFunc {
    return func(c *gin.Context) {
        if len(config.AllowOrigins) > 0 {
            c.Header("Access-Control-Allow-Origin", strings.Join(config.AllowOrigins, ","))
        }

        if len(config.AllowMethods) > 0 {
            c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ","))
        }

        if len(config.AllowHeaders) > 0 {
            c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ","))
        }

        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }

        c.Next()
    }
}
```

### 请求 ID 中间件

```go
package middleware

import (
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

func RequestID() gin.HandlerFunc {
    return func(c *gin.Context) {
        requestID := c.GetHeader("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }

        c.Header("X-Request-ID", requestID)
        c.Set("request_id", requestID)

        c.Next()
    }
}
```

---

## 数据库集成示例

### GORM 集成

```go
package main

import (
    "log"

    "github.com/your-org/basic/pkg/chi"
    "github.com/gin-gonic/gin"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

type User struct {
    ID    uint   `json:"id" gorm:"primarykey"`
    Name  string `json:"name"`
    Email string `json:"email" gorm:"unique"`
}

var db *gorm.DB

func main() {
    var err error

    // 连接数据库
    dsn := "host=localhost user=postgres password=password dbname=testdb port=5432 sslmode=disable"
    db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }

    // 自动迁移
    db.AutoMigrate(&User{})

    app := chi.New()

    // 数据库中间件
    app.Use(func(c *gin.Context) {
        c.Set("db", db)
        c.Next()
    })

    // API 路由
    api := app.Group("/api/v1")
    {
        api.GET("/users", getUsers)
        api.GET("/users/:id", getUser)
        api.POST("/users", createUser)
        api.PUT("/users/:id", updateUser)
        api.DELETE("/users/:id", deleteUser)
    }

    app.Run(":8080")
}

func getDB(c *gin.Context) *gorm.DB {
    return c.MustGet("db").(*gorm.DB)
}

func getUsers(c *gin.Context) {
    db := getDB(c)

    var users []User
    result := db.Find(&users)
    if result.Error != nil {
        c.JSON(500, gin.H{"error": result.Error.Error()})
        return
    }

    c.JSON(200, gin.H{
        "users": users,
        "total": len(users),
    })
}

func getUser(c *gin.Context) {
    db := getDB(c)
    id := c.Param("id")

    var user User
    result := db.First(&user, id)
    if result.Error != nil {
        if result.Error == gorm.ErrRecordNotFound {
            c.JSON(404, gin.H{"error": "User not found"})
        } else {
            c.JSON(500, gin.H{"error": result.Error.Error()})
        }
        return
    }

    c.JSON(200, user)
}

func createUser(c *gin.Context) {
    db := getDB(c)

    var user User
    if err := c.ShouldBindJSON(&user); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    result := db.Create(&user)
    if result.Error != nil {
        c.JSON(500, gin.H{"error": result.Error.Error()})
        return
    }

    c.JSON(201, user)
}

func updateUser(c *gin.Context) {
    db := getDB(c)
    id := c.Param("id")

    var user User
    if result := db.First(&user, id); result.Error != nil {
        c.JSON(404, gin.H{"error": "User not found"})
        return
    }

    if err := c.ShouldBindJSON(&user); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    db.Save(&user)
    c.JSON(200, user)
}

func deleteUser(c *gin.Context) {
    db := getDB(c)
    id := c.Param("id")

    result := db.Delete(&User{}, id)
    if result.Error != nil {
        c.JSON(500, gin.H{"error": result.Error.Error()})
        return
    }

    if result.RowsAffected == 0 {
        c.JSON(404, gin.H{"error": "User not found"})
        return
    }

    c.JSON(200, gin.H{"message": "User deleted successfully"})
}
```

---

## WebSocket 示例

### 聊天室

```go
package main

import (
    "log"
    "net/http"

    "github.com/your-org/basic/pkg/chi"
    "github.com/your-org/basic/pkg/chi/websocket"
    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
)

type Message struct {
    Type    string `json:"type"`
    Room    string `json:"room"`
    User    string `json:"user"`
    Content string `json:"content"`
}

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}

func main() {
    // 创建 WebSocket Hub
    hub := websocket.NewHub(&websocket.HubConfig{
        MaxConnections: 1000,
        BufferSize:     256,
    })

    // 启动 Hub
    go hub.Run()

    app := chi.New()

    // WebSocket 路由
    app.GET("/ws", func(c *gin.Context) {
        conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
        if err != nil {
            log.Printf("WebSocket upgrade failed: %v", err)
            return
        }

        userID := c.Query("user_id")
        room := c.Query("room")

        wsConn := &websocket.Connection{
            ID:     generateID(),
            UserID: userID,
            Rooms:  []string{room},
            Conn:   conn,
        }

        hub.Register(wsConn)

        go handleWebSocketConnection(hub, wsConn)
    })

    // HTTP API
    app.POST("/api/broadcast", func(c *gin.Context) {
        var message Message
        if err := c.ShouldBindJSON(&message); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }

        hub.BroadcastToRoom(message.Room, message)
        c.JSON(200, gin.H{"message": "Broadcast sent"})
    })

    app.Run(":8080")
}

func handleWebSocketConnection(hub *websocket.Hub, conn *websocket.Connection) {
    defer func() {
        hub.Unregister(conn)
        conn.Conn.Close()
    }()

    for {
        var message Message
        err := conn.Conn.ReadJSON(&message)
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                log.Printf("WebSocket error: %v", err)
            }
            break
        }

        switch message.Type {
        case "join_room":
            hub.JoinRoom(conn.ID, message.Room)
        case "leave_room":
            hub.LeaveRoom(conn.ID, message.Room)
        case "message":
            hub.BroadcastToRoom(message.Room, message)
        }
    }
}

func generateID() string {
    // 生成唯一 ID 的实现
    return "conn-" + time.Now().Format("20060102150405")
}
```

---

## 部署配置示例

### Docker 配置

**Dockerfile**:
```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/main .
COPY config.yaml .

CMD ["./main"]
```

**docker-compose.yml**:
```yaml
version: '3.8'

services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - ENV=production
      - LOG_LEVEL=info
    depends_on:
      - postgres
      - redis
      - consul
    volumes:
      - ./config.yaml:/root/config.yaml

  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: myapp
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  consul:
    image: consul:1.15
    ports:
      - "8500:8500"
    command: agent -dev -client=0.0.0.0

volumes:
  postgres_data:
```

### Kubernetes 配置

**deployment.yaml**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: chi-app
  labels:
    app: chi-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: chi-app
  template:
    metadata:
      labels:
        app: chi-app
    spec:
      containers:
      - name: chi-app
        image: your-registry/chi-app:latest
        ports:
        - containerPort: 8080
        env:
        - name: ENV
          value: "production"
        - name: LOG_LEVEL
          value: "info"
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5

---
apiVersion: v1
kind: Service
metadata:
  name: chi-app-service
spec:
  selector:
    app: chi-app
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: LoadBalancer

---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: chi-app-ingress
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  rules:
  - host: api.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: chi-app-service
            port:
              number: 80
```

### 配置文件示例

**config.yaml**:
```yaml
# 基础配置
port: 8080
host: 0.0.0.0
debug: false

# TLS 配置
tls:
  enabled: true
  cert_file: "/etc/ssl/certs/server.crt"
  key_file: "/etc/ssl/private/server.key"

# 数据库配置
database:
  driver: postgres
  host: postgres
  port: 5432
  username: postgres
  password: password
  database: myapp
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: "1h"

# Redis 配置
redis:
  host: redis
  port: 6379
  password: ""
  db: 0
  pool_size: 10

# 服务发现配置
discovery:
  provider: consul
  address: consul:8500
  health_check:
    enabled: true
    interval: "10s"
    timeout: "5s"

# 监控配置
monitoring:
  metrics:
    enabled: true
    path: "/metrics"
  tracing:
    enabled: true
    endpoint: "http://jaeger:14268/api/traces"
    sampler: 1.0

# 日志配置
logging:
  level: info
  format: json
  output: stdout

# 微服务配置
microservice:
  service_name: "my-service"
  version: "1.0.0"
  tags:
    - "api"
    - "production"

# 服务治理配置
governance:
  circuit_breaker:
    enabled: true
    max_requests: 10
    interval: "60s"
    timeout: "30s"
  rate_limit:
    enabled: true
    requests_per_second: 100
  retry:
    enabled: true
    max_attempts: 3
    backoff: "exponential"
```

---

这些示例涵盖了 Chi 框架的主要使用场景，从基础的 HTTP 服务到复杂的微服务架构。每个示例都包含完整的可运行代码，可以作为开发的起点和参考。

**最后更新**: 2025-09-15
**文档版本**: v1.0.0