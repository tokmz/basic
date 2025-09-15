# Chi Framework API 参考

本文档提供 Chi 框架的完整 API 参考，包括所有公开接口、方法签名、参数说明和使用示例。

## 目录

- [核心应用 API](#核心应用-api)
- [微服务 API](#微服务-api)
- [网关 API](#网关-api)
- [配置 API](#配置-api)
- [插件系统 API](#插件系统-api)
- [服务发现 API](#服务发现-api)
- [服务治理 API](#服务治理-api)
- [可观测性 API](#可观测性-api)
- [WebSocket API](#websocket-api)
- [事件系统 API](#事件系统-api)

---

## 核心应用 API

### chi.New()

创建一个新的 Chi 应用实例。

```go
func New(options ...Option) *App
```

**参数**
- `options` - 可选配置参数

**返回值**
- `*App` - Chi 应用实例

**示例**
```go
app := chi.New()

// 带配置的创建方式
app := chi.New(
    chi.WithPort(8080),
    chi.WithTLS("cert.pem", "key.pem"),
)
```

### chi.NewWithConfig()

使用配置文件创建 Chi 应用实例。

```go
func NewWithConfig(config *Config) *App
```

**参数**
- `config` - 配置对象

**返回值**
- `*App` - Chi 应用实例

**示例**
```go
config := &chi.Config{
    Port: 8080,
    Debug: true,
}
app := chi.NewWithConfig(config)
```

### App 方法

#### HTTP 路由方法

```go
func (app *App) GET(path string, handler gin.HandlerFunc) *App
func (app *App) POST(path string, handler gin.HandlerFunc) *App
func (app *App) PUT(path string, handler gin.HandlerFunc) *App
func (app *App) DELETE(path string, handler gin.HandlerFunc) *App
func (app *App) PATCH(path string, handler gin.HandlerFunc) *App
func (app *App) HEAD(path string, handler gin.HandlerFunc) *App
func (app *App) OPTIONS(path string, handler gin.HandlerFunc) *App
```

**参数**
- `path` - 路由路径，支持路径参数如 `/users/:id`
- `handler` - 处理函数

**示例**
```go
app.GET("/users/:id", func(c *gin.Context) {
    id := c.Param("id")
    c.JSON(200, gin.H{"user_id": id})
})
```

#### 路由组

```go
func (app *App) Group(path string) *gin.RouterGroup
```

**参数**
- `path` - 组路径前缀

**返回值**
- `*gin.RouterGroup` - 路由组实例

**示例**
```go
v1 := app.Group("/api/v1")
{
    v1.GET("/users", getUsersHandler)
    v1.POST("/users", createUserHandler)
}
```

#### 中间件

```go
func (app *App) Use(middleware ...gin.HandlerFunc) *App
```

**参数**
- `middleware` - 中间件函数列表

**示例**
```go
app.Use(gin.Logger())
app.Use(gin.Recovery())
app.Use(chi.CORS())
```

#### 启动服务

```go
func (app *App) Run(addr ...string) error
func (app *App) RunTLS(addr, certFile, keyFile string) error
```

**参数**
- `addr` - 监听地址，默认 `:8080`
- `certFile` - TLS 证书文件路径
- `keyFile` - TLS 私钥文件路径

**示例**
```go
// HTTP 启动
app.Run(":8080")

// HTTPS 启动
app.RunTLS(":8443", "cert.pem", "key.pem")
```

---

## 微服务 API

### microservice.New()

创建微服务实例。

```go
func New(serviceName string, config *Config) *Service
```

**参数**
- `serviceName` - 服务名称
- `config` - 微服务配置

**返回值**
- `*Service` - 微服务实例

**示例**
```go
service := microservice.New("user-service", &microservice.Config{
    Port: 8080,
    Registry: "consul://localhost:8500",
    ConfigCenter: "consul://localhost:8500",
})
```

### Service 方法

#### 注册 gRPC 服务

```go
func (s *Service) RegisterGRPC(serviceImpl interface{}) error
```

**参数**
- `serviceImpl` - gRPC 服务实现

**示例**
```go
service.RegisterGRPC(&userServiceImpl{})
```

#### HTTP 路由

```go
func (s *Service) GET(path string, handler gin.HandlerFunc) *Service
func (s *Service) POST(path string, handler gin.HandlerFunc) *Service
// ... 其他 HTTP 方法
```

#### 服务治理

```go
func (s *Service) UseCircuitBreaker(config ...CircuitBreakerConfig) *Service
func (s *Service) UseRateLimit(rps int) *Service
func (s *Service) UseTracing() *Service
func (s *Service) UseMetrics() *Service
```

**示例**
```go
service.UseCircuitBreaker()
service.UseRateLimit(100) // 100 req/s
service.UseTracing()
```

#### 启动微服务

```go
func (s *Service) Start() error
func (s *Service) Stop() error
```

**示例**
```go
service.Start()
```

---

## 网关 API

### gateway.New()

创建 API 网关实例。

```go
func New(config *Config) *Gateway
```

**参数**
- `config` - 网关配置

**示例**
```go
gw := gateway.New(&gateway.Config{
    Port: 8080,
    TLS: &gateway.TLSConfig{
        Enabled:  true,
        CertFile: "cert.pem",
        KeyFile:  "key.pem",
    },
})
```

### Gateway 方法

#### 添加路由

```go
func (gw *Gateway) AddRoute(route *Route) error
```

**参数**
- `route` - 路由配置

**Route 结构**
```go
type Route struct {
    Path         string   // 路由路径
    Service      string   // 目标服务名
    Methods      []string // 允许的HTTP方法
    StripPrefix  string   // 移除的路径前缀
    LoadBalancer string   // 负载均衡策略
    Timeout      time.Duration // 超时时间
    Retries      int      // 重试次数
}
```

**示例**
```go
gw.AddRoute(&gateway.Route{
    Path:         "/api/users/*",
    Service:      "user-service",
    Methods:      []string{"GET", "POST", "PUT", "DELETE"},
    StripPrefix:  "/api",
    LoadBalancer: "round_robin",
    Timeout:      30 * time.Second,
    Retries:      3,
})
```

#### 中间件

```go
func (gw *Gateway) UseAuth(provider string) *Gateway
func (gw *Gateway) UseRateLimit(rps int) *Gateway
func (gw *Gateway) UseCircuitBreaker() *Gateway
func (gw *Gateway) UseCORS() *Gateway
```

**示例**
```go
gw.UseAuth("jwt")
gw.UseRateLimit(1000)
gw.UseCircuitBreaker()
```

---

## 配置 API

### config.Load()

加载配置文件。

```go
func Load(filename string) *Config
```

**参数**
- `filename` - 配置文件路径

**返回值**
- `*Config` - 配置对象

**示例**
```go
cfg := config.Load("config.yaml")
```

### Config 结构

```go
type Config struct {
    // 基础配置
    Port     int    `yaml:"port"`
    Host     string `yaml:"host"`
    Debug    bool   `yaml:"debug"`

    // TLS 配置
    TLS struct {
        Enabled  bool   `yaml:"enabled"`
        CertFile string `yaml:"cert_file"`
        KeyFile  string `yaml:"key_file"`
    } `yaml:"tls"`

    // 数据库配置
    Database struct {
        Driver   string `yaml:"driver"`
        Host     string `yaml:"host"`
        Port     int    `yaml:"port"`
        Username string `yaml:"username"`
        Password string `yaml:"password"`
        Database string `yaml:"database"`
    } `yaml:"database"`

    // Redis 配置
    Redis struct {
        Host     string `yaml:"host"`
        Port     int    `yaml:"port"`
        Password string `yaml:"password"`
        DB       int    `yaml:"db"`
    } `yaml:"redis"`

    // 服务发现配置
    Discovery struct {
        Provider string `yaml:"provider"` // consul, etcd, nacos
        Address  string `yaml:"address"`
    } `yaml:"discovery"`

    // 监控配置
    Monitoring struct {
        Metrics struct {
            Enabled bool   `yaml:"enabled"`
            Path    string `yaml:"path"`
        } `yaml:"metrics"`

        Tracing struct {
            Enabled  bool   `yaml:"enabled"`
            Endpoint string `yaml:"endpoint"`
            Sampler  float64 `yaml:"sampler"`
        } `yaml:"tracing"`
    } `yaml:"monitoring"`
}
```

---

## 插件系统 API

### plugins.Register()

注册插件。

```go
func Register(name string, plugin Plugin) error
```

**参数**
- `name` - 插件名称
- `plugin` - 插件实现

### Plugin 接口

```go
type Plugin interface {
    Name() string
    Version() string
    Init(config map[string]interface{}) error
    Start() error
    Stop() error
}
```

**示例**
```go
type MyPlugin struct{}

func (p *MyPlugin) Name() string { return "my-plugin" }
func (p *MyPlugin) Version() string { return "1.0.0" }
func (p *MyPlugin) Init(config map[string]interface{}) error {
    // 初始化逻辑
    return nil
}
func (p *MyPlugin) Start() error {
    // 启动逻辑
    return nil
}
func (p *MyPlugin) Stop() error {
    // 停止逻辑
    return nil
}

// 注册插件
plugins.Register("my-plugin", &MyPlugin{})
```

---

## 服务发现 API

### discovery.NewConsul()

创建 Consul 服务发现客户端。

```go
func NewConsul(address string) (*ConsulClient, error)
```

### discovery.NewEtcd()

创建 Etcd 服务发现客户端。

```go
func NewEtcd(endpoints []string) (*EtcdClient, error)
```

### Client 接口

```go
type Client interface {
    Register(service *ServiceInfo) error
    Deregister(serviceID string) error
    Discover(serviceName string) ([]*ServiceInfo, error)
    Watch(serviceName string) (<-chan []*ServiceInfo, error)
}
```

**ServiceInfo 结构**
```go
type ServiceInfo struct {
    ID       string            `json:"id"`
    Name     string            `json:"name"`
    Address  string            `json:"address"`
    Port     int               `json:"port"`
    Tags     []string          `json:"tags"`
    Meta     map[string]string `json:"meta"`
    Health   string            `json:"health"`
}
```

---

## 服务治理 API

### 熔断器

```go
type CircuitBreaker interface {
    Call(fn func() error) error
    State() State
    Reset()
}

func NewCircuitBreaker(config *CircuitBreakerConfig) CircuitBreaker
```

**配置结构**
```go
type CircuitBreakerConfig struct {
    MaxRequests    uint32        // 半开状态最大请求数
    Interval       time.Duration // 统计时间窗口
    Timeout        time.Duration // 超时时间
    ReadyToTrip    func(counts Counts) bool // 熔断判断函数
    OnStateChange  func(name string, from State, to State) // 状态变化回调
}
```

### 限流器

```go
type RateLimiter interface {
    Allow() bool
    Wait(ctx context.Context) error
}

func NewTokenBucket(rate float64, capacity int) RateLimiter
func NewLeakyBucket(rate float64, capacity int) RateLimiter
```

### 负载均衡

```go
type LoadBalancer interface {
    Select(services []*ServiceInfo) *ServiceInfo
}

func NewRoundRobin() LoadBalancer
func NewWeightedRoundRobin() LoadBalancer
func NewLeastConnections() LoadBalancer
func NewConsistentHash() LoadBalancer
```

---

## 可观测性 API

### 指标监控

```go
type Metrics interface {
    Counter(name string) Counter
    Gauge(name string) Gauge
    Histogram(name string) Histogram
    Summary(name string) Summary
}

func NewPrometheusMetrics() Metrics
```

### 分布式追踪

```go
type Tracer interface {
    StartSpan(operationName string) Span
    StartSpanFromContext(ctx context.Context, operationName string) (Span, context.Context)
}

func NewJaegerTracer(config *JaegerConfig) (Tracer, error)
```

### 日志

```go
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    Fatal(msg string, fields ...Field)
}

func NewZapLogger(config *LogConfig) Logger
```

---

## WebSocket API

### websocket.NewHub()

创建 WebSocket 连接管理中心。

```go
func NewHub(config *HubConfig) *Hub
```

### Hub 方法

```go
func (h *Hub) Register(conn *Connection) error
func (h *Hub) Unregister(conn *Connection) error
func (h *Hub) Broadcast(message []byte) error
func (h *Hub) BroadcastToRoom(room string, message []byte) error
func (h *Hub) SendToUser(userID string, message []byte) error
```

### Connection 结构

```go
type Connection struct {
    ID     string
    UserID string
    Rooms  []string
    Conn   *websocket.Conn
}
```

---

## 事件系统 API

### events.NewEventBus()

创建事件总线。

```go
func NewEventBus(config *EventBusConfig) *EventBus
```

### EventBus 方法

```go
func (eb *EventBus) Publish(topic string, event interface{}) error
func (eb *EventBus) Subscribe(topic string, handler EventHandler) error
func (eb *EventBus) Unsubscribe(topic string, handler EventHandler) error
```

### EventHandler 接口

```go
type EventHandler interface {
    Handle(event interface{}) error
}
```

**示例**
```go
// 订阅事件
eb.Subscribe("user.created", func(event interface{}) error {
    user := event.(*User)
    fmt.Printf("New user created: %s\n", user.Name)
    return nil
})

// 发布事件
eb.Publish("user.created", &User{Name: "John Doe"})
```

---

## 错误处理

### 错误类型

```go
type Error struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}

func (e *Error) Error() string
```

### 常用错误

```go
var (
    ErrServiceNotFound     = &Error{Code: 404, Message: "Service not found"}
    ErrInvalidConfig       = &Error{Code: 400, Message: "Invalid configuration"}
    ErrConnectionFailed    = &Error{Code: 500, Message: "Connection failed"}
    ErrCircuitBreakerOpen  = &Error{Code: 503, Message: "Circuit breaker is open"}
    ErrRateLimitExceeded   = &Error{Code: 429, Message: "Rate limit exceeded"}
)
```

---

## 版本兼容性

| API 版本 | Chi 版本 | Go 版本要求 | 状态 |
|----------|----------|-------------|------|
| v1.0 | 1.0.x | >= 1.19 | 开发中 |

---

**最后更新**: 2025-09-15
**文档版本**: v1.0.0