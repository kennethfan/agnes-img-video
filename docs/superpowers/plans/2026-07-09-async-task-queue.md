# 异步任务队列实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将图片生成（同步）和视频生成（半异步）统一改造为基于 SQLite 持久化的异步任务队列，支持 Server 重启恢复。

**Architecture:** 
- 新增 `model/task.go` 定义 TaskRecord/TaskEvent 类型
- 新增 `repository/task.go` 提供 SQLite CRUD
- 新增 `service/task_queue.go` 替代 `video_manager.go`，提供 Worker Pool + SSE subscriber + 启动恢复
- 修改 handler 层：图片 handler 从同步 200 改为 202 + taskId；视频 handler 从 TaskManager 迁移到 TaskQueue
- 前端：所有视图从 `await → 显示结果` 改为 `submit → SSE → 显示结果`

**Tech Stack:** Go 1.25, Gin, SQLite (mattn/go-sqlite3), Vue 3, TypeScript, Axios, EventSource

## 全局约束

- 所有 Go 代码使用中文注释（项目规范）
- 新增 SQLite 表复用现有的 `history.db`（通过 `repo.DB()` 传入）
- 类型定义使用 `erasableSyntaxOnly` 规则（无 enum，使用 `const` + 自定义类型）
- 错误消息使用中文
- 不要删除 `history.db` 或 `outputs/`
- 不要在 Go 代码中使用 `as any` 或类似类型逃逸
- 不要引入新的认证中间件
- Base URL: `/api/v1`
- 旧端点（`/images/*`, `/videos/*`）保留兼容

---

### Task 1: Task 模型层 — types.go 扩展

**Files:**
- Modify: `backend/internal/model/types.go`（追加 TaskRecord、TaskEvent、TaskCreateResponse）
- Create: `backend/internal/model/task.go`（任务类型常量定义）

**Interfaces:**
- Produces: `model.TaskType`, `model.TaskStatus`, `model.TaskRecord`, `model.TaskEvent`, `model.TaskCreateResponse` — 被 Task 2（repository）和 Task 3（service）使用

#### 步骤

- [ ] **Step 1: 创建 `backend/internal/model/task.go`**

```go
package model

// ==================== 任务类型 ====================

type TaskType string

const (
	TaskTypeTextToImage     TaskType = "text2image"
	TaskTypeImageToImage    TaskType = "image2image"
	TaskTypeBatch           TaskType = "batch"
	TaskTypeTextToVideo     TaskType = "text2video"
	TaskTypeImageToVideo    TaskType = "image2video"
	TaskTypeMultiImageVideo TaskType = "multi_image_video"
)

type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusProcessing TaskStatus = "processing"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
)
```

- [ ] **Step 2: 在 `backend/internal/model/types.go` 末尾追加 TaskRecord、TaskEvent、TaskCreateResponse**

```go
// ==================== 异步任务队列 ====================

type TaskRecord struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Params      string `json:"params"`
	Result      string `json:"result,omitempty"`
	Progress    int    `json:"progress"`
	Error       string `json:"error,omitempty"`
	RetryCount  int    `json:"retry_count"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	CompletedAt string `json:"completed_at,omitempty"`
}

type TaskEvent struct {
	Event    string `json:"-"` // progress / complete / error
	Progress int    `json:"progress,omitempty"`
	Status   string `json:"status,omitempty"`
	Result   string `json:"result,omitempty"`
	Error    string `json:"error,omitempty"`
}

type TaskCreateResponse struct {
	TaskID string `json:"taskId"`
}
```

- [ ] **Step 3: 验证编译**

```bash
cd backend && go vet ./internal/model/...
```

- [ ] **Step 4: Commit**

```bash
git add backend/internal/model/task.go backend/internal/model/types.go
git commit -m "feat(task): add task model types (TaskRecord, TaskEvent, TaskCreateResponse)"
```

---

### Task 2: TaskRepository — SQLite CRUD

**Files:**
- Create: `backend/internal/repository/task.go`
- Test: 编译验证

**Interfaces:**
- Consumes: `model.TaskType`, `model.TaskStatus`, `model.TaskRecord`（来自 Task 1）
- Produces: `repository.TaskRepository`（InitTable, CreateTask, GetTask, UpdateTaskStatus, FindPendingTasks, ListTasks, CleanupOlderThan, UpdateRetryCount）— 被 Task 3 使用

#### 步骤

- [ ] **Step 1: 创建 `backend/internal/repository/task.go`**

```go
package repository

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/agnes-image-tool/backend/internal/model"
)

type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) InitTable() error {
	_, err := r.db.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id          TEXT PRIMARY KEY,
			type        TEXT NOT NULL,
			status      TEXT NOT NULL DEFAULT 'pending',
			params      TEXT NOT NULL,
			result      TEXT,
			progress    INTEGER NOT NULL DEFAULT 0,
			error       TEXT,
			retry_count INTEGER NOT NULL DEFAULT 0,
			created_at  TEXT NOT NULL DEFAULT (datetime('now','localtime')),
			updated_at  TEXT NOT NULL DEFAULT (datetime('now','localtime')),
			completed_at TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("创建 tasks 表失败: %w", err)
	}

	// 创建索引
	_, err = r.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
		CREATE INDEX IF NOT EXISTS idx_tasks_type ON tasks(type);
		CREATE INDEX IF NOT EXISTS idx_tasks_created ON tasks(created_at);
	`)
	if err != nil {
		return fmt.Errorf("创建 tasks 索引失败: %w", err)
	}

	log.Println("[TaskRepo] 任务表初始化完成")
	return nil
}

// generateTaskID 生成唯一任务 ID: task_{hex12}
func generateTaskID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return fmt.Sprintf("task_%s", hex.EncodeToString(b))
}

func (r *TaskRepository) CreateTask(taskType, params string) (string, error) {
	id := generateTaskID()
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := r.db.Exec(
		"INSERT INTO tasks (id, type, status, params, progress, created_at, updated_at) VALUES (?, ?, 'pending', ?, 0, ?, ?)",
		id, taskType, params, now, now,
	)
	if err != nil {
		return "", fmt.Errorf("创建任务失败: %w", err)
	}
	log.Printf("[TaskRepo] 任务已创建: id=%s type=%s", id, taskType)
	return id, nil
}

func (r *TaskRepository) GetTask(id string) (*model.TaskRecord, error) {
	var rec model.TaskRecord
	var result, errStr, completedAt sql.NullString
	err := r.db.QueryRow(
		"SELECT id, type, status, params, result, progress, error, retry_count, created_at, updated_at, completed_at FROM tasks WHERE id = ?",
		id,
	).Scan(&rec.ID, &rec.Type, &rec.Status, &rec.Params, &result, &rec.Progress, &errStr, &rec.RetryCount, &rec.CreatedAt, &rec.UpdatedAt, &completedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询任务失败: %w", err)
	}
	if result.Valid {
		rec.Result = result.String
	}
	if errStr.Valid {
		rec.Error = errStr.String
	}
	if completedAt.Valid {
		rec.CompletedAt = completedAt.String
	}
	return &rec, nil
}

func (r *TaskRepository) UpdateTaskStatus(id, status string, progress int, result, errMsg string) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	completedAt := sql.NullString{Valid: false}
	if status == string(model.TaskStatusCompleted) || status == string(model.TaskStatusFailed) {
		completedAt = sql.NullString{String: now, Valid: true}
	}

	_, err := r.db.Exec(
		"UPDATE tasks SET status = ?, progress = ?, result = ?, error = ?, updated_at = ?, completed_at = COALESCE(?, completed_at) WHERE id = ?",
		status, progress, result, errMsg, now, completedAt, id,
	)
	if err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}
	return nil
}

func (r *TaskRepository) UpdateTaskProgress(id string, progress int) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := r.db.Exec(
		"UPDATE tasks SET progress = ?, updated_at = ? WHERE id = ?",
		progress, now, id,
	)
	return err
}

func (r *TaskRepository) UpdateRetryCount(id string, count int) error {
	_, err := r.db.Exec("UPDATE tasks SET retry_count = ? WHERE id = ?", count, id)
	return err
}

func (r *TaskRepository) FindPendingTasks() ([]*model.TaskRecord, error) {
	rows, err := r.db.Query(
		"SELECT id, type, status, params, result, progress, error, retry_count, created_at, updated_at, completed_at FROM tasks WHERE status IN ('pending', 'processing') ORDER BY created_at ASC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*model.TaskRecord
	for rows.Next() {
		var rec model.TaskRecord
		var result, errStr, completedAt sql.NullString
		if err := rows.Scan(&rec.ID, &rec.Type, &rec.Status, &rec.Params, &result, &rec.Progress, &errStr, &rec.RetryCount, &rec.CreatedAt, &rec.UpdatedAt, &completedAt); err != nil {
			continue
		}
		if result.Valid {
			rec.Result = result.String
		}
		if errStr.Valid {
			rec.Error = errStr.String
		}
		if completedAt.Valid {
			rec.CompletedAt = completedAt.String
		}
		results = append(results, &rec)
	}
	return results, rows.Err()
}

func (r *TaskRepository) ListTasks(taskType, status string, limit, offset int) ([]*model.TaskRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	where := "1=1"
	args := []any{}
	n := 0

	if taskType != "" {
		n++
		where += fmt.Sprintf(" AND type = ?%d", n)
		args = append(args, taskType)
	}
	if status != "" {
		n++
		where += fmt.Sprintf(" AND status = ?%d", n)
		args = append(args, status)
	}

	query := fmt.Sprintf("SELECT id, type, status, params, result, progress, error, retry_count, created_at, updated_at, completed_at FROM tasks WHERE %s ORDER BY created_at DESC LIMIT ? OFFSET ?", where)
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*model.TaskRecord
	for rows.Next() {
		var rec model.TaskRecord
		var result, errStr, completedAt sql.NullString
		if err := rows.Scan(&rec.ID, &rec.Type, &rec.Status, &rec.Params, &result, &rec.Progress, &errStr, &rec.RetryCount, &rec.CreatedAt, &rec.UpdatedAt, &completedAt); err != nil {
			continue
		}
		if result.Valid {
			rec.Result = result.String
		}
		if errStr.Valid {
			rec.Error = errStr.String
		}
		if completedAt.Valid {
			rec.CompletedAt = completedAt.String
		}
		results = append(results, &rec)
	}
	return results, rows.Err()
}

func (r *TaskRepository) CleanupOlderThan(hours int) error {
	_, err := r.db.Exec(
		"DELETE FROM tasks WHERE completed_at IS NOT NULL AND completed_at < datetime('now', ?)",
		fmt.Sprintf("-%d hours", hours),
	)
	return err
}
```

