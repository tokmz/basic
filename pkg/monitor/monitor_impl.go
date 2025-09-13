package monitor

import (
	"fmt"
	"time"
)

// GetSystemInfo 获取系统信息
func (m *SystemMonitor) GetSystemInfo() (*SystemInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.systemCollector == nil {
		return nil, fmt.Errorf("system collector not initialized")
	}
	
	return m.systemCollector.CollectSystemInfo()
}

// GetCPUInfo 获取CPU信息
func (m *SystemMonitor) GetCPUInfo() (*CPUInfo, error) {
	systemInfo, err := m.GetSystemInfo()
	if err != nil {
		return nil, err
	}
	
	if systemInfo.CPU == nil {
		return nil, fmt.Errorf("CPU info not available")
	}
	
	return systemInfo.CPU, nil
}

// GetMemoryInfo 获取内存信息
func (m *SystemMonitor) GetMemoryInfo() (*MemoryInfo, error) {
	systemInfo, err := m.GetSystemInfo()
	if err != nil {
		return nil, err
	}
	
	if systemInfo.Memory == nil {
		return nil, fmt.Errorf("memory info not available")
	}
	
	return systemInfo.Memory, nil
}

// GetDiskInfo 获取磁盘信息
func (m *SystemMonitor) GetDiskInfo() ([]*DiskInfo, error) {
	systemInfo, err := m.GetSystemInfo()
	if err != nil {
		return nil, err
	}
	
	return systemInfo.Disks, nil
}

// GetNetworkInfo 获取网络信息
func (m *SystemMonitor) GetNetworkInfo() ([]*NetworkInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.networkCollector == nil {
		return nil, fmt.Errorf("network collector not initialized")
	}
	
	return m.networkCollector.CollectNetworkInfo()
}

// GetProcessInfo 获取进程信息
func (m *SystemMonitor) GetProcessInfo() ([]*ProcessInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.processCollector == nil {
		return nil, fmt.Errorf("process collector not initialized")
	}
	
	return m.processCollector.CollectProcessInfo()
}

// GetLoadInfo 获取负载信息
func (m *SystemMonitor) GetLoadInfo() (*LoadInfo, error) {
	systemInfo, err := m.GetSystemInfo()
	if err != nil {
		return nil, err
	}
	
	if systemInfo.Load == nil {
		return nil, fmt.Errorf("load info not available")
	}
	
	return systemInfo.Load, nil
}

// GetMetricsHistory 获取监控历史数据
func (m *SystemMonitor) GetMetricsHistory(duration time.Duration) (*MetricsHistory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.aggregator == nil {
		return nil, fmt.Errorf("aggregator not initialized")
	}
	
	endTime := time.Now()
	startTime := endTime.Add(-duration)
	
	dataRange, err := m.aggregator.GetDataRange(startTime, endTime)
	if err != nil {
		return nil, err
	}
	
	history := &MetricsHistory{
		StartTime: startTime,
		EndTime:   endTime,
		Interval:  m.config.HistoryInterval,
		CPU:       make([]CPUInfo, 0),
		Memory:    make([]MemoryInfo, 0),
		Disk:      make(map[string][]DiskInfo),
		Network:   make(map[string][]NetworkInfo),
		Load:      make([]LoadInfo, 0),
		Custom:    make(map[string][]MetricValue),
	}
	
	// 从数据范围中提取历史数据
	for _, system := range dataRange.Systems {
		if system.CPU != nil {
			history.CPU = append(history.CPU, *system.CPU)
		}
		if system.Memory != nil {
			history.Memory = append(history.Memory, *system.Memory)
		}
		if system.Load != nil {
			history.Load = append(history.Load, *system.Load)
		}
		
		// 处理磁盘历史数据
		for _, disk := range system.Disks {
			if _, exists := history.Disk[disk.Device]; !exists {
				history.Disk[disk.Device] = make([]DiskInfo, 0)
			}
			history.Disk[disk.Device] = append(history.Disk[disk.Device], *disk)
		}
	}
	
	return history, nil
}

// monitorLoop 监控循环
func (m *SystemMonitor) monitorLoop() {
	defer m.wg.Done()
	
	ticker := time.NewTicker(m.config.Interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.collectData()
		}
	}
}

// collectData 收集数据
func (m *SystemMonitor) collectData() {
	// 收集系统信息
	if m.config.EnableCPU || m.config.EnableMemory || m.config.EnableDisk || m.config.EnableLoad {
		if systemInfo, err := m.systemCollector.CollectSystemInfo(); err == nil {
			m.aggregator.AddSystemData(systemInfo)
			
			// 存储到持久化存储
			if m.storage != nil {
				m.storage.Save(systemInfo)
			}
		}
	}
	
	// 收集网络信息
	if m.config.EnableNetwork {
		if networkInfo, err := m.networkCollector.CollectNetworkInfo(); err == nil {
			m.aggregator.AddNetworkData(networkInfo)
			
			// 存储每个网络接口信息
			if m.storage != nil {
				for _, info := range networkInfo {
					m.storage.Save(info)
				}
			}
		}
	}
	
	// 收集进程信息
	if m.config.EnableProcess {
		if processInfo, err := m.processCollector.CollectProcessInfo(); err == nil {
			m.aggregator.AddProcessData(processInfo)
			
			// 只存储前N个进程，避免存储过多数据
			topProcesses := processInfo
			if len(topProcesses) > m.config.TopProcessCount {
				topProcesses = topProcesses[:m.config.TopProcessCount]
			}
			
			if m.storage != nil {
				for _, info := range topProcesses {
					m.storage.Save(info)
				}
			}
		}
	}
}

// alertLoop 告警检查循环
func (m *SystemMonitor) alertLoop() {
	defer m.wg.Done()
	
	if m.alertManager != nil {
		m.alertManager.Start(m)
	}
	
	<-m.ctx.Done()
	
	if m.alertManager != nil {
		m.alertManager.Stop()
	}
}

// persistLoop 持久化循环
func (m *SystemMonitor) persistLoop() {
	defer m.wg.Done()
	
	ticker := time.NewTicker(m.config.PersistInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			// 定期清理和同步操作可以在这里执行
		}
	}
}

// GetSystemSummary 获取系统摘要
func (m *SystemMonitor) GetSystemSummary() *SystemSummary {
	if m.aggregator == nil {
		return nil
	}
	
	return m.aggregator.GetSystemSummary()
}

// 错误定义
var (
	ErrMonitorAlreadyRunning = fmt.Errorf("monitor is already running")
	ErrMonitorNotRunning     = fmt.Errorf("monitor is not running")
	ErrMonitorRunning        = fmt.Errorf("cannot modify config while monitor is running")
)