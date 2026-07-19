# 图片精修流程改进 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为图生图（Image-to-Image）流程增加迭代闭环（结果可一键继续精修）和输入源扩展（可从作品库选取图片）。

**Architecture:** 纯前端修改，复用现有 Pinia redo store 实现跨视图参数传递。ImageResult.vue 新增「继续精修」按钮触发 `redoStore.setRedoData()` 跳转到 ImageToImage 页面填写参数。ImageToImage.vue 新增「从作品库选择」按钮弹出 el-dialog 展示资产列表，选中后自动填充 URL。

**Tech Stack:** Vue 3 (Composition API + `<script setup>`) · Element Plus · Pinia · TypeScript 6 · Axios

**Spec:** `docs/superpowers/specs/2026-07-11-image-refinement-flow-design.md`

## Global Constraints

- 仅前端修改，无后端变更
- TypeScript 6 `erasableSyntaxOnly` — 无 enum，使用 `as const` 或 union type
- 使用 Composition API + `<script setup>`，不使用 Options API
- 不允许 `as any` / `@ts-ignore` / `@ts-expect-error`
- 所有 UI 文本使用中文
- 复用现有 `src/api/assets.ts` 中的 `getAssets()` 和 `saveAsset()` 接口
- 复用现有 `src/stores/redo.ts` 中的 `useRedoStore`

---
---

### Task 1: ImageResult.vue — 新增「继续精修」按钮

**Files:**
- Modify: `frontend/src/components/ImageResult.vue`
- API: `src/stores/redo.ts` (仅使用，不修改)

**Interfaces:**
- Consumes: `props.mode: string`, `props.images: string[]`, `props.prompt: string`（已存在）
- Produces: 按钮点击触发 `redoStore.setRedoData(...)` 跳转到 ImageToImage 页面

- [ ] **Step 1: 在 ImageResult.vue 的 `<script setup>` 中导入 redo store**

在现有 import 块末尾（`'../api/assets'` 之后）添加：

```ts
import { useRedoStore } from '../stores/redo'
```

并将 store 实例化：

```ts
const redoStore = useRedoStore()
```

- [ ] **Step 2: 在 `<script setup>` 中新增 `handleRefine` 函数**

在 `handleSaveToGallery` 函数之后添加：

```ts
function handleRefine(img: string) {
  redoStore.setRedoData({
    mode: 'image2image',
    imageUrl: img,
    prompt: props.prompt,
    inputMode: 'url',
  })
}
```

> 说明：`setRedoData` 会自动通过 `router.push({ name: 'img2img' })` 跳转到图生图页面。
> ImageToImage.vue 的 `watch(redoData)` 会收到 `mode === 'image2image'` 的数据，自动填充 `prompt`、`imageUrl`、`inputMode`。
> `size`、`strength`、`negativePrompt` 不会通过 redo 传递（保持为上次用户手动设置的值，更符合直觉）。

- [ ] **Step 3: 在 template 中添加「继续精修」按钮**

在现有的操作区（`class="image-actions"`）中，`保存到作品库` 按钮之后，添加：

```vue
<el-button
  v-if="props.mode === 'image2image'"
  size="small"
  @click="handleRefine(img)"
>
  继续精修
</el-button>
```

- [ ] **Step 4: 验证 TypeScript**

Run: `pnpm build` 或 `npx vue-tsc -b`
Expected: 无类型错误

- [ ] **Step 5: 提交**

```bash
git add frontend/src/components/ImageResult.vue
git commit -m "feat: ImageResult 新增继续精修按钮 - mode=image2image 时显示, 点击通过 redo store 跳转到图生图页面"
```

---

### Task 2: ImageToImage.vue — 新增「从作品库选择」按钮 + 弹窗

**Files:**
- Modify: `frontend/src/views/ImageToImage.vue`
- API: `src/api/assets.ts` 中的 `getAssets()`
- Type: `src/types/index.ts` 中的 `AssetItem`

**Interfaces:**
- Consumes: `getAssets(params): Promise<AssetListResponse>`（来自 `src/api/assets.ts`），`AssetItem`（来自 `src/types/index.ts`）
- Produces: 选中作品库图片后填充 `imageUrl` + 切换到 `url` 模式

- [ ] **Step 1: 在 ImageToImage.vue 中导入所需模块**

