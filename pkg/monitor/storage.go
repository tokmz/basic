package monitor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Storage 监控数据存储接口
type Storage interface {
	// Save 保存数据
	Save(data interface{}) error
	// Load 加载数据
	Load(query *Query) ([]interface{}, error)
	// Delete 删除数据
	Delete(query *Query) error
	// Close 关闭存储
	Close() error
}

// Query 查询条件
type Query struct {
	Type      string            `json:"type"`       // 数据类型: system, network, process, alert
	StartTime *time.Time        `json:"start_time"` // 开始时间
	EndTime   *time.Time        `json:"end_time"`   // 结束时间
	Labels    map[string]string `json:"labels"`     // 标签过滤
	Limit     int               `json:"limit"`      // 限制数量
	Offset    int               `json:"offset"`     // 偏移量
	OrderBy   string            `json:"order_by"`   // 排序字段
	OrderDesc bool              `json:"order_desc"` // 降序排序
}

// StorageRecord 存储记录
type StorageRecord struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Timestamp time.Time         `json:"timestamp"`
	Data      interface{}       `json:"data"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// FileStorage 文件存储实现
type FileStorage struct {
	mu      sync.RWMutex
	dataDir string
	config  *FileStorageConfig
}

// FileStorageConfig 文件存储配置
type FileStorageConfig struct {
	DataDir          string        `json:"data_dir"`
	MaxFileSize      int64         `json:"max_file_size"`      // 最大文件大小（字节）
	MaxFiles         int           `json:"max_files"`          // 最大文件数量
	CompressionLevel int           `json:"compression_level"`  // 压缩级别
	RetentionPeriod  time.Duration `json:"retention_period"`   // 数据保留期
	SyncInterval     time.Duration `json:"sync_interval"`      // 同步间隔
}

// NewFileStorage 创建文件存储
func NewFileStorage(config *FileStorageConfig) (*FileStorage, error) {
	if config == nil {
		config = &FileStorageConfig{
			DataDir:          "./monitor_data",
			MaxFileSize:      100 * 1024 * 1024, // 100MB
			MaxFiles:         100,
			CompressionLevel: 6,
			RetentionPeriod:  7 * 24 * time.Hour, // 7天
			SyncInterval:     5 * time.Minute,
		}
	}
	
	// 确保数据目录存在
	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}
	
	storage := &FileStorage{
		dataDir: config.DataDir,
		config:  config,
	}
	
	// 启动清理任务
	go storage.startCleanupTask()
	
	return storage, nil
}

// Save 保存数据到文件
func (fs *FileStorage) Save(data interface{}) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	record := &StorageRecord{
		ID:        generateID(),
		Timestamp: time.Now(),
		Data:      data,
		Labels:    make(map[string]string),
	}
	
	// 根据数据类型设置记录类型和标签
	switch v := data.(type) {
	case *SystemInfo:
		record.Type = "system"
		record.Labels["hostname"] = v.Hostname
		record.Labels["os"] = v.OS
	case *NetworkInfo:
		record.Type = "network"
		record.Labels["interface"] = v.Interface
	case *ProcessInfo:
		record.Type = "process"
		record.Labels["pid"] = fmt.Sprintf("%d", v.PID)
		record.Labels["name"] = v.Name
	case *Alert:
		record.Type = "alert"
		record.Labels["rule_id"] = v.RuleID
		record.Labels["severity"] = string(v.Severity)
		record.Labels["status"] = string(v.Status)
	default:
		record.Type = "unknown"
	}
	
	// 序列化数据
	jsonData, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	
	// 获取文件路径
	filePath := fs.getFilePath(record.Type, record.Timestamp)
	
	// 检查文件大小
	if fs.shouldRotateFile(filePath) {
		filePath = fs.getRotatedFilePath(record.Type, record.Timestamp)
	}
	
	// 追加数据到文件
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	
	if _, err := file.Write(append(jsonData, '\n')); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}
	
	return nil
}

// Load 从文件加载数据
func (fs *FileStorage) Load(query *Query) ([]interface{}, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	var results []interface{}
	
	// 获取所有相关文件
	files, err := fs.getFiles(query.Type, query.StartTime, query.EndTime)
	if err != nil {
		return nil, err
	}
	
	// 从文件中读取数据
	for _, file := range files {
		records, err := fs.readFile(file)
		if err != nil {
			continue // 忽略读取失败的文件
		}
		
		// 过滤数据
		for _, record := range records {
			if fs.matchesQuery(record, query) {
				results = append(results, record.Data)
			}
		}
	}
	
	// 排序
	if query.OrderBy != "" {
		fs.sortResults(results, query.OrderBy, query.OrderDesc)
	}
	
	// 分页
	if query.Offset > 0 || query.Limit > 0 {
		results = fs.paginateResults(results, query.Offset, query.Limit)
	}
	
	return results, nil
}

// Delete 删除数据
func (fs *FileStorage) Delete(query *Query) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	// 获取所有相关文件
	files, err := fs.getFiles(query.Type, query.StartTime, query.EndTime)
	if err != nil {
		return err
	}
	
	// 如果查询条件过于宽泛，直接删除文件
	if query.Labels == nil && query.StartTime == nil && query.EndTime == nil {
		for _, file := range files {
			if err := os.Remove(file); err != nil {
				return fmt.Errorf("failed to delete file %s: %w", file, err)
			}
		}
		return nil
	}
	
	// 否则需要重写文件，移除匹配的记录
	for _, file := range files {
		if err := fs.deleteFromFile(file, query); err != nil {
			return err
		}
	}
	
	return nil
}

// Close 关闭存储
func (fs *FileStorage) Close() error {
	// 文件存储不需要特殊的关闭操作
	return nil
}

// getFilePath 获取文件路径
func (fs *FileStorage) getFilePath(dataType string, timestamp time.Time) string {
	dateStr := timestamp.Format("2006-01-02")
	filename := fmt.Sprintf("%s_%s.jsonl", dataType, dateStr)
	return filepath.Join(fs.dataDir, filename)
}

// getRotatedFilePath 获取轮转后的文件路径
func (fs *FileStorage) getRotatedFilePath(dataType string, timestamp time.Time) string {
	dateStr := timestamp.Format("2006-01-02")
	timeStr := timestamp.Format("15-04-05")
	filename := fmt.Sprintf("%s_%s_%s.jsonl", dataType, dateStr, timeStr)
	return filepath.Join(fs.dataDir, filename)
}

// shouldRotateFile 检查是否需要轮转文件
func (fs *FileStorage) shouldRotateFile(filePath string) bool {
	stat, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	return stat.Size() >= fs.config.MaxFileSize
}

// getFiles 获取相关文件列表
func (fs *FileStorage) getFiles(dataType string, startTime, endTime *time.Time) ([]string, error) {
	pattern := fmt.Sprintf("%s_*.jsonl", dataType)
	matches, err := filepath.Glob(filepath.Join(fs.dataDir, pattern))
	if err != nil {
		return nil, err
	}
	
	var files []string
	for _, match := range matches {
		// 如果有时间范围限制，检查文件是否在范围内
		if startTime != nil || endTime != nil {
			if !fs.fileInTimeRange(match, startTime, endTime) {
				continue
			}
		}
		files = append(files, match)
	}
	
	return files, nil
}

// fileInTimeRange 检查文件是否在时间范围内
func (fs *FileStorage) fileInTimeRange(filePath string, startTime, endTime *time.Time) bool {
	// 简化实现：总是返回true，实际实现应该解析文件名中的日期
	return true
}

// readFile 读取文件
func (fs *FileStorage) readFile(filePath string) ([]*StorageRecord, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	
	var records []*StorageRecord
	lines := fs.splitLines(string(data))
	
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		var record StorageRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			continue // 忽略解析失败的行
		}
		
		records = append(records, &record)
	}
	
	return records, nil
}

// splitLines 分割行
func (fs *FileStorage) splitLines(data string) []string {
	var lines []string
	start := 0
	
	for i, char := range data {
		if char == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	
	return lines
}

// matchesQuery 检查记录是否匹配查询条件
func (fs *FileStorage) matchesQuery(record *StorageRecord, query *Query) bool {
	// 检查时间范围
	if query.StartTime != nil && record.Timestamp.Before(*query.StartTime) {
		return false
	}
	if query.EndTime != nil && record.Timestamp.After(*query.EndTime) {
		return false
	}
	
	// 检查标签
	if query.Labels != nil {
		for key, value := range query.Labels {
			if recordValue, exists := record.Labels[key]; !exists || recordValue != value {
				return false
			}
		}
	}
	
	return true
}

// sortResults 排序结果
func (fs *FileStorage) sortResults(results []interface{}, orderBy string, orderDesc bool) {
	// 简化实现：按时间戳排序
	sort.Slice(results, func(i, j int) bool {
		var timeI, timeJ time.Time
		
		switch v := results[i].(type) {
		case *SystemInfo:
			timeI = v.Timestamp
		case *Alert:
			timeI = v.StartTime
		}
		
		switch v := results[j].(type) {
		case *SystemInfo:
			timeJ = v.Timestamp
		case *Alert:
			timeJ = v.StartTime
		}
		
		if orderDesc {
			return timeI.After(timeJ)
		}
		return timeI.Before(timeJ)
	})
}

// paginateResults 分页结果
func (fs *FileStorage) paginateResults(results []interface{}, offset, limit int) []interface{} {
	if offset >= len(results) {
		return []interface{}{}
	}
	
	end := len(results)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	
	return results[offset:end]
}

// deleteFromFile 从文件中删除匹配的记录
func (fs *FileStorage) deleteFromFile(filePath string, query *Query) error {
	records, err := fs.readFile(filePath)
	if err != nil {
		return err
	}
	
	var keptRecords []*StorageRecord
	for _, record := range records {
		if !fs.matchesQuery(record, query) {
			keptRecords = append(keptRecords, record)
		}
	}
	
	// 重写文件
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	for _, record := range keptRecords {
		jsonData, err := json.Marshal(record)
		if err != nil {
			continue
		}
		
		if _, err := file.Write(append(jsonData, '\n')); err != nil {
			return err
		}
	}
	
	return nil
}

// startCleanupTask 启动清理任务
func (fs *FileStorage) startCleanupTask() {
	ticker := time.NewTicker(fs.config.SyncInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		fs.cleanup()
	}
}

// cleanup 清理过期数据
func (fs *FileStorage) cleanup() {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	pattern := "*.jsonl"
	matches, err := filepath.Glob(filepath.Join(fs.dataDir, pattern))
	if err != nil {
		return
	}
	
	cutoff := time.Now().Add(-fs.config.RetentionPeriod)
	
	for _, match := range matches {
		stat, err := os.Stat(match)
		if err != nil {
			continue
		}
		
		if stat.ModTime().Before(cutoff) {
			os.Remove(match)
		}
	}
}

// generateID 生成唯一ID
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// MemoryStorage 内存存储实现（用于测试和临时存储）
type MemoryStorage struct {
	mu      sync.RWMutex
	records []*StorageRecord
	maxSize int
}

// NewMemoryStorage 创建内存存储
func NewMemoryStorage(maxSize int) *MemoryStorage {
	if maxSize <= 0 {
		maxSize = 10000
	}
	
	return &MemoryStorage{
		records: make([]*StorageRecord, 0),
		maxSize: maxSize,
	}
}

// Save 保存数据到内存
func (ms *MemoryStorage) Save(data interface{}) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	record := &StorageRecord{
		ID:        generateID(),
		Timestamp: time.Now(),
		Data:      data,
		Labels:    make(map[string]string),
	}
	
	// 设置记录类型
	switch data.(type) {
	case *SystemInfo:
		record.Type = "system"
	case *NetworkInfo:
		record.Type = "network"
	case *ProcessInfo:
		record.Type = "process"
	case *Alert:
		record.Type = "alert"
	default:
		record.Type = "unknown"
	}
	
	ms.records = append(ms.records, record)
	
	// 限制记录数量
	if len(ms.records) > ms.maxSize {
		ms.records = ms.records[1:]
	}
	
	return nil
}

// Load 从内存加载数据
func (ms *MemoryStorage) Load(query *Query) ([]interface{}, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	
	var results []interface{}
	
	for _, record := range ms.records {
		if record.Type == query.Type || query.Type == "" {
			// 检查时间范围
			if query.StartTime != nil && record.Timestamp.Before(*query.StartTime) {
				continue
			}
			if query.EndTime != nil && record.Timestamp.After(*query.EndTime) {
				continue
			}
			
			results = append(results, record.Data)
		}
	}
	
	// 分页
	if query.Offset > 0 || query.Limit > 0 {
		if query.Offset >= len(results) {
			return []interface{}{}, nil
		}
		
		end := len(results)
		if query.Limit > 0 && query.Offset+query.Limit < end {
			end = query.Offset + query.Limit
		}
		
		results = results[query.Offset:end]
	}
	
	return results, nil
}

// Delete 从内存删除数据
func (ms *MemoryStorage) Delete(query *Query) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	var keptRecords []*StorageRecord
	
	for _, record := range ms.records {
		shouldDelete := record.Type == query.Type || query.Type == ""
		
		if shouldDelete && query.StartTime != nil && record.Timestamp.Before(*query.StartTime) {
			shouldDelete = false
		}
		if shouldDelete && query.EndTime != nil && record.Timestamp.After(*query.EndTime) {
			shouldDelete = false
		}
		
		if !shouldDelete {
			keptRecords = append(keptRecords, record)
		}
	}
	
	ms.records = keptRecords
	return nil
}

// Close 关闭内存存储
func (ms *MemoryStorage) Close() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	ms.records = nil
	return nil
}