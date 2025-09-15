# 可观测性

## 概述

Chi 框架提供了完整的可观测性解决方案，包括分布式链路追踪、实时指标监控、结构化日志聚合和多层级健康检查。通过集成 OpenTelemetry、Prometheus 和 Jaeger 等业界标准工具，为企业应用提供全方位的可观测能力。

## 可观测性架构

### 整体架构图
```
┌─────────────────────────────────────────────────────────────────┐
│                     Observability                               │
│                                                                 │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐          │
│  │ Distributed │   │   Metrics   │   │    Logs     │          │
│  │   Tracing   │   │ Collection  │   │ Aggregation │          │
│  │   (OTEL)    │   │(Prometheus) │   │  (Fluent)   │          │
│  └─────────────┘   └─────────────┘   └─────────────┘          │
│          │                 │                 │                 │
│          └─────────────────┼─────────────────┘                 │
│                            │                                   │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │               Observability Gateway                        ││
│  └─────────────────────────────────────────────────────────────┐│
│                            │                                   │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐          │
│  │   Health    │   │   Alert     │   │ Dashboard   │          │
│  │   Check     │   │  Manager    │   │   Portal    │          │
│  └─────────────┘   └─────────────┘   └─────────────┘          │
└─────────────────────────────────────────────────────────────────┘
```

### 三大支柱

1. **Tracing (链路追踪)**: 分布式系统中请求的完整路径追踪
2. **Metrics (指标监控)**: 系统和业务指标的实时收集与监控
3. **Logging (日志聚合)**: 结构化日志的收集、存储和分析

## 分布式链路追踪

### OpenTelemetry 集成

```go
package observability

import (
    "context"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/exporters/jaeger"
    "go.opentelemetry.io/otel/sdk/resource"
    "go.opentelemetry.io/otel/sdk/trace"
    "go.opentelemetry.io/otel/semconv/v1.12.0"
)

// 追踪配置
type TracingConfig struct {
    ServiceName     string
    ServiceVersion  string
    Environment     string
    JaegerEndpoint  string
    SampleRatio     float64
    BatchTimeout    time.Duration
    ExportTimeout   time.Duration
}

// 初始化追踪器
func InitTracing(config TracingConfig) (*trace.TracerProvider, error) {
    // 创建 Jaeger exporter
    exp, err := jaeger.New(jaeger.WithCollectorEndpoint(
        jaeger.WithEndpoint(config.JaegerEndpoint),
    ))
    if err != nil {
        return nil, fmt.Errorf("failed to create jaeger exporter: %w", err)
    }

    // 创建资源
    res, err := resource.Merge(
        resource.Default(),
        resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String(config.ServiceName),
            semconv.ServiceVersionKey.String(config.ServiceVersion),
            semconv.DeploymentEnvironmentKey.String(config.Environment),
        ),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create resource: %w", err)
    }

    // 创建追踪器提供者
    tp := trace.NewTracerProvider(
        trace.WithBatcher(exp,
            trace.WithBatchTimeout(config.BatchTimeout),
            trace.WithExportTimeout(config.ExportTimeout),
        ),
        trace.WithResource(res),
        trace.WithSampler(trace.TraceIDRatioBased(config.SampleRatio)),
    )

    // 设置全局追踪器提供者
    otel.SetTracerProvider(tp)

    return tp, nil
}

// 追踪装饰器
func TraceFunction(ctx context.Context, operationName string, fn func(context.Context) error, attrs ...attribute.KeyValue) error {
    tracer := otel.Tracer("chi-framework")
    ctx, span := tracer.Start(ctx, operationName)
    defer span.End()

    // 添加属性
    span.SetAttributes(attrs...)

    // 执行函数
    err := fn(ctx)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
    }

    return err
}
```

### HTTP 追踪中间件

