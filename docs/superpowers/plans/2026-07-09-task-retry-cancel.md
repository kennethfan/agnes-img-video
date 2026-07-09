# 任务重试与取消 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 TaskQueue 补充自动重试（3 次指数退避）、手动重试 API、取消 pending 任务 API 及前端按钮。

**Architecture:** Worker 函数从单次执行改为 retry loop；TaskQueue 新增 cancelChans 支持 per-task 取消；Repository 层提供原子条件更新；前端 TaskProgress 组件条件展示重试/取消按钮。

**Tech Stack:** Go 1.25 · Gin · SQLite · Vue 3 · TypeScript 6 · Element Plus

## Global Constraints

- 所有代码注释使用中文
- Go 错误返回 string（无自定义 error type）
- SSE 事件类型不变：progress / complete / error
- 前端 TypeScript 6: erasableSyntaxOnly — 禁止 enum，使用 `as const` 或 union types
- Vue 3 Composition API + `<script setup>`
- 不新增数据库表，不新增第三方依赖

---

### Task 1: 模型层 — TaskStatusCancelled + TaskEvent.Message

**Files:**
- Modify: `backend/internal/model/task.go` — 追加 TaskStatusCancelled
- Modify: `backend/internal/model/types.go` — TaskEvent 加 Message 字段

**Interfaces:**
- Consumes: (无)
- Produces: `model.TaskStatusCancelled = "cancelled"`, `model.TaskEvent.Message string`

- [ ] **Step 1: 修改 model/task.go**

```go
const (
    TaskStatusPending    TaskStatus = "pending"
    TaskStatusProcessing TaskStatus = "processing"
    TaskStatusCompleted  TaskStatus = "completed"
    TaskStatusFailed     TaskStatus = "failed"
    TaskStatusCancelled  TaskStatus = "cancelled" // 新增
)
```

- [ ] **Step 2: 修改 model/types.go — TaskEvent 加 Message**

```go
type TaskEvent struct {
    Event    string `json:"-"` // progress / complete / error
    Progress int    `json:"progress,omitempty"`
    Status   string `json:"status,omitempty"`
    Result   string `json:"result,omitempty"`
    Error    string `json:"error,omitempty"`
    Message  string `json:"message,omitempty"` // 新增：重试说明等
}
```

- [ ] **Step 3: 验证编译**

```bash
cd /Users/kenneth/agent/agnes-img-video/backend
go vet ./internal/model/...
```

- [ ] **Step 4: 提交**

```bash
git add backend/internal/model/task.go backend/internal/model/types.go
git commit -m "feat(task): add TaskStatusCancelled and TaskEvent.Message field"
```

---

### Task 2: TaskRepository — CancelTask 原子条件更新

**Files:**
- Modify: `backend/internal/repository/task.go` — 新增 CancelTaskAtomic 方法

**Interfaces:**
- Consumes: (来自 Task 1) `model.TaskStatusCancelled`, `model.TaskStatusPending`
- Produces: `TaskRepository.CancelTaskAtomic(id string) (bool, error)` — 返回是否实际取消

- [ ] **Step 1: 追加 CancelTaskAtomic 方法**

在 `repository/task.go` 文件中追加，放在 UpdateRetryCount 之后：

```go
// CancelTaskAtomic 原子取消 pending 任务（仅当 status = 'pending' 时取消）
// 返回 cancelled=true 表示实际取消了任务，false 表示任务已不在 pending 状态
func (r *TaskRepository) CancelTaskAtomic(id string) (bool, error) {
    now := time.Now().Format("2006-01-02 15:04:05")
    res, err := r.db.Exec(
        "UPDATE tasks SET status = ?, updated_at = ?, completed_at = ? WHERE id = ? AND status = ?",
        string(model.TaskStatusCancelled), now, now, id, string(model.TaskStatusPending),
    )
    if err != nil {
        return false, fmt.Errorf("取消任务失败: %w", err)
    }
    rows, _ := res.RowsAffected()
    if rows == 0 {
        return false, nil // 任务不在 pending 状态
    }
    log.Printf("[TaskRepo] 任务已取消: id=%s", id)
    return true, nil
}
```

- [ ] **Step 2: 验证编译**

```bash
cd /Users/kenneth/agent/agnes-img-video/backend
go vet ./internal/repository/...
```

- [ ] **Step 3: 提交**

```bash
git add backend/internal/repository/task.go
git commit -m "feat(task): add CancelTaskAtomic for conditional cancel of pending tasks"
```

---

### Task 3: TaskQueue — 自动重试 + CancelTask + RetryTask + cancelChans

**Files:**
- Modify: `backend/internal/service/task_queue.go` — 核心改动

