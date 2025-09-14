package scheduler

import (
	"context"
	"time"
)

// TaskStatus 任务状态
type TaskStatus int

const (
	TaskStatusPending TaskStatus = iota // 待执行
	TaskStatusRunning                   // 运行中
	TaskStatusSuccess                   // 成功完成
	TaskStatusFailed                    // 执行失败
	TaskStatusCanceled                  // 已取消
	TaskStatusPaused                    // 已暂停
)

func (s TaskStatus) String() string {
	switch s {
	case TaskStatusPending:
		return "pending"
	case TaskStatusRunning:
		return "running"
	case TaskStatusSuccess:
		return "success"
	case TaskStatusFailed:
		return "failed"
	case TaskStatusCanceled:
		return "canceled"
	case TaskStatusPaused:
		return "paused"
	default:
		return "unknown"
	}
}

// TaskType 任务类型
type TaskType int

const (
	TaskTypeOnce      TaskType = iota // 一次性任务
	TaskTypeRecurring                 // 周期性任务
	TaskTypeCron                      // Cron表达式任务
	TaskTypeInterval                  // 间隔任务
)

func (t TaskType) String() string {
	switch t {
	case TaskTypeOnce:
		return "once"
	case TaskTypeRecurring:
		return "recurring"
	case TaskTypeCron:
		return "cron"
	case TaskTypeInterval:
		return "interval"
	default:
		return "unknown"
	}
}

// Priority 任务优先级
type Priority int

const (
	PriorityLow    Priority = iota // 低优先级
	PriorityNormal                 // 普通优先级
	PriorityHigh                   // 高优先级
	PriorityCritical               // 关键优先级
)

func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityNormal:
		return "normal"
	case PriorityHigh:
		return "high"
	case PriorityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// TaskFunc 任务执行函数类型
type TaskFunc func(ctx context.Context, task *Task) error

// Task 任务定义
type Task struct {
	ID          string            `json:"id"`                    // 任务ID
	Name        string            `json:"name"`                  // 任务名称
	Description string            `json:"description,omitempty"` // 任务描述
	Type        TaskType          `json:"type"`                  // 任务类型
	Status      TaskStatus        `json:"status"`                // 任务状态
	Priority    Priority          `json:"priority"`              // 优先级
	
	// 调度配置
	CronExpr     string        `json:"cron_expr,omitempty"`     // Cron表达式
	Interval     time.Duration `json:"interval,omitempty"`      // 执行间隔
	StartTime    *time.Time    `json:"start_time,omitempty"`    // 开始时间
	EndTime      *time.Time    `json:"end_time,omitempty"`      // 结束时间
	NextRunTime  *time.Time    `json:"next_run_time,omitempty"` // 下次执行时间
	
	// 重试配置
	MaxRetries   int           `json:"max_retries"`             // 最大重试次数
	RetryDelay   time.Duration `json:"retry_delay"`             // 重试延迟
	CurrentRetry int           `json:"current_retry"`           // 当前重试次数
	
	// 超时配置
	Timeout time.Duration `json:"timeout,omitempty"` // 执行超时时间
	
	// 元数据
	Tags      map[string]string `json:"tags,omitempty"`      // 任务标签
	Metadata  map[string]string `json:"metadata,omitempty"`  // 元数据
	CreatedAt time.Time         `json:"created_at"`          // 创建时间
	UpdatedAt time.Time         `json:"updated_at"`          // 更新时间
	
	// 执行历史
	LastRunTime   *time.Time `json:"last_run_time,omitempty"`   // 上次执行时间
	LastDuration  *time.Duration `json:"last_duration,omitempty"` // 上次执行时长
	LastError     string     `json:"last_error,omitempty"`      // 上次错误信息
	RunCount      int64      `json:"run_count"`                 // 执行次数
	SuccessCount  int64      `json:"success_count"`             // 成功次数
	FailureCount  int64      `json:"failure_count"`             // 失败次数
	
	// 内部字段
	fn       TaskFunc        `json:"-"` // 执行函数
	cancelFn context.CancelFunc `json:"-"` // 取消函数
}

// TaskExecution 任务执行记录
type TaskExecution struct {
	ID        string        `json:"id"`         // 执行记录ID
	TaskID    string        `json:"task_id"`    // 任务ID
	Status    TaskStatus    `json:"status"`     // 执行状态
	StartTime time.Time     `json:"start_time"` // 开始时间
	EndTime   *time.Time    `json:"end_time,omitempty"` // 结束时间
	Duration  *time.Duration `json:"duration,omitempty"` // 执行时长
	Error     string        `json:"error,omitempty"`    // 错误信息
	Output    string        `json:"output,omitempty"`   // 输出信息
	Metadata  map[string]string `json:"metadata,omitempty"` // 执行元数据
}

// TaskBuilder 任务构建器
type TaskBuilder struct {
	task *Task
}

