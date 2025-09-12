# Database 数据库包

基于 GORM 构建的综合数据库包，具有读写分离、慢查询监控、健康检查和连接池管理等高级功能。

## 功能特性

### 1. 读写分离
- 写操作自动路由到主数据库
- 读操作自动路由到从数据库
- 支持多个从数据库，可配置权重
- 使用 GORM 的 dbresolver 插件实现从数据库间的负载均衡

### 2. 外部日志集成
- 与外部 zap 日志实例无缝集成
- 回退到 GORM 内置日志系统
- 可配置日志级别（Silent、Error、Warn、Info）
- 结构化日志，包含上下文信息

### 3. 慢查询监控
- 自动检测和记录慢查询
- 可配置慢查询阈值
- 全面的统计跟踪（总计数、查询类型等）
- 最近慢查询历史记录
- 慢查询事件自定义日志记录

### 4. 连接池管理
- 可配置最大开放连接数
- 可配置最大空闲连接数
- 可配置连接最大生存时间
- 可配置连接最大空闲时间

### 5. 健康监控
- 对所有数据库连接进行定期健康检查
- 可配置检查间隔和超时时间
- 自动恢复检测
- 可配置阈值的失败计数
- 健康状态报告和日志记录

## 安装

```bash
go get github.com/tokmz/basic/pkg/database
```

## 快速开始

```go
package main

import (
    "log"
    "time"
    
    "github.com/tokmz/basic/pkg/database"
    "go.uber.org/zap"
)

func main() {
    // 创建配置
    config := &database.DatabaseConfig{
        Master: database.MasterConfig{
            Driver: "mysql",
            DSN:    "user:password@tcp(localhost:3306)/master_db?charset=utf8mb4&parseTime=True&loc=Local",
        },
        Slaves: []database.SlaveConfig{
            {
                Driver: "mysql",
                DSN:    "user:password@tcp(localhost:3307)/slave_db?charset=utf8mb4&parseTime=True&loc=Local",
                Weight: 2,
            },
        },
        Pool: database.PoolConfig{
            MaxOpenConns:    50,
            MaxIdleConns:    5,
            ConnMaxLifetime: time.Hour,
            ConnMaxIdleTime: time.Minute * 30,
        },
    }

    // 创建客户端
    client, err := database.NewClient(config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 使用数据库
    db := client.DB()
    // ... 执行数据库操作
}
```

## 配置说明

### DatabaseConfig 数据库配置

| 字段 | 类型 | 描述 |
|------|------|------|
| Master | MasterConfig | 主数据库配置 |
| Slaves | []SlaveConfig | 从数据库配置 |
| Pool | PoolConfig | 连接池设置 |
| Logger | LoggerConfig | 日志配置 |
| SlowQuery | SlowQueryConfig | 慢查询监控配置 |
| HealthCheck | HealthCheckConfig | 健康检查配置 |

### MasterConfig / SlaveConfig 主/从数据库配置

| 字段 | 类型 | 描述 |
|------|------|------|
| Driver | string | 数据库驱动（"mysql"、"postgres"） |
| DSN | string | 数据库连接字符串 |
| Weight | int | 负载均衡权重（仅从库） |

### PoolConfig 连接池配置

| 字段 | 类型 | 描述 |
|------|------|------|
| MaxOpenConns | int | 最大开放连接数 |
| MaxIdleConns | int | 最大空闲连接数 |
| ConnMaxLifetime | time.Duration | 连接最大生存时间 |
| ConnMaxIdleTime | time.Duration | 连接最大空闲时间 |

### LoggerConfig 日志配置

| 字段 | 类型 | 描述 |
|------|------|------|
| ZapLogger | *zap.Logger | 外部 zap 日志器（可选） |
| Level | LogLevel | 日志级别（Silent、Error、Warn、Info） |
| Colorful | bool | 启用彩色日志 |
| SlowThreshold | time.Duration | 慢查询阈值用于日志记录 |

### SlowQueryConfig 慢查询配置

| 字段 | 类型 | 描述 |
|------|------|------|
| Enabled | bool | 启用慢查询监控 |
| Threshold | time.Duration | 慢查询阈值 |
| Logger | *zap.Logger | 慢查询自定义日志器 |

### HealthCheckConfig 健康检查配置

