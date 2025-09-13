package monitor

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// NetworkCollector 网络信息收集器
type NetworkCollector struct {
	config *MonitorConfig
	lastNetStats map[string]*NetworkStats
}

// NetworkStats 网络统计信息
type NetworkStats struct {
	Interface   string
	RxBytes     uint64
	RxPackets   uint64
	RxErrors    uint64
	RxDropped   uint64
	TxBytes     uint64
	TxPackets   uint64
	TxErrors    uint64
	TxDropped   uint64
	Timestamp   time.Time
}

// NewNetworkCollector 创建网络信息收集器
func NewNetworkCollector(config *MonitorConfig) *NetworkCollector {
	return &NetworkCollector{
		config: config,
		lastNetStats: make(map[string]*NetworkStats),
	}
}

// CollectNetworkInfo 收集网络信息
func (c *NetworkCollector) CollectNetworkInfo() ([]*NetworkInfo, error) {
	switch runtime.GOOS {
	case "linux":
		return c.collectNetworkInfoLinux()
	case "darwin":
		return c.collectNetworkInfoDarwin()
	default:
		// 其他系统使用基本的网络接口信息
		return c.collectBasicNetworkInfo()
	}
}

// collectNetworkInfoLinux 在Linux系统上收集网络信息
func (c *NetworkCollector) collectNetworkInfoLinux() ([]*NetworkInfo, error) {
	var networks []*NetworkInfo
	
	// 读取网络统计信息
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	
	// 跳过头两行
	scanner.Scan() // Inter-|   Receive                                                |  Transmit
	scanner.Scan() //  face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		
		// 解析网络接口统计信息
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}
		
		interfaceName := strings.TrimSpace(parts[0])
		stats := strings.Fields(strings.TrimSpace(parts[1]))
		
		if len(stats) < 16 {
			continue
		}
		
		// 解析统计数据
		rxBytes, _ := strconv.ParseUint(stats[0], 10, 64)
		rxPackets, _ := strconv.ParseUint(stats[1], 10, 64)
		rxErrors, _ := strconv.ParseUint(stats[2], 10, 64)
		rxDropped, _ := strconv.ParseUint(stats[3], 10, 64)
		txBytes, _ := strconv.ParseUint(stats[8], 10, 64)
		txPackets, _ := strconv.ParseUint(stats[9], 10, 64)
		txErrors, _ := strconv.ParseUint(stats[10], 10, 64)
		txDropped, _ := strconv.ParseUint(stats[11], 10, 64)
		
		currentStats := &NetworkStats{
			Interface: interfaceName,
			RxBytes:   rxBytes,
			RxPackets: rxPackets,
			RxErrors:  rxErrors,
			RxDropped: rxDropped,
			TxBytes:   txBytes,
			TxPackets: txPackets,
			TxErrors:  txErrors,
			TxDropped: txDropped,
			Timestamp: time.Now(),
		}
		
		// 获取网络接口详细信息
		netInterface, err := net.InterfaceByName(interfaceName)
		if err != nil {
			continue
		}
		
		// 获取IP地址
		addrs, err := netInterface.Addrs()
		var ipAddresses []string
		if err == nil {
			for _, addr := range addrs {
				ipAddresses = append(ipAddresses, addr.String())
			}
		}
		
		networkInfo := &NetworkInfo{
			Interface:   interfaceName,
			IPAddresses: ipAddresses,
			MACAddress:  netInterface.HardwareAddr.String(),
			MTU:         netInterface.MTU,
			IsUp:        (netInterface.Flags & net.FlagUp) != 0,
			RxBytes:     rxBytes,
			RxPackets:   rxPackets,
			RxErrors:    rxErrors,
			RxDropped:   rxDropped,
			TxBytes:     txBytes,
			TxPackets:   txPackets,
			TxErrors:    txErrors,
			TxDropped:   txDropped,
		}
		
		// 计算速率（如果有历史数据）
		if lastStats, exists := c.lastNetStats[interfaceName]; exists {
			timeDiff := currentStats.Timestamp.Sub(lastStats.Timestamp).Seconds()
			if timeDiff > 0 {
				networkInfo.RxBytesPerSec = float64(rxBytes-lastStats.RxBytes) / timeDiff
				networkInfo.TxBytesPerSec = float64(txBytes-lastStats.TxBytes) / timeDiff
			}
		}
		
		// 保存当前统计信息
		c.lastNetStats[interfaceName] = currentStats
		
		networks = append(networks, networkInfo)
	}
	
	return networks, nil
}

// collectNetworkInfoDarwin 在macOS系统上收集网络信息
func (c *NetworkCollector) collectNetworkInfoDarwin() ([]*NetworkInfo, error) {
	// macOS上的网络信息收集
	// 这里先使用基本的接口信息，后续可以扩展使用系统调用
	return c.collectBasicNetworkInfo()
}

// collectBasicNetworkInfo 收集基本网络接口信息
func (c *NetworkCollector) collectBasicNetworkInfo() ([]*NetworkInfo, error) {
	var networks []*NetworkInfo
	
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	
	for _, netInterface := range interfaces {
		// 获取IP地址
		addrs, err := netInterface.Addrs()
		var ipAddresses []string
		if err == nil {
			for _, addr := range addrs {
				ipAddresses = append(ipAddresses, addr.String())
			}
		}
		
		networkInfo := &NetworkInfo{
			Interface:   netInterface.Name,
			IPAddresses: ipAddresses,
			MACAddress:  netInterface.HardwareAddr.String(),
			MTU:         netInterface.MTU,
			IsUp:        (netInterface.Flags & net.FlagUp) != 0,
			// 统计信息在非Linux系统上无法直接获取
			RxBytes:       0,
			RxPackets:     0,
			RxErrors:      0,
			RxDropped:     0,
			TxBytes:       0,
			TxPackets:     0,
			TxErrors:      0,
			TxDropped:     0,
			RxBytesPerSec: 0,
			TxBytesPerSec: 0,
		}
		
		networks = append(networks, networkInfo)
	}
	
	return networks, nil
}

