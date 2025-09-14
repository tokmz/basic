package scheduler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// MemoryPersistenceStore 内存持久化存储实现
type MemoryPersistenceStore struct {
	tasks       map[string]*Task
	executions  map[string]*TaskExecution
	tasksMutex  sync.RWMutex
	execMutex   sync.RWMutex
}

// NewMemoryPersistenceStore 创建内存持久化存储
func NewMemoryPersistenceStore() PersistenceStore {
	return &MemoryPersistenceStore{
		tasks:      make(map[string]*Task),
		executions: make(map[string]*TaskExecution),
	}
}

func (m *MemoryPersistenceStore) SaveTask(task *Task) error {
	m.tasksMutex.Lock()
	defer m.tasksMutex.Unlock()
	
	// 深拷贝任务以避免并发修改
	taskCopy := task.Clone()
	m.tasks[task.ID] = taskCopy
	
	return nil
}

func (m *MemoryPersistenceStore) LoadTask(id string) (*Task, error) {
	m.tasksMutex.RLock()
	defer m.tasksMutex.RUnlock()
	
	task, exists := m.tasks[id]
	if !exists {
		return nil, ErrTaskNotFound
	}
	
	return task.Clone(), nil
}

func (m *MemoryPersistenceStore) LoadAllTasks() ([]*Task, error) {
	m.tasksMutex.RLock()
	defer m.tasksMutex.RUnlock()
	
	tasks := make([]*Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		tasks = append(tasks, task.Clone())
	}
	
	return tasks, nil
}

func (m *MemoryPersistenceStore) DeleteTask(id string) error {
	m.tasksMutex.Lock()
	defer m.tasksMutex.Unlock()
	
	if _, exists := m.tasks[id]; !exists {
		return ErrTaskNotFound
	}
	
	delete(m.tasks, id)
	return nil
}

func (m *MemoryPersistenceStore) UpdateTask(task *Task) error {
	return m.SaveTask(task)
}

func (m *MemoryPersistenceStore) SaveExecution(execution *TaskExecution) error {
	m.execMutex.Lock()
	defer m.execMutex.Unlock()
	
	// 深拷贝执行记录
	execCopy := *execution
	if execution.Metadata != nil {
		execCopy.Metadata = make(map[string]string)
		for k, v := range execution.Metadata {
			execCopy.Metadata[k] = v
		}
	}
	
	m.executions[execution.ID] = &execCopy
	return nil
}

func (m *MemoryPersistenceStore) LoadExecutions(taskID string, limit int) ([]*TaskExecution, error) {
	m.execMutex.RLock()
	defer m.execMutex.RUnlock()
	
	var executions []*TaskExecution
	count := 0
	
	for _, exec := range m.executions {
		if exec.TaskID == taskID {
			executions = append(executions, exec)
			count++
			if limit > 0 && count >= limit {
				break
			}
		}
	}
	
	return executions, nil
}

func (m *MemoryPersistenceStore) LoadExecutionsByTimeRange(start, end time.Time) ([]*TaskExecution, error) {
	m.execMutex.RLock()
	defer m.execMutex.RUnlock()
	
	var executions []*TaskExecution
	
	for _, exec := range m.executions {
		if exec.StartTime.After(start) && exec.StartTime.Before(end) {
			executions = append(executions, exec)
		}
	}
	
	return executions, nil
}

func (m *MemoryPersistenceStore) DeleteExecution(id string) error {
	m.execMutex.Lock()
	defer m.execMutex.Unlock()
	
	delete(m.executions, id)
	return nil
}

func (m *MemoryPersistenceStore) CleanupExecutions(olderThan time.Time) (int64, error) {
	m.execMutex.Lock()
	defer m.execMutex.Unlock()
	
	var deleted int64
	for id, exec := range m.executions {
		if exec.StartTime.Before(olderThan) {
			delete(m.executions, id)
			deleted++
		}
	}
	
	return deleted, nil
}

func (m *MemoryPersistenceStore) Begin() (Transaction, error) {
	return &MemoryTransaction{store: m}, nil
}

func (m *MemoryPersistenceStore) Close() error {
	return nil
}

// MemoryTransaction 内存事务实现
type MemoryTransaction struct {
	store     *MemoryPersistenceStore
	tasks     map[string]*Task
	committed bool
}

func (t *MemoryTransaction) SaveTask(task *Task) error {
	if t.tasks == nil {
		t.tasks = make(map[string]*Task)
	}
	t.tasks[task.ID] = task.Clone()
	return nil
}

func (t *MemoryTransaction) UpdateTask(task *Task) error {
	return t.SaveTask(task)
}

