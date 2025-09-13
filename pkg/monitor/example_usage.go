package monitor

import (
	"context"
	"fmt"
	"log"
	"time"
)

// ExampleBasicUsage 基本使用示例
func ExampleBasicUsage() {
	fmt.Println("=== Monitor Package Basic Usage Example ===")
	
	// 1. 创建监控器
	config := DefaultMonitorConfig()
	config.Interval = 5 * time.Second // 每5秒收集一次数据
	
	monitor := NewSystemMonitor(config)
	
	// 2. 获取当前系统信息
	fmt.Println("\n--- Current System Information ---")
	
	systemInfo, err := monitor.GetSystemInfo()
	if err != nil {
		log.Printf("Error getting system info: %v", err)
		return
	}
	
	fmt.Printf("Hostname: %s\n", systemInfo.Hostname)
	fmt.Printf("OS: %s\n", systemInfo.OS)
	fmt.Printf("Architecture: %s\n", systemInfo.Arch)
	
	// 3. 获取CPU信息
	if systemInfo.CPU != nil {
		fmt.Printf("CPU Cores: %d\n", systemInfo.CPU.Cores)
		fmt.Printf("CPU Usage: %.2f%%\n", systemInfo.CPU.Usage)
	}
	
	// 4. 获取内存信息
	if systemInfo.Memory != nil {
		fmt.Printf("Total Memory: %d bytes (%.2f GB)\n", 
			systemInfo.Memory.Total,
			float64(systemInfo.Memory.Total)/1024/1024/1024)
		fmt.Printf("Used Memory: %d bytes (%.2f GB)\n",
			systemInfo.Memory.Used,
			float64(systemInfo.Memory.Used)/1024/1024/1024)
		fmt.Printf("Memory Usage: %.2f%%\n", systemInfo.Memory.UsagePercent)
	}
	
	// 5. 获取磁盘信息
	diskInfo, err := monitor.GetDiskInfo()
	if err == nil && len(diskInfo) > 0 {
		fmt.Println("\n--- Disk Information ---")
		for _, disk := range diskInfo {
			fmt.Printf("Device: %s, Mount: %s\n", disk.Device, disk.MountPoint)
			fmt.Printf("  Total: %d bytes (%.2f GB)\n", 
				disk.Total, float64(disk.Total)/1024/1024/1024)
			fmt.Printf("  Used: %d bytes (%.2f GB)\n", 
				disk.Used, float64(disk.Used)/1024/1024/1024)
			fmt.Printf("  Usage: %.2f%%\n", disk.UsagePercent)
		}
	}
	
	// 6. 获取网络信息
	networkInfo, err := monitor.GetNetworkInfo()
	if err == nil && len(networkInfo) > 0 {
		fmt.Println("\n--- Network Information ---")
		for _, network := range networkInfo[:min(3, len(networkInfo))] { // 只显示前3个
			fmt.Printf("Interface: %s\n", network.Interface)
			fmt.Printf("  Status: Up=%t, MTU=%d\n", network.IsUp, network.MTU)
			fmt.Printf("  MAC: %s\n", network.MACAddress)
			if len(network.IPAddresses) > 0 {
				fmt.Printf("  IPs: %v\n", network.IPAddresses)
			}
			fmt.Printf("  RX: %d bytes, TX: %d bytes\n", network.RxBytes, network.TxBytes)
		}
	}
}

// ExampleWithMonitoring 带监控循环的示例
func ExampleWithMonitoring() {
	fmt.Println("\n=== Monitor Package with Monitoring Loop Example ===")
	
	// 1. 创建监控器配置
	config := &MonitorConfig{
		Interval:     2 * time.Second,
		EnableCPU:    true,
		EnableMemory: true,
		EnableDisk:   true,
		EnableNetwork: false, // 关闭网络监控以减少输出
		EnableLoad:   true,
		HistorySize:  50,
		EnableAlert:  true,
	}
	
	monitor := NewSystemMonitor(config)
	
	// 2. 设置告警规则
	cpuAlert := &AlertRule{
		ID:        "cpu-high",
		Name:      "CPU使用率过高",
		Type:      TypeCPU,
		Severity:  SeverityWarning,
		Enabled:   true,
		Threshold: 50.0, // CPU使用率超过50%时告警
		Operator:  ">",
		Duration:  10 * time.Second,
	}
	monitor.SetAlertRule(cpuAlert)
	
	memoryAlert := &AlertRule{
		ID:        "memory-high",
		Name:      "内存使用率过高",
		Type:      TypeMemory,
		Severity:  SeverityCritical,
		Enabled:   true,
		Threshold: 80.0, // 内存使用率超过80%时告警
		Operator:  ">",
		Duration:  5 * time.Second,
	}
	monitor.SetAlertRule(memoryAlert)
	
	// 3. 注册告警处理器
	logHandler := NewLogAlertHandler(nil)
	monitor.RegisterAlertHandler(logHandler)
	
	// 4. 启动监控
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	
	fmt.Println("Starting monitoring for 15 seconds...")
	
	err := monitor.Start(ctx)
	if err != nil {
		log.Printf("Failed to start monitor: %v", err)
		return
	}
	
	// 5. 定期显示监控状态
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Monitoring finished")
			monitor.Stop()
			return
		case <-ticker.C:
			// 显示当前状态
			stats := monitor.GetStats()
			fmt.Printf("\n--- Monitor Stats ---\n")
			fmt.Printf("Running: %t\n", stats.IsRunning)
			fmt.Printf("Active Alerts: %d\n", stats.ActiveAlertCount)
			fmt.Printf("History Size: %d\n", stats.HistorySize)
			
			// 显示系统摘要
			if monitor.aggregator != nil {
				summary := monitor.GetSystemSummary()
				if summary != nil && summary.CPU != nil {
					fmt.Printf("Current CPU Usage: %.2f%%\n", summary.CPU.Current)
				}
				if summary != nil && summary.Memory != nil {
					fmt.Printf("Current Memory Usage: %.2f%%\n", summary.Memory.UsagePercent)
				}
			}
		}
	}
}

