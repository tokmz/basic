package monitor

import (
	"context"
	"sync"
	"time"
)

// Aggregator 监控数据聚合器
type Aggregator struct {
	mu            sync.RWMutex
	systemData    []*SystemInfo
	networkData   []*NetworkInfo
	processData   []*ProcessInfo
	maxDataPoints int
	
	// 计算缓存
	avgCache map[string]interface{}
	cacheTTL time.Duration
	lastCacheUpdate time.Time
}

// MetricSummary 指标摘要
type MetricSummary struct {
	Average float64 `json:"average"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Current float64 `json:"current"`
	Count   int     `json:"count"`
}

// SystemSummary 系统摘要
type SystemSummary struct {
	Timestamp    time.Time                   `json:"timestamp"`
	CPU          *MetricSummary              `json:"cpu"`
	Memory       *MemorySummary              `json:"memory"`
	Load         *LoadSummary                `json:"load"`
	Disks        map[string]*DiskSummary     `json:"disks"`
	Networks     map[string]*NetworkSummary  `json:"networks"`
	ProcessCount int                         `json:"process_count"`
	TopProcesses []*ProcessInfo              `json:"top_processes"`
}

// MemorySummary 内存摘要
type MemorySummary struct {
	Total           uint64  `json:"total"`
	Used            uint64  `json:"used"`
	Available       uint64  `json:"available"`
	UsagePercent    float64 `json:"usage_percent"`
	SwapTotal       uint64  `json:"swap_total"`
	SwapUsed        uint64  `json:"swap_used"`
	SwapUsagePercent float64 `json:"swap_usage_percent"`
}

// LoadSummary 负载摘要
type LoadSummary struct {
	Load1  *MetricSummary `json:"load1"`
	Load5  *MetricSummary `json:"load5"`
	Load15 *MetricSummary `json:"load15"`
}

// DiskSummary 磁盘摘要
type DiskSummary struct {
	Device       string         `json:"device"`
	MountPoint   string         `json:"mount_point"`
	Total        uint64         `json:"total"`
	Used         uint64         `json:"used"`
	Free         uint64         `json:"free"`
	UsagePercent *MetricSummary `json:"usage_percent"`
}

// NetworkSummary 网络摘要
type NetworkSummary struct {
	Interface     string         `json:"interface"`
	IsUp          bool           `json:"is_up"`
	RxBytes       *MetricSummary `json:"rx_bytes"`
	TxBytes       *MetricSummary `json:"tx_bytes"`
	RxPackets     *MetricSummary `json:"rx_packets"`
	TxPackets     *MetricSummary `json:"tx_packets"`
	RxBytesPerSec *MetricSummary `json:"rx_bytes_per_sec"`
	TxBytesPerSec *MetricSummary `json:"tx_bytes_per_sec"`
}

// NewAggregator 创建监控数据聚合器
func NewAggregator(maxDataPoints int) *Aggregator {
	if maxDataPoints <= 0 {
		maxDataPoints = 1000 // 默认保留1000个数据点
	}
	
	return &Aggregator{
		maxDataPoints: maxDataPoints,
		avgCache:      make(map[string]interface{}),
		cacheTTL:      5 * time.Minute, // 缓存5分钟
	}
}

// AddSystemData 添加系统数据
func (a *Aggregator) AddSystemData(data *SystemInfo) {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	a.systemData = append(a.systemData, data)
	
	// 限制数据点数量
	if len(a.systemData) > a.maxDataPoints {
		// 删除最旧的数据点
		copy(a.systemData, a.systemData[1:])
		a.systemData = a.systemData[:len(a.systemData)-1]
	}
	
	// 清除缓存
	a.clearCache()
}

// AddNetworkData 添加网络数据
func (a *Aggregator) AddNetworkData(data []*NetworkInfo) {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	for _, networkInfo := range data {
		a.networkData = append(a.networkData, networkInfo)
	}
	
	// 限制数据点数量
	if len(a.networkData) > a.maxDataPoints {
		excessCount := len(a.networkData) - a.maxDataPoints
		a.networkData = a.networkData[excessCount:]
	}
	
	// 清除缓存
	a.clearCache()
}

// AddProcessData 添加进程数据
func (a *Aggregator) AddProcessData(data []*ProcessInfo) {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	a.processData = append(a.processData, data...)
	
	// 限制数据点数量
	if len(a.processData) > a.maxDataPoints*10 { // 进程数据可能很多，允许更多数据点
		excessCount := len(a.processData) - a.maxDataPoints*10
		a.processData = a.processData[excessCount:]
	}
	
	// 清除缓存
	a.clearCache()
}

// GetSystemSummary 获取系统摘要
func (a *Aggregator) GetSystemSummary() *SystemSummary {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	if len(a.systemData) == 0 {
		return nil
	}
	
	// 检查缓存
	if time.Since(a.lastCacheUpdate) < a.cacheTTL {
		if cached, exists := a.avgCache["system_summary"]; exists {
			if summary, ok := cached.(*SystemSummary); ok {
				return summary
			}
		}
	}
	
	summary := &SystemSummary{
		Timestamp:    time.Now(),
		CPU:          a.calculateCPUSummary(),
		Memory:       a.calculateMemorySummary(),
		Load:         a.calculateLoadSummary(),
		Disks:        a.calculateDiskSummary(),
		Networks:     a.calculateNetworkSummary(),
		ProcessCount: len(a.processData),
		TopProcesses: a.getTopProcesses(10),
	}
	
	// 更新缓存
	a.avgCache["system_summary"] = summary
	a.lastCacheUpdate = time.Now()
	
	return summary
}

// calculateCPUSummary 计算CPU摘要
func (a *Aggregator) calculateCPUSummary() *MetricSummary {
	if len(a.systemData) == 0 {
		return nil
	}
	
	var values []float64
	for _, data := range a.systemData {
		if data.CPU != nil {
			values = append(values, data.CPU.Usage)
		}
	}
	
	return a.calculateMetricSummary(values)
}

// calculateMemorySummary 计算内存摘要
func (a *Aggregator) calculateMemorySummary() *MemorySummary {
	if len(a.systemData) == 0 {
		return nil
	}
	
	latest := a.systemData[len(a.systemData)-1]
	if latest.Memory == nil {
		return nil
	}
	
	return &MemorySummary{
		Total:            latest.Memory.Total,
		Used:             latest.Memory.Used,
		Available:        latest.Memory.Available,
		UsagePercent:     latest.Memory.UsagePercent,
		SwapTotal:        latest.Memory.SwapTotal,
		SwapUsed:         latest.Memory.SwapUsed,
		SwapUsagePercent: latest.Memory.SwapUsagePercent,
	}
}

// calculateLoadSummary 计算负载摘要
func (a *Aggregator) calculateLoadSummary() *LoadSummary {
	if len(a.systemData) == 0 {
		return nil
	}
	
	var load1Values, load5Values, load15Values []float64
	
	for _, data := range a.systemData {
		if data.Load != nil {
			load1Values = append(load1Values, data.Load.Load1)
			load5Values = append(load5Values, data.Load.Load5)
			load15Values = append(load15Values, data.Load.Load15)
		}
	}
	
	return &LoadSummary{
		Load1:  a.calculateMetricSummary(load1Values),
		Load5:  a.calculateMetricSummary(load5Values),
		Load15: a.calculateMetricSummary(load15Values),
	}
}

// calculateDiskSummary 计算磁盘摘要
func (a *Aggregator) calculateDiskSummary() map[string]*DiskSummary {
	if len(a.systemData) == 0 {
		return nil
	}
	
	diskSummaries := make(map[string]*DiskSummary)
	diskUsageHistory := make(map[string][]float64)
	
	// 收集所有磁盘的使用率历史
	for _, data := range a.systemData {
		if data.Disks != nil {
			for _, disk := range data.Disks {
				diskUsageHistory[disk.Device] = append(diskUsageHistory[disk.Device], disk.UsagePercent)
			}
		}
	}
	
	// 获取最新的磁盘信息
	latest := a.systemData[len(a.systemData)-1]
	if latest.Disks != nil {
		for _, disk := range latest.Disks {
			usageValues := diskUsageHistory[disk.Device]
			diskSummaries[disk.Device] = &DiskSummary{
				Device:       disk.Device,
				MountPoint:   disk.MountPoint,
				Total:        disk.Total,
				Used:         disk.Used,
				Free:         disk.Free,
				UsagePercent: a.calculateMetricSummary(usageValues),
			}
		}
	}
	
	return diskSummaries
}

// calculateNetworkSummary 计算网络摘要
func (a *Aggregator) calculateNetworkSummary() map[string]*NetworkSummary {
	if len(a.networkData) == 0 {
		return nil
	}
	
	networkSummaries := make(map[string]*NetworkSummary)
	networkHistory := make(map[string]map[string][]float64)
	
	// 收集网络接口的历史数据
	for _, data := range a.networkData {
		if _, exists := networkHistory[data.Interface]; !exists {
			networkHistory[data.Interface] = make(map[string][]float64)
		}
		
		history := networkHistory[data.Interface]
		history["rx_bytes"] = append(history["rx_bytes"], float64(data.RxBytes))
		history["tx_bytes"] = append(history["tx_bytes"], float64(data.TxBytes))
		history["rx_packets"] = append(history["rx_packets"], float64(data.RxPackets))
		history["tx_packets"] = append(history["tx_packets"], float64(data.TxPackets))
		history["rx_bytes_per_sec"] = append(history["rx_bytes_per_sec"], data.RxBytesPerSec)
		history["tx_bytes_per_sec"] = append(history["tx_bytes_per_sec"], data.TxBytesPerSec)
	}
	
	// 创建网络摘要
	interfaceMap := make(map[string]*NetworkInfo)
	for _, data := range a.networkData {
		interfaceMap[data.Interface] = data
	}
	
	for interfaceName, latestData := range interfaceMap {
		if history, exists := networkHistory[interfaceName]; exists {
			networkSummaries[interfaceName] = &NetworkSummary{
				Interface:     interfaceName,
				IsUp:          latestData.IsUp,
				RxBytes:       a.calculateMetricSummary(history["rx_bytes"]),
				TxBytes:       a.calculateMetricSummary(history["tx_bytes"]),
				RxPackets:     a.calculateMetricSummary(history["rx_packets"]),
				TxPackets:     a.calculateMetricSummary(history["tx_packets"]),
				RxBytesPerSec: a.calculateMetricSummary(history["rx_bytes_per_sec"]),
				TxBytesPerSec: a.calculateMetricSummary(history["tx_bytes_per_sec"]),
			}
		}
	}
	
	return networkSummaries
}

// calculateMetricSummary 计算指标摘要
func (a *Aggregator) calculateMetricSummary(values []float64) *MetricSummary {
	if len(values) == 0 {
		return &MetricSummary{}
	}
	
	var sum, min, max float64
	min = values[0]
	max = values[0]
	
	for _, value := range values {
		sum += value
		if value < min {
			min = value
		}
		if value > max {
			max = value
		}
	}
	
	return &MetricSummary{
		Average: sum / float64(len(values)),
		Min:     min,
		Max:     max,
		Current: values[len(values)-1],
		Count:   len(values),
	}
}

// getTopProcesses 获取资源使用最多的进程
func (a *Aggregator) getTopProcesses(limit int) []*ProcessInfo {
	if len(a.processData) == 0 {
		return nil
	}
	
	// 简单实现：返回最近的进程数据
	processMap := make(map[int]*ProcessInfo)
	
	// 使用最新的进程数据
	for _, process := range a.processData {
		processMap[process.PID] = process
	}
	
	var processes []*ProcessInfo
	for _, process := range processMap {
		processes = append(processes, process)
	}
	
	// 简单排序：按内存使用量排序
	if len(processes) > 1 {
		// 这里应该有更复杂的排序逻辑
		// 暂时返回前limit个进程
	}
	
	if len(processes) > limit {
		processes = processes[:limit]
	}
	
	return processes
}

// GetDataRange 获取指定时间范围的数据
func (a *Aggregator) GetDataRange(start, end time.Time) (*DataRange, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	dataRange := &DataRange{
		Start:   start,
		End:     end,
		Systems: make([]*SystemInfo, 0),
	}
	
	for _, data := range a.systemData {
		if data.Timestamp.After(start) && data.Timestamp.Before(end) {
			dataRange.Systems = append(dataRange.Systems, data)
		}
	}
	
	return dataRange, nil
}

// DataRange 数据范围
type DataRange struct {
	Start   time.Time     `json:"start"`
	End     time.Time     `json:"end"`
	Systems []*SystemInfo `json:"systems"`
}

// ClearData 清除数据
func (a *Aggregator) ClearData() {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	a.systemData = a.systemData[:0]
	a.networkData = a.networkData[:0]
	a.processData = a.processData[:0]
	a.clearCache()
}

// clearCache 清除缓存
func (a *Aggregator) clearCache() {
	a.avgCache = make(map[string]interface{})
	a.lastCacheUpdate = time.Time{}
}

// GetDataCount 获取数据点数量
func (a *Aggregator) GetDataCount() (int, int, int) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	return len(a.systemData), len(a.networkData), len(a.processData)
}

// StartDataCollection 启动数据收集
func (a *Aggregator) StartDataCollection(ctx context.Context, monitor Monitor, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 收集系统信息
			if systemInfo, err := monitor.GetSystemInfo(); err == nil {
				a.AddSystemData(systemInfo)
			}
			
			// 收集网络信息
			if networkInfo, err := monitor.GetNetworkInfo(); err == nil {
				a.AddNetworkData(networkInfo)
			}
			
			// 收集进程信息
			if processInfo, err := monitor.GetProcessInfo(); err == nil {
				a.AddProcessData(processInfo)
			}
		}
	}
}