func (t *MemoryTransaction) DeleteTask(id string) error {
	if t.tasks == nil {
		t.tasks = make(map[string]*Task)
	}
	t.tasks[id] = nil // 标记为删除
	return nil
}

func (t *MemoryTransaction) Commit() error {
	if t.committed {
		return fmt.Errorf("transaction already committed")
	}
	
	t.store.tasksMutex.Lock()
	defer t.store.tasksMutex.Unlock()
	
	for id, task := range t.tasks {
		if task == nil {
			delete(t.store.tasks, id)
		} else {
			t.store.tasks[id] = task
		}
	}
	
	t.committed = true
	return nil
}

func (t *MemoryTransaction) Rollback() error {
	t.tasks = nil
	return nil
}

// FilePersistenceStore 文件持久化存储实现
type FilePersistenceStore struct {
	baseDir string
	*MemoryPersistenceStore
}

// NewFilePersistenceStore 创建文件持久化存储
func NewFilePersistenceStore(baseDir string) (PersistenceStore, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}
	
	store := &FilePersistenceStore{
		baseDir:                   baseDir,
		MemoryPersistenceStore: &MemoryPersistenceStore{
			tasks:      make(map[string]*Task),
			executions: make(map[string]*TaskExecution),
		},
	}
	
	// 加载现有数据
	if err := store.loadFromDisk(); err != nil {
		return nil, fmt.Errorf("failed to load data from disk: %w", err)
	}
	
	return store, nil
}

func (f *FilePersistenceStore) SaveTask(task *Task) error {
	// 先保存到内存
	if err := f.MemoryPersistenceStore.SaveTask(task); err != nil {
		return err
	}
	
	// 然后保存到文件
	return f.saveTaskToDisk(task)
}

func (f *FilePersistenceStore) DeleteTask(id string) error {
	// 先从内存删除
	if err := f.MemoryPersistenceStore.DeleteTask(id); err != nil {
		return err
	}
	
	// 然后从文件删除
	taskFile := filepath.Join(f.baseDir, "tasks", id+".json")
	if err := os.Remove(taskFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete task file: %w", err)
	}
	
	return nil
}

func (f *FilePersistenceStore) SaveExecution(execution *TaskExecution) error {
	// 先保存到内存
	if err := f.MemoryPersistenceStore.SaveExecution(execution); err != nil {
		return err
	}
	
	// 然后保存到文件
	return f.saveExecutionToDisk(execution)
}

func (f *FilePersistenceStore) saveTaskToDisk(task *Task) error {
	tasksDir := filepath.Join(f.baseDir, "tasks")
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		return fmt.Errorf("failed to create tasks directory: %w", err)
	}
	
	taskFile := filepath.Join(tasksDir, task.ID+".json")
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}
	
	return os.WriteFile(taskFile, data, 0644)
}

func (f *FilePersistenceStore) saveExecutionToDisk(execution *TaskExecution) error {
	execDir := filepath.Join(f.baseDir, "executions")
	if err := os.MkdirAll(execDir, 0755); err != nil {
		return fmt.Errorf("failed to create executions directory: %w", err)
	}
	
	execFile := filepath.Join(execDir, execution.ID+".json")
	data, err := json.MarshalIndent(execution, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal execution: %w", err)
	}
	
	return os.WriteFile(execFile, data, 0644)
}

func (f *FilePersistenceStore) loadFromDisk() error {
	// 加载任务
	tasksDir := filepath.Join(f.baseDir, "tasks")
	if err := f.loadTasksFromDir(tasksDir); err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}
	
	// 加载执行记录
	execDir := filepath.Join(f.baseDir, "executions")
	if err := f.loadExecutionsFromDir(execDir); err != nil {
		return fmt.Errorf("failed to load executions: %w", err)
	}
	
	return nil
}

func (f *FilePersistenceStore) loadTasksFromDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil // 目录不存在，跳过
	}
	
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return err
	}
	
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue // 跳过无法读取的文件
		}
		
		var task Task
		if err := json.Unmarshal(data, &task); err != nil {
			continue // 跳过无法解析的文件
		}
		
		f.MemoryPersistenceStore.tasks[task.ID] = &task
	}
	
	return nil
}

func (f *FilePersistenceStore) loadExecutionsFromDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil // 目录不存在，跳过
	}
	
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return err
	}
	
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue // 跳过无法读取的文件
		}
		
		var execution TaskExecution
		if err := json.Unmarshal(data, &execution); err != nil {
			continue // 跳过无法解析的文件
		}
		
		f.MemoryPersistenceStore.executions[execution.ID] = &execution
	}
	
	return nil
}