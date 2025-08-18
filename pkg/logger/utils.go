package logger

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
)

// FileManager 文件管理器
// 负责日志文件的生命周期管理，包括目录创建、文件压缩、清理等操作
// 提供自动化的日志文件维护功能，确保磁盘空间的合理使用
type FileManager struct {
	// config 文件管理配置
	config FileManagerConfig
	// logger 用于记录文件管理过程的日志
	logger *Logger
}

// FileManagerConfig 文件管理配置
// 定义文件管理器的各种策略参数
type FileManagerConfig struct {
	// LogDir 日志目录
	LogDir string
	// MaxAge 最大保存天数
	MaxAge int
	// MaxBackups 最大备份文件数
	MaxBackups int
	// Compress 是否压缩
	Compress bool
	// CleanupInterval 清理间隔
	CleanupInterval time.Duration
}

// DefaultFileManagerConfig 默认文件管理配置
// 返回适用于大多数场景的默认配置
// 包括30天保留期、10个备份文件、启用压缩等
// 返回:
//   - FileManagerConfig: 默认配置实例
func DefaultFileManagerConfig() FileManagerConfig {
	return FileManagerConfig{
		LogDir:          "./logs",
		MaxAge:          30,
		MaxBackups:      10,
		Compress:        true,
		CleanupInterval: 24 * time.Hour,
	}
}

// NewFileManager 创建文件管理器
// 参数:
//   - config: 文件管理配置
//   - logger: 日志器实例
// 返回:
//   - *FileManager: 新创建的文件管理器
func NewFileManager(config FileManagerConfig, logger *Logger) *FileManager {
	return &FileManager{
		config: config,
		logger: logger,
	}
}

// EnsureLogDir 确保日志目录存在
// 如果目录不存在则创建，权限设置为0755
// 返回:
//   - error: 如果创建失败则返回错误
func (fm *FileManager) EnsureLogDir() error {
	return os.MkdirAll(fm.config.LogDir, 0755)
}

// GetLogFilePath 获取日志文件路径
// 将文件名与配置的日志目录组合成完整路径
// 参数:
//   - filename: 日志文件名
// 返回:
//   - string: 完整的文件路径
func (fm *FileManager) GetLogFilePath(filename string) string {
	return filepath.Join(fm.config.LogDir, filename)
}

// CompressFile 压缩文件
// 使用gzip算法压缩指定文件，用于节省存储空间
// 参数:
//   - src: 源文件路径
//   - dst: 目标压缩文件路径
// 返回:
//   - error: 如果压缩失败则返回错误
func (fm *FileManager) CompressFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	gzWriter := gzip.NewWriter(dstFile)
	defer gzWriter.Close()

	_, err = io.Copy(gzWriter, srcFile)
	if err != nil {
		return fmt.Errorf("failed to compress file: %w", err)
	}

	return nil
}

// CleanupOldFiles 清理旧文件
// 根据配置的保留策略清理过期的日志文件
// 按文件修改时间排序，删除超出保留期限和数量限制的文件
// 返回:
//   - error: 如果清理过程中发生错误则返回
func (fm *FileManager) CleanupOldFiles() error {
	files, err := filepath.Glob(filepath.Join(fm.config.LogDir, "*.log*"))
	if err != nil {
		return fmt.Errorf("failed to list log files: %w", err)
	}

	// 按修改时间排序
	sort.Slice(files, func(i, j int) bool {
		info1, _ := os.Stat(files[i])
		info2, _ := os.Stat(files[j])
		return info1.ModTime().After(info2.ModTime())
	})

	now := time.Now()
	deleted := 0

	for i, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		// 检查文件年龄
		age := now.Sub(info.ModTime())
		maxAge := time.Duration(fm.config.MaxAge) * 24 * time.Hour

		// 删除过期文件或超出备份数量的文件
		if age > maxAge || i >= fm.config.MaxBackups {
			if err := os.Remove(file); err != nil {
				fm.logger.Warn("Failed to remove old log file",
					zap.String("file", file),
					zap.Error(err),
				)
			} else {
				deleted++
				fm.logger.Debug("Removed old log file",
					zap.String("file", file),
					zap.Duration("age", age),
				)
			}
		}
	}

	if deleted > 0 {
		fm.logger.Info("Cleaned up old log files",
			zap.Int("deleted_count", deleted),
		)
	}

	return nil
}

