# UI Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refresh the frontend SPA with a modern sidebar navigation, whitespace-heavy visual style, and dual-column page layouts.

**Architecture:** Three independent work streams — (1) replace `el-tabs` with a new `NavSidebar` component and restructure App.vue, (2) apply new visual tokens (colors/borders/radius) to all existing components, (3) refactor generation pages to left-input/right-preview dual columns.

**Tech Stack:** Vue 3 + TypeScript 6 + Element Plus + Vite 8

**Spec:** `docs/superpowers/specs/2026-07-07-ui-redesign-design.md`

## Global Constraints

- TypeScript 6 `erasableSyntaxOnly` — no enums or namespaces, use `as const` or union types
- No vue-router — all navigation via reactive state (`activePage` ref) + conditional rendering
- No backend API changes
- No new features or pages
- Element Plus components used throughout
- Scoped styles preferred over global CSS overrides
- All modified views remain functional — generate button must still call the API

---

### Task 1: NavSidebar component + App.vue navigation restructure

**Files:**
- Create: `frontend/src/components/NavSidebar.vue`
- Modify: `frontend/src/App.vue`
- Verify: `frontend/src/views/*.vue` (ensure they work without el-tabs wrapper)

**Interfaces:**
- Produces: `NavSidebar` component with `v-model:active-page` prop. Active page values: `text2img | img2img | batch | script_gen | text2vid | img2vid | multi_vid | ideas | storyboard | assets | history`.
- Produces: `App.vue` renders NavSidebar + top header + conditional page view (no el-tabs).

- [ ] **Step 1: Create `frontend/src/components/NavSidebar.vue`**

Left icon bar (52px wide) + flyout group menu on click. Icons for 4 groups:

```vue
<script setup lang="ts">
import { ref, computed } from 'vue'

const activeGroup = ref('image')
const activePage = defineModel<string>('activePage', { required: true })

const groups = [
  {
    id: 'image',
    icon: '🖼',
    label: '图片',
    items: [
      { id: 'text2img', label: '文生图' },
      { id: 'img2img', label: '图生图' },
      { id: 'batch', label: '批量生成' },
    ],
  },
  {
    id: 'video',
    icon: '🎬',
    label: '视频',
    items: [
      { id: 'text2vid', label: '文生视频' },
      { id: 'img2vid', label: '图生视频' },
      { id: 'multi_vid', label: '多图视频' },
    ],
  },
  {
    id: 'tools',
    icon: '📝',
    label: '工具',
    items: [
      { id: 'script_gen', label: '脚本生成' },
      { id: 'ideas', label: '点子库' },
      { id: 'storyboard', label: '分镜' },
    ],
  },
  {
    id: 'works',
    icon: '🖥',
    label: '作品',
    items: [
      { id: 'assets', label: '作品库' },
      { id: 'history', label: '历史记录' },
    ],
  },
]

const isOpen = ref(false)

function toggleGroup(groupId: string) {
  if (activeGroup.value === groupId && isOpen.value) {
    isOpen.value = false
  } else {
    activeGroup.value = groupId
    isOpen.value = true
  }
}

function selectPage(pageId: string) {
  activePage.value = pageId
  isOpen.value = false
}

function closeFlyout() {
  isOpen.value = false
}

const currentGroup = computed(() => groups.find(g => g.id === activeGroup.value))
</script>

<template>
  <div class="nav-sidebar" @mouseleave="closeFlyout">
    <!-- Icon bar -->
    <div class="icon-bar">
      <button
        v-for="g in groups"
        :key="g.id"
        class="icon-btn"
        :class="{ active: activeGroup === g.id && isOpen }"
        @click="toggleGroup(g.id)"
        :title="g.label"
      >
        {{ g.icon }}
      </button>
    </div>

    <!-- Flyout panel -->
    <Transition name="flyout">
      <div v-if="isOpen && currentGroup" class="flyout" @mouseenter="isOpen = true">
        <div class="flyout-header">{{ currentGroup.label }}</div>
        <button
          v-for="item in currentGroup.items"
          :key="item.id"
          class="flyout-item"
          :class="{ active: activePage === item.id }"
          @click="selectPage(item.id)"
        >
          {{ item.label }}
        </button>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.nav-sidebar {
  position: relative;
  display: flex;
  align-items: flex-start;
}
.icon-bar {
  width: 52px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 12px 0;
  background: #ffffff;
  border-right: 1px solid #f0f0f0;
}
.icon-btn {
  width: 36px;
  height: 36px;
  border: none;
  border-radius: 8px;
  background: #f5f5f5;
  cursor: pointer;
  font-size: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s;
}
.icon-btn.active {
  background: #000;
}
.flyout {
  position: absolute;
  left: 56px;
  top: 0;
  width: 160px;
  background: #ffffff;
  border: 1px solid #eaeaea;
  border-radius: 12px;
  padding: 8px;
  box-shadow: 0 4px 20px rgba(0,0,0,0.06);
  z-index: 100;
}
.flyout-header {
  font-size: 11px;
  font-weight: 600;
  color: #909399;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  padding: 6px 10px 8px;
}
.flyout-item {
  display: block;
  width: 100%;
  padding: 8px 10px;
  border: none;
  border-radius: 8px;
  background: transparent;
  cursor: pointer;
  font-size: 13px;
  color: #000;
  text-align: left;
  transition: background 0.15s;
}
.flyout-item:hover {
  background: #f5f5f5;
}
.flyout-item.active {
  background: #f5f5f5;
  font-weight: 500;
}
.flyout-enter-active,
.flyout-leave-active {
  transition: opacity 0.15s, transform 0.15s;
}
.flyout-enter-from,
.flyout-leave-to {
  opacity: 0;
  transform: translateX(-6px);
}
</style>
```

