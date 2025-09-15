# 微服务最佳实践

## 概述

本文档总结了使用 Chi 框架开发微服务的最佳实践，包括架构设计、代码组织、性能优化、安全考虑、运维管理等方面的经验和建议。

## 架构设计原则

### 1. 单一职责原则

每个微服务应该专注于一个业务领域，具有明确的边界和职责。

```go
// ✅ 好的例子：用户服务只处理用户相关的业务
type UserService struct {
    repo UserRepository
}

func (s *UserService) CreateUser(ctx context.Context, user *User) error {
    // 只处理用户创建逻辑
    return s.repo.Create(ctx, user)
}

func (s *UserService) GetUser(ctx context.Context, id string) (*User, error) {
    // 只处理用户查询逻辑
    return s.repo.GetByID(ctx, id)
}

// ❌ 不好的例子：一个服务处理多个不相关的业务
type BusinessService struct {
    userRepo    UserRepository
    orderRepo   OrderRepository
    paymentRepo PaymentRepository
}
```

### 2. 数据库分离

每个微服务应该拥有自己的数据库，避免直接访问其他服务的数据库。

```go
// ✅ 好的架构：每个服务有独立的数据库
type UserService struct {
    userDB    *sql.DB  // 用户服务专用数据库
    userCache cache.Cache
}

type OrderService struct {
    orderDB    *sql.DB  // 订单服务专用数据库
    orderCache cache.Cache
}

// 通过API调用获取其他服务的数据
func (s *OrderService) CreateOrder(ctx context.Context, order *Order) error {
    // 通过API调用验证用户
    user, err := s.userClient.GetUser(ctx, order.UserID)
    if err != nil {
        return err
    }

    // 创建订单
    return s.orderRepo.Create(ctx, order)
}
```

### 3. API版本管理

设计向后兼容的API，合理管理版本演进。

```go
// API版本管理最佳实践
func (s *UserService) RegisterRoutes(router *gin.Engine) {
    // v1 API - 稳定版本
    v1 := router.Group("/api/v1")
    {
        v1.GET("/users/:id", s.GetUserV1)
        v1.POST("/users", s.CreateUserV1)
    }

    // v2 API - 新版本，保持向后兼容
    v2 := router.Group("/api/v2")
    {
        v2.GET("/users/:id", s.GetUserV2)
        v2.POST("/users", s.CreateUserV2)
        v2.GET("/users/:id/profile", s.GetUserProfile) // 新功能
    }
}

// 保持向后兼容的响应结构
type UserResponseV1 struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

type UserResponseV2 struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    Avatar    string    `json:"avatar,omitempty"`    // 新字段，可选
    CreatedAt time.Time `json:"created_at"`          // 新字段
}
```

## 代码组织最佳实践

### 1. 项目结构

```
microservice/
├── cmd/
│   └── service/
│       └── main.go              # 应用入口
├── internal/                    # 私有代码
│   ├── handler/                 # HTTP处理器
│   │   ├── user.go
│   │   └── health.go
│   ├── service/                 # 业务逻辑层
│   │   ├── user.go
│   │   └── interfaces.go
│   ├── repository/              # 数据访问层
│   │   ├── user.go
│   │   └── interfaces.go
│   ├── model/                   # 数据模型
│   │   ├── user.go
│   │   └── common.go
│   ├── config/                  # 配置管理
│   │   └── config.go
│   └── middleware/              # 中间件
│       ├── auth.go
│       └── logging.go
├── pkg/                         # 可导出的包
│   ├── client/                  # 客户端代码
│   └── errors/                  # 错误定义
├── api/                         # API定义
│   ├── openapi/
│   └── proto/
├── deployments/                 # 部署配置
│   ├── docker/
│   └── k8s/
├── test/                        # 测试
│   ├── integration/
│   └── fixtures/
├── scripts/                     # 脚本
├── docs/                        # 文档
├── go.mod
├── go.sum
├── Dockerfile
└── README.md
```

### 2. 依赖注入

使用依赖注入提高代码的可测试性和可维护性。

