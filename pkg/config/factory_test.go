package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFactory(t *testing.T) {
	tests := []struct {
		name        string
		defaultOpts *Options
		wantNil     bool
	}{
		{
			name:        "with default options",
			defaultOpts: DefaultOptions(),
			wantNil:     false,
		},
		{
			name:        "with nil options",
			defaultOpts: nil,
			wantNil:     false,
		},
		{
			name: "with custom options",
			defaultOpts: &Options{
				Environment: Production,
				ConfigName:  "app",
				ConfigType:  "json",
				Debug:       true,
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewFactory(tt.defaultOpts)
			if tt.wantNil {
				assert.Nil(t, factory)
			} else {
				assert.NotNil(t, factory)
				assert.NotNil(t, factory.(*ConfigFactory).configs)
				assert.NotNil(t, factory.(*ConfigFactory).defaultOpts)
			}
		})
	}
}

func TestConfigFactory_CreateConfig(t *testing.T) {
	factory := NewFactory(&Options{
		Environment: Development,
		ConfigName:  "test",
		ConfigType:  "yaml",
		Debug:       true,
	})
	defer factory.CloseAll()

	tests := []struct {
		name    string
		env     Environment
		opts    *Options
		wantErr bool
	}{
		{
			name:    "create development config",
			env:     Development,
			opts:    nil,
			wantErr: false,
		},
		{
			name:    "create testing config",
			env:     Testing,
			opts:    nil,
			wantErr: false,
		},
		{
			name: "create production config with custom options",
			env:  Production,
			opts: &Options{
				ConfigName: "prod",
				ConfigType: "json",
				Debug:      false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := factory.CreateConfig(tt.env, tt.opts)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, config)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config)

				// 验证配置实例已被缓存
				cachedConfig, exists := factory.GetConfig(tt.env)
				assert.True(t, exists)
				assert.Equal(t, config, cachedConfig)
			}
		})
	}
}

func TestConfigFactory_GetConfig(t *testing.T) {
	factory := NewFactory(DefaultOptions())
	defer factory.CloseAll()

	// 测试获取不存在的配置
	config, exists := factory.GetConfig(Development)
	assert.False(t, exists)
	assert.Nil(t, config)

	// 创建配置
	createdConfig, err := factory.CreateConfig(Development, nil)
	require.NoError(t, err)

	// 测试获取存在的配置
	config, exists = factory.GetConfig(Development)
	assert.True(t, exists)
	assert.Equal(t, createdConfig, config)
}

func TestConfigFactory_ListEnvironments(t *testing.T) {
	factory := NewFactory(DefaultOptions())
	defer factory.CloseAll()

	// 初始状态应该没有环境
	envs := factory.ListEnvironments()
	assert.Empty(t, envs)

	// 创建多个环境的配置
	_, err := factory.CreateConfig(Development, nil)
	require.NoError(t, err)

	_, err = factory.CreateConfig(Testing, nil)
	require.NoError(t, err)

	_, err = factory.CreateConfig(Production, nil)
	require.NoError(t, err)

	// 验证环境列表
	envs = factory.ListEnvironments()
	assert.Len(t, envs, 3)
	assert.Contains(t, envs, Development)
	assert.Contains(t, envs, Testing)
	assert.Contains(t, envs, Production)
}

func TestConfigFactory_CloseAll(t *testing.T) {
	factory := NewFactory(DefaultOptions())

	// 创建多个配置
	_, err := factory.CreateConfig(Development, nil)
	require.NoError(t, err)

	_, err = factory.CreateConfig(Testing, nil)
	require.NoError(t, err)

	// 验证配置存在
	envs := factory.ListEnvironments()
	assert.Len(t, envs, 2)

	// 关闭所有配置
	err = factory.CloseAll()
	assert.NoError(t, err)

	// 验证配置已被清空
	envs = factory.ListEnvironments()
	assert.Empty(t, envs)

	// 验证无法获取已关闭的配置
	_, exists := factory.GetConfig(Development)
	assert.False(t, exists)
}

