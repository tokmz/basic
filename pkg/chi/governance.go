package chi

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// GovernanceConfig 治理配置
type GovernanceConfig struct {
	EnableCircuitBreaker bool                    `json:"enable_circuit_breaker"`
	EnableRateLimit      bool                    `json:"enable_rate_limit"`
	EnableBulkhead       bool                    `json:"enable_bulkhead"`
	EnableTracing        bool                    `json:"enable_tracing"`
	EnableCache          bool                    `json:"enable_cache"`
	CircuitBreaker       *CircuitBreakerConfig   `json:"circuit_breaker,omitempty"`
	RateLimit           *RateLimitConfig        `json:"rate_limit,omitempty"`
	Bulkhead            *BulkheadConfig         `json:"bulkhead,omitempty"`
	Tracing             *TracerConfig           `json:"tracing,omitempty"`
	Cache               *CacheConfig            `json:"cache,omitempty"`
}

// BulkheadConfig 舱壁配置
type BulkheadConfig struct {
	MaxConcurrent int           `json:"max_concurrent"`
	Timeout       time.Duration `json:"timeout"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Provider string        `json:"provider"`
	TTL      time.Duration `json:"ttl"`
	MaxSize  int           `json:"max_size"`
}

// DefaultGovernanceConfig 默认治理配置
func DefaultGovernanceConfig() *GovernanceConfig {
	return &GovernanceConfig{
		EnableCircuitBreaker: true,
		EnableRateLimit:      true,
		EnableBulkhead:       false,
		EnableTracing:        true,
		EnableCache:          false,
		CircuitBreaker:       func() *CircuitBreakerConfig { c := DefaultCircuitBreakerConfig("default"); return &c }(),
		RateLimit:           DefaultRateLimitConfig(),
		Tracing:             func() *TracerConfig { c := DefaultTracerConfig(); return &c }(),
	}
}

// GovernanceManager 治理管理器
type GovernanceManager struct {
	mu                    sync.RWMutex
	configs               map[string]*GovernanceConfig
	circuitBreakerManager *CircuitBreakerManager
	rateLimiters          map[string]*RateLimiter
	bulkheadExecutors     map[string]*BulkheadExecutor
	tracer                Tracer
	cacheManager          *CacheManager
}

// NewGovernanceManager 创建治理管理器
func NewGovernanceManager() *GovernanceManager {
	return &GovernanceManager{
		configs:               make(map[string]*GovernanceConfig),
		circuitBreakerManager: NewCircuitBreakerManager(),
		rateLimiters:          make(map[string]*RateLimiter),
		bulkheadExecutors:     make(map[string]*BulkheadExecutor),
		tracer:                NewMemoryTracer(DefaultTracerConfig()),
		cacheManager:          NewCacheManager(),
	}
}

// SetConfig 设置服务治理配置
func (gm *GovernanceManager) SetConfig(serviceName string, config *GovernanceConfig) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	
	gm.configs[serviceName] = config
	
	// 初始化相关组件
	if config.EnableRateLimit && config.RateLimit != nil {
		gm.rateLimiters[serviceName] = NewRateLimiter(config.RateLimit.Rate, config.RateLimit.Window)
	}
	
	if config.EnableBulkhead && config.Bulkhead != nil {
		gm.bulkheadExecutors[serviceName] = NewBulkheadExecutor(serviceName, config.Bulkhead.MaxConcurrent)
	}
	
	if config.EnableCircuitBreaker && config.CircuitBreaker != nil {
		gm.circuitBreakerManager.Register(serviceName, *config.CircuitBreaker)
	}
}

// GetConfig 获取服务治理配置
func (gm *GovernanceManager) GetConfig(serviceName string) *GovernanceConfig {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	
	return gm.configs[serviceName]
}

// CreateGovernanceMiddleware 创建治理中间件
func (gm *GovernanceManager) CreateGovernanceMiddleware(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		config := gm.GetConfig(serviceName)
		if config == nil {
			c.Next()
			return
		}
		
		// 应用各种治理策略
		middlewares := gm.buildMiddlewareChain(serviceName, config)
		
		// 执行中间件链
		for _, middleware := range middlewares {
			middleware(c)
			if c.IsAborted() {
				return
			}
		}
	}
}

// buildMiddlewareChain 构建中间件链
func (gm *GovernanceManager) buildMiddlewareChain(serviceName string, config *GovernanceConfig) []gin.HandlerFunc {
	var middlewares []gin.HandlerFunc
	
	// 追踪中间件（最外层）
	if config.EnableTracing {
		middlewares = append(middlewares, TracingMiddleware(gm.tracer))
	}
	
	// 限流中间件
	if config.EnableRateLimit {
		if _, exists := gm.rateLimiters[serviceName]; exists {
			middlewares = append(middlewares, RateLimitMiddleware(*config.RateLimit))
		}
	}
	
	// 舱壁中间件
	if config.EnableBulkhead {
		if bulkhead, exists := gm.bulkheadExecutors[serviceName]; exists {
			middlewares = append(middlewares, BulkheadMiddleware(bulkhead))
		}
	}
	
	// 熔断器中间件
	if config.EnableCircuitBreaker {
		if cb := gm.circuitBreakerManager.Get(serviceName); cb != nil {
			middlewares = append(middlewares, CircuitBreakerMiddleware(cb))
		}
	}
	
	// 缓存中间件（最内层，靠近业务逻辑）
	if config.EnableCache && config.Cache != nil {
		middlewares = append(middlewares, ResponseCacheMiddleware(*gm.cacheManager, config.Cache.TTL))
	}
	
	return middlewares
}

// GetServiceStats 获取服务统计信息
func (gm *GovernanceManager) GetServiceStats(serviceName string) ServiceStats {
	gm.mu.RLock()
	defer gm.mu.RUnlock()
	
	stats := ServiceStats{
		ServiceName: serviceName,
		Timestamp:   time.Now(),
	}
	
	// 熔断器统计
	if cb := gm.circuitBreakerManager.Get(serviceName); cb != nil {
		stats.CircuitBreaker = &CircuitBreakerStats{
			Name:   cb.Name(),
			State:  cb.State().String(),
			Counts: cb.Counts(),
		}
	}
	
	// 舱壁统计
	if bulkhead, exists := gm.bulkheadExecutors[serviceName]; exists {
		bulkheadStats := bulkhead.GetStats()
		stats.Bulkhead = &bulkheadStats
	}
	
	// 追踪统计
	if memTracer, ok := gm.tracer.(*MemoryTracer); ok {
		tracingStats := memTracer.GetTracingStats()
		stats.Tracing = &tracingStats
	}
	
	return stats
}

// ServiceStats 服务统计信息
type ServiceStats struct {
	ServiceName    string                `json:"service_name"`
	Timestamp      time.Time             `json:"timestamp"`
	CircuitBreaker *CircuitBreakerStats  `json:"circuit_breaker,omitempty"`
	Bulkhead       *BulkheadStats        `json:"bulkhead,omitempty"`
	Tracing        *TracingStats         `json:"tracing,omitempty"`
}

// GetAllServiceStats 获取所有服务统计信息
func (gm *GovernanceManager) GetAllServiceStats() map[string]ServiceStats {
	gm.mu.RLock()
	services := make([]string, 0, len(gm.configs))
	for serviceName := range gm.configs {
		services = append(services, serviceName)
	}
	gm.mu.RUnlock()
	
	stats := make(map[string]ServiceStats)
	for _, serviceName := range services {
		stats[serviceName] = gm.GetServiceStats(serviceName)
	}
	return stats
}

// ServiceDiscovery 服务发现接口
type ServiceDiscovery interface {
	Register(service ServiceInfo) error
	Deregister(serviceID string) error
	Discover(serviceName string) ([]ServiceInfo, error)
	Health() error
}

// ServiceInfo 服务信息
type ServiceInfo struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Version  string            `json:"version"`
	Address  string            `json:"address"`
	Port     int               `json:"port"`
	Tags     []string          `json:"tags"`
	Metadata map[string]string `json:"metadata"`
	Health   ServiceHealth     `json:"health"`
}

// ServiceHealth 服务健康状态
type ServiceHealth struct {
	Status      string    `json:"status"`
	LastCheck   time.Time `json:"last_check"`
	CheckURL    string    `json:"check_url"`
	CheckInterval time.Duration `json:"check_interval"`
}

// MemoryServiceDiscovery 内存服务发现实现
type MemoryServiceDiscovery struct {
	mu       sync.RWMutex
	services map[string]map[string]ServiceInfo // serviceName -> serviceID -> ServiceInfo
}

// NewMemoryServiceDiscovery 创建内存服务发现
func NewMemoryServiceDiscovery() *MemoryServiceDiscovery {
	return &MemoryServiceDiscovery{
		services: make(map[string]map[string]ServiceInfo),
	}
}

// Register 注册服务
func (msd *MemoryServiceDiscovery) Register(service ServiceInfo) error {
	msd.mu.Lock()
	defer msd.mu.Unlock()
	
	if msd.services[service.Name] == nil {
		msd.services[service.Name] = make(map[string]ServiceInfo)
	}
	
	msd.services[service.Name][service.ID] = service
	return nil
}

// Deregister 注销服务
func (msd *MemoryServiceDiscovery) Deregister(serviceID string) error {
	msd.mu.Lock()
	defer msd.mu.Unlock()
	
	for serviceName, instances := range msd.services {
		if _, exists := instances[serviceID]; exists {
			delete(instances, serviceID)
			if len(instances) == 0 {
				delete(msd.services, serviceName)
			}
			return nil
		}
	}
	
	return fmt.Errorf("service %s not found", serviceID)
}

// Discover 发现服务
func (msd *MemoryServiceDiscovery) Discover(serviceName string) ([]ServiceInfo, error) {
	msd.mu.RLock()
	defer msd.mu.RUnlock()
	
	instances, exists := msd.services[serviceName]
	if !exists {
		return nil, fmt.Errorf("service %s not found", serviceName)
	}
	
	result := make([]ServiceInfo, 0, len(instances))
	for _, service := range instances {
		result = append(result, service)
	}
	
	return result, nil
}

// Health 健康检查
func (msd *MemoryServiceDiscovery) Health() error {
	return nil // 内存实现总是健康的
}

// LoadBalancer 负载均衡器接口
type LoadBalancer interface {
	Choose(services []ServiceInfo) (*ServiceInfo, error)
	Name() string
}

// RoundRobinLoadBalancer 轮询负载均衡器
type RoundRobinLoadBalancer struct {
	mu      sync.Mutex
	counter int
}

// NewRoundRobinLoadBalancer 创建轮询负载均衡器
func NewRoundRobinLoadBalancer() *RoundRobinLoadBalancer {
	return &RoundRobinLoadBalancer{}
}

// Choose 选择服务实例
func (rrb *RoundRobinLoadBalancer) Choose(services []ServiceInfo) (*ServiceInfo, error) {
	if len(services) == 0 {
		return nil, fmt.Errorf("no available services")
	}
	
	rrb.mu.Lock()
	defer rrb.mu.Unlock()
	
	service := &services[rrb.counter%len(services)]
	rrb.counter++
	
	return service, nil
}

// Name 获取负载均衡器名称
func (rrb *RoundRobinLoadBalancer) Name() string {
	return "round_robin"
}

// ServiceRegistry 服务注册表
type ServiceRegistry struct {
	discovery    ServiceDiscovery
	loadBalancer LoadBalancer
	healthChecker *ServiceHealthChecker
}

// NewServiceRegistry 创建服务注册表
func NewServiceRegistry(discovery ServiceDiscovery, loadBalancer LoadBalancer) *ServiceRegistry {
	return &ServiceRegistry{
		discovery:    discovery,
		loadBalancer: loadBalancer,
		healthChecker: NewServiceHealthChecker(discovery),
	}
}

// ServiceHealthChecker 服务健康检查器
type ServiceHealthChecker struct {
	discovery ServiceDiscovery
	mu        sync.RWMutex
	checkers  map[string]*time.Ticker
}

// NewServiceHealthChecker 创建服务健康检查器
func NewServiceHealthChecker(discovery ServiceDiscovery) *ServiceHealthChecker {
	return &ServiceHealthChecker{
		discovery: discovery,
		checkers:  make(map[string]*time.Ticker),
	}
}

// StartHealthCheck 开始健康检查
func (shc *ServiceHealthChecker) StartHealthCheck(serviceID string, checkURL string, interval time.Duration) {
	shc.mu.Lock()
	defer shc.mu.Unlock()
	
	// 停止现有的检查器
	if ticker, exists := shc.checkers[serviceID]; exists {
		ticker.Stop()
	}
	
	// 创建新的检查器
	ticker := time.NewTicker(interval)
	shc.checkers[serviceID] = ticker
	
	go func() {
		for range ticker.C {
			// 执行健康检查
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Get(checkURL)
			
			isHealthy := err == nil && resp.StatusCode == 200
			if resp != nil {
				resp.Body.Close()
			}
			
			// 更新服务状态（这里简化处理）
			_ = isHealthy
		}
	}()
}

// StopHealthCheck 停止健康检查
func (shc *ServiceHealthChecker) StopHealthCheck(serviceID string) {
	shc.mu.Lock()
	defer shc.mu.Unlock()
	
	if ticker, exists := shc.checkers[serviceID]; exists {
		ticker.Stop()
		delete(shc.checkers, serviceID)
	}
}

// GovernanceMiddleware 综合治理中间件
func GovernanceMiddleware(manager *GovernanceManager, serviceName string) gin.HandlerFunc {
	return manager.CreateGovernanceMiddleware(serviceName)
}

// GovernanceDashboardHandler 治理仪表板处理器
func GovernanceDashboardHandler(manager *GovernanceManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats := manager.GetAllServiceStats()
		
		dashboard := gin.H{
			"title":     "Service Governance Dashboard",
			"timestamp": time.Now(),
			"services":  stats,
			"summary": gin.H{
				"total_services": len(stats),
				"healthy_services": countHealthyServices(stats),
			},
		}
		
		// 根据Accept头返回JSON或HTML
		if c.GetHeader("Accept") == "application/json" {
			c.JSON(200, dashboard)
		} else {
			c.HTML(200, "governance_dashboard.html", dashboard)
		}
	}
}

// countHealthyServices 计算健康服务数量
func countHealthyServices(stats map[string]ServiceStats) int {
	healthy := 0
	for _, stat := range stats {
		if stat.CircuitBreaker == nil || stat.CircuitBreaker.State != "OPEN" {
			healthy++
		}
	}
	return healthy
}

// ConfigurationAPI 配置API处理器
func ConfigurationAPI(manager *GovernanceManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceName := c.Param("service")
		if serviceName == "" {
			c.JSON(400, gin.H{"error": "service name required"})
			return
		}
		
		switch c.Request.Method {
		case "GET":
			config := manager.GetConfig(serviceName)
			if config == nil {
				c.JSON(404, gin.H{"error": "service not found"})
				return
			}
			c.JSON(200, config)
			
		case "PUT", "POST":
			var config GovernanceConfig
			if err := c.ShouldBindJSON(&config); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}
			
			manager.SetConfig(serviceName, &config)
			c.JSON(200, gin.H{"message": "configuration updated"})
			
		default:
			c.JSON(405, gin.H{"error": "method not allowed"})
		}
	}
}