```go
// HTTP追踪中间件
func TracingMiddleware() gin.HandlerFunc {
    return gin.HandlerFunc(func(c *gin.Context) {
        tracer := otel.Tracer("chi-http")

        // 提取或创建span上下文
        ctx := otel.GetTextMapPropagator().Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

        spanName := fmt.Sprintf("%s %s", c.Request.Method, c.FullPath())
        ctx, span := tracer.Start(ctx, spanName)
        defer span.End()

        // 设置span属性
        span.SetAttributes(
            semconv.HTTPMethodKey.String(c.Request.Method),
            semconv.HTTPURLKey.String(c.Request.URL.String()),
            semconv.HTTPUserAgentKey.String(c.Request.UserAgent()),
            semconv.HTTPClientIPKey.String(c.ClientIP()),
        )

        // 将上下文传递给下游
        c.Request = c.Request.WithContext(ctx)

        c.Next()

        // 记录响应信息
        span.SetAttributes(
            semconv.HTTPStatusCodeKey.Int(c.Writer.Status()),
        )

        if c.Writer.Status() >= 400 {
            span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", c.Writer.Status()))
        }
    })
}
```

### 数据库追踪

```go
// 数据库追踪包装器
type TracedDB struct {
    db     *sql.DB
    tracer trace.Tracer
}

func NewTracedDB(db *sql.DB) *TracedDB {
    return &TracedDB{
        db:     db,
        tracer: otel.Tracer("chi-database"),
    }
}

func (tdb *TracedDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
    ctx, span := tdb.tracer.Start(ctx, "db.query")
    defer span.End()

    span.SetAttributes(
        attribute.String("db.statement", query),
        attribute.String("db.operation", "query"),
    )

    rows, err := tdb.db.QueryContext(ctx, query, args...)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
    }

    return rows, err
}

func (tdb *TracedDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
    ctx, span := tdb.tracer.Start(ctx, "db.exec")
    defer span.End()

    span.SetAttributes(
        attribute.String("db.statement", query),
        attribute.String("db.operation", "exec"),
    )

    result, err := tdb.db.ExecContext(ctx, query, args...)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
    }

    return result, err
}
```

### gRPC 追踪

```go
// gRPC客户端追踪拦截器
func TracingUnaryClientInterceptor() grpc.UnaryClientInterceptor {
    return func(ctx context.Context, method string, req, reply interface{},
        cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

        tracer := otel.Tracer("chi-grpc-client")
        ctx, span := tracer.Start(ctx, method)
        defer span.End()

        span.SetAttributes(
            attribute.String("rpc.system", "grpc"),
            attribute.String("rpc.method", method),
            attribute.String("rpc.service", serviceFromMethod(method)),
        )

        // 注入追踪上下文
        ctx = metadata.AppendToOutgoingContext(ctx, getTracingMetadata(ctx)...)

        err := invoker(ctx, method, req, reply, cc, opts...)
        if err != nil {
            span.RecordError(err)
            span.SetStatus(codes.Error, err.Error())
        }

        return err
    }
}

// gRPC服务端追踪拦截器
func TracingUnaryServerInterceptor() grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler) (interface{}, error) {

        // 提取追踪上下文
        md, ok := metadata.FromIncomingContext(ctx)
        if ok {
            ctx = otel.GetTextMapPropagator().Extract(ctx, &metadataCarrier{md})
        }

        tracer := otel.Tracer("chi-grpc-server")
        ctx, span := tracer.Start(ctx, info.FullMethod)
        defer span.End()

        span.SetAttributes(
            attribute.String("rpc.system", "grpc"),
            attribute.String("rpc.method", info.FullMethod),
            attribute.String("rpc.service", serviceFromMethod(info.FullMethod)),
        )

        resp, err := handler(ctx, req)
        if err != nil {
            span.RecordError(err)
            span.SetStatus(codes.Error, err.Error())
        }

        return resp, err
    }
}
```

## 指标监控

### Prometheus 集成

