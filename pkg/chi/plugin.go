package chi

import (
	"context"
	"fmt"
	"sync"

	"github.com/gin-gonic/gin"
)

// Plugin 插件接口
type Plugin interface {
	// Name 获取插件名称
	Name() string
	
	// Version 获取插件版本
	Version() string
	
	// Init 初始化插件
	Init(app *App) error
	
	// Start 启动插件
	Start(ctx context.Context) error
	
	// Stop 停止插件
	Stop(ctx context.Context) error
	
	// Health 插件健康检查
	Health() PluginHealth
}

// PluginHealth 插件健康状态
type PluginHealth struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// PluginManager 插件管理器
type PluginManager struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
	app     *App
	started bool
}

// NewPluginManager 创建插件管理器
func NewPluginManager(app *App) *PluginManager {
	return &PluginManager{
		plugins: make(map[string]Plugin),
		app:     app,
	}
}

// Register 注册插件
func (pm *PluginManager) Register(plugin Plugin) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	name := plugin.Name()
	if _, exists := pm.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}
	
	// 初始化插件
	if err := plugin.Init(pm.app); err != nil {
		return fmt.Errorf("failed to initialize plugin %s: %w", name, err)
	}
	
	pm.plugins[name] = plugin
	
	// 如果管理器已启动，立即启动插件
	if pm.started {
		ctx := context.Background()
		if err := plugin.Start(ctx); err != nil {
			delete(pm.plugins, name)
			return fmt.Errorf("failed to start plugin %s: %w", name, err)
		}
	}
	
	return nil
}

// Unregister 注销插件
func (pm *PluginManager) Unregister(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}
	
	// 停止插件
	if pm.started {
		ctx := context.Background()
		if err := plugin.Stop(ctx); err != nil {
			return fmt.Errorf("failed to stop plugin %s: %w", name, err)
		}
	}
	
	delete(pm.plugins, name)
	return nil
}

// Get 获取插件
func (pm *PluginManager) Get(name string) (Plugin, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	plugin, exists := pm.plugins[name]
	return plugin, exists
}

// List 列出所有插件
func (pm *PluginManager) List() []Plugin {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	plugins := make([]Plugin, 0, len(pm.plugins))
	for _, plugin := range pm.plugins {
		plugins = append(plugins, plugin)
	}
	return plugins
}

// StartAll 启动所有插件
func (pm *PluginManager) StartAll(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	for name, plugin := range pm.plugins {
		if err := plugin.Start(ctx); err != nil {
			return fmt.Errorf("failed to start plugin %s: %w", name, err)
		}
	}
	
	pm.started = true
	return nil
}

// StopAll 停止所有插件
func (pm *PluginManager) StopAll(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	var errs []error
	for name, plugin := range pm.plugins {
		if err := plugin.Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop plugin %s: %w", name, err))
		}
	}
	
	pm.started = false
	
	if len(errs) > 0 {
		return fmt.Errorf("multiple plugin stop errors: %v", errs)
	}
	
	return nil
}

// HealthCheck 检查所有插件健康状态
func (pm *PluginManager) HealthCheck() map[string]PluginHealth {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	health := make(map[string]PluginHealth)
	for name, plugin := range pm.plugins {
		health[name] = plugin.Health()
	}
	return health
}

// BasePlugin 基础插件实现
type BasePlugin struct {
	name    string
	version string
}

// NewBasePlugin 创建基础插件
func NewBasePlugin(name, version string) *BasePlugin {
	return &BasePlugin{
		name:    name,
		version: version,
	}
}

// Name 获取插件名称
func (bp *BasePlugin) Name() string {
	return bp.name
}

// Version 获取插件版本
func (bp *BasePlugin) Version() string {
	return bp.version
}

// Init 初始化插件（默认实现）
func (bp *BasePlugin) Init(app *App) error {
	return nil
}

// Start 启动插件（默认实现）
func (bp *BasePlugin) Start(ctx context.Context) error {
	return nil
}

// Stop 停止插件（默认实现）
func (bp *BasePlugin) Stop(ctx context.Context) error {
	return nil
}

// Health 插件健康检查（默认实现）
func (bp *BasePlugin) Health() PluginHealth {
	return PluginHealth{
		Status:  "healthy",
		Message: "Plugin is running",
	}
}

// MiddlewarePlugin 中间件插件
type MiddlewarePlugin struct {
	*BasePlugin
	middleware gin.HandlerFunc
	priority   int
}

// NewMiddlewarePlugin 创建中间件插件
func NewMiddlewarePlugin(name, version string, middleware gin.HandlerFunc, priority int) *MiddlewarePlugin {
	return &MiddlewarePlugin{
		BasePlugin: NewBasePlugin(name, version),
		middleware: middleware,
		priority:   priority,
	}
}

// Init 初始化中间件插件
func (mp *MiddlewarePlugin) Init(app *App) error {
	app.Use(mp.middleware)
	return nil
}

