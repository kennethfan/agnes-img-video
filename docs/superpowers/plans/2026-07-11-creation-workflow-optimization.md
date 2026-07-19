# 创作项目工作流体验优化 — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 优化创作项目编辑器的三大体验问题：跨步骤自动传图、作品库选择源图、显式保存到作品库

**Architecture:** 纯前端改动（4 个组件修改 + 1 个新组件），无后端改动 — 已有 saveAsset/getAssets API 全覆盖

**Tech Stack:** Vue 3 · TypeScript 6 · Element Plus · Pinia · Axios

## Global Constraints

- 前端使用 Composition API + `<script setup>`，TypeScript 6 `erasableSyntaxOnly`（无 enum）
- 所有代码注释使用中文
- 组件 props/emits 使用 TypeScript `defineProps`/`defineEmits` 类型声明
- 已有 API 签名请参考 `frontend/src/api/assets.ts`: `saveAsset({ image_url, prompt, mode })` → `Promise<{ id: number }>`、`getAssets({ type: 'image' })` → `AssetListResponse`
- 验证：`pnpm build` 无 error（warning 可忽略）
- 提交使用 semantic commit（`feat:` / `refactor:` / `fix:`）

---

### Task 1: ImageResult 移除自动保存逻辑

**Files:**
- Modify: `frontend/src/components/ImageResult.vue`

**Interfaces:**
- Consumes: `ImageResult` 当前 props: `images`, `loading`, `prompt`, `mode`
- Produces: 清理后的 ImageResult，保留 `handleSaveToGallery` 和 `savingUrls`

- [ ] **Step 1: 删除自动保存相关代码块**

在 `frontend/src/components/ImageResult.vue` 的 `<script setup>` 中：

1. 删除 `watch` 导入（`import { ref, watch } from 'vue'` → `import { ref } from 'vue'`）
2. 删除以下三块代码：

```typescript
// 删除
const autoSaveEnabled = ref(localStorage.getItem('autoSaveToGallery') !== 'false')

// 删除
const autoSavedUrls = ref<Set<string>>(new Set())

// 删除整个 watch
watch(() => props.images, (newImages) => {
  if (!autoSaveEnabled.value) return
  newImages.forEach(img => {
    if (!autoSavedUrls.value.has(img)) {
      autoSavedUrls.value = new Set([...autoSavedUrls.value, img])
      saveAsset({ image_url: img, prompt: props.prompt, mode: props.mode }).catch(() => {
        // 静默失败，不影响用户体验
      })
    }
  })
}, { immediate: true })
```

3. 保留 `savingUrls`、`handleSaveToGallery`、`downloadImage` 等手动操作逻辑

- [ ] **Step 2: 更新模板中 auto-save 条件渲染**

在 `<template>` 中，找到以下片段：

```html
<el-button
  v-if="autoSaveEnabled && autoSavedUrls.has(img)"
  size="small"
  type="info"
  disabled
>
  已保存
</el-button>
<el-button
  v-else
  size="small"
  type="success"
  :loading="savingUrls.has(img)"
  :disabled="savingUrls.has(img)"
  @click="handleSaveToGallery(img)"
>
  保存到作品库
</el-button>
```

替换为：

```html
<el-button
  size="small"
  type="success"
  :loading="savingUrls.has(img)"
  :disabled="savingUrls.has(img)"
  @click="handleSaveToGallery(img)"
>
  {{ savingUrls.has(img) ? '保存中...' : '保存到作品库' }}
</el-button>
```

- [ ] **Step 3: 验证构建**

```bash
cd frontend && pnpm build
```

Expected: `✓ built`，无 TypeScript error。

- [ ] **Step 4: 提交**

```bash
git add frontend/src/components/ImageResult.vue
git commit -m "refactor: remove auto-save logic from ImageResult"
```

---

### Task 2: 创建 AssetPickerDialog 作品库选择器组件

**Files:**
- Create: `frontend/src/components/AssetPickerDialog.vue`

**Interfaces:**
- Produces: `AssetPickerDialog` — props: `visible: boolean`, emit: `update:visible`, `selected(url: string)`
- Consumes: `getAssets({ type: 'image' })` from `src/api/assets.ts`、`AssetItem` from `src/types`

