package scheduler

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	// 基础配置
	Name            string        `json:"name"`             // 调度器名称
	MaxWorkers      int           `json:"max_workers"`      // 最大工作者数量
	WorkerQueueSize int           `json:"worker_queue_size"` // 工作者队列大小
	TickInterval    time.Duration `json:"tick_interval"`    // 调度检查间隔
	
	// 任务执行配置
	DefaultTimeout     time.Duration `json:"default_timeout"`      // 默认任务超时时间
	MaxRetries         int           `json:"max_retries"`          // 默认最大重试次数
	RetryDelay         time.Duration `json:"retry_delay"`          // 默认重试延迟
	
	// 持久化配置
	EnablePersistence  bool   `json:"enable_persistence"`  // 是否启用持久化
	PersistenceDriver  string `json:"persistence_driver"`  // 持久化驱动
	PersistenceConfig  map[string]interface{} `json:"persistence_config"` // 持久化配置
	
	// 监控配置
	EnableMetrics      bool          `json:"enable_metrics"`       // 是否启用指标收集
	MetricsInterval    time.Duration `json:"metrics_interval"`     // 指标收集间隔
	
	// 分布式配置
	EnableDistributed  bool   `json:"enable_distributed"`  // 是否启用分布式
	NodeID            string `json:"node_id"`             // 节点ID
	
	// 其他配置
	EnableRecovery     bool          `json:"enable_recovery"`      // 是否启用崩溃恢复
	CleanupInterval    time.Duration `json:"cleanup_interval"`     // 清理间隔
	MaxHistorySize     int           `json:"max_history_size"`     // 最大历史记录数
}

// DefaultSchedulerConfig 默认调度器配置
func DefaultSchedulerConfig() *SchedulerConfig {
	return &SchedulerConfig{
		Name:               "default-scheduler",
		MaxWorkers:         10,
		WorkerQueueSize:    100,
		TickInterval:       time.Second,
		DefaultTimeout:     time.Minute * 10,
		MaxRetries:         3,
		RetryDelay:         time.Second * 5,
		EnablePersistence:  false,
		PersistenceDriver:  "memory",
		EnableMetrics:      true,
		MetricsInterval:    time.Second * 30,
		EnableDistributed:  false,
		EnableRecovery:     true,
		CleanupInterval:    time.Hour,
		MaxHistorySize:     1000,
	}
}

