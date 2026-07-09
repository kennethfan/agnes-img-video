# Agnes Creator Studio — 统一异步任务队列设计

## 概述

将当前图片生成（同步）和视频生成（半异步）统一改造为**基于 SQLite 持久化的异步任务队列**，支持：

- **统一接口**: 所有生成任务（图片/视频）提交后立即返回 `taskId`，前端通过 SSE 等待结果
- **server 重启恢复**: 未完成的任务通过 SQLite 持久化，重启后自动恢复轮询
- **统一 SSE**: 所有任务通过 `/api/v1/tasks/:id/stream` 推送进度

## 现状 vs 目标

| 方面 | 现状 | 目标 |
|------|------|------|
| **图片生成** | 同步 HTTP，阻塞 180s | 异步 submit + SSE，立即返回 taskId |
| **批量生图** | 串行循环，20 个 prompt 可能等 30 分钟 | 拆分为独立 task，并行 Worker 执行 |
| **视频生成** | 半异步（`TaskManager` 内存 map + goroutine） | 迁移到统一 TaskQueue |
| **持久化** | 无 — 重启后丢失内存中的视频任务 | SQLite `tasks` 表持久化全部任务 |
| **重启恢复** | `recoverPendingVideoTasks()` 一次快照检查，不恢复轮询 | Worker Pool 启动时自动加载 pending 任务并恢复轮询 |
| **SSE** | 仅视频 `/videos/stream/:taskId` | 统一 `/api/v1/tasks/:id/stream` |
| **查询状态** | 仅视频 `/videos/:taskId` | 统一 `/api/v1/tasks/:id` |
| **Go 文件** | 图片 handler (sync) + video manager (async) | 统一: `TaskQueue` + `TaskRepository` + 统一 handler |

## API 变更

### 新增端点

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/tasks/image/text-to-image` | 异步文生图 → `{taskId}` |
| POST | `/api/v1/tasks/image/image-to-image` | 异步图生图 → `{taskId}` |
| POST | `/api/v1/tasks/image/batch` | 异步批量 → `{taskId}` (任务拆分) |
| POST | `/api/v1/tasks/video/text-to-video` | 迁移 → `{taskId}` |
| POST | `/api/v1/tasks/video/image-to-video` | 迁移 → `{taskId}` |
| POST | `/api/v1/tasks/video/multi-image` | 迁移 → `{taskId}` |
| GET | `/api/v1/tasks/:id` | 统一任务状态查询 |
| GET | `/api/v1/tasks/:id/stream` | 统一 SSE 进度推送 |

### 保留的旧端点（兼容过渡期）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/images/text-to-image` | 保留，行为改为异步 |
| POST | `/api/v1/images/image-to-image` | 保留，行为改为异步 |
| POST | `/api/v1/images/batch` | 保留，行为改为异步 |
| POST | `/api/v1/videos/text-to-video` | 保留，内部迁移到 TaskQueue |
| POST | `/api/v1/videos/image-to-video` | 保留，内部迁移到 TaskQueue |
| POST | `/api/v1/videos/multi-image` | 保留，内部迁移到 TaskQueue |
| GET | `/api/v1/videos/:taskId` | 保留，内部查询 TaskQueue |
| GET | `/api/v1/videos/stream/:taskId` | **保留**，内部路由到 `/tasks/:id/stream` |

`/videos/:taskId` 和 `/videos/stream/:taskId` 路径存在路由歧义（`:taskId` 可能匹配 `generate-script`、`text-to-video` 等）。当前已通过 Gin 路由顺序（静态路由优先于参数路由）解决。新设计通过统一前缀 `/tasks/` 避免此问题。旧端点保留仅用于兼容过渡，推荐前端迁移到新端点。逐步迁移后再移除旧端点。

## 数据层设计

### SQLite `tasks` 表

```sql
CREATE TABLE IF NOT EXISTS tasks (
    id          TEXT PRIMARY KEY,          -- 格式: task_{timestamp}_{random}
    type        TEXT NOT NULL,             -- text2image / image2image / batch / text2video / image2video / multi_image_video
    status      TEXT NOT NULL DEFAULT 'pending',  -- pending / processing / completed / failed
    params      TEXT NOT NULL,             -- JSON: 原始请求参数
    result      TEXT,                      -- JSON: 结果数据（图片 URL 列表或视频 URL）
    progress    INTEGER NOT NULL DEFAULT 0, -- 0-100
    error       TEXT,                      -- 错误信息
    retry_count INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL DEFAULT (datetime('now','localtime')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now','localtime')),
    completed_at TEXT                      -- 完成时间
);

CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_type ON tasks(type);
CREATE INDEX idx_tasks_created ON tasks(created_at);
```

