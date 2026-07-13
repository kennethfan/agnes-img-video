# frontend/src/components/ — 共享 UI 组件

**14 个组件** — 跨视图复用的 UI 片段。

## 组件索引

| 组件 | 文件 | 用途 |
|------|------|------|
| `NavSidebar` | `NavSidebar.vue` | 左侧导航栏 — 接收 `activePage` prop，emit `navigate` 事件 |
| `ImageResult` | `ImageResult.vue` | 图片画廊 — 预览 + 下载 + 保存到作品库 + 上传到 GitHub |
| `ShotCard` | `ShotCard.vue` | 分镜卡片 — 显示分镜镜头（prompt/type/refImage） |
| `AssetCard` | `AssetCard.vue` | 作品卡片 — 收藏、选择、批量操作 |
| `TaskProgress` | `TaskProgress.vue` | 任务进度条 — SSE 驱动的进度显示 |
| `IdeateStep` | `IdeateStep.vue` | 创意发想步骤 — textarea 输入 → AI 生成项目简介 |
| `GenStep` | `GenStep.vue` | 生成步骤 — 从简介批量生图 |
| `RefineStep` | `RefineStep.vue` | 优化步骤 — 精修图片 + 提示词 |
| `FinalStep` | `FinalStep.vue` | 定稿步骤 — 选择封面 + 标记完成 |
| `StepProgressBar` | `StepProgressBar.vue` | 步骤进度条 — 展示项目 4 步进度 |
| `ProjectStatsCards` | `ProjectStatsCards.vue` | 仪表盘统计卡片 — 图片/视频/文件数量 |
| `ProjectFileGrid` | `ProjectFileGrid.vue` | 仪表盘文件网格 — 分类筛选的媒体文件展示 |
| `AssetPickerDialog` | `AssetPickerDialog.vue` | 作品库选择器弹窗 — 跨步骤选取已有图片 |
| `AIPanel` | `AIPanel.vue` | AI 操作面板 — 通用 AI 交互界面 |

## 约定

- 所有组件使用 `<script setup>` + Composition API
- Props 导出类型接口（定义在 `src/types/index.ts` 或组件内 `defineProps`）
- 纯展示组件不引用 Pinia store — 由父视图注入 props
- 步骤组件（IdeateStep/GenStep/RefineStep/FinalStep）可直接调用 `src/api/*` 发送请求
- 步骤组件通过 emit (`@brief-generated`, `@generated`, `@updated`) 向 ProjectEditor 回传数据

## 反模式

- 纯展示组件（卡片/进度条）不要直接 import `src/api/*`
- 不要在 `ImageResult` 外重复图片预览逻辑 — 复用此组件
- 步骤组件不要自己管理导航 — 只 emit 事件，由 ProjectEditor 控制步骤切换