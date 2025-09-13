package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Environment 环境类型
type Environment string

const (
	Development Environment = "development"
	Testing     Environment = "testing"
	Staging     Environment = "staging"
	Production  Environment = "production"
)

// String 返回环境字符串
func (e Environment) String() string {
	return string(e)
}

// IsValid 检查环境是否有效
func (e Environment) IsValid() bool {
	switch e {
	case Development, Testing, Staging, Production:
		return true
	default:
		return false
	}
}

// EnvironmentManager 环境配置管理器
type EnvironmentManager struct {
	currentEnv    Environment
	configPaths   map[Environment][]string
	configNames   map[Environment]string
	baseManager   Manager
	envOverrides  map[Environment]map[string]interface{}
}

// NewEnvironmentManager 创建环境配置管理器
func NewEnvironmentManager(env Environment) *EnvironmentManager {
	if !env.IsValid() {
		env = Development
	}

	return &EnvironmentManager{
		currentEnv:   env,
		configPaths:  make(map[Environment][]string),
		configNames:  make(map[Environment]string),
		envOverrides: make(map[Environment]map[string]interface{}),
	}
}

// SetEnvironment 设置当前环境
func (e *EnvironmentManager) SetEnvironment(env Environment) error {
	if !env.IsValid() {
		return fmt.Errorf("invalid environment: %s", env)
	}
	
	e.currentEnv = env
	return e.reloadForEnvironment()
}

// GetCurrentEnvironment 获取当前环境
func (e *EnvironmentManager) GetCurrentEnvironment() Environment {
	return e.currentEnv
}

// AddConfigPath 为指定环境添加配置路径
func (e *EnvironmentManager) AddConfigPath(env Environment, paths ...string) {
	if e.configPaths[env] == nil {
		e.configPaths[env] = make([]string, 0)
	}
	e.configPaths[env] = append(e.configPaths[env], paths...)
}

// SetConfigName 设置环境的配置文件名
func (e *EnvironmentManager) SetConfigName(env Environment, name string) {
	e.configNames[env] = name
}

// SetEnvironmentOverride 设置环境特定的配置覆盖
func (e *EnvironmentManager) SetEnvironmentOverride(env Environment, key string, value interface{}) {
	if e.envOverrides[env] == nil {
		e.envOverrides[env] = make(map[string]interface{})
	}
	e.envOverrides[env][key] = value
}

// LoadWithEnvironment 加载指定环境的配置
func (e *EnvironmentManager) LoadWithEnvironment(env Environment, baseConfig *Config) (Manager, error) {
	if !env.IsValid() {
		return nil, fmt.Errorf("invalid environment: %s", env)
	}

	// 创建环境特定的配置
	envConfig := e.createEnvironmentConfig(env, baseConfig)
	
	// 创建配置管理器
	manager := NewViperManager(envConfig)
	
	// 加载配置
	if err := manager.Load(); err != nil {
		return nil, fmt.Errorf("failed to load config for environment %s: %w", env, err)
	}
	
	// 应用环境覆盖
	e.applyEnvironmentOverrides(manager, env)
	
	e.baseManager = manager
	return manager, nil
}

// createEnvironmentConfig 创建环境特定的配置
func (e *EnvironmentManager) createEnvironmentConfig(env Environment, baseConfig *Config) *Config {
	if baseConfig == nil {
		baseConfig = DefaultConfig()
	}
	
	// 复制基础配置
	envConfig := *baseConfig
	envConfig.Environment = env.String()
	
	// 设置环境特定的配置文件名
	if name, exists := e.configNames[env]; exists {
		envConfig.ConfigName = name
	} else {
		// 默认使用环境名作为配置文件名的一部分
		envConfig.ConfigName = fmt.Sprintf("config.%s", env)
	}
	
	// 设置环境特定的配置路径
	if paths, exists := e.configPaths[env]; exists {
		envConfig.ConfigPaths = append(envConfig.ConfigPaths, paths...)
	}
	
	// 根据环境调整配置
	switch env {
	case Development:
		envConfig.WatchConfig = true
		envConfig.ValidateOnLoad = true
		
	case Testing:
		envConfig.WatchConfig = false
		envConfig.ValidateOnLoad = false
		
	case Staging:
		envConfig.WatchConfig = false
		envConfig.ValidateOnLoad = true
		envConfig.CaseSensitive = true
		
	case Production:
		envConfig.WatchConfig = false
		envConfig.ValidateOnLoad = true
		envConfig.CaseSensitive = true
	}
	
	return &envConfig
}

// applyEnvironmentOverrides 应用环境特定的配置覆盖
func (e *EnvironmentManager) applyEnvironmentOverrides(manager Manager, env Environment) {
	if overrides, exists := e.envOverrides[env]; exists {
		for key, value := range overrides {
			manager.Set(key, value)
		}
	}
}

// reloadForEnvironment 为当前环境重新加载配置
func (e *EnvironmentManager) reloadForEnvironment() error {
	if e.baseManager == nil {
		return fmt.Errorf("no base manager configured")
	}
	
	return e.baseManager.Reload()
}

// GetManager 获取当前环境的配置管理器
func (e *EnvironmentManager) GetManager() Manager {
	return e.baseManager
}

// ConfigFileResolver 配置文件解析器
type ConfigFileResolver struct {
	basePath    string
	environments []Environment
	fileTypes   []string
}

