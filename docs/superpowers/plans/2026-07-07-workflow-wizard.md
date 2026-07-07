# Workflow Wizard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a step-by-step workflow wizard covering 3 creative flows (image refine, comic generation, novel generation) with shared wizard infrastructure.

**Architecture:** All frontend — no new backend APIs. Each workflow is a sequence of steps managed by a Pinia store. WorkflowWizard.vue is the container; each step is a separate component in a subdirectory.

**Tech Stack:** Vue 3 + TypeScript 6 + Element Plus + Pinia · Existing Image/Video/Ideas API

## Global Constraints

- `erasableSyntaxOnly` enabled — no enums or namespaces, use `as const` or union types
- No vue-router — use `el-tabs` / conditional rendering in App.vue
- Follow existing patterns in ImageToImage.vue, BatchGen.vue for image API calls
- No new backend endpoints — reuse `POST /images/text-to-image`, `POST /images/image-to-image`, `POST /images/batch`, `POST /ideas/expand`
- Video workflow NOT implemented (spec marked ⚠️ 暂缓)
- All new files in `frontend/src/`

---

### Task 1: Wizard Infrastructure (Pinia store + NavSidebar + App.vue + WorkflowWizard container)

**Files:**
- Create: `frontend/src/stores/wizard.ts`
- Create: `frontend/src/views/WorkflowWizard.vue`
- Modify: `frontend/src/components/NavSidebar.vue` (add "创作" group)
- Modify: `frontend/src/App.vue` (add imports + conditional renders)

**Interfaces:**
- Produces: `useWizardStore()` — Pinia store managing current workflow type, active step, and per-workflow data
- Produces: `WorkflowWizard.vue` — container that renders active step component based on store state

- [ ] **Step 1: Create `frontend/src/stores/wizard.ts`**

```typescript
import { defineStore } from 'pinia'

export interface NovelData {
  theme: string
  genre: string
  characters: { name: string; personality: string; appearance: string }[]
  outline: string
  chapters: { title: string; content: string; illustration?: string }[]
}

export interface ImageRefineData {
  sourceType: 'generate' | 'upload'
  sourcePrompt: string
  sourceImage: string // data URL or uploaded URL
  sourceFile?: File
  refinePrompt: string
  strength: number
  size: string
  resultImage: string
}

export interface ComicData {
  theme: string
  layout: 'single' | 'dual' | 'quad' | 'six'
  panels: { prompt: string; image: string; caption: string }[]
}

export type WorkflowType = 'image_refine' | 'comic' | 'novel'

export const useWizardStore = defineStore('wizard', {
  state: () => ({
    workflow: null as WorkflowType | null,
    step: 0,
    totalSteps: 0,
    novel: {
      theme: '',
      genre: '',
      characters: [] as NovelData['characters'],
      outline: '',
      chapters: [] as NovelData['chapters'],
    } as NovelData,
    image: {
      sourceType: 'generate' as ImageRefineData['sourceType'],
      sourcePrompt: '',
      sourceImage: '',
      refinePrompt: '',
      strength: 0.75,
      size: '1024x1024',
      resultImage: '',
    } as ImageRefineData,
    comic: {
      theme: '',
      layout: 'quad' as ComicData['layout'],
      panels: [],
    } as ComicData,
  }),
  getters: {
    stepConfig: (state) => {
      const configs: Record<WorkflowType, string[]> = {
        image_refine: ['选择来源', '精修调参', '对比预览', '导出'],
        comic: ['设定主题', '选择布局', '填写提示词', '批量生成', '添加台词', '导出'],
        novel: ['选题设定', '风格选择', '角色设置', '大纲生成', '确认大纲', '逐章生成', '配插图', '导出'],
      }
      if (!state.workflow) return []
      return configs[state.workflow]
    },
    currentStepLabel: (state) => {
      const labels = state.stepConfig
      return labels[state.step] || ''
    },
    isFirstStep: (state) => state.step === 0,
    isLastStep: (state) => state.step === state.totalSteps - 1,
  },
  actions: {
    startWorkflow(type: WorkflowType) {
      this.workflow = type
      this.step = 0
      this.totalSteps = this.stepConfig.length
    },
    nextStep() {
      if (this.step < this.totalSteps - 1) this.step++
    },
    prevStep() {
      if (this.step > 0) this.step--
    },
    goToStep(n: number) {
      if (n >= 0 && n < this.totalSteps) this.step = n
    },
    reset() {
      this.workflow = null
      this.step = 0
      this.totalSteps = 0
    },
  },
})
```

- [ ] **Step 2: Create `frontend/src/views/WorkflowWizard.vue`**

