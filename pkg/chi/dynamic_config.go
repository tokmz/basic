package chi

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// DynamicConfigManager 动态配置管理器
type DynamicConfigManager struct {
	mu      sync.RWMutex
	configs map[string]interface{}
	watches map[string][]ConfigWatcher
}

// ConfigWatcher 配置监听器
type ConfigWatcher interface {
	OnConfigChange(key string, oldValue, newValue interface{}) error
}

// ConfigWatcherFunc 配置监听函数
type ConfigWatcherFunc func(key string, oldValue, newValue interface{}) error

func (f ConfigWatcherFunc) OnConfigChange(key string, oldValue, newValue interface{}) error {
	return f(key, oldValue, newValue)
}

// NewDynamicConfigManager 创建动态配置管理器
func NewDynamicConfigManager() *DynamicConfigManager {
	return &DynamicConfigManager{
		configs: make(map[string]interface{}),
		watches: make(map[string][]ConfigWatcher),
	}
}

// Set 设置配置
func (dcm *DynamicConfigManager) Set(key string, value interface{}) error {
	dcm.mu.Lock()
	defer dcm.mu.Unlock()

	oldValue := dcm.configs[key]
	dcm.configs[key] = value

	// 通知监听器
	if watchers, exists := dcm.watches[key]; exists {
		for _, watcher := range watchers {
			if err := watcher.OnConfigChange(key, oldValue, value); err != nil {
				// 记录错误但不回滚
				fmt.Printf("Config watcher error for key %s: %v\n", key, err)
			}
		}
	}

	return nil
}

// Get 获取配置
func (dcm *DynamicConfigManager) Get(key string) (interface{}, bool) {
	dcm.mu.RLock()
	defer dcm.mu.RUnlock()

	value, exists := dcm.configs[key]
	return value, exists
}

// GetString 获取字符串配置
func (dcm *DynamicConfigManager) GetString(key string, defaultValue string) string {
	if value, exists := dcm.Get(key); exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}

// GetInt 获取整数配置
func (dcm *DynamicConfigManager) GetInt(key string, defaultValue int) int {
	if value, exists := dcm.Get(key); exists {
		switch v := value.(type) {
		case int:
			return v
		case float64:
			return int(v)
		}
	}
	return defaultValue
}

