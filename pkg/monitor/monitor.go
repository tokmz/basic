package monitor

import (
	"context"
	"sync"
	"time"
)

// Monitor 系统监控器接口
type Monitor interface {
	// 启动监控
	Start(ctx context.Context) error
	
	// 停止监控
	Stop() error
	
	// 获取当前系统信息
	GetSystemInfo() (*SystemInfo, error)
	
	// 获取CPU信息
	GetCPUInfo() (*CPUInfo, error)
	
	// 获取内存信息
	GetMemoryInfo() (*MemoryInfo, error)
	
	// 获取磁盘信息
	GetDiskInfo() ([]*DiskInfo, error)
	
	// 获取网络信息
	GetNetworkInfo() ([]*NetworkInfo, error)
	
	// 获取进程信息
	GetProcessInfo() ([]*ProcessInfo, error)
	
	// 获取负载信息
	GetLoadInfo() (*LoadInfo, error)
	
	// 获取监控历史数据
	GetMetricsHistory(duration time.Duration) (*MetricsHistory, error)
	
	// 添加监控指标
	AddMetric(name string, collector MetricCollector)
	
	// 设置告警规则
	SetAlertRule(rule *AlertRule)
	
	// 注册告警处理器
	RegisterAlertHandler(handler AlertHandler)
}

// SystemInfo 系统基本信息
type SystemInfo struct {
	Hostname  string       `json:"hostname"`
	OS        string       `json:"os"`
	Arch      string       `json:"arch"`
	Timestamp time.Time    `json:"timestamp"`
	CPU       *CPUInfo     `json:"cpu,omitempty"`
	Memory    *MemoryInfo  `json:"memory,omitempty"`
	Disks     []*DiskInfo  `json:"disks,omitempty"`
	Load      *LoadInfo    `json:"load,omitempty"`
}

// CPUInfo CPU信息
type CPUInfo struct {
	Cores int     `json:"cores"`
	Usage float64 `json:"usage_percent"`
}

// MemoryInfo 内存信息
type MemoryInfo struct {
	Total            uint64  `json:"total"`
	Used             uint64  `json:"used"`
	Free             uint64  `json:"free"`
	Available        uint64  `json:"available"`
	Buffers          uint64  `json:"buffers"`
	Cached           uint64  `json:"cached"`
	SwapTotal        uint64  `json:"swap_total"`
	SwapUsed         uint64  `json:"swap_used"`
	SwapFree         uint64  `json:"swap_free"`
	UsagePercent     float64 `json:"usage_percent"`
	SwapUsagePercent float64 `json:"swap_usage_percent"`
}

// DiskInfo 磁盘信息
type DiskInfo struct {
	Device       string  `json:"device"`
	MountPoint   string  `json:"mount_point"`
	Total        uint64  `json:"total"`
	Used         uint64  `json:"used"`
	Free         uint64  `json:"free"`
	UsagePercent float64 `json:"usage_percent"`
}

// NetworkInfo 网络信息
type NetworkInfo struct {
	Interface     string   `json:"interface"`
	IPAddresses   []string `json:"ip_addresses"`
	MACAddress    string   `json:"mac_address"`
	MTU           int      `json:"mtu"`
	IsUp          bool     `json:"is_up"`
	RxBytes       uint64   `json:"rx_bytes"`
	RxPackets     uint64   `json:"rx_packets"`
	RxErrors      uint64   `json:"rx_errors"`
	RxDropped     uint64   `json:"rx_dropped"`
	TxBytes       uint64   `json:"tx_bytes"`
	TxPackets     uint64   `json:"tx_packets"`
	TxErrors      uint64   `json:"tx_errors"`
	TxDropped     uint64   `json:"tx_dropped"`
	RxBytesPerSec float64  `json:"rx_bytes_per_sec"`
	TxBytesPerSec float64  `json:"tx_bytes_per_sec"`
}

// ProcessInfo 进程信息
type ProcessInfo struct {
	PID             int                       `json:"pid"`
	PPID            int                       `json:"ppid"`
	Name            string                    `json:"name"`
	State           string                    `json:"state"`
	UID             int                       `json:"uid"`
	GID             int                       `json:"gid"`
	VirtualMemory   uint64                    `json:"virtual_memory"`
	ResidentMemory  uint64                    `json:"resident_memory"`
	SharedMemory    uint64                    `json:"shared_memory"`
	UserTime        uint64                    `json:"user_time"`
	SystemTime      uint64                    `json:"system_time"`
	StartTime       time.Time                 `json:"start_time"`
	Priority        int                       `json:"priority"`
	Nice            int                       `json:"nice"`
	NumThreads      int                       `json:"num_threads"`
	Command         string                    `json:"command"`
	Environment     map[string]string         `json:"environment,omitempty"`
}

// LoadInfo 系统负载信息
type LoadInfo struct {
	Load1  float64 `json:"load_1"`
	Load5  float64 `json:"load_5"`
	Load15 float64 `json:"load_15"`
}

