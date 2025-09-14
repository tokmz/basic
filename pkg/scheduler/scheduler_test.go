package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestSchedulerBasic(t *testing.T) {
	config := DefaultSchedulerConfig()
	config.MaxWorkers = 2
	config.TickInterval = time.Millisecond * 100

	scheduler, err := NewScheduler(config)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}

	// 启动调度器
	if err := scheduler.Start(); err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer scheduler.Stop()

	// 验证调度器状态
	if !scheduler.IsRunning() {
		t.Error("Scheduler should be running")
	}

	stats := scheduler.GetStats()
	if stats.WorkerCount != 2 {
		t.Errorf("Expected 2 workers, got %d", stats.WorkerCount)
	}
}

func TestTaskBuilder(t *testing.T) {
	var executed int32

	task, err := NewTaskBuilder("test-task").
		WithDescription("Test task").
		WithType(TaskTypeOnce).
		WithPriority(PriorityHigh).
		WithTimeout(time.Second * 30).
		WithRetry(3, time.Second * 5).
		WithTag("category", "test").
		WithMetadata("version", "1.0").
		WithFunc(func(ctx context.Context, task *Task) error {
			atomic.AddInt32(&executed, 1)
			return nil
		}).
		Build()

	if err != nil {
		t.Fatalf("Failed to build task: %v", err)
	}

	// 验证任务属性
	if task.Name != "test-task" {
		t.Errorf("Expected task name 'test-task', got '%s'", task.Name)
	}

	if task.Type != TaskTypeOnce {
		t.Errorf("Expected task type TaskTypeOnce, got %v", task.Type)
	}

	if task.Priority != PriorityHigh {
		t.Errorf("Expected priority PriorityHigh, got %v", task.Priority)
	}

	if task.Tags["category"] != "test" {
		t.Errorf("Expected tag category=test, got %s", task.Tags["category"])
	}
}

func TestSchedulerTaskExecution(t *testing.T) {
	config := DefaultSchedulerConfig()
	config.MaxWorkers = 1
	config.TickInterval = time.Millisecond * 50

	scheduler, err := NewScheduler(config)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}

	if err := scheduler.Start(); err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer scheduler.Stop()

	var executed int32

	task, err := NewTaskBuilder("test-execution").
		WithType(TaskTypeOnce).
		WithFunc(func(ctx context.Context, task *Task) error {
			atomic.AddInt32(&executed, 1)
			return nil
		}).
		Build()

	if err != nil {
		t.Fatalf("Failed to build task: %v", err)
	}

	// 添加任务
	if err := scheduler.AddTask(task); err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	// 等待任务执行
	time.Sleep(time.Millisecond * 200)

	if atomic.LoadInt32(&executed) != 1 {
		t.Errorf("Expected task to be executed once, got %d", atomic.LoadInt32(&executed))
	}

	// 验证任务已被移除（一次性任务）
	_, err = scheduler.GetTask(task.ID)
	if err == nil {
		t.Error("Expected task to be removed after execution")
	}
}

func TestTaskFilter(t *testing.T) {
	// 创建测试任务
	task1, _ := NewTaskBuilder("task1").
		WithType(TaskTypeOnce).
		WithTag("env", "test").
		WithFunc(func(ctx context.Context, task *Task) error { return nil }).
		Build()

	task2, _ := NewTaskBuilder("task2").
		WithType(TaskTypeInterval).
		WithInterval(time.Minute).
		WithTag("env", "prod").
		WithFunc(func(ctx context.Context, task *Task) error { return nil }).
		Build()

	// 测试过滤器
	filter := &TaskFilter{
		Types: []TaskType{TaskTypeOnce},
		Tags:  map[string]string{"env": "test"},
	}

	// 测试匹配
	if !filter.Match(task1) {
		t.Error("Task1 should match the filter")
	}

	if filter.Match(task2) {
		t.Error("Task2 should not match the filter")
	}
}

