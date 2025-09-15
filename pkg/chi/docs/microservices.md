# 微服务架构开发指南

## 概述

Chi 框架专为微服务架构设计，提供了完整的微服务开发、部署和运维解决方案。框架内置了服务发现、负载均衡、服务网格、配置管理、分布式追踪等核心功能，帮助开发者轻松构建大规模的微服务系统。

## 微服务架构设计

### 整体架构图
```
┌─────────────────────────────────────────────────────────────────┐
│                    Microservices Ecosystem                      │
│                                                                 │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐          │
│  │    API      │   │   Service   │   │  Service    │          │
│  │   Gateway   │   │   Mesh      │   │ Registry    │          │
│  └─────────────┘   └─────────────┘   └─────────────┘          │
│          │                 │                 │                 │
│          └─────────────────┼─────────────────┘                 │
│                            │                                   │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │              Microservices Layer                           ││
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐          ││
│  │  │  User   │ │ Order   │ │Payment  │ │Inventory│          ││
│  │  │Service  │ │Service  │ │Service  │ │Service  │          ││
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘          ││
│  └─────────────────────────────────────────────────────────────┘│
│                            │                                   │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │              Infrastructure Layer                          ││
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐          ││
│  │  │Database │ │Message  │ │ Cache   │ │Monitoring│          ││
│  │  │Cluster  │ │ Queue   │ │ Layer   │ │& Logging │          ││
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘          ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

### 核心组件

1. **API Gateway**: 统一的API入口，负责路由、认证、限流
2. **Service Mesh**: 服务间通信的基础设施层
3. **Service Registry**: 服务注册与发现中心
4. **Configuration Center**: 集中式配置管理
5. **Message Bus**: 异步消息通信
6. **Monitoring Stack**: 全方位监控和可观测性

## 微服务基础框架

### 微服务基类

```go
package microservice

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/your-org/basic/pkg/chi"
    "github.com/your-org/basic/pkg/chi/config"
    "github.com/your-org/basic/pkg/chi/discovery"
    "github.com/your-org/basic/pkg/chi/events"
    "github.com/your-org/basic/pkg/chi/governance"
    "github.com/your-org/basic/pkg/chi/observability"
)

// 微服务接口
type Microservice interface {
    // 服务基础信息
    Name() string
    Version() string
    Description() string

    // 生命周期管理
    Initialize(ctx context.Context) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Health() HealthStatus

    // 路由注册
    RegisterRoutes(router *gin.Engine)

    // 事件处理
    RegisterEventHandlers(eventBus events.EventBus)
}

// 微服务基础实现
type BaseService struct {
    name        string
    version     string
    description string
    config      *config.Config
    app         *chi.Application
    router      *gin.Engine

    // 核心组件
    serviceRegistry discovery.ServiceRegistry
    eventBus       events.EventBus
    healthChecker  *governance.HealthManager

    // 状态管理
    started      bool
    startTime    time.Time
    shutdownChan chan os.Signal
}

// 创建微服务实例
func NewMicroservice(name, version, description string, config *config.Config) *BaseService {
    service := &BaseService{
        name:         name,
        version:      version,
        description:  description,
        config:       config,
        shutdownChan: make(chan os.Signal, 1),
    }

    // 初始化Chi应用
    service.app = chi.NewWithConfig(config)
    service.router = service.app.Engine

    return service
}

func (bs *BaseService) Name() string        { return bs.name }
func (bs *BaseService) Version() string     { return bs.version }
func (bs *BaseService) Description() string { return bs.description }

// 初始化微服务
func (bs *BaseService) Initialize(ctx context.Context) error {
    // 初始化服务注册
    registry, err := discovery.NewConsulRegistry(&discovery.ConsulConfig{
        Address: bs.config.ServiceDiscovery.Consul.Address,
    })
    if err != nil {
        return fmt.Errorf("failed to create service registry: %w", err)
    }
    bs.serviceRegistry = registry

    // 初始化事件总线
    bs.eventBus = events.NewEventBus(bs.config.Events)

    // 初始化健康检查
    bs.healthChecker = governance.NewHealthManager(bs.config.Health)

    // 注册基础健康检查器
    bs.healthChecker.RegisterChecker(&BasicHealthChecker{service: bs})

    // 设置中间件
    bs.setupMiddleware()

    // 注册基础路由
    bs.registerBaseRoutes()

    return nil
}