```go
// 指标配置
type MetricsConfig struct {
    Namespace   string
    Subsystem   string
    EnabledMetrics []string
    CustomMetrics  map[string]MetricConfig
}

type MetricConfig struct {
    Type        string            // counter, gauge, histogram, summary
    Help        string
    Labels      []string
    Buckets     []float64         // for histogram
    Objectives  map[float64]float64 // for summary
}

// 指标收集器
type MetricsCollector struct {
    registry   *prometheus.Registry
    counters   map[string]*prometheus.CounterVec
    gauges     map[string]*prometheus.GaugeVec
    histograms map[string]*prometheus.HistogramVec
    summaries  map[string]*prometheus.SummaryVec
}

func NewMetricsCollector(config MetricsConfig) *MetricsCollector {
    collector := &MetricsCollector{
        registry:   prometheus.NewRegistry(),
        counters:   make(map[string]*prometheus.CounterVec),
        gauges:     make(map[string]*prometheus.GaugeVec),
        histograms: make(map[string]*prometheus.HistogramVec),
        summaries:  make(map[string]*prometheus.SummaryVec),
    }

    collector.initDefaultMetrics(config)
    collector.initCustomMetrics(config)

    return collector
}

func (mc *MetricsCollector) initDefaultMetrics(config MetricsConfig) {
    // HTTP请求计数器
    httpRequestsTotal := prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: config.Namespace,
            Subsystem: config.Subsystem,
            Name:      "http_requests_total",
            Help:      "Total number of HTTP requests",
        },
        []string{"method", "endpoint", "status"},
    )

    // HTTP请求延迟直方图
    httpRequestDuration := prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Namespace: config.Namespace,
            Subsystem: config.Subsystem,
            Name:      "http_request_duration_seconds",
            Help:      "HTTP request duration in seconds",
            Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
        },
        []string{"method", "endpoint"},
    )

    // 数据库连接池指标
    dbConnections := prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Namespace: config.Namespace,
            Subsystem: config.Subsystem,
            Name:      "db_connections",
            Help:      "Current number of database connections",
        },
        []string{"state"}, // active, idle, waiting
    )

    // gRPC请求指标
    grpcRequestsTotal := prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: config.Namespace,
            Subsystem: config.Subsystem,
            Name:      "grpc_requests_total",
            Help:      "Total number of gRPC requests",
        },
        []string{"service", "method", "status"},
    )

    mc.counters["http_requests_total"] = httpRequestsTotal
    mc.histograms["http_request_duration_seconds"] = httpRequestDuration
    mc.gauges["db_connections"] = dbConnections
    mc.counters["grpc_requests_total"] = grpcRequestsTotal

    mc.registry.MustRegister(
        httpRequestsTotal,
        httpRequestDuration,
        dbConnections,
        grpcRequestsTotal,
    )
}

// 业务指标记录
func (mc *MetricsCollector) RecordHTTPRequest(method, endpoint string, status int, duration time.Duration) {
    statusStr := fmt.Sprintf("%d", status)

    if counter, exists := mc.counters["http_requests_total"]; exists {
        counter.WithLabelValues(method, endpoint, statusStr).Inc()
    }

    if histogram, exists := mc.histograms["http_request_duration_seconds"]; exists {
        histogram.WithLabelValues(method, endpoint).Observe(duration.Seconds())
    }
}

func (mc *MetricsCollector) UpdateDBConnections(active, idle, waiting int) {
    if gauge, exists := mc.gauges["db_connections"]; exists {
        gauge.WithLabelValues("active").Set(float64(active))
        gauge.WithLabelValues("idle").Set(float64(idle))
        gauge.WithLabelValues("waiting").Set(float64(waiting))
    }
}
```

### 自定义业务指标

