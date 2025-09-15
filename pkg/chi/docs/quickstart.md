# Chi 框架快速开始指南

## 概述

本指南将帮助您在 5 分钟内快速搭建并运行一个基于 Chi 框架的企业级微服务应用。我们将从最简单的 Hello World 开始，逐步展示 Chi 框架在微服务开发、API 网关、服务治理等方面的强大功能。

## 学习路径

本指南包含以下内容：
1. **基础应用** - 创建第一个 Chi 应用
2. **微服务开发** - 构建微服务架构
3. **API 网关** - 配置服务网关
4. **服务治理** - 启用监控和治理
5. **生产部署** - 容器化和 Kubernetes 部署

## 环境要求

### 系统要求
- **Go版本**: 1.19 或更高版本
- **操作系统**: Linux, macOS, Windows
- **内存**: 最少 512MB，推荐 2GB 或更多
- **磁盘空间**: 最少 100MB

### 依赖服务（可选）
- **Redis**: 用于缓存和会话管理
- **PostgreSQL/MySQL**: 用于数据持久化
- **Consul/Etcd**: 用于服务发现
- **Jaeger**: 用于分布式追踪

## 快速安装

### 1. 安装 Go 环境

```bash
# 检查 Go 版本
go version

# 如果需要安装 Go，请访问: https://golang.org/dl/
```

### 2. 创建新项目

```bash
# 创建项目目录
mkdir chi-demo
cd chi-demo

# 初始化 Go 模块
go mod init chi-demo

# 安装 Chi 框架
go get github.com/your-org/basic/pkg/chi
```

### 3. 创建第一个应用

创建 `main.go` 文件：

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/your-org/basic/pkg/chi"
)

func main() {
    // 创建 Chi 应用实例
    app := chi.New()

    // 添加基础路由
    app.GET("/", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "message": "Hello Chi Framework!",
            "version": "1.0.0",
        })
    })

    // 添加健康检查端点
    app.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status": "healthy",
            "timestamp": time.Now(),
        })
    })

    // 启动服务器
    app.Run(":8080")
}
```

### 4. 运行应用

```bash
# 运行应用
go run main.go

# 应用将在 http://localhost:8080 启动
```

### 5. 测试应用

```bash
# 测试基础路由
curl http://localhost:8080

# 测试健康检查
curl http://localhost:8080/health
```

## 基础功能示例

### 1. RESTful API 服务

创建 `api_example.go`：

```go
package main

import (
    "strconv"
    "github.com/gin-gonic/gin"
    "github.com/your-org/basic/pkg/chi"
)

type User struct {
    ID       int    `json:"id"`
    Name     string `json:"name"`
    Email    string `json:"email"`
    CreateAt string `json:"created_at"`
}

var users = []User{
    {1, "Alice", "alice@example.com", "2023-01-01"},
    {2, "Bob", "bob@example.com", "2023-01-02"},
}

func main() {
    app := chi.New()

    // 用户管理路由组
    userGroup := app.Group("/api/users")
    {
        // 获取所有用户
        userGroup.GET("/", func(c *gin.Context) {
            c.JSON(200, gin.H{
                "data":  users,
                "count": len(users),
            })
        })

        // 获取单个用户
        userGroup.GET("/:id", func(c *gin.Context) {
            id, _ := strconv.Atoi(c.Param("id"))

            for _, user := range users {
                if user.ID == id {
                    c.JSON(200, gin.H{
                        "data": user,
                    })
                    return
                }
            }

            c.JSON(404, gin.H{
                "error": "User not found",
            })
        })

        // 创建用户
        userGroup.POST("/", func(c *gin.Context) {
            var newUser User
            if err := c.ShouldBindJSON(&newUser); err != nil {
                c.JSON(400, gin.H{
                    "error": err.Error(),
                })
                return
            }

            newUser.ID = len(users) + 1
            newUser.CreateAt = time.Now().Format("2006-01-02")
            users = append(users, newUser)

            c.JSON(201, gin.H{
                "data": newUser,
            })
        })

        // 更新用户
        userGroup.PUT("/:id", func(c *gin.Context) {
            id, _ := strconv.Atoi(c.Param("id"))

            for i, user := range users {
                if user.ID == id {
                    var updatedUser User
                    if err := c.ShouldBindJSON(&updatedUser); err != nil {
                        c.JSON(400, gin.H{
                            "error": err.Error(),
                        })
                        return
                    }

                    updatedUser.ID = id
                    updatedUser.CreateAt = user.CreateAt
                    users[i] = updatedUser

                    c.JSON(200, gin.H{
                        "data": updatedUser,
                    })
                    return
                }
            }

            c.JSON(404, gin.H{
                "error": "User not found",
            })
        })

        // 删除用户
        userGroup.DELETE("/:id", func(c *gin.Context) {
            id, _ := strconv.Atoi(c.Param("id"))

            for i, user := range users {
                if user.ID == id {
                    users = append(users[:i], users[i+1:]...)
                    c.JSON(200, gin.H{
                        "message": "User deleted successfully",
                    })
                    return
                }
            }

            c.JSON(404, gin.H{
                "error": "User not found",
            })
        })
    }

    app.Run(":8080")
}
```

测试 RESTful API：

```bash
# 获取所有用户
curl http://localhost:8080/api/users