// 启动微服务
func (bs *BaseService) Start(ctx context.Context) error {
    if bs.started {
        return fmt.Errorf("service already started")
    }

    // 注册服务到服务发现
    serviceConfig := discovery.ServiceConfig{
        Name:    bs.name,
        Version: bs.version,
        Address: bs.config.Server.Host,
        Port:    bs.config.Server.Port,
        Tags:    []string{bs.name, bs.version, "chi-microservice"},
        Meta: map[string]string{
            "description": bs.description,
            "framework":   "chi",
            "started_at":  time.Now().Format(time.RFC3339),
        },
        HealthCheck: discovery.HealthCheck{
            HTTP:     fmt.Sprintf("http://%s:%d/health", bs.config.Server.Host, bs.config.Server.Port),
            Interval: 10 * time.Second,
            Timeout:  3 * time.Second,
        },
    }

    if err := bs.serviceRegistry.Register(ctx, serviceConfig); err != nil {
        return fmt.Errorf("failed to register service: %w", err)
    }

    // 启动事件总线
    if err := bs.eventBus.Start(ctx); err != nil {
        return fmt.Errorf("failed to start event bus: %w", err)
    }

    // 设置优雅关闭
    bs.setupGracefulShutdown()

    // 启动HTTP服务器
    bs.started = true
    bs.startTime = time.Now()

    go func() {
        addr := fmt.Sprintf("%s:%d", bs.config.Server.Host, bs.config.Server.Port)
        if err := bs.router.Run(addr); err != nil {
            fmt.Printf("Failed to start server: %v\n", err)
        }
    }()

    fmt.Printf("Microservice %s:%s started at %s\n", bs.name, bs.version, bs.startTime.Format(time.RFC3339))
    return nil
}

// 停止微服务
func (bs *BaseService) Stop(ctx context.Context) error {
    if !bs.started {
        return nil
    }

    // 从服务发现注销
    if err := bs.serviceRegistry.Deregister(ctx, bs.name); err != nil {
        fmt.Printf("Failed to deregister service: %v\n", err)
    }

    // 停止事件总线
    if err := bs.eventBus.Stop(ctx); err != nil {
        fmt.Printf("Failed to stop event bus: %v\n", err)
    }

    bs.started = false
    fmt.Printf("Microservice %s stopped\n", bs.name)
    return nil
}

// 健康状态
func (bs *BaseService) Health() HealthStatus {
    if !bs.started {
        return HealthStatus{
            Status:  "down",
            Message: "Service not started",
        }
    }

    checks := bs.healthChecker.CheckAll(context.Background())
    overallStatus := bs.healthChecker.GetOverallHealth()

    return HealthStatus{
        Status:    string(overallStatus),
        Uptime:    time.Since(bs.startTime).String(),
        Checks:    checks,
        Timestamp: time.Now(),
    }
}

// 设置中间件
func (bs *BaseService) setupMiddleware() {
    // 请求追踪
    bs.router.Use(observability.TracingMiddleware())

    // 指标收集
    bs.router.Use(observability.MetricsMiddleware())

    // 服务治理
    bs.router.Use(governance.GovernanceMiddleware(bs.config.Governance))

    // 请求日志
    bs.router.Use(observability.LoggingMiddleware())
}

// 注册基础路由
func (bs *BaseService) registerBaseRoutes() {
    // 健康检查
    bs.router.GET("/health", func(c *gin.Context) {
        health := bs.Health()
        status := 200
        if health.Status != "healthy" {
            status = 503
        }
        c.JSON(status, health)
    })

    // 服务信息
    bs.router.GET("/info", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "name":        bs.name,
            "version":     bs.version,
            "description": bs.description,
            "uptime":      time.Since(bs.startTime).String(),
            "started_at":  bs.startTime.Format(time.RFC3339),
        })
    })

    // 指标端点
    bs.router.GET("/metrics", observability.PrometheusHandler())
}

// 优雅关闭
func (bs *BaseService) setupGracefulShutdown() {
    signal.Notify(bs.shutdownChan, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-bs.shutdownChan
        fmt.Println("Shutting down microservice...")

        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()

        bs.Stop(ctx)
        os.Exit(0)
    }()
}

// 基础健康检查器
type BasicHealthChecker struct {
    service *BaseService
}

func (bhc *BasicHealthChecker) Name() string {
    return "basic"
}

func (bhc *BasicHealthChecker) IsCritical() bool {
    return true
}

func (bhc *BasicHealthChecker) Check(ctx context.Context) governance.HealthCheckResult {
    result := governance.HealthCheckResult{
        Timestamp: time.Now(),
        Status:    governance.HealthStatusHealthy,
        Message:   "Service is running",
        Details: map[string]interface{}{
            "uptime":     time.Since(bhc.service.startTime).String(),
            "started_at": bhc.service.startTime.Format(time.RFC3339),
        },
    }

    if !bhc.service.started {
        result.Status = governance.HealthStatusUnhealthy
        result.Message = "Service not started"
    }

    return result
}

type HealthStatus struct {
    Status    string                                    `json:"status"`
    Message   string                                    `json:"message,omitempty"`
    Uptime    string                                    `json:"uptime,omitempty"`
    Checks    map[string]governance.HealthCheckResult   `json:"checks,omitempty"`
    Timestamp time.Time                                 `json:"timestamp"`
}
```

### 微服务生成器

```go
package generator

import (
    "fmt"
    "os"
    "path/filepath"
    "text/template"
)

// 微服务模板生成器
type ServiceGenerator struct {
    ProjectName string
    ServiceName string
    Version     string
    Author      string
    OutputDir   string
}

// 生成微服务项目
func (sg *ServiceGenerator) Generate() error {
    // 创建项目目录结构
    if err := sg.createDirectoryStructure(); err != nil {
        return err
    }

    // 生成代码文件
    if err := sg.generateFiles(); err != nil {
        return err
    }

    fmt.Printf("Microservice %s generated successfully in %s\n", sg.ServiceName, sg.OutputDir)
    return nil
}