// SchedulerStats 调度器统计信息
type SchedulerStats struct {
	Name             string    `json:"name"`
	Running          bool      `json:"running"`
	WorkerCount      int       `json:"worker_count"`
	TotalTasks       int       `json:"total_tasks"`
	PendingTasks     int       `json:"pending_tasks"`
	RunningTasks     int       `json:"running_tasks"`
	CompletedTasks   int       `json:"completed_tasks"`
	FailedTasks      int       `json:"failed_tasks"`
	PausedTasks      int       `json:"paused_tasks"`
	QueueLength      int       `json:"queue_length"`
	TotalExecutions  int64     `json:"total_executions"`
	TotalSuccesses   int64     `json:"total_successes"`
	TotalFailures    int64     `json:"total_failures"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Scheduler 任务调度器
type Scheduler struct {
	config   *SchedulerConfig
	tasks    map[string]*Task
	workers  []Worker
	taskChan chan *Task
	
	// 状态管理
	running   bool
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	
	// 组件
	store       PersistenceStore
	metrics     MetricsCollector
	eventBus    EventBus
	hookManager HookManager
	logger      Logger
	
	// 执行历史
	executions   map[string]*TaskExecution
	executionsMu sync.RWMutex
}

// NewScheduler 创建新的调度器
func NewScheduler(config *SchedulerConfig) (*Scheduler, error) {
	if config == nil {
		config = DefaultSchedulerConfig()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	scheduler := &Scheduler{
		config:     config,
		tasks:      make(map[string]*Task),
		taskChan:   make(chan *Task, config.WorkerQueueSize),
		ctx:        ctx,
		cancel:     cancel,
		executions: make(map[string]*TaskExecution),
	}
	
	// 初始化持久化存储
	if config.EnablePersistence {
		if config.PersistenceDriver == "file" {
			baseDir := "./scheduler_data"
			if config.PersistenceConfig != nil {
				if dir, ok := config.PersistenceConfig["base_dir"].(string); ok {
					baseDir = dir
				}
			}
			store, err := NewFilePersistenceStore(baseDir)
			if err != nil {
				return nil, NewSchedulerError("create file persistence store", err)
			}
			scheduler.store = store
		} else {
			scheduler.store = NewMemoryPersistenceStore()
		}
	} else {
		scheduler.store = NewMemoryPersistenceStore()
	}
	
	// 初始化指标收集器
	if config.EnableMetrics {
		scheduler.metrics = NewSimpleMetricsCollector()
	} else {
		scheduler.metrics = NewNoOpMetricsCollector()
	}
	
	// 初始化事件总线
	scheduler.eventBus = NewSimpleEventBus()
	
	// 初始化钩子管理器
	scheduler.hookManager = NewSimpleHookManager()
	
	// 初始化日志器（简单实现）
	scheduler.logger = &simpleLogger{}
	
	// 创建工作者
	scheduler.workers = make([]Worker, config.MaxWorkers)
	for i := 0; i < config.MaxWorkers; i++ {
		workerID := fmt.Sprintf("worker-%d", i)
		worker := NewWorker(workerID, scheduler, scheduler.taskChan)
		scheduler.workers[i] = worker
	}
	
	return scheduler, nil
}

// Start 启动调度器
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.running {
		return ErrSchedulerAlreadyStarted
	}
	
	s.running = true
	
	// 启动工作者
	for _, worker := range s.workers {
		s.wg.Add(1)
		go func(w Worker) {
			defer s.wg.Done()
			w.Start(s.ctx)
		}(worker)
	}
	
	// 启动调度循环
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.scheduleLoop()
	}()
	
	// 更新指标
	if s.metrics != nil {
		s.metrics.SetWorkerCount(len(s.workers))
	}
	
	// 启动清理任务
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.cleanupLoop()
	}()
	
	// 恢复任务
	if s.config.EnableRecovery {
		if err := s.recoverTasks(); err != nil {
			return NewSchedulerError("recover tasks", err)
		}
	}
	
	// 发布启动事件
	s.eventBus.Publish(Event{
		Type:      EventSchedulerStarted,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"name": s.config.Name,
		},
	})
	
	return nil
}

// Stop 停止调度器
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.running {
		return ErrSchedulerNotStarted
	}
	
	s.running = false
	
	// 取消上下文
	s.cancel()
	
	// 等待所有goroutine完成
	s.wg.Wait()
	
	// 关闭任务通道
	close(s.taskChan)
	
	// 保存状态
	if s.store != nil {
		if err := s.saveState(); err != nil {
			return NewSchedulerError("save state", err)
		}
	}
	
	// 发布停止事件
	s.eventBus.Publish(Event{
		Type:      EventSchedulerStopped,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"name": s.config.Name,
		},
	})
	
	return nil
}

// AddTask 添加任务
func (s *Scheduler) AddTask(task *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.tasks[task.ID]; exists {
		return ErrTaskAlreadyExists
	}
	
	// 设置默认值
	if task.Timeout == 0 {
		task.Timeout = s.config.DefaultTimeout
	}
	if task.MaxRetries == 0 {
		task.MaxRetries = s.config.MaxRetries
	}
	if task.RetryDelay == 0 {
		task.RetryDelay = s.config.RetryDelay
	}
	
	s.tasks[task.ID] = task
	
	// 持久化任务
	if s.store != nil {
		if err := s.store.SaveTask(task); err != nil {
			delete(s.tasks, task.ID)
			return NewTaskError(task.ID, task.Name, "persist", err)
		}
	}
	
	// 执行钩子
	s.executeHooks(HookTaskAdded, task)
	
	// 发布事件
	s.eventBus.Publish(Event{
		Type:      EventTaskAdded,
		Timestamp: time.Now(),
		TaskID:    task.ID,
		TaskName:  task.Name,
		Data: map[string]interface{}{
			"task_id":   task.ID,
			"task_name": task.Name,
		},
	})
	
	// 更新指标
	if s.metrics != nil {
		s.metrics.Increment("tasks_added_total", nil)
	}
	
	return nil
}

// RemoveTask 移除任务
func (s *Scheduler) RemoveTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	task, exists := s.tasks[taskID]
	if !exists {
		return ErrTaskNotFound
	}
	
	// 取消正在运行的任务
	if task.cancelFn != nil {
		task.cancelFn()
	}
	
	delete(s.tasks, taskID)
	
	// 从持久化存储中删除
	if s.store != nil {
		if err := s.store.DeleteTask(taskID); err != nil {
			return NewTaskError(taskID, task.Name, "delete from persistence", err)
		}
	}
	
	// 执行钩子
	s.executeHooks(HookTaskRemoved, task)
	
	// 发布事件
	s.eventBus.Publish(Event{
		Type:      EventTaskRemoved,
		Timestamp: time.Now(),
		TaskID:    taskID,
		TaskName:  task.Name,
		Data: map[string]interface{}{
			"task_id":   taskID,
			"task_name": task.Name,
		},
	})
	
	// 更新指标
	if s.metrics != nil {
		s.metrics.Increment("tasks_removed_total", nil)
	}
	
	return nil
}

// GetTask 获取任务
func (s *Scheduler) GetTask(taskID string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	task, exists := s.tasks[taskID]
	if !exists {
		return nil, ErrTaskNotFound
	}
	
	return task.Clone(), nil
}

// ListTasks 列出所有任务
func (s *Scheduler) ListTasks(filter *TaskFilter) ([]*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var result []*Task
	
	for _, task := range s.tasks {
		if filter == nil || filter.Match(task) {
			result = append(result, task.Clone())
		}
	}
	
	// 按优先级和创建时间排序
	sort.Slice(result, func(i, j int) bool {
		if result[i].Priority != result[j].Priority {
			return result[i].Priority > result[j].Priority
		}
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	
	return result, nil
}

// PauseTask 暂停任务
func (s *Scheduler) PauseTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	task, exists := s.tasks[taskID]
	if !exists {
		return ErrTaskNotFound
	}
	
	if task.Status == TaskStatusRunning {
		if task.cancelFn != nil {
			task.cancelFn()
		}
	}
	
	task.Status = TaskStatusPaused
	task.UpdatedAt = time.Now()
	
	// 持久化更新
	if s.store != nil {
		if err := s.store.UpdateTask(task); err != nil {
			return NewTaskError(taskID, task.Name, "persist pause", err)
		}
	}
	
	// 执行钩子
	s.executeHooks(HookTaskPaused, task)
	
	// 发布事件
	s.eventBus.Publish(Event{
		Type:      EventTaskPaused,
		Timestamp: time.Now(),
		TaskID:    taskID,
		TaskName:  task.Name,
		Data: map[string]interface{}{
			"task_id":   taskID,
			"task_name": task.Name,
		},
	})
	
	return nil
}

// ResumeTask 恢复任务
func (s *Scheduler) ResumeTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	task, exists := s.tasks[taskID]
	if !exists {
		return ErrTaskNotFound
	}
	
	if task.Status != TaskStatusPaused {
		return fmt.Errorf("task is not paused")
	}
	
	task.Status = TaskStatusPending
	task.UpdatedAt = time.Now()
	
	// 重新计算下次执行时间
	s.calculateNextRunTime(task)
	
	// 持久化更新
	if s.store != nil {
		if err := s.store.UpdateTask(task); err != nil {
			return NewTaskError(taskID, task.Name, "persist resume", err)
		}
	}
	
	// 执行钩子
	s.executeHooks(HookTaskResumed, task)
	
	// 发布事件
	s.eventBus.Publish(Event{
		Type:      EventTaskResumed,
		Timestamp: time.Now(),
		TaskID:    taskID,
		TaskName:  task.Name,
		Data: map[string]interface{}{
			"task_id":   taskID,
			"task_name": task.Name,
		},
	})
	
	return nil
}

// scheduleLoop 调度循环
func (s *Scheduler) scheduleLoop() {
	ticker := time.NewTicker(s.config.TickInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkAndScheduleTasks()
		}
	}
}

// checkAndScheduleTasks 检查并调度任务
func (s *Scheduler) checkAndScheduleTasks() {
	s.mu.RLock()
	var readyTasks []*Task
	now := time.Now()
	
	for _, task := range s.tasks {
		if task.CanRun() && task.NextRunTime != nil && now.After(*task.NextRunTime) {
			readyTasks = append(readyTasks, task)
		}
	}
	s.mu.RUnlock()
	
	// 按优先级排序
	sort.Slice(readyTasks, func(i, j int) bool {
		return readyTasks[i].Priority > readyTasks[j].Priority
	})
	
	// 提交任务执行
	for _, task := range readyTasks {
		select {
		case s.taskChan <- task:
			s.mu.Lock()
			task.Status = TaskStatusRunning
			task.UpdatedAt = time.Now()
			s.mu.Unlock()
		case <-s.ctx.Done():
			return
		default:
			// 队列已满，跳过此次调度
			break
		}
	}
}

// onTaskComplete 任务完成回调
func (s *Scheduler) onTaskComplete(execution *TaskExecution) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[execution.TaskID]
	if !exists {
		return
	}

	// 更新任务状态
	task.Status = execution.Status
	task.LastRunTime = &execution.StartTime
	task.LastDuration = execution.Duration
	task.RunCount++
	task.UpdatedAt = time.Now()

	if execution.Error != "" {
		task.LastError = execution.Error
		task.FailureCount++
	} else {
		task.LastError = ""
		task.SuccessCount++
	}

	// 保存执行记录
	s.executionsMu.Lock()
	s.executions[execution.ID] = execution

	// 限制历史记录数量
	if len(s.executions) > s.config.MaxHistorySize {
		s.cleanupOldExecutions()
	}
	s.executionsMu.Unlock()

	// 处理重试
	if execution.Status == TaskStatusFailed && task.ShouldRetry() {
		task.CurrentRetry++
		task.Status = TaskStatusPending

		// 计算重试时间
		retryTime := time.Now().Add(task.RetryDelay)
		task.NextRunTime = &retryTime

		s.executeHooks(HookTaskRetrying, task)

		s.eventBus.Publish(Event{
			Type:      EventTaskRetrying,
			Timestamp: time.Now(),
			TaskID:    task.ID,
			TaskName:  task.Name,
			Data: map[string]interface{}{
				"task_id":       task.ID,
				"task_name":     task.Name,
				"retry_count":   task.CurrentRetry,
				"max_retries":   task.MaxRetries,
			},
		})
	} else {
		// 重置重试计数
		task.CurrentRetry = 0

		// 计算下次执行时间（对于周期性任务）
		s.calculateNextRunTime(task)

		// 检查任务是否应该移除
		if task.Type == TaskTypeOnce || task.IsExpired() {
			delete(s.tasks, task.ID)
			if s.store != nil {
				s.store.DeleteTask(task.ID)
			}
		}
	}

	// 持久化更新
	if s.store != nil {
		// 只有当任务还存在时才保存
		if _, stillExists := s.tasks[task.ID]; stillExists {
			s.store.UpdateTask(task)
		}
		s.store.SaveExecution(execution)
	}

	// 执行钩子
	if execution.Status == TaskStatusSuccess {
		s.executeHooks(HookTaskCompleted, task)
	} else {
		s.executeHooks(HookTaskFailed, task)
	}

	// 发布事件
	eventType := EventTaskCompleted
	if execution.Status == TaskStatusFailed {
		eventType = EventTaskFailed
	}

	eventData := Event{
		Type:      eventType,
		Timestamp: time.Now(),
		TaskID:    task.ID,
		TaskName:  task.Name,
		Data: map[string]interface{}{
			"task_id":   task.ID,
			"task_name": task.Name,
			"duration":  execution.Duration,
			"error":     execution.Error,
		},
	}
	if execution.Error != "" {
		eventData.Error = execution.Error
	}

	s.eventBus.Publish(eventData)

	// 更新指标
	if s.metrics != nil {
		if execution.Status == TaskStatusSuccess {
			s.metrics.Increment("tasks_completed_total", nil)
			s.metrics.Histogram("task_duration_seconds", execution.Duration.Seconds(), nil)
		} else {
			s.metrics.Increment("tasks_failed_total", nil)
		}
	}
}

// calculateNextRunTime 计算下次执行时间
func (s *Scheduler) calculateNextRunTime(task *Task) {
	if task.Type == TaskTypeOnce {
		return // 一次性任务不需要下次执行时间
	}
	
	now := time.Now()
	
	switch task.Type {
	case TaskTypeInterval, TaskTypeRecurring:
		if task.Interval > 0 {
			nextTime := now.Add(task.Interval)
			task.NextRunTime = &nextTime
		}
		
	case TaskTypeCron:
		if task.CronExpr != "" {
			nextTime, err := getNextCronTime(task.CronExpr, now)
			if err == nil {
				task.NextRunTime = &nextTime
			}
		}
	}
}

// cleanupLoop 清理循环
func (s *Scheduler) cleanupLoop() {
	ticker := time.NewTicker(s.config.CleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.cleanup()
		}
	}
}

// cleanup 清理过期数据
func (s *Scheduler) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// 清理过期任务
	for taskID, task := range s.tasks {
		if task.IsExpired() {
			delete(s.tasks, taskID)
			if s.store != nil {
				s.store.DeleteTask(taskID)
			}
		}
	}
	
	// 清理旧的执行记录
	s.executionsMu.Lock()
	s.cleanupOldExecutions()
	s.executionsMu.Unlock()
}

// cleanupOldExecutions 清理旧的执行记录
func (s *Scheduler) cleanupOldExecutions() {
	if len(s.executions) <= s.config.MaxHistorySize {
		return
	}
	
	// 按时间排序，保留最新的记录
	var execList []*TaskExecution
	for _, exec := range s.executions {
		execList = append(execList, exec)
	}
	
	sort.Slice(execList, func(i, j int) bool {
		return execList[i].StartTime.After(execList[j].StartTime)
	})
	
	// 保留最新的记录，删除旧的
	s.executions = make(map[string]*TaskExecution)
	for i := 0; i < s.config.MaxHistorySize && i < len(execList); i++ {
		s.executions[execList[i].ID] = execList[i]
	}
}

// recoverTasks 恢复任务
func (s *Scheduler) recoverTasks() error {
	if s.store == nil {
		return nil
	}

	tasks, err := s.store.LoadAllTasks()
	if err != nil {
		return err
	}

	for _, task := range tasks {
		// 重置运行状态
		if task.Status == TaskStatusRunning {
			task.Status = TaskStatusPending
		}

		s.tasks[task.ID] = task
	}

	return nil
}

// saveState 保存状态
func (s *Scheduler) saveState() error {
	if s.store == nil {
		return nil
	}

	// 保存所有任务
	for _, task := range s.tasks {
		if err := s.store.UpdateTask(task); err != nil {
			return err
		}
	}

	return nil
}

// IsRunning 检查调度器是否运行中
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetStats 获取统计信息
func (s *Scheduler) GetStats() *SchedulerStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	stats := &SchedulerStats{
		Name:        s.config.Name,
		Running:     s.running,
		WorkerCount: len(s.workers),
		UpdatedAt:   time.Now(),
	}
	
	// 计算任务统计
	for _, task := range s.tasks {
		stats.TotalTasks++
		switch task.Status {
		case TaskStatusPending:
			stats.PendingTasks++
		case TaskStatusRunning:
			stats.RunningTasks++
		case TaskStatusSuccess:
			stats.CompletedTasks++
		case TaskStatusFailed:
			stats.FailedTasks++
		case TaskStatusPaused:
			stats.PausedTasks++
		}
		
		stats.TotalExecutions += task.RunCount
		stats.TotalSuccesses += task.SuccessCount
		stats.TotalFailures += task.FailureCount
	}
	
	// 计算队列长度
	stats.QueueLength = len(s.taskChan)
	
	return stats
}

// GetExecutions 获取执行记录
func (s *Scheduler) GetExecutions(taskID string, limit int) ([]*TaskExecution, error) {
	s.executionsMu.RLock()
	defer s.executionsMu.RUnlock()
	
	var executions []*TaskExecution
	
	for _, exec := range s.executions {
		if taskID == "" || exec.TaskID == taskID {
			executions = append(executions, exec)
		}
	}
	
	// 按时间倒序排序
	sort.Slice(executions, func(i, j int) bool {
		return executions[i].StartTime.After(executions[j].StartTime)
	})
	
	// 限制数量
	if limit > 0 && len(executions) > limit {
		executions = executions[:limit]
	}
	
	return executions, nil
}

// executeHooks 执行钩子
func (s *Scheduler) executeHooks(hookType HookType, task *Task) {
	if s.hookManager == nil {
		return
	}

	hookData := map[string]interface{}{
		"task":     task,
		"scheduler": s.config.Name,
		"time":     time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	if err := s.hookManager.ExecuteHook(ctx, hookType, hookData); err != nil {
		s.logger.Warn("Hook execution failed",
			"hook_type", string(hookType),
			"task_id", task.ID,
			"error", err)
	}
}

// scheduleTask 调度任务
func (s *Scheduler) scheduleTask(task *Task) {
	select {
	case s.taskChan <- task:
	default:
		s.logger.Warn("Task queue full, dropping task", "task_id", task.ID)
	}
}

// scheduleNextRun 调度下次执行
func (s *Scheduler) scheduleNextRun(task *Task) error {
	s.calculateNextRunTime(task)

	if task.NextRunTime != nil {
		task.Status = TaskStatusPending
		return nil
	}

	return fmt.Errorf("failed to calculate next run time")
}

// simpleLogger 简单日志实现
type simpleLogger struct{}

func (l *simpleLogger) Debug(msg string, fields ...interface{}) {
	// 简化实现，可以后续集成到更复杂的日志系统
}

func (l *simpleLogger) Info(msg string, fields ...interface{}) {
	fmt.Printf("[INFO] %s", msg)
	if len(fields) > 0 {
		fmt.Printf(" %+v", fields)
	}
	fmt.Println()
}

func (l *simpleLogger) Warn(msg string, fields ...interface{}) {
	fmt.Printf("[WARN] %s", msg)
	if len(fields) > 0 {
		fmt.Printf(" %+v", fields)
	}
	fmt.Println()
}

func (l *simpleLogger) Error(msg string, fields ...interface{}) {
	fmt.Printf("[ERROR] %s", msg)
	if len(fields) > 0 {
		fmt.Printf(" %+v", fields)
	}
	fmt.Println()
}

func (l *simpleLogger) Fatal(msg string, fields ...interface{}) {
	fmt.Printf("[FATAL] %s", msg)
	if len(fields) > 0 {
		fmt.Printf(" %+v", fields)
	}
	fmt.Println()
}

func (l *simpleLogger) WithFields(fields map[string]interface{}) Logger {
	return l // 简化实现
}

func (l *simpleLogger) WithError(err error) Logger {
	return l // 简化实现
}