```vue
<script setup lang="ts">
import { computed } from 'vue'
import { useWizardStore } from '../stores/wizard'
import { ArrowLeft, ArrowRight, Close } from '@element-plus/icons-vue'
import { ElSteps, ElStep, ElButton } from 'element-plus'

// Image refine steps
import StepSource from './image/StepSource.vue'
import StepRefine from './image/StepRefine.vue'
import StepCompare from './image/StepCompare.vue'
import StepExport from './image/StepExport.vue'

// Comic steps
import ComicStepTheme from './comic/StepTheme.vue'
import ComicStepLayout from './comic/StepLayout.vue'
import ComicStepPanels from './comic/StepPanels.vue'
import ComicStepGenerate from './comic/StepGenerate.vue'
import ComicStepCaptions from './comic/StepCaptions.vue'
import ComicStepExport from './comic/StepExport.vue'

// Novel steps
import NovelStepTheme from './novel/StepTheme.vue'
import NovelStepGenre from './novel/StepGenre.vue'
import NovelStepCharacters from './novel/StepCharacters.vue'
import NovelStepOutline from './novel/StepOutline.vue'
import NovelStepOutlineConfirm from './novel/StepOutlineConfirm.vue'
import NovelStepGenerateChapters from './novel/StepGenerateChapters.vue'
import NovelStepIllustrate from './novel/StepIllustrate.vue'
import NovelStepExport from './novel/StepExport.vue'

const store = useWizardStore()

const workType = computed(() => store.workflow)
const step = computed(() => store.step)
const steps = computed(() => store.stepConfig)
</script>

<template>
  <div class="wizard">
    <div class="wizard__header">
      <h3>创作工作流</h3>
      <el-button text :icon="Close" @click="store.reset()">关闭</el-button>
    </div>

    <div class="wizard__steps">
      <el-steps :active="step" align-center>
        <el-step v-for="(label, i) in steps" :key="i" :title="label" />
      </el-steps>
    </div>

    <div class="wizard__body">
      <!-- Image Refine -->
      <template v-if="workType === 'image_refine'">
        <StepSource v-if="step === 0" />
        <StepRefine v-else-if="step === 1" />
        <StepCompare v-else-if="step === 2" />
        <StepExport v-else-if="step === 3" />
      </template>

      <!-- Comic -->
      <template v-if="workType === 'comic'">
        <ComicStepTheme v-if="step === 0" />
        <ComicStepLayout v-else-if="step === 1" />
        <ComicStepPanels v-else-if="step === 2" />
        <ComicStepGenerate v-else-if="step === 3" />
        <ComicStepCaptions v-else-if="step === 4" />
        <ComicStepExport v-else-if="step === 5" />
      </template>

      <!-- Novel -->
      <template v-if="workType === 'novel'">
        <NovelStepTheme v-if="step === 0" />
        <NovelStepGenre v-else-if="step === 1" />
        <NovelStepCharacters v-else-if="step === 2" />
        <NovelStepOutline v-else-if="step === 3" />
        <NovelStepOutlineConfirm v-else-if="step === 4" />
        <NovelStepGenerateChapters v-else-if="step === 5" />
        <NovelStepIllustrate v-else-if="step === 6" />
        <NovelStepExport v-else-if="step === 7" />
      </template>
    </div>

    <div class="wizard__footer">
      <el-button :disabled="store.isFirstStep" :icon="ArrowLeft" @click="store.prevStep()">
        上一步
      </el-button>
      <el-button v-if="!store.isLastStep" type="primary" :icon="ArrowRight" @click="store.nextStep()">
        下一步
      </el-button>
    </div>
  </div>
</template>

<style scoped>
.wizard { max-width: 800px; margin: 0 auto; }
.wizard__header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 24px; }
.wizard__steps { margin-bottom: 32px; }
.wizard__body { min-height: 300px; }
.wizard__footer { display: flex; justify-content: space-between; margin-top: 24px; padding-top: 16px; border-top: 1px solid #f0f0f0; }
</style>
```

- [ ] **Step 3: Modify `frontend/src/components/NavSidebar.vue`**

Add new group to the `groups` array (after `tools` group):

```typescript
{
  id: 'workflow',
  icon: '⚡',
  label: '创作',
  items: [
    { id: 'image_refine', label: '图片精修' },
    { id: 'comic', label: '漫画生成' },
    { id: 'novel', label: '小说生成' },
  ],
},
```

- [ ] **Step 4: Modify `frontend/src/App.vue`**

Add import:
```typescript
import WorkflowWizard from './views/WorkflowWizard.vue'
```

Add conditional renders after the existing `v-else-if` chain:
```vue
<WorkflowWizard v-else-if="activePage === 'image_refine'" />
<WorkflowWizard v-else-if="activePage === 'comic'" />
<WorkflowWizard v-else-if="activePage === 'novel'" />
```

- [ ] **Step 5: Verify typecheck**

```bash
cd frontend && npx vue-tsc --noEmit
```

Expected: No type errors.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/stores/wizard.ts frontend/src/views/WorkflowWizard.vue frontend/src/components/NavSidebar.vue frontend/src/App.vue
git commit -m "feat: add workflow wizard infrastructure"
```

---

### Task 2: Image Refine Workflow

**Files:**
- Create: `frontend/src/views/image/StepSource.vue`
- Create: `frontend/src/views/image/StepRefine.vue`
- Create: `frontend/src/views/image/StepCompare.vue`
- Create: `frontend/src/views/image/StepExport.vue`

**Interfaces:**
- Consumes: `useWizardStore()` from Task 1 (read/write `image` state)
- Produces: 4 step components for the image refine workflow

- [ ] **Step 1: Create `frontend/src/views/image/StepSource.vue`**

Dual entry: text-to-image (generate) or upload.

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { useWizardStore } from '../../stores/wizard'
import { textToImage } from '../../api/image'
import { ElMessage, ElButton, ElInput, ElRadioGroup, ElRadioButton } from 'element-plus'

const store = useWizardStore()
const sourceMode = ref<'generate' | 'upload'>('generate')
const prompt = ref('')
const loading = ref(false)

async function handleGenerate() {
  if (!prompt.value) { ElMessage.warning('请输入提示词'); return }
  loading.value = true
  try {
    const res = await textToImage({ prompt: prompt.value, size: store.image.size })
    store.image.sourceImage = res.images?.[0] || ''
    store.image.sourcePrompt = prompt.value
    store.image.refinePrompt = prompt.value
    store.nextStep()
  } catch (e: any) {
    ElMessage.error('生成失败: ' + (e.message || ''))
  } finally {
    loading.value = false
  }
}

function handleFileUpload(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  store.image.sourceFile = file
  const reader = new FileReader()
  reader.onload = () => {
    store.image.sourceImage = reader.result as string
  }
  reader.readAsDataURL(file)
  store.image.sourceType = 'upload'
}

function proceedWithUpload() {
  if (!store.image.sourceImage) { ElMessage.warning('请先上传图片'); return }
  store.nextStep()
}
</script>

<template>
  <div class="step">
    <h4>选择图片来源</h4>
    <el-radio-group v-model="sourceMode">
      <el-radio-button value="generate">文生图</el-radio-button>
      <el-radio-button value="upload">上传图片</el-radio-button>
    </el-radio-group>

    <div v-if="sourceMode === 'generate'" style="margin-top: 16px">
      <el-input v-model="prompt" type="textarea" :rows="4" placeholder="描述你想生成的图片内容..." />
      <el-button type="primary" :loading="loading" style="margin-top: 12px" @click="handleGenerate">
        生成图片
      </el-button>
      <img v-if="store.image.sourceImage" :src="store.image.sourceImage" style="max-width: 300px; margin-top: 12px; border-radius: 8px" />
    </div>

    <div v-else style="margin-top: 16px">
      <input type="file" accept="image/*" @change="handleFileUpload" />
      <img v-if="store.image.sourceImage" :src="store.image.sourceImage" style="max-width: 300px; margin-top: 12px; border-radius: 8px" />
      <el-button v-if="store.image.sourceImage" type="primary" style="margin-top: 12px" @click="proceedWithUpload">
        下一步：精修
      </el-button>
    </div>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
</style>
```