// 创建目录结构
func (sg *ServiceGenerator) createDirectoryStructure() error {
    dirs := []string{
        filepath.Join(sg.OutputDir, "cmd", sg.ServiceName),
        filepath.Join(sg.OutputDir, "internal", "handler"),
        filepath.Join(sg.OutputDir, "internal", "service"),
        filepath.Join(sg.OutputDir, "internal", "repository"),
        filepath.Join(sg.OutputDir, "internal", "model"),
        filepath.Join(sg.OutputDir, "internal", "config"),
        filepath.Join(sg.OutputDir, "pkg"),
        filepath.Join(sg.OutputDir, "api", "proto"),
        filepath.Join(sg.OutputDir, "deployments"),
        filepath.Join(sg.OutputDir, "scripts"),
        filepath.Join(sg.OutputDir, "docs"),
        filepath.Join(sg.OutputDir, "test"),
    }

    for _, dir := range dirs {
        if err := os.MkdirAll(dir, 0755); err != nil {
            return fmt.Errorf("failed to create directory %s: %w", dir, err)
        }
    }

    return nil
}

// 生成代码文件
func (sg *ServiceGenerator) generateFiles() error {
    files := map[string]string{
        "main.go":                    mainTemplate,
        "internal/handler/handler.go": handlerTemplate,
        "internal/service/service.go": serviceTemplate,
        "internal/model/model.go":     modelTemplate,
        "internal/config/config.go":  configTemplate,
        "go.mod":                     goModTemplate,
        "Dockerfile":                 dockerfileTemplate,
        "docker-compose.yml":         dockerComposeTemplate,
        "deployments/k8s.yaml":       k8sTemplate,
        "README.md":                  readmeTemplate,
    }

    for filename, tmplContent := range files {
        if err := sg.generateFile(filename, tmplContent); err != nil {
            return err
        }
    }

    return nil
}

// 生成单个文件
func (sg *ServiceGenerator) generateFile(filename, tmplContent string) error {
    tmpl, err := template.New(filename).Parse(tmplContent)
    if err != nil {
        return fmt.Errorf("failed to parse template for %s: %w", filename, err)
    }

    filepath := filepath.Join(sg.OutputDir, filename)
    if err := os.MkdirAll(filepath.Dir(filepath), 0755); err != nil {
        return fmt.Errorf("failed to create directory for %s: %w", filename, err)
    }

    file, err := os.Create(filepath)
    if err != nil {
        return fmt.Errorf("failed to create file %s: %w", filename, err)
    }
    defer file.Close()

    if err := tmpl.Execute(file, sg); err != nil {
        return fmt.Errorf("failed to execute template for %s: %w", filename, err)
    }

    return nil
}

// CLI命令
func GenerateService(projectName, serviceName, version, author, outputDir string) error {
    generator := &ServiceGenerator{
        ProjectName: projectName,
        ServiceName: serviceName,
        Version:     version,
        Author:      author,
        OutputDir:   outputDir,
    }

    return generator.Generate()
}

// 模板定义
const mainTemplate = `package main

import (
    "context"
    "log"

    "{{.ProjectName}}/internal/config"
    "{{.ProjectName}}/internal/handler"
    "{{.ProjectName}}/internal/service"
    "github.com/your-org/basic/pkg/chi/microservice"
)

func main() {
    // 加载配置
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // 创建微服务实例
    ms := microservice.NewMicroservice(
        "{{.ServiceName}}",
        "{{.Version}}",
        "{{.ServiceName}} microservice",
        cfg,
    )

    // 初始化业务服务
    businessService := service.New{{.ServiceName}}Service()

    // 创建处理器
    handler := handler.New{{.ServiceName}}Handler(businessService)

    // 注册路由
    ms.RegisterRoutes = func(router *gin.Engine) {
        handler.RegisterRoutes(router)
    }

    // 初始化微服务
    ctx := context.Background()
    if err := ms.Initialize(ctx); err != nil {
        log.Fatalf("Failed to initialize microservice: %v", err)
    }

    // 启动微服务
    if err := ms.Start(ctx); err != nil {
        log.Fatalf("Failed to start microservice: %v", err)
    }

    // 等待关闭信号
    select {}
}
`

