package monitor

import (
	"context"
	"testing"
	"time"
)

func TestNewSystemMonitor(t *testing.T) {
	config := DefaultMonitorConfig()
	monitor := NewSystemMonitor(config)
	
	if monitor == nil {
		t.Fatal("NewSystemMonitor returned nil")
	}
	
	if monitor.config != config {
		t.Error("Monitor config not set correctly")
	}
	
	if monitor.systemCollector == nil {
		t.Error("System collector not initialized")
	}
	
	if monitor.networkCollector == nil {
		t.Error("Network collector not initialized")
	}
	
	if monitor.processCollector == nil {
		t.Error("Process collector not initialized")
	}
	
	if monitor.aggregator == nil {
		t.Error("Aggregator not initialized")
	}
}

func TestMonitorLifecycle(t *testing.T) {
	monitor := NewSystemMonitor(nil)
	
	// 测试初始状态
	if monitor.IsRunning() {
		t.Error("Monitor should not be running initially")
	}
	
	// 启动监控
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	err := monitor.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start monitor: %v", err)
	}
	
	if !monitor.IsRunning() {
		t.Error("Monitor should be running after Start()")
	}
	
	// 重复启动应该返回错误
	err = monitor.Start(ctx)
	if err != ErrMonitorAlreadyRunning {
		t.Errorf("Expected ErrMonitorAlreadyRunning, got %v", err)
	}
	
	// 停止监控
	err = monitor.Stop()
	if err != nil {
		t.Fatalf("Failed to stop monitor: %v", err)
	}
	
	if monitor.IsRunning() {
		t.Error("Monitor should not be running after Stop()")
	}
	
	// 重复停止应该返回错误
	err = monitor.Stop()
	if err != ErrMonitorNotRunning {
		t.Errorf("Expected ErrMonitorNotRunning, got %v", err)
	}
}

func TestMonitorDataCollection(t *testing.T) {
	monitor := NewSystemMonitor(nil)
	
	// 测试系统信息收集
	systemInfo, err := monitor.GetSystemInfo()
	if err != nil {
		t.Errorf("Failed to get system info: %v", err)
	} else {
		if systemInfo.Hostname == "" {
			t.Error("Hostname should not be empty")
		}
		if systemInfo.OS == "" {
			t.Error("OS should not be empty")
		}
	}
	
	// 测试CPU信息收集
	cpuInfo, err := monitor.GetCPUInfo()
	if err != nil {
		t.Logf("CPU info not available: %v", err)
	} else {
		if cpuInfo.Cores <= 0 {
			t.Error("CPU cores should be greater than 0")
		}
	}
	
	// 测试内存信息收集
	memoryInfo, err := monitor.GetMemoryInfo()
	if err != nil {
		t.Logf("Memory info not available: %v", err)
	} else {
		if memoryInfo.Total == 0 {
			t.Error("Total memory should be greater than 0")
		}
	}
	
	// 测试磁盘信息收集
	diskInfo, err := monitor.GetDiskInfo()
	if err != nil {
		t.Logf("Disk info not available: %v", err)
	} else {
		if len(diskInfo) == 0 {
			t.Log("No disk info available")
		}
	}
	
	// 测试网络信息收集
	networkInfo, err := monitor.GetNetworkInfo()
	if err != nil {
		t.Logf("Network info not available: %v", err)
	} else {
		if len(networkInfo) == 0 {
			t.Log("No network interfaces found")
		}
	}
}

func TestMonitorConfig(t *testing.T) {
	// 测试默认配置
	defaultConfig := DefaultMonitorConfig()
	if defaultConfig.Interval != 30*time.Second {
		t.Errorf("Expected default interval 30s, got %v", defaultConfig.Interval)
	}
	
	if !defaultConfig.EnableCPU {
		t.Error("CPU monitoring should be enabled by default")
	}
	
	if !defaultConfig.EnableMemory {
		t.Error("Memory monitoring should be enabled by default")
	}
	
	// 测试自定义配置
	customConfig := &MonitorConfig{
		Interval:     10 * time.Second,
		EnableCPU:    false,
		EnableMemory: true,
		HistorySize:  500,
	}
	
	monitor := NewSystemMonitor(customConfig)
	retrievedConfig := monitor.GetConfig()
	
	if retrievedConfig.Interval != customConfig.Interval {
		t.Errorf("Expected interval %v, got %v", customConfig.Interval, retrievedConfig.Interval)
	}
	
	if retrievedConfig.EnableCPU != customConfig.EnableCPU {
		t.Errorf("Expected EnableCPU %v, got %v", customConfig.EnableCPU, retrievedConfig.EnableCPU)
	}
}