// RoutePlugin 路由插件
type RoutePlugin struct {
	*BasePlugin
	setupFunc func(*App) error
}

// NewRoutePlugin 创建路由插件
func NewRoutePlugin(name, version string, setupFunc func(*App) error) *RoutePlugin {
	return &RoutePlugin{
		BasePlugin: NewBasePlugin(name, version),
		setupFunc:  setupFunc,
	}
}

// Init 初始化路由插件
func (rp *RoutePlugin) Init(app *App) error {
	return rp.setupFunc(app)
}

// ServicePlugin 服务插件接口
type ServicePlugin interface {
	Plugin
	Service() interface{}
}

// BackgroundServicePlugin 后台服务插件
type BackgroundServicePlugin struct {
	*BasePlugin
	serviceFunc func(context.Context) error
	cancel      context.CancelFunc
	done        chan struct{}
}

// NewBackgroundServicePlugin 创建后台服务插件
func NewBackgroundServicePlugin(name, version string, serviceFunc func(context.Context) error) *BackgroundServicePlugin {
	return &BackgroundServicePlugin{
		BasePlugin:  NewBasePlugin(name, version),
		serviceFunc: serviceFunc,
		done:        make(chan struct{}),
	}
}

// Start 启动后台服务
func (bsp *BackgroundServicePlugin) Start(ctx context.Context) error {
	serviceCtx, cancel := context.WithCancel(ctx)
	bsp.cancel = cancel
	
	go func() {
		defer close(bsp.done)
		if err := bsp.serviceFunc(serviceCtx); err != nil {
			// 记录错误
		}
	}()
	
	return nil
}

// Stop 停止后台服务
func (bsp *BackgroundServicePlugin) Stop(ctx context.Context) error {
	if bsp.cancel != nil {
		bsp.cancel()
	}
	
	select {
	case <-bsp.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// PluginRegistry 插件注册表
type PluginRegistry struct {
	mu       sync.RWMutex
	registry map[string]func() Plugin
}

// GlobalPluginRegistry 全局插件注册表
var GlobalPluginRegistry = &PluginRegistry{
	registry: make(map[string]func() Plugin),
}

// RegisterPluginFactory 注册插件工厂
func (pr *PluginRegistry) RegisterPluginFactory(name string, factory func() Plugin) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.registry[name] = factory
}

// CreatePlugin 创建插件实例
func (pr *PluginRegistry) CreatePlugin(name string) (Plugin, error) {
	pr.mu.RLock()
	factory, exists := pr.registry[name]
	pr.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("plugin factory %s not found", name)
	}
	
	return factory(), nil
}

// ListFactories 列出所有注册的插件工厂
func (pr *PluginRegistry) ListFactories() []string {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	
	names := make([]string, 0, len(pr.registry))
	for name := range pr.registry {
		names = append(names, name)
	}
	return names
}

// RegisterPlugin 注册插件到全局注册表
func RegisterPlugin(name string, factory func() Plugin) {
	GlobalPluginRegistry.RegisterPluginFactory(name, factory)
}

// CreatePlugin 从全局注册表创建插件
func CreatePlugin(name string) (Plugin, error) {
	return GlobalPluginRegistry.CreatePlugin(name)
}

// 内置插件工厂
func init() {
	// 注册一些内置插件
	RegisterPlugin("cors", func() Plugin {
		return NewMiddlewarePlugin("cors", "1.0.0", CORSMiddleware(), 100)
	})
	
	RegisterPlugin("security", func() Plugin {
		return NewMiddlewarePlugin("security", "1.0.0", SecurityMiddleware(*DefaultSecurityConfig()), 90)
	})
	
	RegisterPlugin("request-id", func() Plugin {
		return NewMiddlewarePlugin("request-id", "1.0.0", RequestIDMiddleware(), 80)
	})
}

// PluginInfo 插件信息
type PluginInfo struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Status      string                 `json:"status"`
	Health      PluginHealth           `json:"health"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// GetPluginInfo 获取插件信息
func (pm *PluginManager) GetPluginInfo() []PluginInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	infos := make([]PluginInfo, 0, len(pm.plugins))
	for _, plugin := range pm.plugins {
		info := PluginInfo{
			Name:    plugin.Name(),
			Version: plugin.Version(),
			Status:  "running",
			Health:  plugin.Health(),
		}
		infos = append(infos, info)
	}
	return infos
}

// ListPlugins 列出所有插件
func (pm *PluginManager) ListPlugins() []PluginInfo {
	return pm.GetPluginInfo()
}

// StartPlugin 启动特定插件
func (pm *PluginManager) StartPlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}
	
	return plugin.Start(context.Background())
}

// StopPlugin 停止特定插件
func (pm *PluginManager) StopPlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}
	
	return plugin.Stop(context.Background())
}