const handlerTemplate = `package handler

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
    "{{.ProjectName}}/internal/service"
    "{{.ProjectName}}/internal/model"
)

type {{.ServiceName}}Handler struct {
    service *service.{{.ServiceName}}Service
}

func New{{.ServiceName}}Handler(service *service.{{.ServiceName}}Service) *{{.ServiceName}}Handler {
    return &{{.ServiceName}}Handler{
        service: service,
    }
}

func (h *{{.ServiceName}}Handler) RegisterRoutes(router *gin.Engine) {
    api := router.Group("/api/v1")
    {
        api.GET("/{{.ServiceName | ToLower}}", h.List)
        api.GET("/{{.ServiceName | ToLower}}/:id", h.Get)
        api.POST("/{{.ServiceName | ToLower}}", h.Create)
        api.PUT("/{{.ServiceName | ToLower}}/:id", h.Update)
        api.DELETE("/{{.ServiceName | ToLower}}/:id", h.Delete)
    }
}

func (h *{{.ServiceName}}Handler) List(c *gin.Context) {
    items, err := h.service.List(c.Request.Context())
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "data": items,
        "count": len(items),
    })
}

func (h *{{.ServiceName}}Handler) Get(c *gin.Context) {
    id, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }

    item, err := h.service.GetByID(c.Request.Context(), id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *{{.ServiceName}}Handler) Create(c *gin.Context) {
    var req model.Create{{.ServiceName}}Request
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    item, err := h.service.Create(c.Request.Context(), &req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *{{.ServiceName}}Handler) Update(c *gin.Context) {
    id, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }

    var req model.Update{{.ServiceName}}Request
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    item, err := h.service.Update(c.Request.Context(), id, &req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *{{.ServiceName}}Handler) Delete(c *gin.Context) {
    id, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }

    if err := h.service.Delete(c.Request.Context(), id); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}
`

const serviceTemplate = `package service

import (
    "context"
    "time"

    "{{.ProjectName}}/internal/model"
)

type {{.ServiceName}}Service struct {
    // Add dependencies like repository, cache, etc.
}

func New{{.ServiceName}}Service() *{{.ServiceName}}Service {
    return &{{.ServiceName}}Service{}
}

func (s *{{.ServiceName}}Service) List(ctx context.Context) ([]*model.{{.ServiceName}}, error) {
    // TODO: Implement list logic
    return []*model.{{.ServiceName}}{}, nil
}

func (s *{{.ServiceName}}Service) GetByID(ctx context.Context, id int64) (*model.{{.ServiceName}}, error) {
    // TODO: Implement get by id logic
    return &model.{{.ServiceName}}{
        ID:        id,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }, nil
}

func (s *{{.ServiceName}}Service) Create(ctx context.Context, req *model.Create{{.ServiceName}}Request) (*model.{{.ServiceName}}, error) {
    // TODO: Implement create logic
    return &model.{{.ServiceName}}{
        ID:        1,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }, nil
}

func (s *{{.ServiceName}}Service) Update(ctx context.Context, id int64, req *model.Update{{.ServiceName}}Request) (*model.{{.ServiceName}}, error) {
    // TODO: Implement update logic
    return &model.{{.ServiceName}}{
        ID:        id,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }, nil
}

func (s *{{.ServiceName}}Service) Delete(ctx context.Context, id int64) error {
    // TODO: Implement delete logic
    return nil
}
`

const modelTemplate = `package model

import "time"

type {{.ServiceName}} struct {
    ID        int64     ` + "`json:\"id\" db:\"id\"`" + `
    CreatedAt time.Time ` + "`json:\"created_at\" db:\"created_at\"`" + `
    UpdatedAt time.Time ` + "`json:\"updated_at\" db:\"updated_at\"`" + `
    // Add your fields here
}

type Create{{.ServiceName}}Request struct {
    // Add create request fields here
}

type Update{{.ServiceName}}Request struct {
    // Add update request fields here
}
`

const configTemplate = `package config

import (
    "github.com/spf13/viper"
    chiconfig "github.com/your-org/basic/pkg/chi/config"
)

func Load() (*chiconfig.Config, error) {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath(".")
    viper.AddConfigPath("./config")

    viper.AutomaticEnv()

    if err := viper.ReadInConfig(); err != nil {
        return nil, err
    }

    var config chiconfig.Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, err
    }

    return &config, nil
}
`

const goModTemplate = `module {{.ProjectName}}

go 1.19

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/spf13/viper v1.16.0
    github.com/your-org/basic/pkg/chi v0.1.0
)
`

const dockerfileTemplate = `FROM golang:1.19-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/{{.ServiceName}}

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/main .
COPY config.yaml .

EXPOSE 8080

CMD ["./main"]
`

const dockerComposeTemplate = `version: '3.8'

services:
  {{.ServiceName}}:
    build: .
    ports:
      - "8080:8080"
    environment:
      - CHI_ENV=development
    depends_on:
      - postgres
      - redis

  postgres:
    image: postgres:13
    environment:
      POSTGRES_DB: {{.ServiceName}}_db
      POSTGRES_USER: {{.ServiceName}}_user
      POSTGRES_PASSWORD: {{.ServiceName}}_password
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  redis:
    image: redis:6-alpine
    ports:
      - "6379:6379"

volumes:
  postgres_data:
`

const k8sTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.ServiceName}}
spec:
  replicas: 3
  selector:
    matchLabels:
      app: {{.ServiceName}}
  template:
    metadata:
      labels:
        app: {{.ServiceName}}
    spec:
      containers:
      - name: {{.ServiceName}}
        image: {{.ProjectName}}/{{.ServiceName}}:{{.Version}}
        ports:
        - containerPort: 8080
        env:
        - name: CHI_ENV
          value: "production"

---
apiVersion: v1
kind: Service
metadata:
  name: {{.ServiceName}}-service
spec:
  selector:
    app: {{.ServiceName}}
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
`

const readmeTemplate = `# {{.ServiceName}} Microservice

## Overview