- [ ] **Step 2: Modify `frontend/src/App.vue`**

Remove `el-tabs` wrapper, add NavSidebar + top header + conditional page rendering:

```vue
<script setup lang="ts">
import { ref } from 'vue'
import NavSidebar from './components/NavSidebar.vue'
import TextToImage from './views/TextToImage.vue'
import ImageToImage from './views/ImageToImage.vue'
import BatchGen from './views/BatchGen.vue'
import ScriptGen from './views/ScriptGen.vue'
import TextToVideo from './views/TextToVideo.vue'
import ImageToVideo from './views/ImageToVideo.vue'
import MultiImageVideo from './views/MultiImageVideo.vue'
import History from './views/History.vue'
import Ideas from './views/Ideas.vue'
import Assets from './views/Assets.vue'
import Storyboard from './views/Storyboard.vue'
import { onMounted, onUnmounted } from 'vue'
import { useRedoStore } from './stores/redo'

const activePage = ref('text2img')
const redoStore = useRedoStore()

function handleRedoTrigger() {
  const tab = redoStore.targetTab
  if (tab) {
    activePage.value = tab
  }
}

onMounted(() => {
  window.addEventListener('redo-trigger', handleRedoTrigger)
})

onUnmounted(() => {
  window.removeEventListener('redo-trigger', handleRedoTrigger)
})
</script>

<template>
  <div class="app-layout">
    <!-- Top header bar -->
    <header class="top-bar">
      <span class="app-title">Agnes Creator Studio</span>
      <span class="app-subtitle">AI Image & Video Studio</span>
    </header>

    <div class="app-body">
      <NavSidebar v-model:active-page="activePage" />

      <main class="main-content">
        <TextToImage v-if="activePage === 'text2img'" />
        <ImageToImage v-else-if="activePage === 'img2img'" />
        <BatchGen v-else-if="activePage === 'batch'" />
        <ScriptGen v-else-if="activePage === 'script_gen'" />
        <TextToVideo v-else-if="activePage === 'text2vid'" />
        <ImageToVideo v-else-if="activePage === 'img2vid'" />
        <MultiImageVideo v-else-if="activePage === 'multi_vid'" />
        <Ideas v-else-if="activePage === 'ideas'" />
        <Storyboard v-else-if="activePage === 'storyboard'" />
        <Assets v-else-if="activePage === 'assets'" />
        <History v-else-if="activePage === 'history'" />
      </main>
    </div>
  </div>
</template>

<style>
/* Global reset */
* { box-sizing: border-box; }
body {
  margin: 0;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif;
  background: #ffffff;
  color: #000;
}
</style>

<style scoped>
.app-layout {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}
.top-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 20px;
  border-bottom: 1px solid #f0f0f0;
  background: #ffffff;
}
.app-title {
  font-weight: 600;
  font-size: 15px;
  color: #000;
}
.app-subtitle {
  font-size: 12px;
  color: #909399;
}
.app-body {
  display: flex;
  flex: 1;
}
.main-content {
  flex: 1;
  padding: 24px;
  max-width: 1200px;
  overflow-y: auto;
}
</style>
```