- [ ] **Step 1: 编写组件模板和脚本**

创建 `frontend/src/components/AssetPickerDialog.vue`，完整内容如下：

```vue
<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { getAssets } from '../api/assets'
import type { AssetItem } from '../types'

const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{
  'update:visible': [value: boolean]
  selected: [url: string]
}>()

const loading = ref(false)
const assets = ref<AssetItem[]>([])
const selectedUrl = ref('')

async function loadAssets() {
  loading.value = true
  try {
    const res = await getAssets({ type: 'image' })
    assets.value = res.items || []
  } catch (e: any) {
    ElMessage.error('加载作品库失败: ' + (e.message || ''))
  } finally {
    loading.value = false
  }
}

watch(() => props.visible, (v) => {
  if (v) {
    selectedUrl.value = ''
    loadAssets()
  }
})

function toggleSelect(url: string) {
  selectedUrl.value = selectedUrl.value === url ? '' : url
}

function confirm() {
  if (!selectedUrl.value) {
    ElMessage.warning('请选择一张图片')
    return
  }
  emit('selected', selectedUrl.value)
  emit('update:visible', false)
}

function cancel() {
  emit('update:visible', false)
}
</script>

<template>
  <el-dialog
    :model-value="visible"
    @update:model-value="emit('update:visible', $event)"
    title="从作品库选择图片"
    width="700px"
    top="5vh"
    :close-on-click-modal="false"
  >
    <div v-loading="loading" style="min-height: 200px">
      <div v-if="assets.length === 0 && !loading" style="text-align: center; padding: 60px 0; color: #c0c4cc">
        <p>作品库暂无图片</p>
      </div>
      <el-row :gutter="12" v-else>
        <el-col
          v-for="item in assets"
          :key="item.id"
          :xs="12" :sm="8" :md="6"
          style="margin-bottom: 12px"
        >
          <div
            class="asset-thumb"
            :class="{ selected: selectedUrl === item.original_url }"
            @click="toggleSelect(item.original_url)"
          >
            <el-image
              :src="item.thumbnail || item.original_url"
              fit="cover"
              style="width: 100%; height: 120px"
            />
            <div class="asset-label">{{ item.prompt?.slice(0, 30) }}</div>
          </div>
        </el-col>
      </el-row>
    </div>
    <template #footer>
      <el-button @click="cancel">取消</el-button>
      <el-button type="primary" @click="confirm">确认</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.asset-thumb {
  border: 2px solid #dcdfe6;
  border-radius: 6px;
  overflow: hidden;
  cursor: pointer;
  transition: border-color 0.2s;
}
.asset-thumb:hover {
  border-color: #409eff;
}
.asset-thumb.selected {
  border-color: #409eff;
  box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.3);
}
.asset-label {
  padding: 4px 6px;
  font-size: 12px;
  color: #606266;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
```

- [ ] **Step 2: 验证构建**

```bash
cd frontend && pnpm build
```

Expected: `✓ built`

- [ ] **Step 3: 提交**

```bash
git add frontend/src/components/AssetPickerDialog.vue
git commit -m "feat: add AssetPickerDialog component for picking gallery images"
```

---

### Task 3: 跨步骤自动传图 — GenStep emit → ProjectEditor bridge → RefineStep prop

**Files:**
- Modify: `frontend/src/components/GenStep.vue`
- Modify: `frontend/src/views/ProjectEditor.vue`
- Modify: `frontend/src/components/RefineStep.vue`

**Interfaces:**
- Consumes: GenStep 现有 props `project: Project | null`
- Produces: GenStep emit `generated(urls: string[])`、RefineStep prop `defaultImageUrl: string`
- Bridges: ProjectEditor 监听 `@generated`，存储 `latestGenUrls`，导航至 refine 时传 `latestGenUrls[0]`

- [ ] **Step 1: GenStep 添加 `generated` emit**

在 `frontend/src/components/GenStep.vue` 的 `<script setup>` 中：

在 `defineProps` 之后添加 emit 声明：

```typescript
const emit = defineEmits<{
  generated: [urls: string[]]
}>()
```

