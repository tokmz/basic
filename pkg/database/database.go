package database

import (
	"fmt"
	"sync"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

// Client 表示具有读写分离功能的数据库客户端
type Client struct {
	db           *gorm.DB
	config       *DatabaseConfig
	healthStatus *HealthStatus
	slowQuery    *SlowQueryMonitor
	mu           sync.RWMutex
	closed       bool
}

// HealthStatus 跟踪数据库连接的健康状态
type HealthStatus struct {
	Master struct {
		Healthy      bool      `json:"healthy"`
		LastCheck    time.Time `json:"last_check"`
		FailureCount int       `json:"failure_count"`
		LastError    string    `json:"last_error"`
	} `json:"master"`

	Slaves []struct {
		Index        int       `json:"index"`
		Healthy      bool      `json:"healthy"`
		LastCheck    time.Time `json:"last_check"`
		FailureCount int       `json:"failure_count"`
		LastError    string    `json:"last_error"`
	} `json:"slaves"`
}

// NewClient 使用给定配置创建新的数据库客户端
func NewClient(config *DatabaseConfig) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// 验证配置
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	client := &Client{
		config:       config,
		healthStatus: &HealthStatus{},
	}

	// 初始化从库健康状态
	client.healthStatus.Slaves = make([]struct {
		Index        int       `json:"index"`
		Healthy      bool      `json:"healthy"`
		LastCheck    time.Time `json:"last_check"`
		FailureCount int       `json:"failure_count"`
		LastError    string    `json:"last_error"`
	}, len(config.Slaves))

	for i := range client.healthStatus.Slaves {
		client.healthStatus.Slaves[i].Index = i
		client.healthStatus.Slaves[i].Healthy = true
	}

	// 初始化数据库连接
	if err := client.initDB(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// 初始化慢查询监控器
	if config.SlowQuery.Enabled {
		client.slowQuery = NewSlowQueryMonitor(config.SlowQuery)
	}

	// 如果启用则开始健康监控
	if config.HealthCheck.Enabled {
		go client.startHealthMonitoring()
	}

	return client, nil
}

// initDB 初始化具有读写分离功能的GORM数据库连接
func (c *Client) initDB() error {
	// 创建主库数据库连接
	masterDB, err := c.openDatabase(c.config.Master.Driver, c.config.Master.DSN)
	if err != nil {
		return fmt.Errorf("failed to connect to master database: %w", err)
	}

	// 使用自定义日志器配置GORM
	gormConfig := &gorm.Config{
		Logger: c.createGormLogger(),
	}

	db, err := gorm.Open(masterDB, gormConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize GORM: %w", err)
	}

	// 为主库配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	c.configureConnectionPool(sqlDB)

	// 如果配置了从库则配置读写分离
	if len(c.config.Slaves) > 0 {
		if err := c.configureReadWriteSeparation(db); err != nil {
			return fmt.Errorf("failed to configure read-write separation: %w", err)
		}
	}

	c.db = db
	return nil
}

// configureReadWriteSeparation 为读写分离设置GORM dbresolver
func (c *Client) configureReadWriteSeparation(db *gorm.DB) error {
	// 准备从库配置
	var replicas []gorm.Dialector
	var sources []gorm.Dialector

	// 添加主库作为源
	masterDialector, err := c.openDatabase(c.config.Master.Driver, c.config.Master.DSN)
	if err != nil {
		return fmt.Errorf("failed to create master dialector: %w", err)
	}
	sources = append(sources, masterDialector)

	// 添加从库作为副本
	for _, slave := range c.config.Slaves {
		slaveDialector, err := c.openDatabase(slave.Driver, slave.DSN)
		if err != nil {
			return fmt.Errorf("failed to create slave dialector: %w", err)
		}

		// 根据权重多次添加以实现负载均衡
		weight := slave.Weight
		if weight <= 0 {
			weight = 1
		}
		for i := 0; i < weight; i++ {
			replicas = append(replicas, slaveDialector)
		}
	}

	// 配置dbresolver
	resolver := dbresolver.Register(dbresolver.Config{
		Sources:  sources,
		Replicas: replicas,
		Policy:   dbresolver.RandomPolicy{}, // 随机负载均衡
	})

	// 为副本配置连接池
	resolver.SetConnMaxIdleTime(c.config.Pool.ConnMaxIdleTime)
	resolver.SetConnMaxLifetime(c.config.Pool.ConnMaxLifetime)
	resolver.SetMaxIdleConns(c.config.Pool.MaxIdleConns)
	resolver.SetMaxOpenConns(c.config.Pool.MaxOpenConns)

	return db.Use(resolver)
}

// openDatabase 根据驱动类型创建数据库方言器
func (c *Client) openDatabase(driver, dsn string) (gorm.Dialector, error) {
	switch driver {
	case "mysql":
		return mysql.Open(dsn), nil
	case "postgres":
		return postgres.Open(dsn), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}
}

// configureConnectionPool 配置连接池设置
func (c *Client) configureConnectionPool(sqlDB interface{}) {
	if db, ok := sqlDB.(interface {
		SetMaxOpenConns(int)
		SetMaxIdleConns(int)
		SetConnMaxLifetime(time.Duration)
		SetConnMaxIdleTime(time.Duration)
	}); ok {
		db.SetMaxOpenConns(c.config.Pool.MaxOpenConns)
		db.SetMaxIdleConns(c.config.Pool.MaxIdleConns)
		db.SetConnMaxLifetime(c.config.Pool.ConnMaxLifetime)
		db.SetConnMaxIdleTime(c.config.Pool.ConnMaxIdleTime)
	}
}

// DB 返回底层的GORM数据库实例
func (c *Client) DB() *gorm.DB {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.db
}

// Close 关闭数据库连接
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true

	if c.db != nil {
		sqlDB, err := c.db.DB()
		if err != nil {
			return fmt.Errorf("failed to get underlying sql.DB: %w", err)
		}
		return sqlDB.Close()
	}

	return nil
}

