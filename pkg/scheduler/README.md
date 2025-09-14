# 任务调度包 (Task Scheduler)

一个功能强大、企业级的Go任务调度库，支持多种任务类型、持久化存储、指标监控和事件系统。

## ✨ 特性

- 🔄 **多种任务类型**: 支持一次性任务、间隔任务、Cron表达式任务
- 🏭 **企业级功能**: 工作者池、任务优先级、重试机制、超时控制
- 💾 **持久化存储**: 支持内存和文件存储，可扩展其他存储后端
- 📊 **指标监控**: 内置指标收集，支持任务执行统计和性能监控
- 🎪 **事件系统**: 完整的事件发布订阅机制
- 🎣 **钩子系统**: 支持任务生命周期钩子
- 🔧 **高度可配置**: 丰富的配置选项适应不同场景
- 🧪 **完整测试**: 全面的单元测试覆盖

## 📦 安装

```bash
go get github.com/tokmz/basic/pkg/scheduler
```

## 🚀 快速开始

### 基本使用

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/tokmz/basic/pkg/scheduler"
)

func main() {
    // 创建调度器
    config := scheduler.DefaultSchedulerConfig()
    sched, err := scheduler.NewScheduler(config)
    if err != nil {
        panic(err)
    }

    // 启动调度器
    if err := sched.Start(); err != nil {
        panic(err)
    }
    defer sched.Stop()

    // 创建一个简单的任务
    task, err := scheduler.NewTaskBuilder("hello-task").
        WithDescription("每5秒打印Hello").
        WithType(scheduler.TaskTypeInterval).
        WithInterval(time.Second * 5).
        WithFunc(func(ctx context.Context, task *scheduler.Task) error {
            fmt.Printf("Hello from task %s at %s\n", task.Name, time.Now().Format("15:04:05"))
            return nil
        }).
        Build()

    if err != nil {
        panic(err)
    }

    // 添加任务到调度器
    if err := sched.AddTask(task); err != nil {
        panic(err)
    }

    // 让程序运行一段时间
    time.Sleep(time.Minute)
}
```

## 📚 详细用法

### 任务类型

#### 1. 一次性任务 (TaskTypeOnce)

```go
task, _ := scheduler.NewTaskBuilder("one-time-task").
    WithType(scheduler.TaskTypeOnce).
    WithFunc(func(ctx context.Context, task *scheduler.Task) error {
        fmt.Println("这个任务只会执行一次")
        return nil
    }).
    Build()
```

#### 2. 间隔任务 (TaskTypeInterval)

```go
task, _ := scheduler.NewTaskBuilder("interval-task").
    WithType(scheduler.TaskTypeInterval).
    WithInterval(time.Second * 30). // 每30秒执行一次
    WithFunc(func(ctx context.Context, task *scheduler.Task) error {
        fmt.Println("定时执行任务")
        return nil
    }).
    Build()
```

#### 3. Cron任务 (TaskTypeCron)

```go
task, _ := scheduler.NewTaskBuilder("cron-task").
    WithType(scheduler.TaskTypeCron).
    WithCron("0 9 * * 1-5"). // 工作日每天9点执行
    WithFunc(func(ctx context.Context, task *scheduler.Task) error {
        fmt.Println("工作日早上9点的例行任务")
        return nil
    }).
    Build()
```

### 任务配置

#### 优先级设置

```go
task, _ := scheduler.NewTaskBuilder("priority-task").
    WithPriority(scheduler.PriorityHigh). // 高优先级
    WithFunc(taskFunc).
    Build()
```

#### 重试配置

```go
task, _ := scheduler.NewTaskBuilder("retry-task").
    WithRetry(3, time.Second * 5). // 最多重试3次，间隔5秒
    WithFunc(taskFunc).
    Build()
```

#### 超时设置

```go
task, _ := scheduler.NewTaskBuilder("timeout-task").
    WithTimeout(time.Minute * 10). // 10分钟超时
    WithFunc(taskFunc).
    Build()
```

#### 标签和元数据

```go
task, _ := scheduler.NewTaskBuilder("tagged-task").
    WithTag("environment", "production").
    WithTag("team", "backend").
    WithMetadata("version", "1.2.3").
    WithFunc(taskFunc).
    Build()
