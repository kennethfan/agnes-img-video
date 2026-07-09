# 异步任务重试与取消 设计文档

**日期:** 2026-07-09
**状态:** 待实现

## 概述

为异步任务队列（TaskQueue）补充重试（自动 + 手动）和取消功能。当前系统支持任务提交、状态查询、SSE 推送，但失败后无重试机制，排队中的任务也无法取消。

## 1. 模型层

### 新增状态

`backend/internal/model/task.go` 追加：

```go
const (
    TaskStatusCancelled TaskStatus = "cancelled"
)
```

现有状态：`pending`、`processing`、`completed`、`failed`。

### TaskRecord 字段

`retry_count` 字段已存在，无需扩展。

## 2. 自动重试

### Worker 改造

`worker()` 从单次执行改为 retry loop：

```
worker 开始
  → 检查 status 是否为 "cancelled"（防止取消后仍启动）
  → 标记 status = "processing"，通知 SSE
  → 循环（最多 MaxRetries=3 次）：
      → 执行 exec*(taskID, paramsJSON)
      → 成功 → 标记 completed，退出
      → 失败 → retry_count++
              → 更新 DB retry_count，通知 SSE "重试中 (N/3)"
              → 指数退避等待（5s → 15s → 30s）
              → 继续循环
  → 全部失败 → 标记 failed，通知 SSE error
```

### 常量

```go
const (
    defaultMaxRetries    = 3
    retryBackoffBase     = 5 * time.Second  // 第一次重试前等 5s
    retryBackoffFactor   = 3                 // 每次翻 3 倍：5s, 15s, 30s
)
```

### SSE 通知

重试进度通过 SSE 的 `progress` 事件推送：
```json
{
  "event": "progress",
  "status": "processing",
  "progress": 50,
  "message": "失败后自动重试中 (1/3)"
}
```

`TaskEvent` 结构体新增 `Message` 字段：
```go
type TaskEvent struct {
    Event    string `json:"-"`
    Status   string `json:"status,omitempty"`
    Progress int    `json:"progress,omitempty"`
    Result   string `json:"result,omitempty"`
    Error    string `json:"error,omitempty"`
    Message  string `json:"message,omitempty"` // 新增：重试说明等
}
```

SSE 的 `progress` 事件响应也加上 `message` 字段。

## 3. 手动重试 API

### 端点

```
POST /api/v1/tasks/:id/retry
```

### 逻辑

1. 查询任务是否存在
2. 检查 status 是否为 `failed`
3. 重置：`status = "pending"`, `retry_count = 0`, `error = ""`, `progress = 0`
4. 重新调用 `SubmitTask` 的相同逻辑：启动 goroutine 重新执行

### 实现

`TaskQueue` 新增 `RetryTask(id string) error` 方法：

```go
func (tq *TaskQueue) RetryTask(id string) error {
    rec, err := tq.repo.GetTask(id)
    if err != nil {
        return err
    }
    if rec == nil {
        return fmt.Errorf("任务不存在")
    }
    if rec.Status != string(model.TaskStatusFailed) {
        return fmt.Errorf("只能重试失败的任务")
    }

    // 重置状态
    tq.repo.UpdateTaskStatus(id, string(model.TaskStatusPending), 0, "", "")
    tq.repo.UpdateRetryCount(id, 0)

    // 重新提交
    _, err = tq.submitExistingTask(id, rec.Type, rec.Params)
    return err
}
```

`submitExistingTask` 抽取自 `SubmitTask` 的 goroutine 启动逻辑（不 CreateTask）。

`TaskHandler` 新增 `RetryTask` handler。

## 4. 取消 API

### 端点

```
POST /api/v1/tasks/:id/cancel
```

### 逻辑

1. 查询任务是否存在
2. 检查 status 是否为 `pending`
3. 原子更新：`UPDATE tasks SET status = 'cancelled' WHERE id = ? AND status = 'pending'`
   - 若影响行数为 0，说明任务已不在 pending 状态，返回错误
4. 通过 cancel channel 通知等待中的 goroutine 停止

### 实现

`TaskQueue` 新增字段和方法：

```go
type TaskQueue struct {
    // ... 现有字段
    cancelChans map[string]chan struct{}
}

func (tq *TaskQueue) CancelTask(id string) error {
    rec, err := tq.repo.GetTask(id)
    if err != nil { return err }
    if rec == nil { return fmt.Errorf("任务不存在") }

    // 原子更新：只取消 pending 的任务
    result, err := tq.repo.db.Exec(
        "UPDATE tasks SET status = ?, updated_at = ? WHERE id = ? AND status = ?",
        string(model.TaskStatusCancelled), time.Now().Format("..."), id, string(model.TaskStatusPending),
    )
    if err != nil { return err }
    rows, _ := result.RowsAffected()
    if rows == 0 { return fmt.Errorf("任务已开始执行，无法取消") }

    // 通知等待中的 goroutine
    tq.mu.RLock()
    ch, ok := tq.cancelChans[id]
    tq.mu.RUnlock()
    if ok {
        close(ch)
    }
    return nil
}
```

`SubmitTask` 中创建 goroutine 时注册 cancel channel：

```go
cancelCh := make(chan struct{})
tq.mu.Lock()
tq.cancelChans[id] = cancelCh
tq.mu.Unlock()

go func() {
    select {
    case tq.workerSem <- struct{}{}:
        tq.mu.Lock()
        delete(tq.cancelChans, id)
        tq.mu.Unlock()
        // worker 开始前再检查一次状态
        rec, _ := tq.repo.GetTask(id)
        if rec != nil && rec.Status == string(model.TaskStatusCancelled) {
            <-tq.workerSem // 释放 semaphore
            return
        }
        tq.worker(id, taskType, paramsJSON)
    case <-cancelCh:
        // 已被取消，释放 cancel channel
        close(cancelCh) // 实际 close 已在 CancelTask 中完成
        return
    }
}()
```

## 5. 前端改造

### TaskProgress.vue

在现有组件上增加条件按钮：

| 状态 | 按钮 | API |
|------|------|-----|
| pending | "取消" | `POST /api/v1/tasks/:id/cancel` |
| failed | "重试" | `POST /api/v1/tasks/:id/retry` |

- `cancelled` 状态显示 "已取消"
- 重试成功后：重新连接 SSE 监听新任务
- 取消成功后：停止 SSE，显示 "已取消"

### API 层

`frontend/src/api/task.ts` 新增：

```typescript
export async function retryTask(taskId: string): Promise<void>
export async function cancelTask(taskId: string): Promise<void>
```

## 6. 涉及文件

| 文件 | 改动 |
|------|------|
| `backend/internal/model/task.go` | +TaskStatusCancelled |
| `backend/internal/model/types.go` | TaskEvent 加 Message 字段 |
| `backend/internal/service/task_queue.go` | retry loop + CancelTask/RetryTask + cancelChans |
| `backend/internal/handler/task_handler.go` | +RetryTask/+CancelTask handlers |
| `backend/cmd/server/main.go` | 注册新路由 |
| `frontend/src/components/TaskProgress.vue` | +重试/+取消按钮，+cancelled显示 |
| `frontend/src/api/task.ts` | 新增（或追加到现有 api 文件） |

## 7. 不变事项

- 不新增数据库表（tasks 表已有所有字段）
- 不改变现有 API 响应格式
- 不改变现有 SSE 事件类型
- 不引入新依赖
