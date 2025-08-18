package logger

import (
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// FieldPool 字段对象池，减少内存分配
var (
	fieldPool = sync.Pool{
		New: func() interface{} {
			return make([]zap.Field, 0, 8)
		},
	}

	bufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 0, 1024)
		},
	}

	// 字符串构建器对象池
	stringBuilderPool = sync.Pool{
		New: func() interface{} {
			return &strings.Builder{}
		},
	}
)

// GetFields 从对象池获取字段切片
func GetFields() []zap.Field {
	return fieldPool.Get().([]zap.Field)[:0]
}

// PutFields 归还字段切片到对象池
func PutFields(fields []zap.Field) {
	if cap(fields) > 32 { // 避免池中对象过大
		return
	}
	fieldPool.Put(fields)
}

// GetBuffer 从对象池获取字节缓冲区
func GetBuffer() []byte {
	return bufferPool.Get().([]byte)[:0]
}

// PutBuffer 归还字节缓冲区到对象池
func PutBuffer(buf []byte) {
	if cap(buf) > 4096 { // 避免池中对象过大
		return
	}
	bufferPool.Put(buf)
}

// getStringBuilder 从对象池获取字符串构建器
func getStringBuilder() *strings.Builder {
	return stringBuilderPool.Get().(*strings.Builder)
}

// putStringBuilder 归还字符串构建器到对象池
func putStringBuilder(sb *strings.Builder) {
	sb.Reset()
	stringBuilderPool.Put(sb)
}

// ZeroAllocLogger 零内存分配日志器
type ZeroAllocLogger struct {
	*Logger
	fields []zap.Field
}

// NewZeroAllocLogger 创建零内存分配日志器
func NewZeroAllocLogger(logger *Logger) *ZeroAllocLogger {
	return &ZeroAllocLogger{
		Logger: logger,
		fields: GetFields(),
	}
}

// AddString 添加字符串字段（零分配）
func (zl *ZeroAllocLogger) AddString(key, value string) *ZeroAllocLogger {
	zl.fields = append(zl.fields, zap.String(key, value))
	return zl
}

// AddInt 添加整数字段（零分配）
func (zl *ZeroAllocLogger) AddInt(key string, value int) *ZeroAllocLogger {
	zl.fields = append(zl.fields, zap.Int(key, value))
	return zl
}

// AddInt64 添加64位整数字段（零分配）
func (zl *ZeroAllocLogger) AddInt64(key string, value int64) *ZeroAllocLogger {
	zl.fields = append(zl.fields, zap.Int64(key, value))
	return zl
}

// AddFloat64 添加浮点数字段（零分配）
func (zl *ZeroAllocLogger) AddFloat64(key string, value float64) *ZeroAllocLogger {
	zl.fields = append(zl.fields, zap.Float64(key, value))
	return zl
}

// AddBool 添加布尔字段（零分配）
func (zl *ZeroAllocLogger) AddBool(key string, value bool) *ZeroAllocLogger {
	zl.fields = append(zl.fields, zap.Bool(key, value))
	return zl
}

// AddTime 添加时间字段（零分配）
func (zl *ZeroAllocLogger) AddTime(key string, value time.Time) *ZeroAllocLogger {
	zl.fields = append(zl.fields, zap.Time(key, value))
	return zl
}

// AddDuration 添加时长字段（零分配）
func (zl *ZeroAllocLogger) AddDuration(key string, value time.Duration) *ZeroAllocLogger {
	zl.fields = append(zl.fields, zap.Duration(key, value))
	return zl
}

// AddError 添加错误字段（零分配）
func (zl *ZeroAllocLogger) AddError(err error) *ZeroAllocLogger {
	if err != nil {
		zl.fields = append(zl.fields, zap.Error(err))
	}
	return zl
}

// Log 记录日志并释放资源
func (zl *ZeroAllocLogger) Log(level LogLevel, msg string) {
	defer PutFields(zl.fields)

	switch level {
	case DebugLevel:
		zl.Logger.Debug(msg, zl.fields...)
	case InfoLevel:
		zl.Logger.Info(msg, zl.fields...)
	case WarnLevel:
		zl.Logger.Warn(msg, zl.fields...)
	case ErrorLevel:
		zl.Logger.Error(msg, zl.fields...)
	case FatalLevel:
		zl.Logger.Fatal(msg, zl.fields...)
	}
}

// Debug 记录调试日志
func (zl *ZeroAllocLogger) Debug(msg string) {
	zl.Log(DebugLevel, msg)
}

// Info 记录信息日志
func (zl *ZeroAllocLogger) Info(msg string) {
	zl.Log(InfoLevel, msg)
}

// Warn 记录警告日志
func (zl *ZeroAllocLogger) Warn(msg string) {
	zl.Log(WarnLevel, msg)
}

// Error 记录错误日志
func (zl *ZeroAllocLogger) Error(msg string) {
	zl.Log(ErrorLevel, msg)
}

// Fatal 记录致命日志
func (zl *ZeroAllocLogger) Fatal(msg string) {
	zl.Log(FatalLevel, msg)
}

// PerformanceMonitor 性能监控器
type PerformanceMonitor struct {
	logger      *Logger
	mu          sync.RWMutex
	stats       PerformanceStats
	latencySum  time.Duration // 延迟总和，用于计算平均值
	latencyCount int64        // 延迟计数
}

