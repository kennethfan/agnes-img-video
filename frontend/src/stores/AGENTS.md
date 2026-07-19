# frontend/src/stores/ — Pinia Stores

**2 files** — cross-view data passing (`redo.ts`) and wizard workflow state machine (`wizard.ts`).

## FILE INDEX

| File | Export | Purpose |
|------|--------|---------|
| `redo.ts` | `useRedoStore`, `RedoData`, `modeToTab` | Cross-view redo: push input data from one view to another via sessionStorage + router navigation |
| `wizard.ts` | `useWizardStore`, `WorkflowType`, `ComicData`, `NovelData`, `ImageRefineData` | Multi-step wizard state machine for 3 workflows (image refine / comic / novel) |

## CORE

### Cross-View Redo (`redo.ts`)

The redo store lets one view pass its generation parameters to any other view. The flow:

```
setRedoData({ mode: 'text2image', prompt: '...' })
  → saves to sessionStorage
  → router.push({ name: modeToTab[mode] })
  → target view reads via consumeRedoData()
```

**`consumeRedoData()`** is a one-shot reader — it reads from `sessionStorage` first, then falls back to in-memory `redoData`, then clears both. Views call this in `onMounted`:

```ts
const redoStore = useRedoStore()
onMounted(() => {
  const redo = redoStore.consumeRedoData()
  if (redo) { /* populate form fields from redo.data */ }
})
```

**`modeToTab`** maps mode strings to Vue Router route names:

| Mode | Route Name |
|------|-----------|
| `text2image` | `text2img` |
| `image2image` | `img2img` |
| `batch` | `batch` |
| `script_gen` | `script_gen` |
| `text2video` | `text2vid` |
| `image2video` | `img2vid` |
| `multi_image_video` | `multi_vid` |
| `image_refine` | `image_refine` |
| `comic` | `comic` |
| `novel` | `novel` |

**Available `RedoData` fields** — each mode populates a subset:

- **All modes**: `mode`
- **Text-to-Image**: `prompt`, `negativePrompt`, `size`, `n`
- **Image-to-Image / Image-to-Video**: `inputMode` (`'upload'` / `'url'`), `imageUrl`, `strength`
- **Batch**: `promptsText`
- **Script generation**: `topic`, `style`, `language`, `script`
- **Video**: `duration`, `aspectRatio`, `frameRate`
- **Multi-image video**: `imageUrlsText`, `videoMode` (`'normal'` / `'keyframes'`)

### Wizard State Machine (`wizard.ts`)

The wizard store manages multi-step creation workflows for image refinement, comics, and novels.

**Workflows and step counts:**

| Workflow | Steps | Flow |
|----------|-------|------|
| `image_refine` | 4 | 选择来源 → 精修调参 → 对比预览 → 导出 |
| `comic` | 6 | 设定主题 → 选择布局 → 填写提示词 → 批量生成 → 添加台词 → 导出 |
| `novel` | 8 | 选题设定 → 风格选择 → 角色设置 → 大纲生成 → 确认大纲 → 逐章生成 → 配插图 → 导出 |

**Core actions:**

```
startWorkflow(type)  — initialize step=0, totalSteps from config
nextStep()           — step++
prevStep()           — step--
goToStep(n)          — jump to specific step
reset()              — clear all state (workflow=null, step=0)
```

**Data structures** — each workflow has its own typed data in the store:

```ts
// ComicData — panels are indexed by layout cells
interface ComicData {
  theme: string
  layout: 'single' | 'dual' | 'quad' | 'six'
  panels: { prompt: string; image: string; caption: string }[]
}

// NovelData — chapters with optional illustrations
interface NovelData {
  theme: string
  genre: string
  characters: { name: string; personality: string; appearance: string }[]
  outline: string
  chapters: { title: string; content: string; illustration?: string }[]
}

// ImageRefineData — source → refine → result lifecycle
interface ImageRefineData {
  sourceType: 'generate' | 'upload'
  sourcePrompt: string
  sourceImage: string
  sourceFile?: File
  refinePrompt: string
  strength: number
  size: string
  resultImage: string
}
```

**Getters:**

- `stepConfig` — auto-selects step labels based on active workflow
- `currentStepLabel` — human-readable current step name
- `isFirstStep` / `isLastStep` — navigation guards

## ANTI-PATTERNS

- **Do NOT call `consumeRedoData()` outside `onMounted`** — it consumes and clears the data; calling it in watchers or event handlers will miss the data or consume it before the target view mounts
- **Do NOT read/write `sessionStorage` directly** — use the store's `setRedoData()` / `consumeRedoData()` to ensure consistency with in-memory state
- **Do NOT persist wizard state** — the wizard is a one-session flow; refreshing the page should reset it (no localStorage)
- **Do NOT store file blobs in Pinia state** — `ImageRefineData.sourceFile` is a `File` reference only; actual file uploads happen via multipart form at the API layer
