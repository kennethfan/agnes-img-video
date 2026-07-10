# frontend/src/components/ — 共享 UI 组件

**5 个组件** — 跨视图复用的 UI 片段。

## 组件索引

| 组件 | 文件 | 用途 |
|------|------|------|
| `NavSidebar` | `NavSidebar.vue` | 左侧导航栏 — 接收 `activePage` prop，emit `navigate` 事件 |
| `ImageResult` | `ImageResult.vue` | 图片画廊 — 预览 + 下载 + 保存到作品库 + 上传到 GitHub |
| `ShotCard` | `ShotCard.vue` | 分镜卡片 — 显示分镜镜头（prompt/type/refImage） |
| `AssetCard` | `AssetCard.vue` | 作品卡片 — 收藏、选择、批量操作 |
| `TaskProgress` | `TaskProgress.vue` | 任务进度条 — SSE 驱动的进度显示 |

## 约定

- 所有组件使用 `<script setup>` + Composition API
- Props 导出类型接口（定义在 `src/types/index.ts` 或组件内 `defineProps`）
- 不直接引用 Pinia store — 由父视图注入 props
- 不发送 API 请求 — 纯展示组件（`NavSidebar` 除外，仅 emit 事件）

## 反模式

- 不要在组件内直接 import `src/api/*`
- 不要在组件内使用 Vue Router — 由 `App.vue` 处理导航
- 不要在 `ImageResult` 外重复图片预览逻辑 — 复用此组件