| 字段 | 类型 | 描述 |
|------|------|------|
| Enabled | bool | 启用健康检查 |
| Interval | time.Duration | 健康检查间隔 |
| Timeout | time.Duration | 健康检查超时时间 |
| MaxFailures | int | 标记为不健康前的最大失败次数 |
| Logger | *zap.Logger | 健康事件自定义日志器 |

## 使用示例

### 基础用法

参见 [examples/basic/main.go](examples/basic/main.go) 获取完整的基础用法示例。

### 高级用法

参见 [examples/advanced/main.go](examples/advanced/main.go) 获取高级功能和生产就绪配置示例。

### 核心方法

#### Client 客户端方法

```go
// 获取底层 GORM DB 实例
db := client.DB()

// 检查整体健康状态
healthy := client.IsHealthy()

// 检查主库健康状态
masterHealthy := client.IsMasterHealthy()

// 获取健康状态详情
status := client.GetHealthStatus()

// 获取慢查询统计
stats := client.GetSlowQueryStats()

// 强制立即健康检查
client.ForceHealthCheck()

// 等待数据库变为健康状态
err := client.WaitForHealthy(time.Minute)

// 运行时更新 zap 日志器
err := client.SetZapLogger(newZapLogger)

// 关闭客户端
client.Close()
```

## 支持的数据库驱动

- **MySQL**: 使用 `gorm.io/driver/mysql`
- **PostgreSQL**: 使用 `gorm.io/driver/postgres`

可以通过扩展 `openDatabase` 方法轻松添加其他驱动。

## 默认配置

包通过 `DefaultConfig()` 提供合理的默认值：

```go
config := database.DefaultConfig()
// 根据需要自定义
config.Master.Driver = "mysql"
config.Master.DSN = "your-dsn-here"
```

默认设置：
- 连接池：100 个最大开放连接，10 个最大空闲连接
- 连接生存时间：1 小时
- 连接空闲时间：30 分钟
- 日志级别：Info，启用颜色
- 慢查询阈值：200ms
- 健康检查：每 5 分钟一次，30s 超时，3 次最大失败

## 错误处理

包提供全面的错误处理：
- 配置验证错误
- 连接建立错误
- 健康检查失败与自动恢复
- 从库不可用时的优雅降级

## 性能考虑

- 读操作自动在健康的从库间负载均衡
- 写操作始终路由到主数据库
- 连接池防止连接耗尽
- 慢查询监控有助于识别性能瓶颈
- 健康监控确保仅使用健康连接

## 线程安全

包完全线程安全，可在多个 goroutine 间安全使用。内部状态通过适当的同步机制保护。

## 测试

运行测试套件：

```bash
go test ./...
```

测试覆盖所有主要功能，包括：
- 配置验证
- 读写分离
- 慢查询监控
- 健康检查
- 连接池管理
- 错误处理场景

## 架构设计

### 核心组件

1. **Client 客户端** - 主要的数据库客户端，管理连接和功能协调
2. **Config 配置** - 配置结构和验证逻辑
3. **Logger 日志器** - Zap 日志集成和 GORM 日志适配
4. **SlowQuery 慢查询** - 慢查询检测和统计
5. **Health 健康检查** - 连接健康监控和管理

### 数据流

```
应用程序 → Client → GORM → dbresolver → 主库/从库
                 ↓
            日志/监控/健康检查
```

## 最佳实践

### 配置建议

1. **连接池大小**：根据应用并发量合理设置
2. **慢查询阈值**：生产环境建议 50-100ms
3. **健康检查间隔**：建议 1-5 分钟
4. **从库权重**：根据服务器性能分配

### 监控建议

1. 定期检查慢查询统计
2. 监控健康检查状态
3. 观察连接池使用情况
4. 设置健康状态告警

### 故障处理

1. 主库故障：应用层处理，考虑只读模式
2. 从库故障：自动排除，其他从库接管
3. 连接池耗尽：检查查询效率和并发控制
4. 慢查询过多：优化 SQL 和索引

## 更新日志

### v1.0.0
- 实现基础读写分离功能
- 添加慢查询监控
- 集成健康检查系统
- 支持 Zap 日志集成
- 完整的测试覆盖

## 贡献指南

欢迎提交 Issue 和 Pull Request。在提交代码前，请确保：

1. 通过所有测试
2. 添加相应的测试用例
3. 更新相关文档
4. 遵循现有代码风格

## 许可证

MIT 许可证 - 详见 LICENSE 文件。