package monitor

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// SystemCollector 系统信息收集器
type SystemCollector struct {
	config *MonitorConfig
	lastCPUStats *CPUStats
}

// CPUStats CPU统计信息
type CPUStats struct {
	User   uint64
	Nice   uint64
	System uint64
	Idle   uint64
	IOWait uint64
	IRQ    uint64
	SoftIRQ uint64
	Steal  uint64
	Guest  uint64
	GuestNice uint64
	Total  uint64
}

// NewSystemCollector 创建系统信息收集器
func NewSystemCollector(config *MonitorConfig) *SystemCollector {
	if config == nil {
		config = DefaultMonitorConfig()
	}
	
	return &SystemCollector{
		config: config,
	}
}

// CollectSystemInfo 收集系统信息
func (c *SystemCollector) CollectSystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{
		Timestamp: time.Now(),
		Hostname:  getHostname(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
	
	var err error
	
	// 收集CPU信息
	if info.CPU, err = c.collectCPUInfo(); err != nil {
		return nil, fmt.Errorf("failed to collect CPU info: %w", err)
	}
	
	// 收集内存信息
	if info.Memory, err = c.collectMemoryInfo(); err != nil {
		return nil, fmt.Errorf("failed to collect memory info: %w", err)
	}
	
	// 收集磁盘信息
	if info.Disks, err = c.collectDiskInfo(); err != nil {
		return nil, fmt.Errorf("failed to collect disk info: %w", err)
	}
	
	// 收集负载信息
	if info.Load, err = c.collectLoadInfo(); err != nil {
		return nil, fmt.Errorf("failed to collect load info: %w", err)
	}
	
	return info, nil
}

// collectCPUInfo 收集CPU信息
func (c *SystemCollector) collectCPUInfo() (*CPUInfo, error) {
	cpuInfo := &CPUInfo{
		Cores: runtime.NumCPU(),
	}
	
	// 根据操作系统收集CPU使用率
	switch runtime.GOOS {
	case "linux":
		usage, err := c.collectCPUUsageLinux()
		if err != nil {
			return nil, err
		}
		cpuInfo.Usage = usage
	case "darwin":
		usage, err := c.collectCPUUsageDarwin()
		if err != nil {
			return nil, err
		}
		cpuInfo.Usage = usage
	default:
		// 其他系统使用基本的统计信息
		cpuInfo.Usage = 0.0
	}
	
	return cpuInfo, nil
}

// collectCPUUsageLinux 在Linux系统上收集CPU使用率
func (c *SystemCollector) collectCPUUsageLinux() (float64, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return 0, fmt.Errorf("failed to read /proc/stat")
	}
	
	line := scanner.Text()
	if !strings.HasPrefix(line, "cpu ") {
		return 0, fmt.Errorf("invalid /proc/stat format")
	}
	
	fields := strings.Fields(line)
	if len(fields) < 8 {
		return 0, fmt.Errorf("insufficient CPU fields in /proc/stat")
	}
	
	stats := &CPUStats{}
	stats.User, _ = strconv.ParseUint(fields[1], 10, 64)
	stats.Nice, _ = strconv.ParseUint(fields[2], 10, 64)
	stats.System, _ = strconv.ParseUint(fields[3], 10, 64)
	stats.Idle, _ = strconv.ParseUint(fields[4], 10, 64)
	stats.IOWait, _ = strconv.ParseUint(fields[5], 10, 64)
	stats.IRQ, _ = strconv.ParseUint(fields[6], 10, 64)
	stats.SoftIRQ, _ = strconv.ParseUint(fields[7], 10, 64)
	if len(fields) > 8 {
		stats.Steal, _ = strconv.ParseUint(fields[8], 10, 64)
	}
	if len(fields) > 9 {
		stats.Guest, _ = strconv.ParseUint(fields[9], 10, 64)
	}
	if len(fields) > 10 {
		stats.GuestNice, _ = strconv.ParseUint(fields[10], 10, 64)
	}
	
	stats.Total = stats.User + stats.Nice + stats.System + stats.Idle + 
		stats.IOWait + stats.IRQ + stats.SoftIRQ + stats.Steal + 
		stats.Guest + stats.GuestNice
	
	// 如果有上一次的统计，计算使用率
	if c.lastCPUStats != nil {
		totalDiff := stats.Total - c.lastCPUStats.Total
		idleDiff := stats.Idle - c.lastCPUStats.Idle
		
		if totalDiff > 0 {
			usage := 100.0 * (float64(totalDiff) - float64(idleDiff)) / float64(totalDiff)
			c.lastCPUStats = stats
			return usage, nil
		}
	}
	
	c.lastCPUStats = stats
	return 0.0, nil
}

