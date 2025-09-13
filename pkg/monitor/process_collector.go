package monitor

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ProcessCollector 进程信息收集器
type ProcessCollector struct {
	config *MonitorConfig
}

// NewProcessCollector 创建进程信息收集器
func NewProcessCollector(config *MonitorConfig) *ProcessCollector {
	return &ProcessCollector{
		config: config,
	}
}

// CollectProcessInfo 收集进程信息
func (c *ProcessCollector) CollectProcessInfo() ([]*ProcessInfo, error) {
	switch runtime.GOOS {
	case "linux":
		return c.collectProcessInfoLinux()
	case "darwin":
		return c.collectProcessInfoDarwin()
	default:
		// 其他系统使用基本的进程信息
		return c.collectBasicProcessInfo()
	}
}

// collectProcessInfoLinux 在Linux系统上收集进程信息
func (c *ProcessCollector) collectProcessInfoLinux() ([]*ProcessInfo, error) {
	var processes []*ProcessInfo
	
	// 读取 /proc 目录
	procDir := "/proc"
	entries, err := ioutil.ReadDir(procDir)
	if err != nil {
		return nil, err
	}
	
	for _, entry := range entries {
		// 只处理数字目录（PID目录）
		if !entry.IsDir() {
			continue
		}
		
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		
		// 收集进程信息
		processInfo, err := c.getProcessInfoLinux(pid)
		if err != nil {
			continue // 忽略无法读取的进程
		}
		
		processes = append(processes, processInfo)
	}
	
	return processes, nil
}

// getProcessInfoLinux 获取Linux系统上单个进程的信息
func (c *ProcessCollector) getProcessInfoLinux(pid int) (*ProcessInfo, error) {
	processInfo := &ProcessInfo{
		PID: pid,
	}
	
	// 读取进程状态信息
	err := c.readProcStat(pid, processInfo)
	if err != nil {
		return nil, err
	}
	
	// 读取进程状态详细信息
	err = c.readProcStatus(pid, processInfo)
	if err != nil {
		return nil, err
	}
	
	// 读取进程内存映射信息
	err = c.readProcStatm(pid, processInfo)
	if err != nil {
		// 内存信息读取失败不是致命错误
	}
	
	// 读取进程命令行
	err = c.readProcCmdline(pid, processInfo)
	if err != nil {
		// 命令行读取失败不是致命错误
	}
	
	// 读取进程环境变量
	if c.config != nil && c.config.CollectProcessEnv {
		err = c.readProcEnviron(pid, processInfo)
		if err != nil {
			// 环境变量读取失败不是致命错误
		}
	}
	
	return processInfo, nil
}

// readProcStat 读取 /proc/[pid]/stat 文件
func (c *ProcessCollector) readProcStat(pid int, processInfo *ProcessInfo) error {
	filename := fmt.Sprintf("/proc/%d/stat", pid)
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	
	line := string(data)
	fields := strings.Fields(line)
	
	if len(fields) < 44 {
		return fmt.Errorf("insufficient fields in /proc/%d/stat", pid)
	}
	
	// 解析字段
	processInfo.Name = strings.Trim(fields[1], "()")
	processInfo.State = fields[2]
	
	if ppid, err := strconv.Atoi(fields[3]); err == nil {
		processInfo.PPID = ppid
	}
	
	if utime, err := strconv.ParseUint(fields[13], 10, 64); err == nil {
		processInfo.UserTime = utime
	}
	
	if stime, err := strconv.ParseUint(fields[14], 10, 64); err == nil {
		processInfo.SystemTime = stime
	}
	
	if starttime, err := strconv.ParseUint(fields[21], 10, 64); err == nil {
		// 转换为实际时间（需要系统启动时间）
		bootTime := c.getBootTime()
		clockTick := c.getClockTick()
		processInfo.StartTime = time.Unix(int64(bootTime+starttime/clockTick), 0)
	}
	
	if vsize, err := strconv.ParseUint(fields[22], 10, 64); err == nil {
		processInfo.VirtualMemory = vsize
	}
	
	if rss, err := strconv.ParseInt(fields[23], 10, 64); err == nil {
		pageSize := int64(os.Getpagesize())
		processInfo.ResidentMemory = uint64(rss * pageSize)
	}
	
	if priority, err := strconv.Atoi(fields[17]); err == nil {
		processInfo.Priority = priority
	}
	
	if nice, err := strconv.Atoi(fields[18]); err == nil {
		processInfo.Nice = nice
	}
	
	if numThreads, err := strconv.Atoi(fields[19]); err == nil {
		processInfo.NumThreads = numThreads
	}
	
	return nil
}