```go
// 定义接口
type UserRepository interface {
    Create(ctx context.Context, user *User) error
    GetByID(ctx context.Context, id string) (*User, error)
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, id string) error
}

type UserService interface {
    CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error)
    GetUser(ctx context.Context, id string) (*User, error)
}

// 实现依赖注入
type userService struct {
    repo   UserRepository
    cache  cache.Cache
    logger *zap.Logger
    eventBus events.EventBus
}

func NewUserService(
    repo UserRepository,
    cache cache.Cache,
    logger *zap.Logger,
    eventBus events.EventBus,
) UserService {
    return &userService{
        repo:     repo,
        cache:    cache,
        logger:   logger,
        eventBus: eventBus,
    }
}

// 便于单元测试
func TestUserService_CreateUser(t *testing.T) {
    mockRepo := &MockUserRepository{}
    mockCache := &MockCache{}
    mockEventBus := &MockEventBus{}
    logger, _ := zap.NewDevelopment()

    service := NewUserService(mockRepo, mockCache, logger, mockEventBus)

    // 测试逻辑...
}
```

### 3. 错误处理

建立统一的错误处理机制。

```go
package errors

import (
    "fmt"
    "net/http"
)

// 业务错误类型
type ErrorType string

const (
    ErrorTypeValidation   ErrorType = "VALIDATION_ERROR"
    ErrorTypeNotFound     ErrorType = "NOT_FOUND"
    ErrorTypeUnauthorized ErrorType = "UNAUTHORIZED"
    ErrorTypeInternal     ErrorType = "INTERNAL_ERROR"
    ErrorTypeConflict     ErrorType = "CONFLICT"
)

// 业务错误结构
type BusinessError struct {
    Type    ErrorType `json:"type"`
    Code    string    `json:"code"`
    Message string    `json:"message"`
    Details string    `json:"details,omitempty"`
}

func (e *BusinessError) Error() string {
    return fmt.Sprintf("[%s] %s: %s", e.Type, e.Code, e.Message)
}

func (e *BusinessError) HTTPStatus() int {
    switch e.Type {
    case ErrorTypeValidation:
        return http.StatusBadRequest
    case ErrorTypeNotFound:
        return http.StatusNotFound
    case ErrorTypeUnauthorized:
        return http.StatusUnauthorized
    case ErrorTypeConflict:
        return http.StatusConflict
    default:
        return http.StatusInternalServerError
    }
}

// 错误构造函数
func NewValidationError(code, message string) *BusinessError {
    return &BusinessError{
        Type:    ErrorTypeValidation,
        Code:    code,
        Message: message,
    }
}

func NewNotFoundError(resource, id string) *BusinessError {
    return &BusinessError{
        Type:    ErrorTypeNotFound,
        Code:    "RESOURCE_NOT_FOUND",
        Message: fmt.Sprintf("%s with id %s not found", resource, id),
    }
}

// 统一错误处理中间件
func ErrorHandlingMiddleware() gin.HandlerFunc {
    return gin.RecoveryWithWriter(gin.DefaultWriter, func(c *gin.Context, err interface{}) {
        var businessErr *BusinessError
        var ok bool

        if businessErr, ok = err.(*BusinessError); !ok {
            businessErr = &BusinessError{
                Type:    ErrorTypeInternal,
                Code:    "INTERNAL_ERROR",
                Message: "Internal server error",
            }
        }

        c.JSON(businessErr.HTTPStatus(), gin.H{
            "error": businessErr,
            "timestamp": time.Now(),
            "path": c.Request.URL.Path,
        })
    })
}
```

## 性能优化

### 1. 连接池优化

```go
// 数据库连接池配置
func setupDatabase(config DatabaseConfig) (*sql.DB, error) {
    db, err := sql.Open(config.Driver, config.DSN)
    if err != nil {
        return nil, err
    }

    // 连接池配置
    db.SetMaxOpenConns(config.MaxOpenConns)    // 最大连接数
    db.SetMaxIdleConns(config.MaxIdleConns)    // 最大空闲连接数
    db.SetConnMaxLifetime(config.ConnMaxLifetime) // 连接最大生命周期
    db.SetConnMaxIdleTime(config.ConnMaxIdleTime) // 连接最大空闲时间

    return db, nil
}

// Redis连接池配置
func setupRedis(config RedisConfig) (*redis.Client, error) {
    client := redis.NewClient(&redis.Options{
        Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
        Password:     config.Password,
        DB:           config.DB,
        PoolSize:     config.PoolSize,
        MinIdleConns: config.MinIdleConns,
        MaxConnAge:   config.MaxConnAge,
        PoolTimeout:  config.PoolTimeout,
        IdleTimeout:  config.IdleTimeout,
    })

    return client, nil
}
```

