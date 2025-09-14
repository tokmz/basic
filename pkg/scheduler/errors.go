package scheduler

import (
	"errors"
	"fmt"
)

// 预定义错误
var (
	ErrSchedulerNotStarted = errors.New("scheduler not started")
	ErrSchedulerAlreadyStarted = errors.New("scheduler already started")
	ErrTaskNotFound = errors.New("task not found")
	ErrTaskAlreadyExists = errors.New("task already exists")
	ErrTaskFuncNotSet = errors.New("task function not set")
	ErrInvalidCronExpr = errors.New("invalid cron expression")
	ErrInvalidInterval = errors.New("invalid interval")
	ErrInvalidTaskType = errors.New("invalid task type")
	ErrTaskCanceled = errors.New("task canceled")
	ErrTaskTimeout = errors.New("task timeout")
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")
	ErrInvalidPriority = errors.New("invalid priority")
	ErrSchedulerShuttingDown = errors.New("scheduler is shutting down")
)

// TaskError 任务错误
type TaskError struct {
	TaskID  string
	TaskName string
	Op      string // 操作
	Err     error
}

func (e *TaskError) Error() string {
	if e.TaskName != "" {
		return fmt.Sprintf("task %s (%s): %s: %v", e.TaskName, e.TaskID, e.Op, e.Err)
	}
	return fmt.Sprintf("task %s: %s: %v", e.TaskID, e.Op, e.Err)
}

func (e *TaskError) Unwrap() error {
	return e.Err
}

// NewTaskError 创建任务错误
func NewTaskError(taskID, taskName, op string, err error) *TaskError {
	return &TaskError{
		TaskID:   taskID,
		TaskName: taskName,
		Op:       op,
		Err:      err,
	}
}

// SchedulerError 调度器错误
type SchedulerError struct {
	Op  string
	Err error
}

func (e *SchedulerError) Error() string {
	return fmt.Sprintf("scheduler: %s: %v", e.Op, e.Err)
}

func (e *SchedulerError) Unwrap() error {
	return e.Err
}

// NewSchedulerError 创建调度器错误
func NewSchedulerError(op string, err error) *SchedulerError {
	return &SchedulerError{
		Op:  op,
		Err: err,
	}
}

// CronError Cron表达式错误
type CronError struct {
	Expr string
	Err  error
}

func (e *CronError) Error() string {
	return fmt.Sprintf("cron expression '%s': %v", e.Expr, e.Err)
}

func (e *CronError) Unwrap() error {
	return e.Err
}

// NewCronError 创建Cron错误
func NewCronError(expr string, err error) *CronError {
	return &CronError{
		Expr: expr,
		Err:  err,
	}
}