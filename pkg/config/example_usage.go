package config

import (
	"fmt"
	"log"
	"time"
)

// ExampleBasicUsage 基本使用示例
func ExampleBasicUsage() {
	// 创建基本配置
	config := DefaultConfig()
	config.ConfigName = "app"
	config.ConfigType = "yaml"
	config.ConfigPaths = []string{".", "./config"}
	
	// 创建配置管理器
	manager := NewViperManager(config)
	
	// 加载配置
	if err := manager.Load(); err != nil {
		log.Printf("Failed to load config: %v", err)
		// 在实际应用中，可以使用默认值继续运行
	}
	
	// 读取配置值
	dbHost := manager.GetString("database.host")
	dbPort := manager.GetInt("database.port")
	debugMode := manager.GetBool("debug")
	timeout := manager.GetDuration("timeout")
	
	fmt.Printf("Database: %s:%d\n", dbHost, dbPort)
	fmt.Printf("Debug: %t\n", debugMode)
	fmt.Printf("Timeout: %v\n", timeout)
	
	// 设置配置值
	manager.Set("runtime.started_at", time.Now())
	manager.Set("runtime.version", "1.0.0")
	
	// 获取所有配置
	allSettings := manager.GetAllSettings()
	fmt.Printf("Total config keys: %d\n", len(allSettings))
	
	// 关闭管理器
	manager.Close()
}

// ExampleConfigBuilder Builder模式使用示例
func ExampleConfigBuilder() {
	// 使用Builder模式创建配置
	manager, err := NewConfigBuilder().
		WithConfigName("myapp").
		WithConfigType("yaml").
		WithConfigPaths("./config", "/etc/myapp").
		WithEnvPrefix("MYAPP").
		WithWatchConfig(true).
		WithValidateOnLoad(true).
		WithDefault("database.port", 5432).
		WithDefault("debug", false).
		Build()
	
	if err != nil {
		log.Fatalf("Failed to build config: %v", err)
	}
	defer manager.Close()
	
	// 使用配置
	fmt.Printf("App started with config: %s\n", manager.GetString("app.name"))
	
	// 监听配置变化
	manager.Watch(func(event Event) {
		fmt.Printf("Config changed: %s = %v (was: %v)\n", 
			event.Key, event.Value, event.OldValue)
	})
	
	// 模拟运行一段时间
	time.Sleep(1 * time.Second)
}

// ExampleQuickStart 快速启动示例
func ExampleQuickStart() {
	// 最简单的启动方式
	manager, err := QuickStart(
		WithYAMLFile("./config/app.yaml"),
		WithDevelopment(),
	)
	
	if err != nil {
		log.Printf("Quick start failed: %v", err)
		return
	}
	defer manager.Close()
	
	// 使用配置
	serverPort := manager.GetInt("server.port")
	if serverPort == 0 {
		serverPort = 8080 // 默认端口
	}
	
	fmt.Printf("Server will start on port: %d\n", serverPort)
}

// ExampleEnvironmentConfig 环境配置示例
func ExampleEnvironmentConfig() {
	// 自动检测环境
	detector := NewEnvironmentDetector()
	currentEnv := detector.DetectEnvironment()
	
	fmt.Printf("Detected environment: %s\n", currentEnv)
	
	// 创建环境管理器
	envManager := NewEnvironmentManager(currentEnv)
	
	// 为不同环境设置不同的配置路径
	envManager.AddConfigPath(Development, "./config/dev")
	envManager.AddConfigPath(Production, "./config/prod")
	envManager.AddConfigPath(Testing, "./config/test")
	
	// 加载环境特定的配置
	manager, err := envManager.LoadWithEnvironment(currentEnv, nil)
	if err != nil {
		log.Printf("Failed to load environment config: %v", err)
		return
	}
	defer manager.Close()
	
	// 使用配置
	logLevel := manager.GetString("log.level")
	fmt.Printf("Log level for %s: %s\n", currentEnv, logLevel)
}

