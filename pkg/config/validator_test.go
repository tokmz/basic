package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequiredRule(t *testing.T) {
	rule := &RequiredRule{Name: "test"}

	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{"nil value", nil, true},
		{"empty string", "", true},
		{"whitespace string", "   ", true},
		{"valid string", "hello", false},
		{"empty slice", []string{}, true},
		{"valid slice", []string{"item"}, false},
		{"zero int", 0, false},
		{"valid int", 42, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

	assert.Equal(t, "test", rule.GetName())
}

func TestRangeRule(t *testing.T) {
	tests := []struct {
		name    string
		rule    *RangeRule
		value   interface{}
		wantErr bool
	}{
		{
			name:    "int in range",
			rule:    &RangeRule{Name: "test", Min: 1, Max: 10},
			value:   5,
			wantErr: false,
		},
		{
			name:    "int below range",
			rule:    &RangeRule{Name: "test", Min: 1, Max: 10},
			value:   0,
			wantErr: true,
		},
		{
			name:    "int above range",
			rule:    &RangeRule{Name: "test", Min: 1, Max: 10},
			value:   11,
			wantErr: true,
		},
		{
			name:    "int64 in range",
			rule:    &RangeRule{Name: "test", Min: int64(1), Max: int64(10)},
			value:   int64(5),
			wantErr: false,
		},
		{
			name:    "float64 in range",
			rule:    &RangeRule{Name: "test", Min: 1.0, Max: 10.0},
			value:   5.5,
			wantErr: false,
		},
		{
			name:    "string length in range",
			rule:    &RangeRule{Name: "test", Min: 3, Max: 10},
			value:   "hello",
			wantErr: false,
		},
		{
			name:    "string length below range",
			rule:    &RangeRule{Name: "test", Min: 3, Max: 10},
			value:   "hi",
			wantErr: true,
		},
		{
			name:    "nil value",
			rule:    &RangeRule{Name: "test", Min: 1, Max: 10},
			value:   nil,
			wantErr: false,
		},
		{
			name:    "unsupported type",
			rule:    &RangeRule{Name: "test", Min: 1, Max: 10},
			value:   true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.Validate(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRegexRule(t *testing.T) {
	// 测试创建正则规则
	rule, err := NewRegexRule("email", `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	require.NoError(t, err)
	assert.Equal(t, "email", rule.GetName())

	// 测试无效正则表达式
	_, err = NewRegexRule("invalid", "[")
	assert.Error(t, err)

	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{"valid email", "test@example.com", false},
		{"invalid email", "invalid-email", true},
		{"nil value", nil, false},
		{"non-string value", 123, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEnumRule(t *testing.T) {
	rule := &EnumRule{
		Name:   "environment",
		Values: []interface{}{"dev", "test", "prod"},
	}

	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{"valid enum value", "dev", false},
		{"another valid enum value", "prod", false},
		{"invalid enum value", "staging", true},
		{"nil value", nil, false},
		{"wrong type", 123, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Validate(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

	assert.Equal(t, "environment", rule.GetName())
}

func TestValidator(t *testing.T) {
	validator := NewValidator(true)
	assert.NotNil(t, validator)
	assert.True(t, validator.debug)

	// 添加验证规则
	validator.AddRule("app.name", &RequiredRule{Name: "app.name"})
	validator.AddRule("app.port", &RangeRule{Name: "app.port", Min: 1000, Max: 65535})

	emailRule, err := NewRegexRule("app.email", `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	require.NoError(t, err)
	validator.AddRule("app.email", emailRule)

	validator.AddRule("app.env", &EnumRule{
		Name:   "app.env",
		Values: []interface{}{"dev", "test", "prod"},
	})

	// 创建测试配置
	config, err := NewManager(&Options{
		Environment: Testing,
		Debug:       true,
	})
	require.NoError(t, err)
	defer config.Close()

	// 设置有效配置
	config.Set("app.name", "test-app")
	config.Set("app.port", 8080)
	config.Set("app.email", "test@example.com")
	config.Set("app.env", "test")

	// 验证应该通过
	err = validator.Validate(config)
	assert.NoError(t, err)

	// 测试验证失败的情况
	config.Set("app.name", "") // 空字符串应该失败
	err = validator.Validate(config)
	assert.Error(t, err)

	config.Set("app.name", "test-app") // 恢复有效值
	config.Set("app.port", 99999)      // 超出范围应该失败
	err = validator.Validate(config)
	assert.Error(t, err)

	config.Set("app.port", 8080)             // 恢复有效值
	config.Set("app.email", "invalid-email") // 无效邮箱应该失败
	err = validator.Validate(config)
	assert.Error(t, err)

	config.Set("app.email", "test@example.com") // 恢复有效值
	config.Set("app.env", "staging")            // 无效枚举值应该失败
	err = validator.Validate(config)
	assert.Error(t, err)
}

func TestValidator_ValidateStruct(t *testing.T) {
	type TestStruct struct {
		Name  string `mapstructure:"name"`
		Port  int    `mapstructure:"port"`
		Email string `json:"email"`
		Debug bool   // 使用字段名小写
	}

	validator := NewValidator(true)

	// 添加验证规则
	validator.AddRule("name", &RequiredRule{Name: "name"})
	validator.AddRule("port", &RangeRule{Name: "port", Min: 1000, Max: 65535})

	emailRule, err := NewRegexRule("email", `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	require.NoError(t, err)
	validator.AddRule("email", emailRule)

	// 测试有效结构体
	validStruct := &TestStruct{
		Name:  "test-app",
		Port:  8080,
		Email: "test@example.com",
		Debug: true,
	}

	err = validator.ValidateStruct(validStruct)
	assert.NoError(t, err)

	// 测试无效结构体
	invalidStruct := &TestStruct{
		Name:  "", // 空名称应该失败
		Port:  8080,
		Email: "test@example.com",
		Debug: true,
	}

	err = validator.ValidateStruct(invalidStruct)
	assert.Error(t, err)

	// 测试nil指针
	err = validator.ValidateStruct(nil)
	assert.Error(t, err)

	// 测试非指针
	err = validator.ValidateStruct(*validStruct)
	assert.Error(t, err)

	// 测试非结构体指针
	var str string
	err = validator.ValidateStruct(&str)
	assert.Error(t, err)
}

func TestConfigUtils_ParseDuration(t *testing.T) {
	utils := NewConfigUtils()

	tests := []struct {
		name     string
		value    string
		expected time.Duration
		wantErr  bool
	}{
		{"valid duration", "5m", 5 * time.Minute, false},
		{"valid duration with seconds", "30s", 30 * time.Second, false},
		{"valid duration with hours", "2h", 2 * time.Hour, false},
		{"empty string", "", 0, true},
		{"invalid format", "invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := utils.ParseDuration(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestConfigUtils_ParseSize(t *testing.T) {
	utils := NewConfigUtils()

	tests := []struct {
		name     string
		value    string
		expected int64
		wantErr  bool
	}{
		{"bytes", "1024", 1024, false},
		{"kilobytes", "1KB", 1024, false},
		{"megabytes", "1MB", 1024 * 1024, false},
		{"gigabytes", "1GB", 1024 * 1024 * 1024, false},
		{"terabytes", "1TB", 1024 * 1024 * 1024 * 1024, false},
		{"decimal", "1.5MB", int64(1.5 * 1024 * 1024), false},
		{"lowercase", "1mb", 1024 * 1024, false},
		{"with spaces", " 1 MB ", 1024 * 1024, false},
		{"empty string", "", 0, true},
		{"invalid format", "invalid", 0, true},
		{"invalid number", "abcMB", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := utils.ParseSize(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestConfigUtils_FormatSize(t *testing.T) {
	utils := NewConfigUtils()

	tests := []struct {
		name     string
		size     int64
		expected string
	}{
		{"bytes", 512, "512B"},
		{"kilobytes", 1024, "1.00KB"},
		{"megabytes", 1024 * 1024, "1.00MB"},
		{"gigabytes", 1024 * 1024 * 1024, "1.00GB"},
		{"terabytes", 1024 * 1024 * 1024 * 1024, "1.00TB"},
		{"decimal", int64(1.5 * 1024 * 1024), "1.50MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.FormatSize(tt.size)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigUtils_MergeConfigs(t *testing.T) {
	utils := NewConfigUtils()

	// 创建两个配置
	config1, err := NewManager(&Options{Environment: Testing, Debug: true})
	require.NoError(t, err)
	defer config1.Close()

	config2, err := NewManager(&Options{Environment: Testing, Debug: true})
	require.NoError(t, err)
	defer config2.Close()

	// 设置不同的配置值
	config1.Set("app.name", "app1")
	config1.Set("app.port", 8080)
	config1.Set("database.host", "localhost")

	config2.Set("app.name", "app2")     // 覆盖
	config2.Set("app.version", "1.0.0") // 新增
	config2.Set("database.port", 5432)  // 新增

	// 合并配置
	merged, err := utils.MergeConfigs(config1, config2)
	require.NoError(t, err)

	// 验证合并结果
	assert.Equal(t, "app2", merged["app"].(map[string]interface{})["name"])           // 被覆盖
	assert.Equal(t, 8080, merged["app"].(map[string]interface{})["port"])             // 保留
	assert.Equal(t, "1.0.0", merged["app"].(map[string]interface{})["version"])       // 新增
	assert.Equal(t, "localhost", merged["database"].(map[string]interface{})["host"]) // 保留
	assert.Equal(t, 5432, merged["database"].(map[string]interface{})["port"])        // 新增

	// 测试nil配置
	merged, err = utils.MergeConfigs(config1, nil, config2)
	require.NoError(t, err)
	assert.NotEmpty(t, merged)
}

func TestConfigUtils_DiffConfigs(t *testing.T) {
	utils := NewConfigUtils()

	// 创建两个配置
	config1, err := NewManager(&Options{Environment: Testing, Debug: true})
	require.NoError(t, err)
	defer config1.Close()

	config2, err := NewManager(&Options{Environment: Testing, Debug: true})
	require.NoError(t, err)
	defer config2.Close()

	// 设置配置值
	config1.Set("app.name", "app1")
	config1.Set("app.port", 8080)
	config1.Set("database.host", "localhost")

	config2.Set("app.name", "app2")     // 修改
	config2.Set("app.port", 8080)       // 相同
	config2.Set("app.version", "1.0.0") // 新增
	// database.host 被删除

	// 比较配置
	diff := utils.DiffConfigs(config1, config2)

	// 验证差异
	assert.Contains(t, diff, "app")
	appDiff := diff["app"].(map[string]interface{})
	assert.Contains(t, appDiff, "old")
	assert.Contains(t, appDiff, "new")

	// 测试nil配置
	diff = utils.DiffConfigs(nil, config2)
	assert.Empty(t, diff)

	diff = utils.DiffConfigs(config1, nil)
	assert.Empty(t, diff)
}

func BenchmarkValidator_Validate(b *testing.B) {
	validator := NewValidator(false)
	validator.AddRule("app.name", &RequiredRule{Name: "app.name"})
	validator.AddRule("app.port", &RangeRule{Name: "app.port", Min: 1000, Max: 65535})

	config, err := NewManager(&Options{Environment: Testing, Debug: false})
	require.NoError(b, err)
	defer config.Close()

	config.Set("app.name", "test-app")
	config.Set("app.port", 8080)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate(config)
	}
}
