package monitor

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// AlertManager 告警管理器
type AlertManager struct {
	mu       sync.RWMutex
	rules    map[string]*AlertRule
	handlers map[string]AlertHandler
	alerts   map[string]*Alert
	config   *AlertConfig
	
	// 通知渠道
	notifyChan chan *Alert
	ctx        context.Context
	cancel     context.CancelFunc
}

// AlertConfig 告警配置
type AlertConfig struct {
	// 检查间隔
	CheckInterval time.Duration `json:"check_interval"`
	// 重复通知间隔
	RepeatInterval time.Duration `json:"repeat_interval"`
	// 最大并发通知数
	MaxConcurrentNotifications int `json:"max_concurrent_notifications"`
	// 告警历史保留时间
	AlertHistoryTTL time.Duration `json:"alert_history_ttl"`
	// 是否启用告警
	Enabled bool `json:"enabled"`
}

// AlertSeverity 告警严重程度
type AlertSeverity string

const (
	SeverityCritical AlertSeverity = "critical"
	SeverityWarning  AlertSeverity = "warning"
	SeverityInfo     AlertSeverity = "info"
)

// AlertStatus 告警状态
type AlertStatus string

const (
	StatusFiring   AlertStatus = "firing"
	StatusResolved AlertStatus = "resolved"
	StatusPending  AlertStatus = "pending"
)

// AlertType 告警类型
type AlertType string

const (
	TypeCPU     AlertType = "cpu"
	TypeMemory  AlertType = "memory"
	TypeDisk    AlertType = "disk"
	TypeNetwork AlertType = "network"
	TypeProcess AlertType = "process"
	TypeCustom  AlertType = "custom"
)

