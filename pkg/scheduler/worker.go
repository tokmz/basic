package scheduler

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// worker 工作协程实现
type worker struct {
	id           string
	scheduler    *Scheduler
	taskChan     <-chan *Task
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	stats        WorkerStats
	statsMutex   sync.RWMutex
	running      int32
	startTime    time.Time
}

// NewWorker 创建新的工作协程
func NewWorker(id string, scheduler *Scheduler, taskChan <-chan *Task) Worker {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &worker{
		id:        id,
		scheduler: scheduler,
		taskChan:  taskChan,
		ctx:       ctx,
		cancel:    cancel,
		stats: WorkerStats{
			ID:     id,
			Status: "stopped",
		},
	}
}

// Start 启动工作协程
func (w *worker) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&w.running, 0, 1) {
		return fmt.Errorf("worker %s already running", w.id)
	}
	
	w.startTime = time.Now()
	w.updateStats("running", 0, nil)
	
	w.wg.Add(1)
	go w.run()
	
	return nil
}

// Stop 停止工作协程
func (w *worker) Stop() error {
	if !atomic.CompareAndSwapInt32(&w.running, 1, 0) {
		return fmt.Errorf("worker %s not running", w.id)
	}
	
	w.cancel()
	w.wg.Wait()
	
	w.updateStats("stopped", 0, nil)
	return nil
}

// IsRunning 检查是否正在运行
func (w *worker) IsRunning() bool {
	return atomic.LoadInt32(&w.running) == 1
}

// GetID 获取工作协程ID
func (w *worker) GetID() string {
	return w.id
}

// GetStats 获取统计信息
func (w *worker) GetStats() WorkerStats {
	w.statsMutex.RLock()
	defer w.statsMutex.RUnlock()
	
	stats := w.stats
	if w.IsRunning() {
		stats.TotalRunTime = time.Since(w.startTime)
	}
	
	return stats
}

// run 主循环
func (w *worker) run() {
	defer w.wg.Done()
	
	w.scheduler.logger.Info("Worker started", "worker_id", w.id)
	
	for {
		select {
		case <-w.ctx.Done():
			w.scheduler.logger.Info("Worker stopped", "worker_id", w.id)
			return
			
		case task := <-w.taskChan:
			if task != nil {
				w.executeTask(task)
			}
		}
	}
}

