# 创作区项目式设计设计文档

> 将「创作」导航组重构为项目式闭环：从创意简报到最终出品的端到端流程。

---

## 现状

当前「创作」导航组（NavSidebar 中的 `workflow` 组）包含三个独立 wizard：

| 页面 | 路由 | 组件 |
|------|------|------|
| 图片精修 | `/image-refine` | WorkflowWizard (4 步) |
| 漫画生成 | `/comic` | WorkflowWizard (6 步) |
| 小说生成 | `/novel` | WorkflowWizard (8 步) |

另外「图片」导航组下还有文生图/图生图/批量生成三个独立页面，与创作流程割裂。

### 问题

1. **流程碎片化**：文生图 → 图生图精修 → 保存需要手动跳转，状态无法跨页面保持
2. **无项目概念**：一次创作活动没有上下文管理，历史混杂在全局历史中
3. **无法迭代**：生成结果不能回到源项目继续修改，整体是线性而非闭环
4. **Wizard 过重**：现有 WorkflowWizard 对简单创作场景来说步骤太多

### 设计约束

- 只改造「创作」导航组，不动 `图片/视频/工具/作品/系统` 组
- 保持现有「作品库」「模板管理」「历史记录」等已有功能不动
- 新功能使用新的 SQLite 表，独立于已有模型

---

## 方案：项目式 4 步创作闭环

### 核心概念

**项目（Project）** = 一次创作活动，从创意到出品的完整闭环。

```
创意简报 → 生成+对比 → 精修 → 定稿
```

每个项目包含：
- 基本信息（标题、创意简报、AI 推荐参数）
- 多个步骤（生成轮次、精修轮次）
- 定稿结果（最终文件 + 自动入库）

### 导航结构变化

**当前「创作」组：**
```
⚡ 创作
  ├── 图片精修
  ├── 漫画生成
  └── 小说生成
```

**改为：**
```
⚡ 创作
  └── 创作项目
```

点击「创作项目」进入项目列表页，展示所有项目的卡片式列表。点击项目进入项目编辑器（4 步流程）。右上角有「新建项目」按钮。

### 4 步流程详细

#### 第 1 步：创意简报（Creative Brief）

用户输入：
- **一句话描述**（必填）：例如「一只橘猫在樱花树下睡觉」
- **目标平台/用途**（选填）：社媒封面 / 海报 / 电商主图 / ...

AI 推荐（复用现有 `POST /ideas/expand` 或 `/chat/completions`）：
- **风格推荐**：基于描述推荐 3-5 种风格（写实/水彩/赛博朋克/...）
- **尺寸推荐**：根据用途推荐尺寸（1:1 / 16:9 / 3:4 / ...）
- **模型推荐**：推荐适合的图片模型
- **Prompt 扩写**：将一句话扩写成详细的 AI 生图提示词
- **模板推荐**：从已有 PromptTemplate 中匹配推荐

用户确认/修改后进入下一步。

#### 第 2 步：生成+对比（Generate & Compare）

参数面板（继承上一步的推荐值，可手动修改）：
- Prompt / Negative Prompt
- Size / Strength / Model
- 生成数量（1-4 张）

生成后并排展示缩略图，每张卡片支持：
- 收藏（选定候选）
- 精修（进入第 3 步）
- 查看更多
- 保存到作品库

选中 1-2 张候选进入精修步骤。

#### 第 3 步：精修（Refine）

基于选中的图片进行迭代优化（相当于图生图 + 参数调整）：
- 输入图固定为选中的候选图
- 可修改 Prompt / Strength / Size
- 生成新版本
- 历史版本可回溯比较

支持多次迭代，每次生成记录保留在项目时间线中。

#### 第 4 步：定稿（Finalize）

确认最终版本：
- 展示最终图 + 所有参数
- 可添加备注/标签
- **自动保存到作品库**（调用现有 `POST /api/v1/assets`）
- 项目状态改为 `completed`

定稿后项目可在列表中查看/复制。

---

## 数据模型

### `projects` 表

```go
type Project struct {
    ID          int64      `gorm:"primaryKey"`
    Title       string     `gorm:"size:200"`
    Brief       string     `gorm:"type:text"`          // 原始创意简报
    AIEnhanced  string     `gorm:"type:text"`          // AI 增强后的推荐结果 (JSON)
    Status      string     `gorm:"size:20;default:draft"` // draft | generating | refining | completed
    CoverURL    string     `gorm:"type:text"`           // 定稿封面图
    FinalURL    string     `gorm:"type:text"`           // 最终输出文件 URL
    AssetIDs    string     `gorm:"type:text"`           // 关联作品库资产 ID 列表 (JSON array)
    Notes       string     `gorm:"type:text"`           // 用户备注
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### `project_steps` 表

```go
type ProjectStep struct {
    ID        int64      `gorm:"primaryKey"`
    ProjectID int64      `gorm:"index"`
    StepType  string     `gorm:"size:20"`  // generate | refine | finalize
    Position  int        // 步骤序号
    Input     string     `gorm:"type:text"` // 输入参数 (JSON)
    Output    string     `gorm:"type:text"` // 输出结果 (JSON)
    CreatedAt time.Time
}
```

**`Input` JSON 结构（按 StepType）：**

```typescript
// generate: 生成步骤的输入
{ "prompt": "...", "negative_prompt": "...", "size": "1:1", "strength": 0.75, "model": "...", "count": 4 }