```go
// 业务指标定义
type BusinessMetrics struct {
    collector *MetricsCollector

    // 用户相关指标
    UserRegistrations *prometheus.CounterVec
    UserLogins        *prometheus.CounterVec
    ActiveUsers       prometheus.Gauge

    // 订单相关指标
    OrdersTotal       *prometheus.CounterVec
    OrderValue        *prometheus.HistogramVec
    OrderProcessTime  *prometheus.HistogramVec

    // 支付相关指标
    PaymentAttempts   *prometheus.CounterVec
    PaymentAmount     *prometheus.SummaryVec
    PaymentLatency    *prometheus.HistogramVec
}

func NewBusinessMetrics(collector *MetricsCollector) *BusinessMetrics {
    bm := &BusinessMetrics{collector: collector}
    bm.initMetrics()
    return bm
}

func (bm *BusinessMetrics) initMetrics() {
    bm.UserRegistrations = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "user_registrations_total",
            Help: "Total number of user registrations",
        },
        []string{"source", "result"},
    )

    bm.UserLogins = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "user_logins_total",
            Help: "Total number of user logins",
        },
        []string{"method", "result"},
    )

    bm.ActiveUsers = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "active_users",
            Help: "Current number of active users",
        },
    )

    bm.OrdersTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "orders_total",
            Help: "Total number of orders",
        },
        []string{"status", "payment_method"},
    )

    bm.OrderValue = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "order_value_dollars",
            Help:    "Order value in dollars",
            Buckets: []float64{10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
        },
        []string{"category"},
    )

    // 注册指标
    bm.collector.registry.MustRegister(
        bm.UserRegistrations,
        bm.UserLogins,
        bm.ActiveUsers,
        bm.OrdersTotal,
        bm.OrderValue,
    )
}

// 业务事件记录
func (bm *BusinessMetrics) RecordUserRegistration(source string, success bool) {
    result := "success"
    if !success {
        result = "failure"
    }
    bm.UserRegistrations.WithLabelValues(source, result).Inc()
}

func (bm *BusinessMetrics) RecordOrder(status, paymentMethod string, value float64, category string) {
    bm.OrdersTotal.WithLabelValues(status, paymentMethod).Inc()
    bm.OrderValue.WithLabelValues(category).Observe(value)
}

func (bm *BusinessMetrics) UpdateActiveUsers(count int) {
    bm.ActiveUsers.Set(float64(count))
}
```

### 指标中间件

```go
// HTTP指标中间件
func MetricsMiddleware(collector *MetricsCollector) gin.HandlerFunc {
    return gin.HandlerFunc(func(c *gin.Context) {
        start := time.Now()

        c.Next()

        duration := time.Since(start)
        method := c.Request.Method
        endpoint := c.FullPath()
        status := c.Writer.Status()

        collector.RecordHTTPRequest(method, endpoint, status, duration)
    })
}

// 实时指标更新
func StartMetricsUpdater(collector *MetricsCollector, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for range ticker.C {
        updateSystemMetrics(collector)
        updateBusinessMetrics(collector)
    }
}

func updateSystemMetrics(collector *MetricsCollector) {
    // 更新数据库连接池指标
    dbStats := database.GetStats()
    collector.UpdateDBConnections(
        dbStats.OpenConnections,
        dbStats.Idle,
        dbStats.WaitCount,
    )

    // 更新内存使用指标
    var m runtime.MemStats
    runtime.ReadMemStats(&m)

    if gauge, exists := collector.gauges["memory_usage_bytes"]; exists {
        gauge.WithLabelValues("alloc").Set(float64(m.Alloc))
        gauge.WithLabelValues("sys").Set(float64(m.Sys))
    }
}
```

## 结构化日志

### 日志配置和初始化

```go
// 日志配置
type LogConfig struct {
    Level       string            // debug, info, warn, error
    Format      string            // json, text
    Output      []string          // stdout, file, remote
    Rotation    RotationConfig
    Fields      map[string]interface{} // 全局字段
    Sampling    SamplingConfig
}

type RotationConfig struct {
    MaxSize    int    // MB
    MaxAge     int    // days
    MaxBackups int
    Compress   bool
}

type SamplingConfig struct {
    Initial    int
    Thereafter int
}

// 结构化日志器
type StructuredLogger struct {
    logger   *zap.Logger
    sugar    *zap.SugaredLogger
    config   LogConfig
    hostname string
    service  string
}

func NewStructuredLogger(config LogConfig, service string) (*StructuredLogger, error) {
    // 配置编码器
    encoderConfig := zap.NewProductionEncoderConfig()
    encoderConfig.TimeKey = "timestamp"
    encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
    encoderConfig.LevelKey = "level"
    encoderConfig.MessageKey = "message"

    var encoder zapcore.Encoder
    if config.Format == "json" {
        encoder = zapcore.NewJSONEncoder(encoderConfig)
    } else {
        encoder = zapcore.NewConsoleEncoder(encoderConfig)
    }

    // 配置输出
    var cores []zapcore.Core
    level := getLogLevel(config.Level)

    for _, output := range config.Output {
        var writer zapcore.WriteSyncer

        switch output {
        case "stdout":
            writer = zapcore.AddSync(os.Stdout)
        case "file":
            writer = getFileWriter(config.Rotation)
        case "remote":
            writer = getRemoteWriter()
        }

        core := zapcore.NewCore(encoder, writer, level)
        cores = append(cores, core)
    }

    // 合并cores
    core := zapcore.NewTee(cores...)

    // 添加采样
    if config.Sampling.Initial > 0 {
        core = zapcore.NewSamplerWithOptions(core, time.Second,
            config.Sampling.Initial,
            config.Sampling.Thereafter)
    }

    logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))

    // 添加全局字段
    hostname, _ := os.Hostname()
    fields := []zap.Field{
        zap.String("service", service),
        zap.String("hostname", hostname),
    }

    for key, value := range config.Fields {
        fields = append(fields, zap.Any(key, value))
    }

    logger = logger.With(fields...)

    return &StructuredLogger{
        logger:   logger,
        sugar:    logger.Sugar(),
        config:   config,
        hostname: hostname,
        service:  service,
    }, nil
}
```