### 2. 缓存策略

```go
// 缓存模式封装
type CacheService struct {
    cache  cache.Cache
    repo   Repository
    logger *zap.Logger
}

// Cache-Aside模式
func (cs *CacheService) Get(ctx context.Context, key string) (interface{}, error) {
    // 1. 先查缓存
    data, err := cs.cache.Get(ctx, key)
    if err == nil {
        return data, nil
    }

    // 2. 缓存未命中，查数据库
    data, err = cs.repo.GetByKey(ctx, key)
    if err != nil {
        return nil, err
    }

    // 3. 更新缓存
    go func() {
        if err := cs.cache.Set(ctx, key, data, 30*time.Minute); err != nil {
            cs.logger.Warn("Failed to update cache", zap.String("key", key), zap.Error(err))
        }
    }()

    return data, nil
}

// Write-Through模式
func (cs *CacheService) Set(ctx context.Context, key string, data interface{}) error {
    // 1. 更新数据库
    if err := cs.repo.Update(ctx, key, data); err != nil {
        return err
    }

    // 2. 更新缓存
    if err := cs.cache.Set(ctx, key, data, 30*time.Minute); err != nil {
        cs.logger.Warn("Failed to update cache", zap.String("key", key), zap.Error(err))
        // 缓存更新失败不影响主流程
    }

    return nil
}

// 缓存预热
func (cs *CacheService) WarmUp(ctx context.Context) error {
    keys, err := cs.repo.GetHotKeys(ctx)
    if err != nil {
        return err
    }

    var wg sync.WaitGroup
    sem := make(chan struct{}, 10) // 限制并发数

    for _, key := range keys {
        wg.Add(1)
        go func(k string) {
            defer wg.Done()
            sem <- struct{}{}
            defer func() { <-sem }()

            data, err := cs.repo.GetByKey(ctx, k)
            if err != nil {
                cs.logger.Warn("Failed to load data for warmup", zap.String("key", k), zap.Error(err))
                return
            }

            cs.cache.Set(ctx, k, data, 1*time.Hour)
        }(key)
    }

    wg.Wait()
    return nil
}
```

### 3. 并发控制

```go
// 使用Worker Pool处理并发任务
type WorkerPool struct {
    tasks    chan Task
    workers  int
    wg       sync.WaitGroup
    ctx      context.Context
    cancel   context.CancelFunc
}

type Task interface {
    Execute() error
}

func NewWorkerPool(workers int) *WorkerPool {
    ctx, cancel := context.WithCancel(context.Background())
    return &WorkerPool{
        tasks:   make(chan Task, workers*2),
        workers: workers,
        ctx:     ctx,
        cancel:  cancel,
    }
}

func (wp *WorkerPool) Start() {
    for i := 0; i < wp.workers; i++ {
        wp.wg.Add(1)
        go wp.worker()
    }
}

func (wp *WorkerPool) Submit(task Task) error {
    select {
    case wp.tasks <- task:
        return nil
    case <-wp.ctx.Done():
        return wp.ctx.Err()
    default:
        return errors.New("worker pool is full")
    }
}

func (wp *WorkerPool) Stop() {
    wp.cancel()
    close(wp.tasks)
    wp.wg.Wait()
}

func (wp *WorkerPool) worker() {
    defer wp.wg.Done()
    for {
        select {
        case task, ok := <-wp.tasks:
            if !ok {
                return
            }
            if err := task.Execute(); err != nil {
                log.Printf("Task execution failed: %v", err)
            }
        case <-wp.ctx.Done():
            return
        }
    }
}

// 使用信号量控制并发
type Semaphore struct {
    ch chan struct{}
}

func NewSemaphore(n int) *Semaphore {
    return &Semaphore{
        ch: make(chan struct{}, n),
    }
}

func (s *Semaphore) Acquire() {
    s.ch <- struct{}{}
}

func (s *Semaphore) Release() {
    <-s.ch
}

func (s *Semaphore) TryAcquire() bool {
    select {
    case s.ch <- struct{}{}:
        return true
    default:
        return false
    }
}

// 使用示例
func processRequests(requests []Request) {
    sem := NewSemaphore(10) // 最多10个并发
    var wg sync.WaitGroup

    for _, req := range requests {
        wg.Add(1)
        go func(r Request) {
            defer wg.Done()

            sem.Acquire()
            defer sem.Release()

            processRequest(r)
        }(req)
    }

    wg.Wait()
}
```