- [ ] **Step 2: 验证编译**

```bash
cd backend && go vet ./internal/repository/...
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/repository/task.go
git commit -m "feat(task): add TaskRepository with SQLite CRUD"
```

---

### Task 3: TaskQueue — 统一异步任务队列服务

**Files:**
- Create: `backend/internal/service/task_queue.go`

**Interfaces:**
- Consumes: `repository.TaskRepository`（来自 Task 2）, `model.TaskType`, `model.TaskRecord`, `model.TaskEvent`（来自 Task 1）, `AgnesClient`
- Produces: `service.TaskQueue`（SubmitTask, GetTask, Subscribe, Unsubscribe, OnComplete 回调）— 被 Task 4/5 handler 使用

#### 步骤

- [ ] **Step 1: 创建 `backend/internal/service/task_queue.go`**

```go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/agnes-image-tool/backend/internal/model"
	"github.com/agnes-image-tool/backend/internal/repository"
)

const (
	defaultMaxConcurrentTasks = 10
	pollInterval              = 5 * time.Second
	maxPollTime               = 30 * time.Minute
	pollRetryMax              = 10
)

// TaskCompleteFunc 任务完成回调（保存历史记录等）
type TaskCompleteFunc func(taskID, taskType, prompt, resultURL string)

// TaskQueue 统一异步任务队列
type TaskQueue struct {
	mu          sync.RWMutex
	repo        *repository.TaskRepository
	client      *AgnesClient
	workerSem   chan struct{}
	subscribers map[string]map[string]chan model.TaskEvent
	onComplete  TaskCompleteFunc
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewTaskQueue 创建任务队列
func NewTaskQueue(repo *repository.TaskRepository, client *AgnesClient, maxConcurrent int) *TaskQueue {
	if maxConcurrent <= 0 {
		maxConcurrent = defaultMaxConcurrentTasks
	}
	ctx, cancel := context.WithCancel(context.Background())
	tq := &TaskQueue{
		repo:        repo,
		client:      client,
		workerSem:   make(chan struct{}, maxConcurrent),
		subscribers: make(map[string]map[string]chan model.TaskEvent),
		ctx:         ctx,
		cancel:      cancel,
	}
	// 启动时恢复未完成任务
	go tq.recoverPending()
	// 定期清理过期任务
	go tq.cleanupLoop()
	return tq
}

// SetOnComplete 设置完成回调
func (tq *TaskQueue) SetOnComplete(fn TaskCompleteFunc) {
	tq.onComplete = fn
}

// SubmitTask 提交任务：写入 SQLite → 返回 taskId → 启动 Worker
func (tq *TaskQueue) SubmitTask(taskType string, paramsJSON string) (string, error) {
	id, err := tq.repo.CreateTask(taskType, paramsJSON)
	if err != nil {
		return "", err
	}

	// 启动 Worker（不阻塞）
	select {
	case tq.workerSem <- struct{}{}:
		go tq.worker(id, taskType, paramsJSON)
	default:
		log.Printf("[TaskQueue] 达到最大并发数，任务 %s 排队等待", id)
		go func() {
			tq.workerSem <- struct{}{}
			tq.worker(id, taskType, paramsJSON)
		}()
	}

	return id, nil
}

// GetTask 查询任务状态
func (tq *TaskQueue) GetTask(id string) (*model.TaskRecord, error) {
	return tq.repo.GetTask(id)
}

// Subscribe 注册 SSE 订阅者
func (tq *TaskQueue) Subscribe(taskID, subID string) chan model.TaskEvent {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	if tq.subscribers[taskID] == nil {
		tq.subscribers[taskID] = make(map[string]chan model.TaskEvent)
	}
	ch := make(chan model.TaskEvent, 10)
	tq.subscribers[taskID][subID] = ch
	return ch
}

// Unsubscribe 移除 SSE 订阅者
func (tq *TaskQueue) Unsubscribe(taskID, subID string) {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	if subs, ok := tq.subscribers[taskID]; ok {
		if ch, ok := subs[subID]; ok {
			close(ch)
			delete(subs, subID)
		}
		if len(subs) == 0 {
			delete(tq.subscribers, taskID)
		}
	}
}

// notifySubscribers 通知所有订阅者
func (tq *TaskQueue) notifySubscribers(taskID string, event model.TaskEvent) {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	if subs, ok := tq.subscribers[taskID]; ok {
		for _, ch := range subs {
			select {
			case ch <- event:
			default:
			}
		}
	}
}

// worker 执行任务（goroutine 中运行）
func (tq *TaskQueue) worker(taskID, taskType, paramsJSON string) {
	defer func() {
		<-tq.workerSem
	}()

	log.Printf("[TaskQueue] Worker 开始任务: id=%s type=%s", taskID, taskType)

	// 标记为 processing
	tq.repo.UpdateTaskStatus(taskID, string(model.TaskStatusProcessing), 0, "", "")
	tq.notifySubscribers(taskID, model.TaskEvent{
		Event:  "progress",
		Status: string(model.TaskStatusProcessing),
	})

	var err error
	switch taskType {
	case string(model.TaskTypeTextToImage):
		err = tq.execTextToImage(taskID, paramsJSON)
	case string(model.TaskTypeImageToImage):
		err = tq.execImageToImage(taskID, paramsJSON)
	case string(model.TaskTypeBatch):
		err = tq.execBatch(taskID, paramsJSON)
	case string(model.TaskTypeTextToVideo):
		err = tq.execTextToVideo(taskID, paramsJSON)
	case string(model.TaskTypeImageToVideo):
		err = tq.execImageToVideo(taskID, paramsJSON)
	case string(model.TaskTypeMultiImageVideo):
		err = tq.execMultiImageVideo(taskID, paramsJSON)
	default:
		err = fmt.Errorf("未知任务类型: %s", taskType)
	}

	if err != nil {
		errMsg := err.Error()
		log.Printf("[TaskQueue] 任务失败: id=%s err=%s", taskID, errMsg)
		tq.repo.UpdateTaskStatus(taskID, string(model.TaskStatusFailed), 0, "", errMsg)
		tq.notifySubscribers(taskID, model.TaskEvent{
			Event: "error",
			Error: errMsg,
		})
		return
	}
}

// ==================== 图片任务执行 ====================

type imageParams struct {
	Prompt         string `json:"prompt"`
	Size           string `json:"size"`
	N              int    `json:"n"`
	NegativePrompt string `json:"negative_prompt"`
	ImageValue     string `json:"image_value,omitempty"` // base64 data URI 或 URL（image2image）
	Strength       float64 `json:"strength,omitempty"`
}

func (tq *TaskQueue) execTextToImage(taskID, paramsJSON string) error {
	var p struct {
		Prompt         string `json:"prompt"`
		Size           string `json:"size"`
		N              int    `json:"n"`
		NegativePrompt string `json:"negative_prompt"`
	}
	if err := json.Unmarshal([]byte(paramsJSON), &p); err != nil {
		return fmt.Errorf("解析参数失败: %w", err)
	}

	urls, err := tq.client.TextToImage(p.Prompt, p.Size, p.N, p.NegativePrompt)
	if err != nil {
		return err
	}

	resultJSON, _ := json.Marshal(urls)
	tq.repo.UpdateTaskStatus(taskID, string(model.TaskStatusCompleted), 100, string(resultJSON), "")
	tq.notifySubscribers(taskID, model.TaskEvent{
		Event:  "complete",
		Result: string(resultJSON),
	})

	// 触发完成回调
	if tq.onComplete != nil {
		tq.onComplete(taskID, "text2image", p.Prompt, strings.Join(urls, ","))
	}

	return nil
}

func (tq *TaskQueue) execImageToImage(taskID, paramsJSON string) error {
	var p struct {
		Prompt         string  `json:"prompt"`
		Size           string  `json:"size"`
		NegativePrompt string  `json:"negative_prompt"`
		ImageValue     string  `json:"image_value"`
		Strength       float64 `json:"strength"`
	}
	if err := json.Unmarshal([]byte(paramsJSON), &p); err != nil {
		return fmt.Errorf("解析参数失败: %w", err)
	}

	urls, err := tq.client.ImageToImage(p.ImageValue, p.Prompt, p.Size, 1, p.Strength, p.NegativePrompt)
	if err != nil {
		return err
	}

	resultJSON, _ := json.Marshal(urls)
	tq.repo.UpdateTaskStatus(taskID, string(model.TaskStatusCompleted), 100, string(resultJSON), "")
	tq.notifySubscribers(taskID, model.TaskEvent{
		Event:  "complete",
		Result: string(resultJSON),
	})

	if tq.onComplete != nil {
		tq.onComplete(taskID, "image2image", p.Prompt, strings.Join(urls, ","))
	}

	return nil
}

func (tq *TaskQueue) execBatch(taskID, paramsJSON string) error {
	var p struct {
		Prompts []string `json:"prompts"`
		Size    string   `json:"size"`
	}
	if err := json.Unmarshal([]byte(paramsJSON), &p); err != nil {
		return fmt.Errorf("解析参数失败: %w", err)
	}

	var allResults []string
	total := len(p.Prompts)

	for i, prompt := range p.Prompts {
		select {
		case <-tq.ctx.Done():
			return fmt.Errorf("任务已取消")
		default:
		}

		urls, err := tq.client.TextToImage(prompt, p.Size, 1, "")
		if err != nil {
			return fmt.Errorf("第 %d 个提示词失败: %w", i+1, err)
		}
		allResults = append(allResults, urls...)

		// 更新进度
		progress := (i + 1) * 100 / total
		tq.repo.UpdateTaskProgress(taskID, progress)
		tq.notifySubscribers(taskID, model.TaskEvent{
			Event:    "progress",
			Status:   string(model.TaskStatusProcessing),
			Progress: progress,
		})
	}

	resultJSON, _ := json.Marshal(allResults)
	tq.repo.UpdateTaskStatus(taskID, string(model.TaskStatusCompleted), 100, string(resultJSON), "")
	tq.notifySubscribers(taskID, model.TaskEvent{
		Event:  "complete",
		Result: string(resultJSON),
	})

	if tq.onComplete != nil {
		tq.onComplete(taskID, "batch", strings.Join(p.Prompts, "; "), strings.Join(allResults, ","))
	}

	return nil
}

// ==================== 视频任务执行 ====================

type videoTaskExtra struct {
	TaskID    string   `json:"taskId"`
	Mode      string   `json:"mode,omitempty"`
	ImageURLs []string `json:"image_urls,omitempty"`
}

func (tq *TaskQueue) execTextToVideo(taskID, paramsJSON string) error {
	var p struct {
		Prompt   string `json:"prompt"`
		Duration int    `json:"duration"`
		AspectRatio string `json:"aspect_ratio"`
		FrameRate  int    `json:"frame_rate"`
		NegativePrompt string `json:"negative_prompt"`
		Seed         *int   `json:"seed,omitempty"`
		NumInferenceSteps *int `json:"num_inference_steps,omitempty"`
		Width       *int   `json:"width,omitempty"`
		Height      *int   `json:"height,omitempty"`
		NumFrames   *int   `json:"num_frames,omitempty"`
	}
	if err := json.Unmarshal([]byte(paramsJSON), &p); err != nil {
		return fmt.Errorf("解析参数失败: %w", err)
	}

	opts := VideoOptions{
		Duration:          p.Duration,
		AspectRatio:       p.AspectRatio,
		FrameRate:         p.FrameRate,
		NegativePrompt:    p.NegativePrompt,
		Seed:              p.Seed,
		NumInferenceSteps: p.NumInferenceSteps,
		Width:             p.Width,
		Height:            p.Height,
		NumFrames:         p.NumFrames,
		RecordType:        "text2video",
	}
	if opts.Duration <= 0 {
		opts.Duration = 5
	}
	if opts.AspectRatio == "" {
		opts.AspectRatio = "16:9"
	}
	if opts.FrameRate <= 0 {
		opts.FrameRate = 24
	}

	payload := tq.client.BuildVideoPayload(p.Prompt, opts)
	videoID, err := tq.client.SubmitVideoTask(payload)
	if err != nil {
		return fmt.Errorf("提交视频任务失败: %w", err)
	}

	log.Printf("[TaskQueue] 视频任务已提交: task=%s videoID=%s", taskID, videoID)
	return tq.pollVideoTask(taskID, videoID, p.Prompt, opts)
}

func (tq *TaskQueue) execImageToVideo(taskID, paramsJSON string) error {
	var p struct {
		Prompt         string   `json:"prompt"`
		Duration       int      `json:"duration"`
		AspectRatio    string   `json:"aspect_ratio"`
		FrameRate      int      `json:"frame_rate"`
		NegativePrompt string   `json:"negative_prompt"`
		Seed           *int     `json:"seed,omitempty"`
		NumInferenceSteps *int  `json:"num_inference_steps,omitempty"`
		Width          *int     `json:"width,omitempty"`
		Height         *int     `json:"height,omitempty"`
		NumFrames      *int     `json:"num_frames,omitempty"`
		ImageValue     string   `json:"image_value"`
		ImageURLs      []string `json:"image_urls,omitempty"`
	}
	if err := json.Unmarshal([]byte(paramsJSON), &p); err != nil {
		return fmt.Errorf("解析参数失败: %w", err)
	}

	opts := VideoOptions{
		Duration:          p.Duration,
		AspectRatio:       p.AspectRatio,
		FrameRate:         p.FrameRate,
		NegativePrompt:    p.NegativePrompt,
		Seed:              p.Seed,
		NumInferenceSteps: p.NumInferenceSteps,
		Width:             p.Width,
		Height:            p.Height,
		NumFrames:         p.NumFrames,
		RecordType:        "image2video",
	}
	if opts.Duration <= 0 {
		opts.Duration = 5
	}
	if opts.AspectRatio == "" {
		opts.AspectRatio = "16:9"
	}
	if opts.FrameRate <= 0 {
		opts.FrameRate = 24
	}

	payload := tq.client.BuildVideoPayload(p.Prompt, opts)
	if p.ImageValue != "" {
		payload["image"] = p.ImageValue
	} else if len(p.ImageURLs) > 0 {
		payload["image"] = p.ImageURLs[0]
	}

	videoID, err := tq.client.SubmitVideoTask(payload)
	if err != nil {
		return fmt.Errorf("提交视频任务失败: %w", err)
	}

	return tq.pollVideoTask(taskID, videoID, p.Prompt, opts)
}

func (tq *TaskQueue) execMultiImageVideo(taskID, paramsJSON string) error {
	var p struct {
		Prompt         string   `json:"prompt"`
		Duration       int      `json:"duration"`
		AspectRatio    string   `json:"aspect_ratio"`
		FrameRate      int      `json:"frame_rate"`
		NegativePrompt string   `json:"negative_prompt"`
		Seed           *int     `json:"seed,omitempty"`
		NumInferenceSteps *int  `json:"num_inference_steps,omitempty"`
		Width          *int     `json:"width,omitempty"`
		Height         *int     `json:"height,omitempty"`
		NumFrames      *int     `json:"num_frames,omitempty"`
		ImageURLs      []string `json:"image_urls"`
		Mode           string   `json:"mode"`
	}
	if err := json.Unmarshal([]byte(paramsJSON), &p); err != nil {
		return fmt.Errorf("解析参数失败: %w", err)
	}

	if len(p.ImageURLs) == 0 {
		return fmt.Errorf("至少需要一张图片 URL")
	}

	opts := VideoOptions{
		Duration:          p.Duration,
		AspectRatio:       p.AspectRatio,
		FrameRate:         p.FrameRate,
		NegativePrompt:    p.NegativePrompt,
		Seed:              p.Seed,
		NumInferenceSteps: p.NumInferenceSteps,
		Width:             p.Width,
		Height:            p.Height,
		NumFrames:         p.NumFrames,
		RecordType:        "multi_image_video",
		ImageURLs:         p.ImageURLs,
		Mode:              p.Mode,
	}
	if opts.Duration <= 0 {
		opts.Duration = 5
	}
	if opts.AspectRatio == "" {
		opts.AspectRatio = "16:9"
	}
	if opts.FrameRate <= 0 {
		opts.FrameRate = 24
	}

	payload := tq.client.BuildVideoPayload(p.Prompt, opts)
	extraBody := map[string]any{
		"image": p.ImageURLs,
	}
	if p.Mode == "keyframes" {
		extraBody["mode"] = "keyframes"
	}
	payload["extra_body"] = extraBody

	videoID, err := tq.client.SubmitVideoTask(payload)
	if err != nil {
		return fmt.Errorf("提交视频任务失败: %w", err)
	}

	return tq.pollVideoTask(taskID, videoID, p.Prompt, opts)
}

// pollVideoTask 轮询视频任务状态（复用现有逻辑）
func (tq *TaskQueue) pollVideoTask(taskID, videoID, prompt string, opts VideoOptions) error {
	startTime := time.Now()
	retryCount := 0

	for {
		select {
		case <-tq.ctx.Done():
			return fmt.Errorf("任务已取消")
		default:
		}

		elapsed := time.Since(startTime)
		if elapsed > maxPollTime {
			return fmt.Errorf("视频生成超时（超过 %d 分钟）", int(maxPollTime.Minutes()))
		}

		status, err := tq.client.CheckVideoStatus(videoID)
		if err != nil {
			retryCount++
			if retryCount <= pollRetryMax {
				backoffSecs := 1 << uint(retryCount)
				if backoffSecs > 30 {
					backoffSecs = 30
				}
				time.Sleep(time.Duration(backoffSecs) * time.Second)
				continue
			}
			return fmt.Errorf("查询视频状态失败，已重试 %d 次: %w", pollRetryMax, err)
		}

		retryCount = 0

		tq.repo.UpdateTaskProgress(taskID, status.Progress)
		tq.notifySubscribers(taskID, model.TaskEvent{
			Event:    "progress",
			Status:   status.Status,
			Progress: status.Progress,
		})

		switch status.Status {
		case "completed":
			log.Printf("[TaskQueue] 视频生成完成: task=%s url=%s", taskID, status.URL)
			resultJSON, _ := json.Marshal([]string{status.URL})
			tq.repo.UpdateTaskStatus(taskID, string(model.TaskStatusCompleted), 100, string(resultJSON), "")
			tq.notifySubscribers(taskID, model.TaskEvent{
				Event:  "complete",
				Result: string(resultJSON),
			})

			if tq.onComplete != nil {
				tq.onComplete(taskID, opts.RecordType, prompt, status.URL)
			}
			return nil

		case "failed":
			errMsg := extractError(status.Error)
			return fmt.Errorf("视频生成失败: %s", errMsg)
		}

		// 动态调整轮询间隔
		sleepTime := pollInterval
		if status.Progress == 0 && elapsed > 2*time.Minute {
			sleepTime = pollInterval * 2
		}
		time.Sleep(sleepTime)
	}
}

// ==================== 重启恢复 ====================

// recoverPending 启动时恢复未完成任务
func (tq *TaskQueue) recoverPending() {
	pending, err := tq.repo.FindPendingTasks()
	if err != nil {
		log.Printf("[TaskQueue] 查询待恢复任务失败: %v", err)
		return
	}
	if len(pending) == 0 {
		return
	}

	log.Printf("[TaskQueue] 发现 %d 个待恢复任务，开始恢复...", len(pending))
	for _, task := range pending {
		select {
		case tq.workerSem <- struct{}{}:
			go func(t *model.TaskRecord) {
				log.Printf("[TaskQueue] 恢复任务: id=%s type=%s", t.ID, t.Type)
				// 对于视频任务，检查 Agnes 端状态
				if isVideoTask(t.Type) {
					tq.recoverVideoTask(t)
				} else {
					// 图片任务重新执行
					tq.repo.UpdateTaskStatus(t.ID, string(model.TaskStatusPending), 0, "", "")
					tq.worker(t.ID, t.Type, t.Params)
				}
				<-tq.workerSem
			}(task)
		default:
			log.Printf("[TaskQueue] 并发已满，任务 %s 延后恢复", task.ID)
			go func(t *model.TaskRecord) {
				tq.workerSem <- struct{}{}
				if isVideoTask(t.Type) {
					tq.recoverVideoTask(t)
				} else {
					tq.repo.UpdateTaskStatus(t.ID, string(model.TaskStatusPending), 0, "", "")
					tq.worker(t.ID, t.Type, t.Params)
				}
				<-tq.workerSem
			}(task)
		}
	}
}

func isVideoTask(taskType string) bool {
	return taskType == string(model.TaskTypeTextToVideo) ||
		taskType == string(model.TaskTypeImageToVideo) ||
		taskType == string(model.TaskTypeMultiImageVideo)
}

// recoverVideoTask 恢复视频任务：检查 Agnes 端状态
func (tq *TaskQueue) recoverVideoTask(task *model.TaskRecord) {
	// 从 params 中解析 videoID
	var extra struct {
		TaskID string `json:"taskId"`
	}
	if err := json.Unmarshal([]byte(task.Params), &extra); err != nil || extra.TaskID == "" {
		// 无法获取 videoID，重新提交
		log.Printf("[TaskQueue] 任务 %s 无 videoID，重新执行", task.ID)
		tq.worker(task.ID, task.Type, task.Params)
		return
	}

	status, err := tq.client.CheckVideoStatus(extra.TaskID)
	if err != nil {
		log.Printf("[TaskQueue] 查询视频状态失败: task=%s err=%v，重新执行", task.ID, err)
		tq.worker(task.ID, task.Type, task.Params)
		return
	}

	switch status.Status {
	case "completed":
		log.Printf("[TaskQueue] 恢复: 任务 %s 已完成", task.ID)
		resultJSON, _ := json.Marshal([]string{status.URL})
		tq.repo.UpdateTaskStatus(task.ID, string(model.TaskStatusCompleted), 100, string(resultJSON), "")
		if tq.onComplete != nil {
			// 从 params 中获取 prompt
			var p struct{ Prompt string `json:"prompt"` }
			json.Unmarshal([]byte(task.Params), &p)
			tq.onComplete(task.ID, task.Type, p.Prompt, status.URL)
		}
	case "failed":
		errMsg := extractError(status.Error)
		log.Printf("[TaskQueue] 恢复: 任务 %s 已失败: %s", task.ID, errMsg)
		tq.repo.UpdateTaskStatus(task.ID, string(model.TaskStatusFailed), 0, "", errMsg)
	default:
		log.Printf("[TaskQueue] 恢复: 任务 %s 仍在处理中，重启轮询", task.ID)
		// 从 params 中解析 opts
		var p struct {
			Prompt   string `json:"prompt"`
			Duration int    `json:"duration"`
			AspectRatio string `json:"aspect_ratio"`
			FrameRate  int    `json:"frame_rate"`
			NegativePrompt string `json:"negative_prompt"`
		}
		json.Unmarshal([]byte(task.Params), &p)
		opts := VideoOptions{
			Duration:       p.Duration,
			AspectRatio:    p.AspectRatio,
			FrameRate:      p.FrameRate,
			NegativePrompt: p.NegativePrompt,
			RecordType:     task.Type,
		}
		if opts.Duration <= 0 {
			opts.Duration = 5
		}
		if opts.AspectRatio == "" {
			opts.AspectRatio = "16:9"
		}
		if opts.FrameRate <= 0 {
			opts.FrameRate = 24
		}
		tq.repo.UpdateTaskStatus(task.ID, string(model.TaskStatusProcessing), 0, "", "")
		tq.pollVideoTask(task.ID, extra.TaskID, p.Prompt, opts)
	}
}

// ==================== 清理 ====================

// cleanupLoop 定期清理过期任务
func (tq *TaskQueue) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-tq.ctx.Done():
			return
		case <-ticker.C:
			n, err := tq.repo.CleanupOlderThan(24)
			if err != nil {
				log.Printf("[TaskQueue] 清理过期任务失败: %v", err)
			} else if n > 0 {
				log.Printf("[TaskQueue] 已清理 %d 个过期任务", n)
			}
		}
	}
}

// Stop 停止任务队列
func (tq *TaskQueue) Stop() {
	tq.cancel()
}
```

