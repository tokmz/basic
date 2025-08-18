package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"basic/pkg/config"
)

// AppConfig 应用配置结构体
type AppConfig struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Logging  LoggingConfig  `mapstructure:"logging"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	TLS          TLSConfig     `mapstructure:"tls"`
}

// TLSConfig TLS配置
type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver          string        `mapstructure:"driver"`
	DSN             string        `mapstructure:"dsn"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

func main() {
	fmt.Println("=== 配置管理包使用示例 ===")

	// 示例1: 基础用法
	exampleBasicUsage()

	// 示例2: 环境变量覆盖
	exampleEnvironmentOverride()

	// 示例3: 多种配置格式
	exampleMultipleFormats()

	// 示例4: 结构体绑定
	exampleStructBinding()

	// 示例5: 配置热重载
	exampleHotReload()

	// 示例6: 远程配置
	exampleRemoteConfig()

	// 示例7: 工厂模式
	exampleFactoryPattern()

	// 示例8: 调试模式
	exampleDebugMode()
}

// 示例1: 基础用法
func exampleBasicUsage() {
	fmt.Println("\n--- 示例1: 基础用法 ---")

	// 创建配置管理器
	cfg, err := config.NewManager(nil) // 使用默认选项
	if err != nil {
		log.Printf("创建配置管理器失败: %v", err)
		return
	}
	defer cfg.Close()

	// 设置一些默认值
	cfg.SetDefault("app.name", "示例应用")
	cfg.SetDefault("app.version", "1.0.0")
	cfg.SetDefault("server.port", 8080)

	// 获取配置值
	appName := cfg.GetString("app.name")
	appVersion := cfg.GetString("app.version")
	serverPort := cfg.GetInt("server.port")

	fmt.Printf("应用名称: %s\n", appName)
	fmt.Printf("应用版本: %s\n", appVersion)
	fmt.Printf("服务器端口: %d\n", serverPort)

	// 动态设置配置
	cfg.Set("runtime.start_time", time.Now().Format(time.RFC3339))
	startTime := cfg.GetString("runtime.start_time")
	fmt.Printf("启动时间: %s\n", startTime)
}

// 示例2: 环境变量覆盖
func exampleEnvironmentOverride() {
	fmt.Println("\n--- 示例2: 环境变量覆盖 ---")

	// 设置环境变量
	os.Setenv("APP_SERVER_PORT", "9090")
	os.Setenv("APP_DEBUG", "true")
	defer func() {
		os.Unsetenv("APP_SERVER_PORT")
		os.Unsetenv("APP_DEBUG")
	}()

	// 创建配置管理器，启用环境变量
	opts := &config.Options{
		Environment:    config.Development,
		ConfigName:     "app",
		ConfigType:     "yaml",
		EnvPrefix:      "APP",
		EnvKeyReplacer: strings.NewReplacer(".", "_"),
		Defaults: map[string]interface{}{
			"server.port": 8080,
			"debug":       false,
		},
		Debug: true,
	}

	cfg, err := config.NewManager(opts)
	if err != nil {
		log.Printf("创建配置管理器失败: %v", err)
		return
	}
	defer cfg.Close()

	// 环境变量会覆盖默认值
	serverPort := cfg.GetInt("server.port")
	debugMode := cfg.GetBool("debug")

	fmt.Printf("服务器端口 (环境变量覆盖): %d\n", serverPort)
	fmt.Printf("调试模式 (环境变量覆盖): %t\n", debugMode)
}

// 示例3: 多种配置格式
func exampleMultipleFormats() {
	fmt.Println("\n--- 示例3: 多种配置格式 ---")

	// 创建临时配置文件
	createTestConfigFiles()
	defer cleanupTestConfigFiles()

	formats := []string{"yaml", "json", "toml"}

	for _, format := range formats {
		fmt.Printf("\n加载 %s 格式配置:\n", strings.ToUpper(format))

		opts := &config.Options{
			ConfigName:  "test",
			ConfigType:  format,
			ConfigPaths: []string{"./testdata"},
			Debug:       true,
		}

		cfg, err := config.NewManager(opts)
		if err != nil {
			log.Printf("创建配置管理器失败: %v", err)
			continue
		}

		// 尝试读取配置文件
		if err := cfg.ReadInConfig(); err != nil {
			log.Printf("读取配置文件失败: %v", err)
		} else {
			fmt.Printf("  应用名称: %s\n", cfg.GetString("app.name"))
			fmt.Printf("  服务器端口: %d\n", cfg.GetInt("server.port"))
			fmt.Printf("  数据库驱动: %s\n", cfg.GetString("database.driver"))
		}

		cfg.Close()
	}
}

