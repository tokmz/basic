package config

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ValidationRule 验证规则接口
type ValidationRule interface {
	Validate(value interface{}) error
	GetName() string
}

// RequiredRule 必填验证规则
type RequiredRule struct {
	Name string
}

func (r *RequiredRule) Validate(value interface{}) error {
	if value == nil {
		return fmt.Errorf("field %s is required", r.Name)
	}

	// 检查字符串是否为空
	if str, ok := value.(string); ok && strings.TrimSpace(str) == "" {
		return fmt.Errorf("field %s cannot be empty", r.Name)
	}

	// 检查切片是否为空
	if reflect.TypeOf(value).Kind() == reflect.Slice {
		v := reflect.ValueOf(value)
		if v.Len() == 0 {
			return fmt.Errorf("field %s cannot be empty", r.Name)
		}
	}

	return nil
}

func (r *RequiredRule) GetName() string {
	return r.Name
}

// RangeRule 范围验证规则
type RangeRule struct {
	Name string
	Min  interface{}
	Max  interface{}
}

func (r *RangeRule) Validate(value interface{}) error {
	if value == nil {
		return nil // 范围验证不检查nil值
	}

	switch v := value.(type) {
	case int:
		min, _ := r.Min.(int)
		max, _ := r.Max.(int)
		if v < min || v > max {
			return fmt.Errorf("field %s must be between %d and %d, got %d", r.Name, min, max, v)
		}
	case int64:
		min, _ := r.Min.(int64)
		max, _ := r.Max.(int64)
		if v < min || v > max {
			return fmt.Errorf("field %s must be between %d and %d, got %d", r.Name, min, max, v)
		}
	case float64:
		min, _ := r.Min.(float64)
		max, _ := r.Max.(float64)
		if v < min || v > max {
			return fmt.Errorf("field %s must be between %f and %f, got %f", r.Name, min, max, v)
		}
	case string:
		min, _ := r.Min.(int)
		max, _ := r.Max.(int)
		if len(v) < min || len(v) > max {
			return fmt.Errorf("field %s length must be between %d and %d, got %d", r.Name, min, max, len(v))
		}
	default:
		return fmt.Errorf("range validation not supported for type %T", value)
	}

	return nil
}

func (r *RangeRule) GetName() string {
	return r.Name
}

// RegexRule 正则表达式验证规则
type RegexRule struct {
	Name    string
	Pattern string
	regex   *regexp.Regexp
}

func NewRegexRule(name, pattern string) (*RegexRule, error) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	return &RegexRule{
		Name:    name,
		Pattern: pattern,
		regex:   regex,
	}, nil
}

func (r *RegexRule) Validate(value interface{}) error {
	if value == nil {
		return nil // 正则验证不检查nil值
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("regex validation only supports string values, got %T", value)
	}

	if !r.regex.MatchString(str) {
		return fmt.Errorf("field %s does not match pattern %s", r.Name, r.Pattern)
	}

	return nil
}

func (r *RegexRule) GetName() string {
	return r.Name
}

// EnumRule 枚举验证规则
type EnumRule struct {
	Name   string
	Values []interface{}
}

func (r *EnumRule) Validate(value interface{}) error {
	if value == nil {
		return nil // 枚举验证不检查nil值
	}

	for _, validValue := range r.Values {
		if reflect.DeepEqual(value, validValue) {
			return nil
		}
	}

	return fmt.Errorf("field %s must be one of %v, got %v", r.Name, r.Values, value)
}

func (r *EnumRule) GetName() string {
	return r.Name
}

// Validator 配置验证器
type Validator struct {
	rules map[string][]ValidationRule
	debug bool
}

// NewValidator 创建新的验证器
func NewValidator(debug bool) *Validator {
	return &Validator{
		rules: make(map[string][]ValidationRule),
		debug: debug,
	}
}

// AddRule 添加验证规则
func (v *Validator) AddRule(key string, rule ValidationRule) {
	if v.rules[key] == nil {
		v.rules[key] = make([]ValidationRule, 0)
	}
	v.rules[key] = append(v.rules[key], rule)

	if v.debug {
		fmt.Printf("[VALIDATOR] Added rule %s for key %s\n", rule.GetName(), key)
	}
}

// Validate 验证配置
func (v *Validator) Validate(config Config) error {
	if v.debug {
		fmt.Printf("[VALIDATOR] Starting validation...\n")
	}

	allSettings := config.AllSettings()
	errorCount := 0

	for key, rules := range v.rules {
		value := config.Get(key)

		for _, rule := range rules {
			if err := rule.Validate(value); err != nil {
				errorCount++
				if v.debug {
					fmt.Printf("[VALIDATOR] Validation failed for key %s: %v\n", key, err)
				}
				return err
			}
		}

		if v.debug {
			fmt.Printf("[VALIDATOR] Key %s passed validation\n", key)
		}
	}

	// 检查是否有未验证的配置项
	for key := range allSettings {
		if _, hasRules := v.rules[key]; !hasRules {
			if v.debug {
				fmt.Printf("[VALIDATOR] Warning: No validation rules for key %s\n", key)
			}
		}
	}

	if v.debug {
		fmt.Printf("[VALIDATOR] Validation completed with %d errors\n", errorCount)
	}

	return nil
}