## 安全最佳实践

### 1. 认证和授权

```go
// JWT认证中间件
func JWTAuthMiddleware(secretKey []byte) gin.HandlerFunc {
    return func(c *gin.Context) {
        token := extractToken(c)
        if token == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
            c.Abort()
            return
        }

        claims, err := validateJWT(token, secretKey)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
            c.Abort()
            return
        }

        // 设置用户信息到上下文
        c.Set("user_id", claims.UserID)
        c.Set("user_roles", claims.Roles)
        c.Next()
    }
}

// RBAC权限检查
func RequireRole(requiredRole string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userRoles, exists := c.Get("user_roles")
        if !exists {
            c.JSON(http.StatusForbidden, gin.H{"error": "no roles found"})
            c.Abort()
            return
        }

        roles, ok := userRoles.([]string)
        if !ok {
            c.JSON(http.StatusForbidden, gin.H{"error": "invalid roles format"})
            c.Abort()
            return
        }

        for _, role := range roles {
            if role == requiredRole || role == "admin" {
                c.Next()
                return
            }
        }

        c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
        c.Abort()
    }
}

// 使用示例
func setupRoutes(router *gin.Engine) {
    // 公开路由
    router.POST("/api/v1/login", loginHandler)

    // 需要认证的路由
    auth := router.Group("/api/v1")
    auth.Use(JWTAuthMiddleware([]byte("secret")))
    {
        auth.GET("/profile", getProfile)

        // 需要管理员权限的路由
        admin := auth.Group("/admin")
        admin.Use(RequireRole("admin"))
        {
            admin.GET("/users", listUsers)
            admin.DELETE("/users/:id", deleteUser)
        }
    }
}
```

### 2. 输入验证

```go
// 请求验证
type CreateUserRequest struct {
    Email    string `json:"email" binding:"required,email" validate:"email"`
    Name     string `json:"name" binding:"required,min=2,max=50" validate:"min=2,max=50"`
    Password string `json:"password" binding:"required,min=8" validate:"min=8,containsany=!@#$%^&*"`
    Age      int    `json:"age" binding:"min=18,max=120" validate:"min=18,max=120"`
}

// 自定义验证器
func setupCustomValidators() {
    if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
        v.RegisterValidation("password_strength", validatePasswordStrength)
        v.RegisterValidation("phone", validatePhone)
    }
}

func validatePasswordStrength(fl validator.FieldLevel) bool {
    password := fl.Field().String()

    // 检查密码强度
    hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
    hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
    hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
    hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password)

    return hasUpper && hasLower && hasNumber && hasSpecial
}

// SQL注入防护
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
    // ✅ 使用参数化查询
    query := "SELECT id, name, email FROM users WHERE email = $1"

    var user User
    err := r.db.QueryRowContext(ctx, query, email).Scan(&user.ID, &user.Name, &user.Email)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, ErrUserNotFound
        }
        return nil, err
    }

    return &user, nil
}

// XSS防护
func sanitizeInput(input string) string {
    // 使用html包转义HTML特殊字符
    return html.EscapeString(input)
}

// CSRF防护
func CSRFMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "DELETE" {
            token := c.GetHeader("X-CSRF-Token")
            if !validateCSRFToken(token, c) {
                c.JSON(http.StatusForbidden, gin.H{"error": "invalid CSRF token"})
                c.Abort()
                return
            }
        }
        c.Next()
    }
}
```

### 3. 敏感数据保护

```go
// 密码加密
func hashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    return string(bytes), err
}

func checkPassword(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}

// 数据加密
type EncryptionService struct {
    key []byte
}

func NewEncryptionService(key string) *EncryptionService {
    return &EncryptionService{key: []byte(key)}
}

func (e *EncryptionService) Encrypt(plaintext string) (string, error) {
    block, err := aes.NewCipher(e.key)
    if err != nil {
        return "", err
    }

    ciphertext := make([]byte, aes.BlockSize+len(plaintext))
    iv := ciphertext[:aes.BlockSize]
    if _, err := io.ReadFull(rand.Reader, iv); err != nil {
        return "", err
    }

    stream := cipher.NewCFBEncrypter(block, iv)
    stream.XORKeyStream(ciphertext[aes.BlockSize:], []byte(plaintext))

    return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// 日志脱敏
type SensitiveData struct {
    Email    string `json:"email"`
    Phone    string `json:"phone"`
    Password string `json:"-"`        // 不序列化
    IDCard   string `json:"id_card"`
}

func (s SensitiveData) MarshalJSON() ([]byte, error) {
    type Alias SensitiveData
    return json.Marshal(&struct {
        Email  string `json:"email"`
        Phone  string `json:"phone"`
        IDCard string `json:"id_card"`
        *Alias
    }{
        Email:  maskEmail(s.Email),
        Phone:  maskPhone(s.Phone),
        IDCard: maskIDCard(s.IDCard),
        Alias:  (*Alias)(&s),
    })
}

func maskEmail(email string) string {
    parts := strings.Split(email, "@")
    if len(parts) != 2 {
        return email
    }
    username := parts[0]
    if len(username) <= 2 {
        return email
    }
    return username[:2] + "***@" + parts[1]
}
```