// MetricsHistory 监控历史数据
type MetricsHistory struct {
	StartTime time.Time                  `json:"start_time"`
	EndTime   time.Time                  `json:"end_time"`
	Interval  time.Duration              `json:"interval"`
	CPU       []CPUInfo                  `json:"cpu"`
	Memory    []MemoryInfo               `json:"memory"`
	Disk      map[string][]DiskInfo      `json:"disk"`      // device -> []DiskInfo
	Network   map[string][]NetworkInfo   `json:"network"`   // interface -> []NetworkInfo
	Load      []LoadInfo                 `json:"load"`
	Custom    map[string][]MetricValue   `json:"custom"`    // metric_name -> []MetricValue
}

// MetricValue 监控指标值
type MetricValue struct {
	Value     interface{} `json:"value"`
	Timestamp time.Time   `json:"timestamp"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// MetricCollector 指标收集器接口
type MetricCollector interface {
	Collect() (interface{}, error)
	Name() string
	Labels() map[string]string
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	// 基本配置
	Interval        time.Duration `json:"interval"`
	EnableCPU       bool          `json:"enable_cpu"`
	EnableMemory    bool          `json:"enable_memory"`
	EnableDisk      bool          `json:"enable_disk"`
	EnableNetwork   bool          `json:"enable_network"`
	EnableProcess   bool          `json:"enable_process"`
	EnableLoad      bool          `json:"enable_load"`
	
	// 历史数据配置
	HistorySize     int           `json:"history_size"`
	HistoryInterval time.Duration `json:"history_interval"`
	
	// 进程监控配置
	ProcessFilters    []string      `json:"process_filters"`
	TopProcessCount   int           `json:"top_process_count"`
	CollectProcessEnv bool          `json:"collect_process_env"`
	
	// 磁盘监控配置
	DiskFilters     []string      `json:"disk_filters"`
	
	// 网络监控配置
	NetworkFilters  []string      `json:"network_filters"`
	
	// 告警配置
	EnableAlert     bool              `json:"enable_alert"`
	
	// 持久化配置
	EnablePersist   bool          `json:"enable_persist"`
	PersistPath     string        `json:"persist_path"`
	PersistInterval time.Duration `json:"persist_interval"`
}

// DefaultMonitorConfig 返回默认监控配置
func DefaultMonitorConfig() *MonitorConfig {
	return &MonitorConfig{
		Interval:          30 * time.Second,
		EnableCPU:         true,
		EnableMemory:      true,
		EnableDisk:        true,
		EnableNetwork:     true,
		EnableProcess:     false, // 默认关闭，性能开销较大
		EnableLoad:        true,
		HistorySize:       1000,
		HistoryInterval:   time.Minute,
		TopProcessCount:   10,
		EnableAlert:       true,
		EnablePersist:     false,
		PersistInterval:   5 * time.Minute,
		CollectProcessEnv: false,
	}
}

// SystemMonitor 系统监控器实现
type SystemMonitor struct {
	config     *MonitorConfig
	collectors map[string]MetricCollector
	rules      []*AlertRule
	handlers   []AlertHandler
	history    *MetricsHistory
	alerts     map[string]*Alert
	
	// 内部状态
	running    bool
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.RWMutex
	
	// 数据收集器
	systemCollector  *SystemCollector
	networkCollector *NetworkCollector
	processCollector *ProcessCollector
	aggregator       *Aggregator
	alertManager     *AlertManager
	storage          Storage
	
	// 历史数据存储
	cpuHistory     []CPUInfo
	memoryHistory  []MemoryInfo
	diskHistory    map[string][]DiskInfo
	networkHistory map[string][]NetworkInfo
	loadHistory    []LoadInfo
	customHistory  map[string][]MetricValue
}

// NewSystemMonitor 创建系统监控器
func NewSystemMonitor(config *MonitorConfig) *SystemMonitor {
	if config == nil {
		config = DefaultMonitorConfig()
	}
	
	m := &SystemMonitor{
		config:         config,
		collectors:     make(map[string]MetricCollector),
		alerts:         make(map[string]*Alert),
		diskHistory:    make(map[string][]DiskInfo),
		networkHistory: make(map[string][]NetworkInfo),
		customHistory:  make(map[string][]MetricValue),
	}
	
	// 初始化组件
	m.systemCollector = NewSystemCollector(config)
	m.networkCollector = NewNetworkCollector(config)
	m.processCollector = NewProcessCollector(config)
	m.aggregator = NewAggregator(config.HistorySize)
	
	// 初始化告警管理器
	if config.EnableAlert {
		alertConfig := &AlertConfig{
			CheckInterval:               config.Interval,
			RepeatInterval:              5 * time.Minute,
			MaxConcurrentNotifications:  10,
			AlertHistoryTTL:             24 * time.Hour,
			Enabled:                     true,
		}
		m.alertManager = NewAlertManager(alertConfig)
	}
	
	// 初始化存储
	if config.EnablePersist {
		storageConfig := &FileStorageConfig{
			DataDir:         config.PersistPath,
			MaxFileSize:     100 * 1024 * 1024, // 100MB
			MaxFiles:        100,
			RetentionPeriod: 7 * 24 * time.Hour, // 7天
			SyncInterval:    config.PersistInterval,
		}
		if storage, err := NewFileStorage(storageConfig); err == nil {
			m.storage = storage
		} else {
			// 回退到内存存储
			m.storage = NewMemoryStorage(config.HistorySize)
		}
	} else {
		m.storage = NewMemoryStorage(config.HistorySize)
	}
	
	return m
}

// Start 启动监控
func (m *SystemMonitor) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.running {
		return ErrMonitorAlreadyRunning
	}
	
	m.ctx, m.cancel = context.WithCancel(ctx)
	m.running = true
	
	// 启动主监控循环
	m.wg.Add(1)
	go m.monitorLoop()
	
	// 启动告警检查循环
	if m.config.EnableAlert {
		m.wg.Add(1)
		go m.alertLoop()
	}
	
	// 启动持久化循环
	if m.config.EnablePersist {
		m.wg.Add(1)
		go m.persistLoop()
	}
	
	return nil
}

// Stop 停止监控
func (m *SystemMonitor) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.running {
		return ErrMonitorNotRunning
	}
	
	m.cancel()
	m.wg.Wait()
	m.running = false
	
	return nil
}

// IsRunning 检查监控是否运行中
func (m *SystemMonitor) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// GetConfig 获取配置
func (m *SystemMonitor) GetConfig() *MonitorConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// 返回配置副本
	config := *m.config
	return &config
}

// SetConfig 设置配置
func (m *SystemMonitor) SetConfig(config *MonitorConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.running {
		return ErrMonitorRunning
	}
	
	m.config = config
	return nil
}

// AddMetric 添加自定义监控指标
func (m *SystemMonitor) AddMetric(name string, collector MetricCollector) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.collectors[name] = collector
}

// RemoveMetric 移除监控指标
func (m *SystemMonitor) RemoveMetric(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.collectors, name)
}

// SetAlertRule 设置告警规则
func (m *SystemMonitor) SetAlertRule(rule *AlertRule) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// 查找是否存在同名规则
	for i, r := range m.rules {
		if r.Name == rule.Name {
			m.rules[i] = rule
			return
		}
	}
	
	// 添加新规则
	m.rules = append(m.rules, rule)
}

// RemoveAlertRule 移除告警规则
func (m *SystemMonitor) RemoveAlertRule(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for i, rule := range m.rules {
		if rule.Name == name {
			m.rules = append(m.rules[:i], m.rules[i+1:]...)
			break
		}
	}
}

// GetAlertRules 获取所有告警规则
func (m *SystemMonitor) GetAlertRules() []*AlertRule {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	rules := make([]*AlertRule, len(m.rules))
	copy(rules, m.rules)
	return rules
}

// RegisterAlertHandler 注册告警处理器
func (m *SystemMonitor) RegisterAlertHandler(handler AlertHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.handlers = append(m.handlers, handler)
}

// GetActiveAlerts 获取活跃告警
func (m *SystemMonitor) GetActiveAlerts() []*Alert {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var alerts []*Alert
	for _, alert := range m.alerts {
		if alert.Status == StatusPending || alert.Status == StatusFiring {
			alerts = append(alerts, alert)
		}
	}
	
	return alerts
}

// GetStats 获取监控统计信息
func (m *SystemMonitor) GetStats() *MonitorStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	stats := &MonitorStats{
		IsRunning:        m.running,
		CollectorCount:   len(m.collectors),
		RuleCount:        len(m.rules),
		HandlerCount:     len(m.handlers),
		ActiveAlertCount: 0,
		HistorySize:      0,
	}
	
	// 统计活跃告警
	for _, alert := range m.alerts {
		if alert.Status == StatusPending || alert.Status == StatusFiring {
			stats.ActiveAlertCount++
		}
	}
	
	// 统计历史数据大小
	stats.HistorySize += len(m.cpuHistory)
	stats.HistorySize += len(m.memoryHistory)
	stats.HistorySize += len(m.loadHistory)
	for _, diskHist := range m.diskHistory {
		stats.HistorySize += len(diskHist)
	}
	for _, netHist := range m.networkHistory {
		stats.HistorySize += len(netHist)
	}
	
	return stats
}

// MonitorStats 监控统计信息
type MonitorStats struct {
	IsRunning        bool `json:"is_running"`
	CollectorCount   int  `json:"collector_count"`
	RuleCount        int  `json:"rule_count"`
	HandlerCount     int  `json:"handler_count"`
	ActiveAlertCount int  `json:"active_alert_count"`
	HistorySize      int  `json:"history_size"`
}