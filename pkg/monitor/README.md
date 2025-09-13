# Monitor Package - 企业级系统监控包

一个功能完整的Go语言系统监控包，提供实时系统资源监控、告警管理、数据持久化等企业级功能。

## 🌟 核心特性

### 📊 全面的系统监控
- **CPU监控** - 实时CPU使用率、核心数统计
- **内存监控** - 内存使用量、可用量、交换分区监控
- **磁盘监控** - 磁盘使用率、I/O统计、挂载点信息
- **网络监控** - 网络接口状态、流量统计、连接信息
- **进程监控** - 进程列表、资源使用、进程树分析
- **系统负载** - 系统负载平均值监控

### 🚨 智能告警系统
- **规则引擎** - 灵活的告警规则配置
- **多级告警** - 支持Info、Warning、Critical等多种告警级别
- **告警处理器** - 支持日志、邮件、Webhook等多种通知方式
- **告警聚合** - 防止告警风暴，支持告警去重和聚合
- **历史记录** - 完整的告警历史记录和状态跟踪

### 💾 数据持久化
- **文件存储** - 高效的文件存储系统，支持数据压缩和轮转
- **内存存储** - 高性能内存存储，适用于临时数据
- **查询接口** - 支持时间范围、标签过滤等复杂查询
- **数据保留** - 自动数据清理和保留策略

### 📈 数据聚合分析
- **实时聚合** - 实时计算平均值、最大值、最小值等统计指标
- **历史趋势** - 支持历史数据趋势分析
- **缓存优化** - 内置缓存机制，提高查询性能
- **分页查询** - 支持大数据量的分页查询

### 🔧 高度可扩展
- **自定义收集器** - 支持自定义监控指标收集器
- **插件架构** - 模块化设计，易于扩展
- **配置驱动** - 灵活的配置系统，支持动态配置
- **跨平台** - 支持Linux、macOS、Windows等多种操作系统

## 📦 安装

```bash
go get github.com/tokmz/basic/pkg/monitor
```

## 🚀 快速开始

### 基本使用

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "github.com/tokmz/basic/pkg/monitor"
)

func main() {
    // 1. 创建监控器
    config := monitor.DefaultMonitorConfig()
    monitor := monitor.NewSystemMonitor(config)
    
    // 2. 获取系统信息
    systemInfo, err := monitor.GetSystemInfo()
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Hostname: %s\n", systemInfo.Hostname)
    fmt.Printf("OS: %s\n", systemInfo.OS)
    
    if systemInfo.CPU != nil {
        fmt.Printf("CPU Cores: %d, Usage: %.2f%%\n", 
            systemInfo.CPU.Cores, systemInfo.CPU.Usage)
    }
    
    if systemInfo.Memory != nil {
        fmt.Printf("Memory Usage: %.2f%%\n", systemInfo.Memory.UsagePercent)
    }
}
```

### 带监控循环的使用

```go
func main() {
    // 创建配置
    config := &monitor.MonitorConfig{
        Interval:      5 * time.Second,
        EnableCPU:     true,
        EnableMemory:  true,
        EnableDisk:    true,
        EnableNetwork: true,
        HistorySize:   1000,
    }
    
    monitor := monitor.NewSystemMonitor(config)
    
    // 启动监控
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    
    err := monitor.Start(ctx)
    if err != nil {
        log.Fatal(err)
    }
    defer monitor.Stop()
    
    // 定期获取监控数据
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            summary := monitor.GetSystemSummary()
            if summary != nil {
                fmt.Printf("CPU: %.2f%%, Memory: %.2f%%\n", 
                    summary.CPU.Current, summary.Memory.UsagePercent)
            }
        }
    }
}
```

## 🔔 告警配置

### 设置告警规则

```go
// CPU使用率告警
cpuAlert := &monitor.AlertRule{
    ID:        "cpu-high",
    Name:      "CPU使用率过高",
    Type:      monitor.TypeCPU,
    Severity:  monitor.SeverityWarning,
    Enabled:   true,
    Threshold: 80.0,
    Operator:  ">",
    Duration:  time.Minute,
    Labels:    map[string]string{"team": "ops"},
}