// ValidateStruct 验证结构体
func (v *Validator) ValidateStruct(structPtr interface{}) error {
	if structPtr == nil {
		return fmt.Errorf("struct pointer cannot be nil")
	}

	val := reflect.ValueOf(structPtr)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("expected pointer to struct, got %T", structPtr)
	}

	val = val.Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 获取配置键名
		configKey := v.getConfigKey(fieldType)
		if configKey == "" {
			continue
		}

		// 验证字段
		if rules, exists := v.rules[configKey]; exists {
			fieldValue := field.Interface()
			for _, rule := range rules {
				if err := rule.Validate(fieldValue); err != nil {
					return fmt.Errorf("struct field %s: %w", fieldType.Name, err)
				}
			}
		}
	}

	return nil
}

// getConfigKey 获取结构体字段对应的配置键名
func (v *Validator) getConfigKey(field reflect.StructField) string {
	// 优先使用mapstructure标签
	if tag := field.Tag.Get("mapstructure"); tag != "" && tag != "-" {
		return tag
	}

	// 其次使用json标签
	if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
		return strings.Split(tag, ",")[0]
	}

	// 最后使用字段名的小写形式
	return strings.ToLower(field.Name)
}

// ConfigUtils 配置工具类
type ConfigUtils struct{}

// NewConfigUtils 创建配置工具实例
func NewConfigUtils() *ConfigUtils {
	return &ConfigUtils{}
}

// ParseDuration 解析时间间隔字符串
func (cu *ConfigUtils) ParseDuration(value string) (time.Duration, error) {
	if value == "" {
		return 0, fmt.Errorf("duration string cannot be empty")
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid duration format: %w", err)
	}

	return duration, nil
}

// ParseSize 解析大小字符串（如 "1MB", "512KB"）
func (cu *ConfigUtils) ParseSize(value string) (int64, error) {
	if value == "" {
		return 0, fmt.Errorf("size string cannot be empty")
	}

	// 去除空格并转换为大写
	value = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(value), " ", ""))

	// 定义单位映射（按长度排序，优先匹配长单位）
	units := []struct {
		suffix     string
		multiplier int64
	}{
		{"TB", 1024 * 1024 * 1024 * 1024},
		{"GB", 1024 * 1024 * 1024},
		{"MB", 1024 * 1024},
		{"KB", 1024},
		{"B", 1},
	}

	// 查找单位
	for _, unit := range units {
		if strings.HasSuffix(value, unit.suffix) {
			numStr := strings.TrimSuffix(value, unit.suffix)
			if numStr == "" {
				continue // 避免空字符串
			}
			num, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid size format: %w", err)
			}
			return int64(num * float64(unit.multiplier)), nil
		}
	}

	// 如果没有单位，假设是字节
	num, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size format: %w", err)
	}

	return num, nil
}

// FormatSize 格式化大小为可读字符串
func (cu *ConfigUtils) FormatSize(size int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	floatSize := float64(size)

	for i, unit := range units {
		if floatSize < 1024 || i == len(units)-1 {
			if i == 0 {
				return fmt.Sprintf("%.0f%s", floatSize, unit)
			}
			return fmt.Sprintf("%.2f%s", floatSize, unit)
		}
		floatSize /= 1024
	}

	return fmt.Sprintf("%.2f%s", floatSize, units[len(units)-1])
}

// MergeConfigs 合并多个配置
func (cu *ConfigUtils) MergeConfigs(configs ...Config) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for i, config := range configs {
		if config == nil {
			continue
		}

		settings := config.AllSettings()
		cu.mergeMap(result, settings, i)
	}

	return result, nil
}

// mergeMap 递归合并map
func (cu *ConfigUtils) mergeMap(dst, src map[string]interface{}, configIndex int) {
	for key, srcValue := range src {
		if dstValue, exists := dst[key]; exists {
			// 如果两个值都是map，递归合并
			if dstMap, dstOk := dstValue.(map[string]interface{}); dstOk {
				if srcMap, srcOk := srcValue.(map[string]interface{}); srcOk {
					cu.mergeMap(dstMap, srcMap, configIndex)
					continue
				}
			}
			// 如果值不同，使用后面配置的值
			if !reflect.DeepEqual(dstValue, srcValue) {
				fmt.Printf("[MERGE] Key %s overridden by config %d: %v -> %v\n", key, configIndex, dstValue, srcValue)
			}
		}
		dst[key] = srcValue
	}
}

// DiffConfigs 比较两个配置的差异
func (cu *ConfigUtils) DiffConfigs(config1, config2 Config) map[string]interface{} {
	diff := make(map[string]interface{})

	if config1 == nil || config2 == nil {
		return diff
	}

	settings1 := config1.AllSettings()
	settings2 := config2.AllSettings()

	// 检查config1中的键
	for key, value1 := range settings1 {
		if value2, exists := settings2[key]; exists {
			if !reflect.DeepEqual(value1, value2) {
				diff[key] = map[string]interface{}{
					"old": value1,
					"new": value2,
				}
			}
		} else {
			diff[key] = map[string]interface{}{
				"old":     value1,
				"new":     nil,
				"deleted": true,
			}
		}
	}

	// 检查config2中新增的键
	for key, value2 := range settings2 {
		if _, exists := settings1[key]; !exists {
			diff[key] = map[string]interface{}{
				"old":   nil,
				"new":   value2,
				"added": true,
			}
		}
	}

	return diff
}