在 `generate()` 函数的 `resultUrls.value = resp.images` 之后（即 `try` 块末尾，`catch` 之前）添加：

```typescript
    // 生成完成后通知父组件
    if (resultUrls.value.length > 0) {
      emit('generated', resultUrls.value)
    }
```

完整后的 `generate()` 函数（关键部分）：

```typescript
async function generate() {
  if (!prompt.value.trim()) {
    ElMessage.warning('请输入提示词')
    return
  }
  generating.value = true
  resultUrls.value = []
  try {
    if (mode.value === 'text2image') {
      const resp = await textToImage({ prompt: prompt.value, size: size.value, n: 1 })
      if (resp.images?.length) {
        resultUrls.value = resp.images
      }
    } else {
      if (!imageUrl.value) {
        ElMessage.warning('请输入参考图片 URL')
        return
      }
      const resp = await imageToImage(imageUrl.value, prompt.value, size.value, strength.value)
      if (resp.images?.length) {
        resultUrls.value = resp.images
      }
    }
    if (resultUrls.value.length > 0) {
      emit('generated', resultUrls.value)
    }
  } catch (e: any) {
    ElMessage.error('生成失败: ' + (e.message || ''))
  } finally {
    generating.value = false
  }
}
```

- [ ] **Step 2: ProjectEditor 添加 latestGenUrls 跨步骤传图**

在 `frontend/src/views/ProjectEditor.vue` 的 `<script setup>` 中：

在 `aiLoading` 之后添加：

```typescript
const latestGenUrls = ref<string[]>([])
```

在 `<template>` 的 GenStep 调用处（`currentStep === 'generate'`）添加 `@generated` 监听和 `latestGenUrls` 传参：

```html
<div v-if="currentStep === 'generate'" class="step-body">
  <GenStep :project="project" @generated="latestGenUrls = $event" />
</div>
```

在 RefineStep 调用处（`currentStep === 'refine'`）添加 `:defaultImageUrl`：

```html
<div v-if="currentStep === 'refine'" class="step-body">
  <RefineStep :project="project" :defaultImageUrl="latestGenUrls[0] || ''" />
</div>
```

- [ ] **Step 3: RefineStep 添加 `defaultImageUrl` prop 及 watch 填充**

在 `frontend/src/components/RefineStep.vue` 的 `<script setup>` 中：

修改 `defineProps`：

```typescript
const props = defineProps<{
  project: Project | null
  defaultImageUrl?: string
}>()
```

在 `resultUrls` 定义之后添加 watch：

```typescript
import { ref, watch } from 'vue'  // 确保 watch 被导入

// 在 sourceImage 定义之后
watch(() => props.defaultImageUrl, (url) => {
  if (url) {
    sourceImage.value = url
  }
})
```

- [ ] **Step 4: 验证构建**

```bash
cd frontend && pnpm build
```

Expected: `✓ built`

- [ ] **Step 5: 提交**

```bash
git add frontend/src/components/GenStep.vue frontend/src/views/ProjectEditor.vue frontend/src/components/RefineStep.vue
git commit -m "feat: add cross-step image transfer from generate to refine"
```

---

### Task 4: 集成 AssetPickerDialog + 显式保存按钮到 GenStep、RefineStep、FinalStep

**Files:**
- Modify: `frontend/src/components/GenStep.vue`
- Modify: `frontend/src/components/RefineStep.vue`
- Modify: `frontend/src/components/FinalStep.vue`

**Interfaces:**
- Consumes: `AssetPickerDialog`（Task 2）、`saveAsset` API、`ImageResult`（已清理自动保存的 Task 1 版本）

- [ ] **Step 1: GenStep 添加「从作品库」按钮和 AssetPickerDialog**

在 `frontend/src/components/GenStep.vue` 的 `<script setup>` 中添加：

```typescript
import { Plus } from '@element-plus/icons-vue'
import AssetPickerDialog from './AssetPickerDialog.vue'

const showAssetPicker = ref(false)

function onAssetSelected(url: string) {
  imageUrl.value = url
}
```

在 `<template>` 中，找到图生图的 imageUrl 输入框行：