func TestCronParser(t *testing.T) {
	tests := []struct {
		expr    string
		valid   bool
		desc    string
	}{
		{"0 9 * * 1-5", true, "工作日每天9点"},
		{"*/5 * * * *", true, "每5分钟"},
		{"0 0 1 * *", true, "每月1日零点"},
		{"invalid", false, "无效表达式"},
		{"0 25 * * *", false, "无效小时"},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			_, err := parseCronExpr(test.expr)
			if test.valid && err != nil {
				t.Errorf("Expected valid cron expression '%s', got error: %v", test.expr, err)
			}
			if !test.valid && err == nil {
				t.Errorf("Expected invalid cron expression '%s', but got no error", test.expr)
			}
		})
	}
}

func TestEventBus(t *testing.T) {
	bus := NewSimpleEventBus()
	defer bus.Close()

	var received int32

	// 订阅事件
	subscription := bus.Subscribe(EventTaskAdded, func(event Event) {
		atomic.AddInt32(&received, 1)
	})

	if subscription == nil {
		t.Fatal("Failed to create subscription")
	}

	// 发布事件
	event := Event{
		Type:      EventTaskAdded,
		Timestamp: time.Now(),
		TaskID:    "test-task",
		TaskName:  "Test Task",
		Data:      map[string]interface{}{"test": true},
	}

	bus.Publish(event)

	// 等待事件处理
	time.Sleep(time.Millisecond * 100)

	if atomic.LoadInt32(&received) != 1 {
		t.Errorf("Expected 1 event received, got %d", atomic.LoadInt32(&received))
	}

	// 取消订阅
	bus.Unsubscribe(subscription)

	// 再次发布事件
	bus.Publish(event)
	time.Sleep(time.Millisecond * 100)

	// 应该还是1，因为已经取消订阅
	if atomic.LoadInt32(&received) != 1 {
		t.Errorf("Expected 1 event received after unsubscribe, got %d", atomic.LoadInt32(&received))
	}
}

func TestMetricsCollector(t *testing.T) {
	collector := NewSimpleMetricsCollector()

	// 测试计数器
	collector.Increment("test_counter", map[string]string{"label": "value"})
	collector.Increment("test_counter", map[string]string{"label": "value"})

	// 测试仪表
	collector.Gauge("test_gauge", 42.5, nil)

	// 测试直方图
	collector.Histogram("test_histogram", 1.5, nil)
	collector.Histogram("test_histogram", 2.5, nil)

	metrics := collector.GetMetrics()

	if metrics == nil {
		t.Fatal("Expected metrics to be returned")
	}

	// 验证计数器
	counters, ok := metrics["counters"].(map[string]int64)
	if !ok {
		t.Fatal("Expected counters to be map[string]int64")
	}

	if counters["test_counter_label_value"] != 2 {
		t.Errorf("Expected counter value 2, got %d", counters["test_counter_label_value"])
	}

	// 验证仪表
	gauges, ok := metrics["gauges"].(map[string]float64)
	if !ok {
		t.Fatal("Expected gauges to be map[string]float64")
	}

	if gauges["test_gauge"] != 42.5 {
		t.Errorf("Expected gauge value 42.5, got %f", gauges["test_gauge"])
	}
}

func TestPersistenceStore(t *testing.T) {
	store := NewMemoryPersistenceStore()

	// 创建测试任务
	task, _ := NewTaskBuilder("test-persistence").
		WithType(TaskTypeOnce).
		WithFunc(func(ctx context.Context, task *Task) error { return nil }).
		Build()

	// 保存任务
	if err := store.SaveTask(task); err != nil {
		t.Fatalf("Failed to save task: %v", err)
	}

	// 加载任务
	loadedTask, err := store.LoadTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to load task: %v", err)
	}

	if loadedTask.ID != task.ID {
		t.Errorf("Expected task ID %s, got %s", task.ID, loadedTask.ID)
	}

	if loadedTask.Name != task.Name {
		t.Errorf("Expected task name %s, got %s", task.Name, loadedTask.Name)
	}

	// 删除任务
	if err := store.DeleteTask(task.ID); err != nil {
		t.Fatalf("Failed to delete task: %v", err)
	}

	// 验证任务已删除
	_, err = store.LoadTask(task.ID)
	if err != ErrTaskNotFound {
		t.Errorf("Expected ErrTaskNotFound, got %v", err)
	}
}