func TestMonitorStats(t *testing.T) {
	monitor := NewSystemMonitor(nil)
	
	stats := monitor.GetStats()
	if stats == nil {
		t.Fatal("GetStats returned nil")
	}
	
	if stats.IsRunning {
		t.Error("Monitor should not be running initially")
	}
	
	if stats.CollectorCount < 0 {
		t.Error("Collector count should not be negative")
	}
	
	if stats.HistorySize < 0 {
		t.Error("History size should not be negative")
	}
}

func TestMonitorAlertRules(t *testing.T) {
	monitor := NewSystemMonitor(nil)
	
	// 添加告警规则
	rule := &AlertRule{
		ID:          "test-rule",
		Name:        "Test CPU Alert",
		Type:        TypeCPU,
		Severity:    SeverityWarning,
		Enabled:     true,
		Threshold:   80.0,
		Operator:    ">",
		Duration:    time.Minute,
	}
	
	monitor.SetAlertRule(rule)
	
	rules := monitor.GetAlertRules()
	if len(rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(rules))
	}
	
	if rules[0].Name != rule.Name {
		t.Errorf("Expected rule name %s, got %s", rule.Name, rules[0].Name)
	}
	
	// 移除告警规则
	monitor.RemoveAlertRule(rule.Name)
	rules = monitor.GetAlertRules()
	if len(rules) != 0 {
		t.Errorf("Expected 0 rules after removal, got %d", len(rules))
	}
}

func TestSystemCollector(t *testing.T) {
	collector := NewSystemCollector(nil)
	if collector == nil {
		t.Fatal("NewSystemCollector returned nil")
	}
	
	systemInfo, err := collector.CollectSystemInfo()
	if err != nil {
		t.Fatalf("Failed to collect system info: %v", err)
	}
	
	if systemInfo.Hostname == "" {
		t.Error("Hostname should not be empty")
	}
	
	if systemInfo.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

func TestNetworkCollector(t *testing.T) {
	collector := NewNetworkCollector(nil)
	if collector == nil {
		t.Fatal("NewNetworkCollector returned nil")
	}
	
	networkInfo, err := collector.CollectNetworkInfo()
	if err != nil {
		t.Fatalf("Failed to collect network info: %v", err)
	}
	
	// 至少应该有回环接口
	if len(networkInfo) == 0 {
		t.Log("No network interfaces found (this might be normal in some environments)")
	}
}

func TestProcessCollector(t *testing.T) {
	collector := NewProcessCollector(nil)
	if collector == nil {
		t.Fatal("NewProcessCollector returned nil")
	}
	
	processInfo, err := collector.CollectProcessInfo()
	if err != nil {
		t.Fatalf("Failed to collect process info: %v", err)
	}
	
	// 至少应该有当前进程
	if len(processInfo) == 0 {
		t.Error("No processes found")
	}
}

func TestAggregator(t *testing.T) {
	aggregator := NewAggregator(100)
	if aggregator == nil {
		t.Fatal("NewAggregator returned nil")
	}
	
	// 创建测试系统信息
	systemInfo := &SystemInfo{
		Hostname:  "test-host",
		OS:        "test-os",
		Timestamp: time.Now(),
		CPU: &CPUInfo{
			Cores: 4,
			Usage: 25.5,
		},
		Memory: &MemoryInfo{
			Total:        8000000000,
			Used:         2000000000,
			Available:    6000000000,
			UsagePercent: 25.0,
		},
	}
	
	// 添加系统数据
	aggregator.AddSystemData(systemInfo)
	
	// 获取摘要
	summary := aggregator.GetSystemSummary()
	if summary == nil {
		t.Fatal("GetSystemSummary returned nil")
	}
	
	if summary.CPU == nil {
		t.Error("CPU summary should not be nil")
	}
	
	if summary.Memory == nil {
		t.Error("Memory summary should not be nil")
	}
}

func TestStorage(t *testing.T) {
	// 测试内存存储
	storage := NewMemoryStorage(100)
	if storage == nil {
		t.Fatal("NewMemoryStorage returned nil")
	}
	
	// 测试数据存储和查询
	testData := &SystemInfo{
		Hostname:  "test-host",
		OS:        "test-os",
		Timestamp: time.Now(),
	}
	
	err := storage.Save(testData)
	if err != nil {
		t.Errorf("Failed to save data: %v", err)
	}
	
	// 查询数据
	query := &Query{
		Type:  "system",
		Limit: 10,
	}
	
	results, err := storage.Load(query)
	if err != nil {
		t.Errorf("Failed to load data: %v", err)
	}
	
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}