- [ ] **Step 2: Create `frontend/src/views/image/StepRefine.vue`**

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { useWizardStore } from '../../stores/wizard'
import { imageToImage } from '../../api/image'
import { ElMessage, ElButton, ElInput, ElSlider, ElSelect, ElOption } from 'element-plus'

const store = useWizardStore()
const loading = ref(false)

async function handleRefine() {
  loading.value = true
  try {
    const formData = new FormData()
    if (store.image.sourceFile) {
      formData.append('image', store.image.sourceFile)
    } else {
      formData.append('image_url', store.image.sourceImage)
    }
    formData.append('prompt', store.image.refinePrompt)
    formData.append('size', store.image.size)
    formData.append('strength', String(store.image.strength))

    const res = await imageToImage(formData)
    store.image.resultImage = res.images?.[0] || ''
    store.nextStep()
  } catch (e: any) {
    ElMessage.error('精修失败: ' + (e.message || ''))
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="step refine-layout">
    <div class="refine-layout__preview">
      <h4>原图</h4>
      <img :src="store.image.sourceImage" style="max-width: 100%; border-radius: 8px" />
    </div>
    <div class="refine-layout__params">
      <h4>精修参数</h4>
      <div style="margin-bottom: 16px">
        <label>提示词</label>
        <el-input v-model="store.image.refinePrompt" type="textarea" :rows="3" />
      </div>
      <div style="margin-bottom: 16px">
        <label>强度 ({{ store.image.strength }})</label>
        <el-slider v-model="store.image.strength" :min="0" :max="1" :step="0.05" />
      </div>
      <div style="margin-bottom: 16px">
        <label>尺寸</label>
        <el-select v-model="store.image.size">
          <el-option label="1024x1024" value="1024x1024" />
          <el-option label="768x768" value="768x768" />
          <el-option label="512x512" value="512x512" />
        </el-select>
      </div>
      <el-button type="primary" :loading="loading" @click="handleRefine">开始精修</el-button>
    </div>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.refine-layout { display: flex; gap: 24px; }
.refine-layout__preview { flex: 1; }
.refine-layout__params { width: 320px; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
label { display: block; font-size: 13px; color: #666; margin-bottom: 6px; }
</style>
```

- [ ] **Step 3: Create `frontend/src/views/image/StepCompare.vue`**

```vue
<script setup lang="ts">
import { useWizardStore } from '../../stores/wizard'
import { ElButton } from 'element-plus'

const store = useWizardStore()
</script>

<template>
  <div class="step">
    <h4>对比预览</h4>
    <div class="compare-layout">
      <div class="compare-layout__item">
        <h5>原图</h5>
        <img :src="store.image.sourceImage" style="width: 100%; border-radius: 8px" />
      </div>
      <div class="compare-layout__item">
        <h5>精修结果</h5>
        <img :src="store.image.resultImage" style="width: 100%; border-radius: 8px" />
      </div>
    </div>
    <div style="display: flex; gap: 12px; margin-top: 16px; justify-content: center">
      <el-button @click="store.goToStep(1)">继续精修</el-button>
      <el-button type="primary" @click="store.nextStep()">满意，下一步</el-button>
    </div>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
.compare-layout { display: flex; gap: 24px; }
.compare-layout__item { flex: 1; }
.compare-layout__item h5 { margin: 0 0 8px; font-size: 14px; color: #666; }
</style>
```

- [ ] **Step 4: Create `frontend/src/views/image/StepExport.vue`**

```vue
<script setup lang="ts">
import { useWizardStore } from '../../stores/wizard'
import { ElButton, ElMessage } from 'element-plus'

const store = useWizardStore()

function downloadImage(url: string) {
  const a = document.createElement('a')
  a.href = url
  a.download = `refined_${Date.now()}.png`
  a.click()
  ElMessage.success('下载已开始')
}

function startNew() {
  store.reset()
}
</script>

<template>
  <div class="step" style="text-align: center; padding: 40px 0">
    <h4>精修完成</h4>
    <img :src="store.image.resultImage" style="max-width: 400px; border-radius: 8px; margin: 16px auto" />
    <div style="display: flex; gap: 12px; justify-content: center; margin-top: 16px">
      <el-button type="primary" @click="downloadImage(store.image.resultImage)">下载图片</el-button>
      <el-button @click="startNew">重新开始</el-button>
    </div>
  </div>
</template>

<style scoped>
.step h4 { margin: 0 0 16px; font-size: 16px; }
</style>
```

- [ ] **Step 5: Verify typecheck**

```bash
cd frontend && npx vue-tsc --noEmit
```

Expected: No type errors.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/views/image/
git commit -m "feat: add image refine workflow"
```

---

### Task 3: Comic Workflow

**Files:**
- Create: `frontend/src/views/comic/StepTheme.vue`
- Create: `frontend/src/views/comic/StepLayout.vue`
- Create: `frontend/src/views/comic/StepPanels.vue`
- Create: `frontend/src/views/comic/StepGenerate.vue`
- Create: `frontend/src/views/comic/StepCaptions.vue`
- Create: `frontend/src/views/comic/StepExport.vue`

**Interfaces:**
- Consumes: `useWizardStore()` from Task 1 (read/write `comic` state)

- [ ] **Step 1: Create `frontend/src/views/comic/StepTheme.vue`**

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { useWizardStore } from '../../stores/wizard'
import { ElButton, ElInput, ElMessage } from 'element-plus'

const store = useWizardStore()
const theme = ref(store.comic.theme)

function proceed() {
  if (!theme.value) { ElMessage.warning('请输入漫画主题'); return }
  store.comic.theme = theme.value
  store.nextStep()
}
</script>

<template>
  <div class="step">
    <h4>设定漫画主题</h4>
    <el-input v-model="theme" type="textarea" :rows="4" placeholder="描述你想创作的漫画主题，例如「一只猫的太空冒险」" />
    <el-button type="primary" style="margin-top: 16px" @click="proceed">下一步：选择布局</el-button>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
</style>
```

- [ ] **Step 2: Create `frontend/src/views/comic/StepLayout.vue`**

```vue
<script setup lang="ts">
import { useWizardStore } from '../../stores/wizard'
import { ElButton, ElMessage } from 'element-plus'

const store = useWizardStore()
const layouts = [
  { id: 'single', label: '单格', cols: 1, rows: 1, desc: '一张图' },
  { id: 'dual', label: '双格', cols: 1, rows: 2, desc: '上下两格' },
  { id: 'quad', label: '四格', cols: 2, rows: 2, desc: '田字四格' },
  { id: 'six', label: '六格', cols: 2, rows: 3, desc: '2x3 六格' },
] as const

function selectLayout(id: string) {
  store.comic.layout = id as typeof store.comic.layout
  // Initialize panels
  const layout = layouts.find(l => l.id === id)!
  const count = layout.cols * layout.rows
  store.comic.panels = Array.from({ length: count }, () => ({ prompt: '', image: '', caption: '' }))
  store.nextStep()
}
</script>

<template>
  <div class="step">
    <h4>选择分格布局</h4>
    <div class="layout-grid">
      <div
        v-for="layout in layouts"
        :key="layout.id"
        class="layout-card"
        @click="selectLayout(layout.id)"
      >
        <div class="layout-card__preview" :style="{ gridTemplateColumns: `repeat(${layout.cols}, 1fr)`, gridTemplateRows: `repeat(${layout.rows}, 1fr)` }">
          <div v-for="n in layout.cols * layout.rows" :key="n" class="layout-card__cell">{{ n }}</div>
        </div>
        <div class="layout-card__label">{{ layout.label }}</div>
        <div class="layout-card__desc">{{ layout.desc }}</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
.layout-grid { display: flex; gap: 16px; flex-wrap: wrap; }
.layout-card { border: 1px solid #eaeaea; border-radius: 12px; padding: 16px; cursor: pointer; transition: border-color 0.2s; width: 160px; text-align: center; }
.layout-card:hover { border-color: #000; }
.layout-card__preview { display: grid; gap: 4px; width: 120px; height: 120px; margin: 0 auto 8px; }
.layout-card__cell { border: 1px solid #ddd; border-radius: 4px; display: flex; align-items: center; justify-content: center; font-size: 12px; color: #909399; background: #fafafa; }
.layout-card__label { font-weight: 600; font-size: 14px; }
.layout-card__desc { font-size: 12px; color: #909399; }
</style>
```

- [ ] **Step 3: Create `frontend/src/views/comic/StepPanels.vue`**

```vue
<script setup lang="ts">
import { useWizardStore } from '../../stores/wizard'
import { ElButton, ElInput, ElMessage } from 'element-plus'

const store = useWizardStore()

function proceed() {
  const empty = store.comic.panels.some(p => !p.prompt)
  if (empty) { ElMessage.warning('请填写所有分格的提示词'); return }
  store.nextStep()
}
</script>

<template>
  <div class="step">
    <h4>填写每格提示词</h4>
    <div class="panels-grid" :style="{ gridTemplateColumns: store.comic.layout === 'six' ? 'repeat(2, 1fr)' : 'repeat(2, 1fr)' }">
      <div v-for="(panel, i) in store.comic.panels" :key="i" class="panel-card">
        <div class="panel-card__header">第 {{ i + 1 }} 格</div>
        <el-input v-model="panel.prompt" type="textarea" :rows="3" placeholder="描述这个格子的画面内容" />
      </div>
    </div>
    <el-button type="primary" style="margin-top: 16px" @click="proceed">下一步：批量生成</el-button>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
.panels-grid { display: grid; gap: 16px; }
.panel-card { border: 1px solid #eaeaea; border-radius: 12px; padding: 12px; }
.panel-card__header { font-weight: 600; font-size: 13px; margin-bottom: 8px; }
</style>
```

- [ ] **Step 4: Create `frontend/src/views/comic/StepGenerate.vue`**

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { useWizardStore } from '../../stores/wizard'
import { textToImage } from '../../api/image'
import { ElButton, ElMessage } from 'element-plus'

const store = useWizardStore()
const generating = ref(false)
const currentIndex = ref(-1)

async function generateAll() {
  generating.value = true
  for (let i = 0; i < store.comic.panels.length; i++) {
    currentIndex.value = i
    const panel = store.comic.panels[i]
    if (panel.image) continue // skip if already generated
    try {
      const res = await textToImage({ prompt: panel.prompt, size: '1024x1024' })
      panel.image = res.images?.[0] || ''
    } catch (e: any) {
      ElMessage.error(`第 ${i + 1} 格生成失败: ${e.message || ''}`)
    }
  }
  currentIndex.value = -1
  generating.value = false
  store.nextStep()
}
</script>

<template>
  <div class="step">
    <h4>批量生成</h4>
    <div class="panels-grid" :style="{ gridTemplateColumns: 'repeat(2, 1fr)' }">
      <div v-for="(panel, i) in store.comic.panels" :key="i" class="panel-card" :class="{ generating: currentIndex === i }">
        <div class="panel-card__header">第 {{ i + 1 }} 格</div>
        <div class="panel-card__prompt">{{ panel.prompt }}</div>
        <img v-if="panel.image" :src="panel.image" style="width: 100%; border-radius: 8px; margin-top: 8px" />
        <div v-else class="panel-card__placeholder">{{ currentIndex === i ? '生成中...' : '待生成' }}</div>
      </div>
    </div>
    <el-button type="primary" :loading="generating" style="margin-top: 16px" @click="generateAll">
      {{ generating ? `正在生成第 ${currentIndex + 1}/${store.comic.panels.length} 格...` : '全部生成' }}
    </el-button>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
.panels-grid { display: grid; gap: 16px; }
.panel-card { border: 1px solid #eaeaea; border-radius: 12px; padding: 12px; }
.panel-card.generating { border-color: #409eff; background: #f0f7ff; }
.panel-card__header { font-weight: 600; font-size: 13px; margin-bottom: 4px; }
.panel-card__prompt { font-size: 12px; color: #666; margin-bottom: 8px; }
.panel-card__placeholder { height: 100px; display: flex; align-items: center; justify-content: center; background: #fafafa; border-radius: 8px; color: #c0c4cc; font-size: 13px; }
</style>
```

- [ ] **Step 5: Create `frontend/src/views/comic/StepCaptions.vue`**

```vue
<script setup lang="ts">
import { useWizardStore } from '../../stores/wizard'
import { ElButton, ElInput } from 'element-plus'

const store = useWizardStore()
</script>

<template>
  <div class="step">
    <h4>添加台词</h4>
    <div class="captions-grid" :style="{ gridTemplateColumns: 'repeat(2, 1fr)' }">
      <div v-for="(panel, i) in store.comic.panels" :key="i" class="caption-card">
        <img :src="panel.image" style="width: 100%; border-radius: 8px; margin-bottom: 8px" />
        <el-input v-model="panel.caption" placeholder="输入台词/对白..." />
      </div>
    </div>
    <el-button type="primary" style="margin-top: 16px" @click="store.nextStep()">下一步：导出</el-button>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
.captions-grid { display: grid; gap: 16px; }
.caption-card { border: 1px solid #eaeaea; border-radius: 12px; padding: 12px; }
</style>
```

- [ ] **Step 6: Create `frontend/src/views/comic/StepExport.vue`**

```vue
<script setup lang="ts">
import { useWizardStore } from '../../stores/wizard'
import { ElButton, ElMessage, ElRadioGroup, ElRadioButton, ref } from 'element-plus'

const store = useWizardStore()
const format = ref('html')

function exportComic() {
  if (format.value === 'html') {
    let html = `<!DOCTYPE html><html><head><meta charset="utf-8"><title>${store.comic.theme}</title><style>body{font-family:sans-serif;max-width:800px;margin:0 auto;padding:20px}.grid{display:grid;grid-template-columns:repeat(2,1fr);gap:16px}.panel{border:1px solid #eaeaea;border-radius:12px;padding:12px;text-align:center}.panel img{width:100%;border-radius:8px}.caption{margin-top:8px;font-size:14px;color:#333}</style></head><body><h1>${store.comic.theme}</h1><div class="grid">`
    store.comic.panels.forEach(p => {
      html += `<div class="panel"><img src="${p.image}"><div class="caption">${p.caption || ''}</div></div>`
    })
    html += `</div></body></html>`
    const blob = new Blob([html], { type: 'text/html' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url; a.download = `${store.comic.theme}.html`; a.click()
    URL.revokeObjectURL(url)
    ElMessage.success('导出成功')
  } else {
    ElMessage.info('其他格式导出待实现')
  }
}
</script>

<template>
  <div class="step" style="text-align: center; padding: 40px 0">
    <h4>漫画完成！</h4>
    <div class="export-preview">
      <div v-for="panel in store.comic.panels" :key="panel.prompt" class="export-panel">
        <img :src="panel.image" style="width: 100%; border-radius: 8px" />
        <div v-if="panel.caption" class="export-caption">{{ panel.caption }}</div>
      </div>
    </div>
    <div style="margin-top: 16px">
      <label>导出格式：</label>
      <el-radio-group v-model="format">
        <el-radio-button value="html">HTML</el-radio-button>
        <el-radio-button value="png">PNG</el-radio-button>
      </el-radio-group>
    </div>
    <el-button type="primary" style="margin-top: 16px" @click="exportComic">导出</el-button>
    <el-button style="margin-top: 16px; margin-left: 8px" @click="store.reset()">重新开始</el-button>
  </div>
</template>

<style scoped>
.step h4 { margin: 0 0 16px; font-size: 16px; }
.export-preview { display: grid; grid-template-columns: repeat(2, 1fr); gap: 12px; max-width: 500px; margin: 0 auto; }
.export-panel { border: 1px solid #eaeaea; border-radius: 12px; padding: 8px; }
.export-caption { margin-top: 8px; font-size: 13px; color: #333; text-align: center; }
</style>
```

- [ ] **Step 7: Verify typecheck**

```bash
cd frontend && npx vue-tsc --noEmit
```

Expected: No type errors.

- [ ] **Step 8: Commit**

```bash
git add frontend/src/views/comic/
git commit -m "feat: add comic generation workflow"
```

---

### Task 4: Novel Workflow

**Files:**
- Create: `frontend/src/views/novel/StepTheme.vue`
- Create: `frontend/src/views/novel/StepGenre.vue`
- Create: `frontend/src/views/novel/StepCharacters.vue`
- Create: `frontend/src/views/novel/StepOutline.vue`
- Create: `frontend/src/views/novel/StepOutlineConfirm.vue`
- Create: `frontend/src/views/novel/StepGenerateChapters.vue`
- Create: `frontend/src/views/novel/StepIllustrate.vue`
- Create: `frontend/src/views/novel/StepExport.vue`

**Interfaces:**
- Consumes: `useWizardStore()` from Task 1 (read/write `novel` state)

- [ ] **Step 1: Create `frontend/src/views/novel/StepTheme.vue`**

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { useWizardStore } from '../../stores/wizard'
import { ElButton, ElInput, ElMessage } from 'element-plus'

const store = useWizardStore()
const theme = ref(store.novel.theme)

function proceed() {
  if (!theme.value) { ElMessage.warning('请输入小说主题'); return }
  store.novel.theme = theme.value
  store.nextStep()
}
</script>

<template>
  <div class="step">
    <h4>选题设定</h4>
    <el-input v-model="theme" type="textarea" :rows="4" placeholder="输入小说主题或一句话灵感，例如「一个程序员穿越到魔法世界」" />
    <el-button type="primary" style="margin-top: 16px" @click="proceed">下一步：选择风格</el-button>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
</style>
```

- [ ] **Step 2: Create `frontend/src/views/novel/StepGenre.vue`**

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { useWizardStore } from '../../stores/wizard'
import { expandIdea } from '../../api/ideas'
import { ElButton, ElTag, ElMessage } from 'element-plus'

const store = useWizardStore()
const genres = ['玄幻', '科幻', '言情', '悬疑', '现实主义', '历史', '武侠', '恐怖', '喜剧', '冒险']
const selected = ref(store.novel.genre)
const suggestions = ref<string[]>([])

async function getSuggestions() {
  try {
    const res = await expandIdea({ prompt: `基于主题"${store.novel.theme}"推荐3个创意写作风格方向`, language: 'zh' })
    suggestions.value = (res.result || '').split('\n').filter(Boolean)
  } catch { /* ignore */ }
}

function selectGenre(genre: string) {
  selected.value = genre
  store.novel.genre = genre
}

function proceed() {
  if (!selected.value) { ElMessage.warning('请选择一个风格'); return }
  store.nextStep()
}
</script>

<template>
  <div class="step">
    <h4>选择风格/流派</h4>
    <div style="display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 16px">
      <el-tag
        v-for="g in genres"
        :key="g"
        :type="selected === g ? 'primary' : 'info'"
        style="cursor: pointer; padding: 6px 16px; font-size: 14px"
        @click="selectGenre(g)"
      >
        {{ g }}
      </el-tag>
    </div>
    <el-button text @click="getSuggestions">💡 灵感建议</el-button>
    <div v-if="suggestions.length" style="margin-top: 8px; color: #666; font-size: 13px">
      <p v-for="(s, i) in suggestions" :key="i">{{ s }}</p>
    </div>
    <el-button type="primary" style="margin-top: 16px" @click="proceed">下一步：角色设置</el-button>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
</style>
```

- [ ] **Step 3: Create `frontend/src/views/novel/StepCharacters.vue`**

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { useWizardStore } from '../../stores/wizard'
import { ElButton, ElInput, ElMessage } from 'element-plus'

const store = useWizardStore()
const characters = ref(store.novel.characters.length ? store.novel.characters : [{ name: '', personality: '', appearance: '' }])

function addCharacter() {
  characters.value.push({ name: '', personality: '', appearance: '' })
}

function removeCharacter(i: number) {
  characters.value.splice(i, 1)
}

function proceed() {
  const empty = characters.value.some(c => !c.name)
  if (empty) { ElMessage.warning('请填写所有角色名'); return }
  store.novel.characters = JSON.parse(JSON.stringify(characters.value))
  store.nextStep()
}
</script>

<template>
  <div class="step">
    <h4>角色设置</h4>
    <div v-for="(char, i) in characters" :key="i" class="char-card">
      <div class="char-card__header">角色 {{ i + 1 }} <el-button text type="danger" size="small" @click="removeCharacter(i)">删除</el-button></div>
      <el-input v-model="char.name" placeholder="角色名" style="margin-bottom: 8px" />
      <el-input v-model="char.personality" placeholder="性格描述" style="margin-bottom: 8px" />
      <el-input v-model="char.appearance" placeholder="外观描述" />
    </div>
    <el-button text style="margin-top: 8px" @click="addCharacter">+ 添加角色</el-button>
    <div style="margin-top: 16px">
      <el-button type="primary" @click="proceed">下一步：生成大纲</el-button>
    </div>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
.char-card { border: 1px solid #eaeaea; border-radius: 12px; padding: 12px; margin-bottom: 12px; }
.char-card__header { display: flex; justify-content: space-between; font-weight: 600; margin-bottom: 8px; }
</style>
```

- [ ] **Step 4: Create `frontend/src/views/novel/StepOutline.vue`**

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { useWizardStore } from '../../stores/wizard'
import { expandIdea } from '../../api/ideas'
import { ElButton, ElMessage } from 'element-plus'

const store = useWizardStore()
const loading = ref(false)
const outline = ref(store.novel.outline)

async function generateOutline() {
  loading.value = true
  try {
    const charDesc = store.novel.characters.map(c => `${c.name}（${c.personality}，${c.appearance}）`).join('；')
    const prompt = `请为一部小说创作详细的大纲。主题：${store.novel.theme}。风格：${store.novel.genre}。角色：${charDesc}。请包含：故事背景、主要冲突、章节概要（5-8章）。用中文回复。`
    const res = await expandIdea({ prompt, language: 'zh' })
    outline.value = res.result || ''
    store.novel.outline = outline.value
  } catch (e: any) {
    ElMessage.error('生成大纲失败: ' + (e.message || ''))
  } finally {
    loading.value = false
  }
}

function proceed() {
  if (!outline.value) { ElMessage.warning('请先生成大纲'); return }
  store.novel.outline = outline.value
  store.nextStep()
}
</script>

<template>
  <div class="step">
    <h4>大纲生成</h4>
    <el-button :loading="loading" @click="generateOutline">生成大纲</el-button>
    <div v-if="outline" class="outline-content">{{ outline }}</div>
    <el-button v-if="outline" type="primary" style="margin-top: 16px" @click="proceed">确认大纲</el-button>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
.outline-content { white-space: pre-wrap; background: #fafafa; border-radius: 8px; padding: 16px; margin-top: 16px; line-height: 1.8; font-size: 14px; }
</style>
```

- [ ] **Step 5: Create `frontend/src/views/novel/StepOutlineConfirm.vue`**

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { useWizardStore } from '../../stores/wizard'
import { expandIdea } from '../../api/ideas'
import { ElButton, ElInput, ElMessage } from 'element-plus'

const store = useWizardStore()
const editing = ref(false)
const editedOutline = ref(store.novel.outline)
const reloading = ref(false)

async function regenerate() {
  reloading.value = true
  try {
    const charDesc = store.novel.characters.map(c => `${c.name}（${c.personality}，${c.appearance}）`).join('；')
    const prompt = `请重新为一部小说创作大纲。主题：${store.novel.theme}。风格：${store.novel.genre}。角色：${charDesc}。请提供全新的章节结构。用中文回复。`
    const res = await expandIdea({ prompt, language: 'zh' })
    editedOutline.value = res.result || ''
    store.novel.outline = editedOutline.value
  } finally {
    reloading.value = false
  }
}

function confirm() {
  store.novel.outline = editedOutline.value
  store.nextStep()
}
</script>

<template>
  <div class="step">
    <h4>确认大纲</h4>
    <el-button text @click="editing = !editing">{{ editing ? '预览' : '手动编辑' }}</el-button>
    <el-button text :loading="reloading" @click="regenerate">重新生成</el-button>
    <div v-if="editing">
      <el-input v-model="editedOutline" type="textarea" :rows="12" />
    </div>
    <div v-else class="outline-content">{{ editedOutline }}</div>
    <el-button type="primary" style="margin-top: 16px" @click="confirm">确认，开始创作</el-button>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
.outline-content { white-space: pre-wrap; background: #fafafa; border-radius: 8px; padding: 16px; margin-top: 16px; line-height: 1.8; font-size: 14px; }
</style>
```

- [ ] **Step 6: Create `frontend/src/views/novel/StepGenerateChapters.vue`**

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { useWizardStore } from '../../stores/wizard'
import { expandIdea } from '../../api/ideas'
import { ElButton, ElMessage } from 'element-plus'

const store = useWizardStore()
const generating = ref(false)

async function generateNextChapter() {
  generating.value = true
  const chapterNum = store.novel.chapters.length + 1
  const previousContent = store.novel.chapters.map(c => c.title + '\n' + c.content).join('\n\n')
  try {
    const prompt = `继续写小说「${store.novel.theme}」（${store.novel.genre}风格）。大纲：${store.novel.outline}。已写内容：\n${previousContent || '尚未开始'}\n\n请写第${chapterNum}章，包含标题和正文。用中文回复。`
    const res = await expandIdea({ prompt, language: 'zh' })
    const text = res.result || ''
    const lines = text.split('\n')
    const title = lines[0]?.replace(/^#+\s*/, '') || `第${chapterNum}章`
    const content = lines.slice(1).join('\n').trim()
    store.novel.chapters.push({ title, content })
  } catch (e: any) {
    ElMessage.error('生成章节失败: ' + (e.message || ''))
  } finally {
    generating.value = false
  }
}
</script>

<template>
  <div class="step">
    <h4>逐章生成</h4>
    <div v-for="(ch, i) in store.novel.chapters" :key="i" class="chapter-card">
      <div class="chapter-card__title">{{ ch.title }}</div>
      <div class="chapter-card__content">{{ ch.content.slice(0, 200) }}...</div>
    </div>
    <el-button type="primary" :loading="generating" style="margin-top: 16px" @click="generateNextChapter">
      {{ generating ? '生成中...' : `生成第 ${store.novel.chapters.length + 1} 章` }}
    </el-button>
    <el-button v-if="store.novel.chapters.length >= 3" style="margin-top: 16px; margin-left: 8px" @click="store.nextStep()">
      下一步：配插图
    </el-button>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
.chapter-card { border: 1px solid #eaeaea; border-radius: 12px; padding: 12px; margin-bottom: 12px; }
.chapter-card__title { font-weight: 600; margin-bottom: 8px; }
.chapter-card__content { font-size: 13px; color: #666; line-height: 1.6; }
</style>
```

- [ ] **Step 7: Create `frontend/src/views/novel/StepIllustrate.vue`**

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { useWizardStore } from '../../stores/wizard'
import { textToImage } from '../../api/image'
import { ElButton, ElMessage } from 'element-plus'

const store = useWizardStore()
const generating = ref(false)
const currentChapter = ref(-1)

async function generateIllustration(chapterIndex: number) {
  currentChapter.value = chapterIndex
  const ch = store.novel.chapters[chapterIndex]
  try {
    const prompt = `为小说章节绘制插图。小说主题：${store.novel.theme}。章节：${ch.title}。内容提要：${ch.content.slice(0, 100)}`
    const res = await textToImage({ prompt, size: '1024x1024' })
    ch.illustration = res.images?.[0] || ''
  } catch (e: any) {
    ElMessage.error(`插图生成失败: ${e.message || ''}`)
  } finally {
    currentChapter.value = -1
  }
}
</script>

<template>
  <div class="step">
    <h4>配插图</h4>
    <div v-for="(ch, i) in store.novel.chapters" :key="i" class="chapter-card">
      <div class="chapter-card__header">
        <span class="chapter-card__title">{{ ch.title }}</span>
        <el-button
          size="small"
          :loading="currentChapter === i"
          :disabled="!!ch.illustration"
          @click="generateIllustration(i)"
        >
          {{ ch.illustration ? '已生成' : '生成插图' }}
        </el-button>
      </div>
      <img v-if="ch.illustration" :src="ch.illustration" style="max-width: 200px; border-radius: 8px; margin-top: 8px" />
    </div>
    <el-button type="primary" style="margin-top: 16px" @click="store.nextStep()">下一步：导出</el-button>
  </div>
</template>

<style scoped>
.step { padding: 16px 0; }
.step h4 { margin: 0 0 16px; font-size: 16px; }
.chapter-card { border: 1px solid #eaeaea; border-radius: 12px; padding: 12px; margin-bottom: 12px; }
.chapter-card__header { display: flex; justify-content: space-between; align-items: center; }
.chapter-card__title { font-weight: 600; }
</style>
```

- [ ] **Step 8: Create `frontend/src/views/novel/StepExport.vue`**

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { useWizardStore } from '../../stores/wizard'
import { ElButton, ElMessage, ElRadioGroup, ElRadioButton } from 'element-plus'

const store = useWizardStore()
const format = ref('markdown')

function exportNovel() {
  let content = ''
  if (format.value === 'markdown') {
    content = `# ${store.novel.theme}\n\n**风格：** ${store.novel.genre}\n\n---\n\n`
    store.novel.chapters.forEach(ch => {
      content += `## ${ch.title}\n\n${ch.content}\n\n`
      if (ch.illustration) content += `![插图](${ch.illustration})\n\n`
    })
  } else {
    content = `${store.novel.theme}\n${'='.repeat(store.novel.theme.length)}\n\n`
    store.novel.chapters.forEach(ch => {
      content += `${ch.title}\n${'-'.repeat(ch.title.length)}\n\n${ch.content}\n\n`
    })
  }
  const blob = new Blob([content], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url; a.download = `${store.novel.theme}.${format.value === 'markdown' ? 'md' : 'txt'}`; a.click()
  URL.revokeObjectURL(url)
  ElMessage.success('导出成功')
}
</script>

<template>
  <div class="step" style="text-align: center; padding: 40px 0">
    <h4>小说完成！</h4>
    <p style="color: #666; margin: 8px 0">共 {{ store.novel.chapters.length }} 章</p>
    <div style="margin-top: 16px">
      <el-radio-group v-model="format">
        <el-radio-button value="markdown">Markdown</el-radio-button>
        <el-radio-button value="text">纯文本</el-radio-button>
      </el-radio-group>
    </div>
    <el-button type="primary" style="margin-top: 16px" @click="exportNovel">导出</el-button>
    <el-button style="margin-top: 16px; margin-left: 8px" @click="store.reset()">重新开始</el-button>
  </div>
</template>

<style scoped>
.step h4 { margin: 0 0 16px; font-size: 16px; }
</style>
```

- [ ] **Step 9: Verify typecheck**

```bash
cd frontend && npx vue-tsc --noEmit
```

Expected: No type errors.

- [ ] **Step 10: Commit**

```bash
git add frontend/src/views/novel/
git commit -m "feat: add novel generation workflow"
```

---

### Task 5: Integration Verification

**Files:** No new files — verify everything works together.

- [ ] **Step 1: Run frontend typecheck**

```bash
cd frontend && npx vue-tsc --noEmit
```

Expected: No type errors.

- [ ] **Step 2: Run frontend build**

```bash
cd frontend && pnpm build
```

Expected: Build succeeds.

- [ ] **Step 3: Verify backend still builds**

```bash
cd backend && go build ./...
```

Expected: Build succeeds (no backend changes, but verify nothing broken).

- [ ] **Step 4: Start dev server and verify**

```bash
cd frontend && pnpm dev &
# Open http://localhost:5173
# Verify: ⚡ icon in NavSidebar, click to open workflow group, each workflow loads correct steps
```

Expected: Dev server starts, navigation works, steps render.

- [ ] **Step 5: Final commit**

```bash
git add -A
git commit -m "feat: complete workflow wizard with 3 creative flows"
```

---

## File Summary

| File | Action |
|---|---|
| `frontend/src/stores/wizard.ts` | CREATE |
| `frontend/src/views/WorkflowWizard.vue` | CREATE |
| `frontend/src/views/image/StepSource.vue` | CREATE |
| `frontend/src/views/image/StepRefine.vue` | CREATE |
| `frontend/src/views/image/StepCompare.vue` | CREATE |
| `frontend/src/views/image/StepExport.vue` | CREATE |
| `frontend/src/views/comic/StepTheme.vue` | CREATE |
| `frontend/src/views/comic/StepLayout.vue` | CREATE |
| `frontend/src/views/comic/StepPanels.vue` | CREATE |
| `frontend/src/views/comic/StepGenerate.vue` | CREATE |
| `frontend/src/views/comic/StepCaptions.vue` | CREATE |
| `frontend/src/views/comic/StepExport.vue` | CREATE |
| `frontend/src/views/novel/StepTheme.vue` | CREATE |
| `frontend/src/views/novel/StepGenre.vue` | CREATE |
| `frontend/src/views/novel/StepCharacters.vue` | CREATE |
| `frontend/src/views/novel/StepOutline.vue` | CREATE |
| `frontend/src/views/novel/StepOutlineConfirm.vue` | CREATE |
| `frontend/src/views/novel/StepGenerateChapters.vue` | CREATE |
| `frontend/src/views/novel/StepIllustrate.vue` | CREATE |
| `frontend/src/views/novel/StepExport.vue` | CREATE |
| `frontend/src/components/NavSidebar.vue` | MODIFY |
| `frontend/src/App.vue` | MODIFY |
