# Chi API 网关系统

Chi 框架内置了功能强大的 API 网关系统，为微服务架构提供统一的服务入口、流量管理、安全认证和协议转换等核心功能。

## 🏗️ 网关架构

### 整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│                          Chi API Gateway                        │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │    HTTP     │  │   gRPC     │  │  WebSocket  │  Protocol    │
│  │   Server    │  │   Server   │  │    Server   │  Adapters    │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                   Router Engine                          │   │
│  │  ┌──────────────┐ ┌──────────────┐ ┌─────────────────┐  │   │
│  │  │    Route     │ │   Matcher    │ │   URL Rewrite   │  │   │
│  │  │   Registry   │ │   Engine     │ │     Engine      │  │   │
│  │  └──────────────┘ └──────────────┘ └─────────────────┘  │   │
│  └──────────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                 Middleware Pipeline                      │   │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌──────────────┐   │   │
│  │  │  Auth   │ │  Rate   │ │Circuit  │ │   Request    │   │   │
│  │  │Middleware│ │Limiting │ │Breaker  │ │Transformation│   │   │
│  │  └─────────┘ └─────────┘ └─────────┘ └──────────────┘   │   │
│  └──────────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                 Service Discovery                        │   │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────────┐    │   │
│  │  │   Consul    │ │    Etcd     │ │      Nacos      │    │   │
│  │  │  Adapter    │ │   Adapter   │ │     Adapter     │    │   │
│  │  └─────────────┘ └─────────────┘ └─────────────────┘    │   │
│  └──────────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                 Load Balancer                            │   │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────────┐    │   │
│  │  │Round Robin  │ │  Weighted   │ │ Consistent Hash │    │   │
│  │  │             │ │ Round Robin │ │                 │    │   │
│  │  └─────────────┘ └─────────────┘ └─────────────────┘    │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### 核心组件

#### 1. Protocol Adapters（协议适配器）
- **HTTP Adapter**：处理 RESTful API 请求
- **gRPC Adapter**：处理 gRPC 服务调用
- **WebSocket Adapter**：处理实时通信连接

#### 2. Router Engine（路由引擎）
- **Route Registry**：路由规则注册表
- **Matcher Engine**：路径匹配引擎
- **URL Rewrite Engine**：URL 重写引擎

#### 3. Middleware Pipeline（中间件管道）
- **Authentication**：身份认证
- **Authorization**：权限控制
- **Rate Limiting**：流量限制
- **Circuit Breaker**：熔断保护

## 🚀 核心功能

### 1. 统一服务入口

```go
package main

import (
    "github.com/your-org/basic/pkg/chi/gateway"
    "github.com/your-org/basic/pkg/chi/config"
)

func main() {
    // 创建网关实例
    gw := gateway.New(&gateway.Config{
        Port: 8080,
        TLS: &gateway.TLSConfig{
            Enabled:  true,
            CertFile: "cert.pem",
            KeyFile:  "key.pem",
        },
    })

    // 注册服务路由
    gw.AddRoute(&gateway.Route{
        Path:        "/api/users/*",
        Service:     "user-service",
        Method:      "GET,POST,PUT,DELETE",
        StripPrefix: "/api",
    })

    gw.AddRoute(&gateway.Route{
        Path:     "/api/orders/*",
        Service:  "order-service",
        Upstream: "consul://localhost:8500",
    })

    // 启动网关
    gw.Start()
}
```

### 2. 动态路由配置

```yaml
# gateway.yaml
gateway:
  routes:
    - path: "/api/v1/users/*"
      service: "user-service"
      methods: ["GET", "POST", "PUT", "DELETE"]
      strip_prefix: "/api/v1"
      upstream:
        discovery: "consul://localhost:8500"
        load_balancer: "round_robin"
        health_check:
          enabled: true
          interval: "30s"
          timeout: "5s"
          path: "/health"

    - path: "/api/v1/orders/*"
      service: "order-service"
      methods: ["GET", "POST"]
      rewrite:
        from: "/api/v1/orders/(.*)"
        to: "/orders/$1"
      middleware:
        - "auth"
        - "rate-limit"
        - "circuit-breaker"

    - path: "/ws/*"
      service: "websocket-service"
      protocol: "websocket"
      upgrade: true
```

### 3. 智能负载均衡