{{.ServiceName}} microservice built with Chi framework.

## Features

- RESTful API
- Service Discovery
- Health Checks
- Distributed Tracing
- Metrics Collection
- Configuration Management

## Quick Start

### Development

` + "```bash" + `
go run cmd/{{.ServiceName}}/main.go
` + "```" + `

### Docker

` + "```bash" + `
docker-compose up
` + "```" + `

### Kubernetes

` + "```bash" + `
kubectl apply -f deployments/k8s.yaml
` + "```" + `

## API Endpoints

- GET /api/v1/{{.ServiceName | ToLower}} - List all items
- GET /api/v1/{{.ServiceName | ToLower}}/:id - Get item by ID
- POST /api/v1/{{.ServiceName | ToLower}} - Create new item
- PUT /api/v1/{{.ServiceName | ToLower}}/:id - Update item
- DELETE /api/v1/{{.ServiceName | ToLower}}/:id - Delete item

## Health Check

- GET /health - Service health status
- GET /info - Service information
- GET /metrics - Prometheus metrics
`
```

## 服务间通信

### gRPC 通信

```go
package grpc

import (
    "context"
    "fmt"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "github.com/your-org/basic/pkg/chi/discovery"
    "github.com/your-org/basic/pkg/chi/observability"
)

// gRPC客户端管理器
type ClientManager struct {
    discovery   discovery.ServiceDiscovery
    connections map[string]*grpc.ClientConn
    mutex       sync.RWMutex
}

func NewClientManager(discovery discovery.ServiceDiscovery) *ClientManager {
    return &ClientManager{
        discovery:   discovery,
        connections: make(map[string]*grpc.ClientConn),
    }
}

// 获取服务连接
func (cm *ClientManager) GetConnection(serviceName string) (*grpc.ClientConn, error) {
    cm.mutex.RLock()
    if conn, exists := cm.connections[serviceName]; exists {
        cm.mutex.RUnlock()
        return conn, nil
    }
    cm.mutex.RUnlock()

    return cm.createConnection(serviceName)
}

func (cm *ClientManager) createConnection(serviceName string) (*grpc.ClientConn, error) {
    cm.mutex.Lock()
    defer cm.mutex.Unlock()

    // 双重检查
    if conn, exists := cm.connections[serviceName]; exists {
        return conn, nil
    }

    // 从服务发现获取实例
    instances, err := cm.discovery.Discover(serviceName)
    if err != nil {
        return nil, fmt.Errorf("failed to discover service %s: %w", serviceName, err)
    }

    if len(instances) == 0 {
        return nil, fmt.Errorf("no instances found for service %s", serviceName)
    }

    // 选择第一个健康实例
    instance := instances[0]
    target := fmt.Sprintf("%s:%d", instance.Address, instance.Port)

    // 创建连接
    conn, err := grpc.Dial(target,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithUnaryInterceptor(observability.TracingUnaryClientInterceptor()),
        grpc.WithStreamInterceptor(observability.TracingStreamClientInterceptor()),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to connect to %s: %w", target, err)
    }

    cm.connections[serviceName] = conn
    return conn, nil
}

// 服务客户端包装器
type ServiceClient struct {
    serviceName string
    manager     *ClientManager
}

func NewServiceClient(serviceName string, manager *ClientManager) *ServiceClient {
    return &ServiceClient{
        serviceName: serviceName,
        manager:     manager,
    }
}

func (sc *ServiceClient) Call(ctx context.Context, method string, req interface{}, resp interface{}) error {
    conn, err := sc.manager.GetConnection(sc.serviceName)
    if err != nil {
        return err
    }

    // 使用反射或预生成的客户端代码调用方法
    return sc.invoke(ctx, conn, method, req, resp)
}

func (sc *ServiceClient) invoke(ctx context.Context, conn *grpc.ClientConn, method string, req, resp interface{}) error {
    // 这里应该使用具体的gRPC客户端代码
    // 或者使用反射来调用方法
    return nil
}
```

### HTTP 客户端