// executeTask 执行任务
func (w *worker) executeTask(task *Task) {
	startTime := time.Now()
	execution := &TaskExecution{
		ID:        generateExecutionID(),
		TaskID:    task.ID,
		Status:    TaskStatusRunning,
		StartTime: startTime,
		Metadata:  make(map[string]string),
	}
	
	// 更新任务状态
	task.Status = TaskStatusRunning
	task.UpdatedAt = startTime
	
	// 创建执行上下文
	ctx, cancel := context.WithCancel(w.ctx)
	if task.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, task.Timeout)
	}
	task.cancelFn = cancel
	
	// 保存执行记录
	if w.scheduler.store != nil {
		if err := w.scheduler.store.SaveExecution(execution); err != nil {
			w.scheduler.logger.Error("Failed to save execution record", 
				"execution_id", execution.ID, 
				"task_id", task.ID, 
				"error", err)
		}
	}
	
	// 发布任务开始事件
	if w.scheduler.eventBus != nil {
		w.scheduler.eventBus.Publish(Event{
			Type:      EventTaskStarted,
			Timestamp: startTime,
			TaskID:    task.ID,
			TaskName:  task.Name,
			Data: map[string]interface{}{
				"execution_id": execution.ID,
				"worker_id":    w.id,
			},
		})
	}
	
	// 执行前置钩子
	if w.scheduler.hookManager != nil {
		hookData := map[string]interface{}{
			"task":        task,
			"execution":   execution,
			"worker_id":   w.id,
		}
		if err := w.scheduler.hookManager.ExecuteHook(ctx, HookBeforeTaskRun, hookData); err != nil {
			w.scheduler.logger.Warn("Before task run hook failed", 
				"task_id", task.ID, 
				"error", err)
		}
	}
	
	// 执行任务
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("task panic: %v", r)
				w.scheduler.logger.Error("Task panicked", 
					"task_id", task.ID, 
					"panic", r)
			}
		}()
		
		err = task.fn(ctx, task)
	}()
	
	// 完成时间和持续时间
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	execution.EndTime = &endTime
	execution.Duration = &duration
	
	// 处理执行结果
	w.handleTaskResult(task, execution, err, ctx, cancel)

	// 调用调度器的任务完成回调
	w.scheduler.onTaskComplete(execution)
	
	// 更新统计信息
	now := time.Now()
	if err != nil {
		w.updateStats("", 1, &now)
	} else {
		w.updateStats("", 0, &now)
	}
	
	// 更新任务计数
	atomic.AddInt64(&task.RunCount, 1)
	task.LastRunTime = &endTime
	task.LastDuration = &duration
	
	if err != nil {
		atomic.AddInt64(&task.FailureCount, 1)
		task.LastError = err.Error()
	} else {
		atomic.AddInt64(&task.SuccessCount, 1)
		task.LastError = ""
	}
	
	// 指标收集
	if w.scheduler.metrics != nil {
		w.scheduler.metrics.ObserveTaskDuration(task.Type.String(), duration)
		if err != nil {
			w.scheduler.metrics.IncTasksFailed(task.Type.String())
		} else {
			w.scheduler.metrics.IncTasksSucceeded(task.Type.String())
		}
	}
	
	// 清理取消函数
	task.cancelFn = nil
	
	w.scheduler.logger.Info("Task execution completed", 
		"task_id", task.ID, 
		"task_name", task.Name,
		"status", task.Status.String(),
		"duration", duration.String(),
		"worker_id", w.id)
}

