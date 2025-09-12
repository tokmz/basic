package database

import (
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/utils"
)

// SlowQueryMonitor 监控慢查询并记录它们
type SlowQueryMonitor struct {
	config SlowQueryConfig
	logger *zap.Logger
	stats  SlowQueryStats
	mu     sync.RWMutex
}

// SlowQueryStats 保存慢查询的统计信息
type SlowQueryStats struct {
	TotalSlowQueries  int64                      `json:"total_slow_queries"`
	SlowestQuery      *SlowQueryRecord           `json:"slowest_query"`
	RecentSlowQueries []*SlowQueryRecord         `json:"recent_slow_queries"`
	QueryTypeStats    map[string]*QueryTypeStats `json:"query_type_stats"`
}

// SlowQueryRecord 表示单个慢查询记录
type SlowQueryRecord struct {
	SQL          string        `json:"sql"`
	Duration     time.Duration `json:"duration"`
	RowsAffected int64         `json:"rows_affected"`
	Timestamp    time.Time     `json:"timestamp"`
	File         string        `json:"file"`
	QueryType    string        `json:"query_type"`
	Error        string        `json:"error,omitempty"`
}

// QueryTypeStats 保存特定查询类型的统计信息
type QueryTypeStats struct {
	Count       int64         `json:"count"`
	TotalTime   time.Duration `json:"total_time"`
	AverageTime time.Duration `json:"average_time"`
	MaxTime     time.Duration `json:"max_time"`
	MinTime     time.Duration `json:"min_time"`
}

// NewSlowQueryMonitor 创建新的慢查询监控器
func NewSlowQueryMonitor(config SlowQueryConfig) *SlowQueryMonitor {
	logger := config.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &SlowQueryMonitor{
		config: config,
		logger: logger,
		stats: SlowQueryStats{
			RecentSlowQueries: make([]*SlowQueryRecord, 0),
			QueryTypeStats:    make(map[string]*QueryTypeStats),
		},
	}
}

// RecordSlowQuery 记录慢查询并进行线程安全的统计更新
func (m *SlowQueryMonitor) RecordSlowQuery(sql string, duration time.Duration, rowsAffected int64, err error) {
	if duration < m.config.Threshold {
		return
	}

	record := &SlowQueryRecord{
		SQL:          sql,
		Duration:     duration,
		RowsAffected: rowsAffected,
		Timestamp:    time.Now(),
		File:         utils.FileWithLineNum(),
		QueryType:    m.extractQueryType(sql),
	}

	if err != nil {
		record.Error = err.Error()
	}

	// 使用写锁进行统计更新以防止竞态条件
	m.mu.Lock()
	defer m.mu.Unlock()

	// 原子性地更新统计信息
	m.updateStatsAtomic(record)

	// 记录慢查询（这可以在释放锁后完成）
	// 但我们在这里做以保持一致性
	m.logSlowQuery(record)
}

// updateStatsAtomic 原子性地更新慢查询统计信息
// 调用者必须持有写锁
func (m *SlowQueryMonitor) updateStatsAtomic(record *SlowQueryRecord) {
	m.stats.TotalSlowQueries++

	// 更新最慢查询
	if m.stats.SlowestQuery == nil || record.Duration > m.stats.SlowestQuery.Duration {
		m.stats.SlowestQuery = record
	}

	// 添加到最近查询（保留最后100个）
	m.stats.RecentSlowQueries = append(m.stats.RecentSlowQueries, record)
	if len(m.stats.RecentSlowQueries) > 100 {
		m.stats.RecentSlowQueries = m.stats.RecentSlowQueries[1:]
	}

	// 更新查询类型统计信息
	queryType := record.QueryType
	if stats, exists := m.stats.QueryTypeStats[queryType]; exists {
		stats.Count++
		stats.TotalTime += record.Duration
		stats.AverageTime = stats.TotalTime / time.Duration(stats.Count)
		if record.Duration > stats.MaxTime {
			stats.MaxTime = record.Duration
		}
		if record.Duration < stats.MinTime || stats.MinTime == 0 {
			stats.MinTime = record.Duration
		}
	} else {
		m.stats.QueryTypeStats[queryType] = &QueryTypeStats{
			Count:       1,
			TotalTime:   record.Duration,
			AverageTime: record.Duration,
			MaxTime:     record.Duration,
			MinTime:     record.Duration,
		}
	}
}