```go
// 负载均衡策略配置
type LoadBalancerConfig struct {
    Strategy string                 `yaml:"strategy"`
    Options  map[string]interface{} `yaml:"options"`
}

// 支持的负载均衡策略
const (
    RoundRobin         = "round_robin"
    WeightedRoundRobin = "weighted_round_robin"
    LeastConnections   = "least_connections"
    ConsistentHash     = "consistent_hash"
    IPHash            = "ip_hash"
    Random            = "random"
)

// 配置示例
gw.AddRoute(&gateway.Route{
    Path:    "/api/data/*",
    Service: "data-service",
    LoadBalancer: &gateway.LoadBalancerConfig{
        Strategy: gateway.ConsistentHash,
        Options: map[string]interface{}{
            "hash_key": "user_id", // 基于用户ID进行一致性哈希
            "replicas": 150,       // 虚拟节点数量
        },
    },
})
```

### 4. 协议转换

```go
// HTTP 到 gRPC 转换
gw.AddRoute(&gateway.Route{
    Path:     "/api/grpc/users/*",
    Service:  "user-grpc-service",
    Protocol: "grpc",
    Transform: &gateway.ProtocolTransform{
        From: "http",
        To:   "grpc",
        Mapping: map[string]string{
            "GET /users/:id":    "UserService.GetUser",
            "POST /users":       "UserService.CreateUser",
            "PUT /users/:id":    "UserService.UpdateUser",
            "DELETE /users/:id": "UserService.DeleteUser",
        },
    },
})

// WebSocket 代理
gw.AddRoute(&gateway.Route{
    Path:     "/ws/chat/*",
    Service:  "chat-service",
    Protocol: "websocket",
    WebSocket: &gateway.WebSocketConfig{
        BufferSize:      1024,
        MaxMessageSize:  65536,
        ReadTimeout:     "60s",
        WriteTimeout:    "10s",
        EnableBroadcast: true,
    },
})
```

## 🛡️ 安全特性

### 1. 身份认证

```go
// JWT 认证配置
gw.UseMiddleware(&gateway.JWTAuth{
    Secret:    "your-jwt-secret",
    Algorithm: "HS256",
    Claims: gateway.JWTClaims{
        Issuer:   "chi-gateway",
        Audience: "chi-services",
        TTL:      time.Hour * 24,
    },
    Exclude: []string{
        "/api/auth/login",
        "/api/auth/register",
        "/health",
    },
})

// OAuth2 认证配置
gw.UseMiddleware(&gateway.OAuth2Auth{
    Provider: "google",
    ClientID: "your-client-id",
    ClientSecret: "your-client-secret",
    Scopes: []string{"openid", "email", "profile"},
    RedirectURL: "https://your-domain.com/auth/callback",
})

// API Key 认证
gw.UseMiddleware(&gateway.APIKeyAuth{
    Header:   "X-API-Key",
    Storage:  "redis://localhost:6379",
    RateLimit: &gateway.RateLimit{
        Requests: 1000,
        Window:   time.Hour,
    },
})
```

### 2. 权限控制

```go
// RBAC 权限控制
gw.UseMiddleware(&gateway.RBAC{
    Rules: []gateway.RBACRule{
        {
            Path:   "/api/admin/*",
            Roles:  []string{"admin"},
            Action: "allow",
        },
        {
            Path:   "/api/users/:id",
            Roles:  []string{"user", "admin"},
            Action: "allow",
            Condition: func(ctx *gateway.Context) bool {
                userID := ctx.GetClaim("user_id")
                resourceID := ctx.Param("id")
                return userID == resourceID || ctx.HasRole("admin")
            },
        },
    },
})

// IP 白名单/黑名单
gw.UseMiddleware(&gateway.IPFilter{
    Whitelist: []string{
        "192.168.1.0/24",
        "10.0.0.0/8",
    },
    Blacklist: []string{
        "192.168.1.100",
    },
})
```

## ⚡ 性能优化

### 1. 缓存策略

```go
// 响应缓存
gw.UseMiddleware(&gateway.Cache{
    Storage: "redis://localhost:6379",
    Rules: []gateway.CacheRule{
        {
            Path:   "/api/public/*",
            TTL:    time.Hour,
            Vary:   []string{"Accept", "Accept-Language"},
            Keys:   []string{"path", "query"},
        },
        {
            Path:      "/api/users/:id",
            TTL:       time.Minute * 10,
            Condition: func(ctx *gateway.Context) bool {
                return ctx.Method == "GET"
            },
        },
    },
})

// 请求去重
gw.UseMiddleware(&gateway.Deduplication{
    Window:   time.Second * 5,
    KeyFunc: func(ctx *gateway.Context) string {
        return fmt.Sprintf("%s:%s:%s",
            ctx.ClientIP(),
            ctx.Method,
            ctx.Path)
    },
})
```

### 2. 连接池管理

