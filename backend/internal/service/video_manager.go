package service

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/agnes-image-tool/backend/internal/model"
)

const (
	maxConcurrentTasks = 10
	pollInterval       = 5 * time.Second
	maxPollTime        = 30 * time.Minute
	pollRetryMax       = 10
)

// VideoTask 表示一个视频生成任务
type VideoTask struct {
	ID          string
	Status      string // queued / in_progress / completed / failed
	Progress    int
	ResultURL   string
	Error       string
	Seconds     string
	subscribers map[string]chan model.VideoEvent
	mu          sync.RWMutex
}

// TaskManager 视频任务管理器
type TaskManager struct {
	mu         sync.RWMutex
	tasks      map[string]*VideoTask
	client     *AgnesClient
	sem        chan struct{} // 并发限制信号量
	onComplete VideoCompleteFunc
}

// SetOnComplete 设置视频完成回调（供 handler 调用，用于保存历史记录等）
func (tm *TaskManager) SetOnComplete(fn VideoCompleteFunc) {
	tm.onComplete = fn
}

// NewTaskManager 创建任务管理器
func NewTaskManager(client *AgnesClient) *TaskManager {
	return &TaskManager{
		tasks:  make(map[string]*VideoTask),
		client: client,
		sem:    make(chan struct{}, maxConcurrentTasks),
	}
}

// CreateTask 创建视频任务
func (tm *TaskManager) CreateTask(videoID, prompt string, opts VideoOptions) *VideoTask {
	task := &VideoTask{
		ID:          videoID,
		Status:      "queued",
		Progress:    0,
		subscribers: make(map[string]chan model.VideoEvent),
	}

	tm.mu.Lock()
	tm.tasks[videoID] = task
	tm.mu.Unlock()

	// 启动轮询 goroutine（带并发限制）
	tm.sem <- struct{}{}
	go tm.startPolling(task, prompt, opts)

	return task
}

// GetTask 获取任务
func (tm *TaskManager) GetTask(taskID string) *VideoTask {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.tasks[taskID]
}

// Subscribe 为任务注册 SSE 订阅者
func (tm *TaskManager) Subscribe(taskID, subID string) chan model.VideoEvent {
	task := tm.GetTask(taskID)
	if task == nil {
		return nil
	}

	ch := make(chan model.VideoEvent, 10)
	task.mu.Lock()
	task.subscribers[subID] = ch
	task.mu.Unlock()
	return ch
}

// Unsubscribe 移除 SSE 订阅者
func (tm *TaskManager) Unsubscribe(taskID, subID string) {
	task := tm.GetTask(taskID)
	if task == nil {
		return
	}

	task.mu.Lock()
	if ch, ok := task.subscribers[subID]; ok {
		close(ch)
		delete(task.subscribers, subID)
	}
	task.mu.Unlock()
}

// notifySubscribers 通知所有订阅者（轮询 goroutine 调用）
func (task *VideoTask) notifySubscribers(event model.VideoEvent) {
	task.mu.RLock()
	defer task.mu.RUnlock()

	for _, ch := range task.subscribers {
		select {
		case ch <- event:
		default:
			// 如果 channel 满了，跳过以防止阻塞
		}
	}
}

// updateTask 原子更新任务状态
func (task *VideoTask) updateTask(status string, progress int, resultURL, errMsg, seconds string) {
	task.mu.Lock()
	defer task.mu.Unlock()

	task.Status = status
	task.Progress = progress
	if resultURL != "" {
		task.ResultURL = resultURL
	}
	if errMsg != "" {
		task.Error = errMsg
	}
	if seconds != "" {
		task.Seconds = seconds
	}
}