## 监控和日志

### 1. 结构化日志

```go
// 统一日志格式
type Logger struct {
    *zap.Logger
}

func NewLogger(config LogConfig) (*Logger, error) {
    var zapConfig zap.Config

    if config.Environment == "production" {
        zapConfig = zap.NewProductionConfig()
    } else {
        zapConfig = zap.NewDevelopmentConfig()
    }

    zapConfig.Level = zap.NewAtomicLevelAt(getLogLevel(config.Level))

    logger, err := zapConfig.Build(
        zap.AddCallerSkip(1),
        zap.AddStacktrace(zap.ErrorLevel),
    )
    if err != nil {
        return nil, err
    }

    return &Logger{logger}, nil
}

// 业务日志记录
func (l *Logger) LogUserAction(ctx context.Context, action string, userID string, details map[string]interface{}) {
    fields := []zap.Field{
        zap.String("action", action),
        zap.String("user_id", userID),
        zap.Any("details", details),
    }

    // 从上下文中提取追踪信息
    if traceID := getTraceIDFromContext(ctx); traceID != "" {
        fields = append(fields, zap.String("trace_id", traceID))
    }

    if requestID := getRequestIDFromContext(ctx); requestID != "" {
        fields = append(fields, zap.String("request_id", requestID))
    }

    l.Info("User action", fields...)
}

// 错误日志记录
func (l *Logger) LogError(ctx context.Context, err error, message string, fields ...zap.Field) {
    allFields := []zap.Field{
        zap.Error(err),
        zap.String("error_type", fmt.Sprintf("%T", err)),
    }

    // 添加上下文信息
    if traceID := getTraceIDFromContext(ctx); traceID != "" {
        allFields = append(allFields, zap.String("trace_id", traceID))
    }

    allFields = append(allFields, fields...)
    l.Error(message, allFields...)
}
```

### 2. 指标监控

```go
// 业务指标定义
var (
    RequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "endpoint", "status"},
    )

    RequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "endpoint"},
    )

    BusinessMetrics = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "business_operations_total",
            Help: "Total number of business operations",
        },
        []string{"operation", "result"},
    )
)

// 指标收集中间件
func MetricsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()

        c.Next()

        duration := time.Since(start)
        status := fmt.Sprintf("%d", c.Writer.Status())

        RequestsTotal.WithLabelValues(c.Request.Method, c.FullPath(), status).Inc()
        RequestDuration.WithLabelValues(c.Request.Method, c.FullPath()).Observe(duration.Seconds())
    }
}

// 业务指标记录
func (s *UserService) CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error) {
    start := time.Now()

    user, err := s.repo.Create(ctx, req)

    // 记录业务指标
    result := "success"
    if err != nil {
        result = "failure"
    }
    BusinessMetrics.WithLabelValues("user_creation", result).Inc()

    // 记录操作延迟
    duration := time.Since(start)
    if duration > 5*time.Second {
        s.logger.Warn("Slow user creation",
            zap.Duration("duration", duration),
            zap.String("user_id", req.Email),
        )
    }

    return user, err
}
```

## 测试策略

### 1. 单元测试