### 上下文日志

```go
// 上下文日志器
type ContextLogger struct {
    *StructuredLogger
    ctx context.Context
}

func (sl *StructuredLogger) WithContext(ctx context.Context) *ContextLogger {
    return &ContextLogger{
        StructuredLogger: sl,
        ctx:             ctx,
    }
}

func (cl *ContextLogger) Info(msg string, fields ...zap.Field) {
    fields = cl.enrichWithContext(fields...)
    cl.logger.Info(msg, fields...)
}

func (cl *ContextLogger) Error(msg string, fields ...zap.Field) {
    fields = cl.enrichWithContext(fields...)
    cl.logger.Error(msg, fields...)
}

func (cl *ContextLogger) enrichWithContext(fields ...zap.Field) []zap.Field {
    // 从上下文中提取追踪信息
    span := trace.SpanFromContext(cl.ctx)
    if span.SpanContext().IsValid() {
        fields = append(fields,
            zap.String("trace_id", span.SpanContext().TraceID().String()),
            zap.String("span_id", span.SpanContext().SpanID().String()),
        )
    }

    // 提取用户信息
    if userID := getUserIDFromContext(cl.ctx); userID != "" {
        fields = append(fields, zap.String("user_id", userID))
    }

    // 提取请求ID
    if requestID := getRequestIDFromContext(cl.ctx); requestID != "" {
        fields = append(fields, zap.String("request_id", requestID))
    }

    return fields
}
```

### 日志中间件

```go
// HTTP日志中间件
func LoggingMiddleware(logger *StructuredLogger) gin.HandlerFunc {
    return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
        // 使用结构化日志记录HTTP请求
        contextLogger := logger.WithContext(param.Request.Context())

        fields := []zap.Field{
            zap.String("method", param.Method),
            zap.String("path", param.Path),
            zap.String("raw_query", param.Request.URL.RawQuery),
            zap.Int("status", param.StatusCode),
            zap.Duration("latency", param.Latency),
            zap.String("client_ip", param.ClientIP),
            zap.String("user_agent", param.Request.UserAgent()),
            zap.Int("body_size", param.BodySize),
        }

        if param.ErrorMessage != "" {
            fields = append(fields, zap.String("error", param.ErrorMessage))
            contextLogger.Error("HTTP request failed", fields...)
        } else {
            contextLogger.Info("HTTP request completed", fields...)
        }

        return ""
    })
}

// 错误日志记录
func ErrorLoggingMiddleware(logger *StructuredLogger) gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()

        // 记录所有错误
        for _, ginErr := range c.Errors {
            contextLogger := logger.WithContext(c.Request.Context())

            fields := []zap.Field{
                zap.String("error_type", ginErr.Type.String()),
                zap.String("error_message", ginErr.Error()),
            }

            if ginErr.Meta != nil {
                fields = append(fields, zap.Any("error_meta", ginErr.Meta))
            }

            contextLogger.Error("Request error occurred", fields...)
        }
    }
}
```

### 业务日志记录