// readProcStatus 读取 /proc/[pid]/status 文件
func (c *ProcessCollector) readProcStatus(pid int, processInfo *ProcessInfo) error {
	filename := fmt.Sprintf("/proc/%d/status", pid)
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		
		key := strings.TrimSuffix(fields[0], ":")
		value := strings.Join(fields[1:], " ")
		
		switch key {
		case "Uid":
			uids := strings.Fields(value)
			if len(uids) > 0 {
				if uid, err := strconv.Atoi(uids[0]); err == nil {
					processInfo.UID = uid
				}
			}
		case "Gid":
			gids := strings.Fields(value)
			if len(gids) > 0 {
				if gid, err := strconv.Atoi(gids[0]); err == nil {
					processInfo.GID = gid
				}
			}
		case "VmSize":
			if strings.HasSuffix(value, " kB") {
				sizeStr := strings.TrimSuffix(value, " kB")
				if size, err := strconv.ParseUint(sizeStr, 10, 64); err == nil {
					processInfo.VirtualMemory = size * 1024
				}
			}
		case "VmRSS":
			if strings.HasSuffix(value, " kB") {
				sizeStr := strings.TrimSuffix(value, " kB")
				if size, err := strconv.ParseUint(sizeStr, 10, 64); err == nil {
					processInfo.ResidentMemory = size * 1024
				}
			}
		}
	}
	
	return scanner.Err()
}

// readProcStatm 读取 /proc/[pid]/statm 文件
func (c *ProcessCollector) readProcStatm(pid int, processInfo *ProcessInfo) error {
	filename := fmt.Sprintf("/proc/%d/statm", pid)
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	
	line := strings.TrimSpace(string(data))
	fields := strings.Fields(line)
	
	if len(fields) < 7 {
		return fmt.Errorf("insufficient fields in /proc/%d/statm", pid)
	}
	
	pageSize := uint64(os.Getpagesize())
	
	// size - 程序大小（页数）
	if size, err := strconv.ParseUint(fields[0], 10, 64); err == nil {
		processInfo.VirtualMemory = size * pageSize
	}
	
	// resident - 驻留集大小（页数）
	if resident, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
		processInfo.ResidentMemory = resident * pageSize
	}
	
	// share - 共享页数
	if shared, err := strconv.ParseUint(fields[2], 10, 64); err == nil {
		processInfo.SharedMemory = shared * pageSize
	}
	
	return nil
}

// readProcCmdline 读取 /proc/[pid]/cmdline 文件
func (c *ProcessCollector) readProcCmdline(pid int, processInfo *ProcessInfo) error {
	filename := fmt.Sprintf("/proc/%d/cmdline", pid)
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	
	// cmdline 使用 null 字节分隔参数
	cmdline := string(data)
	if len(cmdline) > 0 {
		// 替换 null 字节为空格
		cmdline = strings.ReplaceAll(cmdline, "\000", " ")
		processInfo.Command = strings.TrimSpace(cmdline)
	}
	
	return nil
}