// ExampleEncryptedConfig 加密配置示例
func ExampleEncryptedConfig() {
	config := DefaultConfig()
	config.EncryptionKey = "my-secret-encryption-key"
	
	// 指定需要加密的配置键
	encryptedKeys := []string{
		"database.password",
		"api.secret_key", 
		"jwt.secret",
	}
	
	manager, err := NewEncryptedViperManager(config, encryptedKeys)
	if err != nil {
		log.Fatalf("Failed to create encrypted manager: %v", err)
	}
	defer manager.Close()
	
	// 设置敏感信息（会自动加密）
	manager.SetString("database.password", "super-secret-password")
	manager.SetString("api.secret_key", "api-secret-12345")
	manager.SetString("normal.config", "not-encrypted")
	
	// 读取配置（会自动解密）
	dbPassword := manager.GetString("database.password")
	apiSecret := manager.GetString("api.secret_key")
	normalConfig := manager.GetString("normal.config")
	
	fmt.Printf("DB Password: %s\n", dbPassword) // 自动解密
	fmt.Printf("API Secret: %s\n", apiSecret)   // 自动解密
	fmt.Printf("Normal Config: %s\n", normalConfig) // 不加密
	
	// 显式加密/解密操作
	if err := manager.SetEncrypted("manual.secret", "manually-encrypted-value"); err != nil {
		log.Printf("Manual encryption failed: %v", err)
	}
	
	if decrypted, err := manager.GetDecrypted("manual.secret"); err == nil {
		fmt.Printf("Manual decryption: %s\n", decrypted)
	}
}

// ExampleConfigValidation 配置验证示例
func ExampleConfigValidation() {
	// 创建验证器
	validator := NewDefaultValidator()
	
	// 添加验证规则
	validator.AddRule(ValidationRule{
		Key:      "server.port",
		Required: true,
		Type:     "int",
		MinValue: 1024,
		MaxValue: 65535,
	})
	
	validator.AddRule(ValidationRule{
		Key:     "log.level",
		Type:    "string",
		Options: []interface{}{"debug", "info", "warn", "error"},
	})
	
	validator.AddRule(ValidationRule{
		Key:      "database.url",
		Required: true,
		Type:     "string",
		Pattern:  `^(mysql|postgres)://.*`, // 正则表达式验证
	})
	
	// 自定义验证函数
	validator.AddRule(ValidationRule{
		Key:      "timeout",
		Type:     "duration",
		CustomFunc: func(value interface{}) error {
			if dur, ok := value.(time.Duration); ok {
				if dur < time.Second || dur > time.Hour {
					return fmt.Errorf("timeout must be between 1s and 1h")
				}
			}
			return nil
		},
	})
	
	// 创建配置管理器
	manager := NewViperManager(DefaultConfig())
	
	// 设置配置值
	manager.Set("server.port", 8080)
	manager.Set("log.level", "info")
	manager.Set("database.url", "mysql://user:pass@localhost/db")
	manager.Set("timeout", 30*time.Second)
	
	// 验证配置
	if err := manager.Validate(validator); err != nil {
		log.Printf("Config validation failed: %v", err)
	} else {
		fmt.Println("Config validation passed!")
	}
	
	// 测试无效配置
	manager.Set("server.port", 80) // 端口太小
	manager.Set("log.level", "invalid") // 无效级别
	
	if err := manager.Validate(validator); err != nil {
		fmt.Printf("Expected validation failure: %v\n", err)
	}
}