### TaskRepository (`internal/repository/task.go`)

```
TaskRepository
├── CreateTask(type, paramsJSON) → id
├── GetTask(id) → TaskRecord
├── UpdateStatus(id, status, progress, result/error)
├── FindPendingTasks() → []TaskRecord  (重启恢复用)
├── ListTasks(type, status, limit, offset)
├── CleanupOlderThan(duration)
└── DB() → *sql.DB  (复用 history.db)
```

### TaskRecord 结构体

```go
type TaskRecord struct {
    ID          string    `json:"id"`
    Type        string    `json:"type"`
    Status      string    `json:"status"`
    Params      string    `json:"params"`    // JSON string
    Result      string    `json:"result,omitempty"` // JSON string
    Progress    int       `json:"progress"`
    Error       string    `json:"error,omitempty"`
    RetryCount  int       `json:"retry_count"`
    CreatedAt   string    `json:"created_at"`
    UpdatedAt   string    `json:"updated_at"`
    CompletedAt string    `json:"completed_at,omitempty"`
}
```

## 服务层设计

### TaskQueue (`internal/service/task_queue.go`)

替换 `TaskManager`（`video_manager.go` 可删除）。

```go
type TaskQueue struct {
    mu         sync.RWMutex
    repo       *repository.TaskRepository
    client     *AgnesClient
    workerSem  chan struct{}             // 并发限制信号量
    subscribers map[string]map[string]chan model.TaskEvent  // taskID → subID → chan
    onComplete TaskCompleteFunc          // 完成回调（保存历史记录等）
    ctx        context.Context
    cancel     context.CancelFunc
}
```

**关键行为**：

| 方法 | 说明 |
|------|------|
| `NewTaskQueue(repo, client, maxConcurrent)` | 创建队列，自动调用 `recoverPending()` |
| `SubmitTask(type, params, payloadBuilder)` | 写入 SQLite → 返回 taskId → 启动 Worker |
| `GetTask(id)` | 从 SQLite 查询 TaskRecord |
| `Subscribe(id, subID) → chan` | 注册 SSE 订阅者 |
| `Unsubscribe(id, subID)` | 移除订阅者 |
| `recoverPending()` | 启动时加载所有 pending/processing 任务，恢复轮询 |
| `cleanupStaleTasks()` | 定期清理超时（30min）且无订阅者的任务 |

### Task Worker 生命周期

```
SubmitTask()
  ├── 生成 taskId (task_{timestamp}_{rand6})
  ├── 写入 SQLite (status=pending)
  └── select workerSem → go worker()

worker()
  ├── update status=processing
  ├── notify subscribers (progress: 0, status: processing)
  │
  ├── 根据 task.Type 分发:
  │   ├── text2image → client.TextToImage()
  │   ├── image2image → client.ImageToImage()
  │   ├── batch → 拆分为多个子任务（同一 TaskQueue 中的独立 Task）
  │   ├── text2video → client.SubmitVideoTask() + poll loop
  │   ├── image2video → client.SubmitVideoTask() + poll loop
  │   └── multi_image_video → client.SubmitVideoTask() + poll loop
  │
  ├── 成功 → update SQLite (status=completed, result=...)
  │        → notify subscribers (event=complete)
  │        → 触发 onComplete 回调（保存历史记录）
  │
  └── 失败 → update SQLite (status=failed, error=...)
           → notify subscribers (event=error)
```

### 图片生成 Worker（新增异步逻辑）

图片生成现在需要和视频类似的 submit + poll 模式（因为 Agnes API 图片生成当前只支持同步，但我们需要统一接口）。有两种策略：

**策略 A（推荐）：图片生成直接同步执行，但在 Worker goroutine 中运行**

```
worker() for text2image:
  update processing
  url, err := client.TextToImage(...)
  if err → failed
  else → completed, notify subscriber
```

即：Worker goroutine 内部仍是同步调用，但 handler 不阻塞。这是最简单的方案，且对前端透明。

**策略 B：如果未来 Agnes API 支持图片异步 submit**

Worker 可以升级为 submit → poll 模式而不改 handler/前端。

### 批量任务的处理