**Interfaces:**
- Consumes: `model.TaskStatusCancelled`, `model.TaskEvent.Message` (Task 1), `repository.CancelTaskAtomic` (Task 2)
- Produces:
  - `TaskQueue.cancelChans map[string]chan struct{}`
  - `TaskQueue.CancelTask(id string) error`
  - `TaskQueue.RetryTask(id string) error`
  - `TaskQueue.submitExistingTask(id, taskType, paramsJSON string) error` — 内部方法
  - 修改 `SubmitTask` 注册 cancel channel
  - 修改 `worker` 加入 retry loop

- [ ] **Step 1: 添加 retry 常量和 cancelChans 字段**

在 `task_queue.go` 的 const 区域追加：

```go
const (
    defaultMaxConcurrentTasks = 10
    pollInterval              = 5 * time.Second
    maxPollTime               = 30 * time.Minute
    pollRetryMax              = 10
    defaultMaxRetries         = 3                     // 新增
    retryBackoffBase          = 5 * time.Second       // 新增
)
```

在 TaskQueue struct 追加字段：

```go
type TaskQueue struct {
    mu          sync.RWMutex
    repo        *repository.TaskRepository
    client      *AgnesClient
    workerSem   chan struct{}
    subscribers map[string]map[string]chan model.TaskEvent
    onComplete  TaskCompleteFunc
    ctx         context.Context
    cancel      context.CancelFunc
    cancelChans map[string]chan struct{} // 新增：per-task 取消信号
}
```

在 NewTaskQueue 初始化：

```go
tq := &TaskQueue{
    repo:        repo,
    client:      client,
    workerSem:   make(chan struct{}, maxConcurrent),
    subscribers: make(map[string]map[string]chan model.TaskEvent),
    cancelChans: make(map[string]chan struct{}), // 新增
    ctx:         ctx,
    cancel:      cancel,
}
```

- [ ] **Step 2: 修改 SubmitTask — 注册 cancel channel**

```go
func (tq *TaskQueue) SubmitTask(taskType string, paramsJSON string) (string, error) {
    id, err := tq.repo.CreateTask(taskType, paramsJSON)
    if err != nil {
        return "", err
    }

    if err := tq.submitExistingTask(id, taskType, paramsJSON); err != nil {
        return "", err
    }
    return id, nil
}

// submitExistingTask 启动 worker goroutine（供 SubmitTask 和 RetryTask 共用）
func (tq *TaskQueue) submitExistingTask(id, taskType, paramsJSON string) error {
    cancelCh := make(chan struct{})
    tq.mu.Lock()
    tq.cancelChans[id] = cancelCh
    tq.mu.Unlock()

    select {
    case tq.workerSem <- struct{}{}:
        tq.mu.Lock()
        delete(tq.cancelChans, id)
        tq.mu.Unlock()
        go tq.worker(id, taskType, paramsJSON)
    default:
        log.Printf("[TaskQueue] 达到最大并发数，任务 %s 排队等待", id)
        go func() {
            select {
            case tq.workerSem <- struct{}{}:
                tq.mu.Lock()
                delete(tq.cancelChans, id)
                tq.mu.Unlock()
                // 再检查一次是否在排队期间被取消
                rec, _ := tq.repo.GetTask(id)
                if rec != nil && rec.Status == string(model.TaskStatusCancelled) {
                    <-tq.workerSem
                    return
                }
                tq.worker(id, taskType, paramsJSON)
            case <-cancelCh:
                return
            }
        }()
    }
    return nil
}
```

- [ ] **Step 3: 修改 worker — retry loop**

```go
func (tq *TaskQueue) worker(taskID, taskType, paramsJSON string) {
    defer func() {
        <-tq.workerSem
    }()

    // 检查是否已被取消
    rec, _ := tq.repo.GetTask(taskID)
    if rec != nil && rec.Status == string(model.TaskStatusCancelled) {
        log.Printf("[TaskQueue] 任务 %s 已取消，跳过执行", taskID)
        return
    }

    log.Printf("[TaskQueue] Worker 开始任务: id=%s type=%s", taskID, taskType)
    tq.repo.UpdateTaskStatus(taskID, string(model.TaskStatusProcessing), 0, "", "")
    tq.notifySubscribers(taskID, model.TaskEvent{
        Event:  "progress",
        Status: string(model.TaskStatusProcessing),
    })

    var lastErr error
    maxRetries := defaultMaxRetries

    for attempt := 0; attempt <= maxRetries; attempt++ {
        if attempt > 0 {
            backoff := retryBackoffBase * time.Duration(1<<uint(attempt-1))
            log.Printf("[TaskQueue] 任务 %s 失败，第 %d/%d 次重试，等待 %v", taskID, attempt, maxRetries, backoff)

            tq.repo.UpdateRetryCount(taskID, attempt)
            tq.notifySubscribers(taskID, model.TaskEvent{
                Event:    "progress",
                Status:   string(model.TaskStatusProcessing),
                Message:  fmt.Sprintf("失败后自动重试中 (%d/%d)", attempt, maxRetries),
            })

            time.Sleep(backoff)
        }

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

        if err == nil {
            return // 成功
        }
        lastErr = err
    }

    // 全部重试失败
    errMsg := lastErr.Error()
    log.Printf("[TaskQueue] 任务失败: id=%s err=%s", taskID, errMsg)
    tq.repo.UpdateTaskStatus(taskID, string(model.TaskStatusFailed), 0, "", errMsg)
    tq.notifySubscribers(taskID, model.TaskEvent{
        Event: "error",
        Error: errMsg,
    })
}
```