// ExampleStructUnmarshal 结构体解析示例
func ExampleStructUnmarshal() {
	manager := NewViperManager(DefaultConfig())
	
	// 设置一些嵌套配置
	manager.Set("database.host", "localhost")
	manager.Set("database.port", 5432)
	manager.Set("database.username", "admin")
	manager.Set("database.password", "secret")
	manager.Set("database.ssl_mode", "require")
	
	manager.Set("redis.host", "127.0.0.1")
	manager.Set("redis.port", 6379)
	manager.Set("redis.db", 0)
	
	manager.Set("server.host", "0.0.0.0")
	manager.Set("server.port", 8080)
	manager.Set("server.read_timeout", "30s")
	manager.Set("server.write_timeout", "30s")
	
	// 定义配置结构体
	type DatabaseConfig struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
		SSLMode  string `mapstructure:"ssl_mode"`
	}
	
	type RedisConfig struct {
		Host string `mapstructure:"host"`
		Port int    `mapstructure:"port"`
		DB   int    `mapstructure:"db"`
	}
	
	type ServerConfig struct {
		Host         string        `mapstructure:"host"`
		Port         int           `mapstructure:"port"`
		ReadTimeout  time.Duration `mapstructure:"read_timeout"`
		WriteTimeout time.Duration `mapstructure:"write_timeout"`
	}
	
	type AppConfig struct {
		Database DatabaseConfig `mapstructure:"database"`
		Redis    RedisConfig    `mapstructure:"redis"`
		Server   ServerConfig   `mapstructure:"server"`
	}
	
	// 解析配置到结构体
	var appConfig AppConfig
	if err := manager.Unmarshal(&appConfig); err != nil {
		log.Fatalf("Failed to unmarshal config: %v", err)
	}
	
	fmt.Printf("Database: %s:%d\n", appConfig.Database.Host, appConfig.Database.Port)
	fmt.Printf("Redis: %s:%d\n", appConfig.Redis.Host, appConfig.Redis.Port)
	fmt.Printf("Server: %s:%d\n", appConfig.Server.Host, appConfig.Server.Port)
	
	// 也可以只解析特定部分
	var dbConfig DatabaseConfig
	if err := manager.UnmarshalKey("database", &dbConfig); err != nil {
		log.Printf("Failed to unmarshal database config: %v", err)
	} else {
		fmt.Printf("DB SSL Mode: %s\n", dbConfig.SSLMode)
	}
}

// ExampleGlobalConfig 全局配置示例
func ExampleGlobalConfig() {
	// 初始化全局配置
	config := DevelopmentConfig()
	config.ConfigName = "global-app"
	
	if err := Init(config); err != nil {
		log.Printf("Failed to initialize global config: %v", err)
		return
	}
	defer Close()
	
	// 使用全局函数
	Set("app.name", "My Global App")
	Set("app.version", "1.0.0")
	Set("app.started_at", time.Now())
	
	appName := GetString("app.name")
	appVersion := GetString("app.version")
	debugMode := GetBool("debug") // 从配置文件或环境变量读取
	
	fmt.Printf("App: %s v%s\n", appName, appVersion)
	fmt.Printf("Debug mode: %t\n", debugMode)
	
	// 监听全局配置变化
	Watch(func(event Event) {
		fmt.Printf("Global config changed: %s\n", event.Key)
	})
	
	// 验证全局配置
	validator := NewDefaultValidator()
	validator.AddRule(ValidationRule{
		Key:      "app.name",
		Required: true,
		Type:     "string",
	})
	
	if err := Validate(validator); err != nil {
		log.Printf("Global config validation failed: %v", err)
	}
}

// ExampleFactoryPattern 工厂模式示例
func ExampleFactoryPattern() {
	// 使用默认工厂创建不同类型的管理器
	
	// 1. 创建基本Viper管理器
	basicManager, err := CreateManager(ViperManagerType, DefaultConfig())
	if err != nil {
		log.Printf("Failed to create basic manager: %v", err)
		return
	}
	defer basicManager.Close()
	
	// 2. 创建加密管理器
	encConfig := DefaultConfig()
	encConfig.EncryptionKey = "factory-encryption-key"
	encryptedKeys := []string{"secret.key"}
	
	encryptedManager, err := CreateManagerWithOptions(EncryptedManagerType, encConfig, encryptedKeys)
	if err != nil {
		log.Printf("Failed to create encrypted manager: %v", err)
		return
	}
	defer encryptedManager.Close()
	
	// 3. 创建环境管理器
	envManager, err := CreateManagerWithOptions(EnvironmentManagerType, nil, Production)
	if err != nil {
		log.Printf("Failed to create environment manager: %v", err)
		return
	}
	defer envManager.Close()
	
	// 使用管理器注册表
	registry := GetGlobalRegistry()
	
	// 注册管理器
	registry.Register("basic", basicManager)
	registry.Register("encrypted", encryptedManager)
	registry.Register("production", envManager)
	
	// 获取并使用管理器
	if manager, exists := registry.Get("basic"); exists {
		manager.Set("factory.test", "success")
		fmt.Printf("Factory test: %s\n", manager.GetString("factory.test"))
	}
	
	// 列出所有注册的管理器
	managers := registry.List()
	fmt.Printf("Registered managers: %v\n", managers)
}