// collectCPUUsageDarwin 在macOS系统上收集CPU使用率
func (c *SystemCollector) collectCPUUsageDarwin() (float64, error) {
	// macOS上的CPU使用率收集比较复杂，这里提供简化版本
	// 实际生产环境可能需要使用CGO调用系统API
	return 0.0, nil
}

// collectMemoryInfo 收集内存信息
func (c *SystemCollector) collectMemoryInfo() (*MemoryInfo, error) {
	memInfo := &MemoryInfo{}
	
	switch runtime.GOOS {
	case "linux":
		return c.collectMemoryInfoLinux()
	case "darwin":
		return c.collectMemoryInfoDarwin()
	default:
		// 使用Go runtime的基本信息
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		
		memInfo.Total = m.Sys
		memInfo.Used = m.Alloc
		memInfo.Available = m.Sys - m.Alloc
		memInfo.UsagePercent = float64(m.Alloc) / float64(m.Sys) * 100
		
		return memInfo, nil
	}
}

// collectMemoryInfoLinux 在Linux系统上收集内存信息
func (c *SystemCollector) collectMemoryInfoLinux() (*MemoryInfo, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	memInfo := &MemoryInfo{}
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		
		// 转换为字节 (kB -> bytes)
		value *= 1024
		
		switch fields[0] {
		case "MemTotal:":
			memInfo.Total = value
		case "MemFree:":
			memInfo.Free = value
		case "MemAvailable:":
			memInfo.Available = value
		case "Buffers:":
			memInfo.Buffers = value
		case "Cached:":
			memInfo.Cached = value
		case "SwapTotal:":
			memInfo.SwapTotal = value
		case "SwapFree:":
			memInfo.SwapFree = value
		}
	}
	
	// 计算使用量和使用率
	memInfo.Used = memInfo.Total - memInfo.Free - memInfo.Buffers - memInfo.Cached
	memInfo.SwapUsed = memInfo.SwapTotal - memInfo.SwapFree
	
	if memInfo.Total > 0 {
		memInfo.UsagePercent = float64(memInfo.Used) / float64(memInfo.Total) * 100
	}
	
	if memInfo.SwapTotal > 0 {
		memInfo.SwapUsagePercent = float64(memInfo.SwapUsed) / float64(memInfo.SwapTotal) * 100
	}
	
	return memInfo, nil
}

// collectMemoryInfoDarwin 在macOS系统上收集内存信息
func (c *SystemCollector) collectMemoryInfoDarwin() (*MemoryInfo, error) {
	// macOS上的内存信息收集需要使用系统调用
	// 这里提供简化版本，实际实现可能需要CGO
	memInfo := &MemoryInfo{}
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	memInfo.Total = m.Sys
	memInfo.Used = m.Alloc
	memInfo.Available = m.Sys - m.Alloc
	memInfo.UsagePercent = float64(m.Alloc) / float64(m.Sys) * 100
	
	return memInfo, nil
}

// collectDiskInfo 收集磁盘信息
func (c *SystemCollector) collectDiskInfo() ([]*DiskInfo, error) {
	var disks []*DiskInfo
	
	switch runtime.GOOS {
	case "linux":
		return c.collectDiskInfoLinux()
	case "darwin":
		return c.collectDiskInfoDarwin()
	default:
		// 默认收集根目录信息
		diskInfo, err := c.getDiskUsage("/")
		if err != nil {
			return nil, err
		}
		disks = append(disks, diskInfo)
		return disks, nil
	}
}