// IsClosed 返回客户端是否已关闭
func (c *Client) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

// GetHealthStatus 返回当前健康状态的线程安全深拷贝
func (c *Client) GetHealthStatus() *HealthStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 使用安全深拷贝避免数据竞争
	var status HealthStatus
	if !safeDeepCopy(c.healthStatus, &status) {
		// 如果深拷贝失败则回退到空状态
		return &HealthStatus{
			Slaves: make([]struct {
				Index        int       `json:"index"`
				Healthy      bool      `json:"healthy"`
				LastCheck    time.Time `json:"last_check"`
				FailureCount int       `json:"failure_count"`
				LastError    string    `json:"last_error"`
			}, 0),
		}
	}

	return &status
}

// validateConfig 验证数据库配置
func validateConfig(config *DatabaseConfig) error {
	if config.Master.Driver == "" {
		return fmt.Errorf("master database driver is required")
	}
	if config.Master.DSN == "" {
		return fmt.Errorf("master database DSN is required")
	}

	// 如果配置了从库则验证从库
	for i, slave := range config.Slaves {
		if slave.Driver == "" {
			return fmt.Errorf("slave %d: driver is required", i)
		}
		if slave.DSN == "" {
			return fmt.Errorf("slave %d: DSN is required", i)
		}
	}

	// 验证连接池配置
	if config.Pool.MaxOpenConns <= 0 {
		return fmt.Errorf("max open connections must be positive")
	}
	if config.Pool.MaxIdleConns < 0 {
		return fmt.Errorf("max idle connections must be non-negative")
	}
	if config.Pool.MaxIdleConns > config.Pool.MaxOpenConns {
		return fmt.Errorf("max idle connections cannot exceed max open connections")
	}

	return nil
}