// ExamplePresetFactory 预设工厂示例
func ExamplePresetFactory() {
	factory := GetPresetFactory()
	
	// 快速创建不同环境的管理器
	devManager, err := factory.CreateDevelopment("./config/dev")
	if err == nil {
		defer devManager.Close()
		fmt.Println("Development manager created")
	}
	
	prodManager, err := factory.CreateProduction("./config/prod")
	if err == nil {
		defer prodManager.Close()
		fmt.Println("Production manager created")
	}
	
	testManager, err := factory.CreateTesting("./config/test")
	if err == nil {
		defer testManager.Close()
		fmt.Println("Testing manager created")
	}
	
	// 创建加密管理器
	encManager, err := factory.CreateEncrypted(
		"preset-encryption-key",
		[]string{"db.password", "api.key"},
		nil,
	)
	if err == nil {
		defer encManager.Close()
		fmt.Println("Encrypted manager created")
	}
}

// ExampleAutoDetection 自动检测示例
func ExampleAutoDetection() {
	// 配置1：基本配置，应该使用Viper管理器
	config1 := DefaultConfig()
	manager1, err := CreateAuto(config1)
	if err == nil {
		defer manager1.Close()
		fmt.Println("Auto-detected manager type: basic")
	}
	
	// 配置2：带加密的配置，应该使用加密管理器
	config2 := DefaultConfig()
	config2.EncryptionKey = "auto-detect-key"
	manager2, err := CreateAuto(config2, []string{"secret"})
	if err == nil {
		defer manager2.Close()
		fmt.Println("Auto-detected manager type: encrypted")
	}
	
	// 配置3：指定环境的配置，应该使用环境管理器
	config3 := DefaultConfig()
	config3.Environment = "production"
	manager3, err := CreateAuto(config3)
	if err == nil {
		defer manager3.Close()
		fmt.Println("Auto-detected manager type: environment")
	}
}

// ExampleErrorHandling 错误处理示例
func ExampleErrorHandling() {
	// 创建错误处理器
	errorHandler := NewDefaultErrorHandler(func(format string, args ...interface{}) {
		log.Printf("[CONFIG] "+format, args...)
	})
	
	// 模拟各种错误
	configErr := ErrConfigNotFound.WithDetails("file: /path/to/config.yaml")
	errorHandler.HandleError(configErr)
	
	validationErr := NewValidationError("database.port", "range", "port must be between 1024-65535", 80)
	errorHandler.HandleError(validationErr)
	
	validationErrs := &ValidationErrors{}
	validationErrs.Add("host", "required", "host is required", nil)
	validationErrs.Add("port", "type", "port must be integer", "invalid")
	errorHandler.HandleError(validationErrs)
	
	// 检查错误类型
	err := ErrEncryptionKeyMissing
	if IsConfigError(err) {
		fmt.Println("This is a config error")
	}
	
	if IsConfigErrorWithCode(err, "ENCRYPTION_KEY_MISSING") {
		fmt.Println("This is an encryption key missing error")
	}
}

// 运行所有示例的函数
func RunAllExamples() {
	fmt.Println("=== Basic Usage ===")
	ExampleBasicUsage()
	
	fmt.Println("\n=== Config Builder ===")
	ExampleConfigBuilder()
	
	fmt.Println("\n=== Quick Start ===")
	ExampleQuickStart()
	
	fmt.Println("\n=== Environment Config ===")
	ExampleEnvironmentConfig()
	
	fmt.Println("\n=== Encrypted Config ===")
	ExampleEncryptedConfig()
	
	fmt.Println("\n=== Config Validation ===")
	ExampleConfigValidation()
	
	fmt.Println("\n=== Struct Unmarshal ===")
	ExampleStructUnmarshal()
	
	fmt.Println("\n=== Global Config ===")
	ExampleGlobalConfig()
	
	fmt.Println("\n=== Factory Pattern ===")
	ExampleFactoryPattern()
	
	fmt.Println("\n=== Preset Factory ===")
	ExamplePresetFactory()
	
	fmt.Println("\n=== Auto Detection ===")
	ExampleAutoDetection()
	
	fmt.Println("\n=== Error Handling ===")
	ExampleErrorHandling()
}