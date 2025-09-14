package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/tokmz/basic/pkg/scheduler"
)

func main() {
	// 创建调度器配置
	config := scheduler.DefaultSchedulerConfig()
	config.Name = "demo-scheduler"
	config.MaxWorkers = 4
	config.TickInterval = time.Second
	config.EnableMetrics = true
	config.EnablePersistence = true
	config.PersistenceDriver = "file"
	config.PersistenceConfig = map[string]interface{}{
		"base_dir": "./scheduler_data",
	}

	// 创建调度器
	sched, err := scheduler.NewScheduler(config)
	if err != nil {
		log.Fatalf("创建调度器失败: %v", err)
	}

	// 启动调度器
	if err := sched.Start(); err != nil {
		log.Fatalf("启动调度器失败: %v", err)
	}
	defer sched.Stop()

	fmt.Println("调度器已启动，正在添加任务...")

	// 添加示例任务
	if err := addExampleTasks(sched); err != nil {
		log.Fatalf("添加任务失败: %v", err)
	}

	// 监听系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 定期打印状态
	statusTicker := time.NewTicker(time.Second * 5)
	defer statusTicker.Stop()

	fmt.Println("调度器运行中... (按 Ctrl+C 退出)")

	for {
		select {
		case <-sigChan:
			fmt.Println("\n收到退出信号，正在关闭调度器...")
			return

		case <-statusTicker.C:
			printSchedulerStatus(sched)
		}
	}
}

func addExampleTasks(sched *scheduler.Scheduler) error {
	var taskCounter int32

	// 1. 一次性任务
	oneTimeTask, err := scheduler.NewTaskBuilder("one-time-task").
		WithDescription("这是一个一次性任务").
		WithType(scheduler.TaskTypeOnce).
		WithPriority(scheduler.PriorityHigh).
		WithTimeout(time.Second * 30).
		WithTag("category", "demo").
		WithFunc(func(ctx context.Context, task *scheduler.Task) error {
			count := atomic.AddInt32(&taskCounter, 1)
			fmt.Printf("执行一次性任务 #%d (任务ID: %s)\n", count, task.ID)
			time.Sleep(time.Second) // 模拟工作
			return nil
		}).
		Build()

	if err != nil {
		return fmt.Errorf("创建一次性任务失败: %v", err)
	}

	if err := sched.AddTask(oneTimeTask); err != nil {
		return fmt.Errorf("添加一次性任务失败: %v", err)
	}

	// 2. 间隔任务
	intervalTask, err := scheduler.NewTaskBuilder("interval-task").
		WithDescription("每3秒执行一次").
		WithType(scheduler.TaskTypeInterval).
		WithInterval(time.Second * 3).
		WithPriority(scheduler.PriorityNormal).
		WithTag("category", "periodic").
		WithFunc(func(ctx context.Context, task *scheduler.Task) error {
			count := atomic.AddInt32(&taskCounter, 1)
			fmt.Printf("执行间隔任务 #%d (任务ID: %s)\n", count, task.ID)
			return nil
		}).
		Build()

	if err != nil {
		return fmt.Errorf("创建间隔任务失败: %v", err)
	}

	if err := sched.AddTask(intervalTask); err != nil {
		return fmt.Errorf("添加间隔任务失败: %v", err)
	}

	fmt.Printf("成功添加 2 个示例任务\n")
	return nil
}

func printSchedulerStatus(sched *scheduler.Scheduler) {
	stats := sched.GetStats()

	fmt.Printf("\n=== 调度器状态 (%s) ===\n", time.Now().Format("15:04:05"))
	fmt.Printf("运行状态: %v\n", stats.Running)
	fmt.Printf("工作者数量: %d\n", stats.WorkerCount)
	fmt.Printf("总任务数: %d\n", stats.TotalTasks)
	fmt.Printf("待执行任务: %d\n", stats.PendingTasks)
	fmt.Printf("运行中任务: %d\n", stats.RunningTasks)
	fmt.Printf("已完成任务: %d\n", stats.CompletedTasks)
	fmt.Printf("失败任务: %d\n", stats.FailedTasks)
	fmt.Printf("总执行次数: %d\n", stats.TotalExecutions)
	fmt.Printf("总成功次数: %d\n", stats.TotalSuccesses)
	fmt.Printf("总失败次数: %d\n", stats.TotalFailures)
	fmt.Printf("===========================\n")
}