# 获取单个用户
curl http://localhost:8080/api/users/1

# 创建用户
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Charlie","email":"charlie@example.com"}'

# 更新用户
curl -X PUT http://localhost:8080/api/users/1 \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice Smith","email":"alice.smith@example.com"}'

# 删除用户
curl -X DELETE http://localhost:8080/api/users/1
```

### 2. 中间件使用

创建 `middleware_example.go`：

```go
package main

import (
    "time"
    "github.com/gin-gonic/gin"
    "github.com/your-org/basic/pkg/chi"
)

// 自定义日志中间件
func CustomLogger() gin.HandlerFunc {
    return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
        return fmt.Sprintf("[CHI] %v | %3d | %13v | %15s | %-7s %#v\n",
            param.TimeStamp.Format("2006/01/02 - 15:04:05"),
            param.StatusCode,
            param.Latency,
            param.ClientIP,
            param.Method,
            param.Path,
        )
    })
}

// CORS 中间件
func CORSMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Access-Control-Allow-Origin", "*")
        c.Header("Access-Control-Allow-Credentials", "true")
        c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
        c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }

        c.Next()
    }
}

// 请求限流中间件
func RateLimitMiddleware() gin.HandlerFunc {
    // 简单的内存限流实现
    requests := make(map[string][]time.Time)
    const maxRequests = 100
    const timeWindow = time.Minute

    return func(c *gin.Context) {
        clientIP := c.ClientIP()
        now := time.Now()

        // 清理过期记录
        if times, exists := requests[clientIP]; exists {
            validTimes := []time.Time{}
            for _, t := range times {
                if now.Sub(t) < timeWindow {
                    validTimes = append(validTimes, t)
                }
            }
            requests[clientIP] = validTimes
        }

        // 检查限流
        if len(requests[clientIP]) >= maxRequests {
            c.JSON(429, gin.H{
                "error": "Too many requests",
                "retry_after": 60,
            })
            c.Abort()
            return
        }

        // 记录请求
        requests[clientIP] = append(requests[clientIP], now)
        c.Next()
    }
}

func main() {
    app := chi.New()

    // 使用中间件
    app.Use(CustomLogger())
    app.Use(CORSMiddleware())
    app.Use(RateLimitMiddleware())

    // 添加路由
    app.GET("/api/data", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "data": "This endpoint is protected by rate limiting",
            "timestamp": time.Now(),
        })
    })

    app.Run(":8080")
}
```

## 企业级功能配置

### 1. 配置文件管理

创建 `config.yaml`：

```yaml
# 服务配置
server:
  host: "0.0.0.0"
  port: 8080
  mode: "release" # debug, release, test
  read_timeout: "30s"
  write_timeout: "30s"

# 数据库配置
database:
  driver: "postgres"
  host: "localhost"
  port: 5432
  username: "chi_user"
  password: "chi_password"
  database: "chi_db"
  sslmode: "disable"
  max_open_conns: 50
  max_idle_conns: 10

# Redis配置
redis:
  host: "localhost"
  port: 6379
  password: ""
  db: 0
  pool_size: 10

# 日志配置
logging:
  level: "info"
  format: "json"
  output: "stdout"

# 服务治理配置
governance:
  circuit_breaker:
    enabled: true
    failure_threshold: 0.6
    timeout: "30s"

  rate_limiter:
    enabled: true
    requests_per_second: 100

  timeout:
    default: "30s"

