# Storyboard Studio Phase 2 — 剩余功能实现设计

**日期:** 2026-07-10
**基于:** `docs/superpowers/specs/2026-07-07-storyboard-design.md`（Phase 1 CRUD 已全部完成）

## 概述

Phase 1 实现了 Storyboard 的 CRUD（项目 + 镜头）和基础页面。Phase 2 补齐剩余 4 个功能模块，按 A→B→C→D 顺序逐一实施：

- **A**: 接入真实视频生成管线（Replace 伪 `GenerateShots`）
- **B**: 拖拽排序镜头
- **C**: 脚本自动拆分为镜头
- **D**: UI 视觉打磨

---

## A — 接入真实视频生成管线

### 现状问题

`POST /api/v1/storyboard/projects/:id/generate` 只统计了 pending shots 数量并返回，**没有实际提交任何视频生成任务**。

### 方案设计

#### 新增 `StoryboardGenerator` service

**文件:** `backend/internal/service/storyboard_generator.go`

```go
type StoryboardGenerator struct {
    client    *AgnesClient
    taskQueue *TaskQueue
    repo      repository.StoryboardRepository
}
```

#### 核心方法

```go
// GenerateAll 批量提交项目下所有 pending shots 到视频生成管线
func (g *StoryboardGenerator) GenerateAll(ctx context.Context, projectID int64) (*GenerateResult, error)

// GenerateOne 提交单个 shot 到视频生成管线
func (g *StoryboardGenerator) GenerateOne(ctx context.Context, shotID int64) error
```

#### 流程

1. `GenerateAll` 加载 project + 所有 pending shots
2. 对每个 shot:
   - 用 `shot.prompt` 调用 `AgnesClient.SubmitVideoTask()` 提交 text-to-video
   - 如果 `shot.reference_image` 非空，调用 image-to-video 模式
   - 拿到 `taskId`（视频任务 ID）
   - 在 `TaskQueue` 中注册任务记录（`TaskRecord`），`type = "shot_video"`
   - 设置 shot 的 `status = "generating"`, `task_id = taskId`
3. 返回已提交数量

#### 后台轮询 + 结果回写

- 利用已有 `TaskQueue` 的后台 goroutine 轮询 `TaskRecord` 状态
- 轮询完成后：
  - 下载视频（调用 `AgnesClient.DownloadVideo()`）
  - 更新 shot 的 `result_video` = 本地/远程 URL
  - 更新 shot 的 `status = "completed"`
  - 通过 SSE 推送进度到前端

#### 修改 GenerateShots handler

`backend/internal/handler/storyboard.go:254`:

```go
func (h *StoryboardHandler) GenerateShots(c *gin.Context) {
    // 原来只返回统计 → 改为调用 StoryboardGenerator.GenerateAll()
    result, err := h.generator.GenerateAll(c.Request.Context(), projectID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusAccepted, gin.H{
        "submitted": result.Submitted,
        "total":     result.Total,
    })
}
```

#### StoryboardHandler 变更

- 构造函数 `NewStoryboardHandler` 增加 `generator *StoryboardGenerator` 参数
- 在 `cmd/server/main.go` 中注入 generator

#### 前端变更

`frontend/src/views/Storyboard.vue`:
- "批量生成"按钮调用 `generateShots()` 后轮询 SSE
- 每个 shot 在生成中显示 `TaskProgress` 进度条
- 完成后自动更新 shot 状态和预览

**文件列表:**

| 文件 | 变更 |
|------|------|
| `backend/internal/service/storyboard_generator.go` | **新增** |
| `backend/internal/handler/storyboard.go` | 修改 `GenerateShots` + 注入 generator |
| `backend/internal/model/types.go` | 新增 `GenerateResult` 类型 |
| `backend/cmd/server/main.go` | 注入 generator |
| `frontend/src/views/Storyboard.vue` | 接入真实生成流程 + SSE |
| `frontend/src/api/storyboard.ts` | 更新 generateShots 返回类型 |

---

## B — 拖拽排序镜头

### 现状

后端 `ReorderShots` API 已实现（`PUT /storyboard/projects/:id/shots/reorder`），接受 `{ids: [1,2,3]}` 数组批量更新 `sort_order`。前端只是静态列表。

### 方案

使用 **HTML5 Drag & Drop API**（零依赖，保持项目简洁）。

#### ShotCard 组件变更

`frontend/src/components/ShotCard.vue`:

```vue
<template>
  <div
    class="shot-card"
    :draggable="true"
    @dragstart="onDragStart"
    @dragover.prevent="onDragOver"
    @dragleave="onDragLeave"
    @drop="onDrop"
    :class="{ 'drag-over': isDragOver }"
  >
    ...
  </div>
</template>
```

#### Storyboard.vue 变更

- `handleDrop(fromIndex, toIndex)` — 重新排列 `shots[]` 数组
- 调用 `storyboardApi.reorderShots(projectId, shots.map(s => s.id))` 持久化
- 视觉反馈：拖拽时半透明，目标位置显示蓝色占位线

#### 实现细节

- `onDragStart`: 设置 `dataTransfer.effectAllowed = "move"`，存储拖拽 shot 的 ID
- `onDragOver`: 显示目标位置指示器
- `onDrop`: 交换数组位置，调用 API
- 为防止频繁 API 调用，只在 drop 时提交一次（不中途保存）

**文件列表:**

| 文件 | 变更 |
|------|------|
| `frontend/src/components/ShotCard.vue` | 添加拖拽事件处理 |
| `frontend/src/views/Storyboard.vue` | 添加 drop handlers + API 调用 |
| `frontend/src/views/Storyboard.vue` | CSS 拖拽样式 |

---

## C — 脚本自动拆分为镜头

### 现状

