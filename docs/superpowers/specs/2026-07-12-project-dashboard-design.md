# 项目仪表盘设计规格

> **功能**: 项目进度管理 + 生成文件聚合查看
> **方案**: B — 项目仪表盘
> **状态**: 设计已确认

## 1. 数据模型变更

### 1.1 TaskRecord（task_queue 表）

```go
// 新增字段
ProjectID int64 `json:"project_id" gorm:"column:project_id;index"`
```

### 1.2 History（history 表）

```go
// 新增字段
ProjectID int64 `json:"project_id" gorm:"column:project_id;index"`
```

### 1.3 Asset（assets 表）

```go
// 新增字段
ProjectID int64 `json:"project_id" gorm:"column:project_id;index"`
```

### 1.4 Project（projects 表）

```go
// 新增字段
StepProgress string `json:"step_progress" gorm:"type:text"`
// JSON 内容：{"ideate":"completed","generate":"in_progress","refine":"pending","finalize":"pending"}
```

### 1.5 迁移策略

所有新字段使用 GORM AutoMigrate 自动添加（ALTER TABLE ADD COLUMN）。
存量数据 `project_id` 为 0，表示未关联项目。

## 2. 新增 API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/projects/:id/files` | 聚合返回项目关联的所有文件（来自 history + assets） |
| GET | `/api/v1/projects/:id/stats` | 返回项目统计：步骤状态、文件数、任务执行中数、最后活动时间 |
| PUT | `/api/v1/projects/:id/step-progress` | 更新步骤进度 `{ step: "generate", status: "completed" }` |
| POST | `/api/v1/projects/:id/tasks` | 创建任务时携带 `project_id`（由前端传入任务 body） |

### GET /projects/:id/files 响应

```json
{
  "files": [
    {
      "id": 1,
      "type": "image",
      "source": "history",
      "url": "/outputs/xxx.png",
      "prompt": "...",
      "step": "generate",
      "created_at": "2026-07-12T10:00:00Z"
    }
  ]
}
```

### GET /projects/:id/stats 响应

```json
{
  "file_count": 12,
  "optimized_count": 4,
  "running_tasks": 3,
  "last_activity": "2026-07-12T10:15:00Z",
  "step_progress": {
    "ideate": "completed",
    "generate": "in_progress",
    "refine": "pending",
    "finalize": "pending"
  }
}
```

### PUT /projects/:id/step-progress 请求

```json
{
  "step": "generate",
  "status": "completed"
}
```

## 3. 前端变更

### 3.1 项目仪表盘页面（新）

**位置**: `frontend/src/views/ProjectDashboard.vue`

**组件结构**:

```
ProjectDashboard
├── StepProgressBar        — 4 步骤状态条（✓ 当前/待办）
├── StatsCards             — 统计卡片（文件数/优化数/任务数/最后活动）
├── FileGrid               — 文件缩略图网格（Tab 筛选：全部/图片/视频）
│   └── FileCard           — 单个文件卡片（预览/步骤标签/操作按钮）
└── TaskProgressPanel      — 实时任务进度（SSE 订阅，进度条 + 状态文本）
```

### 3.2 ProjectEditor 集成

- ProjectEditor 每个步骤完成时调用 `PUT /step-progress` 更新进度
- 编辑器中增加"查看仪表盘"按钮，跳转到 ProjectDashboard

### 3.3 ProjectList 增强

- 卡片增加文件数显示、最后活动时间
- 状态标签更丰富（显示步骤进度缩写）

### 3.4 新增 API 客户端

- `frontend/src/api/projects.ts` — 新增 `getProjectFiles()`、`getProjectStats()`、`updateStepProgress()`

### 3.5 新增类型

```typescript
interface ProjectFile {
  id: number
  type: 'image' | 'video'
  source: 'history' | 'asset'
  url: string
  prompt: string
  step: string
  created_at: string
}

interface ProjectStats {
  file_count: number
  optimized_count: number
  running_tasks: number
  last_activity: string
  step_progress: Record<string, string>
}
```

## 4. 路由

```
/projects/:id/dashboard  → ProjectDashboard
```

Vue Router 新增路由 `/projects/:id/dashboard`，ProjectEditor 中通过按钮导航过去。

## 5. step_progress 状态枚举

| 状态 | 含义 |
|------|------|
| `pending` | 未开始 |
| `in_progress` | 进行中 |
| `completed` | 已完成 |

步骤标识：`ideate` / `generate` / `refine` / `finalize`

## 6. 数据流

```
  生成/优化操作（ProjectEditor / GenStep / RefineStep）
  → 前端传入当前 project.id 到任务创建接口
  → ProjectEditor 调用 PUT /step-progress 更新步骤状态
  → TaskRecord 创建时携带 project_id（由 handlers/image.go、handlers/video.go 传入 task_queue）
  → 仪表盘 GET /stats 聚合统计
  → 仪表盘 GET /files 聚合文件列表
  → TaskProgressPanel SSE 订阅 /tasks/:id/stream 显示实时进度
```

## 7. 不包含的内容

- 拖拽改状态（看板方案 C 的内容）
- 文件删除/编辑操作
- 项目级权限控制