// NewConfigFileResolver 创建配置文件解析器
func NewConfigFileResolver(basePath string) *ConfigFileResolver {
	return &ConfigFileResolver{
		basePath:     basePath,
		environments: []Environment{Development, Testing, Staging, Production},
		fileTypes:    []string{"yaml", "yml", "json", "toml"},
	}
}

// ResolveConfigFiles 解析环境配置文件
func (c *ConfigFileResolver) ResolveConfigFiles() map[Environment][]string {
	result := make(map[Environment][]string)
	
	for _, env := range c.environments {
		files := c.findConfigFilesForEnvironment(env)
		if len(files) > 0 {
			result[env] = files
		}
	}
	
	return result
}

// findConfigFilesForEnvironment 查找指定环境的配置文件
func (c *ConfigFileResolver) findConfigFilesForEnvironment(env Environment) []string {
	var files []string
	
	patterns := []string{
		fmt.Sprintf("config.%s", env),
		fmt.Sprintf("config-%s", env),
		fmt.Sprintf("%s.config", env),
		fmt.Sprintf("%s-config", env),
	}
	
	for _, pattern := range patterns {
		for _, fileType := range c.fileTypes {
			filename := fmt.Sprintf("%s.%s", pattern, fileType)
			fullPath := filepath.Join(c.basePath, filename)
			
			// 这里实际应用中需要检查文件是否存在
			// 为了简化，这里直接添加可能的路径
			files = append(files, fullPath)
		}
	}
	
	return files
}

// EnvironmentDetector 环境检测器
type EnvironmentDetector struct {
	envVarName   string
	defaultEnv   Environment
	customDetector func() Environment
}

// NewEnvironmentDetector 创建环境检测器
func NewEnvironmentDetector() *EnvironmentDetector {
	return &EnvironmentDetector{
		envVarName: "APP_ENV",
		defaultEnv: Development,
	}
}

// SetEnvironmentVariable 设置环境变量名
func (e *EnvironmentDetector) SetEnvironmentVariable(varName string) {
	e.envVarName = varName
}

// SetDefaultEnvironment 设置默认环境
func (e *EnvironmentDetector) SetDefaultEnvironment(env Environment) {
	e.defaultEnv = env
}

// SetCustomDetector 设置自定义环境检测函数
func (e *EnvironmentDetector) SetCustomDetector(detector func() Environment) {
	e.customDetector = detector
}

// DetectEnvironment 检测当前环境
func (e *EnvironmentDetector) DetectEnvironment() Environment {
	// 使用自定义检测器
	if e.customDetector != nil {
		if env := e.customDetector(); env.IsValid() {
			return env
		}
	}
	
	// 从环境变量检测
	if envStr := e.getEnvVar(e.envVarName); envStr != "" {
		env := Environment(strings.ToLower(envStr))
		if env.IsValid() {
			return env
		}
	}
	
	// 从常见的环境变量检测
	commonEnvVars := []string{"NODE_ENV", "GO_ENV", "ENVIRONMENT", "ENV"}
	for _, varName := range commonEnvVars {
		if envStr := e.getEnvVar(varName); envStr != "" {
			env := Environment(strings.ToLower(envStr))
			if env.IsValid() {
				return env
			}
		}
	}
	
	return e.defaultEnv
}

// getEnvVar 获取环境变量（这里简化实现，实际项目中应该使用os.Getenv）
func (e *EnvironmentDetector) getEnvVar(name string) string {
	// TODO: 实际实现中应该使用 os.Getenv(name)
	// 这里为了避免导入os包，暂时返回空字符串
	return ""
}

// MultiEnvironmentManager 多环境配置管理器
type MultiEnvironmentManager struct {
	managers map[Environment]Manager
	current  Environment
	detector *EnvironmentDetector
}

// NewMultiEnvironmentManager 创建多环境配置管理器
func NewMultiEnvironmentManager() *MultiEnvironmentManager {
	return &MultiEnvironmentManager{
		managers: make(map[Environment]Manager),
		detector: NewEnvironmentDetector(),
	}
}

// RegisterEnvironment 注册环境配置管理器
func (m *MultiEnvironmentManager) RegisterEnvironment(env Environment, manager Manager) {
	m.managers[env] = manager
}

// SwitchEnvironment 切换环境
func (m *MultiEnvironmentManager) SwitchEnvironment(env Environment) error {
	if !env.IsValid() {
		return fmt.Errorf("invalid environment: %s", env)
	}
	
	if _, exists := m.managers[env]; !exists {
		return fmt.Errorf("environment %s not registered", env)
	}
	
	m.current = env
	return nil
}

// GetCurrentManager 获取当前环境的配置管理器
func (m *MultiEnvironmentManager) GetCurrentManager() Manager {
	if m.current == "" {
		m.current = m.detector.DetectEnvironment()
	}
	
	return m.managers[m.current]
}

// GetManager 获取指定环境的配置管理器
func (m *MultiEnvironmentManager) GetManager(env Environment) Manager {
	return m.managers[env]
}

// GetCurrentEnvironment 获取当前环境
func (m *MultiEnvironmentManager) GetCurrentEnvironment() Environment {
	if m.current == "" {
		m.current = m.detector.DetectEnvironment()
	}
	return m.current
}