# Vue Router 接入实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为所有页面赋予独立的 hash URL（`/#/text2img`），支持浏览器前进/后退

**Architecture:** 惰性迁移策略——新建路由层作为 URL 同步层，保留现有 `activePage` + `v-if` 渲染链不变。路由变化时单向同步到 `activePage`，不引入 `<router-view>`。

**Tech Stack:** Vue 3 + vue-router（已安装） + TypeScript

## Global Constraints

- Hash 模式（`createWebHashHistory()`）—— 不需要后端 fallback 配置
- 所有组件使用懒加载 `() => import(...)`
- 不引入 `<router-view>` —— `v-if` 渲染链继续保持权威
- 不新增 npm 依赖（vue-router 已存在）
- 不做 TDD（纯前端结构改动，无逻辑测试边界）
- 每个任务完成后需要 `pnpm build` 验证编译通过
- 保持 TypeScript 6 `erasableSyntaxOnly` 约束 —— 不使用 enum

---

### Task 1: 创建路由表

**Files:**
- Create: `frontend/src/router/index.ts`

**Interfaces:**
- Produces: 默认导出 `Router` 实例

- [ ] **Step 1: 创建 `frontend/src/router/index.ts`**

```ts
import { createRouter, createWebHashHistory } from 'vue-router'

const routes = [
  { path: '/', redirect: '/text2img' },
  { path: '/text2img',    name: 'text2img',    component: () => import('../views/TextToImage.vue') },
  { path: '/img2img',     name: 'img2img',     component: () => import('../views/ImageToImage.vue') },
  { path: '/batch',       name: 'batch',       component: () => import('../views/BatchGen.vue') },
  { path: '/script-gen',  name: 'script_gen',  component: () => import('../views/ScriptGen.vue') },
  { path: '/text2vid',    name: 'text2vid',    component: () => import('../views/TextToVideo.vue') },
  { path: '/img2vid',     name: 'img2vid',     component: () => import('../views/ImageToVideo.vue') },
  { path: '/multi-vid',   name: 'multi_vid',   component: () => import('../views/MultiImageVideo.vue') },
  { path: '/ideas',       name: 'ideas',       component: () => import('../views/Ideas.vue') },
  { path: '/storyboard',  name: 'storyboard',  component: () => import('../views/Storyboard.vue') },
  { path: '/assets',      name: 'assets',      component: () => import('../views/Assets.vue') },
  { path: '/tasks',       name: 'tasks',       component: () => import('../views/TaskRecords.vue') },
  { path: '/history',     name: 'history',     component: () => import('../views/History.vue') },
  { path: '/access-logs', name: 'access_logs', component: () => import('../views/AccessLogs.vue') },
  { path: '/db-manage',   name: 'db_manage',   component: () => import('../views/DBManage.vue') },
  { path: '/wizard/:type', name: 'wizard',     component: () => import('../views/WorkflowWizard.vue') },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

export default router
```

- [ ] **Step 2: 验证文件存在且无语法错误**

Run: `npx vue-tsc --noEmit frontend/src/router/index.ts 2>&1 || echo "check manually"`

Expected: 无类型错误（vue-tsc 可能因缺少入口文件报错，属正常）

### Task 2: main.ts 注册 Router

**Files:**
- Modify: `frontend/src/main.ts`

**Interfaces:**
- Consumes: Task 1 的 `router` 默认导出

- [ ] **Step 1: 读取 `frontend/src/main.ts`**

- [ ] **Step 2: 添加 router import 和 app.use**

```ts
import { createApp } from 'vue'
import App from './App.vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import router from './router'    // 新增

const app = createApp(App)
app.use(ElementPlus)
app.use(router)                  // 新增
app.mount('#app')
```

- [ ] **Step 3: 验证编译通过**

Run: `cd frontend && pnpm build`

Expected: 构建成功

### Task 3: App.vue 路由整合

**Files:**
- Modify: `frontend/src/App.vue`

**Interfaces:**
- Consumes: Task 1 的 router 实例（通过 `useRoute`/`useRouter`）
- Produces: `navigateTo(page: string)` 函数，供 NavSidebar 和 redo store 调用

- [ ] **Step 1: 读取 `App.vue` 当前内容，确认 `activePage`、`redo-trigger`、view 列表**

- [ ] **Step 2: 添加路由监听和导航函数**

在 `<script setup>` 中添加：

```ts
import { watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const activePage = ref('text2img')

// 路由变化 → 同步 activePage（浏览器前进/后退）
watch(() => route.name, (name) => {
  if (name && typeof name === 'string') {
    activePage.value = name
  }
})

// 页面切换：同步 activePage + URL
function navigateTo(page: string) {
  activePage.value = page
  router.push({ name: page })
}
```

将 `onMounted` 和 `onUnmounted` 中的 `redo-trigger` 事件监听改为路由方式：

```ts
import { onMounted, onUnmounted, watch, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const activePage = ref('text2img')

// 从 URL 恢复页面（刷新/直接访问）
watch(() => route.name, (name) => {
  if (name && typeof name === 'string') {
    activePage.value = name
  }
}, { immediate: true })
```