Batch 不再串行循环 20 次。改为：
1. 提交 1 个 "batch parent" task（status=pending）
2. 拆分为 N 个独立子 task（关联 parentId）
3. 每个子 task 独立排队、独立执行
4. SSE 推送各子任务的进度，父任务汇总进度
5. 全部完成后父任务标记 completed

**简化方案（v1）**：Batch 仍然在 1 个 Worker 中串行执行，但更新进度清晰。使用 SSE 推送 `progress` 事件，如 `{progress: 5, subTask: "3/5", result: [...]}`。

**推荐 v1**: Batch 的 Worker 串行执行，但每完成一个子项就更新 progress 并推送 SSE。这避免了 v1 的任务树复杂度。

### 视频生成 Worker（从 TaskManager 迁移）

当前视频 Worker 已在 `TaskManager.startPolling()` 中实现轮询逻辑。迁移要点：

```
worker() for text2video:
  1. SubmitVideoTask() → videoID
  2. 写入 TaskRecord.params.taskId = videoID (Agnes 的 task ID)
  3. Poll loop (复用现有逻辑: 5s 间隔, 30min 超时, 指数退避)
  4. 完成 → save result → notify subscriber
  5. 失败 → save error → notify subscriber
```

与当前区别：任务状态持久化到 SQLite，重启后可恢复轮询。

### 重启恢复逻辑 (`recoverPending()`)

启动时自动执行：

```
recoverPending():
  1. repo.FindPendingTasks() → []TaskRecord (status IN ('pending','processing'))
  2. 对每个任务:
     a. 检查类型
     b. 图片任务 (text2image/image2image):
        - 图片任务理论上不应有 pending 状态（除非 server 在提交 Agnes API 前崩溃）
        - 对于 pending 的图片任务: 重新提交 worker
        - 对于 processing 的图片任务: 重新提交（Agnes API 不保证幂等，但生成后之前的结果会被覆盖）
     c. 视频任务 (text2video/image2video/multi_image_video):
        - 从 params 中取出 taskId（Agnes 的 video_id）
        - 检查 CheckVideoStatus()
        - 如果 completed: 写入 result, notify (无 subscriber 则仅更新 DB)
        - 如果 failed: 更新 DB
        - 如果仍 processing: 重启 poll loop
     d. 更新 retry_count
```

## SSE 统一

### 新端点 `/api/v1/tasks/:id/stream`

与当前 `/videos/stream/:taskId` 完全相同的协议。只改路由前缀。

### SSE 事件格式（不变）

```
event: progress  data: {"progress": 45, "status": "processing"}
event: complete  data: {"result": [...], "progress": 100}
event: error     data: {"error": "生成失败: ..."}
```

### 旧端点兼容

`/videos/stream/:taskId` 内部转发：实际代码直接使用 `taskQueue.Subscribe`，不需要显式路由转发——两种路由都调用相同的 `TaskQueue.Subscribe`。

## 文件变更清单

### 新增文件（3 个）

| 文件 | 说明 |
|------|------|
| `internal/repository/task.go` | TaskRepository — SQLite CRUD |
| `internal/service/task_queue.go` | TaskQueue — Worker Pool + subscriber pattern |
| `internal/model/task.go` | Task 相关类型（TaskRecord, TaskEvent, TaskCreateRequest） |

### 修改文件（6 个）

| 文件 | 变更 |
|------|------|
| `internal/handler/image.go` | 改为异步：返回 202 + taskId，不再返回 200 + images |
| `internal/handler/video.go` | 视频 handler 注入 TaskQueue 替换 TaskManager；SSE 和状态查询迁移到统一端点 |
| `cmd/server/main.go` | 初始化 TaskQueue + TaskRepository，替换 TaskManager |
| `frontend/src/api/image.ts` | 不再等待结果，改为 submit → 返回 taskId |
| `frontend/src/api/video.ts` | 不再需要 getTaskStatus（统一到 /tasks/:id） |
| `frontend/src/utils/sse.ts` | 改为 `connectTaskSSE(taskId, handlers)`，可配置 basePath |

### 删除文件（1 个）

| 文件 | 说明 |
|------|------|
| `internal/service/video_manager.go` | 被 `task_queue.go` 替代 |

## 前端改造

### 核心模式变化

```
当前: submit → await → 显示结果
改为: submit → 返回 taskId → 连接 SSE → 实时更新 → 显示结果
```

### 受影响的视图