// 示例4: 结构体绑定
func exampleStructBinding() {
	fmt.Println("\n--- 示例4: 结构体绑定 ---")

	// 创建配置管理器
	cfg, err := config.NewManager(&config.Options{
		ConfigName: "app",
		ConfigType: "yaml",
		Defaults: map[string]interface{}{
			"server.host":          "localhost",
			"server.port":          8080,
			"server.read_timeout":  "30s",
			"server.write_timeout": "30s",
			"server.tls.enabled":   false,
			"database.driver":      "mysql",
			"database.dsn":         "user:pass@tcp(localhost:3306)/dbname",
			"redis.addr":           "localhost:6379",
			"redis.db":             0,
			"logging.level":        "info",
			"logging.format":       "json",
		},
		Debug: true,
	})
	if err != nil {
		log.Printf("创建配置管理器失败: %v", err)
		return
	}
	defer cfg.Close()

	// 绑定到结构体
	var appConfig AppConfig
	if err := cfg.Unmarshal(&appConfig); err != nil {
		log.Printf("绑定配置到结构体失败: %v", err)
		return
	}

	fmt.Printf("服务器配置:\n")
	fmt.Printf("  地址: %s:%d\n", appConfig.Server.Host, appConfig.Server.Port)
	fmt.Printf("  读取超时: %v\n", appConfig.Server.ReadTimeout)
	fmt.Printf("  写入超时: %v\n", appConfig.Server.WriteTimeout)
	fmt.Printf("  TLS启用: %t\n", appConfig.Server.TLS.Enabled)

	fmt.Printf("数据库配置:\n")
	fmt.Printf("  驱动: %s\n", appConfig.Database.Driver)
	fmt.Printf("  DSN: %s\n", appConfig.Database.DSN)

	fmt.Printf("Redis配置:\n")
	fmt.Printf("  地址: %s\n", appConfig.Redis.Addr)
	fmt.Printf("  数据库: %d\n", appConfig.Redis.DB)

	// 部分绑定
	var serverConfig ServerConfig
	if err := cfg.UnmarshalKey("server", &serverConfig); err != nil {
		log.Printf("绑定服务器配置失败: %v", err)
	} else {
		fmt.Printf("\n部分绑定 - 服务器配置:\n")
		fmt.Printf("  地址: %s:%d\n", serverConfig.Host, serverConfig.Port)
	}
}

// 示例5: 配置热重载
func exampleHotReload() {
	fmt.Println("\n--- 示例5: 配置热重载 ---")

	// 创建临时配置文件
	configFile := "./testdata/hotreload.yaml"
	os.MkdirAll("./testdata", 0755)

	initialConfig := `app:
  name: "热重载测试"
  version: "1.0.0"
server:
  port: 8080
`
	if err := os.WriteFile(configFile, []byte(initialConfig), 0644); err != nil {
		log.Printf("创建配置文件失败: %v", err)
		return
	}
	defer os.RemoveAll("./testdata")

	// 创建配置管理器
	opts := &config.Options{
		ConfigName:  "hotreload",
		ConfigType:  "yaml",
		ConfigPaths: []string{"./testdata"},
		Debug:       true,
	}

	cfg, err := config.NewManager(opts)
	if err != nil {
		log.Printf("创建配置管理器失败: %v", err)
		return
	}
	defer cfg.Close()

	// 读取初始配置
	if err := cfg.ReadInConfig(); err != nil {
		log.Printf("读取配置文件失败: %v", err)
		return
	}

	fmt.Printf("初始配置:\n")
	fmt.Printf("  应用名称: %s\n", cfg.GetString("app.name"))
	fmt.Printf("  应用版本: %s\n", cfg.GetString("app.version"))
	fmt.Printf("  服务器端口: %d\n", cfg.GetInt("server.port"))

	// 注册配置变更回调
	configChanged := make(chan bool, 1)
	cfg.OnConfigChange(func() {
		fmt.Println("\n配置文件已变更，重新加载...")
		fmt.Printf("  应用名称: %s\n", cfg.GetString("app.name"))
		fmt.Printf("  应用版本: %s\n", cfg.GetString("app.version"))
		fmt.Printf("  服务器端口: %d\n", cfg.GetInt("server.port"))
		configChanged <- true
	})

	// 启用配置监听
	cfg.WatchConfig()

	// 模拟配置文件变更
	go func() {
		time.Sleep(1 * time.Second)
		updatedConfig := `app:
  name: "热重载测试 - 已更新"
  version: "2.0.0"
server:
  port: 9090
`
		if err := os.WriteFile(configFile, []byte(updatedConfig), 0644); err != nil {
			log.Printf("更新配置文件失败: %v", err)
		}
	}()

	// 等待配置变更
	select {
	case <-configChanged:
		fmt.Println("配置热重载演示完成")
	case <-time.After(5 * time.Second):
		fmt.Println("配置变更超时")
	}
}

// 示例6: 远程配置
func exampleRemoteConfig() {
	fmt.Println("\n--- 示例6: 远程配置 ---")

	// 创建配置管理器，添加远程配置提供者
	opts := &config.Options{
		ConfigName: "app",
		ConfigType: "yaml",
		RemoteProviders: []config.RemoteProvider{
			{
				Provider: "mock", // 使用模拟提供者进行演示
				Endpoint: "localhost:2379",
				Path:     "/config/app",
			},
		},
		Debug: true,
	}

	cfg, err := config.NewManager(opts)
	if err != nil {
		log.Printf("创建配置管理器失败: %v", err)
		return
	}
	defer cfg.Close()

	// 尝试读取远程配置
	if err := cfg.ReadRemoteConfig(); err != nil {
		fmt.Printf("读取远程配置失败 (这是预期的，因为没有真实的远程配置中心): %v\n", err)
	} else {
		fmt.Println("成功读取远程配置")
		fmt.Printf("  配置内容: %+v\n", cfg.AllSettings())
	}

	// 演示远程配置监听
	if err := cfg.WatchRemoteConfig(); err != nil {
		fmt.Printf("启动远程配置监听失败: %v\n", err)
	} else {
		fmt.Println("远程配置监听已启动")
	}
}