```go
// 业务事件日志
type BusinessLogger struct {
    logger *StructuredLogger
}

func NewBusinessLogger(logger *StructuredLogger) *BusinessLogger {
    return &BusinessLogger{logger: logger}
}

func (bl *BusinessLogger) LogUserAction(ctx context.Context, userID, action string, details map[string]interface{}) {
    contextLogger := bl.logger.WithContext(ctx)

    fields := []zap.Field{
        zap.String("event_type", "user_action"),
        zap.String("user_id", userID),
        zap.String("action", action),
        zap.Any("details", details),
        zap.Time("timestamp", time.Now()),
    }

    contextLogger.Info("User action performed", fields...)
}

func (bl *BusinessLogger) LogTransaction(ctx context.Context, txn Transaction) {
    contextLogger := bl.logger.WithContext(ctx)

    fields := []zap.Field{
        zap.String("event_type", "transaction"),
        zap.String("transaction_id", txn.ID),
        zap.String("user_id", txn.UserID),
        zap.Float64("amount", txn.Amount),
        zap.String("currency", txn.Currency),
        zap.String("status", txn.Status),
        zap.String("payment_method", txn.PaymentMethod),
        zap.Time("created_at", txn.CreatedAt),
    }

    if txn.Status == "failed" {
        fields = append(fields, zap.String("error_reason", txn.ErrorReason))
        contextLogger.Error("Transaction failed", fields...)
    } else {
        contextLogger.Info("Transaction processed", fields...)
    }
}

func (bl *BusinessLogger) LogSecurityEvent(ctx context.Context, event SecurityEvent) {
    contextLogger := bl.logger.WithContext(ctx)

    fields := []zap.Field{
        zap.String("event_type", "security"),
        zap.String("security_event", event.Type),
        zap.String("severity", event.Severity),
        zap.String("source_ip", event.SourceIP),
        zap.String("user_agent", event.UserAgent),
        zap.Any("details", event.Details),
    }

    if event.UserID != "" {
        fields = append(fields, zap.String("user_id", event.UserID))
    }

    switch event.Severity {
    case "critical", "high":
        contextLogger.Error("Security event detected", fields...)
    case "medium":
        contextLogger.Warn("Security event detected", fields...)
    default:
        contextLogger.Info("Security event detected", fields...)
    }
}
```

## 健康检查

### 多层级健康检查

```go
// 健康检查配置
type HealthConfig struct {
    Endpoint         string
    CheckInterval    time.Duration
    Timeout          time.Duration
    FailureThreshold int
    SuccessThreshold int
}

// 健康状态
type HealthStatus string

const (
    HealthStatusHealthy   HealthStatus = "healthy"
    HealthStatusUnhealthy HealthStatus = "unhealthy"
    HealthStatusDegraded  HealthStatus = "degraded"
    HealthStatusUnknown   HealthStatus = "unknown"
)

// 健康检查器接口
type HealthChecker interface {
    Check(ctx context.Context) HealthCheckResult
    Name() string
    IsCritical() bool
}

// 健康检查结果
type HealthCheckResult struct {
    Status    HealthStatus  `json:"status"`
    Message   string        `json:"message,omitempty"`
    Duration  time.Duration `json:"duration"`
    Timestamp time.Time     `json:"timestamp"`
    Details   map[string]interface{} `json:"details,omitempty"`
}

// 健康检查管理器
type HealthManager struct {
    checkers map[string]HealthChecker
    cache    map[string]HealthCheckResult
    config   HealthConfig
    mutex    sync.RWMutex
}

func NewHealthManager(config HealthConfig) *HealthManager {
    hm := &HealthManager{
        checkers: make(map[string]HealthChecker),
        cache:    make(map[string]HealthCheckResult),
        config:   config,
    }

    go hm.startPeriodicChecks()
    return hm
}

func (hm *HealthManager) RegisterChecker(checker HealthChecker) {
    hm.mutex.Lock()
    defer hm.mutex.Unlock()

    hm.checkers[checker.Name()] = checker
}

func (hm *HealthManager) CheckAll(ctx context.Context) map[string]HealthCheckResult {
    hm.mutex.RLock()
    checkers := make(map[string]HealthChecker)
    for name, checker := range hm.checkers {
        checkers[name] = checker
    }
    hm.mutex.RUnlock()

    results := make(map[string]HealthCheckResult)
    var wg sync.WaitGroup

    for name, checker := range checkers {
        wg.Add(1)
        go func(name string, checker HealthChecker) {
            defer wg.Done()

            checkCtx, cancel := context.WithTimeout(ctx, hm.config.Timeout)
            defer cancel()

            result := checker.Check(checkCtx)
            results[name] = result
        }(name, checker)
    }

    wg.Wait()

    hm.mutex.Lock()
    for name, result := range results {
        hm.cache[name] = result
    }
    hm.mutex.Unlock()

    return results
}

func (hm *HealthManager) GetOverallHealth() HealthStatus {
    hm.mutex.RLock()
    defer hm.mutex.RUnlock()

    criticalFailed := false
    nonCriticalFailed := false

    for name, checker := range hm.checkers {
        if result, exists := hm.cache[name]; exists {
            if result.Status == HealthStatusUnhealthy {
                if checker.IsCritical() {
                    criticalFailed = true
                } else {
                    nonCriticalFailed = true
                }
            }
        }
    }

    if criticalFailed {
        return HealthStatusUnhealthy
    } else if nonCriticalFailed {
        return HealthStatusDegraded
    }

    return HealthStatusHealthy
}
```