# 可观测性配置
observability:
  tracing:
    enabled: true
    jaeger_endpoint: "http://localhost:14268/api/traces"
    sample_ratio: 0.1

  metrics:
    enabled: true
    prometheus_endpoint: ":9090"

  health_check:
    enabled: true
    endpoint: "/health"
```

创建配置管理 `config_example.go`：

```go
package main

import (
    "fmt"
    "github.com/spf13/viper"
    "github.com/your-org/basic/pkg/chi"
    "github.com/your-org/basic/pkg/chi/config"
)

func main() {
    // 加载配置
    cfg, err := config.Load("config.yaml")
    if err != nil {
        panic(fmt.Sprintf("Failed to load config: %v", err))
    }

    // 创建应用
    app := chi.NewWithConfig(cfg)

    // 使用配置中的设置
    app.GET("/config", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "server_mode": cfg.Server.Mode,
            "database":    cfg.Database.Driver,
            "redis_host":  cfg.Redis.Host,
        })
    })

    // 使用配置中的端口启动
    app.Run(fmt.Sprintf(":%d", cfg.Server.Port))
}
```

### 2. 数据库集成

创建 `database_example.go`：

```go
package main

import (
    "database/sql"
    "github.com/gin-gonic/gin"
    "github.com/your-org/basic/pkg/chi"
    "github.com/your-org/basic/pkg/database"
    _ "github.com/lib/pq"
)

type User struct {
    ID        int    `json:"id" db:"id"`
    Name      string `json:"name" db:"name"`
    Email     string `json:"email" db:"email"`
    CreatedAt string `json:"created_at" db:"created_at"`
}

func main() {
    app := chi.New()

    // 初始化数据库
    db, err := database.NewConnection(database.Config{
        Driver:   "postgres",
        Host:     "localhost",
        Port:     5432,
        Username: "chi_user",
        Password: "chi_password",
        Database: "chi_db",
        SSLMode:  "disable",
    })
    if err != nil {
        panic(err)
    }
    defer db.Close()

    // 创建表（仅示例，生产环境请使用迁移工具）
    createTable := `
        CREATE TABLE IF NOT EXISTS users (
            id SERIAL PRIMARY KEY,
            name VARCHAR(100) NOT NULL,
            email VARCHAR(100) UNIQUE NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    `
    db.Exec(createTable)

    // 用户API
    userAPI := app.Group("/api/users")
    {
        // 获取所有用户
        userAPI.GET("/", func(c *gin.Context) {
            rows, err := db.Query("SELECT id, name, email, created_at FROM users")
            if err != nil {
                c.JSON(500, gin.H{"error": err.Error()})
                return
            }
            defer rows.Close()

            var users []User
            for rows.Next() {
                var user User
                err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
                if err != nil {
                    c.JSON(500, gin.H{"error": err.Error()})
                    return
                }
                users = append(users, user)
            }

            c.JSON(200, gin.H{
                "data": users,
                "count": len(users),
            })
        })

        // 创建用户
        userAPI.POST("/", func(c *gin.Context) {
            var user User
            if err := c.ShouldBindJSON(&user); err != nil {
                c.JSON(400, gin.H{"error": err.Error()})
                return
            }

            err := db.QueryRow(
                "INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, created_at",
                user.Name, user.Email,
            ).Scan(&user.ID, &user.CreatedAt)

            if err != nil {
                c.JSON(500, gin.H{"error": err.Error()})
                return
            }

            c.JSON(201, gin.H{"data": user})
        })
    }

    app.Run(":8080")
}
```

### 3. 缓存集成

创建 `cache_example.go`：

```go
package main

import (
    "encoding/json"
    "time"
    "github.com/gin-gonic/gin"
    "github.com/your-org/basic/pkg/chi"
    "github.com/your-org/basic/pkg/cache"
)

