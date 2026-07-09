# Vue Router 接入设计

> 为 Agnes Creator Studio 前端添加基于 hash 模式的 URL 路由，使每个页面有独立的 URL。

**状态**: 设计已批准

---

## 目标

当前前端使用 `activePage` 变量 + `v-if` 条件渲染切换页面，URL 从不变化。目标是在不破坏现有结构的前提下，为所有页面赋予独立的 hash URL，支持浏览器前进/后退。

## 约束

- 选择 **Vue Router hash 模式**（无需后端 fallback 配置）
- 采用 **惰性迁移策略**：保留 `App.vue` 现有 `v-if` 渲染链，新增路由层作为 URL 同步层
- 保持路由单向同步：**URL → activePage**，不反向破坏现有逻辑
- `vue-router` 已在 `package.json` 中，无需新增依赖

## 路由表

**文件**: `frontend/src/router/index.ts`

所有组件使用懒加载（`() => import(...)`），path 使用连字符风格：

| 页面 ID | 路由 path | 组件 |
|---------|-----------|------|
| `text2img` | `/` → redirect `/text2img` | TextToImage.vue |
| `img2img` | `/img2img` | ImageToImage.vue |
| `batch` | `/batch` | BatchGen.vue |
| `script_gen` | `/script-gen` | ScriptGen.vue |
| `text2vid` | `/text2vid` | TextToVideo.vue |
| `img2vid` | `/img2vid` | ImageToVideo.vue |
| `multi_vid` | `/multi-vid` | MultiImageVideo.vue |
| `ideas` | `/ideas` | Ideas.vue |
| `storyboard` | `/storyboard` | Storyboard.vue |
| `assets` | `/assets` | Assets.vue |
| `tasks` | `/tasks` | TaskRecords.vue |
| `history` | `/history` | History.vue |
| `access_logs` | `/access-logs` | AccessLogs.vue |
| `db_manage` | `/db-manage` | DBManage.vue |
| wizard | `/wizard/:type` | WorkflowWizard.vue |

## App.vue 改造

- `main.ts`: 注册 router（`app.use(router)`）
- **不**使用 `<router-view>` — 渲染仍由 `v-if` 控制（惰性迁移的核心）
- 新增 `watch(() => route.name, ...)` 同步 `activePage`
- 新增 `navigateTo(page)` 函数，调用 `router.push({ name: page })`
- 移除 `redo-trigger` 自定义事件监听（由路由替代）
- view 的静态 import 保留（`v-if` 需要它们）
- `v-if` 渲染链保持不变

## NavSidebar 改造

- `selectPage(pageId)` 改为调用父组件的 `navigateTo`（通过 emit 或 props）
- 移除 `defineModel<string>('activePage')`，改为 `defineProps<{ activePage: string }>()` + `emit('navigate', pageId)`

## Redo Store 改造

- `setRedoData` 设置 `targetTab` 后，改为 `router.push({ name: targetTab })`
- 移除 `window.dispatchEvent(new CustomEvent('redo-trigger'))`
- `App.vue` 移除 `redo-trigger` 事件监听

## 保留不变的部分

- `v-if` 条件渲染（方案 C 惰性迁移的核心）
- `activePage` 状态变量
- 所有 view 组件本身

## 向后兼容

- `/#/text2img` 等 URL 可直接收藏、分享、刷新
- 浏览器前进/后退按钮正常工作
- 所有已有功能不受影响