```go
// 使用mock进行单元测试
func TestUserService_CreateUser(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockRepo := mocks.NewMockUserRepository(ctrl)
    mockCache := mocks.NewMockCache(ctrl)
    mockEventBus := mocks.NewMockEventBus(ctrl)
    logger, _ := zap.NewDevelopment()

    service := NewUserService(mockRepo, mockCache, logger, mockEventBus)

    tests := []struct {
        name    string
        request *CreateUserRequest
        setup   func()
        wantErr bool
    }{
        {
            name: "successful creation",
            request: &CreateUserRequest{
                Email: "test@example.com",
                Name:  "Test User",
            },
            setup: func() {
                mockRepo.EXPECT().
                    Create(gomock.Any(), gomock.Any()).
                    Return(&User{ID: "123", Email: "test@example.com"}, nil)

                mockEventBus.EXPECT().
                    Publish(gomock.Any(), gomock.Any()).
                    Return(nil)
            },
            wantErr: false,
        },
        {
            name: "duplicate email",
            request: &CreateUserRequest{
                Email: "test@example.com",
                Name:  "Test User",
            },
            setup: func() {
                mockRepo.EXPECT().
                    Create(gomock.Any(), gomock.Any()).
                    Return(nil, ErrDuplicateEmail)
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tt.setup()

            user, err := service.CreateUser(context.Background(), tt.request)

            if tt.wantErr {
                assert.Error(t, err)
                assert.Nil(t, user)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, user)
            }
        })
    }
}
```

### 2. 集成测试

```go
// 集成测试设置
func setupTestEnvironment() (*TestEnvironment, func()) {
    // 启动测试数据库
    db := setupTestDB()

    // 启动测试Redis
    redisClient := setupTestRedis()

    // 创建测试服务
    userRepo := repository.NewUserRepository(db)
    userService := service.NewUserService(userRepo, redisClient, logger, eventBus)

    env := &TestEnvironment{
        DB:          db,
        Redis:       redisClient,
        UserService: userService,
    }

    cleanup := func() {
        db.Close()
        redisClient.Close()
    }

    return env, cleanup
}

func TestUserAPI_Integration(t *testing.T) {
    env, cleanup := setupTestEnvironment()
    defer cleanup()

    // 创建测试服务器
    router := setupRouter(env.UserService)

    tests := []struct {
        name           string
        method         string
        path           string
        body           interface{}
        expectedStatus int
    }{
        {
            name:   "create user",
            method: "POST",
            path:   "/api/v1/users",
            body: CreateUserRequest{
                Email: "test@example.com",
                Name:  "Test User",
            },
            expectedStatus: http.StatusCreated,
        },
        {
            name:           "get user",
            method:         "GET",
            path:           "/api/v1/users/123",
            expectedStatus: http.StatusOK,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var bodyReader io.Reader
            if tt.body != nil {
                bodyBytes, _ := json.Marshal(tt.body)
                bodyReader = bytes.NewReader(bodyBytes)
            }

            req := httptest.NewRequest(tt.method, tt.path, bodyReader)
            req.Header.Set("Content-Type", "application/json")

            w := httptest.NewRecorder()
            router.ServeHTTP(w, req)

            assert.Equal(t, tt.expectedStatus, w.Code)
        })
    }
}
```

### 3. 性能测试

```go
// 性能测试
func BenchmarkUserService_CreateUser(b *testing.B) {
    env, cleanup := setupTestEnvironment()
    defer cleanup()

    req := &CreateUserRequest{
        Email: "test@example.com",
        Name:  "Test User",
    }

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            _, err := env.UserService.CreateUser(context.Background(), req)
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}

// 负载测试（使用外部工具配置）
func TestLoadTesting(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping load test in short mode")
    }

    // 启动测试服务器
    server := setupTestServer()
    defer server.Close()

    // 配置负载测试参数
    config := LoadTestConfig{
        URL:             server.URL + "/api/v1/users",
        Concurrency:     50,
        Duration:        30 * time.Second,
        RequestsPerSec:  100,
    }

    // 执行负载测试
    results := runLoadTest(config)

    // 验证性能指标
    assert.True(t, results.AverageResponseTime < 100*time.Millisecond)
    assert.True(t, results.ErrorRate < 0.01) // 错误率小于1%
    assert.True(t, results.P99ResponseTime < 500*time.Millisecond)
}
```

## 运维最佳实践

### 1. 健康检查

