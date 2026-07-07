# Storyboard Studio — 分镜策划功能设计

## 概述

在 Agnes Creator Studio 中新增**分镜策划**功能，允许用户创建分镜项目、管理镜头序列、批量生成视频，弥补视频生产 pipeline 中「前期策划」环节的缺失。

### 核心流程

```
新建项目 → 创建镜头序列 → 调整顺序 → 批量生成 → 产出进入作品库
```

## 数据模型

### StoryboardProject 分镜项目

```json
{
  "id": 1,
  "title": "产品宣传片",
  "script": "完整的脚本内容（可选）",
  "created_at": "2026-07-07T10:00:00Z",
  "updated_at": "2026-07-07T10:30:00Z",
  "shot_count": 5
}
```

### StoryboardShot 镜头

```json
{
  "id": 1,
  "project_id": 1,
  "sequence": 1,
  "prompt": "一只猫在草地上悠闲地走路",
  "type": "text2video",
  "reference_image": "https://...（可选）",
  "status": "completed" | "generating" | "pending",
  "result_video": "https://...（生成结果）",
  "task_id": "ag_xxx（生成任务 ID）",
  "created_at": "2026-07-07T10:00:00Z"
}
```

### SQLite 表结构

```sql
CREATE TABLE IF NOT EXISTS storyboard_projects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL DEFAULT '',
    script TEXT DEFAULT '',
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS storyboard_shots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    sequence INTEGER NOT NULL DEFAULT 0,
    prompt TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL DEFAULT 'text2video',
    reference_image TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending',
    result_video TEXT DEFAULT '',
    task_id TEXT DEFAULT '',
    created_at TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (project_id) REFERENCES storyboard_projects(id) ON DELETE CASCADE
);
```

## API 设计

### 分镜项目 CRUD

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/v1/storyboard/projects` | 项目列表 |
| POST | `/api/v1/storyboard/projects` | 新建项目 |
| GET | `/api/v1/storyboard/projects/:id` | 项目详情（含所有镜头） |
| PUT | `/api/v1/storyboard/projects/:id` | 更新项目标题/脚本 |
| DELETE | `/api/v1/storyboard/projects/:id` | 删除项目（级联删除镜头） |
| POST | `/api/v1/storyboard/projects/:id/duplicate` | 复制项目 |

### 镜头管理

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/v1/storyboard/projects/:id/shots` | 添加镜头 |
| PUT | `/api/v1/storyboard/shots/:id` | 更新镜头 |
| DELETE | `/api/v1/storyboard/shots/:id` | 删除镜头 |
| PUT | `/api/v1/storyboard/projects/:id/shots/reorder` | 排序（传 ID 数组） |

### 生成

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/v1/storyboard/projects/:id/generate` | 批量生成所有待生成镜头 |

批量生成逻辑：
1. 遍历项目中所有 `pending` 状态的镜头
2. 逐个调用视频生成 API（复用现有 text-to-video / image-to-video 端点）
3. 每个镜头完成后更新该镜头的 `result_video` 和 `status`
4. 已有的 SSE 流 (`/api/v1/videos/stream/:taskId`) 可复用

## 前端设计

### 新 Tab 页
- 在 App.vue 新增 **"分镜"** tab-pane
- 新建 `frontend/src/views/Storyboard.vue`

### 页面布局

#### 项目列表视图
```
┌──────────────────────────────────────────┐
│ 分镜项目                    [+ 新建项目] │
├──────────────────────────────────────────┤
│ ┌──────────────────────────────────────┐ │
│ │ 📁 产品宣传片   5 镜头  [编辑] [删除]│ │
│ ├──────────────────────────────────────┤ │
│ │ 📁 社交媒体视频  3 镜头  [编辑] [删除]│ │
│ └──────────────────────────────────────┘ │
└──────────────────────────────────────────┘
```

#### 项目详情视图
```
┌──────────────────────────────────────────────────┐
│ [← 返回]  产品宣传片    [保存] [批量生成]          │
├──────────────────────────────────────────────────┤
│                                                  │
│  ┌─ Shot 1 ──────────────────────────────┐      │
│  │ prompt: "一只猫在走路..."               │      │
│  │ 类型: 文生视频 │ 状态: ✅ 已完成         │      │
│  │ [▶ 预览] [重新生成] [删除]              │      │
│  └────────────────────────────────────────┘      │
│                                                  │
│  ┌─ Shot 2 ──────────────────────────────┐      │
│  │ prompt: "猫跳上桌子..."               │      │
│  │ 类型: 图生视频 │ 状态: ⏳ 生成中       │      │
│  │ [▶ 预览]                              │      │
│  └────────────────────────────────────────┘      │
│                                                  │
│           [+ 添加镜头]                             │
└──────────────────────────────────────────────────┘
```

### 镜头添加弹窗

```
┌─ 添加镜头 ─────────────────────────────┐
│                                        │
│  提示词: [_________________________]   │
│  类型:   [文生视频 ▼]                   │
│  参考图: [选择图片]（可选）              │
│                                        │
│         [取消]    [添加并继续]  [添加]   │
└────────────────────────────────────────┘
```

### 状态说明
- `pending` — 待生成（灰色，显示"待生成"标签）
- `generating` — 生成中（蓝色，显示进度或旋转动画）
- `completed` — 已完成（绿色，显示播放按钮和结果）

## 组件结构

### 新增文件

| 文件 | 说明 |
|---|---|
| `frontend/src/views/Storyboard.vue` | 分镜首页（项目列表 + 项目详情两态） |
| `frontend/src/components/ShotCard.vue` | 镜头卡片组件 |
| `frontend/src/api/storyboard.ts` | 分镜 API 客户端 |
| `backend/internal/handler/storyboard.go` | 分镜 handler |
| `backend/internal/repository/storyboard.go` | 分镜仓库层 |

### 修改文件

| 文件 | 说明 |
|---|---|
| `frontend/src/App.vue` | 新增"分镜"选项卡 |
| `frontend/src/types/index.ts` | 新增 StoryboardProject/StoryboardShot 类型 |
| `backend/internal/model/types.go` | 新增分镜相关类型 |
| `backend/cmd/server/main.go` | 注册分镜路由 |

## 与现有系统的关系

### 复用

- **视频生成 API** — 镜头生成直接调用已有的 `/api/v1/videos/text-to-video` 和 `/api/v1/videos/image-to-video`
- **SSE 推送** — 镜头生成进度复用已有的 `/api/v1/videos/stream/:taskId`
- **作品库** — 生成的视频自动出现在作品库中
- **拖拽排序** — 镜头排序可复用 Element Plus `el-drag` 或原生 HTML5 Drag & Drop

### 不涉及

- 不修改现有的视频生成 API
- 不修改已有的作品管理和历史记录
- 独立的数据库表，不影响现有数据

## 实现顺序

1. 后端：数据模型 + 仓库层（项目 CRUD、镜头 CRUD）
2. 后端：Storyboard Handler + 路由注册
3. 前端：类型定义 + API 客户端
4. 前端：ShotCard 组件
5. 前端：Storyboard 页面 + App.vue 选项卡
6. 前端：批量生成集成
7. 集成测试与打磨