// startPolling 轮询视频生成状态（在 goroutine 中运行）
// 注意：任务已由 handler 提交到 Agnes API，此处仅轮询状态
func (tm *TaskManager) startPolling(task *VideoTask, prompt string, opts VideoOptions) {
	defer func() {
		<-tm.sem // 释放并发槽位
	}()

	log.Printf("[Task %s] 开始轮询视频生成状态", task.ID)

	task.updateTask("in_progress", 0, "", "", "")
	task.notifySubscribers(model.VideoEvent{
		Event:    "progress",
		Status:   "in_progress",
		Progress: 0,
	})

	startTime := time.Now()
	retryCount := 0

	for {
		elapsed := time.Since(startTime)
		if elapsed > maxPollTime {
			errMsg := fmt.Sprintf("视频生成超时（超过 %d 分钟）", int(maxPollTime.Minutes()))
			task.updateTask("failed", 0, "", errMsg, "")
			task.notifySubscribers(model.VideoEvent{
				Event: "error",
				Error: errMsg,
			})
			log.Printf("[Task %s] %s", task.ID, errMsg)
			tm.cleanupTask(task.ID)
			return
		}

		// 查询状态
		status, err := tm.client.CheckVideoStatus(task.ID)
		if err != nil {
			retryCount++
			if retryCount <= pollRetryMax {
				backoffSecs := 1 << uint(retryCount)
				if backoffSecs > 30 {
					backoffSecs = 30
				}
				backoff := time.Duration(backoffSecs) * time.Second
				log.Printf("[Task %s] 查询失败 (%d/%d): %v，%v后重试",
					task.ID, retryCount, pollRetryMax, err, backoff)
				time.Sleep(backoff)
				continue
			}
			errMsg := fmt.Sprintf("查询视频状态失败，已重试 %d 次: %v", pollRetryMax, err)
			task.updateTask("failed", 0, "", errMsg, "")
			task.notifySubscribers(model.VideoEvent{
				Event: "error",
				Error: errMsg,
			})
			log.Printf("[Task %s] %s", task.ID, errMsg)
			tm.cleanupTask(task.ID)
			return
		}

		retryCount = 0 // 重置重试计数

		log.Printf("[Task %s] 状态: %s | 进度: %d%% | 已等待: %d秒",
			task.ID, status.Status, status.Progress, int(elapsed.Seconds()))

		task.updateTask(status.Status, status.Progress, status.URL, extractError(status.Error), status.Seconds)
		task.notifySubscribers(model.VideoEvent{
			Event:    "progress",
			Status:   status.Status,
			Progress: status.Progress,
		})

		switch status.Status {
		case "completed":
			log.Printf("[Task %s] 视频生成完成: %s", task.ID, status.URL)
			task.notifySubscribers(model.VideoEvent{
				Event:   "complete",
				URL:     status.URL,
				Seconds: status.Seconds,
			})
			// 触发完成回调（保存历史记录等）
			if tm.onComplete != nil {
				tm.onComplete(task.ID, prompt, status.URL, opts)
			}
			return

		case "failed":
			errMsg := extractError(status.Error)
			log.Printf("[Task %s] 视频生成失败: %s", task.ID, errMsg)
			task.notifySubscribers(model.VideoEvent{
				Event: "error",
				Error: errMsg,
			})
			// 不删除任务，保留错误信息供查询
			return
		}

		// 动态调整轮询间隔
		sleepTime := pollInterval
		if status.Progress == 0 && elapsed > 2*time.Minute {
			sleepTime = pollInterval * 2
		}
		time.Sleep(sleepTime)
	}
}

// cleanupTask 清理已完成/失败的任务
func (tm *TaskManager) cleanupTask(taskID string) {
	task := tm.GetTask(taskID)
	if task == nil {
		return
	}

	task.mu.Lock()
	for id, ch := range task.subscribers {
		close(ch)
		delete(task.subscribers, id)
	}
	task.mu.Unlock()

	// 保持任务在内存中一段时间以便查询，1 小时后删除
	time.AfterFunc(1*time.Hour, func() {
		tm.mu.Lock()
		delete(tm.tasks, taskID)
		tm.mu.Unlock()
	})
}

// TaskCount 返回当前活跃任务数
func (tm *TaskManager) TaskCount() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return len(tm.tasks)
}

func extractError(e any) string {
	if e == nil {
		return ""
	}
	switch v := e.(type) {
	case string:
		return v
	case map[string]any:
		if msg, ok := v["message"]; ok {
			return fmt.Sprintf("%v", msg)
		}
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