Note: Move `onMounted`/`onUnmounted` imports into the `<script>` block (not `<script setup>`) because the setup block defines the handler function. Or keep both — Vue 3 allows dual script blocks. Use dual script blocks: `<script setup lang="ts">` for component logic + `<script lang="ts">` for imports that conflict with setup context. Actually, the simpler approach: just add `import { onMounted, onUnmounted } from 'vue'` at the top of `<script setup>` in the correct order.

Fix: combine into a single `<script setup>`:

```vue
<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
// ... rest of imports and logic
</script>
```

- [ ] **Step 3: Run typecheck**

```bash
cd frontend && npx vue-tsc --noEmit
```

Expected: No type errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/NavSidebar.vue frontend/src/App.vue
git commit -m "feat: replace el-tabs with NavSidebar sidebar navigation"
```

---

### Task 2: Global visual tokens + card components polish

**Files:**
- Modify: `frontend/src/App.vue` (add global CSS variables in `<style>` block)
- Modify: `frontend/src/components/ShotCard.vue`
- Modify: `frontend/src/components/AssetCard.vue`
- Modify: `frontend/src/components/ImageResult.vue`

**Interfaces:**
- Consumes: App.vue's new structure from Task 1
- Produces: Consistent card/button styling across all components

- [ ] **Step 1: Add global CSS variables in `frontend/src/App.vue`**

Add to the unscoped `<style>` block in App.vue:

```css
/* Global design tokens */
:root {
  --bg-page: #ffffff;
  --bg-subtle: #fafafa;
  --bg-card: #ffffff;
  --border-default: #eaeaea;
  --border-light: #f0f0f0;
  --text-primary: #000000;
  --text-secondary: #666666;
  --text-muted: #909399;
  --accent: #000000;
  --accent-hover: #333333;
  --radius-card: 12px;
  --radius-sm: 8px;
  --shadow-card: none;
}
```

- [ ] **Step 2: Update `frontend/src/components/ShotCard.vue` style**

Replace the `<style scoped>` block with new card styles:

```css
.shot-card {
  border: 1px solid var(--border-default);
  border-radius: var(--radius-card);
  padding: 16px;
  margin-bottom: 12px;
  background: var(--bg-card);
  transition: border-color 0.2s;
}
.shot-card:hover {
  border-color: #d0d0d0;
}
.shot-card.is-generating {
  border-color: #409eff;
  background: #f0f7ff;
}
.shot-card.is-completed {
  border-color: #e1f3d8;
}
```

- [ ] **Step 3: Update `frontend/src/components/AssetCard.vue` style**

Read the current file first, then update borders/radius to the new palette. Replace `box-shadow` with `border: 1px solid var(--border-default)` and `border-radius: var(--radius-card)`.

- [ ] **Step 4: Update `frontend/src/components/ImageResult.vue` style**

Read the current file, update image container styling to use new card tokens. Images should have `border-radius: var(--radius-sm)` and the container should use `--border-light`.

- [ ] **Step 5: Run typecheck + build**

```bash
cd frontend && npx vue-tsc --noEmit && pnpm build
```

Expected: Build succeeds.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/App.vue frontend/src/components/ShotCard.vue frontend/src/components/AssetCard.vue frontend/src/components/ImageResult.vue
git commit -m "style: apply new visual tokens and card styles"
```