- [ ] **Step 4: 添加 CancelTask 方法**

```go
// CancelTask 取消 pending 状态的任务
func (tq *TaskQueue) CancelTask(id string) error {
    rec, err := tq.repo.GetTask(id)
    if err != nil {
        return err
    }
    if rec == nil {
        return fmt.Errorf("任务不存在")
    }
    if rec.Status != string(model.TaskStatusPending) {
        return fmt.Errorf("只能取消 pending 状态的任务（当前: %s）", rec.Status)
    }

    cancelled, err := tq.repo.CancelTaskAtomic(id)
    if err != nil {
        return err
    }
    if !cancelled {
        return fmt.Errorf("任务已不在 pending 状态，无法取消")
    }

    // 通知等待中的 goroutine
    tq.mu.Lock()
    ch, ok := tq.cancelChans[id]
    delete(tq.cancelChans, id)
    tq.mu.Unlock()
    if ok {
        close(ch)
    }

    tq.notifySubscribers(id, model.TaskEvent{
        Event: "progress",
        Status: string(model.TaskStatusCancelled),
    })

    log.Printf("[TaskQueue] 任务已取消: id=%s", id)
    return nil
}
```

- [ ] **Step 5: 添加 RetryTask 方法**

```go
// RetryTask 手动重试失败的任务
func (tq *TaskQueue) RetryTask(id string) error {
    rec, err := tq.repo.GetTask(id)
    if err != nil {
        return err
    }
    if rec == nil {
        return fmt.Errorf("任务不存在")
    }
    if rec.Status != string(model.TaskStatusFailed) {
        return fmt.Errorf("只能重试失败的任务（当前: %s）", rec.Status)
    }

    // 重置状态
    tq.repo.UpdateTaskStatus(id, string(model.TaskStatusPending), 0, "", "")
    tq.repo.UpdateRetryCount(id, 0)

    // 重新提交
    return tq.submitExistingTask(id, rec.Type, rec.Params)
}
```

- [ ] **Step 6: 验证编译**

```bash
cd /Users/kenneth/agent/agnes-img-video/backend
go vet ./internal/service/...
```

- [ ] **Step 7: 提交**

```bash
git add backend/internal/service/task_queue.go
git commit -m "feat(task): add auto retry loop, CancelTask, RetryTask with cancelChans"
```

---

### Task 4: TaskHandler — RetryTask + CancelTask handlers

**Files:**
- Modify: `backend/internal/handler/task_handler.go`

**Interfaces:**
- Consumes: `TaskQueue.CancelTask`, `TaskQueue.RetryTask` (Task 3)
- Produces: `TaskHandler.RetryTask(c *gin.Context)`, `TaskHandler.CancelTask(c *gin.Context)`

- [ ] **Step 1: 添加 CancelTask handler**

```go
// CancelTask 取消 pending 任务
// POST /api/v1/tasks/:id/cancel
func (h *TaskHandler) CancelTask(c *gin.Context) {
    id := c.Param("id")
    if err := h.task.CancelTask(id); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "任务已取消"})
}
```

- [ ] **Step 2: 添加 RetryTask handler**

```go
// RetryTask 手动重试失败任务
// POST /api/v1/tasks/:id/retry
func (h *TaskHandler) RetryTask(c *gin.Context) {
    id := c.Param("id")
    if err := h.task.RetryTask(id); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "任务已重新提交"})
}
```

- [ ] **Step 3: 验证编译**

```bash
cd /Users/kenneth/agent/agnes-img-video/backend
go vet ./internal/handler/...
```

- [ ] **Step 4: 提交**

```bash
git add backend/internal/handler/task_handler.go
git commit -m "feat(task): add CancelTask and RetryTask HTTP handlers"
```

---

### Task 5: main.go — 注册新路由

**Files:**
- Modify: `backend/cmd/server/main.go` — 追加两行路由

**Interfaces:**
- Consumes: `TaskHandler.CancelTask`, `TaskHandler.RetryTask` (Task 4)

- [ ] **Step 1: 在 tasks 路由块中追加**