// readProcEnviron 读取 /proc/[pid]/environ 文件
func (c *ProcessCollector) readProcEnviron(pid int, processInfo *ProcessInfo) error {
	filename := fmt.Sprintf("/proc/%d/environ", pid)
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	
	// environ 使用 null 字节分隔环境变量
	environ := string(data)
	if len(environ) > 0 {
		envVars := strings.Split(environ, "\000")
		processInfo.Environment = make(map[string]string)
		
		for _, env := range envVars {
			if env == "" {
				continue
			}
			
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				processInfo.Environment[parts[0]] = parts[1]
			}
		}
	}
	
	return nil
}

// getBootTime 获取系统启动时间
func (c *ProcessCollector) getBootTime() uint64 {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "btime ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if btime, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
					return btime
				}
			}
		}
	}
	
	return 0
}

// getClockTick 获取系统时钟频率
func (c *ProcessCollector) getClockTick() uint64 {
	// 在大多数系统上，时钟频率是 100 HZ
	return 100
}

// collectProcessInfoDarwin 在macOS系统上收集进程信息
func (c *ProcessCollector) collectProcessInfoDarwin() ([]*ProcessInfo, error) {
	// macOS上的进程信息收集需要使用系统调用
	// 这里提供简化版本，实际实现可能需要CGO调用系统API
	return c.collectBasicProcessInfo()
}

// collectBasicProcessInfo 收集基本进程信息
func (c *ProcessCollector) collectBasicProcessInfo() ([]*ProcessInfo, error) {
	// 基本实现：只返回当前进程信息
	var processes []*ProcessInfo
	
	currentProcess := &ProcessInfo{
		PID:         os.Getpid(),
		PPID:        os.Getppid(),
		Name:        filepath.Base(os.Args[0]),
		State:       "R", // Running
		StartTime:   time.Now(), // 近似值
		Command:     strings.Join(os.Args, " "),
		Environment: make(map[string]string),
	}
	
	// 添加一些环境变量
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			currentProcess.Environment[parts[0]] = parts[1]
		}
	}
	
	processes = append(processes, currentProcess)
	
	return processes, nil
}

// GetProcessByPID 根据PID获取进程信息
func (c *ProcessCollector) GetProcessByPID(pid int) (*ProcessInfo, error) {
	switch runtime.GOOS {
	case "linux":
		return c.getProcessInfoLinux(pid)
	default:
		if pid == os.Getpid() {
			processes, err := c.collectBasicProcessInfo()
			if err != nil {
				return nil, err
			}
			if len(processes) > 0 {
				return processes[0], nil
			}
		}
		return nil, fmt.Errorf("process not found or not supported on this OS")
	}
}

// GetProcessByName 根据进程名获取进程信息
func (c *ProcessCollector) GetProcessByName(name string) ([]*ProcessInfo, error) {
	allProcesses, err := c.CollectProcessInfo()
	if err != nil {
		return nil, err
	}
	
	var matchingProcesses []*ProcessInfo
	for _, process := range allProcesses {
		if process.Name == name || strings.Contains(process.Command, name) {
			matchingProcesses = append(matchingProcesses, process)
		}
	}
	
	return matchingProcesses, nil
}

// KillProcess 终止进程（仅限当前用户拥有的进程）
func (c *ProcessCollector) KillProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	
	return process.Kill()
}

// GetProcessTree 获取进程树
func (c *ProcessCollector) GetProcessTree() (map[int][]*ProcessInfo, error) {
	allProcesses, err := c.CollectProcessInfo()
	if err != nil {
		return nil, err
	}
	
	processTree := make(map[int][]*ProcessInfo)
	
	for _, process := range allProcesses {
		processTree[process.PPID] = append(processTree[process.PPID], process)
	}
	
	return processTree, nil
}

// GetTopProcesses 获取资源使用最多的进程
func (c *ProcessCollector) GetTopProcesses(limit int, sortBy string) ([]*ProcessInfo, error) {
	allProcesses, err := c.CollectProcessInfo()
	if err != nil {
		return nil, err
	}
	
	// 简单排序逻辑（实际应用中可能需要更复杂的排序）
	if len(allProcesses) > limit {
		allProcesses = allProcesses[:limit]
	}
	
	return allProcesses, nil
}