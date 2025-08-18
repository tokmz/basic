package server

import "errors"

// 服务器相关错误定义
var (
	// 配置相关错误
	ErrInvalidPort      = errors.New("invalid port number")
	ErrInvalidTimeout   = errors.New("invalid timeout value")
	ErrInvalidRateLimit = errors.New("invalid rate limit value")

	// 服务器相关错误
	ErrServerNotStarted = errors.New("server not started")
	ErrServerAlreadyRunning = errors.New("server already running")
	ErrShutdownTimeout = errors.New("shutdown timeout")

	// 热重启相关错误
	ErrRestartFailed = errors.New("restart failed")
	ErrPidFileNotFound = errors.New("pid file not found")
	ErrInvalidPid = errors.New("invalid pid")

	// 静态文件相关错误
	ErrStaticDirNotFound = errors.New("static directory not found")
	ErrFileNotFound = errors.New("file not found")
	ErrDirectoryTraversal = errors.New("directory traversal attack detected")
)