### 具体健康检查器实现

```go
// 数据库健康检查器
type DatabaseHealthChecker struct {
    db       *sql.DB
    critical bool
}

func NewDatabaseHealthChecker(db *sql.DB, critical bool) *DatabaseHealthChecker {
    return &DatabaseHealthChecker{
        db:       db,
        critical: critical,
    }
}

func (dhc *DatabaseHealthChecker) Name() string {
    return "database"
}

func (dhc *DatabaseHealthChecker) IsCritical() bool {
    return dhc.critical
}

func (dhc *DatabaseHealthChecker) Check(ctx context.Context) HealthCheckResult {
    start := time.Now()
    result := HealthCheckResult{
        Timestamp: start,
    }

    err := dhc.db.PingContext(ctx)
    result.Duration = time.Since(start)

    if err != nil {
        result.Status = HealthStatusUnhealthy
        result.Message = fmt.Sprintf("Database connection failed: %v", err)
        return result
    }

    // 获取连接池状态
    stats := dhc.db.Stats()
    result.Status = HealthStatusHealthy
    result.Message = "Database is healthy"
    result.Details = map[string]interface{}{
        "open_connections": stats.OpenConnections,
        "idle":            stats.Idle,
        "in_use":          stats.InUse,
        "wait_count":      stats.WaitCount,
    }

    return result
}

// Redis健康检查器
type RedisHealthChecker struct {
    client   redis.Client
    critical bool
}

func (rhc *RedisHealthChecker) Check(ctx context.Context) HealthCheckResult {
    start := time.Now()
    result := HealthCheckResult{
        Timestamp: start,
    }

    pong, err := rhc.client.Ping(ctx).Result()
    result.Duration = time.Since(start)

    if err != nil || pong != "PONG" {
        result.Status = HealthStatusUnhealthy
        result.Message = fmt.Sprintf("Redis connection failed: %v", err)
        return result
    }

    result.Status = HealthStatusHealthy
    result.Message = "Redis is healthy"

    return result
}

// 外部API健康检查器
type ExternalAPIHealthChecker struct {
    url      string
    client   *http.Client
    critical bool
}

func (eahc *ExternalAPIHealthChecker) Check(ctx context.Context) HealthCheckResult {
    start := time.Now()
    result := HealthCheckResult{
        Timestamp: start,
    }

    req, err := http.NewRequestWithContext(ctx, "GET", eahc.url, nil)
    if err != nil {
        result.Status = HealthStatusUnhealthy
        result.Message = fmt.Sprintf("Failed to create request: %v", err)
        result.Duration = time.Since(start)
        return result
    }

    resp, err := eahc.client.Do(req)
    result.Duration = time.Since(start)

    if err != nil {
        result.Status = HealthStatusUnhealthy
        result.Message = fmt.Sprintf("External API unreachable: %v", err)
        return result
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 200 && resp.StatusCode < 300 {
        result.Status = HealthStatusHealthy
        result.Message = "External API is healthy"
    } else {
        result.Status = HealthStatusUnhealthy
        result.Message = fmt.Sprintf("External API returned status: %d", resp.StatusCode)
    }

    result.Details = map[string]interface{}{
        "status_code": resp.StatusCode,
        "url":         eahc.url,
    }

    return result
}
```

