# Basic - Go 常用库封装项目

一个采用 Go workspace 模式的项目，用于封装常用的 Go 库，实现模块化设计，便于维护和扩展。

## 🏗️ 项目架构

### Go Workspace 模式

本项目采用 Go 1.18+ 的 workspace 功能，实现多模块协同开发，同时保持各模块的独立性。

```
basic/
├── go.work                 # Go workspace 配置
├── README.md              # 项目总览文档
├── CHANGELOG.md           # 变更日志
├── CONCURRENT_SAFETY.md   # 并发安全文档
└── pkg/                   # 核心包目录
    └── database/          # 数据库包
        ├── README.md      # 数据库包文档
        ├── go.mod         # 模块定义
        ├── *.go           # 核心实现
        ├── *_test.go      # 单元测试
        ├── concurrent_test.go # 并发安全测试
        └── examples/      # 使用示例
            ├── basic/     # 基础用法示例
            └── advanced/  # 高级用法示例
```

## 📦 已实现的包

### Database 包 - 高级数据库操作封装

基于 GORM 的企业级数据库包，提供以下核心功能：

#### ✨ 核心特性

1. **读写分离** - 自动路由读写操作到不同数据库
   - 写操作 → 主库 (Master)
   - 读操作 → 从库 (Slaves) 
   - 支持多从库负载均衡
   - 可配置权重分配

2. **外部日志集成** - 无缝集成 Zap 日志系统
   - 支持外部 Zap 日志实例
   - 可配置日志级别
   - 结构化日志输出

3. **慢查询监控** - 智能监控查询性能
   - 可配置慢查询阈值
   - 查询类型统计分析
   - 最近慢查询记录
   - 实时统计报告

4. **连接池管理** - 精细化连接池控制
   - 最大连接数配置
   - 空闲连接管理
   - 连接生命周期控制

5. **健康监控** - 主动健康状态监控
   - 定期健康检查
   - 自动故障恢复
   - 健康状态报告
   - 告警机制

#### 🔒 并发安全保证

**重要改进：** 项目已经过专门的并发安全优化，解决了原有的竞态条件问题：

- **线程安全的深拷贝** - 使用 JSON 序列化确保数据完整性
- **原子性统计更新** - SlowQueryMonitor 统计更新完全原子化
- **优化的锁策略** - 减少锁争用，提升并发性能
- **全面的竞态条件测试** - 包含专门的并发安全测试套件

详见 [并发安全文档](CONCURRENT_SAFETY.md)

#### 🚀 快速开始

```go
package main

import (
    \"time\"
    \"github.com/tokmz/basic/pkg/database\"
)

func main() {
    // 配置数据库
    config := &database.DatabaseConfig{
        Master: database.MasterConfig{
            Driver: \"mysql\",
            DSN: \"user:pass@tcp(localhost:3306)/master_db\",
        },
        Slaves: []database.SlaveConfig{
            {
                Driver: \"mysql\",
                DSN: \"user:pass@tcp(localhost:3307)/slave_db\",
                Weight: 2, // 权重配置
            },
        },
        // ... 其他配置
    }

    // 创建客户端
    client, err := database.NewClient(config)
    if err != nil {
        panic(err)
    }
    defer client.Close()

    // 使用数据库
    db := client.DB()
    // db.Create(...) -> 路由到主库
    // db.Find(...) -> 路由到从库
}
```

更多详细用法请参考：[Database 包文档](pkg/database/README.md)

## 🛠️ 开发指南

### 环境要求

- Go 1.24+
- 支持 Go Modules 和 Workspaces

### 本地开发

```bash
# 克隆项目
git clone <repository-url>
cd basic

# Go workspace 会自动识别所有模块
go mod tidy

# 运行测试
go test ./...

# 并发安全测试
go test -race ./...
```

### 添加新包

1. 在 `pkg/` 目录下创建新包
2. 初始化 go.mod: `go mod init github.com/tokmz/basic/pkg/<package-name>`
3. 更新根目录 `go.work` 文件，添加新包路径
4. 实现功能并编写测试
5. 添加使用示例和文档

### 代码质量标准

- **单元测试覆盖率** ≥ 80%
- **并发安全测试** 必须通过 `go test -race`
- **代码文档** 所有公开 API 必须有完整注释
- **错误处理** 统一的错误处理和日志记录
- **性能基准** 关键路径需要性能基准测试

## 🧪 测试策略

### 测试类型

1. **单元测试** - `*_test.go`
   - 功能正确性验证
   - 边界条件测试
   - 错误场景覆盖

2. **并发安全测试** - `concurrent_test.go`
   - 竞态条件检测
   - 高并发场景验证  
   - 内存安全测试

3. **集成测试** - `examples/`
   - 实际使用场景验证
   - 多组件协作测试

4. **性能测试** - `Benchmark*`
   - 关键路径性能验证
   - 内存使用分析
   - 并发性能测试

### 运行测试

```bash
# 所有测试
go test ./...

# 并发安全测试
go test -race ./...

# 性能基准测试
go test -bench=. ./...

# 测试覆盖率
go test -cover ./...

# 详细测试报告
go test -v -race -cover ./...
```

## 📊 项目状态

### Database 包状态

| 功能模块 | 实现状态 | 测试状态 | 并发安全 |
|----------|----------|----------|----------|
| 读写分离 | ✅ 完成 | ✅ 通过 | ✅ 安全 |
| 日志集成 | ✅ 完成 | ✅ 通过 | ✅ 安全 |
| 慢查询监控 | ✅ 完成 | ✅ 通过 | ✅ 安全 |
| 连接池管理 | ✅ 完成 | ✅ 通过 | ✅ 安全 |
| 健康监控 | ✅ 完成 | ✅ 通过 | ✅ 安全 |

### 规划中的包

- **Cache 包** - Redis/内存缓存封装
- **Config 包** - 配置管理和热重载
- **Logger 包** - 统一日志处理
- **Server 包** - HTTP/gRPC 服务框架

## 🤝 贡献指南

### 贡献流程

1. Fork 项目
2. 创建功能分支: `git checkout -b feature/new-package`
3. 提交更改: `git commit -am 'Add new package'`
4. 推送分支: `git push origin feature/new-package`
5. 提交 Pull Request

### 代码规范

- 遵循 Go 官方代码规范
- 使用 `gofmt` 格式化代码
- 通过 `go vet` 静态检查
- 添加必要的单元测试
- 更新相关文档

### 提交消息格式

```
<type>(<scope>): <subject>

<body>

<footer>
```

类型说明：
- `feat`: 新功能
- `fix`: 修复 bug
- `docs`: 文档更新
- `style`: 代码格式调整
- `refactor`: 重构代码
- `test`: 测试相关
- `chore`: 构建/工具相关

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

## 🔗 相关链接

- [Database 包文档](pkg/database/README.md)
- [并发安全指南](CONCURRENT_SAFETY.md)
- [变更日志](CHANGELOG.md)
- [Go Workspace 官方文档](https://go.dev/doc/tutorial/workspaces)

## 📞 支持与反馈

如遇问题或有建议，欢迎：

- 提交 [Issues](../../issues)
- 发起 [Pull Requests](../../pulls)
- 参与 [Discussions](../../discussions)

---

**⚡ 高性能 • 🔒 线程安全 • 📦 模块化 • 🚀 生产就绪**