| 视图 | 当前行为 | 改造后 |
|------|----------|--------|
| **TextToImage.vue** | `textToImage()` await → 返回 images[] → 显示 Gallery | `submitTask()` → SSE → 收到 complete 时显示 Gallery |
| **ImageToImage.vue** | `imageToImage()` await → 返回 images[] → 显示 Gallery | 同上 |
| **BatchGen.vue** | `batchGenerate()` await → 返回 images[] → 显示 Gallery | submit → SSE 逐项推送 progress → 全部完成后显示 |
| **TextToVideo.vue** | `createTextToVideo()` → 已有 SSE | 迁移到新 SSE 端点，逻辑基本不变 |
| **ImageToVideo.vue** | `createImageToVideo()` → 已有 SSE | 同上 |
| **MultiImageVideo.vue** | `createMultiImageVideo()` → 已有 SSE | 同上 |

### 新增可复用组件

**`TaskProgress.vue`** — 通用任务进度组件（替换 `VideoProgress.vue` 的职责）：
- props: `taskId: string`, `type: string`
- 内部使用 `connectTaskSSE()`
- 展示进度条 + 状态文本
- complete 时 emit `@complete(result)`
- error 时 emit `@error(message)`

### 前端 API 层设计

```ts
// src/api/tasks.ts — 新增
export async function submitTask(type: string, params: any): Promise<{taskId: string}>
export async function getTask(taskId: string): Promise<TaskRecord>

// src/utils/sse.ts 增强
export function connectTaskSSE(taskId: string, handlers: SSEHandlers): () => void
```

现有 `image.ts` 和 `video.ts` 的 submit 函数改为返回 `taskId`:

```ts
// image.ts 改造后
export async function submitTextToImage(data: TextToImageRequest): Promise<{taskId: string}> {
  const res = await client.post('/api/v1/tasks/image/text-to-image', data)
  return res.data  // {taskId: "task_...", ...}
}
```

## 边界情况

1. **幂等性**: 图片生成 API 不支持幂等。重启后如果重新提交图片任务，可能生成不同的图片。这是可接受的——用户期望的是"继续执行"，而非"恢复完全相同的结果"。

2. **并发限制**: Worker Pool 的信号量 `workerSem` 控制最大并发数（默认 10，与当前视频并发一致）。图片任务也会占用并发槽位。

3. **任务过期清理**: 完成/失败超过 24 小时的任务可自动清理（从 SQLite 删除）。在 `TaskQueue` 中起定时 goroutine。

4. **SSE 连接丢失**: 前端重连策略——页面可以记录 taskId，重连时调用 `getTask()` 检查状态，如果仍在 processing 则重新连接 SSE。

5. **Batch 的中断恢复**: 当前 v1 设计 Batch 是单个 Worker 串行执行，重启后重新从头开始。如果 batch 已完成部分子项，无法跳过——因为 Agnes API 不保证幂等。这是 v1 的可接受 trade-off（未来可以设计任务树解决）。

6. **历史记录兼容**: 当前视频完成回调 `SetupVideoHistoryCallback` 通过 `onComplete` 写入 `history` 表。TaskQueue 保留同样的 `onComplete` 回调机制。

7. **迁移现有 pending 视频任务**: 部署时，现有的 `history` 表中 `images='[]'` 且 `extra.taskId` 存在的记录不会自动迁移到 `tasks` 表。启动时 `recoverPending()` 只检查 `tasks` 表。需要一个一次性迁移脚本或后台任务，从 `history` 读取 pending 视频记录并插入到 `tasks` 表（启动时运行一次）。

## 附录：与方案 B（直接扩展 TaskManager）的区别

| 方面 | 方案 A（TaskQueue + SQLite） | 方案 B（TaskManager 扩展） |
|------|---------------------------|--------------------------|
| 持久化 | SQLite 持久化 | JSON 文件或新增轻量持久化 |
| 重启恢复 | 自动恢复轮询 | 需要额外实现 |
| 统一 SSE | 自然统一 | 需要修改路由 |
| 图片异步 | 通过 Worker 统一 | 需要单独实现 |
| 复杂度 | 中等（新增 3 个文件） | 低（修改 1 个文件） |
| 可扩展性 | 高（任务树、优先级、队列管理） | 低（专为视频设计） |

方案 A 选择的理由：统一模型带来的长期维护收益远高于初期成本。所有生成任务使用相同的生命周期管理、持久化、错误处理和 SSE 推送逻辑。