```go
// HTTP 连接池配置
gw.SetHTTPClient(&gateway.HTTPClientConfig{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 10,
    IdleConnTimeout:     time.Minute * 5,
    TLSHandshakeTimeout: time.Second * 10,
    ResponseHeaderTimeout: time.Second * 30,
})

// gRPC 连接池配置
gw.SetGRPCClient(&gateway.GRPCClientConfig{
    MaxConnections:      20,
    MaxIdleConnections:  5,
    ConnectionTimeout:   time.Second * 10,
    KeepAliveTime:      time.Minute * 5,
    KeepAliveTimeout:   time.Second * 20,
})
```

## 📊 监控与可观测性

### 1. 指标收集

```go
// Prometheus 指标
gw.UseMiddleware(&gateway.Metrics{
    Namespace: "chi_gateway",
    Subsystem: "http",
    Metrics: []gateway.Metric{
        {
            Name: "requests_total",
            Type: "counter",
            Help: "Total number of HTTP requests",
            Labels: []string{"method", "path", "status"},
        },
        {
            Name: "request_duration_seconds",
            Type: "histogram",
            Help: "HTTP request duration in seconds",
            Buckets: []float64{0.1, 0.5, 1.0, 2.5, 5.0, 10.0},
        },
    },
})
```

### 2. 分布式追踪

```go
// OpenTelemetry 追踪
gw.UseMiddleware(&gateway.Tracing{
    ServiceName: "chi-gateway",
    Sampler: &gateway.TracingSampler{
        Type: "probabilistic",
        Rate: 0.1, // 10% 采样率
    },
    Exporter: &gateway.TracingExporter{
        Type:     "jaeger",
        Endpoint: "http://localhost:14268/api/traces",
    },
})
```

## 🔧 高级配置

### 1. 多环境配置

```go
// 环境特定配置
type GatewayConfig struct {
    Environment string `yaml:"environment"`

    Development *gateway.Config `yaml:"development,omitempty"`
    Staging     *gateway.Config `yaml:"staging,omitempty"`
    Production  *gateway.Config `yaml:"production,omitempty"`
}

func LoadConfig() *gateway.Config {
    env := os.Getenv("CHI_ENV")
    config := &GatewayConfig{}

    // 加载配置文件
    data, _ := ioutil.ReadFile("gateway.yaml")
    yaml.Unmarshal(data, config)

    switch env {
    case "production":
        return config.Production
    case "staging":
        return config.Staging
    default:
        return config.Development
    }
}
```

### 2. 热更新配置

```go
// 配置热更新
gw.EnableHotReload(&gateway.HotReloadConfig{
    ConfigPath:   "gateway.yaml",
    WatchInterval: time.Second * 30,
    OnReload: func(newConfig *gateway.Config) error {
        log.Info("Gateway configuration reloaded")
        return nil
    },
})

// 动态路由更新
gw.EnableDynamicRouting(&gateway.DynamicRoutingConfig{
    Source: "consul://localhost:8500/gateway/routes",
    Format: "yaml",
    WatchInterval: time.Second * 10,
})
```

## 🚀 部署配置

### 1. Docker 部署

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN go build -o gateway ./cmd/gateway

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/gateway .
COPY --from=builder /app/configs ./configs

EXPOSE 8080 8443
CMD ["./gateway"]
```

### 2. Kubernetes 部署

```yaml
# gateway-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: chi-gateway
spec:
  replicas: 3
  selector:
    matchLabels:
      app: chi-gateway
  template:
    metadata:
      labels:
        app: chi-gateway
    spec:
      containers:
      - name: gateway
        image: chi/gateway:v1.0.0
        ports:
        - containerPort: 8080
        - containerPort: 8443
        env:
        - name: CHI_ENV
          value: "production"
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
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
  name: chi-gateway-service
spec:
  selector:
    app: chi-gateway
  ports:
  - name: http
    port: 80
    targetPort: 8080
  - name: https
    port: 443
    targetPort: 8443
  type: LoadBalancer
```

## 📈 最佳实践

### 1. 路由设计
- 使用语义化的路径设计
- 合理设置路由优先级
- 避免路由冲突和歧义

### 2. 性能优化
- 启用适当的缓存策略
- 配置合理的超时时间
- 使用连接池复用连接

### 3. 安全加固
- 实施多层认证和授权
- 配置适当的限流策略
- 启用请求日志和审计

### 4. 监控告警
- 设置关键指标阈值
- 配置异常告警通知
- 定期分析性能趋势

Chi API 网关为微服务架构提供了统一、高性能、安全的服务入口，通过丰富的中间件和插件系统，可以灵活适应各种业务场景需求。