package main

import (
	"log"
	"time"

	"github.com/tokmz/basic/pkg/database"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Product 产品模型
type Product struct {
	ID          uint    `gorm:"primarykey"`
	Name        string  `gorm:"not null"`
	Price       float64 `gorm:"not null"`
	Category    string  `gorm:"index"`
	Description string  `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func main() {
	// 创建生产级 zap 日志器
	zapConfig := zap.NewProductionConfig()
	zapConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	zapLogger, err := zapConfig.Build()
	if err != nil {
		log.Fatal("创建 zap 日志器失败:", err)
	}
	defer zapLogger.Sync()

	// 高级配置，包含多个从库和自定义设置
	config := &database.DatabaseConfig{
		Master: database.MasterConfig{
			Driver: "postgres",
			DSN:    "host=localhost user=admin password=secret dbname=master_db port=5432 sslmode=disable TimeZone=UTC",
		},
		Slaves: []database.SlaveConfig{
			{
				Driver: "postgres",
				DSN:    "host=slave1.example.com user=reader password=secret dbname=slave_db port=5432 sslmode=require TimeZone=UTC",
				Weight: 3, // 主要从库，最高权重
			},
			{
				Driver: "postgres",
				DSN:    "host=slave2.example.com user=reader password=secret dbname=slave_db port=5432 sslmode=require TimeZone=UTC",
				Weight: 2, // 次要从库
			},
			{
				Driver: "postgres",
				DSN:    "host=slave3.example.com user=reader password=secret dbname=slave_db port=5432 sslmode=require TimeZone=UTC",
				Weight: 1, // 备用从库，最低权重
			},
		},
		Pool: database.PoolConfig{
			MaxOpenConns:    100,
			MaxIdleConns:    20,
			ConnMaxLifetime: time.Hour * 2,
			ConnMaxIdleTime: time.Minute * 15,
		},
		Logger: database.LoggerConfig{
			ZapLogger:     zapLogger,
			Level:         database.Warn,         // 生产环境中仅记录警告和错误
			Colorful:      false,                 // 生产日志禁用颜色
			SlowThreshold: time.Millisecond * 50, // 严格的慢查询阈值
		},
		SlowQuery: database.SlowQueryConfig{
			Enabled:   true,
			Threshold: time.Millisecond * 50, // 积极的慢查询检测
			Logger:    zapLogger,
		},
		HealthCheck: database.HealthCheckConfig{
			Enabled:     true,
			Interval:    time.Minute,     // 每分钟检查一次
			Timeout:     time.Second * 5, // 5秒超时
			MaxFailures: 2,               // 2次失败后标记为不健康
			Logger:      zapLogger,
		},
	}

	// 创建数据库客户端
	client, err := database.NewClient(config)
	if err != nil {
		log.Fatal("创建数据库客户端失败:", err)
	}
	defer client.Close()

	// 等待数据库准备就绪
	if err := client.WaitForHealthy(time.Minute); err != nil {
		log.Printf("警告: 数据库未完全健康: %v", err)
	}

	db := client.DB()

	// 自动迁移模式
	if err := db.AutoMigrate(&Product{}); err != nil {
		log.Fatal("模式迁移失败:", err)
	}

	// 演示事务（总是路由到主库）
	err = db.Transaction(func(tx *gorm.DB) error {
		products := []Product{
			{Name: "笔记本电脑", Price: 6999.99, Category: "电子产品", Description: "高性能笔记本"},
			{Name: "鼠标", Price: 199.99, Category: "电子产品", Description: "无线鼠标"},
			{Name: "键盘", Price: 559.99, Category: "电子产品", Description: "机械键盘"},
		}

		if err := tx.CreateInBatches(products, 100).Error; err != nil {
			return err
		}

		log.Printf("在事务中创建了 %d 个产品", len(products))
		return nil
	})

	if err != nil {
		log.Fatal("事务失败:", err)
	}

	// 演示读操作（路由到从库）
	var products []Product
	if err := db.Where("category = ?", "电子产品").Find(&products).Error; err != nil {
		log.Fatal("查询产品失败:", err)
	}
	log.Printf("找到 %d 个电子产品", len(products))

	// 演示可能较慢的复杂查询
	var expensiveProducts []Product
	start := time.Now()
	if err := db.Where("price > ?", 500.0).
		Order("price DESC").
		Limit(10).
		Find(&expensiveProducts).Error; err != nil {
		log.Fatal("查询高价产品失败:", err)
	}
	queryDuration := time.Since(start)
	log.Printf("复杂查询耗时 %v，找到 %d 个高价产品", queryDuration, len(expensiveProducts))

	// 演示原生 SQL（用于复杂查询）
	var avgPrice float64
	if err := db.Raw("SELECT AVG(price) FROM products WHERE category = ?", "电子产品").
		Scan(&avgPrice).Error; err != nil {
		log.Fatal("计算平均价格失败:", err)
	}
	log.Printf("电子产品平均价格: ¥%.2f", avgPrice)

	// 监控慢查询
	if slowStats := client.GetSlowQueryStats(); slowStats != nil {
		log.Printf("=== 慢查询统计 ===")
		log.Printf("慢查询总数: %d", slowStats.TotalSlowQueries)

		if slowStats.SlowestQuery != nil {
			log.Printf("最慢查询: %s", slowStats.SlowestQuery.SQL)
			log.Printf("耗时: %v", slowStats.SlowestQuery.Duration)
			log.Printf("时间戳: %v", slowStats.SlowestQuery.Timestamp)
		}

		log.Printf("查询类型统计:")
		for queryType, stats := range slowStats.QueryTypeStats {
			log.Printf("  %s: %d 次查询，平均: %v，最大: %v",
				queryType, stats.Count, stats.AverageTime, stats.MaxTime)
		}

		log.Printf("最近慢查询: %d 次", len(slowStats.RecentSlowQueries))
	}

	// 监控数据库健康状态
	healthStatus := client.GetHealthStatus()
	log.Printf("=== 数据库健康状态 ===")
	log.Printf("主库健康: %v (失败次数: %d)",
		healthStatus.Master.Healthy, healthStatus.Master.FailureCount)

	if healthStatus.Master.LastError != "" {
		log.Printf("主库最后错误: %s", healthStatus.Master.LastError)
	}

	for i, slave := range healthStatus.Slaves {
		log.Printf("从库 %d 健康: %v (失败次数: %d)",
			i, slave.Healthy, slave.FailureCount)
		if slave.LastError != "" {
			log.Printf("从库 %d 最后错误: %s", i, slave.LastError)
		}
	}

	// 演示运行时更新 zap 日志器
	newZapConfig := zap.NewDevelopmentConfig()
	newZapLogger, err := newZapConfig.Build()
	if err == nil {
		if err := client.SetZapLogger(newZapLogger); err != nil {
			log.Printf("更新 zap 日志器失败: %v", err)
		} else {
			log.Println("成功更新 zap 日志器")
		}
	}

	// 演示优雅关闭
	log.Println("执行优雅关闭...")

	// 关闭前重置慢查询统计（可选）
	client.ResetSlowQueryStats()

	// 最后健康检查
	client.ForceHealthCheck()

	log.Println("高级示例已成功完成")
}