// GetBool 获取布尔配置
func (dcm *DynamicConfigManager) GetBool(key string, defaultValue bool) bool {
	if value, exists := dcm.Get(key); exists {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// GetDuration 获取时间间隔配置
func (dcm *DynamicConfigManager) GetDuration(key string, defaultValue time.Duration) time.Duration {
	if value, exists := dcm.Get(key); exists {
		switch v := value.(type) {
		case time.Duration:
			return v
		case string:
			if d, err := time.ParseDuration(v); err == nil {
				return d
			}
		case float64:
			return time.Duration(v) * time.Second
		}
	}
	return defaultValue
}

// Watch 监听配置变化
func (dcm *DynamicConfigManager) Watch(key string, watcher ConfigWatcher) {
	dcm.mu.Lock()
	defer dcm.mu.Unlock()

	dcm.watches[key] = append(dcm.watches[key], watcher)
}

// WatchFunc 监听配置变化（函数形式）
func (dcm *DynamicConfigManager) WatchFunc(key string, fn func(key string, oldValue, newValue interface{}) error) {
	dcm.Watch(key, ConfigWatcherFunc(fn))
}

// GetAll 获取所有配置
func (dcm *DynamicConfigManager) GetAll() map[string]interface{} {
	dcm.mu.RLock()
	defer dcm.mu.RUnlock()

	result := make(map[string]interface{})
	for k, v := range dcm.configs {
		result[k] = v
	}
	return result
}

// Delete 删除配置
func (dcm *DynamicConfigManager) Delete(key string) {
	dcm.mu.Lock()
	defer dcm.mu.Unlock()

	oldValue := dcm.configs[key]
	delete(dcm.configs, key)

	// 通知监听器
	if watchers, exists := dcm.watches[key]; exists {
		for _, watcher := range watchers {
			if err := watcher.OnConfigChange(key, oldValue, nil); err != nil {
				fmt.Printf("Config watcher error for key %s: %v\n", key, err)
			}
		}
	}
}

// LoadFromJSON 从JSON加载配置
func (dcm *DynamicConfigManager) LoadFromJSON(data []byte) error {
	var configs map[string]interface{}
	if err := json.Unmarshal(data, &configs); err != nil {
		return err
	}

	dcm.mu.Lock()
	defer dcm.mu.Unlock()

	for key, value := range configs {
		oldValue := dcm.configs[key]
		dcm.configs[key] = value

		// 通知监听器
		if watchers, exists := dcm.watches[key]; exists {
			for _, watcher := range watchers {
				if err := watcher.OnConfigChange(key, oldValue, value); err != nil {
					fmt.Printf("Config watcher error for key %s: %v\n", key, err)
				}
			}
		}
	}

	return nil
}

// ExportToJSON 导出配置为JSON
func (dcm *DynamicConfigManager) ExportToJSON() ([]byte, error) {
	dcm.mu.RLock()
	defer dcm.mu.RUnlock()

	return json.Marshal(dcm.configs)
}

// DynamicConfigMiddleware 动态配置中间件
func DynamicConfigMiddleware(dcm *DynamicConfigManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 将动态配置管理器注入到上下文
		c.Set("dynamic_config", dcm)
		c.Next()
	}
}

// GetDynamicConfigFromContext 从上下文获取动态配置管理器
func GetDynamicConfigFromContext(c *gin.Context) *DynamicConfigManager {
	if dcm, exists := c.Get("dynamic_config"); exists {
		if manager, ok := dcm.(*DynamicConfigManager); ok {
			return manager
		}
	}
	return nil
}

// ConfigAPI 配置管理API处理器
func ConfigAPI(dcm *DynamicConfigManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case "GET":
			key := c.Query("key")
			if key == "" {
				// 返回所有配置
				configs := dcm.GetAll()
				c.JSON(200, gin.H{"configs": configs})
			} else {
				// 返回特定配置
				if value, exists := dcm.Get(key); exists {
					c.JSON(200, gin.H{"key": key, "value": value})
				} else {
					c.JSON(404, gin.H{"error": "configuration not found"})
				}
			}

		case "POST", "PUT":
			var request struct {
				Key   string      `json:"key" binding:"required"`
				Value interface{} `json:"value" binding:"required"`
			}

			if err := c.ShouldBindJSON(&request); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}

			if err := dcm.Set(request.Key, request.Value); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, gin.H{"message": "configuration updated"})

		case "DELETE":
			key := c.Query("key")
			if key == "" {
				c.JSON(400, gin.H{"error": "key parameter required"})
				return
			}

			dcm.Delete(key)
			c.JSON(200, gin.H{"message": "configuration deleted"})

		default:
			c.JSON(405, gin.H{"error": "method not allowed"})
		}
	}
}