// logSlowQuery 记录慢查询
func (m *SlowQueryMonitor) logSlowQuery(record *SlowQueryRecord) {
	fields := []zap.Field{
		zap.String("sql", record.SQL),
		zap.Duration("duration", record.Duration),
		zap.Int64("rows_affected", record.RowsAffected),
		zap.Time("timestamp", record.Timestamp),
		zap.String("file", record.File),
		zap.String("query_type", record.QueryType),
	}

	if record.Error != "" {
		fields = append(fields, zap.String("error", record.Error))
	}

	m.logger.Warn("Slow query detected", fields...)
}

// extractQueryType 从SQL中提取查询类型
func (m *SlowQueryMonitor) extractQueryType(sql string) string {
	if len(sql) == 0 {
		return "UNKNOWN"
	}

	// 使用strings.Fields进行高效的单词分割
	fields := strings.Fields(sql)
	if len(fields) == 0 {
		return "UNKNOWN"
	}

	// 转换为大写以进行不区分大小写的比较
	firstWord := strings.ToUpper(fields[0])
	switch firstWord {
	case "SELECT":
		return "SELECT"
	case "INSERT":
		return "INSERT"
	case "UPDATE":
		return "UPDATE"
	case "DELETE":
		return "DELETE"
	case "CREATE":
		return "CREATE"
	case "DROP":
		return "DROP"
	case "ALTER":
		return "ALTER"
	default:
		return "OTHER"
	}
}

// GetStats 返回当前慢查询统计信息的线程安全深拷贝
func (m *SlowQueryMonitor) GetStats() SlowQueryStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 使用安全深拷贝避免数据竞争
	var stats SlowQueryStats
	if !safeDeepCopy(m.stats, &stats) {
		// 如果深拷贝失败则回退到空统计信息
		return SlowQueryStats{
			RecentSlowQueries: make([]*SlowQueryRecord, 0),
			QueryTypeStats:    make(map[string]*QueryTypeStats),
		}
	}

	return stats
}

// ResetStats 重置慢查询统计信息
func (m *SlowQueryMonitor) ResetStats() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stats = SlowQueryStats{
		RecentSlowQueries: make([]*SlowQueryRecord, 0),
		QueryTypeStats:    make(map[string]*QueryTypeStats),
	}
}

// SlowQueryPlugin 是用于监控慢查询的GORM插件
type SlowQueryPlugin struct {
	monitor *SlowQueryMonitor
}

// NewSlowQueryPlugin 创建新的慢查询插件
func NewSlowQueryPlugin(monitor *SlowQueryMonitor) *SlowQueryPlugin {
	return &SlowQueryPlugin{
		monitor: monitor,
	}
}

// Name 返回插件名称
func (p *SlowQueryPlugin) Name() string {
	return "slow_query_monitor"
}

// Initialize 初始化插件
func (p *SlowQueryPlugin) Initialize(db *gorm.DB) error {
	// Register callbacks for different operations
	db.Callback().Create().After("gorm:create").Register("slow_query:after_create", p.afterCallback)
	db.Callback().Query().After("gorm:query").Register("slow_query:after_query", p.afterCallback)
	db.Callback().Update().After("gorm:update").Register("slow_query:after_update", p.afterCallback)
	db.Callback().Delete().After("gorm:delete").Register("slow_query:after_delete", p.afterCallback)
	db.Callback().Raw().After("gorm:raw").Register("slow_query:after_raw", p.afterCallback)
	db.Callback().Row().After("gorm:row").Register("slow_query:after_row", p.afterCallback)

	return nil
}

// afterCallback 在每次查询后调用
func (p *SlowQueryPlugin) afterCallback(db *gorm.DB) {
	// 如果监控器不可用则跳过
	if p.monitor == nil {
		return
	}

	if db.Statement != nil && db.Statement.Context != nil {
		if startTime, ok := db.Statement.Context.Value("gorm:started_at").(time.Time); ok {
			// 获取查询持续时间
			duration := time.Since(startTime)
			// 获取受影响的行数
			rowsAffected := db.RowsAffected
			sql := db.Statement.SQL.String()

			// 记录慢查询
			p.monitor.RecordSlowQuery(sql, duration, rowsAffected, db.Error)
		}
	}
}

// GetSlowQueryStats 返回慢查询统计信息
func (c *Client) GetSlowQueryStats() *SlowQueryStats {
	if c.slowQuery == nil {
		return nil
	}

	stats := c.slowQuery.GetStats()
	return &stats
}

// ResetSlowQueryStats 重置慢查询统计信息
func (c *Client) ResetSlowQueryStats() {
	if c.slowQuery != nil {
		c.slowQuery.ResetStats()
	}
}