```html
<div v-if="mode === 'image2image'" class="form-row">
  <el-input v-model="imageUrl" placeholder="参考图片 URL" />
  <el-button @click="showAssetPicker = true" :icon="Picture">从作品库</el-button>
  <!-- 其余 size/slider 内容不变 -->
</div>
```

在模板末尾（`</div>` 之前，`<style>` 之前），添加 AssetPickerDialog：

```html
<AssetPickerDialog
  v-model:visible="showAssetPicker"
  @selected="onAssetSelected"
/>
```

- [ ] **Step 2: RefineStep 添加「从作品库」按钮和 AssetPickerDialog**

在 `frontend/src/components/RefineStep.vue` 的 `<script setup>` 中添加：

```typescript
import { Picture } from '@element-plus/icons-vue'
import AssetPickerDialog from './AssetPickerDialog.vue'

const showAssetPicker = ref(false)

function onAssetSelected(url: string) {
  sourceImage.value = url
}
```

在 `<template>` 中，找到 sourceImage 输入框：

```html
<div class="refine-form">
  <div style="display: flex; gap: 8px; align-items: center">
    <el-input
      v-model="sourceImage"
      placeholder="源图片 URL（从生成结果复制或输入）"
      style="flex: 1"
    />
    <el-button @click="showAssetPicker = true" :icon="Picture">从作品库</el-button>
  </div>
  <!-- 其余内容不变 -->
</div>
```

在模板末尾添加：

```html
<AssetPickerDialog
  v-model:visible="showAssetPicker"
  @selected="onAssetSelected"
/>
```

- [ ] **Step 3: FinalStep 添加保存到作品库按钮**

在 `frontend/src/components/FinalStep.vue` 的 `<script setup>` 中添加：

```typescript
import { Plus } from '@element-plus/icons-vue'
import { saveAsset } from '../api/assets'

const savingSteps = ref<Set<number>>(new Set())

async function saveStepOutput(stepId: number, imageUrl: string) {
  savingSteps.value = new Set([...savingSteps.value, stepId])
  try {
    await saveAsset({ image_url: imageUrl, prompt: '来自创作项目', mode: 'image' })
    ElMessage.success('已保存到作品库')
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  } finally {
    const next = new Set(savingSteps.value)
    next.delete(stepId)
    savingSteps.value = next
  }
}
```

在 `<template>` 中，找到生成记录的每个 `step.output` 展示处（第 85 行附近），在 `ImageResult` 下方添加保存按钮：

```html
<div v-if="step.output" class="record-output">
  <ImageResult :images="[step.output]" :loading="false" prompt="" mode="" />
  <div style="margin-top: 8px">
    <el-button
      size="small"
      type="success"
      :icon="Plus"
      :loading="savingSteps.has(step.id)"
      :disabled="savingSteps.has(step.id)"
      @click="saveStepOutput(step.id, step.output)"
    >
      {{ savingSteps.has(step.id) ? '保存中...' : '保存到作品库' }}
    </el-button>
  </div>
</div>
```

同样，在优化记录（第 95 行附近）的每个 `step.output` 下方添加相同的按钮（代码相同）。

- [ ] **Step 4: 验证构建**

```bash
cd frontend && pnpm build
```

Expected: `✓ built`

- [ ] **Step 5: 提交**

```bash
git add frontend/src/components/GenStep.vue frontend/src/components/RefineStep.vue frontend/src/components/FinalStep.vue
git commit -m "feat: integrate asset picker and save buttons into workflow steps"
```

---

## 验证清单

所有 Task 完成后，运行：

```bash
cd frontend && pnpm build
```

Expected: `✓ built`，无 TypeScript error。

手动测试：
1. 进入创作项目编辑器 → 生成步骤 → 生成图片 → 确认图片下方有「保存到作品库」按钮
2. 点击保存 → 确认作品库中有新记录
3. 切换图生图模式 → 确认「从作品库」按钮存在 → 点击弹窗选择图片
4. 从生成步骤点击下一步 → 确认优化步骤自动填充源图 URL
5. 进入定稿步骤 → 确认生成/优化记录下方有保存按钮