// 示例7: 工厂模式
func exampleFactoryPattern() {
	fmt.Println("\n--- 示例7: 工厂模式 ---")

	// 获取全局工厂
	factory := config.GetGlobalFactory()

	// 为不同环境创建配置实例
	environments := []config.Environment{
		config.Development,
		config.Testing,
		config.Production,
	}

	for _, env := range environments {
		fmt.Printf("\n创建 %s 环境配置:\n", env)

		opts := &config.Options{
			Environment: env,
			ConfigName:  "app",
			ConfigType:  "yaml",
			Defaults: map[string]interface{}{
				"app.name":    fmt.Sprintf("应用-%s", env),
				"server.port": getPortForEnv(env),
				"debug":       env == config.Development,
			},
			Debug: true,
		}

		cfg, err := factory.CreateConfig(env, opts)
		if err != nil {
			log.Printf("创建 %s 环境配置失败: %v", env, err)
			continue
		}

		fmt.Printf("  应用名称: %s\n", cfg.GetString("app.name"))
		fmt.Printf("  服务器端口: %d\n", cfg.GetInt("server.port"))
		fmt.Printf("  调试模式: %t\n", cfg.GetBool("debug"))
	}

	// 列出所有环境
	fmt.Printf("\n已创建的环境: %v\n", factory.ListEnvironments())

	// 获取特定环境的配置
	devConfig, exists := factory.GetConfig(config.Development)
	if exists {
		fmt.Printf("开发环境配置 - 应用名称: %s\n", devConfig.GetString("app.name"))
	}

	// 关闭所有配置
	factory.CloseAll()
}

// 示例8: 调试模式
func exampleDebugMode() {
	fmt.Println("\n--- 示例8: 调试模式 ---")

	// 创建配置管理器，启用调试模式
	opts := &config.Options{
		ConfigName:  "debug-test",
		ConfigType:  "yaml",
		ConfigPaths: []string{".", "./config"},
		Defaults: map[string]interface{}{
			"app.name":    "调试模式测试",
			"server.port": 8080,
			"debug":       true,
		},
		Debug: true, // 启用调试模式
	}

	cfg, err := config.NewManager(opts)
	if err != nil {
		log.Printf("创建配置管理器失败: %v", err)
		return
	}
	defer cfg.Close()

	fmt.Printf("调试模式状态: %t\n", cfg.Debug())

	// 执行一些配置操作，观察调试输出
	cfg.Set("runtime.test", "调试测试值")
	testValue := cfg.GetString("runtime.test")
	fmt.Printf("测试值: %s\n", testValue)

	// 动态切换调试模式
	cfg.SetDebug(false)
	fmt.Printf("\n关闭调试模式后:\n")
	cfg.Set("runtime.test2", "无调试输出")

	cfg.SetDebug(true)
	fmt.Printf("\n重新启用调试模式后:\n")
	cfg.Set("runtime.test3", "有调试输出")
}

// 辅助函数

// getPortForEnv 根据环境获取端口
func getPortForEnv(env config.Environment) int {
	switch env {
	case config.Development:
		return 8080
	case config.Testing:
		return 8081
	case config.Production:
		return 80
	default:
		return 8080
	}
}

// createTestConfigFiles 创建测试配置文件
func createTestConfigFiles() {
	os.MkdirAll("./testdata", 0755)

	// YAML 格式
	yamlContent := `app:
  name: "YAML测试应用"
  version: "1.0.0"
server:
  port: 8080
database:
  driver: "mysql"
  dsn: "user:pass@tcp(localhost:3306)/test"
`
	os.WriteFile("./testdata/test.yaml", []byte(yamlContent), 0644)

	// JSON 格式
	jsonContent := `{
  "app": {
    "name": "JSON测试应用",
    "version": "1.0.0"
  },
  "server": {
    "port": 8080
  },
  "database": {
    "driver": "mysql",
    "dsn": "user:pass@tcp(localhost:3306)/test"
  }
}`
	os.WriteFile("./testdata/test.json", []byte(jsonContent), 0644)

	// TOML 格式
	tomlContent := `[app]
name = "TOML测试应用"
version = "1.0.0"

[server]
port = 8080

[database]
driver = "mysql"
dsn = "user:pass@tcp(localhost:3306)/test"
`
	os.WriteFile("./testdata/test.toml", []byte(tomlContent), 0644)
}

// cleanupTestConfigFiles 清理测试配置文件
func cleanupTestConfigFiles() {
	os.RemoveAll("./testdata")
}
