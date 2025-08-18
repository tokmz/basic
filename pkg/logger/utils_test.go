package logger

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultFileManagerConfig(t *testing.T) {
	config := DefaultFileManagerConfig()
	assert.Equal(t, "./logs", config.LogDir)
	assert.Equal(t, 30, config.MaxAge)
	assert.Equal(t, 10, config.MaxBackups)
	assert.True(t, config.Compress)
	assert.Equal(t, 24*time.Hour, config.CleanupInterval)
}

func TestNewFileManager(t *testing.T) {
	config := DefaultFileManagerConfig()
	logger := GetGlobalLogger()
	fm := NewFileManager(config, logger)

	assert.NotNil(t, fm)
	assert.Equal(t, config, fm.config)
	assert.Equal(t, logger, fm.logger)
}

func TestFileManager_EnsureLogDir(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "test_logs")

	config := FileManagerConfig{
		LogDir: logDir,
	}
	fm := NewFileManager(config, GetGlobalLogger())

	err := fm.EnsureLogDir()
	assert.NoError(t, err)

	// 验证目录是否创建
	_, err = os.Stat(logDir)
	assert.NoError(t, err)
}

func TestFileManager_GetLogFilePath(t *testing.T) {
	config := FileManagerConfig{
		LogDir: "/test/logs",
	}
	fm := NewFileManager(config, GetGlobalLogger())

	path := fm.GetLogFilePath("app.log")
	expected := filepath.Join("/test/logs", "app.log")
	assert.Equal(t, expected, path)
}

func TestFileManager_CompressFile(t *testing.T) {
	tempDir := t.TempDir()
	srcFile := filepath.Join(tempDir, "test.log")
	dstFile := filepath.Join(tempDir, "test.log.gz")

	// 创建测试文件
	testContent := "test log content\nline 2\nline 3"
	err := os.WriteFile(srcFile, []byte(testContent), 0644)
	require.NoError(t, err)

	config := DefaultFileManagerConfig()
	fm := NewFileManager(config, GetGlobalLogger())

	err = fm.CompressFile(srcFile, dstFile)
	assert.NoError(t, err)

	// 验证压缩文件是否存在
	_, err = os.Stat(dstFile)
	assert.NoError(t, err)
}