```go
// 多层次健康检查
type HealthChecker struct {
    db           *sql.DB
    redis        *redis.Client
    dependencies []ServiceDependency
}

type ServiceDependency struct {
    Name     string
    URL      string
    Critical bool
    Timeout  time.Duration
}

func (hc *HealthChecker) Check(ctx context.Context) HealthStatus {
    status := HealthStatus{
        Status:    "healthy",
        Timestamp: time.Now(),
        Checks:    make(map[string]CheckResult),
    }

    // 检查数据库
    if err := hc.checkDatabase(ctx); err != nil {
        status.Checks["database"] = CheckResult{
            Status:  "unhealthy",
            Message: err.Error(),
        }
        status.Status = "unhealthy"
    } else {
        status.Checks["database"] = CheckResult{Status: "healthy"}
    }

    // 检查Redis
    if err := hc.checkRedis(ctx); err != nil {
        status.Checks["redis"] = CheckResult{
            Status:  "unhealthy",
            Message: err.Error(),
        }
        if hc.isCritical("redis") {
            status.Status = "unhealthy"
        } else {
            status.Status = "degraded"
        }
    } else {
        status.Checks["redis"] = CheckResult{Status: "healthy"}
    }

    // 检查依赖服务
    for _, dep := range hc.dependencies {
        if err := hc.checkDependency(ctx, dep); err != nil {
            status.Checks[dep.Name] = CheckResult{
                Status:  "unhealthy",
                Message: err.Error(),
            }
            if dep.Critical {
                status.Status = "unhealthy"
            } else if status.Status == "healthy" {
                status.Status = "degraded"
            }
        } else {
            status.Checks[dep.Name] = CheckResult{Status: "healthy"}
        }
    }

    return status
}

func (hc *HealthChecker) checkDatabase(ctx context.Context) error {
    ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
    defer cancel()

    return hc.db.PingContext(ctx)
}

func (hc *HealthChecker) checkRedis(ctx context.Context) error {
    ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
    defer cancel()

    return hc.redis.Ping(ctx).Err()
}

func (hc *HealthChecker) checkDependency(ctx context.Context, dep ServiceDependency) error {
    client := &http.Client{Timeout: dep.Timeout}

    req, err := http.NewRequestWithContext(ctx, "GET", dep.URL+"/health", nil)
    if err != nil {
        return err
    }

    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        return fmt.Errorf("dependency returned status %d", resp.StatusCode)
    }

    return nil
}
```

### 2. 优雅关闭

```go
// 优雅关闭实现
type GracefulShutdown struct {
    server    *http.Server
    db        *sql.DB
    redis     *redis.Client
    logger    *zap.Logger
    timeout   time.Duration
    callbacks []func() error
}

func NewGracefulShutdown(server *http.Server, timeout time.Duration, logger *zap.Logger) *GracefulShutdown {
    return &GracefulShutdown{
        server:  server,
        timeout: timeout,
        logger:  logger,
    }
}

func (gs *GracefulShutdown) AddCallback(callback func() error) {
    gs.callbacks = append(gs.callbacks, callback)
}

func (gs *GracefulShutdown) Listen() {
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    <-sigChan
    gs.logger.Info("Shutdown signal received")

    gs.Shutdown()
}

func (gs *GracefulShutdown) Shutdown() {
    ctx, cancel := context.WithTimeout(context.Background(), gs.timeout)
    defer cancel()

    // 停止接受新请求
    gs.logger.Info("Stopping HTTP server")
    if err := gs.server.Shutdown(ctx); err != nil {
        gs.logger.Error("Server shutdown error", zap.Error(err))
    }

    // 执行清理回调
    for i, callback := range gs.callbacks {
        gs.logger.Info("Executing shutdown callback", zap.Int("index", i))
        if err := callback(); err != nil {
            gs.logger.Error("Shutdown callback error", zap.Int("index", i), zap.Error(err))
        }
    }

    // 关闭数据库连接
    if gs.db != nil {
        gs.logger.Info("Closing database connections")
        gs.db.Close()
    }

    // 关闭Redis连接
    if gs.redis != nil {
        gs.logger.Info("Closing Redis connections")
        gs.redis.Close()
    }

    gs.logger.Info("Shutdown completed")
}

// 使用示例
func main() {
    // 初始化服务
    server := &http.Server{Addr: ":8080", Handler: router}

    // 设置优雅关闭
    shutdown := NewGracefulShutdown(server, 30*time.Second, logger)
    shutdown.AddCallback(func() error {
        // 清理缓存
        return cache.Flush()
    })
    shutdown.AddCallback(func() error {
        // 关闭消息队列连接
        return messageQueue.Close()
    })

    // 启动关闭监听
    go shutdown.Listen()

    // 启动服务器
    if err := server.ListenAndServe(); err != http.ErrServerClosed {
        logger.Fatal("Server failed to start", zap.Error(err))
    }
}
```