---

### Task 3: Dual-column layout for generation pages

**Files:**
- Modify: `frontend/src/views/TextToImage.vue`
- Modify: `frontend/src/views/ImageToImage.vue`
- Modify: `frontend/src/views/BatchGen.vue`
- Modify: `frontend/src/views/TextToVideo.vue`
- Modify: `frontend/src/views/ImageToVideo.vue`
- Modify: `frontend/src/views/MultiImageVideo.vue`

**Interfaces:**
- Consumes: App.vue new layout + global CSS tokens from Tasks 1-2
- Produces: All 6 generation pages in left-input/right-preview dual-column layout

Each page should follow this structure:

```vue
<template>
  <div class="gen-page">
    <div class="gen-input">
      <!-- existing form content, wrapped in card -->
    </div>
    <div class="gen-preview">
      <!-- existing result/preview area -->
    </div>
  </div>
</template>

<style scoped>
.gen-page {
  display: flex;
  gap: 24px;
  min-height: 500px;
}
.gen-input {
  flex: 1;
  max-width: 480px;
  padding: 20px;
  background: var(--bg-card);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-card);
}
.gen-preview {
  flex: 1;
  padding: 20px;
  background: var(--bg-subtle);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-card);
  display: flex;
  flex-direction: column;
}
</style>
```

- [ ] **Step 1: Refactor `frontend/src/views/TextToImage.vue`**

Read current file, restructure template to dual-column. Keep the form controls in `gen-input`, move `ImageResult` into `gen-preview`. The `<script>` block stays the same.

- [ ] **Step 2: Refactor `frontend/src/views/ImageToImage.vue`**

Same pattern — form controls in left column, result in right column.

- [ ] **Step 3: Refactor `frontend/src/views/BatchGen.vue`**

Batch prompts in left column, results grid in right column.

- [ ] **Step 4: Refactor `frontend/src/views/TextToVideo.vue`**

Video form in left column, result video player in right column. Keep SSE progress indicator in the preview area.

- [ ] **Step 5: Refactor `frontend/src/views/ImageToVideo.vue`**

Same pattern as TextToVideo.

- [ ] **Step 6: Refactor `frontend/src/views/MultiImageVideo.vue`**

Multi-image upload in left column, video result in right column.

- [ ] **Step 7: Run typecheck + build**

```bash
cd frontend && npx vue-tsc --noEmit && pnpm build
```

Expected: Build succeeds. All pages render correctly.

- [ ] **Step 8: Commit**

```bash
git add frontend/src/views/TextToImage.vue frontend/src/views/ImageToImage.vue frontend/src/views/BatchGen.vue frontend/src/views/TextToVideo.vue frontend/src/views/ImageToVideo.vue frontend/src/views/MultiImageVideo.vue
git commit -m "feat: dual-column layout for generation pages"
```

---

### Task 4: Integration verification

**Files:**
- All modified files — verify nothing is broken

- [ ] **Step 1: Start backend and do quick smoke test**

```bash
cd backend && AGNES_API_KEY=test go run ./cmd/server &
sleep 3
curl -s "http://localhost:8080/api/v1/config" | head -c 200
```

Expected: Server starts, responds to requests.

- [ ] **Step 2: Verify full frontend build**

```bash
cd frontend && npx vue-tsc --noEmit && pnpm build
```

Expected: Clean build with no errors.

- [ ] **Step 3: Visual check — open dev server**

```bash
cd frontend && pnpm dev
```

Open http://localhost:5173 in browser, verify:
- Left icon bar renders with 4 groups
- Clicking each icon opens the flyout menu
- Selecting a page in the flyout switches the view
- Page shows with dual-column layout
- Cards have new border/radius styling

- [ ] **Step 4: Commit final polish**

```bash
git add -A
git commit -m "feat: complete UI redesign with sidebar navigation and dual-column layout"
```