```

### 调度器配置

#### 基本配置

```go
config := scheduler.DefaultSchedulerConfig()
config.Name = "my-scheduler"
config.MaxWorkers = 10                    // 最大工作者数量
config.WorkerQueueSize = 200             // 工作者队列大小
config.TickInterval = time.Second        // 调度检查间隔
config.DefaultTimeout = time.Minute * 5  // 默认任务超时
config.MaxRetries = 3                    // 默认最大重试次数
config.RetryDelay = time.Second * 10     // 默认重试延迟
```

#### 持久化配置

```go
config.EnablePersistence = true
config.PersistenceDriver = "file"
config.PersistenceConfig = map[string]interface{}{
    "base_dir": "./scheduler_data",
}
```

#### 指标监控配置

```go
config.EnableMetrics = true
config.MetricsInterval = time.Second * 30
```

### 任务管理

#### 列出任务

```go
// 列出所有任务
tasks, err := scheduler.ListTasks(nil)

// 使用过滤器
filter := &scheduler.TaskFilter{
    Types:    []scheduler.TaskType{scheduler.TaskTypeInterval},
    Statuses: []scheduler.TaskStatus{scheduler.TaskStatusRunning},
    Tags:     map[string]string{"environment": "production"},
}
tasks, err := scheduler.ListTasks(filter)
```

#### 获取单个任务

```go
task, err := scheduler.GetTask("task-id")
```

#### 暂停和恢复任务

```go
// 暂停任务
err := scheduler.PauseTask("task-id")

// 恢复任务
err := scheduler.ResumeTask("task-id")
```

#### 删除任务

```go
err := scheduler.RemoveTask("task-id")
```

### 监控和统计

#### 获取调度器状态

```go
stats := scheduler.GetStats()
fmt.Printf("总任务数: %d\n", stats.TotalTasks)
fmt.Printf("运行中任务: %d\n", stats.RunningTasks)
fmt.Printf("总执行次数: %d\n", stats.TotalExecutions)
fmt.Printf("成功次数: %d\n", stats.TotalSuccesses)
fmt.Printf("失败次数: %d\n", stats.TotalFailures)
```

#### 获取执行记录

```go
// 获取特定任务的执行记录
executions, err := scheduler.GetExecutions("task-id", 10)

// 获取所有执行记录
executions, err := scheduler.GetExecutions("", 100)
```

### 事件系统

调度器支持丰富的事件类型：

- `EventTaskAdded` - 任务添加
- `EventTaskRemoved` - 任务移除
- `EventTaskStarted` - 任务开始执行
- `EventTaskCompleted` - 任务完成
- `EventTaskFailed` - 任务失败
- `EventTaskRetrying` - 任务重试
- `EventSchedulerStarted` - 调度器启动
- `EventSchedulerStopped` - 调度器停止

### 钩子系统

调度器支持任务生命周期钩子：

- `HookBeforeTaskRun` - 任务执行前
- `HookAfterTaskRun` - 任务执行后
- `HookOnTaskSuccess` - 任务成功时
- `HookOnTaskFailure` - 任务失败时
- `HookOnTaskCancel` - 任务取消时
- `HookOnTaskRetry` - 任务重试时

## 🏗️ 架构设计

### 核心组件

1. **Scheduler** - 核心调度器，管理任务和工作者
2. **Worker** - 工作者协程，执行具体任务
3. **Task** - 任务定义和状态管理
4. **PersistenceStore** - 持久化存储接口
5. **MetricsCollector** - 指标收集器
6. **EventBus** - 事件总线
7. **HookManager** - 钩子管理器

### 设计原则

- **线程安全**: 所有组件都是线程安全的
- **可扩展**: 通过接口设计支持自定义实现
- **高性能**: 使用工作者池模式和非阻塞设计
- **可观测性**: 丰富的日志、指标和事件支持

## 🧪 测试

运行测试套件：

```bash
go test -v github.com/tokmz/basic/pkg/scheduler
```

运行特定测试：

```bash
go test -v -run TestSchedulerBasic github.com/tokmz/basic/pkg/scheduler
```

## 📝 示例

查看 `examples/scheduler_demo.go` 获取完整的使用示例。

运行示例：

```bash
go run examples/scheduler_demo.go
```

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT License

## 🙏 致谢

感谢所有贡献者和使用者！

---

如果这个项目对您有帮助，请给我们一个 ⭐️ Star！