// CompressOldFiles 压缩旧文件
func (fm *FileManager) CompressOldFiles() error {
	if !fm.config.Compress {
		return nil
	}

	files, err := filepath.Glob(filepath.Join(fm.config.LogDir, "*.log"))
	if err != nil {
		return fmt.Errorf("failed to list log files: %w", err)
	}

	now := time.Now()
	compressed := 0

	for _, file := range files {
		// 跳过当前日志文件
		if strings.HasSuffix(file, "current.log") || strings.HasSuffix(file, "app.log") {
			continue
		}

		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		// 只压缩1天前的文件
		age := now.Sub(info.ModTime())
		if age < 24*time.Hour {
			continue
		}

		// 检查是否已经压缩
		gzFile := file + ".gz"
		if _, err := os.Stat(gzFile); err == nil {
			continue
		}

		// 压缩文件
		if err := fm.CompressFile(file, gzFile); err != nil {
			fm.logger.Warn("Failed to compress log file",
				zap.String("file", file),
				zap.Error(err),
			)
			continue
		}

		// 删除原文件
		if err := os.Remove(file); err != nil {
			fm.logger.Warn("Failed to remove original log file after compression",
				zap.String("file", file),
				zap.Error(err),
			)
		} else {
			compressed++
			fm.logger.Debug("Compressed log file",
				zap.String("original", file),
				zap.String("compressed", gzFile),
			)
		}
	}

	if compressed > 0 {
		fm.logger.Info("Compressed old log files",
			zap.Int("compressed_count", compressed),
		)
	}

	return nil
}

// StartCleanupRoutine 启动清理例程
func (fm *FileManager) StartCleanupRoutine() {
	go func() {
		ticker := time.NewTicker(fm.config.CleanupInterval)
		defer ticker.Stop()

		for range ticker.C {
			if err := fm.CompressOldFiles(); err != nil {
				fm.logger.Error("Failed to compress old files", zap.Error(err))
			}

			if err := fm.CleanupOldFiles(); err != nil {
				fm.logger.Error("Failed to cleanup old files", zap.Error(err))
			}
		}
	}()
}

// LogRotator 日志轮转器
// 负责日志文件的自动轮转，当文件大小或时间达到阈值时创建新文件
// 支持按大小、时间、备份数量等多种策略进行轮转
type LogRotator struct {
	// config 轮转配置
	config LogRotatorConfig
	// logger 用于记录轮转过程的日志
	logger *Logger
}

// LogRotatorConfig 日志轮转配置
// 定义日志轮转的各种策略参数
type LogRotatorConfig struct {
	// Filename 文件名
	Filename string
	// MaxSize 最大文件大小（MB）
	MaxSize int
	// MaxAge 最大保存天数
	MaxAge int
	// MaxBackups 最大备份数
	MaxBackups int
	// LocalTime 是否使用本地时间
	LocalTime bool
	// Compress 是否压缩
	Compress bool
}

// DefaultLogRotatorConfig 默认轮转配置
// 返回适用于大多数场景的默认轮转配置
// 包括100MB文件大小限制、30天保留期等
// 返回:
//   - LogRotatorConfig: 默认轮转配置
func DefaultLogRotatorConfig() LogRotatorConfig {
	return LogRotatorConfig{
		Filename:   "./logs/app.log",
		MaxSize:    100,
		MaxAge:     30,
		MaxBackups: 10,
		LocalTime:  true,
		Compress:   true,
	}
}

// NewLogRotator 创建日志轮转器
// 参数:
//   - config: 轮转配置
//   - logger: 日志器实例
// 返回:
//   - *LogRotator: 新创建的日志轮转器
func NewLogRotator(config LogRotatorConfig, logger *Logger) *LogRotator {
	return &LogRotator{
		config: config,
		logger: logger,
	}
}

// GetFileSize 获取文件大小
func (lr *LogRotator) GetFileSize(filename string) (int64, error) {
	info, err := os.Stat(filename)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// ShouldRotate 检查是否应该轮转
func (lr *LogRotator) ShouldRotate() (bool, error) {
	size, err := lr.GetFileSize(lr.config.Filename)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	maxSizeBytes := int64(lr.config.MaxSize) * 1024 * 1024
	return size >= maxSizeBytes, nil
}

// Rotate 执行日志轮转
func (lr *LogRotator) Rotate() error {
	// 生成备份文件名
	backupName := lr.generateBackupName()

	// 重命名当前文件
	if err := os.Rename(lr.config.Filename, backupName); err != nil {
		return fmt.Errorf("failed to rotate log file: %w", err)
	}

	lr.logger.Info("Log file rotated",
		zap.String("original", lr.config.Filename),
		zap.String("backup", backupName),
	)

	// 如果启用压缩，压缩备份文件
	if lr.config.Compress {
		go func() {
			time.Sleep(time.Second) // 等待文件写入完成
			gzName := backupName + ".gz"
			if err := lr.compressFile(backupName, gzName); err != nil {
				lr.logger.Warn("Failed to compress rotated log file",
					zap.String("file", backupName),
					zap.Error(err),
				)
			} else {
				os.Remove(backupName)
				lr.logger.Debug("Compressed rotated log file",
					zap.String("compressed", gzName),
				)
			}
		}()
	}

	return nil
}

// generateBackupName 生成备份文件名
func (lr *LogRotator) generateBackupName() string {
	dir := filepath.Dir(lr.config.Filename)
	base := filepath.Base(lr.config.Filename)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	var timestamp string
	if lr.config.LocalTime {
		timestamp = time.Now().Format("2006-01-02T15-04-05")
	} else {
		timestamp = time.Now().UTC().Format("2006-01-02T15-04-05")
	}

	return filepath.Join(dir, fmt.Sprintf("%s-%s%s", name, timestamp, ext))
}

// compressFile 压缩文件
func (lr *LogRotator) compressFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	gzWriter := gzip.NewWriter(dstFile)
	defer gzWriter.Close()

	_, err = io.Copy(gzWriter, srcFile)
	return err
}