// NewTaskBuilder 创建任务构建器
func NewTaskBuilder(name string) *TaskBuilder {
	return &TaskBuilder{
		task: &Task{
			ID:          generateTaskID(),
			Name:        name,
			Type:        TaskTypeOnce,
			Status:      TaskStatusPending,
			Priority:    PriorityNormal,
			MaxRetries:  3,
			RetryDelay:  time.Second * 5,
			Tags:        make(map[string]string),
			Metadata:    make(map[string]string),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}
}

// WithDescription 设置任务描述
func (tb *TaskBuilder) WithDescription(desc string) *TaskBuilder {
	tb.task.Description = desc
	return tb
}

// WithType 设置任务类型
func (tb *TaskBuilder) WithType(taskType TaskType) *TaskBuilder {
	tb.task.Type = taskType
	return tb
}

// WithPriority 设置优先级
func (tb *TaskBuilder) WithPriority(priority Priority) *TaskBuilder {
	tb.task.Priority = priority
	return tb
}

// WithCron 设置Cron表达式
func (tb *TaskBuilder) WithCron(cronExpr string) *TaskBuilder {
	tb.task.Type = TaskTypeCron
	tb.task.CronExpr = cronExpr
	return tb
}

// WithInterval 设置执行间隔
func (tb *TaskBuilder) WithInterval(interval time.Duration) *TaskBuilder {
	tb.task.Type = TaskTypeInterval
	tb.task.Interval = interval
	return tb
}

// WithStartTime 设置开始时间
func (tb *TaskBuilder) WithStartTime(startTime time.Time) *TaskBuilder {
	tb.task.StartTime = &startTime
	return tb
}

// WithEndTime 设置结束时间
func (tb *TaskBuilder) WithEndTime(endTime time.Time) *TaskBuilder {
	tb.task.EndTime = &endTime
	return tb
}

// WithTimeout 设置执行超时
func (tb *TaskBuilder) WithTimeout(timeout time.Duration) *TaskBuilder {
	tb.task.Timeout = timeout
	return tb
}

// WithRetry 设置重试配置
func (tb *TaskBuilder) WithRetry(maxRetries int, retryDelay time.Duration) *TaskBuilder {
	tb.task.MaxRetries = maxRetries
	tb.task.RetryDelay = retryDelay
	return tb
}

// WithTag 添加标签
func (tb *TaskBuilder) WithTag(key, value string) *TaskBuilder {
	tb.task.Tags[key] = value
	return tb
}

// WithMetadata 添加元数据
func (tb *TaskBuilder) WithMetadata(key, value string) *TaskBuilder {
	tb.task.Metadata[key] = value
	return tb
}

// WithFunc 设置执行函数
func (tb *TaskBuilder) WithFunc(fn TaskFunc) *TaskBuilder {
	tb.task.fn = fn
	return tb
}

// Build 构建任务
func (tb *TaskBuilder) Build() (*Task, error) {
	if tb.task.fn == nil {
		return nil, ErrTaskFuncNotSet
	}
	
	// 根据任务类型设置下次执行时间
	if err := tb.calculateNextRunTime(); err != nil {
		return nil, err
	}
	
	return tb.task, nil
}

// calculateNextRunTime 计算下次执行时间
func (tb *TaskBuilder) calculateNextRunTime() error {
	now := time.Now()
	
	switch tb.task.Type {
	case TaskTypeOnce:
		if tb.task.StartTime != nil {
			tb.task.NextRunTime = tb.task.StartTime
		} else {
			tb.task.NextRunTime = &now
		}
		
	case TaskTypeInterval:
		if tb.task.Interval <= 0 {
			return ErrInvalidInterval
		}
		startTime := now
		if tb.task.StartTime != nil {
			startTime = *tb.task.StartTime
		}
		tb.task.NextRunTime = &startTime
		
	case TaskTypeCron:
		if tb.task.CronExpr == "" {
			return ErrInvalidCronExpr
		}
		nextTime, err := getNextCronTime(tb.task.CronExpr, now)
		if err != nil {
			return err
		}
		tb.task.NextRunTime = &nextTime
		
	case TaskTypeRecurring:
		if tb.task.Interval <= 0 {
			return ErrInvalidInterval
		}
		startTime := now
		if tb.task.StartTime != nil {
			startTime = *tb.task.StartTime
		}
		tb.task.NextRunTime = &startTime
	}
	
	return nil
}

// IsExpired 检查任务是否已过期
func (t *Task) IsExpired() bool {
	if t.EndTime == nil {
		return false
	}
	return time.Now().After(*t.EndTime)
}

// CanRun 检查任务是否可以运行
func (t *Task) CanRun() bool {
	now := time.Now()
	
	// 检查状态
	if t.Status != TaskStatusPending && t.Status != TaskStatusPaused {
		return false
	}
	
	// 检查是否已过期
	if t.IsExpired() {
		return false
	}
	
	// 检查开始时间
	if t.StartTime != nil && now.Before(*t.StartTime) {
		return false
	}
	
	// 检查下次执行时间
	if t.NextRunTime != nil && now.Before(*t.NextRunTime) {
		return false
	}
	
	return true
}

// ShouldRetry 检查是否应该重试
func (t *Task) ShouldRetry() bool {
	return t.Status == TaskStatusFailed && t.CurrentRetry < t.MaxRetries
}

// Clone 克隆任务
func (t *Task) Clone() *Task {
	cloned := *t
	
	// 深拷贝map
	if t.Tags != nil {
		cloned.Tags = make(map[string]string)
		for k, v := range t.Tags {
			cloned.Tags[k] = v
		}
	}
	
	if t.Metadata != nil {
		cloned.Metadata = make(map[string]string)
		for k, v := range t.Metadata {
			cloned.Metadata[k] = v
		}
	}
	
	return &cloned
}