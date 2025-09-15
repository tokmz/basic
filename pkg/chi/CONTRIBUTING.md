# Chi 框架贡献指南

感谢您对 Chi 企业级微服务框架的关注！我们欢迎所有形式的贡献，包括但不限于代码提交、文档改进、问题报告和功能建议。

## 目录

- [行为准则](#行为准则)
- [贡献方式](#贡献方式)
- [开发环境搭建](#开发环境搭建)
- [代码规范](#代码规范)
- [提交流程](#提交流程)
- [问题报告](#问题报告)
- [功能请求](#功能请求)
- [文档贡献](#文档贡献)
- [发布流程](#发布流程)

## 行为准则

参与 Chi 项目的所有人员都应遵守我们的行为准则：

### 我们的承诺

为了创造一个开放和欢迎的环境，我们作为贡献者和维护者承诺：
- 尊重所有参与者，无论其经验水平、性别、性别认同、性取向、残疾、外貌、年龄、种族或宗教
- 接受建设性的批评
- 专注于对社区最有利的事情
- 对其他社区成员表示同理心

### 不当行为

不可接受的行为包括：
- 使用性化的语言或图像
- 人身攻击或侮辱性评论
- 公开或私下骚扰
- 未经许可发布他人私人信息
- 其他在专业环境中被认为不当的行为

## 贡献方式

您可以通过以下方式为 Chi 框架做出贡献：

### 🐛 报告问题
发现 bug 或有改进建议？请 [创建 Issue](https://github.com/your-org/basic/issues/new)

### 💡 功能请求
有新功能想法？请通过 [Feature Request](https://github.com/your-org/basic/issues/new) 与我们分享

### 🔧 代码贡献
- 修复 bug
- 添加新功能
- 改进性能
- 增强测试覆盖率

### 📚 文档改进
- 修正文档错误
- 添加使用示例
- 翻译文档
- 改进 API 文档

### 💬 社区支持
- 回答用户问题
- 参与技术讨论
- 分享使用经验

## 开发环境搭建

### 系统要求

- **Go**: 1.19 或更高版本
- **Git**: 用于版本控制
- **Make**: 用于构建脚本（可选）
- **Docker**: 用于本地测试环境（可选）

### 本地开发设置

1. **Fork 和克隆项目**
   ```bash
   # Fork 项目到您的 GitHub 账户
   git clone https://github.com/YOUR_USERNAME/basic.git
   cd basic/pkg/chi
   ```

2. **添加上游仓库**
   ```bash
   git remote add upstream https://github.com/your-org/basic.git
   ```

3. **安装依赖**
   ```bash
   go mod download
   ```

4. **运行测试**
   ```bash
   go test ./...
   ```

5. **启动示例服务**
   ```bash
   cd examples/simple
   go run main.go
   ```

### 开发工具推荐

- **IDE**: VS Code、GoLand、Vim/Neovim
- **Go 插件**: Go 语言扩展
- **代码格式化**: gofmt、goimports
- **代码检查**: golangci-lint
- **测试工具**: gotestsum

## 代码规范

### Go 代码规范

我们遵循标准的 Go 代码规范：

1. **格式化**: 使用 `gofmt` 格式化代码
2. **导入**: 使用 `goimports` 整理导入
3. **命名**: 遵循 Go 命名约定
4. **注释**: 为公开的函数、类型和包添加文档注释

### 示例代码规范

```go
// Package microservice provides enterprise-grade microservice framework
// for building scalable and maintainable distributed systems.
package microservice

import (
    "context"
    "fmt"
    "log"

    "github.com/gin-gonic/gin"
    "google.golang.org/grpc"
)

// Service represents a microservice instance with built-in service governance,
// discovery, and observability capabilities.
type Service struct {
    name   string
    config *Config
    engine *gin.Engine
}

// New creates a new microservice instance with the specified name and configuration.
// It initializes the service with default middleware and prepares it for registration
// with service discovery systems.
func New(serviceName string, config *Config) *Service {
    if serviceName == "" {
        log.Fatal("service name cannot be empty")
    }

    return &Service{
        name:   serviceName,
        config: config,
        engine: gin.Default(),
    }
}
```

### 错误处理

```go
// 好的错误处理
func (s *Service) Start() error {
    if err := s.validateConfig(); err != nil {
        return fmt.Errorf("invalid configuration: %w", err)
    }

    if err := s.registerService(); err != nil {
        return fmt.Errorf("failed to register service: %w", err)
    }

    return s.engine.Run(fmt.Sprintf(":%d", s.config.Port))
}

// 避免的错误处理
func (s *Service) Start() error {
    s.validateConfig() // 忽略错误
    s.registerService() // 忽略错误
    return s.engine.Run(fmt.Sprintf(":%d", s.config.Port))
}
```

### 测试规范

```go
func TestService_New(t *testing.T) {
    tests := []struct {
        name        string
        serviceName string
        config      *Config
        want        *Service
        wantErr     bool
    }{
        {
            name:        "valid service",
            serviceName: "test-service",
            config:      &Config{Port: 8080},
            want: &Service{
                name:   "test-service",
                config: &Config{Port: 8080},
            },
            wantErr: false,
        },
        {
            name:        "empty service name",
            serviceName: "",
            config:      &Config{},
            want:        nil,
            wantErr:     true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := New(tt.serviceName, tt.config)
            if tt.wantErr {
                // 验证错误情况
                return
            }

            if got.name != tt.want.name {
                t.Errorf("New() name = %v, want %v", got.name, tt.want.name)
            }
        })
    }
}
```

## 提交流程

### Git 工作流

我们使用 GitHub Flow 工作流：

1. **创建功能分支**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **开发功能**
   - 编写代码
   - 添加测试
   - 更新文档

3. **提交更改**
   ```bash
   git add .
   git commit -m "feat: add user authentication middleware"
   ```

4. **推送分支**
   ```bash
   git push origin feature/your-feature-name
   ```

5. **创建 Pull Request**
   - 在 GitHub 上创建 PR
   - 填写 PR 模板
   - 等待代码审查

### 提交信息规范

我们使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

**类型 (type)**:
- `feat`: 新功能
- `fix`: 修复 bug
- `docs`: 文档更新
- `style`: 代码格式化
- `refactor`: 重构
- `perf`: 性能优化
- `test`: 测试相关
- `chore`: 构建工具或依赖更新

**示例**:
```
feat(microservice): add circuit breaker middleware

Add circuit breaker middleware to prevent cascading failures
in microservice communication.

Closes #123
```

### Pull Request 指南

#### PR 标题
- 使用清晰简洁的标题
- 包含变更类型前缀
- 不超过 50 个字符

#### PR 描述
请在 PR 描述中包含：

```markdown
## 变更说明
简要描述这个 PR 做了什么

## 变更类型
- [ ] Bug 修复
- [ ] 新功能
- [ ] 破坏性变更
- [ ] 文档更新
- [ ] 代码重构
- [ ] 性能优化
- [ ] 测试改进

## 测试
描述如何测试这些变更

## 检查清单
- [ ] 代码通过所有测试
- [ ] 代码遵循项目规范
- [ ] 已添加相应测试
- [ ] 已更新相关文档
- [ ] 提交信息符合规范

## 相关 Issue
Closes #123
```

## 问题报告

### Bug 报告模板

```markdown
## Bug 描述
清晰简洁地描述 bug

## 复现步骤
1. 执行 '...'
2. 点击 '....'
3. 滚动到 '....'
4. 看到错误

## 期望行为
描述您期望发生的事情

## 实际行为
描述实际发生的事情

## 环境信息
- OS: [例如 macOS, Linux, Windows]
- Go 版本: [例如 1.20]
- Chi 版本: [例如 v1.0.0]

## 附加信息
添加任何其他关于问题的上下文信息
```

### 问题分类

使用标签来分类问题：
- `bug`: 软件缺陷
- `enhancement`: 功能改进
- `documentation`: 文档相关
- `question`: 使用问题
- `good first issue`: 适合新贡献者
- `help wanted`: 需要帮助

## 功能请求

### 功能请求模板

```markdown
## 功能描述
清晰简洁地描述您想要的功能

## 问题描述
描述这个功能要解决的问题

## 解决方案
描述您想要的解决方案

## 替代方案
描述您考虑过的其他替代方案

## 附加信息
添加任何其他关于功能请求的上下文信息
```

### 评估标准

我们会根据以下标准评估功能请求：
- 是否符合项目目标
- 是否有广泛的用户需求
- 实现复杂度
- 维护成本
- 向后兼容性

## 文档贡献

### 文档类型

- **API 文档**: 函数、方法、类型的说明
- **用户指南**: 使用教程和最佳实践
- **开发者文档**: 架构设计和内部实现
- **示例代码**: 实际使用场景演示

### 文档规范

1. **Markdown 格式**: 所有文档使用 Markdown 格式
2. **结构清晰**: 使用适当的标题层级
3. **代码示例**: 提供完整可运行的示例
4. **链接检查**: 确保所有链接有效

### 文档审查

文档变更需要通过审查：
- 技术准确性
- 语言流畅性
- 格式规范性
- 完整性

## 发布流程

### 版本管理

我们使用 [语义化版本](https://semver.org/)：
- `MAJOR`: 不兼容的 API 变更
- `MINOR`: 向后兼容的功能添加
- `PATCH`: 向后兼容的 bug 修复

### 发布准备

发布前检查清单：
- [ ] 所有测试通过
- [ ] 文档已更新
- [ ] 变更日志已更新
- [ ] 版本号已更新
- [ ] 示例代码已验证

### 发布流程

1. **创建发布分支**
   ```bash
   git checkout -b release/v1.1.0
   ```

2. **更新版本信息**
   - 更新 `version.go`
   - 更新 `CHANGELOG.md`
   - 更新文档版本

3. **创建发布 PR**
   - 合并到 main 分支
   - 创建 Git 标签

4. **发布通知**
   - GitHub Release
   - 社区公告

## 社区参与

### 讨论平台

- **GitHub Discussions**: 技术讨论和问答
- **GitHub Issues**: 问题报告和功能请求
- **Pull Requests**: 代码审查和讨论

### 获得帮助

如果您在贡献过程中遇到问题：

1. 查看现有的 [Issues](https://github.com/your-org/basic/issues)
2. 搜索 [Discussions](https://github.com/your-org/basic/discussions)
3. 创建新的 Issue 或 Discussion
4. 联系维护者

### 认可贡献者

我们会在以下方面认可贡献者：
- 在 README 中列出贡献者
- 在发布说明中感谢贡献者
- 通过 GitHub 徽章表彰活跃贡献者

## 许可证

通过贡献代码，您同意您的贡献将按照与项目相同的 [MIT 许可证](./LICENSE) 进行许可。

---

再次感谢您的贡献！Chi 框架因为像您这样的贡献者而变得更好。

**最后更新**: 2025-09-15
**文档版本**: v1.0.0