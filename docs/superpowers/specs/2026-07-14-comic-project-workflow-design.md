# 漫画项目工作流设计

**日期:** 2026-07-14
**状态:** 设计稿
**关联:** ProjectEditor, ProjectDashboard, comic wizard

## 概述

将漫画创作改造为 Project 的子类型，复用现有的 Project/ProjectStep 数据模型和 ProjectEditor 的步骤流程，取代当前 WorkflowWizard 中的独立漫画向导。

## 数据模型

### Project 表新增字段

```go
type Project struct {
    // ... 现有字段
    Type      string `json:"type"`       // "project" | "comic"
    ComicData string `json:"comic_data"` // JSON: { layout, panels: [...] }
}
```

### ComicData JSON 结构

```json
{
  "layout": "quad",
  "storyline": "",
  "characters": "",
  "style": "",
  "panels": [
    { "prompt": "", "image": "", "caption": "", "refImage": "" }
  ]
}
```

### Steps 定义（5 步）

| StepType | Position | Input (JSON) | Output (JSON) |
|----------|----------|-------------|---------------|
| `ideate` | 0 | `{ theme }` | `{ storyline, characters, style }` |
| `layout` | 1 | `{ layout }` | `{ panels: [{prompt, refImage}] }` |
| `generate` | 2 | `{ panels }` | `{ panels: [{prompt, image, caption}] }` |
| `refine` | 3 | `{ panels }` | `{ panels: [{prompt, image, caption}] }` |
| `finalize` | 4 | `{ panels }` | `{ coverUrl, exportFormat }` |

panels 数据流经 Step Input/Output，同时在 Project.ComicData 保存完整快照。

### 兼容性

- 不新增数据库表
- `type` 字段默认值 `"project"`，现有项目不受影响
- Step 的 Input/Output 本来就是 JSON 字段，无需迁移

## 步骤 UI 流程

ProjectEditor 根据 `project.type` 渲染不同步骤：

```
project.type === 'project' → 现有 4 步（Ideate/Gen/Refine/Final）
project.type === 'comic'  → 新 5 步（Ideate/Layout/Gen/Refine/Final）
```

### Step 0 — ComicIdeateStep（构思）

- 用户输入漫画主题 textarea
- AI 生成：故事线（storyline）、角色简介、画风建议
- 调用 `POST /comic/generate-storyline`（chat completions）
- 输出写入 Step Input/Output，同时更新 Project.ComicData

### Step 1 — ComicLayoutStep（布局）

- 展示网格布局选择：单格 / 双格 / 四格 / 六格
- 选定后 AI 自动生成每格画面提示词（复用 `POST /comic/generate-prompts`）
- 用户可手动编辑每个格子的 prompt
- 合并现有 StepLayout.vue + StepPanels.vue 功能

### Step 2 — ComicGenStep（生成）

- 遍历每个格子调用 text-to-image 批量生图
- 显示每个格子的生成进度（当前格/总格数）
- 支持跳过已有图片的格子（重新生成时）
- 修改现有 StepGenerate.vue 为可被 ProjectEditor 加载的组件

### Step 3 — ComicRefineStep（精修 + 台词）

- 显示所有已生成的格子图片
- 点击单个格子弹出编辑面板：重新生成、编辑提示词、编辑台词
- 支持局部精修（图生图 refine），参考 ImageToImage 的双输入模式
- 合并现有 StepCaptions.vue 功能

### Step 4 — ComicFinalStep（定稿 + 导出）

- 完整漫画预览（现有 StepExport 预览区样式）
- 选择封面图片
- 导出：HTML 单文件 / 单张 PNG（html2canvas）
- 步骤完成后标记 Project.Status = "completed"

## 前端改动

### 新增组件