func TestConfigFactory_MergeOptions(t *testing.T) {
	defaultOpts := &Options{
		Environment: Development,
		ConfigName:  "default",
		ConfigType:  "yaml",
		ConfigPaths: []string{"./config"},
		Debug:       false,
		Defaults: map[string]interface{}{
			"default.key": "default.value",
		},
		EnvPrefix: "DEFAULT",
	}

	factory := NewFactory(defaultOpts).(*ConfigFactory)

	tests := []struct {
		name     string
		env      Environment
		opts     *Options
		expected *Options
	}{
		{
			name: "nil options should use defaults",
			env:  Testing,
			opts: nil,
			expected: &Options{
				Environment: Testing,
				ConfigName:  "default",
				ConfigType:  "yaml",
				ConfigPaths: []string{"./config"},
				Debug:       false,
				Defaults: map[string]interface{}{
					"default.key": "default.value",
				},
				EnvPrefix: "DEFAULT",
			},
		},
		{
			name: "custom options should override defaults",
			env:  Production,
			opts: &Options{
				ConfigName: "custom",
				ConfigType: "json",
				Debug:      true,
				Defaults: map[string]interface{}{
					"custom.key": "custom.value",
				},
				EnvPrefix: "CUSTOM",
			},
			expected: &Options{
				Environment: Production,
				ConfigName:  "custom",
				ConfigType:  "json",
				ConfigPaths: []string{"./config"},
				Debug:       true,
				Defaults: map[string]interface{}{
					"default.key": "default.value",
					"custom.key":  "custom.value",
				},
				EnvPrefix: "CUSTOM",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := factory.mergeOptions(tt.env, tt.opts)
			assert.Equal(t, tt.expected.Environment, result.Environment)
			assert.Equal(t, tt.expected.ConfigName, result.ConfigName)
			assert.Equal(t, tt.expected.ConfigType, result.ConfigType)
			assert.Equal(t, tt.expected.ConfigPaths, result.ConfigPaths)
			assert.Equal(t, tt.expected.Debug, result.Debug)
			assert.Equal(t, tt.expected.EnvPrefix, result.EnvPrefix)

			// 验证默认值合并
			for key, expectedValue := range tt.expected.Defaults {
				actualValue, exists := result.Defaults[key]
				assert.True(t, exists, "Expected default key %s not found", key)
				assert.Equal(t, expectedValue, actualValue, "Default value mismatch for key %s", key)
			}
		})
	}
}

func TestGlobalFactory(t *testing.T) {
	// 测试全局工厂单例
	factory1 := GetGlobalFactory()
	factory2 := GetGlobalFactory()
	assert.Equal(t, factory1, factory2)

	// 测试全局配置创建
	config, err := CreateGlobalConfig(Testing, &Options{
		Debug: true,
	})
	require.NoError(t, err)
	assert.NotNil(t, config)

	// 测试全局配置获取
	cachedConfig, exists := GetGlobalConfig(Testing)
	assert.True(t, exists)
	assert.Equal(t, config, cachedConfig)

	// 测试全局配置关闭
	err = CloseGlobalConfigs()
	assert.NoError(t, err)

	// 验证配置已被清空
	_, exists = GetGlobalConfig(Testing)
	assert.False(t, exists)
}

func TestConfigFactory_CreateConfigTwice(t *testing.T) {
	factory := NewFactory(DefaultOptions())
	defer factory.CloseAll()

	// 第一次创建
	config1, err := factory.CreateConfig(Development, nil)
	require.NoError(t, err)
	assert.NotNil(t, config1)

	// 第二次创建相同环境的配置，应该返回相同实例
	config2, err := factory.CreateConfig(Development, nil)
	require.NoError(t, err)
	assert.Equal(t, config1, config2)
}

func BenchmarkConfigFactory_CreateConfig(b *testing.B) {
	factory := NewFactory(DefaultOptions())
	defer factory.CloseAll()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = factory.CreateConfig(Development, nil)
	}
}

func BenchmarkConfigFactory_GetConfig(b *testing.B) {
	factory := NewFactory(DefaultOptions())
	defer factory.CloseAll()

	// 预先创建配置
	_, err := factory.CreateConfig(Development, nil)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = factory.GetConfig(Development)
	}
}