### 健康检查端点

```go
// 健康检查HTTP处理器
func HealthCheckHandler(hm *HealthManager) gin.HandlerFunc {
    return func(c *gin.Context) {
        ctx := c.Request.Context()

        // 检查是否请求详细信息
        detailed := c.Query("detailed") == "true"

        if detailed {
            results := hm.CheckAll(ctx)
            overallStatus := hm.GetOverallHealth()

            response := gin.H{
                "status": overallStatus,
                "timestamp": time.Now(),
                "checks": results,
            }

            statusCode := getHTTPStatusCode(overallStatus)
            c.JSON(statusCode, response)
        } else {
            // 简单健康检查
            overallStatus := hm.GetOverallHealth()
            statusCode := getHTTPStatusCode(overallStatus)

            c.JSON(statusCode, gin.H{
                "status": overallStatus,
                "timestamp": time.Now(),
            })
        }
    }
}

func getHTTPStatusCode(status HealthStatus) int {
    switch status {
    case HealthStatusHealthy:
        return 200
    case HealthStatusDegraded:
        return 200 // 或者 202
    case HealthStatusUnhealthy:
        return 503
    default:
        return 500
    }
}
```

## 配置示例

### 完整可观测性配置

```yaml
# observability-config.yaml
observability:
  # 追踪配置
  tracing:
    enabled: true
    service_name: "chi-service"
    service_version: "1.0.0"
    environment: "production"

    # Jaeger配置
    jaeger:
      endpoint: "http://jaeger:14268/api/traces"
      sample_ratio: 0.1
      batch_timeout: "5s"
      export_timeout: "30s"

  # 指标配置
  metrics:
    enabled: true
    namespace: "chi"
    subsystem: "api"

    # Prometheus配置
    prometheus:
      listen_address: ":9090"
      metrics_path: "/metrics"

    # 自定义指标
    custom_metrics:
      business_events:
        type: "counter"
        help: "Business events counter"
        labels: ["event_type", "status"]

      response_times:
        type: "histogram"
        help: "Response time histogram"
        labels: ["endpoint"]
        buckets: [0.1, 0.5, 1.0, 2.0, 5.0]

  # 日志配置
  logging:
    level: "info"
    format: "json"
    outputs: ["stdout", "file"]

    # 文件轮转
    rotation:
      max_size: 100 # MB
      max_age: 30   # days
      max_backups: 10
      compress: true

    # 采样配置
    sampling:
      initial: 100
      thereafter: 100

    # 全局字段
    fields:
      environment: "production"
      version: "1.0.0"

  # 健康检查配置
  health:
    enabled: true
    endpoint: "/health"
    check_interval: "30s"
    timeout: "5s"
    failure_threshold: 3
    success_threshold: 1

    # 检查器配置
    checkers:
      database:
        enabled: true
        critical: true
        timeout: "3s"

      redis:
        enabled: true
        critical: false
        timeout: "1s"

      external_apis:
        - name: "payment_api"
          url: "https://api.payment.com/health"
          critical: true
          timeout: "5s"
        - name: "notification_api"
          url: "https://api.notification.com/health"
          critical: false
          timeout: "3s"

  # 告警配置
  alerting:
    enabled: true

    # 告警规则
    rules:
      - name: "high_error_rate"
        expression: "rate(http_requests_total{status=~'5..'}[5m]) > 0.05"
        duration: "2m"
        severity: "critical"

      - name: "high_latency"
        expression: "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 1.0"
        duration: "5m"
        severity: "warning"

      - name: "service_down"
        expression: "up{job='chi-service'} == 0"
        duration: "1m"
        severity: "critical"

  # 仪表板配置
  dashboard:
    enabled: true
    grafana:
      url: "http://grafana:3000"
      dashboard_configs:
        - name: "chi_overview"
          file: "dashboards/overview.json"
        - name: "chi_performance"
          file: "dashboards/performance.json"
```

通过以上完整的可观测性方案，Chi框架能够提供企业级的监控、追踪和日志能力，帮助开发和运维团队快速定位问题，优化系统性能，保障服务质量。