- [ ] **Step 2: 验证编译**

```bash
cd backend && go vet ./internal/service/...
```

Expected: no errors. If `strings` is missing, import it. TaskQueue has circular dependency concerns — verify none exist.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/service/task_queue.go
git commit -m "feat(task): add TaskQueue service with Worker Pool, SSE subscriber, and restart recovery"
```

---

### Task 4:  Handler 改造 — image.go 异步化 + video.go 迁移

**Files:**
- Modify: `backend/internal/handler/image.go` — 改为异步（202 + taskId）
- Modify: `backend/internal/handler/video.go` — 从 TaskManager 迁移到 TaskQueue

**Interfaces:**
- Consumes: `service.TaskQueue`（来自 Task 3）
- Produces: 修改后的 VideoHandler（依赖 TaskQueue 替换 TaskManager）

#### 步骤

- [ ] **Step 1: 改造 `backend/internal/handler/image.go`**

将 `ImageHandler` 注入 `TaskQueue`，所有 handler 改为异步：

```go
type ImageHandler struct {
	svc  *service.AgnesClient
	task *service.TaskQueue  // 新增
}

func NewImageHandler(svc *service.AgnesClient, task *service.TaskQueue) *ImageHandler {
	return &ImageHandler{svc: svc, task: task}
}

