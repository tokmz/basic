package scheduler

import (
	"context"
	"time"
)

// Worker 工作协程接口
type Worker interface {
	Start(ctx context.Context) error
	Stop() error
	IsRunning() bool
	GetID() string
	GetStats() WorkerStats
}

// WorkerStats 工作协程统计信息
type WorkerStats struct {
	ID           string        `json:"id"`
	Status       string        `json:"status"`
	TasksHandled int64         `json:"tasks_handled"`
	LastTaskTime *time.Time    `json:"last_task_time,omitempty"`
	TotalRunTime time.Duration `json:"total_run_time"`
	Errors       int64         `json:"errors"`
}

// PersistenceStore 持久化存储接口
type PersistenceStore interface {
	// 任务相关
	SaveTask(task *Task) error
	LoadTask(id string) (*Task, error)
	LoadAllTasks() ([]*Task, error)
	DeleteTask(id string) error
	UpdateTask(task *Task) error
	
	// 执行记录相关
	SaveExecution(execution *TaskExecution) error
	LoadExecutions(taskID string, limit int) ([]*TaskExecution, error)
	LoadExecutionsByTimeRange(start, end time.Time) ([]*TaskExecution, error)
	DeleteExecution(id string) error
	
	// 清理相关
	CleanupExecutions(olderThan time.Time) (int64, error)
	
	// 事务支持
	Begin() (Transaction, error)
	
	// 关闭
	Close() error
}

// Transaction 事务接口
type Transaction interface {
	SaveTask(task *Task) error
	UpdateTask(task *Task) error
	DeleteTask(id string) error
	Commit() error
	Rollback() error
}

// MetricsCollector 指标收集器接口
type MetricsCollector interface {
	// 任务指标
	IncTasksTotal(taskType string)
	IncTasksSucceeded(taskType string)
	IncTasksFailed(taskType string)
	IncTasksCanceled(taskType string)
	ObserveTaskDuration(taskType string, duration time.Duration)
	
	// 调度器指标
	SetActiveTasks(count int)
	SetPendingTasks(count int)
	SetRunningTasks(count int)
	SetWorkerCount(count int)
	
	// 错误指标
	IncErrors(errorType string)
	
	// 自定义指标
	Increment(name string, labels map[string]string)
	Gauge(name string, value float64, labels map[string]string)
	Histogram(name string, value float64, labels map[string]string)
	
	// 获取指标
	GetMetrics() map[string]interface{}
}

// EventBus 事件总线接口
type EventBus interface {
	// 发布事件
	Publish(event Event)
	
	// 订阅事件
	Subscribe(eventType EventType, handler EventHandler) Subscription
	
	// 取消订阅
	Unsubscribe(subscription Subscription)
	
	// 关闭
	Close() error
}

// EventType 事件类型
type EventType string

const (
	EventTaskAdded     EventType = "task.added"
	EventTaskRemoved   EventType = "task.removed"
	EventTaskUpdated   EventType = "task.updated"
	EventTaskStarted   EventType = "task.started"
	EventTaskCompleted EventType = "task.completed"
	EventTaskFailed    EventType = "task.failed"
	EventTaskCanceled  EventType = "task.canceled"
	EventTaskPaused    EventType = "task.paused"
	EventTaskResumed   EventType = "task.resumed"
	EventTaskRetrying  EventType = "task.retrying"
	
	EventSchedulerStarted  EventType = "scheduler.started"
	EventSchedulerStopped  EventType = "scheduler.stopped"
	EventSchedulerPaused   EventType = "scheduler.paused"
	EventSchedulerResumed  EventType = "scheduler.resumed"
	
	EventWorkerStarted EventType = "worker.started"
	EventWorkerStopped EventType = "worker.stopped"
	EventWorkerError   EventType = "worker.error"
)

// Event 事件定义
type Event struct {
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	TaskID    string                 `json:"task_id,omitempty"`
	TaskName  string                 `json:"task_name,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// EventHandler 事件处理器
type EventHandler func(event Event)

// Subscription 订阅
type Subscription interface {
	ID() string
	EventType() EventType
	IsActive() bool
	Cancel()
}

// HookType 钩子类型
type HookType string

const (
	HookBeforeTaskRun  HookType = "before_task_run"
	HookAfterTaskRun   HookType = "after_task_run"
	HookOnTaskSuccess  HookType = "on_task_success"
	HookOnTaskFailure  HookType = "on_task_failure"
	HookOnTaskCancel   HookType = "on_task_cancel"
	HookOnTaskRetry    HookType = "on_task_retry"

	// 添加缺失的钩子类型
	HookTaskAdded      HookType = "task_added"
	HookTaskRemoved    HookType = "task_removed"
	HookTaskPaused     HookType = "task_paused"
	HookTaskResumed    HookType = "task_resumed"
	HookTaskRetrying   HookType = "task_retrying"
	HookTaskCompleted  HookType = "task_completed"
	HookTaskFailed     HookType = "task_failed"

	HookBeforeSchedulerStart HookType = "before_scheduler_start"
	HookAfterSchedulerStart  HookType = "after_scheduler_start"
	HookBeforeSchedulerStop  HookType = "before_scheduler_stop"
	HookAfterSchedulerStop   HookType = "after_scheduler_stop"
)

// HookFunc 钩子函数
type HookFunc func(ctx context.Context, data map[string]interface{}) error

// HookManager 钩子管理器接口
type HookManager interface {
	// 注册钩子
	RegisterHook(hookType HookType, hookFunc HookFunc) error
	
	// 执行钩子
	ExecuteHook(ctx context.Context, hookType HookType, data map[string]interface{}) error
	
	// 移除钩子
	RemoveHook(hookType HookType, hookFunc HookFunc) error
	
	// 清空钩子
	ClearHooks(hookType HookType)
	
	// 获取钩子数量
	GetHookCount(hookType HookType) int
}

// Logger 日志接口
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Fatal(msg string, fields ...interface{})
	
	WithFields(fields map[string]interface{}) Logger
	WithError(err error) Logger
}