func main() {
    app := chi.New()

    // 初始化缓存
    cacheClient, err := cache.NewRedisCache(cache.RedisConfig{
        Host:     "localhost",
        Port:     6379,
        Password: "",
        DB:       0,
        PoolSize: 10,
    })
    if err != nil {
        panic(err)
    }

    // 缓存中间件
    func CacheMiddleware(duration time.Duration) gin.HandlerFunc {
        return func(c *gin.Context) {
            if c.Request.Method != "GET" {
                c.Next()
                return
            }

            key := "cache:" + c.Request.URL.Path
            if c.Request.URL.RawQuery != "" {
                key += "?" + c.Request.URL.RawQuery
            }

            // 尝试从缓存获取
            cached, err := cacheClient.Get(c.Request.Context(), key)
            if err == nil && cached != "" {
                var result map[string]interface{}
                if json.Unmarshal([]byte(cached), &result) == nil {
                    c.JSON(200, result)
                    c.Header("X-Cache", "HIT")
                    c.Abort()
                    return
                }
            }

            // 缓存拦截响应
            writer := &CacheWriter{
                ResponseWriter: c.Writer,
                key:           key,
                cache:         cacheClient,
                duration:      duration,
                context:       c.Request.Context(),
            }
            c.Writer = writer
            c.Header("X-Cache", "MISS")
            c.Next()
        }
    }

    // 使用缓存的API
    app.GET("/api/expensive-data", CacheMiddleware(5*time.Minute), func(c *gin.Context) {
        // 模拟耗时操作
        time.Sleep(2 * time.Second)

        c.JSON(200, gin.H{
            "data": "This is expensive data",
            "timestamp": time.Now(),
            "cached_for": "5 minutes",
        })
    })

    // 手动缓存操作
    app.POST("/api/cache", func(c *gin.Context) {
        var req struct {
            Key   string      `json:"key"`
            Value interface{} `json:"value"`
            TTL   int         `json:"ttl"` // seconds
        }

        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }

        valueJSON, _ := json.Marshal(req.Value)
        err := cacheClient.Set(c.Request.Context(), req.Key, string(valueJSON), time.Duration(req.TTL)*time.Second)
        if err != nil {
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }

        c.JSON(200, gin.H{"message": "Cache set successfully"})
    })

    app.GET("/api/cache/:key", func(c *gin.Context) {
        key := c.Param("key")
        value, err := cacheClient.Get(c.Request.Context(), key)
        if err != nil {
            c.JSON(404, gin.H{"error": "Key not found"})
            return
        }

        var result interface{}
        json.Unmarshal([]byte(value), &result)

        c.JSON(200, gin.H{
            "key": key,
            "value": result,
        })
    })

    app.Run(":8080")
}

// 缓存写入器
type CacheWriter struct {
    gin.ResponseWriter
    key      string
    cache    cache.Cache
    duration time.Duration
    context  context.Context
    body     []byte
}

func (cw *CacheWriter) Write(data []byte) (int, error) {
    cw.body = append(cw.body, data...)
    return cw.ResponseWriter.Write(data)
}

func (cw *CacheWriter) WriteString(s string) (int, error) {
    cw.body = append(cw.body, []byte(s)...)
    return cw.ResponseWriter.WriteString(s)
}

func (cw *CacheWriter) WriteHeader(statusCode int) {
    cw.ResponseWriter.WriteHeader(statusCode)

    // 只缓存成功响应
    if statusCode == 200 && len(cw.body) > 0 {
        cw.cache.Set(cw.context, cw.key, string(cw.body), cw.duration)
    }
}
```

### 4. WebSocket 实时通信

创建 `websocket_example.go`：

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/your-org/basic/pkg/chi"
    "github.com/your-org/basic/pkg/chi/websocket"
)

func main() {
    app := chi.New()

    // 创建WebSocket配置
    wsConfig := websocket.Config{
        ReadBufferSize:  1024,
        WriteBufferSize: 1024,
        CheckOrigin: func(r *http.Request) bool {
            return true
        },
    }

    // 创建WebSocket服务器
    wsServer := websocket.NewServer(wsConfig)

    // 添加消息处理器
    wsServer.AddHandler("echo", func(client *websocket.Client, message *websocket.Message) error {
        response := &websocket.Message{
            Type: "echo_response",
            Data: message.Data,
            Timestamp: time.Now(),
        }
        return client.SendJSON(response)
    })

    wsServer.AddHandler("broadcast", func(client *websocket.Client, message *websocket.Message) error {
        // 广播给所有连接的客户端
        return wsServer.Broadcast(&websocket.Message{
            Type: "broadcast_message",
            Data: map[string]interface{}{
                "from": client.ID,
                "message": message.Data,
                "timestamp": time.Now(),
            },
        })
    })

    // WebSocket端点
    app.GET("/ws", wsServer.HandleConnection())

    // 静态文件服务（用于演示）
    app.Static("/static", "./static")

    // 启动WebSocket服务器
    wsServer.Start()

    app.Run(":8080")
}
```