在现有 import 块中添加：

```ts
import { getAssets } from '../api/assets'
import type { AssetItem } from '../types'
```

- [ ] **Step 2: 添加响应式状态**

在现有 ref 声明区域（`const errorMsg = ref('')` 之后）添加：

```ts
const galleryDialogVisible = ref(false)
const assetList = ref<AssetItem[]>([])
const assetLoading = ref(false)
```

- [ ] **Step 3: 新增 `openGallery` 函数**

在 `handleGenerate` 函数之前添加：

```ts
async function openGallery() {
  galleryDialogVisible.value = true
  if (assetList.value.length > 0) return // 已有缓存不重复加载
  assetLoading.value = true
  try {
    const res = await getAssets({ type: 'image', per_page: 50 })
    assetList.value = res.items || []
  } catch (e: any) {
    ElMessage.error('加载作品库失败: ' + (e.message || '未知错误'))
  } finally {
    assetLoading.value = false
  }
}

function selectAsset(item: AssetItem) {
  const url = item.local_path || item.github_url || item.original_url
  if (!url) {
    ElMessage.warning('该图片没有可用的 URL')
    return
  }
  imageUrl.value = url
  inputMode.value = 'url'
  galleryDialogVisible.value = false
  ElMessage.success('已选择图片')
}
```

- [ ] **Step 4: 在 template 的 inputMode radio-group 下方添加「从作品库选择」按钮**

在 `<el-form-item label="输入方式">` 的 `el-radio-group` 结束标签 `</el-radio-group>` 之后添加：

```vue
<el-form-item label=" ">
  <el-button @click="openGallery" :icon="Picture" size="small">
    从作品库选择
  </el-button>
</el-form-item>
```

并导入 Element Plus 图标：

在 `import { UploadFilled, Link } from '@element-plus/icons-vue'` 行中追加 `Picture`：

```ts
import { UploadFilled, Link, Picture } from '@element-plus/icons-vue'
```

- [ ] **Step 5: 在 template 末尾、`</div>` 之前添加 el-dialog 弹窗**

在 `</template>` 之前添加：

```vue
<el-dialog
  v-model="galleryDialogVisible"
  title="从作品库选择图片"
  width="700px"
  :close-on-click-modal="false"
>
  <div v-loading="assetLoading" style="min-height: 200px">
    <div v-if="assetList.length === 0 && !assetLoading" style="text-align: center; padding: 40px; color: #909399">
      作品库暂无图片
    </div>
    <div v-else style="display: grid; grid-template-columns: repeat(auto-fill, minmax(140px, 1fr)); gap: 12px;">
      <div
        v-for="item in assetList"
        :key="item.id"
        style="cursor: pointer; border: 2px solid transparent; border-radius: 8px; overflow: hidden; transition: border-color 0.2s;"
        @click="selectAsset(item)"
      >
        <el-image
          :src="item.thumbnail || item.local_path || item.original_url"
          fit="cover"
          style="width: 100%; height: 140px; display: block;"
        />
        <div style="padding: 4px 6px; font-size: 12px; color: #666; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">
          {{ item.prompt || '无描述' }}
        </div>
      </div>
    </div>
  </div>
</el-dialog>
```

- [ ] **Step 6: 验证 TypeScript**

Run: `pnpm build` 或 `npx vue-tsc -b`
Expected: 无类型错误

- [ ] **Step 7: 提交**

```bash
git add frontend/src/views/ImageToImage.vue
git commit -m "feat: ImageToImage 新增从作品库选择图片 - 按钮弹出 el-dialog 展示资产缩略图, 选中后自动填充 URL"
```

---
---

## 功能测试清单

实现完成后手动验证以下场景：

- [ ] 图生图页面：点击「从作品库选择」→ 弹出对话框 → 显示图片列表 → 点击某图 → 对话框关闭，`imageUrl` 填充，预览区域显示图片
- [ ] 图生图页面：作品库为空时对话框显示「作品库暂无图片」
- [ ] 图生图页面：生成结果后，每个图片下方显示「继续精修」按钮
- [ ] 图生图页面：文生图模式下不显示「继续精修」按钮
- [ ] 点击「继续精修」→ 自动跳转到图生图页面 → prompt/imageUrl/inputMode 正确填充
- [ ] 精修后的图片通过「保存到作品库」正常保存
