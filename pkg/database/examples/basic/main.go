package main

import (
	"log"
	"time"

	"github.com/tokmz/basic/pkg/database"
	"go.uber.org/zap"
)

// User 用于演示的用户模型
type User struct {
	ID        uint   `gorm:"primarykey"`
	Name      string `gorm:"not null"`
	Email     string `gorm:"uniqueIndex;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func main() {
	// 创建 zap 日志器（可选）
	zapLogger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("创建 zap 日志器失败:", err)
	}

	// 配置数据库
	config := &database.DatabaseConfig{
		Master: database.MasterConfig{
			Driver: "mysql",
			DSN:    "user:password@tcp(localhost:3306)/master_db?charset=utf8mb4&parseTime=True&loc=Local",
		},
		Slaves: []database.SlaveConfig{
			{
				Driver: "mysql",
				DSN:    "user:password@tcp(localhost:3307)/slave_db1?charset=utf8mb4&parseTime=True&loc=Local",
				Weight: 2, // 更高权重，承担更多流量
			},
			{
				Driver: "mysql",
				DSN:    "user:password@tcp(localhost:3308)/slave_db2?charset=utf8mb4&parseTime=True&loc=Local",
				Weight: 1,
			},
		},
		Pool: database.PoolConfig{
			MaxOpenConns:    50,
			MaxIdleConns:    5,
			ConnMaxLifetime: time.Hour,
			ConnMaxIdleTime: time.Minute * 30,
		},
		Logger: database.LoggerConfig{
			ZapLogger:     zapLogger,
			Level:         database.Info,
			Colorful:      true,
			SlowThreshold: time.Millisecond * 100,
		},
		SlowQuery: database.SlowQueryConfig{
			Enabled:   true,
			Threshold: time.Millisecond * 100,
			Logger:    zapLogger,
		},
		HealthCheck: database.HealthCheckConfig{
			Enabled:     true,
			Interval:    time.Minute * 2,
			Timeout:     time.Second * 10,
			MaxFailures: 3,
			Logger:      zapLogger,
		},
	}

	// 创建数据库客户端
	client, err := database.NewClient(config)
	if err != nil {
		log.Fatal("创建数据库客户端失败:", err)
	}
	defer client.Close()

	// 获取底层 GORM DB 实例
	db := client.DB()

	// 自动迁移模式
	if err := db.AutoMigrate(&User{}); err != nil {
		log.Fatal("迁移模式失败:", err)
	}

	// 写操作示例（路由到主库）
	user := &User{
		Name:  "张三",
		Email: "zhangsan@example.com",
	}
	if err := db.Create(user).Error; err != nil {
		log.Fatal("创建用户失败:", err)
	}
	log.Printf("已创建用户: %+v\n", user)

	// 读操作示例（路由到从库）
	var readUser User
	if err := db.First(&readUser, user.ID).Error; err != nil {
		log.Fatal("读取用户失败:", err)
	}
	log.Printf("已读取用户: %+v\n", readUser)

	// 更新操作示例（路由到主库）
	if err := db.Model(&readUser).Update("name", "李四").Error; err != nil {
		log.Fatal("更新用户失败:", err)
	}
	log.Printf("已更新用户姓名为: 李四\n")

	// 检查数据库健康状态
	if client.IsHealthy() {
		log.Println("数据库状态健康")
	} else {
		log.Println("数据库存在健康问题")
	}

	// 获取健康状态详情
	healthStatus := client.GetHealthStatus()
	log.Printf("主库健康: %v\n", healthStatus.Master.Healthy)
	log.Printf("健康从库: %d/%d\n", client.GetHealthySlavesCount(), client.GetTotalSlavesCount())

	// 获取慢查询统计
	if slowStats := client.GetSlowQueryStats(); slowStats != nil {
		log.Printf("慢查询总数: %d\n", slowStats.TotalSlowQueries)
		if slowStats.SlowestQuery != nil {
			log.Printf("最慢查询: %s (耗时 %v)\n",
				slowStats.SlowestQuery.SQL,
				slowStats.SlowestQuery.Duration,
			)
		}
	}

	// 强制健康检查
	client.ForceHealthCheck()
	log.Println("强制健康检查已完成")

	// 等待数据库变为健康状态（适用于启动检查）
	if err := client.WaitForHealthy(time.Second * 30); err != nil {
		log.Printf("警告: 数据库未完全健康: %v\n", err)
	}

	log.Println("示例已成功完成")
}