创建 WebSocket 客户端演示页面 `static/websocket.html`：

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Chi WebSocket Demo</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .container { max-width: 800px; margin: 0 auto; }
        .messages { border: 1px solid #ccc; height: 400px; overflow-y: scroll; padding: 10px; margin: 10px 0; }
        .message { margin: 5px 0; padding: 5px; background: #f5f5f5; }
        input[type="text"] { width: 70%; padding: 5px; }
        button { padding: 5px 15px; margin-left: 5px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Chi WebSocket Demo</h1>

        <div>
            <button id="connect">Connect</button>
            <button id="disconnect">Disconnect</button>
            <span id="status">Disconnected</span>
        </div>

        <div class="messages" id="messages"></div>

        <div>
            <input type="text" id="messageInput" placeholder="Enter message...">
            <button id="sendEcho">Send Echo</button>
            <button id="sendBroadcast">Send Broadcast</button>
        </div>
    </div>

    <script>
        let ws = null;
        const messagesDiv = document.getElementById('messages');
        const messageInput = document.getElementById('messageInput');
        const statusSpan = document.getElementById('status');

        document.getElementById('connect').onclick = function() {
            ws = new WebSocket('ws://localhost:8080/ws');

            ws.onopen = function() {
                statusSpan.textContent = 'Connected';
                addMessage('Connected to WebSocket server', 'system');
            };

            ws.onmessage = function(event) {
                const message = JSON.parse(event.data);
                addMessage(`[${message.type}] ${JSON.stringify(message.data)}`, 'received');
            };

            ws.onclose = function() {
                statusSpan.textContent = 'Disconnected';
                addMessage('Disconnected from WebSocket server', 'system');
            };

            ws.onerror = function(error) {
                addMessage('WebSocket error: ' + error, 'error');
            };
        };

        document.getElementById('disconnect').onclick = function() {
            if (ws) {
                ws.close();
            }
        };

        document.getElementById('sendEcho').onclick = function() {
            if (ws && ws.readyState === WebSocket.OPEN) {
                const message = {
                    type: 'echo',
                    data: messageInput.value,
                    timestamp: new Date().toISOString()
                };
                ws.send(JSON.stringify(message));
                addMessage(`[echo] ${messageInput.value}`, 'sent');
                messageInput.value = '';
            }
        };

        document.getElementById('sendBroadcast').onclick = function() {
            if (ws && ws.readyState === WebSocket.OPEN) {
                const message = {
                    type: 'broadcast',
                    data: messageInput.value,
                    timestamp: new Date().toISOString()
                };
                ws.send(JSON.stringify(message));
                addMessage(`[broadcast] ${messageInput.value}`, 'sent');
                messageInput.value = '';
            }
        };

        messageInput.addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                document.getElementById('sendEcho').click();
            }
        });

        function addMessage(message, type) {
            const div = document.createElement('div');
            div.className = 'message ' + type;
            div.textContent = new Date().toLocaleTimeString() + ' - ' + message;
            messagesDiv.appendChild(div);
            messagesDiv.scrollTop = messagesDiv.scrollHeight;
        }
    </script>
</body>
</html>
```

## 部署指南

### 1. Docker 部署

创建 `Dockerfile`：

```dockerfile
# 构建阶段
FROM golang:1.19-alpine AS builder

WORKDIR /app

# 复制go mod文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# 运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

# 从构建阶段复制二进制文件
COPY --from=builder /app/main .

# 复制配置文件
COPY config.yaml .

EXPOSE 8080

CMD ["./main"]
```

创建 `docker-compose.yml`：

```yaml
version: '3.8'

services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - CHI_ENV=production
    depends_on:
      - postgres
      - redis
    networks:
      - chi-network

  postgres:
    image: postgres:13
    environment:
      POSTGRES_DB: chi_db
      POSTGRES_USER: chi_user
      POSTGRES_PASSWORD: chi_password
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    networks:
      - chi-network

  redis:
    image: redis:6-alpine
    ports:
      - "6379:6379"
    networks:
      - chi-network

  jaeger:
    image: jaegertracing/all-in-one:1.35
    ports:
      - "16686:16686"
      - "14268:14268"
    environment:
      - COLLECTOR_OTLP_ENABLED=true
    networks:
      - chi-network

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    networks:
      - chi-network

volumes:
  postgres_data:

networks:
  chi-network:
    driver: bridge
```

部署命令：

```bash
# 构建和启动服务
docker-compose up --build

# 后台运行
docker-compose up -d

# 查看日志
docker-compose logs -f app

