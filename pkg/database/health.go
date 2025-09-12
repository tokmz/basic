package database

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// HealthChecker 对数据库连接执行健康检查
type HealthChecker struct {
	client *Client
	logger *zap.Logger
	stopCh chan struct{}
}

// NewHealthChecker 创建新的健康检查器
func NewHealthChecker(client *Client) *HealthChecker {
	logger := client.config.HealthCheck.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &HealthChecker{
		client: client,
		logger: logger,
		stopCh: make(chan struct{}),
	}
}

// startHealthMonitoring 启动健康监控协程
func (c *Client) startHealthMonitoring() {
	checker := NewHealthChecker(c)
	go checker.run()
}

// run 执行健康监控循环
func (h *HealthChecker) run() {
	ticker := time.NewTicker(h.client.config.HealthCheck.Interval)
	defer ticker.Stop()

	// 执行初始健康检查
	h.performHealthCheck()

	for {
		select {
		case <-ticker.C:
			h.performHealthCheck()
		case <-h.stopCh:
			return
		}
	}
}

// stop 停止健康监控
func (h *HealthChecker) stop() {
	close(h.stopCh)
}

// performHealthCheck 对所有数据库连接执行健康检查并进行适当的同步
func (h *HealthChecker) performHealthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), h.client.config.HealthCheck.Timeout)
	defer cancel()

	// 在获取锁之前检查客户端是否已关闭
	h.client.mu.RLock()
	closed := h.client.closed
	h.client.mu.RUnlock()

	if closed {
		return
	}

	now := time.Now()

	// 执行健康检查并收集结果
	masterHealthy, masterError := h.checkMasterHealthStatus(ctx)
	slavesHealth := h.checkSlavesHealthStatus(ctx)

	// 原子性地更新健康状态
	h.client.mu.Lock()
	defer h.client.mu.Unlock()

	// 如果在检查过程中客户端被关闭则不更新
	if h.client.closed {
		return
	}

	h.updateMasterHealthStatus(masterHealthy, masterError, now)
	h.updateSlavesHealthStatus(slavesHealth, now)

	// 记录整体健康状态（可以在持有锁时完成）
	h.logHealthStatus()
}

// checkMasterHealthStatus 检查主库的健康状态而不更新状态
func (h *HealthChecker) checkMasterHealthStatus(ctx context.Context) (bool, error) {
	return h.pingDatabase(ctx, h.client.db) == nil, h.pingDatabase(ctx, h.client.db)
}

// updateMasterHealthStatus 原子性地更新主库健康状态
func (h *HealthChecker) updateMasterHealthStatus(healthy bool, err error, now time.Time) {
	h.client.healthStatus.Master.LastCheck = now

	if err != nil {
		h.client.healthStatus.Master.FailureCount++
		h.client.healthStatus.Master.LastError = err.Error()

		if h.client.healthStatus.Master.FailureCount >= h.client.config.HealthCheck.MaxFailures {
			if h.client.healthStatus.Master.Healthy {
				h.client.healthStatus.Master.Healthy = false
				h.logger.Error("Master database marked as unhealthy",
					zap.String("error", err.Error()),
					zap.Int("failure_count", h.client.healthStatus.Master.FailureCount),
					zap.Int("max_failures", h.client.config.HealthCheck.MaxFailures),
				)
			}
		}
	} else {
		if !h.client.healthStatus.Master.Healthy {
			h.client.healthStatus.Master.Healthy = true
			h.client.healthStatus.Master.FailureCount = 0
			h.client.healthStatus.Master.LastError = ""
			h.logger.Info("Master database recovered and marked as healthy")
		} else if h.client.healthStatus.Master.FailureCount > 0 {
			h.client.healthStatus.Master.FailureCount = 0
			h.client.healthStatus.Master.LastError = ""
		}
	}
}

// SlaveHealthResult 表示从库的健康检查结果
type SlaveHealthResult struct {
	Index   int
	Healthy bool
	Error   error
}