monitor.SetAlertRule(cpuAlert)

// 内存使用率告警
memoryAlert := &monitor.AlertRule{
    ID:        "memory-high", 
    Name:      "内存使用率过高",
    Type:      monitor.TypeMemory,
    Severity:  monitor.SeverityCritical,
    Enabled:   true,
    Threshold: 90.0,
    Operator:  ">",
    Duration:  30 * time.Second,
}

monitor.SetAlertRule(memoryAlert)
```

### 注册告警处理器

```go
// 日志处理器
logHandler := monitor.NewLogAlertHandler(nil)
monitor.RegisterAlertHandler(logHandler)

// 邮件处理器
emailHandler := monitor.NewEmailAlertHandler(
    "smtp.example.com",
    "monitor@example.com", 
    []string{"admin@example.com"},
)
monitor.RegisterAlertHandler(emailHandler)

// Webhook处理器
webhookHandler := monitor.NewWebhookAlertHandler("http://localhost:8080/alerts")
monitor.RegisterAlertHandler(webhookHandler)
```

## 💾 数据持久化

### 启用文件存储

```go
config := &monitor.MonitorConfig{
    EnablePersist:   true,
    PersistPath:     "./monitor_data",
    PersistInterval: 5 * time.Minute,
    // 其他配置...
}

monitor := monitor.NewSystemMonitor(config)
```

### 查询历史数据

```go
// 查询最近1小时的数据
history, err := monitor.GetMetricsHistory(time.Hour)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("CPU history points: %d\n", len(history.CPU))
fmt.Printf("Memory history points: %d\n", len(history.Memory))

// 查询特定时间范围的数据
if monitor.storage != nil {
    startTime := time.Now().Add(-time.Hour)
    endTime := time.Now()
    
    query := &monitor.Query{
        Type:      "system",
        StartTime: &startTime,
        EndTime:   &endTime,
        Limit:     100,
    }
    
    results, err := monitor.storage.Load(query)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d records\n", len(results))
}
```

## 🔧 自定义扩展

### 自定义指标收集器

```go
type CustomCollector struct {
    name string
}

func (c *CustomCollector) Collect() (interface{}, error) {
    return map[string]interface{}{
        "timestamp": time.Now(),
        "value":     rand.Float64() * 100,
        "status":    "healthy",
    }, nil
}

func (c *CustomCollector) Name() string {
    return c.name
}

func (c *CustomCollector) Labels() map[string]string {
    return map[string]string{
        "type": "custom",
        "service": "my-service",
    }
}

// 注册自定义收集器
collector := &CustomCollector{name: "my-metric"}
monitor.AddMetric("my-metric", collector)
```

### 自定义告警处理器

```go
type SlackAlertHandler struct {
    webhookURL string
}

func (h *SlackAlertHandler) Handle(alert *monitor.Alert) error {
    message := fmt.Sprintf("🚨 Alert: %s\nSeverity: %s\nValue: %.2f", 
        alert.Message, alert.Severity, alert.Value)
    
    // 发送到Slack
    return sendSlackMessage(h.webhookURL, message)
}

func (h *SlackAlertHandler) Name() string {
    return "slack"
}

func (h *SlackAlertHandler) IsAsync() bool {
    return true
}

// 注册处理器
slackHandler := &SlackAlertHandler{webhookURL: "https://hooks.slack.com/..."}
monitor.RegisterAlertHandler(slackHandler)
```

## 📊 配置选项

### MonitorConfig 详细配置

```go
type MonitorConfig struct {
    // 基本配置
    Interval        time.Duration // 监控间隔，默认30秒
    EnableCPU       bool          // 启用CPU监控
    EnableMemory    bool          // 启用内存监控  
    EnableDisk      bool          // 启用磁盘监控
    EnableNetwork   bool          // 启用网络监控
    EnableProcess   bool          // 启用进程监控（性能开销较大）
    EnableLoad      bool          // 启用负载监控
    
    // 历史数据配置
    HistorySize     int           // 历史数据大小，默认1000
    HistoryInterval time.Duration // 历史数据间隔，默认1分钟
    
    // 进程监控配置
    ProcessFilters    []string    // 进程过滤器
    TopProcessCount   int         // 保存的top进程数量
    CollectProcessEnv bool        // 是否收集进程环境变量
    
    // 磁盘和网络过滤器
    DiskFilters     []string      // 磁盘过滤器
    NetworkFilters  []string      // 网络接口过滤器
    
    // 告警配置
    EnableAlert     bool          // 启用告警
    
    // 持久化配置  
    EnablePersist   bool          // 启用数据持久化
    PersistPath     string        // 持久化路径
    PersistInterval time.Duration // 持久化间隔
}
```

### 预设配置

```go
// 开发环境配置
config := monitor.DefaultMonitorConfig()