func TestFileManager_CompressFile_Error(t *testing.T) {
	tempDir := t.TempDir()
	config := FileManagerConfig{
		LogDir: tempDir,
	}
	fm := NewFileManager(config, GetGlobalLogger())

	// 测试源文件不存在的情况
	err := fm.CompressFile("/nonexistent/file.log", filepath.Join(tempDir, "output.gz"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open source file")

	// 测试目标目录不存在的情况
	srcFile := filepath.Join(tempDir, "test.log")
	err = os.WriteFile(srcFile, []byte("test content"), 0644)
	require.NoError(t, err)

	err = fm.CompressFile(srcFile, "/nonexistent/dir/output.gz")
	assert.Error(t, err)
}

func TestFileManager_CleanupOldFiles(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "logs")
	err := os.MkdirAll(logDir, 0755)
	require.NoError(t, err)

	// 创建一些测试日志文件
	files := []string{"app.log", "app.log.1", "app.log.2"}
	for _, file := range files {
		filePath := filepath.Join(logDir, file)
		err = os.WriteFile(filePath, []byte("test"), 0644)
		require.NoError(t, err)
	}

	config := FileManagerConfig{
		LogDir:     logDir,
		MaxAge:     1,
		MaxBackups: 2,
	}
	fm := NewFileManager(config, GetGlobalLogger())

	err = fm.CleanupOldFiles()
	assert.NoError(t, err)
}

func TestDefaultLogRotatorConfig(t *testing.T) {
	config := DefaultLogRotatorConfig()

	// 验证默认值
	assert.Equal(t, "./logs/app.log", config.Filename)
	assert.Equal(t, 100, config.MaxSize)
	assert.Equal(t, 30, config.MaxAge)
	assert.Equal(t, 10, config.MaxBackups)
	assert.True(t, config.LocalTime)
	assert.True(t, config.Compress)
}

func TestNewLogRotator(t *testing.T) {
	config := DefaultLogRotatorConfig()
	logger := GetGlobalLogger()
	lr := NewLogRotator(config, logger)

	assert.NotNil(t, lr)
	assert.Equal(t, config, lr.config)
	assert.Equal(t, logger, lr.logger)
}

func TestLogRotator_GetFileSize(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.log")
	testContent := "test content"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	config := DefaultLogRotatorConfig()
	lr := NewLogRotator(config, GetGlobalLogger())

	size, err := lr.GetFileSize(testFile)
	assert.NoError(t, err)
	assert.Equal(t, int64(len(testContent)), size)

	// 测试文件不存在的情况
	_, err = lr.GetFileSize("/nonexistent/file.log")
	assert.Error(t, err)
}

func TestLogRotator_ShouldRotate(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	// 创建一个小文件
	err := os.WriteFile(logFile, []byte("small content"), 0644)
	assert.NoError(t, err)

	config := LogRotatorConfig{
		Filename: logFile,
		MaxSize:  1, // 1MB
	}
	rotator := NewLogRotator(config, GetGlobalLogger())

	// 小文件不应该轮转
	should, err := rotator.ShouldRotate()
	assert.NoError(t, err)
	assert.False(t, should)

	// 测试文件不存在的情况
	config.Filename = filepath.Join(tempDir, "nonexistent.log")
	rotator = NewLogRotator(config, GetGlobalLogger())
	should, err = rotator.ShouldRotate()
	assert.NoError(t, err) // 文件不存在时不应该有错误
	assert.False(t, should)
}

func TestNewLogAnalyzer(t *testing.T) {
	logger := GetGlobalLogger()
	la := NewLogAnalyzer(logger)

	assert.NotNil(t, la)
	assert.Equal(t, logger, la.logger)
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
	}

	for _, test := range tests {
		result := FormatSize(test.bytes)
		assert.Equal(t, test.expected, result)
	}
}

func TestIsValidLogLevel(t *testing.T) {
	tests := []struct {
		level    string
		expected bool
	}{
		{"debug", true},
		{"info", true},
		{"warn", true},
		{"error", true},
		{"fatal", true},
		{"DEBUG", true},
		{"INFO", true},
		{"invalid", false},
		{"", false},
	}

	for _, test := range tests {
		result := IsValidLogLevel(test.level)
		assert.Equal(t, test.expected, result, "level: %s", test.level)
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		level       string
		expected    LogLevel
		expectError bool
	}{
		{"debug", DebugLevel, false},
		{"info", InfoLevel, false},
		{"warn", WarnLevel, false},
		{"error", ErrorLevel, false},
		{"fatal", FatalLevel, false},
		{"DEBUG", DebugLevel, false},
		{"INFO", InfoLevel, false},
		{"invalid", DebugLevel, true},
		{"", DebugLevel, true},
	}

	for _, test := range tests {
		result, err := ParseLogLevel(test.level)
		if test.expectError {
			assert.Error(t, err, "level: %s", test.level)
		} else {
			assert.NoError(t, err, "level: %s", test.level)
			assert.Equal(t, test.expected, result, "level: %s", test.level)
		}
	}
}

func TestGetLogLevelString(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DebugLevel, "DEBUG"},
		{InfoLevel, "INFO"},
		{WarnLevel, "WARN"},
		{ErrorLevel, "ERROR"},
		{FatalLevel, "FATAL"},
		{LogLevel("unknown"), "UNKNOWN"},
	}

	for _, test := range tests {
		result := GetLogLevelString(test.level)
		assert.Equal(t, test.expected, result)
	}
}