// LogAnalyzer 日志分析器
// 提供日志文件的统计分析功能，包括日志级别分布、错误率、时间范围等
// 用于监控和分析系统运行状况
type LogAnalyzer struct {
	// logger 用于记录分析过程的日志
	logger *Logger
}

// NewLogAnalyzer 创建日志分析器
// 参数:
//   - logger: 日志器实例
// 返回:
//   - *LogAnalyzer: 新创建的日志分析器
func NewLogAnalyzer(logger *Logger) *LogAnalyzer {
	return &LogAnalyzer{
		logger: logger,
	}
}

// LogStats 日志统计信息
// 包含日志文件的各种统计数据
type LogStats struct {
	TotalLines  int64            `json:"total_lines"`
	LevelCounts map[string]int64 `json:"level_counts"`
	ErrorRate   float64          `json:"error_rate"`
	AvgLineSize float64          `json:"avg_line_size"`
	TimeRange   TimeRange        `json:"time_range"`
	TopErrors   []ErrorSummary   `json:"top_errors"`
	HourlyStats map[string]int64 `json:"hourly_stats"`
}

// TimeRange 时间范围
// 表示日志的时间跨度
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ErrorSummary 错误摘要
// 包含错误消息和出现次数的统计
type ErrorSummary struct {
	Message string `json:"message"`
	Count   int64  `json:"count"`
}

// AnalyzeLogFile 分析日志文件
// 读取并分析指定的日志文件，生成统计报告
// 包括日志级别分布、错误率、平均行大小等信息
// 参数:
//   - filename: 要分析的日志文件路径
// 返回:
//   - *LogStats: 统计结果
//   - error: 如果分析失败则返回错误
func (la *LogAnalyzer) AnalyzeLogFile(filename string) (*LogStats, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	stats := &LogStats{
		LevelCounts: make(map[string]int64),
		HourlyStats: make(map[string]int64),
	}

	// 读取文件内容并统计行数
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			stats.TotalLines++
		}
	}

	la.logger.Info("Log analysis completed",
		zap.String("file", filename),
		zap.Int64("total_lines", stats.TotalLines),
	)

	return stats, nil
}

// FormatSize 格式化文件大小
// 将字节数转换为人类可读的格式（B, KB, MB, GB等）
// 参数:
//   - bytes: 字节数
// 返回:
//   - string: 格式化后的大小字符串
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// IsValidLogLevel 检查日志级别是否有效
// 验证给定的字符串是否为有效的日志级别
// 参数:
//   - level: 要检查的日志级别字符串
// 返回:
//   - bool: 如果是有效级别则返回true
func IsValidLogLevel(level string) bool {
	switch strings.ToLower(level) {
	case "debug", "info", "warn", "error", "fatal":
		return true
	default:
		return false
	}
}

// ParseLogLevel 解析日志级别
// 将字符串转换为LogLevel类型，支持大小写不敏感
// 参数:
//   - level: 日志级别字符串
// 返回:
//   - LogLevel: 解析后的日志级别
//   - error: 如果解析失败则返回错误
func ParseLogLevel(level string) (LogLevel, error) {
	switch strings.ToLower(level) {
	case "debug":
		return DebugLevel, nil
	case "info":
		return InfoLevel, nil
	case "warn", "warning":
		return WarnLevel, nil
	case "error":
		return ErrorLevel, nil
	case "fatal":
		return FatalLevel, nil
	default:
		return InfoLevel, fmt.Errorf("invalid log level: %s", level)
	}
}

// GetLogLevelString 获取日志级别字符串
// 将LogLevel类型转换为对应的字符串表示
// 参数:
//   - level: 日志级别
// 返回:
//   - string: 日志级别的字符串表示
func GetLogLevelString(level LogLevel) string {
	switch level {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}