// GetNetworkConnections 获取网络连接信息
func (c *NetworkCollector) GetNetworkConnections() ([]*ConnectionInfo, error) {
	switch runtime.GOOS {
	case "linux":
		return c.getConnectionsLinux()
	default:
		// 其他系统暂不支持连接信息收集
		return nil, fmt.Errorf("connection info collection not supported on %s", runtime.GOOS)
	}
}

// ConnectionInfo 连接信息
type ConnectionInfo struct {
	Type        string // tcp, udp
	LocalAddr   string
	LocalPort   int
	RemoteAddr  string
	RemotePort  int
	State       string
	PID         int
	ProcessName string
}

// getConnectionsLinux 在Linux系统上获取网络连接信息
func (c *NetworkCollector) getConnectionsLinux() ([]*ConnectionInfo, error) {
	var connections []*ConnectionInfo
	
	// 读取TCP连接
	tcpConnections, err := c.parseProcNetFile("/proc/net/tcp", "tcp")
	if err == nil {
		connections = append(connections, tcpConnections...)
	}
	
	// 读取UDP连接
	udpConnections, err := c.parseProcNetFile("/proc/net/udp", "udp")
	if err == nil {
		connections = append(connections, udpConnections...)
	}
	
	return connections, nil
}

// parseProcNetFile 解析 /proc/net/* 文件
func (c *NetworkCollector) parseProcNetFile(filename, connType string) ([]*ConnectionInfo, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	var connections []*ConnectionInfo
	scanner := bufio.NewScanner(file)
	
	// 跳过头行
	scanner.Scan()
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		
		if len(fields) < 10 {
			continue
		}
		
		// 解析本地地址和端口
		localAddr, localPort, err := parseAddress(fields[1])
		if err != nil {
			continue
		}
		
		// 解析远程地址和端口
		remoteAddr, remotePort, err := parseAddress(fields[2])
		if err != nil {
			continue
		}
		
		// 解析状态
		state := parseConnectionState(fields[3], connType)
		
		// 解析inode
		inode, _ := strconv.Atoi(fields[9])
		
		connection := &ConnectionInfo{
			Type:       connType,
			LocalAddr:  localAddr,
			LocalPort:  localPort,
			RemoteAddr: remoteAddr,
			RemotePort: remotePort,
			State:      state,
			PID:        c.findPIDByInode(inode),
		}
		
		connections = append(connections, connection)
	}
	
	return connections, nil
}

// parseAddress 解析地址和端口
func parseAddress(addrPort string) (string, int, error) {
	parts := strings.Split(addrPort, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid address format")
	}
	
	// 解析十六进制地址
	addrHex := parts[0]
	addr := ""
	
	// IPv4地址是8位十六进制，IPv6地址是32位十六进制
	if len(addrHex) == 8 {
		// IPv4地址
		for i := 6; i >= 0; i -= 2 {
			if i+2 <= len(addrHex) {
				octet, err := strconv.ParseUint(addrHex[i:i+2], 16, 8)
				if err != nil {
					return "", 0, err
				}
				if addr != "" {
					addr = "." + addr
				}
				addr = fmt.Sprintf("%d", octet) + addr
			}
		}
	} else {
		// IPv6或其他格式，使用原始值
		addr = addrHex
	}
	
	// 解析端口
	portHex := parts[1]
	port, err := strconv.ParseUint(portHex, 16, 16)
	if err != nil {
		return "", 0, err
	}
	
	return addr, int(port), nil
}

// parseConnectionState 解析连接状态
func parseConnectionState(stateHex, connType string) string {
	if connType == "udp" {
		return "ESTABLISHED" // UDP没有连接状态
	}
	
	state, err := strconv.ParseUint(stateHex, 16, 8)
	if err != nil {
		return "UNKNOWN"
	}
	
	states := []string{
		"ESTABLISHED",
		"SYN_SENT",
		"SYN_RECV",
		"FIN_WAIT1",
		"FIN_WAIT2",
		"TIME_WAIT",
		"CLOSE",
		"CLOSE_WAIT",
		"LAST_ACK",
		"LISTEN",
		"CLOSING",
	}
	
	if int(state-1) < len(states) {
		return states[state-1]
	}
	
	return "UNKNOWN"
}

// findPIDByInode 通过inode查找PID
func (c *NetworkCollector) findPIDByInode(inode int) int {
	// 这是一个简化版本，实际实现需要遍历 /proc/*/fd/* 来匹配inode
	// 由于性能考虑，这里返回0表示未找到
	return 0
}

// GetPortListening 检查指定端口是否在监听
func (c *NetworkCollector) GetPortListening() ([]int, error) {
	var listeningPorts []int
	
	connections, err := c.GetNetworkConnections()
	if err != nil {
		return nil, err
	}
	
	for _, conn := range connections {
		if conn.State == "LISTEN" {
			listeningPorts = append(listeningPorts, conn.LocalPort)
		}
	}
	
	return listeningPorts, nil
}

// IsPortOpen 检查指定端口是否开放
func (c *NetworkCollector) IsPortOpen(port int) (bool, error) {
	listeningPorts, err := c.GetPortListening()
	if err != nil {
		return false, err
	}
	
	for _, p := range listeningPorts {
		if p == port {
			return true, nil
		}
	}
	
	return false, nil
}