```go
package http

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    "github.com/your-org/basic/pkg/chi/discovery"
    "github.com/your-org/basic/pkg/chi/governance"
    "github.com/your-org/basic/pkg/chi/observability"
)

// HTTP客户端
type Client struct {
    httpClient      *http.Client
    discovery       discovery.ServiceDiscovery
    loadBalancer    discovery.LoadBalancer
    circuitBreaker  governance.CircuitBreaker
    retryManager    *governance.RetryManager
}

func NewClient(discovery discovery.ServiceDiscovery) *Client {
    return &Client{
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
            Transport: &http.Transport{
                MaxIdleConns:       100,
                IdleConnTimeout:    90 * time.Second,
                DisableCompression: true,
            },
        },
        discovery:      discovery,
        loadBalancer:   discovery.NewRoundRobinBalancer(),
        circuitBreaker: governance.NewCircuitBreaker(governance.CircuitBreakerConfig{
            MaxRequests:      10,
            Interval:         time.Minute,
            Timeout:          30 * time.Second,
            FailureThreshold: 0.6,
        }),
        retryManager: governance.NewRetryManager(
            &governance.ExponentialBackoffStrategy{
                BaseDelay:   100 * time.Millisecond,
                MaxDelay:    5 * time.Second,
                Multiplier:  2.0,
                MaxAttempts: 3,
            },
        ),
    }
}

// 发送请求
func (c *Client) Request(ctx context.Context, serviceName, method, path string, body interface{}, response interface{}) error {
    return c.retryManager.Execute(ctx, func() error {
        return c.circuitBreaker.Execute(ctx, func() (interface{}, error) {
            return nil, c.doRequest(ctx, serviceName, method, path, body, response)
        })
    })
}

func (c *Client) doRequest(ctx context.Context, serviceName, method, path string, body, response interface{}) error {
    // 服务发现
    instances, err := c.discovery.Discover(serviceName)
    if err != nil {
        return fmt.Errorf("failed to discover service %s: %w", serviceName, err)
    }

    if len(instances) == 0 {
        return fmt.Errorf("no instances found for service %s", serviceName)
    }

    // 负载均衡选择实例
    instance, err := c.loadBalancer.Select(instances)
    if err != nil {
        return fmt.Errorf("failed to select instance: %w", err)
    }

    // 构建请求URL
    url := fmt.Sprintf("http://%s:%d%s", instance.Address, instance.Port, path)

    // 准备请求体
    var reqBody io.Reader
    if body != nil {
        bodyBytes, err := json.Marshal(body)
        if err != nil {
            return fmt.Errorf("failed to marshal request body: %w", err)
        }
        reqBody = bytes.NewReader(bodyBytes)
    }

    // 创建HTTP请求
    req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }

    // 设置请求头
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "application/json")

    // 添加追踪信息
    observability.InjectTracing(ctx, req)

    // 发送请求
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to send request: %w", err)
    }
    defer resp.Body.Close()

    // 检查响应状态
    if resp.StatusCode >= 400 {
        return fmt.Errorf("request failed with status %d", resp.StatusCode)
    }

    // 解析响应
    if response != nil {
        respBody, err := io.ReadAll(resp.Body)
        if err != nil {
            return fmt.Errorf("failed to read response body: %w", err)
        }

        if err := json.Unmarshal(respBody, response); err != nil {
            return fmt.Errorf("failed to unmarshal response: %w", err)
        }
    }

    return nil
}

// 便捷方法
func (c *Client) Get(ctx context.Context, serviceName, path string, response interface{}) error {
    return c.Request(ctx, serviceName, "GET", path, nil, response)
}

func (c *Client) Post(ctx context.Context, serviceName, path string, body, response interface{}) error {
    return c.Request(ctx, serviceName, "POST", path, body, response)
}

func (c *Client) Put(ctx context.Context, serviceName, path string, body, response interface{}) error {
    return c.Request(ctx, serviceName, "PUT", path, body, response)
}

func (c *Client) Delete(ctx context.Context, serviceName, path string) error {
    return c.Request(ctx, serviceName, "DELETE", path, nil, nil)
}
```

### 消息队列通信

```go
package messaging

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/rabbitmq/amqp091-go"
    "github.com/your-org/basic/pkg/chi/events"
)

// 消息总线客户端
type MessageBusClient struct {
    connection *amqp091.Connection
    channel    *amqp091.Channel
    exchanges  map[string]bool
    queues     map[string]bool
}

func NewMessageBusClient(amqpURL string) (*MessageBusClient, error) {
    conn, err := amqp091.Dial(amqpURL)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
    }

    ch, err := conn.Channel()
    if err != nil {
        return nil, fmt.Errorf("failed to open channel: %w", err)
    }

    return &MessageBusClient{
        connection: conn,
        channel:    ch,
        exchanges:  make(map[string]bool),
        queues:     make(map[string]bool),
    }, nil
}

// 发布消息
func (mbc *MessageBusClient) Publish(ctx context.Context, exchange, routingKey string, message interface{}) error {
    // 确保exchange存在
    if err := mbc.ensureExchange(exchange); err != nil {
        return err
    }

    // 序列化消息
    body, err := json.Marshal(message)
    if err != nil {
        return fmt.Errorf("failed to marshal message: %w", err)
    }

    // 发布消息
    return mbc.channel.PublishWithContext(
        ctx,
        exchange,
        routingKey,
        false, // mandatory
        false, // immediate
        amqp091.Publishing{
            ContentType:  "application/json",
            Body:         body,
            Timestamp:    time.Now(),
            DeliveryMode: amqp091.Persistent,
        },
    )
}

// 订阅消息
func (mbc *MessageBusClient) Subscribe(exchange, queue, routingKey string, handler MessageHandler) error {
    // 确保exchange和queue存在
    if err := mbc.ensureExchange(exchange); err != nil {
        return err
    }

    if err := mbc.ensureQueue(queue); err != nil {
        return err
    }

    // 绑定queue到exchange
    if err := mbc.channel.QueueBind(
        queue,
        routingKey,
        exchange,
        false,
        nil,
    ); err != nil {
        return fmt.Errorf("failed to bind queue: %w", err)
    }

    // 开始消费
    messages, err := mbc.channel.Consume(
        queue,
        "",    // consumer tag
        false, // auto-ack
        false, // exclusive
        false, // no-local
        false, // no-wait
        nil,   // args
    )
    if err != nil {
        return fmt.Errorf("failed to start consuming: %w", err)
    }

    // 处理消息
    go func() {
        for msg := range messages {
            ctx := context.Background()
            if err := handler.Handle(ctx, &msg); err != nil {
                msg.Nack(false, true) // requeue on error
            } else {
                msg.Ack(false)
            }
        }
    }()

    return nil
}

func (mbc *MessageBusClient) ensureExchange(name string) error {
    if mbc.exchanges[name] {
        return nil
    }

    err := mbc.channel.ExchangeDeclare(
        name,
        "topic", // type
        true,    // durable
        false,   // auto-deleted
        false,   // internal
        false,   // no-wait
        nil,     // arguments
    )
    if err != nil {
        return fmt.Errorf("failed to declare exchange: %w", err)
    }

    mbc.exchanges[name] = true
    return nil
}

func (mbc *MessageBusClient) ensureQueue(name string) error {
    if mbc.queues[name] {
        return nil
    }

    _, err := mbc.channel.QueueDeclare(
        name,
        true,  // durable
        false, // delete when unused
        false, // exclusive
        false, // no-wait
        nil,   // arguments
    )
    if err != nil {
        return fmt.Errorf("failed to declare queue: %w", err)
    }

    mbc.queues[name] = true
    return nil
}

// 消息处理器接口
type MessageHandler interface {
    Handle(ctx context.Context, msg *amqp091.Delivery) error
}

// 事件消息处理器
type EventMessageHandler struct {
    eventBus events.EventBus
}

func NewEventMessageHandler(eventBus events.EventBus) *EventMessageHandler {
    return &EventMessageHandler{eventBus: eventBus}
}

func (emh *EventMessageHandler) Handle(ctx context.Context, msg *amqp091.Delivery) error {
    var event events.BaseEvent
    if err := json.Unmarshal(msg.Body, &event); err != nil {
        return fmt.Errorf("failed to unmarshal event: %w", err)
    }

    return emh.eventBus.Publish(ctx, &event)
}
```