# 停止服务
docker-compose down
```

### 2. Kubernetes 部署

创建 `k8s/deployment.yaml`：

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
        - name: CHI_ENV
          value: "production"
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: chi-secrets
              key: database-url
        resources:
          requests:
            memory: "256Mi"
            cpu: "200m"
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
            path: /health
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
```

## 监控和调试

### 1. 健康检查端点

```go
// 在主应用中添加详细的健康检查
app.GET("/health", func(c *gin.Context) {
    status := "healthy"
    checks := map[string]interface{}{
        "database": checkDatabase(),
        "redis":    checkRedis(),
        "memory":   getMemoryUsage(),
        "uptime":   time.Since(startTime).String(),
    }

    // 检查所有依赖是否健康
    for _, check := range checks {
        if check == "unhealthy" {
            status = "unhealthy"
            break
        }
    }

    httpStatus := 200
    if status == "unhealthy" {
        httpStatus = 503
    }

    c.JSON(httpStatus, gin.H{
        "status": status,
        "checks": checks,
        "timestamp": time.Now(),
        "version": "1.0.0",
    })
})

func checkDatabase() string {
    if err := db.Ping(); err != nil {
        return "unhealthy"
    }
    return "healthy"
}

func checkRedis() string {
    if err := redisClient.Ping().Err(); err != nil {
        return "unhealthy"
    }
    return "healthy"
}

func getMemoryUsage() string {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    return fmt.Sprintf("%.2f MB", float64(m.Alloc)/1024/1024)
}
```

### 2. 指标监控

```go
// 添加Prometheus指标端点
app.GET("/metrics", gin.WrapH(promhttp.Handler()))

// 自定义指标
var (
    httpRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "endpoint", "status"},
    )

    httpRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "http_request_duration_seconds",
            Help: "HTTP request duration in seconds",
        },
        []string{"method", "endpoint"},
    )
)

func init() {
    prometheus.MustRegister(httpRequestsTotal)
    prometheus.MustRegister(httpRequestDuration)
}

// 指标中间件
func PrometheusMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()

        c.Next()

        duration := time.Since(start)
        status := fmt.Sprintf("%d", c.Writer.Status())

        httpRequestsTotal.WithLabelValues(c.Request.Method, c.FullPath(), status).Inc()
        httpRequestDuration.WithLabelValues(c.Request.Method, c.FullPath()).Observe(duration.Seconds())
    }
}
```

## 故障排除

### 常见问题

1. **端口被占用**
   ```bash
   # 查找占用端口的进程
   lsof -i :8080

   # 杀死进程
   kill -9 <PID>
   ```

2. **数据库连接失败**
   - 检查数据库服务是否运行
   - 验证连接参数
   - 检查网络连通性

3. **内存不足**
   ```bash
   # 监控内存使用
   free -h

   # 查看进程内存使用
   ps aux --sort=-%mem | head
   ```

4. **性能问题**
   ```bash
   # 使用pprof分析性能
   go tool pprof http://localhost:8080/debug/pprof/profile

   # 查看堆内存
   go tool pprof http://localhost:8080/debug/pprof/heap
   ```

### 调试技巧

1. **启用调试日志**
   ```go
   gin.SetMode(gin.DebugMode)
   ```

2. **添加调试端点**
   ```go
   if gin.Mode() == gin.DebugMode {
       app.GET("/debug/vars", gin.WrapH(expvar.Handler()))
       app.GET("/debug/pprof/*any", gin.WrapH(http.DefaultServeMux))
   }
   ```

3. **请求追踪**
   ```go
   app.Use(func(c *gin.Context) {
       requestID := uuid.New().String()
       c.Header("X-Request-ID", requestID)
       log.Printf("Request %s: %s %s", requestID, c.Request.Method, c.Request.URL.Path)
       c.Next()
   })
   ```

## 下一步

恭喜！您已经成功创建并运行了一个基于 Chi 框架的应用。接下来，您可以：

1. **深入学习**: 阅读详细的[架构设计文档](./architecture.md)
2. **功能扩展**: 探索[插件系统](./plugins.md)和[事件驱动架构](./events.md)
3. **生产部署**: 了解[服务治理](./governance.md)和[可观测性](./observability.md)
4. **实时通信**: 学习[WebSocket开发](./websocket.md)

Chi 框架提供了丰富的企业级功能，帮助您构建高性能、可扩展的微服务应用。通过逐步学习和实践，您将能够充分发挥框架的强大能力。