注意添加 `{ immediate: true }` 使页面刷新时从 URL 恢复。

移除 `onMounted`/`onUnmounted` 中的 `redo-trigger` 事件监听，替代为通过 `navigateTo` 响应。

在模板中，将 `NavSidebar` 的绑定从 `v-model:active-page` 改为 `:active-page` + `@navigate`：

```vue
<NavSidebar :active-page="activePage" @navigate="navigateTo" />
```

- [ ] **Step 3: 验证编译通过**

Run: `cd frontend && pnpm build`

Expected: 构建成功，无 TS 错误

### Task 4: NavSidebar 改造

**Files:**
- Modify: `frontend/src/components/NavSidebar.vue`

**Interfaces:**
- Consumes: `activePage: string` prop、`navigate` emit
- Produces: `@navigate` emit 事件，参数为页面 ID

- [ ] **Step 1: 读取当前 `NavSidebar.vue` 内容**

- [ ] **Step 2: 修改 props 定义**

将 `defineModel<string>('activePage', ...)` 替换为：

```ts
const props = defineProps<{ activePage: string }>()
const emit = defineEmits<{ navigate: [pageId: string] }>()
```

- [ ] **Step 3: 更新 `selectPage` 函数**

```ts
function selectPage(pageId: string) {
  emit('navigate', pageId)
}
```

模板中 `:class="{ active: activePage === item.id }"` 保持不动（`activePage` 现在是 prop 而不是 model），使用方式完全一样。

- [ ] **Step 4: 验证编译通过**

Run: `cd frontend && pnpm build`

Expected: 构建成功

### Task 5: Redo Store 路由集成

**Files:**
- Modify: `frontend/src/stores/redo.ts`

**Interfaces:**
- Consumes: router 实例（通过 `useRouter`）
- Depends on: Task 1 的 router 导出

- [ ] **Step 1: 读取 `redo.ts` 当前内容**

- [ ] **Step 2: 在 `setRedoData` 中将 dispatchEvent 替换为 router.push**

```ts
import { defineStore } from 'pinia'
import { useRouter } from 'vue-router'

export const useRedoStore = defineStore('redo', () => {
  const targetTab = ref<string | null>(null)

  const modeToTab: Record<string, string> = {
    text2image: 'text2img',
    image2image: 'img2img',
    batch: 'batch',
    script_gen: 'script_gen',
    text2video: 'text2vid',
    image2video: 'img2vid',
    multi_image_video: 'multi_vid',
    // wizard 类型
    image_refine: 'wizard',
    comic: 'wizard',
    novel: 'wizard',
  }

  function setRedoData(data: Record<string, any>) {
    const tab = modeToTab[data.mode] || 'text2img'
    targetTab.value = tab

    // 保存 redo 数据
    sessionStorage.setItem('redoData', JSON.stringify(data))

    // 路由跳转
    const router = useRouter()
    if (data.mode === 'image_refine' || data.mode === 'comic' || data.mode === 'novel') {
      router.push({ name: 'wizard', params: { type: data.mode } })
    } else {
      router.push({ name: tab })
    }
  }

  function consumeRedoData(): Record<string, any> | null {
    const raw = sessionStorage.getItem('redoData')
    if (!raw) return null
    sessionStorage.removeItem('redoData')
    return JSON.parse(raw)
  }

  return { targetTab, modeToTab, setRedoData, consumeRedoData }
})
```

注意：`useRouter()` 必须在 Pinia store 内部调用（或通过函数参数传入），因为 Pinia store 初始化时 router 可能尚未注册。

- [ ] **Step 3: 验证编译通过**

Run: `cd frontend && pnpm build`

Expected: 构建成功

### Task 6: WorkflowWizard 路由参数适配

**Files:**
- Modify: `frontend/src/views/WorkflowWizard.vue`（如果当前使用 props 接收 `workflowType`）

- [ ] **Step 1: 读取 `WorkflowWizard.vue` 当前内容**

- [ ] **Step 2: 将 `workflowType` prop 改为从路由参数读取**

```ts
import { useRoute } from 'vue-router'

const route = useRoute()

// 从 URL `/wizard/:type` 获取 workflowType
const workflowType = computed(() => route.params.type as string)
```

如果组件仍有其他 props（如图片编辑相关），保留它们不变。

- [ ] **Step 3: 清理 `App.vue` 中传入 workdlowType 的写法**

当前 `App.vue` 模板：
```vue
<WorkflowWizard v-else-if="activePage === 'image_refine'" workflowType="image_refine" />
<WorkflowWizard v-else-if="activePage === 'comic'" workflowType="comic" />
<WorkflowWizard v-else-if="activePage === 'novel'" workflowType="novel" />
```

保持不动（`workflowType` prop 仍可通过组件 props 传入），或者如果组件已改为从 `route.params` 读取，则移除 prop 绑定。

- [ ] **Step 4: 验证编译通过**

Run: `cd frontend && pnpm build`

Expected: 构建成功，所有页面可访问