// handleTaskResult 处理任务执行结果
func (w *worker) handleTaskResult(task *Task, execution *TaskExecution, taskErr error, ctx context.Context, cancel context.CancelFunc) {
	defer cancel()
	
	var eventType EventType
	
	if taskErr != nil {
		// 检查是否是取消或超时
		if ctx.Err() == context.Canceled {
			task.Status = TaskStatusCanceled
			execution.Status = TaskStatusCanceled
			execution.Error = "task canceled"
			eventType = EventTaskCanceled
			
			// 指标收集
			if w.scheduler.metrics != nil {
				w.scheduler.metrics.IncTasksCanceled(task.Type.String())
			}
		} else if ctx.Err() == context.DeadlineExceeded {
			task.Status = TaskStatusFailed
			execution.Status = TaskStatusFailed
			execution.Error = "task timeout"
			eventType = EventTaskFailed
			task.LastError = "task timeout"
		} else {
			task.Status = TaskStatusFailed
			execution.Status = TaskStatusFailed
			execution.Error = taskErr.Error()
			eventType = EventTaskFailed
			task.LastError = taskErr.Error()
		}
		
		// 检查是否需要重试
		if task.ShouldRetry() && task.Status == TaskStatusFailed {
			task.CurrentRetry++
			task.Status = TaskStatusPending
			
			// 计算下次重试时间
			retryTime := time.Now().Add(task.RetryDelay * time.Duration(task.CurrentRetry))
			task.NextRunTime = &retryTime
			
			eventType = EventTaskRetrying
			
			w.scheduler.logger.Info("Task will retry", 
				"task_id", task.ID, 
				"retry_count", task.CurrentRetry,
				"next_retry", retryTime.Format(time.RFC3339))
			
			// 执行重试钩子
			if w.scheduler.hookManager != nil {
				hookData := map[string]interface{}{
					"task":        task,
					"execution":   execution,
					"error":       taskErr,
					"retry_count": task.CurrentRetry,
					"next_retry":  retryTime,
				}
				if err := w.scheduler.hookManager.ExecuteHook(ctx, HookOnTaskRetry, hookData); err != nil {
					w.scheduler.logger.Warn("Task retry hook failed", 
						"task_id", task.ID, 
						"error", err)
				}
			}
			
			// 重新调度任务
			w.scheduler.scheduleTask(task)
		} else {
			// 执行失败钩子
			if w.scheduler.hookManager != nil {
				hookData := map[string]interface{}{
					"task":      task,
					"execution": execution,
					"error":     taskErr,
				}
				if err := w.scheduler.hookManager.ExecuteHook(ctx, HookOnTaskFailure, hookData); err != nil {
					w.scheduler.logger.Warn("Task failure hook failed", 
						"task_id", task.ID, 
						"error", err)
				}
			}
		}
	} else {
		task.Status = TaskStatusSuccess
		execution.Status = TaskStatusSuccess
		eventType = EventTaskCompleted
		
		// 重置重试计数
		task.CurrentRetry = 0
		
		// 执行成功钩子
		if w.scheduler.hookManager != nil {
			hookData := map[string]interface{}{
				"task":      task,
				"execution": execution,
			}
			if err := w.scheduler.hookManager.ExecuteHook(ctx, HookOnTaskSuccess, hookData); err != nil {
				w.scheduler.logger.Warn("Task success hook failed", 
					"task_id", task.ID, 
					"error", err)
			}
		}
		
		// 处理重复任务
		if task.Type != TaskTypeOnce && !task.IsExpired() {
			if err := w.scheduler.scheduleNextRun(task); err != nil {
				w.scheduler.logger.Error("Failed to schedule next run", 
					"task_id", task.ID, 
					"error", err)
				task.Status = TaskStatusFailed
				task.LastError = fmt.Sprintf("schedule next run failed: %v", err)
			}
		}
	}
	
	// 更新任务
	task.UpdatedAt = time.Now()
	
	// 执行后置钩子
	if w.scheduler.hookManager != nil {
		hookData := map[string]interface{}{
			"task":      task,
			"execution": execution,
			"error":     taskErr,
		}
		if err := w.scheduler.hookManager.ExecuteHook(ctx, HookAfterTaskRun, hookData); err != nil {
			w.scheduler.logger.Warn("After task run hook failed", 
				"task_id", task.ID, 
				"error", err)
		}
	}
	
	// 保存执行记录
	if w.scheduler.store != nil {
		if err := w.scheduler.store.SaveExecution(execution); err != nil {
			w.scheduler.logger.Error("Failed to save execution record", 
				"execution_id", execution.ID, 
				"task_id", task.ID, 
				"error", err)
		}
		
		// 更新任务
		if err := w.scheduler.store.UpdateTask(task); err != nil {
			w.scheduler.logger.Error("Failed to update task", 
				"task_id", task.ID, 
				"error", err)
		}
	}
	
	// 发布事件
	if w.scheduler.eventBus != nil {
		event := Event{
			Type:      eventType,
			Timestamp: time.Now(),
			TaskID:    task.ID,
			TaskName:  task.Name,
			Data: map[string]interface{}{
				"execution_id": execution.ID,
				"worker_id":    w.id,
				"duration":     execution.Duration.String(),
				"status":       task.Status.String(),
			},
		}
		
		if taskErr != nil {
			event.Error = taskErr.Error()
			event.Data["error"] = taskErr.Error()
		}
		
		w.scheduler.eventBus.Publish(event)
	}
}

// updateStats 更新统计信息
func (w *worker) updateStats(status string, errorInc int64, lastTaskTime *time.Time) {
	w.statsMutex.Lock()
	defer w.statsMutex.Unlock()
	
	if status != "" {
		w.stats.Status = status
	}
	
	if errorInc != 0 {
		w.stats.Errors += errorInc
	}
	
	if lastTaskTime != nil {
		w.stats.TasksHandled++
		w.stats.LastTaskTime = lastTaskTime
	}
}