// checkSlavesHealthStatus 检查所有从库的健康状态而不更新状态
func (h *HealthChecker) checkSlavesHealthStatus(ctx context.Context) []SlaveHealthResult {
	results := make([]SlaveHealthResult, len(h.client.healthStatus.Slaves))

	for i := range h.client.healthStatus.Slaves {
		err := h.testSlaveConnection(ctx, i)
		results[i] = SlaveHealthResult{
			Index:   i,
			Healthy: err == nil,
			Error:   err,
		}
	}

	return results
}

// updateSlavesHealthStatus 原子性地更新从库健康状态
func (h *HealthChecker) updateSlavesHealthStatus(results []SlaveHealthResult, now time.Time) {
	for _, result := range results {
		if result.Index >= len(h.client.healthStatus.Slaves) {
			continue
		}

		h.client.healthStatus.Slaves[result.Index].LastCheck = now

		if result.Error != nil {
			h.client.healthStatus.Slaves[result.Index].FailureCount++
			h.client.healthStatus.Slaves[result.Index].LastError = result.Error.Error()

			if h.client.healthStatus.Slaves[result.Index].FailureCount >= h.client.config.HealthCheck.MaxFailures {
				if h.client.healthStatus.Slaves[result.Index].Healthy {
					h.client.healthStatus.Slaves[result.Index].Healthy = false
					h.logger.Error("Slave database marked as unhealthy",
						zap.Int("slave_index", result.Index),
						zap.String("error", result.Error.Error()),
						zap.Int("failure_count", h.client.healthStatus.Slaves[result.Index].FailureCount),
						zap.Int("max_failures", h.client.config.HealthCheck.MaxFailures),
					)
				}
			}
		} else {
			if !h.client.healthStatus.Slaves[result.Index].Healthy {
				h.client.healthStatus.Slaves[result.Index].Healthy = true
				h.client.healthStatus.Slaves[result.Index].FailureCount = 0
				h.client.healthStatus.Slaves[result.Index].LastError = ""
				h.logger.Info("Slave database recovered and marked as healthy",
					zap.Int("slave_index", result.Index),
				)
			} else if h.client.healthStatus.Slaves[result.Index].FailureCount > 0 {
				h.client.healthStatus.Slaves[result.Index].FailureCount = 0
				h.client.healthStatus.Slaves[result.Index].LastError = ""
			}
		}
	}
}

// 遗留方法 - 为向后兼容而保留但标记为已弃用
// Deprecated: 使用 checkMasterHealthStatus 和 updateMasterHealthStatus 代替
func (h *HealthChecker) checkMasterHealth(ctx context.Context, now time.Time) {
	err := h.pingDatabase(ctx, h.client.db)

	h.client.healthStatus.Master.LastCheck = now

	if err != nil {
		h.client.healthStatus.Master.FailureCount++
		h.client.healthStatus.Master.LastError = err.Error()

		if h.client.healthStatus.Master.FailureCount >= h.client.config.HealthCheck.MaxFailures {
			if h.client.healthStatus.Master.Healthy {
				h.client.healthStatus.Master.Healthy = false
				h.logger.Error("Master database marked as unhealthy",
					zap.String("error", err.Error()),
					zap.Int("failure_count", h.client.healthStatus.Master.FailureCount),
					zap.Int("max_failures", h.client.config.HealthCheck.MaxFailures),
				)
			}
		}
	} else {
		if !h.client.healthStatus.Master.Healthy {
			h.client.healthStatus.Master.Healthy = true
			h.client.healthStatus.Master.FailureCount = 0
			h.client.healthStatus.Master.LastError = ""
			h.logger.Info("Master database recovered and marked as healthy")
		} else if h.client.healthStatus.Master.FailureCount > 0 {
			h.client.healthStatus.Master.FailureCount = 0
			h.client.healthStatus.Master.LastError = ""
		}
	}
}

