package scheduler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// generateTaskID 生成任务ID
func generateTaskID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// generateExecutionID 生成执行记录ID
func generateExecutionID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("exec_%s", hex.EncodeToString(bytes))
}

// CronParser 简化的Cron表达式解析器
type CronParser struct{}

// CronField Cron字段
type CronField struct {
	Min, Max int
	Values   []int
	All      bool
}

// CronSpec Cron规范
type CronSpec struct {
	Minute CronField
	Hour   CronField
	Day    CronField
	Month  CronField
	Week   CronField
}

// parseCronExpr 解析Cron表达式
// 支持标准的5字段格式: 分 时 日 月 周
// 例如: "0 9 * * 1-5" 表示工作日每天9点
func parseCronExpr(expr string) (*CronSpec, error) {
	fields := strings.Fields(strings.TrimSpace(expr))
	if len(fields) != 5 {
		return nil, fmt.Errorf("cron expression must have exactly 5 fields")
	}

	spec := &CronSpec{}
	var err error

	// 解析分钟 (0-59)
	spec.Minute, err = parseCronField(fields[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("invalid minute field: %v", err)
	}

	// 解析小时 (0-23)
	spec.Hour, err = parseCronField(fields[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("invalid hour field: %v", err)
	}

	// 解析日 (1-31)
	spec.Day, err = parseCronField(fields[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("invalid day field: %v", err)
	}

	// 解析月 (1-12)
	spec.Month, err = parseCronField(fields[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("invalid month field: %v", err)
	}

	// 解析星期 (0-6, 0=Sunday)
	spec.Week, err = parseCronField(fields[4], 0, 6)
	if err != nil {
		return nil, fmt.Errorf("invalid week field: %v", err)
	}

	return spec, nil
}

// parseCronField 解析单个Cron字段
func parseCronField(field string, min, max int) (CronField, error) {
	cf := CronField{Min: min, Max: max}

	if field == "*" {
		cf.All = true
		return cf, nil
	}

	// 处理逗号分隔的值
	if strings.Contains(field, ",") {
		parts := strings.Split(field, ",")
		for _, part := range parts {
			values, err := parseFieldPart(strings.TrimSpace(part), min, max)
			if err != nil {
				return cf, err
			}
			cf.Values = append(cf.Values, values...)
		}
		return cf, nil
	}

	// 处理单个字段部分
	values, err := parseFieldPart(field, min, max)
	if err != nil {
		return cf, err
	}
	cf.Values = values

	return cf, nil
}

// parseFieldPart 解析字段部分
func parseFieldPart(part string, min, max int) ([]int, error) {
	// 处理范围 (例如: 1-5)
	if strings.Contains(part, "-") {
		rangeParts := strings.Split(part, "-")
		if len(rangeParts) != 2 {
			return nil, fmt.Errorf("invalid range format: %s", part)
		}

		start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
		if err != nil {
			return nil, fmt.Errorf("invalid range start: %s", rangeParts[0])
		}

		end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
		if err != nil {
			return nil, fmt.Errorf("invalid range end: %s", rangeParts[1])
		}

		if start < min || start > max || end < min || end > max {
			return nil, fmt.Errorf("range values out of bounds: %d-%d", start, end)
		}

		if start > end {
			return nil, fmt.Errorf("invalid range: start > end")
		}

		var values []int
		for i := start; i <= end; i++ {
			values = append(values, i)
		}
		return values, nil
	}

	// 处理步长 (例如: */5, 0-23/2)
	if strings.Contains(part, "/") {
		stepParts := strings.Split(part, "/")
		if len(stepParts) != 2 {
			return nil, fmt.Errorf("invalid step format: %s", part)
		}

		step, err := strconv.Atoi(strings.TrimSpace(stepParts[1]))
		if err != nil {
			return nil, fmt.Errorf("invalid step value: %s", stepParts[1])
		}

		if step <= 0 {
			return nil, fmt.Errorf("step must be positive: %d", step)
		}

		base := strings.TrimSpace(stepParts[0])
		var start, end int

		if base == "*" {
			start, end = min, max
		} else if strings.Contains(base, "-") {
			baseValues, err := parseFieldPart(base, min, max)
			if err != nil {
				return nil, err
			}
			if len(baseValues) == 0 {
				return nil, fmt.Errorf("empty base range")
			}
			start, end = baseValues[0], baseValues[len(baseValues)-1]
		} else {
			start, err = strconv.Atoi(base)
			if err != nil {
				return nil, fmt.Errorf("invalid base value: %s", base)
			}
			end = max
		}

		var values []int
		for i := start; i <= end; i += step {
			if i >= min && i <= max {
				values = append(values, i)
			}
		}
		return values, nil
	}

	// 处理单个数值
	value, err := strconv.Atoi(part)
	if err != nil {
		return nil, fmt.Errorf("invalid number: %s", part)
	}

	if value < min || value > max {
		return nil, fmt.Errorf("value out of bounds: %d", value)
	}

	return []int{value}, nil
}

// getNextCronTime 根据Cron表达式获取下次执行时间
func getNextCronTime(cronExpr string, from time.Time) (time.Time, error) {
	spec, err := parseCronExpr(cronExpr)
	if err != nil {
		return time.Time{}, NewCronError(cronExpr, err)
	}

	// 从下一分钟开始查找
	next := from.Add(time.Minute).Truncate(time.Minute)

	// 最多查找4年的时间
	maxTime := from.AddDate(4, 0, 0)

	for next.Before(maxTime) {
		if spec.matchTime(next) {
			return next, nil
		}
		next = next.Add(time.Minute)
	}

	return time.Time{}, NewCronError(cronExpr, fmt.Errorf("no matching time found in next 4 years"))
}

// matchTime 检查时间是否匹配Cron规范
func (spec *CronSpec) matchTime(t time.Time) bool {
	return spec.matchField(spec.Minute, t.Minute()) &&
		spec.matchField(spec.Hour, t.Hour()) &&
		spec.matchField(spec.Day, t.Day()) &&
		spec.matchField(spec.Month, int(t.Month())) &&
		spec.matchField(spec.Week, int(t.Weekday()))
}

// matchField 检查值是否匹配字段
func (spec *CronSpec) matchField(field CronField, value int) bool {
	if field.All {
		return true
	}

	for _, v := range field.Values {
		if v == value {
			return true
		}
	}

	return false
}

// validateCronExpr 验证Cron表达式
func validateCronExpr(expr string) error {
	_, err := parseCronExpr(expr)
	return err
}

// formatDuration 格式化时间间隔
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
	return fmt.Sprintf("%.1fd", d.Hours()/24)
}

// isValidTaskName 验证任务名称
func isValidTaskName(name string) bool {
	if name == "" {
		return false
	}

	// 任务名称只能包含字母、数字、下划线和连字符
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, name)
	return matched && len(name) <= 100
}

// sanitizeTaskName 清理任务名称
func sanitizeTaskName(name string) string {
	// 移除非法字符
	reg := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	sanitized := reg.ReplaceAllString(name, "_")

	// 限制长度
	if len(sanitized) > 100 {
		sanitized = sanitized[:100]
	}

	return sanitized
}

// TimeWindow 时间窗口
type TimeWindow struct {
	Start time.Time
	End   time.Time
}

// Contains 检查时间是否在窗口内
func (tw TimeWindow) Contains(t time.Time) bool {
	return !t.Before(tw.Start) && !t.After(tw.End)
}

// Duration 获取窗口持续时间
func (tw TimeWindow) Duration() time.Duration {
	return tw.End.Sub(tw.Start)
}

// IsValid 检查时间窗口是否有效
func (tw TimeWindow) IsValid() bool {
	return tw.Start.Before(tw.End)
}

// TaskFilter 任务过滤器
type TaskFilter struct {
	Names     []string              `json:"names,omitempty"`     // 任务名称
	Types     []TaskType            `json:"types,omitempty"`     // 任务类型
	Statuses  []TaskStatus          `json:"statuses,omitempty"`  // 任务状态
	Tags      map[string]string     `json:"tags,omitempty"`      // 标签匹配
	TimeRange *TimeWindow           `json:"time_range,omitempty"` // 时间范围
}

// Match 检查任务是否匹配过滤器
func (tf *TaskFilter) Match(task *Task) bool {
	// 检查名称
	if len(tf.Names) > 0 {
		found := false
		for _, name := range tf.Names {
			if task.Name == name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// 检查类型
	if len(tf.Types) > 0 {
		found := false
		for _, taskType := range tf.Types {
			if task.Type == taskType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// 检查状态
	if len(tf.Statuses) > 0 {
		found := false
		for _, status := range tf.Statuses {
			if task.Status == status {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// 检查标签
	if len(tf.Tags) > 0 {
		for key, value := range tf.Tags {
			if taskValue, exists := task.Tags[key]; !exists || taskValue != value {
				return false
			}
		}
	}

	// 检查时间范围
	if tf.TimeRange != nil {
		if task.CreatedAt.Before(tf.TimeRange.Start) || task.CreatedAt.After(tf.TimeRange.End) {
			return false
		}
	}

	return true
}