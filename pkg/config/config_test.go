package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfig 测试配置结构体
type TestConfig struct {
	App struct {
		Name    string `mapstructure:"name"`
		Version string `mapstructure:"version"`
		Debug   bool   `mapstructure:"debug"`
		Port    int    `mapstructure:"port"`
	} `mapstructure:"app"`
	Database struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
		Name     string `mapstructure:"name"`
	} `mapstructure:"database"`
	Redis struct {
		Host     string        `mapstructure:"host"`
		Port     int           `mapstructure:"port"`
		Password string        `mapstructure:"password"`
		Timeout  time.Duration `mapstructure:"timeout"`
	} `mapstructure:"redis"`
}

func TestNewManager(t *testing.T) {
	tests := []struct {
		name string
		opts *Options
		want bool
	}{
		{
			name: "default options",
			opts: nil,
			want: true,
		},
		{
			name: "custom options",
			opts: &Options{
				Environment: Production,
				ConfigName:  "app",
				ConfigType:  "json",
				Debug:       true,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewManager(tt.opts)
			if tt.want {
				assert.NoError(t, err)
				assert.NotNil(t, config)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestManager_BasicOperations(t *testing.T) {
	config, err := NewManager(&Options{
		Environment: Testing,
		Debug:       true,
	})
	require.NoError(t, err)
	defer config.Close()

	// 测试设置和获取
	config.Set("test.string", "hello")
	config.Set("test.int", 42)
	config.Set("test.bool", true)
	config.Set("test.float", 3.14)
	config.Set("test.slice", []string{"a", "b", "c"})
	config.Set("test.duration", "5m")

	// 测试获取不同类型的值
	assert.Equal(t, "hello", config.GetString("test.string"))
	assert.Equal(t, 42, config.GetInt("test.int"))
	assert.Equal(t, int64(42), config.GetInt64("test.int"))
	assert.Equal(t, true, config.GetBool("test.bool"))
	assert.Equal(t, 3.14, config.GetFloat64("test.float"))
	assert.Equal(t, []string{"a", "b", "c"}, config.GetStringSlice("test.slice"))
	assert.Equal(t, 5*time.Minute, config.GetDuration("test.duration"))

	// 测试默认值
	config.SetDefault("default.value", "default")
	assert.Equal(t, "default", config.GetString("default.value"))

	// 测试不存在的键
	assert.Equal(t, "", config.GetString("nonexistent"))
	assert.Equal(t, 0, config.GetInt("nonexistent"))
	assert.Equal(t, false, config.GetBool("nonexistent"))
}

func TestManager_ConfigFile(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.test.yaml")

	configContent := `
app:
  name: "test-app"
  version: "1.0.0"
  debug: true
  port: 8080
database:
  host: "localhost"
  port: 5432
  username: "testuser"
  password: "testpass"
  name: "testdb"
redis:
  host: "localhost"
  port: 6379
  password: ""
  timeout: "30s"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// 创建配置管理器
	config, err := NewManager(&Options{
		Environment: Testing,
		ConfigName:  "config",
		ConfigType:  "yaml",
		ConfigPaths: []string{tempDir},
		Debug:       true,
	})
	require.NoError(t, err)
	defer config.Close()

	// 读取配置文件
	err = config.ReadInConfig()
	require.NoError(t, err)

	// 验证配置值
	assert.Equal(t, "test-app", config.GetString("app.name"))
	assert.Equal(t, "1.0.0", config.GetString("app.version"))
	assert.Equal(t, true, config.GetBool("app.debug"))
	assert.Equal(t, 8080, config.GetInt("app.port"))
	assert.Equal(t, "localhost", config.GetString("database.host"))
	assert.Equal(t, 5432, config.GetInt("database.port"))
	assert.Equal(t, 30*time.Second, config.GetDuration("redis.timeout"))

	// 测试结构体绑定
	var testConfig TestConfig
	err = config.Unmarshal(&testConfig)
	require.NoError(t, err)

	assert.Equal(t, "test-app", testConfig.App.Name)
	assert.Equal(t, "1.0.0", testConfig.App.Version)
	assert.Equal(t, true, testConfig.App.Debug)
	assert.Equal(t, 8080, testConfig.App.Port)
	assert.Equal(t, "localhost", testConfig.Database.Host)
	assert.Equal(t, 5432, testConfig.Database.Port)
	assert.Equal(t, "testuser", testConfig.Database.Username)
	assert.Equal(t, "testpass", testConfig.Database.Password)
	assert.Equal(t, "testdb", testConfig.Database.Name)
	assert.Equal(t, 30*time.Second, testConfig.Redis.Timeout)
}

func TestManager_EnvironmentVariables(t *testing.T) {
	// 设置环境变量
	os.Setenv("TEST_APP_NAME", "env-app")
	os.Setenv("TEST_APP_PORT", "9090")
	os.Setenv("TEST_APP_DEBUG", "false")
	defer func() {
		os.Unsetenv("TEST_APP_NAME")
		os.Unsetenv("TEST_APP_PORT")
		os.Unsetenv("TEST_APP_DEBUG")
	}()

	// 创建配置管理器
	config, err := NewManager(&Options{
		Environment:    Testing,
		EnvPrefix:      "TEST",
		EnvKeyReplacer: strings.NewReplacer(".", "_"),
		Debug:          true,
	})
	require.NoError(t, err)
	defer config.Close()

	// 验证环境变量覆盖
	assert.Equal(t, "env-app", config.GetString("app.name"))
	assert.Equal(t, 9090, config.GetInt("app.port"))
	assert.Equal(t, false, config.GetBool("app.debug"))
}

func TestManager_MultipleFormats(t *testing.T) {
	formats := []struct {
		name    string
		format  string
		content string
	}{
		{
			name:   "yaml",
			format: "yaml",
			content: `
app:
  name: "yaml-app"
  port: 8080
`,
		},
		{
			name:   "json",
			format: "json",
			content: `{
  "app": {
    "name": "json-app",
    "port": 8081
  }
}`,
		},
		{
			name:   "toml",
			format: "toml",
			content: `[app]
name = "toml-app"
port = 8082
`,
		},
	}

	for _, format := range formats {
		t.Run(format.name, func(t *testing.T) {
			// 创建临时配置文件
			tempDir := t.TempDir()
			configFile := filepath.Join(tempDir, "config.test."+format.format)

			err := os.WriteFile(configFile, []byte(format.content), 0644)
			require.NoError(t, err)

			// 创建配置管理器
			config, err := NewManager(&Options{
				Environment: Testing,
				ConfigName:  "config",
				ConfigType:  format.format,
				ConfigPaths: []string{tempDir},
				Debug:       true,
			})
			require.NoError(t, err)
			defer config.Close()

			// 读取配置文件
			err = config.ReadInConfig()
			require.NoError(t, err)

			// 验证配置值
			expectedName := format.name + "-app"
			var expectedPort int
			switch format.name {
			case "yaml":
				expectedPort = 8080
			case "json":
				expectedPort = 8081
			case "toml":
				expectedPort = 8082
			}
			assert.Equal(t, expectedName, config.GetString("app.name"))
			assert.Equal(t, expectedPort, config.GetInt("app.port"))
		})
	}
}

func TestManager_WatchConfig(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.test.yaml")

	initialContent := `
app:
  name: "initial"
  port: 8080
`

	err := os.WriteFile(configFile, []byte(initialContent), 0644)
	require.NoError(t, err)

	// 创建配置管理器
	config, err := NewManager(&Options{
		Environment: Testing,
		ConfigName:  "config",
		ConfigType:  "yaml",
		ConfigPaths: []string{tempDir},
		Debug:       true,
	})
	require.NoError(t, err)
	defer config.Close()

	// 读取初始配置
	err = config.ReadInConfig()
	require.NoError(t, err)
	assert.Equal(t, "initial", config.GetString("app.name"))

	// 设置配置变化回调
	changed := make(chan bool, 1)
	config.OnConfigChange(func() {
		changed <- true
	})

	// 开始监听配置变化
	config.WatchConfig()

	// 修改配置文件
	updatedContent := `
app:
  name: "updated"
  port: 9090
`

	err = os.WriteFile(configFile, []byte(updatedContent), 0644)
	require.NoError(t, err)

	// 等待配置变化通知
	select {
	case <-changed:
		// 验证配置已更新
		assert.Equal(t, "updated", config.GetString("app.name"))
		assert.Equal(t, 9090, config.GetInt("app.port"))
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for config change")
	}
}

func TestManager_Debug(t *testing.T) {
	config, err := NewManager(&Options{
		Environment: Testing,
		Debug:       false,
	})
	require.NoError(t, err)
	defer config.Close()

	// 测试调试模式
	assert.False(t, config.Debug())

	config.SetDebug(true)
	assert.True(t, config.Debug())

	config.SetDebug(false)
	assert.False(t, config.Debug())
}

func TestManager_AllSettings(t *testing.T) {
	config, err := NewManager(&Options{
		Environment: Testing,
		Debug:       true,
	})
	require.NoError(t, err)
	defer config.Close()

	// 设置一些配置值
	config.Set("app.name", "test")
	config.Set("app.port", 8080)
	config.Set("database.host", "localhost")

	// 获取所有设置
	allSettings := config.AllSettings()
	assert.NotEmpty(t, allSettings)
	assert.Equal(t, "test", allSettings["app"].(map[string]interface{})["name"])
	assert.Equal(t, 8080, allSettings["app"].(map[string]interface{})["port"])
	assert.Equal(t, "localhost", allSettings["database"].(map[string]interface{})["host"])
}

func TestManager_UnmarshalKey(t *testing.T) {
	config, err := NewManager(&Options{
		Environment: Testing,
		Debug:       true,
	})
	require.NoError(t, err)
	defer config.Close()

	// 设置应用配置
	config.Set("app.name", "test-app")
	config.Set("app.version", "1.0.0")
	config.Set("app.debug", true)
	config.Set("app.port", 8080)

	// 只绑定app部分到结构体
	var appConfig struct {
		Name    string `mapstructure:"name"`
		Version string `mapstructure:"version"`
		Debug   bool   `mapstructure:"debug"`
		Port    int    `mapstructure:"port"`
	}

	err = config.UnmarshalKey("app", &appConfig)
	require.NoError(t, err)

	assert.Equal(t, "test-app", appConfig.Name)
	assert.Equal(t, "1.0.0", appConfig.Version)
	assert.Equal(t, true, appConfig.Debug)
	assert.Equal(t, 8080, appConfig.Port)
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	assert.NotNil(t, opts)
	assert.Equal(t, Development, opts.Environment)
	assert.Equal(t, "config", opts.ConfigName)
	assert.Equal(t, "yaml", opts.ConfigType)
	assert.NotEmpty(t, opts.ConfigPaths)
	assert.False(t, opts.Debug)
	assert.NotNil(t, opts.Defaults)
	assert.NotNil(t, opts.EnvKeyReplacer)
}

func BenchmarkManager_Get(b *testing.B) {
	config, err := NewManager(&Options{
		Environment: Testing,
		Debug:       false,
	})
	require.NoError(b, err)
	defer config.Close()

	// 设置一些测试数据
	config.Set("test.key", "test.value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.GetString("test.key")
	}
}

func BenchmarkManager_Set(b *testing.B) {
	config, err := NewManager(&Options{
		Environment: Testing,
		Debug:       false,
	})
	require.NoError(b, err)
	defer config.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config.Set("test.key", "test.value")
	}
}