| 文件 | 说明 |
|------|------|
| `src/views/comic/ComicIdeateStep.vue` | 构思步骤 — 复用 IdeateStep 模式，调用 AI 生成剧本 |
| `src/views/comic/ComicLayoutStep.vue` | 布局步骤 — 合并现有 StepLayout + StepPanels |
| `src/views/comic/ComicRefineStep.vue` | 精修步骤 — 合并 StepCaptions + 精修逻辑 |
| `src/views/comic/ComicFinalStep.vue` | 定稿步骤 — 基于 StepExport 改造 |

### 修改组件

| 文件 | 改动 |
|------|------|
| `src/views/ProjectEditor.vue` | 根据 `project.type` 切换步骤模板（4 步 vs 5 步） |
| `src/views/comic/StepGenerate.vue` | 改为可被 ProjectEditor 加载的子组件，接收 panels prop |
| `src/views/ProjectList.vue` | 项目卡片显示 type 标签（"漫画"/"创作"） |
| `src/views/ProjectDashboard.vue` | type='comic' 时显示漫画特化统计（总格数、已生成、有台词） |
| `src/stores/wizard.ts` | 漫画相关状态可逐步迁移到 Project ComicData，wizard store 保留向后兼容 |

### 路由

无新增。漫画项目复用现有 Project 路由：
- `/projects` — ProjectList
- `/projects/:id` — ProjectEditor（根据 type 渲染不同步骤）
- `/projects/:id/dashboard` — ProjectDashboard（type 感知）

## 后端改动

### 新增 API

| 方法 | 路由 | Handler | 说明 |
|------|------|---------|------|
| `POST` | `/comic/generate-storyline` | `ComicHandler.GenerateStoryline` | AI 生成漫画剧本 — 参考 ExpandIdea 的 chat completions 模式 |

请求体：
```json
{ "theme": "一只猫的太空冒险", "style": "日式漫画" }
```

响应：
```json
{ "storyline": "...", "characters": "...", "style": "..." }
```

### 修改 Handler

**`ProjectHandler.CreateProject`**：
- 接收 `type` 字段
- `type === "comic"` 时创建 5 步模板 steps
- `type === "project"`（或未传）保留现有 4 步行为

### 复用逻辑

- 漫画项目的 Steps CRUD 完全复用现有 `PUT /steps/:stepId` 等接口
- AI 生成提示词复用现有 `POST /comic/generate-prompts`
- 图片生成复用现有 `POST /images/text-to-image`
- 导出为纯前端操作，不涉及后端

## Dashboard 与项目列表

### ProjectList 改动

- 项目卡片显示 type 标签：`"漫画"` / `"创作"`
- 漫画卡片进度条旁显示 `"N 格 / M 格已生成"`

### ProjectDashboard 改动（type === 'comic'）

| 卡片 | 说明 |
|------|------|
| StatsCards | 总格数、已生成图片数、有台词的格数 |
| FileGrid | 所有 panel 的图片 + 提示词，可点击查看大图 |
| StepProgress | 5 步进度显示（indicate/布局/生成/精修/定稿） |

复用现有 `ProjectStatsCards` 和 `ProjectFileGrid`，入参适配漫画数据。

## 导出

纯前端实现，无后端依赖：

- **HTML**：单文件，图片使用 URL 引用（base64 可选）
- **PNG**：使用 html2canvas 将预览区导出为图片
- 导出按钮在 FinalStep 中，完成后不跳转

## 不作的事

- 不新增数据库表
- 不影响现有 Project 的 CRUD 逻辑
- 不改动路由系统
- 不改动 WizardStore 的现有逻辑（逐步迁移，不破坏已有功能）
- 不改动认证/中间件
- 不改动视频相关功能

## 实施顺序

1. 后端：Project 加 type + ComicData 字段；新增 generate-storyline API
2. 数据模型：创建漫画项目时生成 5 步模板
3. 前端：新增 4 个漫画步骤组件
4. 集成：修改 ProjectEditor 支持 type 切换
5. Dashboard：ProjectList + ProjectDashboard 适配漫画类型
6. 导出：完善 HTML/PNG 导出