## API网关集成

### 网关配置

```go
package gateway

import (
    "context"
    "fmt"
    "net/http"
    "net/http/httputil"
    "net/url"
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/your-org/basic/pkg/chi"
    "github.com/your-org/basic/pkg/chi/discovery"
    "github.com/your-org/basic/pkg/chi/governance"
)

// API网关
type APIGateway struct {
    app             *chi.Application
    serviceRegistry discovery.ServiceDiscovery
    loadBalancer    discovery.LoadBalancer
    rateLimiter     governance.RateLimiter
    routes          map[string]*RouteConfig
}

// 路由配置
type RouteConfig struct {
    ServiceName   string            `yaml:"service_name"`
    Path          string            `yaml:"path"`
    StripPrefix   bool              `yaml:"strip_prefix"`
    Timeout       time.Duration     `yaml:"timeout"`
    RateLimit     int               `yaml:"rate_limit"`
    RequireAuth   bool              `yaml:"require_auth"`
    RequiredRoles []string          `yaml:"required_roles"`
    Headers       map[string]string `yaml:"headers"`
}

func NewAPIGateway(config *chi.Config) *APIGateway {
    return &APIGateway{
        app:          chi.NewWithConfig(config),
        routes:       make(map[string]*RouteConfig),
        loadBalancer: discovery.NewRoundRobinBalancer(),
    }
}

// 添加路由
func (gw *APIGateway) AddRoute(path string, config *RouteConfig) {
    gw.routes[path] = config

    // 注册到Gin路由器
    gw.app.Any(path, gw.proxyHandler(config))
    gw.app.Any(path+"/*any", gw.proxyHandler(config))
}

// 代理处理器
func (gw *APIGateway) proxyHandler(config *RouteConfig) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 认证检查
        if config.RequireAuth && !gw.isAuthenticated(c) {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
            return
        }

        // 权限检查
        if len(config.RequiredRoles) > 0 && !gw.hasRequiredRole(c, config.RequiredRoles) {
            c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
            return
        }

        // 限流检查
        if config.RateLimit > 0 && !gw.checkRateLimit(c, config.RateLimit) {
            c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
            return
        }

        // 服务发现
        instances, err := gw.serviceRegistry.Discover(config.ServiceName)
        if err != nil {
            c.JSON(http.StatusServiceUnavailable, gin.H{"error": "service unavailable"})
            return
        }

        if len(instances) == 0 {
            c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no service instances"})
            return
        }

        // 负载均衡
        instance, err := gw.loadBalancer.Select(instances)
        if err != nil {
            c.JSON(http.StatusServiceUnavailable, gin.H{"error": "load balancing failed"})
            return
        }

        // 创建代理
        target := &url.URL{
            Scheme: "http",
            Host:   fmt.Sprintf("%s:%d", instance.Address, instance.Port),
        }

        proxy := httputil.NewSingleHostReverseProxy(target)

        // 修改请求
        gw.modifyRequest(c, config)

        // 代理请求
        proxy.ServeHTTP(c.Writer, c.Request)
    }
}

func (gw *APIGateway) modifyRequest(c *gin.Context, config *RouteConfig) {
    // 移除路径前缀
    if config.StripPrefix {
        originalPath := c.Request.URL.Path
        newPath := strings.TrimPrefix(originalPath, config.Path)
        if newPath == "" {
            newPath = "/"
        }
        c.Request.URL.Path = newPath
    }

    // 添加自定义头
    for key, value := range config.Headers {
        c.Request.Header.Set(key, value)
    }

    // 添加追踪头
    if traceID := c.GetHeader("X-Trace-ID"); traceID == "" {
        c.Request.Header.Set("X-Trace-ID", generateTraceID())
    }

    // 添加网关信息
    c.Request.Header.Set("X-Gateway", "chi-gateway")
    c.Request.Header.Set("X-Forwarded-For", c.ClientIP())
}

func (gw *APIGateway) isAuthenticated(c *gin.Context) bool {
    token := c.GetHeader("Authorization")
    if token == "" {
        return false
    }

    // 验证JWT token
    return gw.validateJWTToken(token)
}

func (gw *APIGateway) hasRequiredRole(c *gin.Context, requiredRoles []string) bool {
    userRoles := gw.getUserRoles(c)
    for _, required := range requiredRoles {
        for _, userRole := range userRoles {
            if userRole == required {
                return true
            }
        }
    }
    return false
}

func (gw *APIGateway) checkRateLimit(c *gin.Context, limit int) bool {
    key := fmt.Sprintf("rate_limit:%s", c.ClientIP())
    return gw.rateLimiter.Allow(key, limit)
}

// 启动网关
func (gw *APIGateway) Start(ctx context.Context, port int) error {
    return gw.app.Run(fmt.Sprintf(":%d", port))
}
```