// FeatureFlag 功能开关
type FeatureFlag struct {
	Name        string                 `json:"name"`
	Enabled     bool                   `json:"enabled"`
	Rules       []FeatureFlagRule      `json:"rules,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Description string                 `json:"description,omitempty"`
}

// FeatureFlagRule 功能开关规则
type FeatureFlagRule struct {
	Type      string      `json:"type"`      // user_id, percentage, time_window
	Condition string      `json:"condition"` // equals, contains, percentage
	Value     interface{} `json:"value"`
}

// FeatureFlagManager 功能开关管理器
type FeatureFlagManager struct {
	mu    sync.RWMutex
	flags map[string]*FeatureFlag
	dcm   *DynamicConfigManager
}

// NewFeatureFlagManager 创建功能开关管理器
func NewFeatureFlagManager(dcm *DynamicConfigManager) *FeatureFlagManager {
	ffm := &FeatureFlagManager{
		flags: make(map[string]*FeatureFlag),
		dcm:   dcm,
	}

	// 监听动态配置变化
	dcm.WatchFunc("feature_flags", func(key string, oldValue, newValue interface{}) error {
		if newValue != nil {
			if flagsData, ok := newValue.(map[string]interface{}); ok {
				ffm.mu.Lock()
				defer ffm.mu.Unlock()

				for name, data := range flagsData {
					if flagBytes, err := json.Marshal(data); err == nil {
						var flag FeatureFlag
						if err := json.Unmarshal(flagBytes, &flag); err == nil {
							ffm.flags[name] = &flag
						}
					}
				}
			}
		}
		return nil
	})

	return ffm
}

// IsEnabled 检查功能是否启用
func (ffm *FeatureFlagManager) IsEnabled(flagName string, context map[string]interface{}) bool {
	ffm.mu.RLock()
	defer ffm.mu.RUnlock()

	flag, exists := ffm.flags[flagName]
	if !exists {
		return false
	}

	if !flag.Enabled {
		return false
	}

	// 检查规则
	for _, rule := range flag.Rules {
		if !ffm.evaluateRule(rule, context) {
			return false
		}
	}

	return true
}

// evaluateRule 评估规则
func (ffm *FeatureFlagManager) evaluateRule(rule FeatureFlagRule, context map[string]interface{}) bool {
	switch rule.Type {
	case "user_id":
		if userID, exists := context["user_id"]; exists {
			switch rule.Condition {
			case "equals":
				return userID == rule.Value
			case "contains":
				if userIDStr, ok := userID.(string); ok {
					if valueStr, ok := rule.Value.(string); ok {
						return userIDStr == valueStr
					}
				}
			}
		}
	case "percentage":
		if pct, ok := rule.Value.(float64); ok {
			// 简单的百分比实现，实际应该基于用户ID进行一致性哈希
			return time.Now().UnixNano()%100 < int64(pct)
		}
	case "time_window":
		// 时间窗口检查
		now := time.Now()
		if timeRange, ok := rule.Value.(map[string]interface{}); ok {
			if startStr, exists := timeRange["start"]; exists {
				if start, err := time.Parse(time.RFC3339, startStr.(string)); err == nil {
					if now.Before(start) {
						return false
					}
				}
			}
			if endStr, exists := timeRange["end"]; exists {
				if end, err := time.Parse(time.RFC3339, endStr.(string)); err == nil {
					if now.After(end) {
						return false
					}
				}
			}
		}
	}

	return true
}

// SetFlag 设置功能开关
func (ffm *FeatureFlagManager) SetFlag(flag *FeatureFlag) {
	ffm.mu.Lock()
	defer ffm.mu.Unlock()

	ffm.flags[flag.Name] = flag

	// 同步到动态配置
	if ffm.dcm != nil {
		flags := make(map[string]*FeatureFlag)
		for name, f := range ffm.flags {
			flags[name] = f
		}
		ffm.dcm.Set("feature_flags", flags)
	}
}

// GetAllFlags 获取所有功能开关
func (ffm *FeatureFlagManager) GetAllFlags() map[string]*FeatureFlag {
	ffm.mu.RLock()
	defer ffm.mu.RUnlock()

	result := make(map[string]*FeatureFlag)
	for name, flag := range ffm.flags {
		result[name] = flag
	}
	return result
}

// FeatureFlagMiddleware 功能开关中间件
func FeatureFlagMiddleware(ffm *FeatureFlagManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("feature_flag_manager", ffm)
		c.Next()
	}
}

// IsFeatureEnabled 检查功能是否启用（从Gin上下文）
func IsFeatureEnabled(c *gin.Context, flagName string) bool {
	if ffm, exists := c.Get("feature_flag_manager"); exists {
		if manager, ok := ffm.(*FeatureFlagManager); ok {
			context := map[string]interface{}{
				"user_id": c.GetString("user_id"),
				"ip":      c.ClientIP(),
				"path":    c.Request.URL.Path,
			}
			return manager.IsEnabled(flagName, context)
		}
	}
	return false
}