在 `api.GET("/tasks/:id/stream", taskHandler.StreamSSE)` 之后添加：

```go
api.POST("/tasks/:id/cancel", taskHandler.CancelTask)
api.POST("/tasks/:id/retry", taskHandler.RetryTask)
```

- [ ] **Step 2: 验证编译**

```bash
cd /Users/kenneth/agent/agnes-img-video/backend
go build ./cmd/server
```

- [ ] **Step 3: 提交**

```bash
git add backend/cmd/server/main.go
git commit -m "feat(task): register cancel and retry routes"
```

---

### Task 6: 前端 — api/task.ts + TaskProgress.vue 按钮

**Files:**
- Create: `frontend/src/api/task.ts` — retryTask / cancelTask API
- Modify: `frontend/src/components/TaskProgress.vue` — 条件按钮

**Interfaces:**
- Consumes: 后端 `POST /api/v1/tasks/:id/cancel`, `POST /api/v1/tasks/:id/retry`
- Produces: TaskProgress 组件 emits `retry` 事件（由父视图处理重新连接 SSE）

- [ ] **Step 1: 创建 api/task.ts**

```typescript
import client from './client'

export async function retryTask(taskId: string): Promise<void> {
  await client.post(`/api/v1/tasks/${taskId}/retry`)
}

export async function cancelTask(taskId: string): Promise<void> {
  await client.post(`/api/v1/tasks/${taskId}/cancel`)
}
```

- [ ] **Step 2: 修改 TaskProgress.vue — 添加取消和重试按钮**

```vue
<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { connectTaskSSE } from '../utils/sse'
import { cancelTask, retryTask } from '../api/task'

const props = defineProps<{
  taskId: string
}>()

const emit = defineEmits<{
  complete: [result: string]
  error: [message: string]
  retry: [taskId: string]  // 新增：重试后通知父组件重新连接
}>()

const progress = ref(0)
const status = ref('pending')
const loading = ref(true)
const cancelling = ref(false)  // 新增
const retrying = ref(false)     // 新增
let cleanup: (() => void) | null = null

async function handleCancel() {
  cancelling.value = true
  try {
    await cancelTask(props.taskId)
    status.value = 'cancelled'
    loading.value = false
    cleanup?.()
  } catch (e: any) {
    // 错误由组件外部显示
  } finally {
    cancelling.value = false
  }
}

async function handleRetry() {
  retrying.value = true
  try {
    cleanup?.()
    await retryTask(props.taskId)
    // 重新连接 SSE
    status.value = 'pending'
    progress.value = 0
    loading.value = true
    cleanup = connectTaskSSE(props.taskId, {
      onProgress: (data) => {
        progress.value = data.progress
        status.value = data.status
      },
      onComplete: (data) => {
        progress.value = 100
        status.value = 'completed'
        loading.value = false
        cleanup?.()
        emit('complete', data.result)
      },
      onError: (data) => {
        status.value = 'failed'
        loading.value = false
        emit('error', data.error)
      },
    })
  } catch (e: any) {
    // 错误由组件外部显示
  } finally {
    retrying.value = false
  }
}

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
    <div v-if="status === 'cancelled'" class="cancelled-message">
      <el-alert title="任务已取消" type="info" show-icon :closable="false" />
    </div>
    <div v-else-if="loading || status === 'processing' || status === 'pending'" class="progress-bar-wrapper">
      <el-progress
        :percentage="progress"
        :status="status === 'failed' ? 'exception' : undefined"
        :stroke-width="16"
        :text-inside="true"
      />
      <p class="status-text">
        {{ status === 'pending' ? '排队中...' : status === 'processing' ? `生成中 ${progress}%` : '' }}
      </p>
      <el-button
        v-if="status === 'pending'"
        type="danger"
        size="small"
        :loading="cancelling"
        @click="handleCancel"
        style="margin-top: 8px"
      >
        {{ cancelling ? '取消中...' : '取消' }}
      </el-button>
    </div>
    <div v-else-if="status === 'failed'" class="error-message">
      <el-alert title="生成失败" type="error" show-icon :closable="false" />
      <el-button
        type="primary"
        size="small"
        :loading="retrying"
        @click="handleRetry"
        style="margin-top: 8px"
      >
        {{ retrying ? '重试中...' : '重试' }}
      </el-button>
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
.cancelled-message {
  margin-top: 8px;
}
</style>
```

- [ ] **Step 3: 验证类型检查**

```bash
cd /Users/kenneth/agent/agnes-img-video/frontend
npx vue-tsc --noEmit 2>&1 | head -30
```

- [ ] **Step 4: 提交**

```bash
git add frontend/src/api/task.ts frontend/src/components/TaskProgress.vue
git commit -m "feat(task): add retry/cancel buttons in TaskProgress and api/task.ts"
```