// Alert 告警信息
type Alert struct {
	ID          string            `json:"id"`
	RuleID      string            `json:"rule_id"`
	Type        AlertType         `json:"type"`
	Severity    AlertSeverity     `json:"severity"`
	Status      AlertStatus       `json:"status"`
	Message     string            `json:"message"`
	Description string            `json:"description"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartTime   time.Time         `json:"start_time"`
	EndTime     *time.Time        `json:"end_time,omitempty"`
	LastNotify  time.Time         `json:"last_notify"`
	Count       int               `json:"count"`
	Value       float64           `json:"value"`
	Threshold   float64           `json:"threshold"`
}

// AlertRule 告警规则
type AlertRule struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        AlertType         `json:"type"`
	Severity    AlertSeverity     `json:"severity"`
	Enabled     bool              `json:"enabled"`
	Condition   string            `json:"condition"`     // 条件表达式
	Threshold   float64           `json:"threshold"`     // 阈值
	Operator    string            `json:"operator"`      // 操作符: >, <, >=, <=, ==, !=
	Duration    time.Duration     `json:"duration"`      // 持续时间
	Labels      map[string]string `json:"labels"`        // 标签
	Annotations map[string]string `json:"annotations"`   // 注释
	
	// 检查函数
	CheckFunc func(data interface{}) (bool, float64, error) `json:"-"`
	
	// 规则状态
	lastCheck   time.Time
	triggeredAt *time.Time
	isTriggered bool
}

// AlertHandler 告警处理器接口
type AlertHandler interface {
	Handle(alert *Alert) error
	Name() string
	IsAsync() bool
}

// NewAlertManager 创建告警管理器
func NewAlertManager(config *AlertConfig) *AlertManager {
	if config == nil {
		config = &AlertConfig{
			CheckInterval:               30 * time.Second,
			RepeatInterval:              5 * time.Minute,
			MaxConcurrentNotifications:  10,
			AlertHistoryTTL:             24 * time.Hour,
			Enabled:                     true,
		}
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &AlertManager{
		rules:      make(map[string]*AlertRule),
		handlers:   make(map[string]AlertHandler),
		alerts:     make(map[string]*Alert),
		config:     config,
		notifyChan: make(chan *Alert, config.MaxConcurrentNotifications),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start 启动告警管理器
func (am *AlertManager) Start(monitor Monitor) {
	if !am.config.Enabled {
		return
	}
	
	// 启动通知处理器
	go am.handleNotifications()
	
	// 启动告警检查器
	go am.startChecker(monitor)
	
	// 启动告警历史清理器
	go am.startHistoryCleaner()
}

// Stop 停止告警管理器
func (am *AlertManager) Stop() {
	am.cancel()
	close(am.notifyChan)
}

// AddRule 添加告警规则
func (am *AlertManager) AddRule(rule *AlertRule) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	am.rules[rule.ID] = rule
}

// RemoveRule 移除告警规则
func (am *AlertManager) RemoveRule(ruleID string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	delete(am.rules, ruleID)
}

// AddHandler 添加告警处理器
func (am *AlertManager) AddHandler(handler AlertHandler) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	am.handlers[handler.Name()] = handler
}

// RemoveHandler 移除告警处理器
func (am *AlertManager) RemoveHandler(name string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	delete(am.handlers, name)
}

// GetActiveAlerts 获取活跃告警
func (am *AlertManager) GetActiveAlerts() []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	var activeAlerts []*Alert
	for _, alert := range am.alerts {
		if alert.Status == StatusFiring {
			activeAlerts = append(activeAlerts, alert)
		}
	}
	
	return activeAlerts
}

// GetAllAlerts 获取所有告警
func (am *AlertManager) GetAllAlerts() []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	var allAlerts []*Alert
	for _, alert := range am.alerts {
		allAlerts = append(allAlerts, alert)
	}
	
	return allAlerts
}

// startChecker 启动告警检查器
func (am *AlertManager) startChecker(monitor Monitor) {
	ticker := time.NewTicker(am.config.CheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-am.ctx.Done():
			return
		case <-ticker.C:
			am.checkRules(monitor)
		}
	}
}

// checkRules 检查告警规则
func (am *AlertManager) checkRules(monitor Monitor) {
	am.mu.RLock()
	rules := make([]*AlertRule, 0, len(am.rules))
	for _, rule := range am.rules {
		if rule.Enabled {
			rules = append(rules, rule)
		}
	}
	am.mu.RUnlock()
	
	for _, rule := range rules {
		am.checkRule(rule, monitor)
	}
}

// checkRule 检查单个告警规则
func (am *AlertManager) checkRule(rule *AlertRule, monitor Monitor) {
	var data interface{}
	var err error
	
	// 根据规则类型获取相应数据
	switch rule.Type {
	case TypeCPU, TypeMemory, TypeDisk:
		data, err = monitor.GetSystemInfo()
	case TypeNetwork:
		data, err = monitor.GetNetworkInfo()
	case TypeProcess:
		data, err = monitor.GetProcessInfo()
	}
	
	if err != nil {
		log.Printf("Failed to get data for rule %s: %v", rule.ID, err)
		return
	}
	
	// 执行规则检查
	triggered, value, err := am.evaluateRule(rule, data)
	if err != nil {
		log.Printf("Failed to evaluate rule %s: %v", rule.ID, err)
		return
	}
	
	now := time.Now()
	rule.lastCheck = now
	
	// 处理触发逻辑
	if triggered {
		if !rule.isTriggered {
			// 新触发
			rule.triggeredAt = &now
			rule.isTriggered = true
		} else {
			// 检查是否满足持续时间
			if rule.Duration > 0 && now.Sub(*rule.triggeredAt) >= rule.Duration {
				am.fireAlert(rule, value)
			} else if rule.Duration == 0 {
				am.fireAlert(rule, value)
			}
		}
	} else {
		if rule.isTriggered {
			// 从触发状态恢复
			rule.isTriggered = false
			rule.triggeredAt = nil
			am.resolveAlert(rule.ID)
		}
	}
}

// evaluateRule 评估告警规则
func (am *AlertManager) evaluateRule(rule *AlertRule, data interface{}) (bool, float64, error) {
	// 如果有自定义检查函数，使用自定义函数
	if rule.CheckFunc != nil {
		return rule.CheckFunc(data)
	}
	
	// 默认检查逻辑
	value, err := am.extractValue(rule, data)
	if err != nil {
		return false, 0, err
	}
	
	triggered := am.compareValue(value, rule.Threshold, rule.Operator)
	return triggered, value, nil
}

// extractValue 从数据中提取值
func (am *AlertManager) extractValue(rule *AlertRule, data interface{}) (float64, error) {
	switch rule.Type {
	case TypeCPU:
		if systemInfo, ok := data.(*SystemInfo); ok && systemInfo.CPU != nil {
			return systemInfo.CPU.Usage, nil
		}
	case TypeMemory:
		if systemInfo, ok := data.(*SystemInfo); ok && systemInfo.Memory != nil {
			return systemInfo.Memory.UsagePercent, nil
		}
	case TypeDisk:
		if systemInfo, ok := data.(*SystemInfo); ok && systemInfo.Disks != nil && len(systemInfo.Disks) > 0 {
			// 返回第一个磁盘的使用率
			return systemInfo.Disks[0].UsagePercent, nil
		}
	}
	
	return 0, fmt.Errorf("unable to extract value for rule type %s", rule.Type)
}

// compareValue 比较值
func (am *AlertManager) compareValue(value, threshold float64, operator string) bool {
	switch operator {
	case ">":
		return value > threshold
	case "<":
		return value < threshold
	case ">=":
		return value >= threshold
	case "<=":
		return value <= threshold
	case "==":
		return value == threshold
	case "!=":
		return value != threshold
	default:
		return value > threshold // 默认使用大于比较
	}
}

// fireAlert 触发告警
func (am *AlertManager) fireAlert(rule *AlertRule, value float64) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	alertID := fmt.Sprintf("%s_%d", rule.ID, time.Now().Unix())
	
	// 检查是否已有相同规则的活跃告警
	for _, existingAlert := range am.alerts {
		if existingAlert.RuleID == rule.ID && existingAlert.Status == StatusFiring {
			// 更新现有告警
			existingAlert.Count++
			existingAlert.Value = value
			
			// 检查是否需要重复通知
			if time.Since(existingAlert.LastNotify) >= am.config.RepeatInterval {
				am.notifyAlert(existingAlert)
			}
			return
		}
	}
	
	// 创建新告警
	alert := &Alert{
		ID:          alertID,
		RuleID:      rule.ID,
		Type:        rule.Type,
		Severity:    rule.Severity,
		Status:      StatusFiring,
		Message:     fmt.Sprintf("Alert for rule %s", rule.Name),
		Description: fmt.Sprintf("Value %.2f exceeds threshold %.2f", value, rule.Threshold),
		Labels:      rule.Labels,
		Annotations: rule.Annotations,
		StartTime:   time.Now(),
		Count:       1,
		Value:       value,
		Threshold:   rule.Threshold,
	}
	
	if alert.Labels == nil {
		alert.Labels = make(map[string]string)
	}
	if alert.Annotations == nil {
		alert.Annotations = make(map[string]string)
	}
	
	am.alerts[alertID] = alert
	am.notifyAlert(alert)
}

// resolveAlert 解决告警
func (am *AlertManager) resolveAlert(ruleID string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	for _, alert := range am.alerts {
		if alert.RuleID == ruleID && alert.Status == StatusFiring {
			alert.Status = StatusResolved
			now := time.Now()
			alert.EndTime = &now
			
			// 发送解决通知
			am.notifyAlert(alert)
			break
		}
	}
}

// notifyAlert 通知告警
func (am *AlertManager) notifyAlert(alert *Alert) {
	alert.LastNotify = time.Now()
	
	select {
	case am.notifyChan <- alert:
		// 成功发送到通知通道
	default:
		// 通知通道满，记录错误
		log.Printf("Notification channel full, dropping alert: %s", alert.ID)
	}
}

// handleNotifications 处理通知
func (am *AlertManager) handleNotifications() {
	for alert := range am.notifyChan {
		am.mu.RLock()
		handlers := make([]AlertHandler, 0, len(am.handlers))
		for _, handler := range am.handlers {
			handlers = append(handlers, handler)
		}
		am.mu.RUnlock()
		
		for _, handler := range handlers {
			go func(h AlertHandler, a *Alert) {
				if err := h.Handle(a); err != nil {
					log.Printf("Failed to handle alert %s with handler %s: %v", a.ID, h.Name(), err)
				}
			}(handler, alert)
		}
	}
}

// startHistoryCleaner 启动历史清理器
func (am *AlertManager) startHistoryCleaner() {
	ticker := time.NewTicker(time.Hour) // 每小时清理一次
	defer ticker.Stop()
	
	for {
		select {
		case <-am.ctx.Done():
			return
		case <-ticker.C:
			am.cleanupHistory()
		}
	}
}

// cleanupHistory 清理历史告警
func (am *AlertManager) cleanupHistory() {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	now := time.Now()
	for id, alert := range am.alerts {
		// 清理已解决且超过TTL的告警
		if alert.Status == StatusResolved && alert.EndTime != nil {
			if now.Sub(*alert.EndTime) > am.config.AlertHistoryTTL {
				delete(am.alerts, id)
			}
		}
	}
}

// 预定义的告警处理器

// LogAlertHandler 日志告警处理器
type LogAlertHandler struct {
	logger *log.Logger
}

// NewLogAlertHandler 创建日志告警处理器
func NewLogAlertHandler(logger *log.Logger) *LogAlertHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &LogAlertHandler{logger: logger}
}

func (h *LogAlertHandler) Handle(alert *Alert) error {
	h.logger.Printf("[ALERT] %s - %s: %s (Value: %.2f, Threshold: %.2f)",
		alert.Severity, alert.Type, alert.Message, alert.Value, alert.Threshold)
	return nil
}

func (h *LogAlertHandler) Name() string {
	return "log"
}

func (h *LogAlertHandler) IsAsync() bool {
	return false
}

// EmailAlertHandler 邮件告警处理器（示例）
type EmailAlertHandler struct {
	smtpServer string
	from       string
	to         []string
}

// NewEmailAlertHandler 创建邮件告警处理器
func NewEmailAlertHandler(smtpServer, from string, to []string) *EmailAlertHandler {
	return &EmailAlertHandler{
		smtpServer: smtpServer,
		from:       from,
		to:         to,
	}
}

func (h *EmailAlertHandler) Handle(alert *Alert) error {
	// 这里应该实现实际的邮件发送逻辑
	log.Printf("[EMAIL] Sending alert to %v: %s", h.to, alert.Message)
	return nil
}

func (h *EmailAlertHandler) Name() string {
	return "email"
}

func (h *EmailAlertHandler) IsAsync() bool {
	return true
}

// WebhookAlertHandler Webhook告警处理器（示例）
type WebhookAlertHandler struct {
	url string
}

// NewWebhookAlertHandler 创建Webhook告警处理器
func NewWebhookAlertHandler(url string) *WebhookAlertHandler {
	return &WebhookAlertHandler{url: url}
}

func (h *WebhookAlertHandler) Handle(alert *Alert) error {
	// 这里应该实现实际的HTTP请求逻辑
	log.Printf("[WEBHOOK] Sending alert to %s: %s", h.url, alert.Message)
	return nil
}

func (h *WebhookAlertHandler) Name() string {
	return "webhook"
}

func (h *WebhookAlertHandler) IsAsync() bool {
	return true
}