### 网关配置文件

```yaml
# gateway-config.yaml
gateway:
  port: 8080

  # 路由配置
  routes:
    # 用户服务路由
    - path: "/api/users"
      service_name: "user-service"
      strip_prefix: false
      timeout: "30s"
      rate_limit: 100
      require_auth: true
      required_roles: ["user", "admin"]
      headers:
        X-Service: "user-service"

    # 订单服务路由
    - path: "/api/orders"
      service_name: "order-service"
      strip_prefix: false
      timeout: "30s"
      rate_limit: 50
      require_auth: true
      required_roles: ["user", "admin"]

    # 支付服务路由
    - path: "/api/payments"
      service_name: "payment-service"
      strip_prefix: false
      timeout: "60s"
      rate_limit: 20
      require_auth: true
      required_roles: ["admin"]

  # 认证配置
  auth:
    jwt_secret: "your-secret-key"
    token_expiry: "24h"

  # 限流配置
  rate_limiting:
    enabled: true
    redis_url: "redis://localhost:6379"

  # 服务发现配置
  service_discovery:
    type: "consul"
    consul:
      address: "localhost:8500"
```

这样，Chi框架就具备了完整的微服务开发能力，包括：

1. **微服务基础框架** - 标准化的微服务结构和生命周期管理
2. **代码生成工具** - 快速生成微服务项目模板
3. **服务间通信** - gRPC、HTTP、消息队列多种通信方式
4. **API网关** - 统一的API入口和路由管理
5. **配置管理** - 集中式配置和环境管理

接下来我会继续完善其他微服务相关的文档。

<function_calls>
<invoke name="TodoWrite">
<parameter name="todos">[{"content": "\u521b\u5efa\u5fae\u670d\u52a1\u67b6\u6784\u6587\u6863", "status": "completed", "activeForm": "\u521b\u5efa\u5fae\u670d\u52a1\u67b6\u6784\u6587\u6863"}, {"content": "\u521b\u5eba\u5fae\u670d\u52a1\u901a\u4fe1\u6587\u6863", "status": "in_progress", "activeForm": "\u521b\u5eba\u5fae\u670d\u52a1\u901a\u4fe1\u6587\u6863"}, {"content": "\u521b\u5eba\u5fae\u670d\u52a1\u914d\u7f6e\u7ba1\u7406\u6587\u6863", "status": "pending", "activeForm": "\u521b\u5eba\u5fae\u670d\u52a1\u914d\u7f6e\u7ba1\u7406\u6587\u6863"}, {"content": "\u521b\u5eba\u5fae\u670d\u52a1\u90e8\u7f72\u6587\u6863", "status": "pending", "activeForm": "\u521b\u5eba\u5fae\u670d\u52a1\u90e8\u7f72\u6587\u6863"}, {"content": "\u521b\u5eba\u5fae\u670d\u52a1\u6700\u4f73\u5b9e\u8df5\u6587\u6863", "status": "pending", "activeForm": "\u521b\u5eba\u5fae\u670d\u52a1\u6700\u4f73\u5b9e\u8df5\u6587\u6863"}, {"content": "\u66f4\u65b0\u4e3bREADME\u6dfb\u52a0\u5fae\u670d\u52a1\u7279\u6027", "status": "pending", "activeForm": "\u66f4\u65b0\u4e3bREADME\u6dfb\u52a0\u5fae\u670d\u52a1\u7279\u6027"}]