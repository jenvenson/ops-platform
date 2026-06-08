package tasks

import (
	"fmt"
	"sync"
	"time"
)

type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
)

type TaskResult struct {
	Success       bool     `json:"success"`
	Message       string   `json:"message"`
	CopiedJobs    []string `json:"copied_jobs"`
	FailedJobs    []string `json:"failed_jobs"`
	SkippedJobs   []string `json:"skipped_jobs"`
	ApprovedCount int      `json:"approved_count"`
	ApprovalNote  string   `json:"approval_note,omitempty"`
}

type TaskInfo struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Status    TaskStatus  `json:"status"`
	Result    *TaskResult `json:"result,omitempty"`
	Progress  int         `json:"progress"`
	Total     int         `json:"total"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type TaskManager struct {
	tasks map[string]*TaskInfo
	mutex sync.RWMutex
}

var defaultManager *TaskManager
var once sync.Once

func GetDefaultTaskManager() *TaskManager {
	once.Do(func() {
		defaultManager = &TaskManager{
			tasks: make(map[string]*TaskInfo),
		}
	})
	return defaultManager
}

func (tm *TaskManager) CreateTask(taskType string) *TaskInfo {
	taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())
	task := &TaskInfo{
		ID:        taskID,
		Type:      taskType,
		Status:    TaskPending,
		Progress:  0,
		Total:     0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	tm.tasks[taskID] = task

	return task
}

func (tm *TaskManager) UpdateTaskProgress(taskID string, progress, total int) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	if task, exists := tm.tasks[taskID]; exists {
		task.Progress = progress
		task.Total = total
		task.UpdatedAt = time.Now()
	}
}

func (tm *TaskManager) UpdateTaskStatus(taskID string, status TaskStatus, result *TaskResult) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	if task, exists := tm.tasks[taskID]; exists {
		task.Status = status
		task.Result = result
		task.UpdatedAt = time.Now()
	}
}

func (tm *TaskManager) GetTask(taskID string) (*TaskInfo, bool) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	task, exists := tm.tasks[taskID]
	return task, exists
}

func (tm *TaskManager) CleanUpOldTasks() {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	now := time.Now()
	for id, task := range tm.tasks {
		// 清理超过1小时的已完成任务
		if now.Sub(task.UpdatedAt) > time.Hour &&
			(task.Status == TaskCompleted || task.Status == TaskFailed) {
			delete(tm.tasks, id)
		}
	}
}

func (tm *TaskManager) StartCleanupRoutine() {
	go func() {
		ticker := time.NewTicker(10 * time.Minute) // 每10分钟清理一次
		defer ticker.Stop()

		for range ticker.C {
			tm.CleanUpOldTasks()
		}
	}()
}