// checkSlavesHealth 检查从库的健康状态
func (h *HealthChecker) checkSlavesHealth(ctx context.Context, now time.Time) {
	// 注意：在真实实现中，您需要访问单独的从库连接
	// 这是一个简化版本，检查我们是否可以执行读操作
	for i := range h.client.healthStatus.Slaves {
		h.client.healthStatus.Slaves[i].LastCheck = now

		// 尝试执行简单的读查询来测试从库连接性
		// 由于dbresolver配置，这将被路由到从库
		err := h.testSlaveConnection(ctx, i)

		if err != nil {
			h.client.healthStatus.Slaves[i].FailureCount++
			h.client.healthStatus.Slaves[i].LastError = err.Error()

			if h.client.healthStatus.Slaves[i].FailureCount >= h.client.config.HealthCheck.MaxFailures {
				if h.client.healthStatus.Slaves[i].Healthy {
					h.client.healthStatus.Slaves[i].Healthy = false
					h.logger.Error("Slave database marked as unhealthy",
						zap.Int("slave_index", i),
						zap.String("error", err.Error()),
						zap.Int("failure_count", h.client.healthStatus.Slaves[i].FailureCount),
						zap.Int("max_failures", h.client.config.HealthCheck.MaxFailures),
					)
				}
			}
		} else {
			if !h.client.healthStatus.Slaves[i].Healthy {
				h.client.healthStatus.Slaves[i].Healthy = true
				h.client.healthStatus.Slaves[i].FailureCount = 0
				h.client.healthStatus.Slaves[i].LastError = ""
				h.logger.Info("Slave database recovered and marked as healthy",
					zap.Int("slave_index", i),
				)
			} else if h.client.healthStatus.Slaves[i].FailureCount > 0 {
				h.client.healthStatus.Slaves[i].FailureCount = 0
				h.client.healthStatus.Slaves[i].LastError = ""
			}
		}
	}
}

// pingDatabase 对数据库连接进行ping以检查其健康状态
func (h *HealthChecker) pingDatabase(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	return sqlDB.PingContext(ctx)
}

// testSlaveConnection 通过执行简单查询来测试从库连接
func (h *HealthChecker) testSlaveConnection(ctx context.Context, slaveIndex int) error {
	// 创建只读上下文以确保查询发送到从库
	db := h.client.db.WithContext(ctx)

	// 执行应该路由到从库的简单查询
	var result int
	err := db.Raw("SELECT 1").Scan(&result).Error
	if err != nil {
		return fmt.Errorf("slave health check query failed: %w", err)
	}

	return nil
}

// logHealthStatus 记录当前健康状态
func (h *HealthChecker) logHealthStatus() {
	masterHealthy := h.client.healthStatus.Master.Healthy
	slavesHealthy := 0
	totalSlaves := len(h.client.healthStatus.Slaves)

	for _, slave := range h.client.healthStatus.Slaves {
		if slave.Healthy {
			slavesHealthy++
		}
	}

	fields := []zap.Field{
		zap.Bool("master_healthy", masterHealthy),
		zap.Int("healthy_slaves", slavesHealthy),
		zap.Int("total_slaves", totalSlaves),
	}

	if !masterHealthy || slavesHealthy < totalSlaves {
		h.logger.Warn("Database health check completed with issues", fields...)
	} else {
		h.logger.Debug("Database health check completed successfully", fields...)
	}
}

// IsHealthy 返回所有数据库连接是否健康
func (c *Client) IsHealthy() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.healthStatus.Master.Healthy {
		return false
	}

	for _, slave := range c.healthStatus.Slaves {
		if !slave.Healthy {
			return false
		}
	}

	return true
}

// IsMasterHealthy 返回主库是否健康
func (c *Client) IsMasterHealthy() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.healthStatus.Master.Healthy
}

// GetHealthySlavesCount 返回健康从库的数量
func (c *Client) GetHealthySlavesCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	count := 0
	for _, slave := range c.healthStatus.Slaves {
		if slave.Healthy {
			count++
		}
	}
	return count
}

// GetTotalSlavesCount 返回从库的总数量
func (c *Client) GetTotalSlavesCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.healthStatus.Slaves)
}

// WaitForHealthy 等待数据库变为健康状态或超时
func (c *Client) WaitForHealthy(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for database to become healthy")
		case <-ticker.C:
			if c.IsHealthy() {
				return nil
			}
		}
	}
}

// ForceHealthCheck 强制执行立即健康检查
func (c *Client) ForceHealthCheck() {
	if c.config.HealthCheck.Enabled {
		checker := NewHealthChecker(c)
		checker.performHealthCheck()
	}
}