// ExampleAdvancedUsage 高级使用示例
func ExampleAdvancedUsage() {
	fmt.Println("\n=== Monitor Package Advanced Usage Example ===")
	
	// 1. 创建带持久化的监控器
	config := &MonitorConfig{
		Interval:        5 * time.Second,
		EnableCPU:       true,
		EnableMemory:    true,
		EnablePersist:   true,
		PersistPath:     "./monitor_data",
		PersistInterval: 10 * time.Second,
		HistorySize:     100,
	}
	
	monitor := NewSystemMonitor(config)
	
	// 2. 添加自定义指标收集器
	customCollector := &CustomMetricCollector{
		name: "custom-metric",
		getValue: func() interface{} {
			return map[string]interface{}{
				"timestamp": time.Now(),
				"value":     42,
				"status":    "healthy",
			}
		},
	}
	monitor.AddMetric("custom", customCollector)
	
	// 3. 设置多个告警处理器
	logHandler := NewLogAlertHandler(nil)
	emailHandler := NewEmailAlertHandler("smtp.example.com", "monitor@example.com", []string{"admin@example.com"})
	webhookHandler := NewWebhookAlertHandler("http://localhost:8080/alerts")
	
	monitor.RegisterAlertHandler(logHandler)
	monitor.RegisterAlertHandler(emailHandler)
	monitor.RegisterAlertHandler(webhookHandler)
	
	// 4. 设置复杂的告警规则
	diskAlert := &AlertRule{
		ID:          "disk-space-low",
		Name:        "磁盘空间不足",
		Type:        TypeDisk,
		Severity:    SeverityCritical,
		Enabled:     true,
		Threshold:   90.0,
		Operator:    ">",
		Duration:    time.Minute,
		Labels:      map[string]string{"team": "ops", "service": "system"},
		Annotations: map[string]string{"description": "磁盘使用率过高，需要立即处理"},
	}
	monitor.SetAlertRule(diskAlert)
	
	// 5. 启动监控并演示功能
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	
	fmt.Println("Starting advanced monitoring...")
	
	err := monitor.Start(ctx)
	if err != nil {
		log.Printf("Failed to start monitor: %v", err)
		return
	}
	
	// 等待一段时间让监控收集数据
	time.Sleep(8 * time.Second)
	
	// 6. 查询历史数据
	fmt.Println("\n--- Querying Historical Data ---")
	history, err := monitor.GetMetricsHistory(10 * time.Second)
	if err == nil && history != nil {
		fmt.Printf("History period: %v to %v\n", history.StartTime.Format(time.RFC3339), history.EndTime.Format(time.RFC3339))
		fmt.Printf("CPU history points: %d\n", len(history.CPU))
		fmt.Printf("Memory history points: %d\n", len(history.Memory))
	}
	
	// 7. 获取活跃告警
	alerts := monitor.GetActiveAlerts()
	if len(alerts) > 0 {
		fmt.Printf("\n--- Active Alerts (%d) ---\n", len(alerts))
		for _, alert := range alerts {
			fmt.Printf("Alert: %s (Severity: %s, Status: %s)\n", 
				alert.Message, alert.Severity, alert.Status)
		}
	} else {
		fmt.Println("\n--- No Active Alerts ---")
	}
	
	// 8. 停止监控
	monitor.Stop()
	fmt.Println("Advanced monitoring finished")
}

// CustomMetricCollector 自定义指标收集器示例
type CustomMetricCollector struct {
	name     string
	getValue func() interface{}
}

func (c *CustomMetricCollector) Collect() (interface{}, error) {
	return c.getValue(), nil
}

func (c *CustomMetricCollector) Name() string {
	return c.name
}

func (c *CustomMetricCollector) Labels() map[string]string {
	return map[string]string{
		"type": "custom",
		"name": c.name,
	}
}

// min 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}