// collectDiskInfoLinux 在Linux系统上收集磁盘信息
func (c *SystemCollector) collectDiskInfoLinux() ([]*DiskInfo, error) {
	var disks []*DiskInfo
	
	// 读取挂载信息
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	mountPoints := make(map[string]bool)
	
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			device := fields[0]
			mountPoint := fields[1]
			
			// 只收集物理磁盘和主要文件系统
			if strings.HasPrefix(device, "/dev/") && 
			   !strings.Contains(mountPoint, "/proc") &&
			   !strings.Contains(mountPoint, "/sys") &&
			   !strings.Contains(mountPoint, "/dev") {
				mountPoints[mountPoint] = true
			}
		}
	}
	
	// 为每个挂载点收集磁盘使用信息
	for mountPoint := range mountPoints {
		diskInfo, err := c.getDiskUsage(mountPoint)
		if err != nil {
			continue // 忽略错误的挂载点
		}
		disks = append(disks, diskInfo)
	}
	
	// 如果没有找到任何磁盘，至少返回根目录
	if len(disks) == 0 {
		diskInfo, err := c.getDiskUsage("/")
		if err != nil {
			return nil, err
		}
		disks = append(disks, diskInfo)
	}
	
	return disks, nil
}

// collectDiskInfoDarwin 在macOS系统上收集磁盘信息
func (c *SystemCollector) collectDiskInfoDarwin() ([]*DiskInfo, error) {
	var disks []*DiskInfo
	
	// macOS通常只需要检查根目录
	diskInfo, err := c.getDiskUsage("/")
	if err != nil {
		return nil, err
	}
	disks = append(disks, diskInfo)
	
	return disks, nil
}

// getDiskUsage 获取指定路径的磁盘使用信息
func (c *SystemCollector) getDiskUsage(path string) (*DiskInfo, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return nil, err
	}
	
	diskInfo := &DiskInfo{
		Device:     path,
		MountPoint: path,
		Total:      stat.Blocks * uint64(stat.Bsize),
		Free:       stat.Bavail * uint64(stat.Bsize),
	}
	
	diskInfo.Used = diskInfo.Total - diskInfo.Free
	if diskInfo.Total > 0 {
		diskInfo.UsagePercent = float64(diskInfo.Used) / float64(diskInfo.Total) * 100
	}
	
	return diskInfo, nil
}

// collectLoadInfo 收集系统负载信息
func (c *SystemCollector) collectLoadInfo() (*LoadInfo, error) {
	switch runtime.GOOS {
	case "linux":
		return c.collectLoadInfoLinux()
	case "darwin":
		return c.collectLoadInfoDarwin()
	default:
		// 其他系统返回默认值
		return &LoadInfo{
			Load1:  0.0,
			Load5:  0.0,
			Load15: 0.0,
		}, nil
	}
}

// collectLoadInfoLinux 在Linux系统上收集负载信息
func (c *SystemCollector) collectLoadInfoLinux() (*LoadInfo, error) {
	file, err := os.Open("/proc/loadavg")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return nil, fmt.Errorf("failed to read /proc/loadavg")
	}
	
	line := scanner.Text()
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return nil, fmt.Errorf("invalid /proc/loadavg format")
	}
	
	load1, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return nil, err
	}
	
	load5, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return nil, err
	}
	
	load15, err := strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return nil, err
	}
	
	return &LoadInfo{
		Load1:  load1,
		Load5:  load5,
		Load15: load15,
	}, nil
}

// collectLoadInfoDarwin 在macOS系统上收集负载信息
func (c *SystemCollector) collectLoadInfoDarwin() (*LoadInfo, error) {
	// macOS上的负载信息收集需要使用系统调用
	// 这里提供简化版本
	return &LoadInfo{
		Load1:  0.0,
		Load5:  0.0,
		Load15: 0.0,
	}, nil
}

// getHostname 获取主机名
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}