`POST /api/v1/comic/generate-prompts` 已有（通过 `AgnesClient.GenerateComicPrompts()` 调用 chat API 生成分格提示词）。但此功能未集成到 storyboard。

### 方案

#### 新增 Storyboard 内部的脚本导入能力

后端:

- **不新增 API** — 复用现有的 `POST /comic/generate-prompts`
- 在 `StoryboardHandler` 新增 `ImportScript` handler:
  - `POST /api/v1/storyboard/projects/:id/import-script`
  - 接受 `{script: string}`（用户输入的完整脚本）
  - 调用 `AgnesClient.GenerateComicPrompts(script, "auto", panelCount)` → 返回 prompt 数组
  - 批量创建 shots（每个 prompt 一个 shot）
  - 返回创建的 shots 列表

或者更简单的方案：

- **不涉及后端 chat API** — 前端直接按段落/句子分割脚本
- 在 Storyboard 项目详情页添加"从脚本导入"按钮
- 弹出对话框：多行文本框输入原始脚本
- 前端按 `\n\n`（段落）分割为多个 prompt，批量创建 shots

#### 选择方案

前端分割方案更简单、零延迟、不消耗 API 额度。如果用户需要 AI 润色再看 chat API 方案。

#### 前端实现

`frontend/src/views/Storyboard.vue`:

```vue
<template>
  <el-button @click="showImportDialog = true">
    <el-icon><Document /></el-icon>
    从脚本导入
  </el-button>
  
  <el-dialog v-model="showImportDialog" title="从脚本导入镜头">
    <el-input
      type="textarea"
      v-model="importScript"
      :rows="10"
      placeholder="每行/每个段落将生成一个镜头..."
    />
    <el-radio-group v-model="splitMode">
      <el-radio value="paragraph">按段落分割</el-radio>
      <el-radio value="line">按行分割</el-radio>
    </el-radio-group>
    <template #footer>
      <el-button @click="doImport">导入并创建 {{ previewCount }} 个镜头</el-button>
    </template>
  </el-dialog>
</template>
```

**文件列表:**

| 文件 | 变更 |
|------|------|
| `frontend/src/views/Storyboard.vue` | 新增导入对话框 + 分割逻辑 + 批量创建 API 调用 |
| `frontend/src/api/storyboard.ts` | 可能新增 `batchCreateShots` 方法（或在 handler 端批量接收） |

#### 后端批量创建 API

新增 `POST /api/v1/storyboard/projects/:id/shots/batch`:

```go
type BatchCreateShotsRequest struct {
    Shots []CreateShotItem `json:"shots"`
}
```

在 handler 中批量插入，减少 N 次 DB round-trip。

---

## D — UI 视觉打磨

### 现状

`Storyboard.vue` 使用基础 Element Plus 表格布局，`ShotCard` 是极简卡片。项目列表是垂直列表。

### 方案

#### 项目列表 → 卡片网格

参考 `Assets.vue` 的网格风格：

- 项目列表改为 `el-row` + `el-col` 卡片布局（每行 2-3 列）
- 每个卡片显示：项目名、创建时间、镜头数量、最近更新
- 悬停效果 + 操作按钮（编辑、删除、复制）

#### 项目详情 → 时间线式布局

- Shots 横向排列（若屏幕宽）或竖向时间线
- 每个 ShotCard 显示：
  - 序号（大号数字）
  - Prompt 预览（截断显示）
  - 缩略图（如有 `result_video` 或 `reference_image`）
  - 状态标签（pending/generating/completed + 对应颜色）
  - 操作按钮（编辑、删除、重新生成）
- 生成中状态：显示动画或进度条

#### ShotCard 改进

`frontend/src/components/ShotCard.vue`:

- 增加 `status` 驱动的视觉状态：
  - `pending`: 灰色虚线边框，显示"待生成"标签
  - `generating`: 蓝色动画边框 + 旋转图标 + 进度条
  - `completed`: 绿色边框 + 视频缩略图 + 播放按钮
- 视频结果可点击预览（复用现有播放能力）
- 拖拽排序的视觉反馈（拖拽时半透明 + 目标位置蓝色指示线）

#### 响应式

- 宽屏：shots 网格排列（2-4 列，取决于屏幕宽度）
- 窄屏：shots 垂直排列

**文件列表:**

| 文件 | 变更 |
|------|------|
| `frontend/src/views/Storyboard.vue` | 项目列表卡片化 + 详情页时间线布局 + 响应式 |
| `frontend/src/components/ShotCard.vue` | 状态视觉 + 缩略图 + 拖拽样式 |

---

## 实现顺序（Phase 2）

```
A（生成管线）→ B（拖拽排序）→ C（脚本导入）→ D（视觉打磨）
```

| 步骤 | 依赖 | 预计新增文件 | 预计修改文件 |
|------|------|------|------|
| A | Phase 1 CRUD | 1 | 4 |
| B | 无 | 0 | 2 |
| C | 无（可选后端 batch API） | 0 | 2 |
| D | A 完成后效果可见 | 0 | 2 |

**总预估**: 1 新增 + 5-6 修改文件（含前端 + 后端）

---

## 风险与注意事项

1. **视频生成时长**: 每个 shot 的视频生成可能需要几分钟。TaskQueue 的轮询超时 30 分钟，多个 shot 并发可能耗尽配额。建议限制单个项目的并发生成数（max 3 个 shot 同时生成）。
2. **API 额度**: 大量 shot 批量生成会消耗大量 API 调用次数。建议前端给出预计消耗提示。
3. **数据库一致性**: 生成完成后回写 shot 状态需使用事务，防止部分更新。
4. **现有 24/25 完成状态**: Phase 1 已有 24/25 项完成，Phase 2 完成后所有 25 项全部闭环。
