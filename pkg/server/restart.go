package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// HotRestart 热重启管理器
type HotRestart struct {
	server     *Server
	listener   net.Listener
	pidFile    string
	isChild    bool
	restartCh  chan struct{}
}

// NewHotRestart 创建热重启管理器
func NewHotRestart(server *Server, pidFile string) *HotRestart {
	return &HotRestart{
		server:    server,
		pidFile:   pidFile,
		isChild:   os.Getenv("HOT_RESTART_CHILD") == "1",
		restartCh: make(chan struct{}),
	}
}

// Start 启动热重启监听
func (hr *HotRestart) Start() error {
	// 写入PID文件
	if err := hr.writePidFile(); err != nil {
		return fmt.Errorf("failed to write pid file: %w", err)
	}

	// 监听重启信号
	go hr.listenForRestartSignals()

	// 如果是子进程，需要继承父进程的监听器
	if hr.isChild {
		return hr.inheritListener()
	}

	return hr.createListener()
}

// listenForRestartSignals 监听重启信号
func (hr *HotRestart) listenForRestartSignals() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGUSR2)

	for range sigCh {
		log.Println("Received restart signal (SIGUSR2)")
		if err := hr.restart(); err != nil {
			log.Printf("Hot restart failed: %v", err)
		}
	}
}

// restart 执行热重启
func (hr *HotRestart) restart() error {
	log.Println("Starting hot restart process...")

	// 1. 启动新进程
	newPid, err := hr.startNewProcess()
	if err != nil {
		return fmt.Errorf("failed to start new process: %w", err)
	}

	log.Printf("New process started with PID: %d", newPid)

	// 2. 等待新进程准备就绪
	if err := hr.waitForNewProcess(newPid); err != nil {
		return fmt.Errorf("new process failed to start: %w", err)
	}

	// 3. 优雅关闭当前进程
	log.Println("New process ready, shutting down current process...")
	go hr.gracefulShutdownCurrent()

	return nil
}

// startNewProcess 启动新进程
func (hr *HotRestart) startNewProcess() (int, error) {
	// 获取当前执行文件路径
	execPath, err := os.Executable()
	if err != nil {
		return 0, fmt.Errorf("failed to get executable path: %w", err)
	}

	// 准备环境变量
	env := os.Environ()
	env = append(env, "HOT_RESTART_CHILD=1")

	// 如果有监听器，传递文件描述符
	var files []*os.File
	if hr.listener != nil {
		if tcpListener, ok := hr.listener.(*net.TCPListener); ok {
			file, err := tcpListener.File()
			if err != nil {
				return 0, fmt.Errorf("failed to get listener file: %w", err)
			}
			files = append(files, file)
			env = append(env, "LISTENER_FD=3") // 文件描述符从3开始
		}
	}

	// 启动新进程
	cmd := &exec.Cmd{
		Path:       execPath,
		Args:       os.Args,
		Env:        env,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		ExtraFiles: files,
	}

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start command: %w", err)
	}

	return cmd.Process.Pid, nil
}

// waitForNewProcess 等待新进程准备就绪
func (hr *HotRestart) waitForNewProcess(pid int) error {
	timeout := 30 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for new process to be ready")
		case <-ticker.C:
			// 检查新进程是否还在运行
			process, err := os.FindProcess(pid)
			if err != nil {
				return fmt.Errorf("new process not found: %w", err)
			}

			// 发送信号0检查进程是否存活
			if err := process.Signal(syscall.Signal(0)); err != nil {
				return fmt.Errorf("new process died: %w", err)
			}

			// 这里可以添加更复杂的健康检查
			// 例如：HTTP健康检查端点
			if hr.healthCheck() {
				return nil
			}
		}
	}
}

// healthCheck 健康检查
func (hr *HotRestart) healthCheck() bool {
	// 简单的健康检查，实际应用中可以更复杂
	// 例如：检查HTTP端点是否响应
	return true
}

// gracefulShutdownCurrent 优雅关闭当前进程
func (hr *HotRestart) gracefulShutdownCurrent() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := hr.server.Shutdown(ctx); err != nil {
		log.Printf("Failed to shutdown gracefully: %v", err)
		os.Exit(1)
	}

	log.Println("Current process shutdown complete")
	os.Exit(0)
}

// inheritListener 继承父进程的监听器
func (hr *HotRestart) inheritListener() error {
	listenerFD := os.Getenv("LISTENER_FD")
	if listenerFD == "" {
		return hr.createListener()
	}

	fd, err := strconv.Atoi(listenerFD)
	if err != nil {
		return fmt.Errorf("invalid listener fd: %w", err)
	}

	file := os.NewFile(uintptr(fd), "listener")
	listener, err := net.FileListener(file)
	if err != nil {
		return fmt.Errorf("failed to create listener from file: %w", err)
	}

	hr.listener = listener
	log.Println("Inherited listener from parent process")

	// 使用继承的监听器启动服务器
	return hr.serveWithListener()
}

// createListener 创建新的监听器
func (hr *HotRestart) createListener() error {
	addr := hr.server.config.GetAddr()
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	hr.listener = listener
	log.Printf("Created new listener on %s", addr)

	return hr.serveWithListener()
}

// serveWithListener 使用指定监听器启动服务
func (hr *HotRestart) serveWithListener() error {
	if hr.listener == nil {
		return fmt.Errorf("no listener available")
	}

	// 启动HTTP服务器
	err := hr.server.httpServer.Serve(hr.listener)
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

// writePidFile 写入PID文件
func (hr *HotRestart) writePidFile() error {
	pid := os.Getpid()
	pidStr := strconv.Itoa(pid)

	file, err := os.Create(hr.pidFile)
	if err != nil {
		return fmt.Errorf("failed to create pid file: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(pidStr); err != nil {
		return fmt.Errorf("failed to write pid: %w", err)
	}

	log.Printf("PID %d written to %s", pid, hr.pidFile)
	return nil
}

// removePidFile 删除PID文件
func (hr *HotRestart) removePidFile() error {
	if err := os.Remove(hr.pidFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove pid file: %w", err)
	}
	return nil
}

// GetListener 获取监听器
func (hr *HotRestart) GetListener() net.Listener {
	return hr.listener
}

// IsChild 检查是否为子进程
func (hr *HotRestart) IsChild() bool {
	return hr.isChild
}

// Cleanup 清理资源
func (hr *HotRestart) Cleanup() error {
	return hr.removePidFile()
}