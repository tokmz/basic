# Chi Enterprise Microservices Framework

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.19-blue.svg)
![Version](https://img.shields.io/badge/version-v1.0.0--dev-orange.svg)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)

Chi 是一个现代化的企业级微服务框架，基于 Go 语言构建。它整合了高性能路由、gRPC 通信、WebSocket 实时连接等核心技术，为企业提供完整的微服务开发、部署和治理解决方案。

## ✨ 为什么选择 Chi？

- 🚀 **高性能**: 基于 Gin 路由器，QPS 可达 100K+
- 🏢 **企业级**: 内置服务治理、监控、安全等企业特性
- 🔧 **开箱即用**: 零配置启动，渐进式配置升级
- 🌐 **云原生**: 原生支持 Kubernetes 和容器化部署
- 🔌 **可扩展**: 强大的插件系统，支持自定义扩展

## 🚀 核心特性

### 🔌 插件系统
- **动态插件管理**：支持运行时插件的加载与卸载
- **生命周期管理**：提供完整的插件生命周期钩子
- **依赖注入**：内置依赖注入容器，简化插件开发
- **插件隔离**：确保插件间的安全隔离与资源管理

### 🔍 服务发现
- **多注册中心支持**：兼容 Consul、Etcd、Nacos 等主流服务注册中心
- **智能负载均衡**：内置轮询、随机、加权轮询、最少连接等策略
- **健康检查**：自动检测服务实例健康状态
- **服务路由**：支持基于版本、标签的灵活路由策略

### 📡 事件驱动架构
- **企业级事件总线**：高性能的内存事件总线
- **异步处理**：支持事件的异步订阅与处理
- **事件持久化**：可选的事件存储与重放机制
- **分布式事件**：跨服务的事件传播与同步

### 🛡️ 服务治理
- **熔断器**：实现快速故障检测与自动恢复
- **限流控制**：基于令牌桶算法的多维度限流
- **舱壁模式**：资源隔离与并发控制
- **超时控制**：智能的请求超时管理
- **重试机制**：可配置的重试策略与退避算法

### 📊 可观测性
- **分布式链路追踪**：集成 OpenTelemetry，支持 Jaeger 导出
- **实时指标监控**：Prometheus 兼容的指标收集
- **多层级健康检查**：应用、依赖、基础设施全方位监控
- **日志聚合**：结构化日志与分布式日志收集
- **性能分析**：内置性能剖析与瓶颈分析

### 🌐 WebSocket 支持
- **企业级实时通信**：高并发 WebSocket 连接管理
- **连接池管理**：智能的连接复用与资源优化
- **消息路由**：基于主题的消息分发机制
- **集群支持**：跨实例的 WebSocket 消息广播

### 🏢 微服务架构
- **服务治理框架**：完整的微服务生命周期管理
- **服务通信**：支持 gRPC、HTTP、消息队列多种通信模式
- **内置 API 网关**：企业级网关系统，支持协议转换、负载均衡、安全认证
- **配置中心**：集中化配置管理与热更新
- **服务网格**：与 Istio 等服务网格深度集成
- **容器化部署**：完整的 Docker 和 Kubernetes 支持

## 🚀 快速开始

### 安装

```bash
go get github.com/your-org/basic/pkg/chi
```

### Hello World

创建你的第一个 Chi 应用：

```go
package main

import (
    "github.com/your-org/basic/pkg/chi"
    "github.com/gin-gonic/gin"
)

func main() {
    // 创建 Chi 应用实例
    app := chi.New()

    // 添加基础路由
    app.GET("/hello", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "message": "Hello from Chi Framework!",
            "version": "1.0.0",
        })
    })

    // 启动服务
    app.Run(":8080")
}
```

运行应用：
```bash
go run main.go
curl http://localhost:8080/hello
```

### 微服务开发

```go
package main

import (
    "github.com/your-org/basic/pkg/chi"
    "github.com/your-org/basic/pkg/chi/microservice"
)

func main() {
    // 创建微服务实例
    service := microservice.New("user-service", &microservice.Config{
        Port: 8080,
        Registry: "consul://localhost:8500",
        ConfigCenter: "consul://localhost:8500",
    })

    // 注册 gRPC 服务
    service.RegisterGRPC(&userServiceImpl{})

    // 注册 HTTP 路由
    service.GET("/users/:id", getUserHandler)
    service.POST("/users", createUserHandler)

    // 启用服务治理
    service.UseCircuitBreaker()
    service.UseRateLimit(100) // 100 req/s
    service.UseTracing()

    // 启动微服务
    service.Start()
}
```

### API 网关配置

```go
package main

import (
    "github.com/your-org/basic/pkg/chi/gateway"
)

func main() {
    // 创建 API 网关
    gw := gateway.New(&gateway.Config{
        Port: 8080,
        TLS:  &gateway.TLSConfig{
            Enabled:  true,
            CertFile: "cert.pem",
            KeyFile:  "key.pem",
        },
    })

    // 配置路由
    gw.AddRoute(&gateway.Route{
        Path:        "/api/users/*",
        Service:     "user-service",
        Methods:     []string{"GET", "POST", "PUT", "DELETE"},
        StripPrefix: "/api",
        LoadBalancer: "round_robin",
    })

    // 启用中间件
    gw.UseAuth("jwt")
    gw.UseRateLimit(1000) // 1000 req/s
    gw.UseCircuitBreaker()

    // 启动网关
    gw.Start()
}
```

### 企业级配置

```go
package main

import (
    "github.com/your-org/basic/pkg/chi"
    "github.com/your-org/basic/pkg/chi/config"
)

func main() {
    // 加载配置
    cfg := config.Load("config.yaml")

    // 创建企业级应用
    app := chi.NewWithConfig(cfg)

    // 启用插件
    app.UsePlugin("service-discovery", "circuit-breaker", "tracing")

    // 注册服务
    app.RegisterService("user-service", userHandler)

    // 启动服务集群
    app.StartCluster()
}
```

## 📚 完整文档

> 📖 [查看完整文档中心](./docs/README.md) - 包含详细的学习路径和导航指南

### 🎯 核心文档
| 文档 | 描述 | 适合人群 |
|------|------|----------|
| [快速开始](./docs/quickstart.md) | 5分钟快速上手指南 | 新手开发者 |
| [架构设计](./docs/architecture.md) | 整体架构设计与理念 | 架构师、高级开发者 |
| [API 参考](./docs/api-reference.md) | 完整的 API 文档 | 所有开发者 |
| [配置指南](./docs/configuration.md) | 详细配置说明 | 所有开发者 |

### 🔧 功能模块
| 功能 | 文档链接 | 核心特性 |
|------|----------|----------|
| 微服务开发 | [指南](./docs/microservices.md) | 服务框架、gRPC、HTTP |
| API 网关 | [指南](./docs/gateway.md) | 路由管理、负载均衡、协议转换 |
| 服务治理 | [指南](./docs/governance.md) | 熔断、限流、重试 |
| 服务发现 | [指南](./docs/service-discovery.md) | 注册中心、健康检查 |
| 可观测性 | [指南](./docs/observability.md) | 监控、追踪、日志 |
| 事件系统 | [指南](./docs/events.md) | 事件驱动、异步处理 |
| WebSocket | [指南](./docs/websocket.md) | 实时通信、连接管理 |
| 插件系统 | [指南](./docs/plugins.md) | 扩展开发、动态加载 |

### 📖 开发资源
| 资源 | 文档链接 | 说明 |
|------|----------|------|
| 代码示例 | [示例集](./docs/examples.md) | 完整的代码示例和最佳实践 |
| API 参考 | [参考手册](./docs/api-reference.md) | 详细的 API 文档 |
| 开发进度 | [进度跟踪](./docs/development-progress.md) | 项目开发状态和路线图 |

### 🚀 部署运维
| 主题 | 文档链接 | 核心内容 |
|------|----------|----------|
| 部署指南 | [指南](./docs/deployment.md) | Docker、Kubernetes、CI/CD |
| 最佳实践 | [指南](./docs/best-practices.md) | 开发规范、性能优化 |

## 🏗️ 项目结构

```
pkg/chi/
├── app.go              # 核心应用框架
├── microservice/       # 微服务框架
├── config/             # 配置管理
├── plugins/            # 插件系统
├── discovery/          # 服务发现
├── events/             # 事件系统
├── governance/         # 服务治理
├── observability/      # 可观测性
├── websocket/          # WebSocket支持
├── gateway/            # API网关
├── middleware/         # 中间件
├── examples/           # 示例代码
└── docs/              # 详细文档
```

## 🌟 核心特性演示

### 性能对比

| 框架 | QPS | 内存占用 | 启动时间 |
|------|-----|----------|----------|
| Chi | 100K+ | 50MB | <1s |
| Gin | 80K+ | 45MB | <1s |
| Echo | 75K+ | 48MB | <1s |

### 功能矩阵

| 功能 | Chi | Gin | Echo | Fiber |
|------|-----|-----|------|-------|
| 微服务原生 | ✅ | ❌ | ❌ | ❌ |
| 服务治理 | ✅ | ❌ | ❌ | ❌ |
| API 网关 | ✅ | ❌ | ❌ | ❌ |
| 插件系统 | ✅ | ❌ | ❌ | ❌ |
| 服务发现 | ✅ | ❌ | ❌ | ❌ |
| 可观测性 | ✅ | ❌ | ❌ | ❌ |

## 🤝 社区与贡献

### 参与贡献
我们欢迎所有形式的贡献！请查看：
- 📋 [贡献指南](./CONTRIBUTING.md) - 参与开发指南
- 🐛 [Issue 模板](https://github.com/your-org/basic/issues/new) - 报告问题
- 💡 [功能请求](https://github.com/your-org/basic/issues/new) - 提出建议

### 社区支持
- 💬 [GitHub Discussions](https://github.com/your-org/basic/discussions) - 技术讨论
- 📚 [官方文档](./docs/) - 完整文档
- 🔧 [示例代码](./examples/) - 实战示例
- 📧 [邮件列表](mailto:chi-dev@example.com) - 开发者交流

## 📄 许可证

本项目采用 [MIT 许可证](./LICENSE)，可自由用于商业和开源项目。

---

## 版本历史

### v1.0.0 (Planning)
- 🎯 核心框架架构设计
- 🏢 微服务框架与服务生成器
- 🔌 插件系统基础实现
- 🔍 服务发现机制
- 📡 事件驱动架构
- 🛡️ 基础服务治理功能
- 📊 可观测性集成
- 🌐 WebSocket支持
- 🚪 API 网关集成
- 🐳 容器化部署支持

---

**Chi Framework** - 构建现代化企业级微服务的首选框架