// refine: 精修步骤的输入（继承上一张图的参数）
{ "image_url": "https://...", "prompt": "...", "strength": 0.5, "size": "1:1", "model": "..." }

// finalize: 定稿步骤的输入
{ "notes": "最终版本", "tags": ["封面", "3月活动"] }
```

**`Output` JSON 结构（按 StepType）：**

```typescript
// generate: 生成结果（可能多张）
{ "images": ["url1", "url2"], "selected": ["url1"] }

// refine: 精修结果
{ "images": ["url1", "url2", ...], "selected": "url1" }

// finalize: 定稿确认
{ "final_url": "https://...", "asset_id": 42 }
```

---

## 模块拆分

### 模块 A：后端 API（新建文件）

| 文件 | 内容 |
|------|------|
| `internal/repository/gorm/project.go` | ProjectRepository + ProjectStepRepository |
| `internal/repository/gorm/models.go` | 添加 Project + ProjectStep 模型 |
| `internal/repository/gorm/gorm.go` | AutoMigrate |
| `internal/handler/project.go` | ProjectHandler CRUD + Step management |
| `internal/repository/interfaces.go` | 添加接口定义 |
| `cmd/server/main.go` | 注册路由 |

**API 端点：**

```
GET    /api/v1/projects           — 项目列表
POST   /api/v1/projects           — 创建项目（含创意简报）
GET    /api/v1/projects/:id       — 项目详情（含步骤）
PUT    /api/v1/projects/:id       — 更新项目（标题/备注/状态）
DELETE /api/v1/projects/:id       — 删除项目
POST   /api/v1/projects/:id/duplicate — 复制项目
POST   /api/v1/projects/:id/steps — 添加步骤
PUT    /api/v1/projects/:id/steps/:stepId — 更新步骤
DELETE /api/v1/projects/:id/steps/:stepId — 删除步骤
POST   /api/v1/projects/:id/ai-recommend — AI 推荐
POST   /api/v1/projects/:id/finalize     — 定稿（保存到作品库）
```

### 模块 B：前端页面（新建文件）

| 文件 | 内容 |
|------|------|
| `src/api/projects.ts` | Project API 客户端 |
| `src/views/ProjectList.vue` | 项目列表页（入口） |
| `src/views/ProjectEditor.vue` | 项目编辑器（4 步流程容器） |
| `src/components/ProjectBrief.vue` | 步骤 1：创意简报 + AI 推荐 |
| `src/components/ProjectGenerate.vue` | 步骤 2：生成+对比 |
| `src/components/ProjectRefine.vue` | 步骤 3：精修 |
| `src/components/ProjectFinalize.vue` | 步骤 4：定稿 |

### 模块 C：导航修改

- `NavSidebar.vue` — 将 `workflow` 组的 3 项合并为 1 个「创作项目」入口
- `App.vue` — 替换 3 个 `WorkflowWizard` 路由为 `ProjectList` + `ProjectEditor`
- `router/index.ts` — 更新路由配置

---

## AI 推荐集成

复用现有 `/v1/chat/completions`（通过 `AgnesClient.Chat()`）：

```
POST /api/v1/projects/:id/ai-recommend
Request: { "brief": "一只橘猫在樱花树下睡觉", "platform": "社媒封面" }
Response: {
  "enhanced_prompt": "一只毛色橘黄的...",
  "style_suggestions": ["写实摄影", "水彩", "吉卜力动画"],
  "size_suggestion": "1:1",
  "model_suggestion": "agnes-image-2.1-flash",
  "template_matches": [...]
}
```

系统 prompt 预设，引导 AI 从风格/尺寸/模型/prompt 四个维度给出建议。

---

## 不做的事

- 不动现有「图片/视频/工具/作品/系统」导航组
- 不动 WorkflowWizard.vue（保留旧代码，但不再从导航访问）
- 不动现有生成 API（`/images/*`、`/videos/*`）
- 不动作品库、模板管理、历史记录等已有功能
- 不动 Pinia redo store 模式（新项目独立管理状态）

---

## 实施顺序

1. **后端**：Project + ProjectStep 模型、Repository、Handler、路由注册
2. **前端**：ProjectList 列表页
3. **前端**：ProjectEditor + 4 个子组件（Brief → Generate → Refine → Finalize）
4. **导航**：更新 NavSidebar + App.vue + router