// 生产环境配置
config := &monitor.MonitorConfig{
    Interval:        10 * time.Second,
    EnableCPU:       true,
    EnableMemory:    true,
    EnableDisk:      true,
    EnableNetwork:   true,
    EnableProcess:   false, // 生产环境建议关闭
    EnableLoad:      true,
    EnableAlert:     true,
    EnablePersist:   true,
    PersistPath:     "/var/log/monitor",
    PersistInterval: time.Minute,
    HistorySize:     5000,
}

// 测试环境配置  
config := &monitor.MonitorConfig{
    Interval:      time.Second,
    EnableCPU:     true,
    EnableMemory:  true,
    EnableDisk:    false,
    EnableNetwork: false,
    EnableProcess: false,
    EnableAlert:   false,
    HistorySize:   100,
}
```

## 🧪 测试

```bash
# 运行所有测试
go test ./...

# 运行特定测试
go test -v -run TestSystemMonitor

# 运行基准测试
go test -bench=.

# 生成测试覆盖率报告
go test -cover ./...
```

## 📈 性能特征

- **内存使用**: 基础配置下约10-20MB
- **CPU开销**: 正常情况下<1%
- **监控精度**: 秒级监控，可配置
- **历史数据**: 支持数千个数据点的内存存储
- **并发安全**: 全局读写锁保护，支持并发访问

## 🚀 最佳实践

### 生产环境建议

1. **合理设置监控间隔**: 建议10-30秒，避免过于频繁的监控
2. **启用数据持久化**: 生产环境建议启用文件存储
3. **关闭进程监控**: 除非必要，建议关闭进程监控以减少开销
4. **配置告警规则**: 设置合理的告警阈值和持续时间
5. **监控自身**: 定期检查监控系统的运行状态

### 告警规则建议

```go
// CPU告警 - 保守策略
&monitor.AlertRule{
    Type:      monitor.TypeCPU,
    Threshold: 70.0,
    Operator:  ">",
    Duration:  2 * time.Minute, // 持续2分钟
    Severity:  monitor.SeverityWarning,
}

// 内存告警 - 激进策略  
&monitor.AlertRule{
    Type:      monitor.TypeMemory,
    Threshold: 85.0,
    Operator:  ">", 
    Duration:  30 * time.Second,
    Severity:  monitor.SeverityCritical,
}

// 磁盘空间告警
&monitor.AlertRule{
    Type:      monitor.TypeDisk,
    Threshold: 90.0,
    Operator:  ">",
    Duration:  time.Minute,
    Severity:  monitor.SeverityCritical,
}
```

## 🤝 贡献指南

### 开发环境设置

```bash
# 克隆仓库
git clone <repository-url>
cd basic/pkg/monitor

# 安装依赖
go mod tidy

# 运行测试
go test ./...
```

### 代码规范

- 遵循Go官方代码规范
- 使用`gofmt`格式化代码
- 通过`go vet`静态检查
- 添加必要的单元测试
- 更新相关文档

## 📄 许可证

MIT License - 详见 [LICENSE](../../LICENSE) 文件

## 🔗 相关链接

- [Logger包文档](../logger/README.md)
- [Config包文档](../config/README.md)
- [项目主页](../../README.md)

## 📞 支持与反馈

- 提交Issues报告问题
- 参与Discussions讨论
- 贡献代码和文档

---

**🔧 高性能 • 🚨 智能告警 • 💾 数据持久化 • 📊 实时监控 • 🔌 易于扩展**