func TestFileManager_CompressOldFiles(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "test_logs")
	err := os.MkdirAll(logDir, 0755)
	require.NoError(t, err)

	// 创建测试日志文件
	oldLogFile := filepath.Join(logDir, "old.log")
	currentLogFile := filepath.Join(logDir, "current.log")
	appLogFile := filepath.Join(logDir, "app.log")

	// 写入测试内容
	err = os.WriteFile(oldLogFile, []byte("test log content"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(currentLogFile, []byte("current log"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(appLogFile, []byte("app log"), 0644)
	require.NoError(t, err)

	// 修改文件时间为2天前
	oldTime := time.Now().Add(-48 * time.Hour)
	err = os.Chtimes(oldLogFile, oldTime, oldTime)
	require.NoError(t, err)

	config := FileManagerConfig{
		LogDir:   logDir,
		Compress: true,
	}
	fm := NewFileManager(config, GetGlobalLogger())

	// 测试压缩功能
	err = fm.CompressOldFiles()
	assert.NoError(t, err)

	// 验证压缩文件是否创建
	gzFile := oldLogFile + ".gz"
	_, err = os.Stat(gzFile)
	assert.NoError(t, err)

	// 验证原文件是否被删除
	_, err = os.Stat(oldLogFile)
	assert.True(t, os.IsNotExist(err))

	// 验证current.log和app.log没有被压缩
	_, err = os.Stat(currentLogFile)
	assert.NoError(t, err)
	_, err = os.Stat(appLogFile)
	assert.NoError(t, err)
}

func TestFileManager_CompressOldFiles_DisableCompress(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "test_logs")
	err := os.MkdirAll(logDir, 0755)
	require.NoError(t, err)

	// 创建一些旧日志文件
	oldFile := filepath.Join(logDir, "old.log")
	err = os.WriteFile(oldFile, []byte("old log content"), 0644)
	require.NoError(t, err)

	// 设置文件修改时间为2天前
	oldTime := time.Now().Add(-48 * time.Hour)
	err = os.Chtimes(oldFile, oldTime, oldTime)
	require.NoError(t, err)

	config := FileManagerConfig{
		LogDir:   logDir,
		Compress: false, // 禁用压缩
	}
	fm := NewFileManager(config, GetGlobalLogger())

	// 测试禁用压缩时的行为
	err = fm.CompressOldFiles()
	assert.NoError(t, err)

	// 验证文件没有被压缩
	_, err = os.Stat(oldFile)
	assert.NoError(t, err) // 原文件应该还存在

	compressedFile := oldFile + ".gz"
	_, err = os.Stat(compressedFile)
	assert.True(t, os.IsNotExist(err)) // 压缩文件不应该存在
}

func TestFileManager_StartCleanupRoutine(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "logs")
	err := os.MkdirAll(logDir, 0755)
	assert.NoError(t, err)

	// 创建一些旧文件
	oldFile := filepath.Join(logDir, "old.log")
	err = os.WriteFile(oldFile, []byte("old content"), 0644)
	assert.NoError(t, err)

	// 设置文件为旧文件
	oldTime := time.Now().Add(-48 * time.Hour)
	err = os.Chtimes(oldFile, oldTime, oldTime)
	assert.NoError(t, err)

	config := FileManagerConfig{
		LogDir:          logDir,
		MaxAge:          1, // 1天
		CleanupInterval: 100 * time.Millisecond,
	}

	logger := GetGlobalLogger()
	fm := NewFileManager(config, logger)

	// 启动清理例程
	fm.StartCleanupRoutine()

	// 等待一段时间让清理例程运行
	time.Sleep(300 * time.Millisecond)

	// 验证旧文件被清理
	_, err = os.Stat(oldFile)
	assert.True(t, os.IsNotExist(err))
}

func TestLogRotator_Rotate(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	// 创建测试日志文件
	err := os.WriteFile(logFile, []byte("test log content for rotation"), 0644)
	require.NoError(t, err)

	config := LogRotatorConfig{
		Filename:   logFile,
		MaxSize:    1, // 1MB
		MaxAge:     7,
		MaxBackups: 3,
		LocalTime:  true,
		Compress:   false,
	}
	lr := NewLogRotator(config, GetGlobalLogger())

	// 测试轮转功能
	err = lr.Rotate()
	assert.NoError(t, err)

	// 重新创建文件用于压缩测试
	err = os.WriteFile(logFile, []byte("test log content for compression"), 0644)
	assert.NoError(t, err)

	// 测试压缩选项
	config.Compress = true
	lr = NewLogRotator(config, GetGlobalLogger())
	err = lr.Rotate()
	assert.NoError(t, err)

	// 测试文件不存在的情况
	config.Filename = filepath.Join(tempDir, "nonexistent.log")
	lr = NewLogRotator(config, GetGlobalLogger())
	err = lr.Rotate()
	assert.Error(t, err)
}

func TestLogRotator_generateBackupName(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	// 测试本地时间
	config := LogRotatorConfig{
		Filename:  logFile,
		LocalTime: true,
	}
	lr := NewLogRotator(config, GetGlobalLogger())

	backupName := lr.generateBackupName()
	assert.Contains(t, backupName, "test")
	assert.Contains(t, backupName, ".log")

	// 测试UTC时间
	config.LocalTime = false
	lr = NewLogRotator(config, GetGlobalLogger())

	backupNameUTC := lr.generateBackupName()
	assert.Contains(t, backupNameUTC, "test")
	assert.Contains(t, backupNameUTC, ".log")

	// 测试不同的文件扩展名
	appFile := filepath.Join(tempDir, "app.txt")
	config.Filename = appFile
	lr = NewLogRotator(config, GetGlobalLogger())

	backupNameTxt := lr.generateBackupName()
	assert.Contains(t, backupNameTxt, "app")
	assert.Contains(t, backupNameTxt, ".txt")

	// 测试无扩展名文件
	noExtFile := filepath.Join(tempDir, "logfile")
	config.Filename = noExtFile
	lr = NewLogRotator(config, GetGlobalLogger())

	backupNameNoExt := lr.generateBackupName()
	assert.Contains(t, backupNameNoExt, "logfile")
}

func TestLogRotator_compressFile(t *testing.T) {
	tempDir := t.TempDir()
	srcFile := filepath.Join(tempDir, "source.log")
	dstFile := filepath.Join(tempDir, "dest.log.gz")

	// 创建源文件
	content := "test content for compression"
	err := os.WriteFile(srcFile, []byte(content), 0644)
	require.NoError(t, err)

	config := LogRotatorConfig{}
	lr := NewLogRotator(config, GetGlobalLogger())

	// 测试压缩功能
	err = lr.compressFile(srcFile, dstFile)
	assert.NoError(t, err)

	// 验证压缩文件是否创建
	_, err = os.Stat(dstFile)
	assert.NoError(t, err)
}

func TestLogAnalyzer_AnalyzeLogFile(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	// 创建测试日志文件
	file, err := os.Create(logFile)
	assert.NoError(t, err)

	// 写入一些测试日志
	file.WriteString("2023-01-01 10:00:00 INFO Test log 1\n")
	file.WriteString("2023-01-01 10:01:00 ERROR Test error\n")
	file.WriteString("2023-01-01 10:02:00 INFO Test log 2\n")
	file.Close()

	logger := GetGlobalLogger()
	analyzer := NewLogAnalyzer(logger)

	stats, err := analyzer.AnalyzeLogFile(logFile)
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, int64(3), stats.TotalLines)
}

func TestLogAnalyzer_AnalyzeLogFile_NotExist(t *testing.T) {
	la := NewLogAnalyzer(GetGlobalLogger())

	// 测试分析不存在的文件
	stats, err := la.AnalyzeLogFile("/nonexistent/file.log")
	assert.Error(t, err)
	assert.Nil(t, stats)
}