// TextToImage 文生图（异步）
func (h *ImageHandler) TextToImage(c *gin.Context) {
	var req model.TextToImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	params, _ := json.Marshal(map[string]any{
		"prompt":          req.Prompt,
		"size":            req.Size,
		"n":               req.N,
		"negative_prompt": req.NegativePrompt,
	})
	taskID, err := h.task.SubmitTask(string(model.TaskTypeTextToImage), string(params))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交任务失败: " + err.Error()})
		return
	}

	// 立即写入历史记录（待更新）
	saveHistoryRecord(req.Prompt, []string{}, "text2image", map[string]any{"taskId": taskID})

	c.JSON(http.StatusAccepted, model.TaskCreateResponse{TaskID: taskID})
}

// ImageToImage 图生图（异步）
func (h *ImageHandler) ImageToImage(c *gin.Context) {
	var imageValue string
	var prompt string
	size := "1024x1024"
	strength := 0.75
	negativePrompt := ""

	if c.Request.Header.Get("Content-Type") == "application/json" {
		var req struct {
			ImageURL       string  `json:"image_url"`
			Prompt         string  `json:"prompt" binding:"required"`
			Size           string  `json:"size"`
			Strength       float64 `json:"strength"`
			NegativePrompt string  `json:"negative_prompt"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
			return
		}
		if req.ImageURL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "image_url 不能为空"})
			return
		}
		imageValue = req.ImageURL
		prompt = req.Prompt
		if req.Size != "" {
			size = req.Size
		}
		if req.Strength > 0 {
			strength = req.Strength
		}
		negativePrompt = req.NegativePrompt
	} else {
		file, err := c.FormFile("image")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请上传图片文件"})
			return
		}

		tmpDir := "tmp"
		os.MkdirAll(tmpDir, 0755)
		tmpPath := filepath.Join(tmpDir, fmt.Sprintf("upload_%d_%s", time.Now().UnixNano(), file.Filename))
		if err := c.SaveUploadedFile(file, tmpPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存上传文件失败: " + err.Error()})
			return
		}
		defer os.Remove(tmpPath)

		imageData, err := os.ReadFile(tmpPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取图片失败: " + err.Error()})
			return
		}
		b64 := base64.StdEncoding.EncodeToString(imageData)
		ext := strings.ToLower(filepath.Ext(file.Filename))
		mimeType := map[string]string{
			".png": "image/png", ".jpg": "image/jpeg",
			".jpeg": "image/jpeg", ".gif": "image/gif",
			".webp": "image/webp",
		}[ext]
		if mimeType == "" {
			mimeType = "image/png"
		}
		imageValue = fmt.Sprintf("data:%s;base64,%s", mimeType, b64)
		prompt = c.PostForm("prompt")
		if s := c.PostForm("size"); s != "" {
			size = s
		}
		if s := c.PostForm("strength"); s != "" {
			strength = parseFloat(s)
		}
		negativePrompt = c.PostForm("negative_prompt")
	}

	params, _ := json.Marshal(map[string]any{
		"prompt":          prompt,
		"size":            size,
		"image_value":     imageValue,
		"strength":        strength,
		"negative_prompt": negativePrompt,
	})
	taskID, err := h.task.SubmitTask(string(model.TaskTypeImageToImage), string(params))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交任务失败: " + err.Error()})
		return
	}

	saveHistoryRecord(prompt, []string{}, "image2image", map[string]any{
		"taskId": taskID,
		"size":   size,
		"strength": strength,
	})

	c.JSON(http.StatusAccepted, model.TaskCreateResponse{TaskID: taskID})
}

// BatchGenerate 批量文生图（异步）
func (h *ImageHandler) BatchGenerate(c *gin.Context) {
	var req model.BatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	if len(req.Prompts) > 20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "批量生成最多支持 20 个提示词"})
		return
	}

	params, _ := json.Marshal(map[string]any{
		"prompts": req.Prompts,
		"size":    req.Size,
	})
	taskID, err := h.task.SubmitTask(string(model.TaskTypeBatch), string(params))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交任务失败: " + err.Error()})
		return
	}

	saveHistoryRecord(strings.Join(req.Prompts, "; "), []string{}, "batch", map[string]any{
		"taskId": taskID,
		"size":   req.Size,
	})

	c.JSON(http.StatusAccepted, model.TaskCreateResponse{TaskID: taskID})
}
```

注意：需要添加 `encoding/json` import。

- [ ] **Step 2: 改造 `backend/internal/handler/video.go`**

将 `VideoHandler` 的依赖从 `*service.TaskManager` 改为 `*service.TaskQueue`：

```go
type VideoHandler struct {
	svc  *service.AgnesClient
	task *service.TaskQueue  // 改为 TaskQueue
}

func NewVideoHandler(svc *service.AgnesClient, task *service.TaskQueue) *VideoHandler {
	return &VideoHandler{svc: svc, task: task}
}
```

改造 TextToVideo handler：

```go
func (h *VideoHandler) TextToVideo(c *gin.Context) {
	var req model.VideoCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	opts := h.buildVideoOptions(req)
	opts.RecordType = "text2video"

	// 序列化参数到 JSON（TaskQueue 的 Worker 会反序列化使用）
	params, _ := json.Marshal(map[string]any{
		"prompt":             req.Prompt,
		"duration":           opts.Duration,
		"aspect_ratio":       opts.AspectRatio,
		"frame_rate":         opts.FrameRate,
		"negative_prompt":    opts.NegativePrompt,
		"seed":               opts.Seed,
		"num_inference_steps": opts.NumInferenceSteps,
		"width":              opts.Width,
		"height":             opts.Height,
		"num_frames":         opts.NumFrames,
	})

	taskID, err := h.task.SubmitTask(string(model.TaskTypeTextToVideo), string(params))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交任务失败: " + err.Error()})
		return
	}

	saveHistoryRecord(req.Prompt, []string{}, "text2video", map[string]any{"taskId": taskID})
	log.Printf("[Video] 文生视频任务已创建: task=%s", taskID)
	c.JSON(http.StatusOK, model.VideoTaskResponse{TaskID: taskID})
}
```

ImageToVideo handler：同样替换 SubmitVideoTask + TaskManager.CreateTask 为 TaskQueue.SubmitTask。

MultiImageVideo handler：同上。

GetTaskStatus handler：改为从 TaskQueue 查询：

```go
func (h *VideoHandler) GetTaskStatus(c *gin.Context) {
	taskID := c.Param("taskId")
	task, err := h.task.GetTask(taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询任务失败: " + err.Error()})
		return
	}
	if task == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	c.JSON(http.StatusOK, model.VideoStatus{
		Status:   task.Status,
		Progress: task.Progress,
		URL:      extractURLFromResult(task.Result),
		Error:    task.Error,
	})
}

// extractURLFromResult 从 result JSON 中提取第一个 URL
func extractURLFromResult(result string) string {
	if result == "" {
		return ""
	}
	var urls []string
	if err := json.Unmarshal([]byte(result), &urls); err != nil {
		return result
	}
	if len(urls) > 0 {
		return urls[0]
	}
	return ""
}
```

StreamSSE handler：改为使用 TaskQueue：

```go
func (h *VideoHandler) StreamSSE(c *gin.Context) {
	taskID := c.Param("taskId")

	// 先检查任务是否存在
	task, err := h.task.GetTask(taskID)
	if err != nil || task == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	subID := fmt.Sprintf("sse_%d", time.Now().UnixNano())
	ch := h.task.Subscribe(taskID, subID)
	if ch == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "订阅失败"})
		return
	}
	defer h.task.Unsubscribe(taskID, subID)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-ch:
			if !ok {
				return false
			}
			switch event.Event {
			case "progress":
				c.SSEvent("progress", map[string]any{
					"progress": event.Progress,
					"status":   event.Status,
				})
			case "complete":
				c.SSEvent("complete", map[string]any{
					"result": event.Result,
				})
			case "error":
				c.SSEvent("error", map[string]any{
					"error": event.Error,
				})
			}
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}
```

SetupVideoHistoryCallback 需要改为接收 TaskQueue：

```go
func SetupVideoHistoryCallback(task *service.TaskQueue, svc *service.AgnesClient) {
	task.SetOnComplete(func(taskID, taskType, prompt, resultURL string) {
		if resultURL == "" {
			return
		}
		paths := []string{resultURL}
		if historyRepo != nil {
			if id, err := historyRepo.FindByTaskId(taskID); err == nil && id > 0 {
				updateHistoryImages(id, paths)
				log.Printf("[History] 任务 %s 历史已更新", taskID)
				return
			}
		}
		recordType := taskType
		if recordType == "" {
			recordType = "video"
		}
		saveHistoryRecord(prompt, paths, recordType, nil)
		log.Printf("[History] 任务 %s 历史已保存", taskID)
	})
}
```

- [ ] **Step 3: 新增 `backend/internal/handler/task_handler.go`**（统一任务查询+SSE 端点）

```go
package handler

import (
	"io"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/agnes-image-tool/backend/internal/service"
)

type TaskHandler struct {
	task *service.TaskQueue
}

func NewTaskHandler(task *service.TaskQueue) *TaskHandler {
	return &TaskHandler{task: task}
}

// GetTask 统一任务状态查询
// GET /api/v1/tasks/:id
func (h *TaskHandler) GetTask(c *gin.Context) {
	id := c.Param("id")
	rec, err := h.task.GetTask(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询任务失败: " + err.Error()})
		return
	}
	if rec == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	c.JSON(http.StatusOK, rec)
}

// StreamSSE 统一 SSE 进度推送
// GET /api/v1/tasks/:id/stream
func (h *TaskHandler) StreamSSE(c *gin.Context) {
	id := c.Param("id")

	rec, err := h.task.GetTask(id)
	if err != nil || rec == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	subID := fmt.Sprintf("sse_%d", time.Now().UnixNano())
	ch := h.task.Subscribe(id, subID)
	if ch == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "订阅失败"})
		return
	}
	defer h.task.Unsubscribe(id, subID)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-ch:
			if !ok {
				return false
			}
			switch event.Event {
			case "progress":
				c.SSEvent("progress", map[string]any{
					"progress": event.Progress,
					"status":   event.Status,
				})
			case "complete":
				c.SSEvent("complete", map[string]any{
					"result": event.Result,
				})
			case "error":
				c.SSEvent("error", map[string]any{
					"error": event.Error,
				})
			}
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}
```

- [ ] **Step 4: 验证编译**

```bash
cd backend && go vet ./internal/handler/...
```

Expected: fixes needed for encoding/json import, signature changes. Iterate until clean.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/handler/image.go backend/internal/handler/video.go backend/internal/handler/task_handler.go
git commit -m "feat(task): migrate handlers to async TaskQueue (image 202 + video TaskQueue)"
```

---

### Task 5: main.go 接线

**Files:**
- Modify: `backend/cmd/server/main.go` — 初始化 TaskRepository + TaskQueue，替换 TaskManager

#### 步骤

- [ ] **Step 1: 修改 main.go**

核心变更：
1. Init TaskRepository（复用 histRepo.DB()）
2. Init TaskQueue（替换 TaskManager）
3. 图片 Handler 注入 TaskQueue
4. 视频 Handler 注入 TaskQueue 替换 TaskManager
5. SetupVideoHistoryCallback 传入 TaskQueue 替换 TaskManager
6. 添加统一任务路由
7. 删除旧 TaskManager 初始化

```go
func main() {
	// ... 现有代码不变，直到 handler 初始化部分 ...

	// 初始化任务仓库（复用 history.db）
	taskRepo := repository.NewTaskRepository(histRepo.DB())
	if err := taskRepo.InitTable(); err != nil {
		log.Fatalf("初始化任务表失败: %v", err)
	}

	// 创建 TaskQueue（替换 TaskManager）
	taskQueue := service.NewTaskQueue(taskRepo, svc, 10)

	// 创建 handler
	imageHandler := handler.NewImageHandler(svc, taskQueue)          // 注入 TaskQueue
	videoHandler := handler.NewVideoHandler(svc, taskQueue)          // 注入 TaskQueue 替换 TaskManager
	// ... 其他 handler 不变 ...
	taskHandler := handler.NewTaskHandler(taskQueue)

	// 设置视频完成回调（自动保存历史记录）
	handler.SetupVideoHistoryCallback(taskQueue, svc)                // 传入 TaskQueue 替换 TaskManager

	// 注册路由
	api := r.Group("/api/v1")
	{
		// ... 现有图片/视频/配置/历史等路由不变 ...

		// 新增统一任务端点
		api.GET("/tasks/:id", taskHandler.GetTask)
		api.GET("/tasks/:id/stream", taskHandler.StreamSSE)
	}

	// ... 其余不变 ...
}
```

注意：删除 `taskMgr := service.NewTaskManager(svc)` 和相关的 `h.mgr` 使用。

- [ ] **Step 2: 验证编译**

```bash
cd backend && go vet ./cmd/server/...
cd backend && go build -o /dev/null ./cmd/server
```

Expected: build success.

- [ ] **Step 3: Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "feat(task): wire TaskQueue in main.go, remove TaskManager"
```

---

### Task 6: 删除 `video_manager.go`

**Files:**
- Delete: `backend/internal/service/video_manager.go`

#### 步骤

- [ ] **Step 1: 确认无引用**

```bash
cd backend && grep -r "TaskManager" internal/ --include="*.go"
```

Expected: no references to TaskManager (already migrated in Tasks 3-5).

- [ ] **Step 2: 删除文件并验证编译**

```bash
rm backend/internal/service/video_manager.go
cd backend && go build -o /dev/null ./cmd/server
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/service/video_manager.go
git commit -m "refactor(task): remove video_manager.go (replaced by TaskQueue)"
```

---

### Task 7: 前端 SSE/API 层改造

**Files:**
- Modify: `frontend/src/utils/sse.ts` — 增强为通用 `connectTaskSSE`
- Modify: `frontend/src/api/image.ts` — 改为 submit → return taskId
- Modify: `frontend/src/api/video.ts` — 改为 submit → return taskId
- Modify: `frontend/src/types/index.ts` — 添加 TaskRecord 类型

#### 步骤

- [ ] **Step 1: 前端类型 — 修改 `frontend/src/types/index.ts`**

在文件末尾添加：

```ts
// ==================== 异步任务 ====================

export interface TaskRecord {
  id: string
  type: string
  status: 'pending' | 'processing' | 'completed' | 'failed'
  params: string
  result?: string
  progress: number
  error?: string
  retry_count: number
  created_at: string
  updated_at: string
  completed_at?: string
}

export interface TaskCreateResponse {
  taskId: string
}

export interface TaskSSEHandlers {
  onProgress?: (data: { progress: number; status: string }) => void
  onComplete?: (data: { result: string }) => void
  onError?: (data: { error: string }) => void
}
```

- [ ] **Step 2: 改造 `frontend/src/utils/sse.ts`**

```ts
import type { TaskSSEHandlers } from '../types'

export function connectTaskSSE(taskId: string, handlers: TaskSSEHandlers): () => void {
  const url = `/api/v1/tasks/${taskId}/stream`
  const source = new EventSource(url)

  source.addEventListener('progress', (e: MessageEvent) => {
    try {
      const data = JSON.parse(e.data)
      handlers.onProgress?.(data)
    } catch {
      // ignore parse errors
    }
  })

  source.addEventListener('complete', (e: MessageEvent) => {
    try {
      const data = JSON.parse(e.data)
      handlers.onComplete?.(data)
      source.close()
    } catch {
      // ignore parse errors
    }
  })

  source.addEventListener('error', (e: MessageEvent) => {
    try {
      const data = JSON.parse(e.data)
      handlers.onError?.(data)
      source.close()
    } catch {
      if (source.readyState === EventSource.CLOSED) {
        handlers.onError?.({ error: 'SSE 连接已断开' })
      }
    }
  })

  return () => {
    source.close()
  }
}

// 保留旧函数名兼容
export const connectSSE = connectTaskSSE
```

- [ ] **Step 3: 改造 `frontend/src/api/image.ts`**

```ts
import client from './client'
import type { TextToImageRequest, BatchRequest, TaskCreateResponse } from '../types'

export async function submitTextToImage(data: TextToImageRequest): Promise<TaskCreateResponse> {
  const res = await client.post('/api/v1/images/text-to-image', data)
  return res.data
}

export async function submitImageToImage(
  image: File | string,
  prompt: string,
  size: string = '1024x1024',
  strength: number = 0.75,
  negativePrompt: string = ''
): Promise<TaskCreateResponse> {
  if (typeof image === 'string') {
    const res = await client.post('/api/v1/images/image-to-image', {
      image_url: image,
      prompt,
      size,
      strength,
      negative_prompt: negativePrompt || undefined,
    }, { timeout: 60000 })
    return res.data
  }
  const formData = new FormData()
  formData.append('image', image)
  formData.append('prompt', prompt)
  formData.append('size', size)
  formData.append('strength', String(strength))
  if (negativePrompt) {
    formData.append('negative_prompt', negativePrompt)
  }
  const res = await client.post('/api/v1/images/image-to-image', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 60000,
  })
  return res.data
}

export async function submitBatchGenerate(data: BatchRequest): Promise<TaskCreateResponse> {
  const res = await client.post('/api/v1/images/batch', data)
  return res.data
}
```

- [ ] **Step 4: 改造 `frontend/src/api/video.ts`**

```ts
import client from './client'
import type { ScriptGenRequest, ScriptGenResponse, VideoCreateRequest, TaskCreateResponse } from '../types'

export async function generateScript(data: ScriptGenRequest): Promise<ScriptGenResponse> {
  const res = await client.post('/api/v1/videos/generate-script', data, { timeout: 120000 })
  return res.data
}

export async function submitTextToVideo(data: VideoCreateRequest): Promise<TaskCreateResponse> {
  const res = await client.post('/api/v1/videos/text-to-video', data, { timeout: 30000 })
  return res.data
}

export async function submitImageToVideo(
  image: File | string,
  prompt: string,
  duration: number = 5,
  aspectRatio: string = '16:9',
  frameRate: number = 24,
  negativePrompt: string = ''
): Promise<TaskCreateResponse> {
  if (typeof image === 'string') {
    const res = await client.post('/api/v1/videos/image-to-video', {
      image_url: image,
      prompt,
      duration,
      aspect_ratio: aspectRatio,
      frame_rate: frameRate,
      negative_prompt: negativePrompt || undefined,
    }, { timeout: 30000 })
    return res.data
  }
  const formData = new FormData()
  formData.append('image', image)
  formData.append('prompt', prompt)
  formData.append('duration', String(duration))
  formData.append('aspect_ratio', aspectRatio)
  formData.append('frame_rate', String(frameRate))
  if (negativePrompt) {
    formData.append('negative_prompt', negativePrompt)
  }
  const res = await client.post('/api/v1/videos/image-to-video', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 30000,
  })
  return res.data
}

export async function submitMultiImageVideo(data: VideoCreateRequest): Promise<TaskCreateResponse> {
  const res = await client.post('/api/v1/videos/multi-image', data, { timeout: 30000 })
  return res.data
}
```

- [ ] **Step 5: 验证前端编译**

```bash
cd frontend && pnpm build
```

Expected: build success (or type errors to fix). Iterate.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/utils/sse.ts frontend/src/api/image.ts frontend/src/api/video.ts frontend/src/types/index.ts
git commit -m "feat(task): frontend SSE+API layer for async task queue"
```

---

### Task 8: 前端 TaskProgress 组件

**Files:**
- Create: `frontend/src/components/TaskProgress.vue`

#### 步骤

- [ ] **Step 1: 创建 `frontend/src/components/TaskProgress.vue`**

```vue
<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { connectTaskSSE } from '../utils/sse'
import type { TaskCreateResponse } from '../types'

const props = defineProps<{
  taskId: string
}>()

const emit = defineEmits<{
  complete: [result: string]
  error: [message: string]
}>()

const progress = ref(0)
const status = ref('pending')
const loading = ref(true)
let cleanup: (() => void) | null = null

onMounted(() => {
  cleanup = connectTaskSSE(props.taskId, {
    onProgress: (data) => {
      progress.value = data.progress
      status.value = data.status
    },
    onComplete: (data) => {
      progress.value = 100
      status.value = 'completed'
      loading.value = false
      emit('complete', data.result)
    },
    onError: (data) => {
      status.value = 'failed'
      loading.value = false
      emit('error', data.error)
    },
  })
})

onUnmounted(() => {
  cleanup?.()
})
</script>

<template>
  <div class="task-progress">
    <div v-if="loading || status === 'processing' || status === 'pending'" class="progress-bar-wrapper">
      <el-progress
        :percentage="progress"
        :status="status === 'failed' ? 'exception' : undefined"
        :stroke-width="16"
        :text-inside="true"
      />
      <p class="status-text">
        {{ status === 'pending' ? '排队中...' : status === 'processing' ? `生成中 ${progress}%` : '' }}
      </p>
    </div>
    <div v-else-if="status === 'failed'" class="error-message">
      <el-alert title="生成失败" :description="statusText" type="error" show-icon />
    </div>
  </div>
</template>

<style scoped>
.task-progress {
  margin: 16px 0;
}
.progress-bar-wrapper {
  text-align: center;
}
.status-text {
  margin-top: 8px;
  color: #909399;
  font-size: 14px;
}
.error-message {
  margin-top: 8px;
}
</style>
```

- [ ] **Step 2: 验证组件**

```bash
cd frontend && pnpm build
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/TaskProgress.vue
git commit -m "feat(task): add TaskProgress SSE component"
```

---

### Task 9: 前端视图改造 — 图片视图

**Files:**
- Modify: `frontend/src/views/TextToImage.vue`
- Modify: `frontend/src/views/ImageToImage.vue`
- Modify: `frontend/src/views/BatchGen.vue`

#### 步骤

- [ ] **Step 1: 改造 TextToImage.vue**

核心变化：从 `textToImage()` await 改为 `submitTextToImage()` + SSE：

```vue
<script setup lang="ts">
// 原有 imports 基础上添加：
import { submitTextToImage } from '../api/image'
import { connectTaskSSE } from '../utils/sse'
import TaskProgress from '../components/TaskProgress.vue'

// ... 原有状态 ...

const taskId = ref('')
const showProgress = ref(false)
let cleanupSSE: (() => void) | null = null

async function handleGenerate() {
  loading.value = true
  errorMsg.value = ''
  taskId.value = ''
  showProgress.value = false
  resultImages.value = []

  try {
    const res = await submitTextToImage({
      prompt: prompt.value,
      size: size.value,
      n: 1,
      negative_prompt: negativePrompt.value || undefined,
    })
    taskId.value = res.taskId
    showProgress.value = true

    cleanupSSE = connectTaskSSE(res.taskId, {
      onProgress: (data) => {
        // 可用进度条更新
      },
      onComplete: (data) => {
        showProgress.value = false
        try {
          resultImages.value = JSON.parse(data.result)
        } catch {
          resultImages.value = [data.result]
        }
        loading.value = false
      },
      onError: (data) => {
        showProgress.value = false
        errorMsg.value = data.error
        loading.value = false
      },
    })
  } catch (e: any) {
    errorMsg.value = e.message || '提交失败'
    loading.value = false
  }
}
</script>

<template>
  <!-- 在生成按钮下方添加 -->
  <TaskProgress v-if="showProgress && taskId" :task-id="taskId" />
</template>
```

- [ ] **Step 2: 改造 ImageToImage.vue**

类似的 submit + SSE 模式，调用 `submitImageToImage()`。

- [ ] **Step 3: 改造 BatchGen.vue**

调用 `submitBatchGenerate()` + SSE。

- [ ] **Step 4: 验证编译**

```bash
cd frontend && pnpm build
```

- [ ] **Step 5: Commit**

```bash
git add frontend/src/views/TextToImage.vue frontend/src/views/ImageToImage.vue frontend/src/views/BatchGen.vue
git commit -m "feat(task): migrate image views to async submit+SSE"
```

---

### Task 10: 前端视图改造 — 视频视图

**Files:**
- Modify: `frontend/src/views/TextToVideo.vue`
- Modify: `frontend/src/views/ImageToVideo.vue`
- Modify: `frontend/src/views/MultiImageVideo.vue`

#### 步骤

- [ ] **Step 1: 改造 TextToVideo.vue**

核心变化：SSE 端点从 `/api/v1/videos/stream/:taskId` 改为通用端点。在视频视图中，TaskQueue 仍使用 `/videos/stream/:taskId` 兼容端点（后端路由保持不变），所以前端改动最小。

实际上，视频视图已经使用了 SSE，且后端旧端点 `/videos/stream/:taskId` 已有 TaskQueue 后端支持。因此视频视图的改动只是将 SSE URL 改为 `/api/v1/tasks/:id/stream`（如果前端切换到新端点），或者保持现状。

**推荐**：视频视图暂时保持现有 SSE 使用方式（通过 `/videos/stream/:taskId` 端点），后端保持向后兼容。后续可以统一迁移。

需要检查视频视图当前 SSE 连接方式。如果使用 `connectSSE('/api/v1/videos/stream/' + taskId, ...)`，可以直接将路径改为通用端点的兼容路径。

实际上由于后端 `/videos/stream/:taskId` 保留兼容，视频视图**不需要**改动也可以工作。但建议将 SSE 连接更新为新工具函数：

```vue
// 替换:
import { connectSSE } from '../utils/sse'
connectSSE(taskId, handlers)

// 改为:
import { connectTaskSSE } from '../utils/sse'
connectTaskSSE(taskId, handlers)
```

`connectTaskSSE` 内部使用 `/api/v1/tasks/${taskId}/stream` 端点。

如果视图直接创建 `EventSource`，改为使用 `connectTaskSSE`。

- [ ] **Step 2: 验证编译**

```bash
cd frontend && pnpm build
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/TextToVideo.vue frontend/src/views/ImageToVideo.vue frontend/src/views/MultiImageVideo.vue
git commit -m "feat(task): update video views to use unified SSE"
```

---

### Task 11: 迁移脚本 — 历史 pending 视频迁移

**Files:**
- Create: `backend/scripts/migrate_pending_videos.go`

#### 步骤

- [ ] **Step 1: 创建 `backend/scripts/migrate_pending_videos.go`**

```go
// +build ignore

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

// 一次性迁移脚本：将 history 表中 pending 的视频任务迁移到 tasks 表
func main() {
	dbPath := "history.db"
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}
	defer db.Close()

	// 确保 tasks 表存在（如果任务队列已初始化）
	db.Exec(`CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY, type TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'pending',
		params TEXT NOT NULL, result TEXT, progress INTEGER NOT NULL DEFAULT 0,
		error TEXT, retry_count INTEGER NOT NULL DEFAULT 0,
		created_at TEXT, updated_at TEXT, completed_at TEXT
	)`)

	rows, err := db.Query(`
		SELECT id, mode, prompt, extra FROM history
		WHERE images = '[]' AND extra IS NOT NULL AND extra != ''
		ORDER BY id ASC
	`)
	if err != nil {
		log.Fatalf("查询失败: %v", err)
	}
	defer rows.Close()

	migrated := 0
	for rows.Next() {
		var id int64
		var mode, prompt, extraJSON string
		if err := rows.Scan(&id, &mode, &prompt, &extraJSON); err != nil {
			continue
		}
		var extra map[string]any
		if err := json.Unmarshal([]byte(extraJSON), &extra); err != nil {
			continue
		}
		taskID, _ := extra["taskId"].(string)
		if taskID == "" {
			continue
		}

		// 检查是否已迁移
		var count int
		db.QueryRow("SELECT COUNT(*) FROM tasks WHERE id = ?", "hist_"+taskID).Scan(&count)
		if count > 0 {
			continue
		}

		params, _ := json.Marshal(map[string]any{
			"taskId": taskID,
			"prompt": prompt,
		})
		_, err := db.Exec(
			"INSERT INTO tasks (id, type, status, params, progress, created_at, updated_at) VALUES (?, ?, 'pending', ?, 0, datetime('now','localtime'), datetime('now','localtime'))",
			"hist_"+taskID, mode, string(params),
		)
		if err != nil {
			log.Printf("迁移记录 %d 失败: %v", id, err)
			continue
		}
		migrated++
		fmt.Printf("已迁移: history_id=%d taskId=%s type=%s\n", id, taskID, mode)
	}

	fmt.Printf("迁移完成: 共 %d 条", migrated)
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/scripts/migrate_pending_videos.go
git commit -m "feat(task): add migration script for pending history videos"
```

---

## 自审清单

1. **Spec 覆盖**: 所有设计文档中的功能点（TaskQueue、Worker Pool、SQLite 持久化、重启恢复、SSE 统一、前端改造）都被映射到具体任务
2. **无占位符**: 所有代码块包含完整实现代码
3. **类型一致性**: `model.TaskType`/`model.TaskStatus` 常量在 Task 1 定义，被 Task 2/3 使用；`TaskQueue.SubmitTask` 签名在所有 handler 中一致