// PerformanceStats 性能统计
type PerformanceStats struct {
	TotalLogs     int64         `json:"total_logs"`
	LogRate       float64       `json:"log_rate"`     // 每秒日志数
	AvgLatency    time.Duration `json:"avg_latency"`  // 平均延迟
	MinLatency    time.Duration `json:"min_latency"`  // 最小延迟
	MaxLatency    time.Duration `json:"max_latency"`  // 最大延迟
	MemoryUsage   int64         `json:"memory_usage"` // 内存使用量
	Goroutines    int           `json:"goroutines"`   // Goroutine数量
	LastResetTime time.Time     `json:"last_reset_time"`
}

// NewPerformanceMonitor 创建性能监控器
func NewPerformanceMonitor(logger *Logger) *PerformanceMonitor {
	return &PerformanceMonitor{
		logger: logger,
		stats: PerformanceStats{
			LastResetTime: time.Now(),
		},
	}
}

// RecordLog 记录日志性能
func (pm *PerformanceMonitor) RecordLog(duration time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.stats.TotalLogs++
	pm.latencyCount++
	pm.latencySum += duration

	// 计算平均延迟（更准确的算法）
	pm.stats.AvgLatency = pm.latencySum / time.Duration(pm.latencyCount)

	// 更新最小和最大延迟
	if pm.stats.MinLatency == 0 || duration < pm.stats.MinLatency {
		pm.stats.MinLatency = duration
	}
	if duration > pm.stats.MaxLatency {
		pm.stats.MaxLatency = duration
	}

	// 计算日志速率
	elapsed := time.Since(pm.stats.LastResetTime)
	if elapsed > 0 {
		pm.stats.LogRate = float64(pm.stats.TotalLogs) / elapsed.Seconds()
	}
}

// GetStats 获取性能统计
func (pm *PerformanceMonitor) GetStats() PerformanceStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// 更新运行时统计
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	pm.stats.MemoryUsage = int64(m.Alloc)
	pm.stats.Goroutines = runtime.NumGoroutine()

	return pm.stats
}

// ResetStats 重置统计
func (pm *PerformanceMonitor) ResetStats() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.stats = PerformanceStats{
		LastResetTime: time.Now(),
	}
	pm.latencySum = 0
	pm.latencyCount = 0
}

// LogStats 记录性能统计日志
func (pm *PerformanceMonitor) LogStats() {
	stats := pm.GetStats()
	pm.logger.Info("Performance Stats",
		Int64("total_logs", stats.TotalLogs),
		Float64("log_rate", stats.LogRate),
		Duration("avg_latency", stats.AvgLatency),
		Duration("min_latency", stats.MinLatency),
		Duration("max_latency", stats.MaxLatency),
		Int64("memory_usage", stats.MemoryUsage),
		Int("goroutines", stats.Goroutines),
	)
}

// FastEncoder 快速编码器，减少反射和内存分配
type FastEncoder struct {
	zapcore.Encoder
	buf []byte
}

// NewFastEncoder 创建快速编码器
func NewFastEncoder(config zapcore.EncoderConfig) *FastEncoder {
	return &FastEncoder{
		Encoder: zapcore.NewJSONEncoder(config),
		buf:     GetBuffer(),
	}
}

// EncodeEntry 快速编码条目
func (fe *FastEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	defer PutBuffer(fe.buf)
	return fe.Encoder.EncodeEntry(ent, fields)
}

// StringToBytes 零拷贝字符串转字节切片
func StringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&struct {
		string
		int
	}{s, len(s)}))
}

// BytesToString 零拷贝字节切片转字符串
func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// BatchLogger 批量日志器，用于高吞吐量场景
type BatchLogger struct {
	logger *Logger
	batch  []LogEntry
	mu     sync.Mutex
	size   int
}

// LogEntry 日志条目
type LogEntry struct {
	Level   LogLevel
	Message string
	Fields  []zap.Field
	Time    time.Time
}

// NewBatchLogger 创建批量日志器
func NewBatchLogger(logger *Logger, batchSize int) *BatchLogger {
	return &BatchLogger{
		logger: logger,
		batch:  make([]LogEntry, 0, batchSize),
		size:   batchSize,
	}
}

// Add 添加日志条目到批次
func (bl *BatchLogger) Add(level LogLevel, msg string, fields ...zap.Field) {
	bl.mu.Lock()
	defer bl.mu.Unlock()

	entry := LogEntry{
		Level:   level,
		Message: msg,
		Fields:  fields,
		Time:    time.Now(),
	}

	bl.batch = append(bl.batch, entry)

	if len(bl.batch) >= bl.size {
		bl.flushLocked()
	}
}

// Flush 刷新批次
func (bl *BatchLogger) Flush() {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	bl.flushLocked()
}

// flushLocked 刷新批次（需要持有锁）
func (bl *BatchLogger) flushLocked() {
	if len(bl.batch) == 0 {
		return
	}

	for _, entry := range bl.batch {
		switch entry.Level {
		case DebugLevel:
			bl.logger.Debug(entry.Message, entry.Fields...)
		case InfoLevel:
			bl.logger.Info(entry.Message, entry.Fields...)
		case WarnLevel:
			bl.logger.Warn(entry.Message, entry.Fields...)
		case ErrorLevel:
			bl.logger.Error(entry.Message, entry.Fields...)
		case FatalLevel:
			bl.logger.Fatal(entry.Message, entry.Fields...)
		}
	}

	bl.batch = bl.batch[:0]
}