### 3. 配置管理

```go
// 配置热重载
type ConfigManager struct {
    config     *Config
    configPath string
    watcher    *fsnotify.Watcher
    callbacks  []func(*Config)
    mutex      sync.RWMutex
}

func NewConfigManager(configPath string) (*ConfigManager, error) {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return nil, err
    }

    cm := &ConfigManager{
        configPath: configPath,
        watcher:    watcher,
        callbacks:  make([]func(*Config), 0),
    }

    // 初始加载配置
    if err := cm.loadConfig(); err != nil {
        return nil, err
    }

    // 监听配置文件变化
    if err := watcher.Add(configPath); err != nil {
        return nil, err
    }

    go cm.watchConfig()

    return cm, nil
}

func (cm *ConfigManager) loadConfig() error {
    data, err := ioutil.ReadFile(cm.configPath)
    if err != nil {
        return err
    }

    var config Config
    if err := yaml.Unmarshal(data, &config); err != nil {
        return err
    }

    cm.mutex.Lock()
    cm.config = &config
    cm.mutex.Unlock()

    return nil
}

func (cm *ConfigManager) watchConfig() {
    for {
        select {
        case event, ok := <-cm.watcher.Events:
            if !ok {
                return
            }

            if event.Op&fsnotify.Write == fsnotify.Write {
                log.Println("Config file modified, reloading...")

                if err := cm.loadConfig(); err != nil {
                    log.Printf("Failed to reload config: %v", err)
                    continue
                }

                // 通知所有回调
                cm.mutex.RLock()
                config := cm.config
                callbacks := cm.callbacks
                cm.mutex.RUnlock()

                for _, callback := range callbacks {
                    callback(config)
                }
            }

        case err, ok := <-cm.watcher.Errors:
            if !ok {
                return
            }
            log.Printf("Config watcher error: %v", err)
        }
    }
}

func (cm *ConfigManager) OnConfigChange(callback func(*Config)) {
    cm.mutex.Lock()
    defer cm.mutex.Unlock()
    cm.callbacks = append(cm.callbacks, callback)
}

func (cm *ConfigManager) GetConfig() *Config {
    cm.mutex.RLock()
    defer cm.mutex.RUnlock()
    return cm.config
}
```

通过遵循这些最佳实践，可以构建出高质量、可维护、安全可靠的微服务系统。这些实践涵盖了微服务开发的各个方面，从架构设计到运维管理，为团队提供了完整的指导方针。

<function_calls>
<invoke name="TodoWrite">
<parameter name="todos">[{"content": "\u521b\u5eba\u5fae\u670d\u52a1\u67b6\u6784\u6587\u6863", "status": "completed", "activeForm": "\u521b\u5eba\u5fae\u670d\u52a1\u67b6\u6784\u6587\u6863"}, {"content": "\u521b\u5eba\u5fae\u670d\u52a1\u901a\u4fe1\u6587\u6863", "status": "completed", "activeForm": "\u521b\u5eba\u5fae\u670d\u52a1\u901a\u4fe1\u6587\u6863"}, {"content": "\u521b\u5eba\u5fae\u670d\u52a1\u914d\u7f6e\u7ba1\u7406\u6587\u6863", "status": "completed", "activeForm": "\u521b\u5eba\u5fae\u670d\u52a1\u914d\u7f6e\u7ba1\u7406\u6587\u6863"}, {"content": "\u521b\u5eba\u5fae\u670d\u52a1\u90e8\u7f72\u6587\u6863", "status": "completed", "activeForm": "\u521b\u5eba\u5fae\u670d\u52a1\u90e8\u7f72\u6587\u6863"}, {"content": "\u521b\u5eba\u5fae\u670d\u52a1\u6700\u4f73\u5b9e\u8df5\u6587\u6863", "status": "completed", "activeForm": "\u521b\u5eba\u5fae\u670d\u52a1\u6700\u4f73\u5b9e\u8df5\u6587\u6863"}, {"content": "\u66f4\u65b0\u4e3bREADME\u6dfb\u52a0\u5fae\u670d\u52a1\u7279\u6027", "status": "in_progress", "activeForm": "\u66f4\u65b0\u4e3bREADME\u6dfb\u52a0\u5fae\u670d\u52a1\u7279\u6027"}]