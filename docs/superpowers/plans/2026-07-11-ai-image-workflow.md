# AI 图片工作流优化 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 以作品库为中心枢纽，构建职场人员高频 AI 生图 + 迭代的高效闭环

**Architecture:** 按三阶段增量建设 — ①自动入库 ②作品库增强（集合/精修入口/批量操作）③Prompt 模板与风格预设。前端 Vue 3 + Element Plus，后端 Go + Gin + GORM + SQLite。

**Tech Stack:** Go 1.25 · Gin · GORM · SQLite · Vue 3 · TypeScript 6 · Element Plus · Pinia

## Global Constraints

- 不改变现有导航结构（作品库不作为默认页）
- 所有新增 GORM 模型在 `backend/internal/repository/gorm/models.go` 中添加，并在 `OpenDB().AutoMigrate()` 注册
- 所有新增 API 路由在 `backend/cmd/server/main.go` 注册（`/api/v1/` 下）
- 前端使用 Composition API + `<script setup>`，TypeScript 6 `erasableSyntaxOnly`（无 enum）
- 后端错误返回中文：`gin.H{"error": "..."}`，日志前缀 `[Collections]`, `[Templates]`
- 所有代码注释使用中文（项目规范）
- 提交信息使用 semantic style（检测到仓库使用 `feat:` / `fix:` / `docs:` 前缀）
- 前端验证：`vue-tsc -b && vite build`；后端验证：`go vet ./... && go test ./...`

---

### Task 1: ImageResult 自动保存到作品库

**Files:**
- Modify: `frontend/src/components/ImageResult.vue`
- Modify: `frontend/src/api/assets.ts`（确认 saveAsset 签名）

**Interfaces:**
- Consumes: `saveAsset({ image_url, prompt, mode }) → Promise<{id}>` from `src/api/assets.ts`
- Produces: ImageResult 在结果图片展示时自动调用 saveAsset

**逻辑说明：**
当前 ImageResult 已有「保存到作品库」按钮（手动点击）。改为：图片加载时**自动静默保存**，不再依赖手动点击。保留按钮但改为「已保存」状态展示。设置中增加开关控制（默认开启），开关状态通过 `localStorage` 存储（后续可改为后端设置）。

- [ ] **Step 1: 确认 saveAsset API 签名**

Read `frontend/src/api/assets.ts` 确认 saveAsset 的请求体和返回格式：

```bash
cat frontend/src/api/assets.ts
```

预期看到:
```typescript
export async function saveAsset(data: { image_url: string; prompt: string; mode: string }): Promise<{ id: number }>
```

- [ ] **Step 2: 修改 ImageResult.vue — 增加自动保存逻辑**

在 `<script setup>` 中添加 autoSave 开关和自动保存逻辑：

```typescript
// 自动保存开关（默认开启，通过 localStorage 持久化）
const autoSaveEnabled = ref(localStorage.getItem('autoSaveToGallery') !== 'false')

// 已自动保存的图片 URL 集合（避免重复保存）
const autoSavedUrls = ref<Set<string>>(new Set())

// watch images 变化，自动保存新出现的图片
import { watch } from 'vue'

watch(() => props.images, (newImages) => {
  if (!autoSaveEnabled.value) return
  newImages.forEach(img => {
    if (!autoSavedUrls.value.has(img)) {
      autoSavedUrls.value = new Set([...autoSavedUrls.value, img])
      // 静默保存，不弹消息
      saveAsset({ image_url: img, prompt: props.prompt, mode: props.mode }).catch(() => {
        // 静默失败，不影响用户体验
      })
    }
  })
}, { immediate: true })
```

- [ ] **Step 3: 修改模板 — 自动保存时不显示「保存到作品库」按钮**

将「保存到作品库」按钮替换为 `已自动保存` 标签，仅在自动保存开启时显示：

```vue
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

- [ ] **Step 4: 验证构建**

```bash
cd frontend && pnpm build 2>&1 | tail -20
```
Expected: 构建成功，无 TS 或 vite 错误。

- [ ] **Step 5: 提交**

```bash
GIT_MASTER=1 git add frontend/src/components/ImageResult.vue
GIT_MASTER=1 git commit -m "feat: ImageResult 自动保存结果到作品库" -m "图片加载时静默调用 saveAsset，保留手动保存按钮作为降级。通过 localStorage autoSaveToGallery 控制开关。" -m "Ultraworked with [Sisyphus](https://github.com/code-yeongyu/oh-my-openagent)" -m "Co-authored-by: Sisyphus <clio-agent@sisyphuslabs.ai>"
```

---

### Task 2: 后端 — Collections（集合）GORM 模型 + CRUD API

**Files:**
- Create: `backend/internal/handler/collection.go`
- Modify: `backend/internal/repository/gorm/models.go` — 添加 Collection 和 AssetCollection 模型
- Modify: `backend/internal/repository/gorm/gorm.go` — AutoMigrate 新模型
- Create: `backend/internal/repository/gorm/collection.go` — CollectionRepository
- Modify: `backend/internal/repository/interfaces.go` — 添加接口定义
- Modify: `backend/cmd/server/main.go` — 注册路由

**Interfaces:**
- Consumes: `*gorm.DB` (injected via constructor)
- Produces: `CollectionRepository` with CRUD methods; `CollectionHandler` with Gin handlers

- [ ] **Step 1: 添加 GORM 模型**

在 `backend/internal/repository/gorm/models.go` 末尾添加：

```go
// Collection 集合（标签式多对多）
type Collection struct {
	ID        uint      `gorm:"primaryKey"`
	Name      string    `gorm:"not null;size:100"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Assets    []Asset   `gorm:"many2many:asset_collections;"`
}

// AssetCollection 关联表
type AssetCollection struct {
	AssetID      uint `gorm:"primaryKey"`
	CollectionID uint `gorm:"primaryKey"`
}
```

- [ ] **Step 2: 在 AutoMigrate 注册**

`backend/internal/repository/gorm/gorm.go` 的 `OpenDB()` 中添加：
```go
&Collection{}, &AssetCollection{},
```

- [ ] **Step 3: 编写 Repository 接口**

`backend/internal/repository/interfaces.go` 中添加：

```go
type CollectionRepository interface {
	List() ([]Collection, error)
	Create(name string) (*Collection, error)
	Update(id uint, name string) error
	Delete(id uint) error
	AddAssets(collectionID uint, assetIDs []uint) error
	RemoveAssets(collectionID uint, assetIDs []uint) error
}
```

- [ ] **Step 4: 编写 GORM 实现**

创建 `backend/internal/repository/gorm/collection.go`：

```go
package gorm

import (
	"time"
	"gorm.io/gorm"
)

type CollectionRepository struct {
	db *gorm.DB
}

func NewCollectionRepository(db *gorm.DB) *CollectionRepository {
	return &CollectionRepository{db: db}
}

func (r *CollectionRepository) List() ([]Collection, error) {
	var collections []Collection
	err := r.db.Preload("Assets").Find(&collections).Error
	return collections, err
}

func (r *CollectionRepository) Create(name string) (*Collection, error) {
	c := &Collection{Name: name}
	err := r.db.Create(c).Error
	return c, err
}

func (r *CollectionRepository) Update(id uint, name string) error {
	return r.db.Model(&Collection{}).Where("id = ?", id).Update("name", name).Error
}

func (r *CollectionRepository) Delete(id uint) error {
	// 删除关联 + 集合本身
	r.db.Where("collection_id = ?", id).Delete(&AssetCollection{})
	return r.db.Delete(&Collection{}, id).Error
}

func (r *CollectionRepository) AddAssets(collectionID uint, assetIDs []uint) error {
	for _, aid := range assetIDs {
		r.db.FirstOrCreate(&AssetCollection{AssetID: aid, CollectionID: collectionID})
	}
	return nil
}

func (r *CollectionRepository) RemoveAssets(collectionID uint, assetIDs []uint) error {
	return r.db.Where("collection_id = ? AND asset_id IN ?", collectionID, assetIDs).
		Delete(&AssetCollection{}).Error
}
```

- [ ] **Step 5: 编写 HTTP Handler**

创建 `backend/internal/handler/collection.go`：

```go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/agnes-image-tool/backend/internal/repository/gorm"
)

type CollectionHandler struct {
	repo *gorm.CollectionRepository
}

func NewCollectionHandler(repo *gorm.CollectionRepository) *CollectionHandler {
	return &CollectionHandler{repo: repo}
}

// ListCollections GET /api/v1/collections
func (h *CollectionHandler) ListCollections(c *gin.Context) {
	collections, err := h.repo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询集合失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, collections)
}

// CreateCollection POST /api/v1/collections
func (h *CollectionHandler) CreateCollection(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	collection, err := h.repo.Create(req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建集合失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, collection)
}

// UpdateCollection PUT /api/v1/collections/:id
func (h *CollectionHandler) UpdateCollection(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	if err := h.repo.Update(uint(id), req.Name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新集合失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// DeleteCollection DELETE /api/v1/collections/:id
func (h *CollectionHandler) DeleteCollection(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.repo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除集合失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// AddAssetsToCollection POST /api/v1/collections/:id/assets
func (h *CollectionHandler) AddAssetsToCollection(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		AssetIDs []uint `json:"asset_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	if err := h.repo.AddAssets(uint(id), req.AssetIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "添加资产到集合失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "添加成功"})
}

// RemoveAssetsFromCollection DELETE /api/v1/collections/:id/assets
func (h *CollectionHandler) RemoveAssetsFromCollection(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		AssetIDs []uint `json:"asset_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	if err := h.repo.RemoveAssets(uint(id), req.AssetIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "从集合移除资产失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "移除成功"})
}
```

- [ ] **Step 6: 注册路由**

在 `backend/cmd/server/main.go` 中添加：

```go
// 初始化
collectionRepo := gormrepo.NewCollectionRepository(db)
collectionHandler := handler.NewCollectionHandler(collectionRepo)

// 路由注册
api := r.Group("/api/v1")
{
    // ... existing routes ...
    
    // Collections
    api.GET("/collections", collectionHandler.ListCollections)
    api.POST("/collections", collectionHandler.CreateCollection)
    api.PUT("/collections/:id", collectionHandler.UpdateCollection)
    api.DELETE("/collections/:id", collectionHandler.DeleteCollection)
    api.POST("/collections/:id/assets", collectionHandler.AddAssetsToCollection)
    api.DELETE("/collections/:id/assets", collectionHandler.RemoveAssetsFromCollection)
}
```

- [ ] **Step 7: 后端验证**

```bash
cd backend && go vet ./... && go build ./cmd/server
```
Expected: 无错误，编译成功。

- [ ] **Step 8: 提交**

```bash
GIT_MASTER=1 git add backend/internal/repository/gorm/models.go backend/internal/repository/gorm/gorm.go backend/internal/repository/gorm/collection.go backend/internal/repository/interfaces.go backend/internal/handler/collection.go backend/cmd/server/main.go
GIT_MASTER=1 git commit -m "feat: 作品库集合功能 — Collections CRUD API + GORM 模型" -m "新增 Collection/AssetCollection 模型、CollectionRepository、CollectionHandler。支持：集合 CRUD、添加/移除资产。" -m "Ultraworked with [Sisyphus](https://github.com/code-yeongyu/oh-my-openagent)" -m "Co-authored-by: Sisyphus <clio-agent@sisyphuslabs.ai>"
```

---

### Task 3: 前端 — 集合管理 UI

**Files:**
- Create: `frontend/src/api/collections.ts` — 集合 API 客户端
- Modify: `frontend/src/views/Assets.vue` — 添加集合筛选、创建集合弹窗、多选操作栏

**Interfaces:**
- Consumes: `CollectionRepository` API from backend (Task 2)
- Produces: 集合选择器组件、集合管理弹窗、筛选功能

- [ ] **Step 1: 创建集合 API 客户端**

`frontend/src/api/collections.ts`：

```typescript
import client from './client'

export interface Collection {
  id: number
  name: string
  created_at: string
  updated_at: string
  assets?: { id: number }[]
}

export async function getCollections(): Promise<Collection[]> {
  const res = await client.get('/api/v1/collections')
  return res.data
}

export async function createCollection(name: string): Promise<Collection> {
  const res = await client.post('/api/v1/collections', { name })
  return res.data
}

export async function updateCollection(id: number, name: string): Promise<void> {
  await client.put(`/api/v1/collections/${id}`, { name })
}

export async function deleteCollection(id: number): Promise<void> {
  await client.delete(`/api/v1/collections/${id}`)
}

export async function addAssetsToCollection(collectionId: number, assetIds: number[]): Promise<void> {
  await client.post(`/api/v1/collections/${collectionId}/assets`, { asset_ids: assetIds })
}

export async function removeAssetsFromCollection(collectionId: number, assetIds: number[]): Promise<void> {
  await client.delete(`/api/v1/collections/${collectionId}/assets`, {
    data: { asset_ids: assetIds }
  })
}
```

- [ ] **Step 2: 修改 Assets.vue — 添加集合筛选栏**

在 Assets.vue 的 `script setup` 中添加：

```typescript
import { ref, onMounted, computed } from 'vue'
import { getCollections, createCollection, deleteCollection, addAssetsToCollection, removeAssetsFromCollection, type Collection } from '../api/collections'

const collections = ref<Collection[]>([])
const selectedCollectionId = ref<number | undefined>(undefined)
const showCreateCollectionDialog = ref(false)
const newCollectionName = ref('')
```

在模板的搜索栏下方添加集合标签行：

```vue
<div style="margin-bottom: 16px; display: flex; align-items: center; gap: 8px; flex-wrap: wrap;">
  <el-tag
    :type="selectedCollectionId === undefined ? 'primary' : 'info'"
    style="cursor: pointer"
    @click="selectedCollectionId = undefined"
  >
    全部
  </el-tag>
  <el-tag
    v-for="c in collections"
    :key="c.id"
    :type="selectedCollectionId === c.id ? 'primary' : 'info'"
    style="cursor: pointer"
    @click="selectedCollectionId = c.id"
  >
    {{ c.name }}
  </el-tag>
  <el-button size="small" type="default" @click="showCreateCollectionDialog = true">
    + 新建集合
  </el-button>
</div>
```

- [ ] **Step 3: 添加创建集合弹窗**

```vue
<el-dialog v-model="showCreateCollectionDialog" title="新建集合" width="400px">
  <el-input v-model="newCollectionName" placeholder="输入集合名称" />
  <template #footer>
    <el-button @click="showCreateCollectionDialog = false">取消</el-button>
    <el-button type="primary" @click="handleCreateCollection">确定</el-button>
  </template>
</el-dialog>
```

```typescript
async function handleCreateCollection() {
  if (!newCollectionName.value.trim()) return
  await createCollection(newCollectionName.value.trim())
  newCollectionName.value = ''
  showCreateCollectionDialog.value = false
  await loadCollections()
}

async function loadCollections() {
  collections.value = await getCollections()
}

onMounted(() => {
  loadCollections()
})
```

- [ ] **Step 4: 筛选逻辑 — 调用 API 时带上集合过滤**

修改 `loadAssets` 函数，当 `selectedCollectionId` 不为空时传递给后端。注意：后端 ListAssets 当前没有按集合过滤参数，需要在 asset.go 的 `ListAssets` 中添加 `collection_id` 查询参数支持。

在 `backend/internal/handler/asset.go` 的 `ListAssets` 方法中添加：
```go
collectionID := c.Query("collection_id")
```

并在调用 `h.repo.List()` 时传递。同时修改 repository 接口和实现。

**简化方案：** 如果集合过滤逻辑复杂，改为前端过滤。加载所有 assets 后在内存中过滤。但更好的方案是后端支持。

采用前端过滤（更简单，作品库数据量通常不大）：

```typescript
const filteredAssets = computed(() => {
  if (!selectedCollectionId.value) return assets.value
  // 需要后端返回每个 asset 所属的 collection_ids
  // 或者前端维护一个 asset→collections 的映射
})
```

**更优的简化方案：** 在 Assets.vue 中，加载集合时同时加载集合的资产 ID 列表，前端做交集过滤：

```typescript
// 每个集合对象已通过 Preload 包含 Assets
const collectionAssetIds = computed(() => {
  if (!selectedCollectionId.value) return null
  const c = collections.value.find(c => c.id === selectedCollectionId.value)
  return new Set(c?.assets?.map(a => a.id) || [])
})

const displayAssets = computed(() => {
  if (!collectionAssetIds.value) return assetItems.value
  return assetItems.value.filter(item => collectionAssetIds.value!.has(item.id))
})
```

在模板中将 `assetItems` 替换为 `displayAssets`。

- [ ] **Step 5: 验证前端构建**

```bash
cd frontend && pnpm build 2>&1 | tail -20
```
Expected: 构建成功。

- [ ] **Step 6: 提交**

```bash
GIT_MASTER=1 git add frontend/src/api/collections.ts frontend/src/views/Assets.vue
GIT_MASTER=1 git commit -m "feat: 作品库集合管理 UI — 创建/筛选集合" -m "集合标签栏筛选、新建集合弹窗。前端按集合资产 ID 做交集过滤。" -m "Ultraworked with [Sisyphus](https://github.com/code-yeongyu/oh-my-openagent)" -m "Co-authored-by: Sisyphus <clio-agent@sisyphuslabs.ai>"
```

---

### Task 4: 作品库 — 精修按钮 + 多选操作栏

**Files:**
- Modify: `frontend/src/views/Assets.vue` — 添加「精修」按钮接入 redo store、多选操作栏

**Interfaces:**
- Consumes: `useRedoStore()` from `src/stores/redo.ts`
- Produces: 作品库中每张图片可一键精修；多选后批量操作

- [ ] **Step 1: 在图片卡片上添加「精修」按钮**

在当前 Assets.vue 的卡片操作区（收藏按钮旁）添加：

```vue
<el-button
  size="small"
  type="primary"
  @click="handleRefine(item)"
>
  精修
</el-button>
```

```typescript
import { useRedoStore } from '../stores/redo'
const redoStore = useRedoStore()

function handleRefine(item: AssetItem) {
  // 使用已有的 assetSrc 函数按 local_path > original_url > github_url 优先级取 URL
  const url = assetSrc(item)
  if (!url) {
    ElMessage.warning('该资产没有可用的图片 URL')
    return
  }
  redoStore.setRedoData({
    mode: 'image2image',
    imageUrl: url,
    prompt: item.prompt || '',
    inputMode: 'url',
  })
}
```

注意：需要将 `AssetItem` 类型导入（当前可能已在 Assets.vue 中使用）。需要确认 `url` 字段名：查看 `AssetItem` 接口：

检查 `frontend/src/types/index.ts` 中 `AssetItem` 的定义，使用正确的字段名传递 imageUrl。

- [ ] **Step 2: 添加多选模式**

Assets.vue 中添加多选状态：

```typescript
const selectedIds = ref<Set<number>>(new Set())
const isMultiSelect = computed(() => selectedIds.value.size > 0)

function toggleSelect(id: number) {
  const next = new Set(selectedIds.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selectedIds.value = next
}
```

在每个图片卡片左上角添加复选框：

```vue
<el-checkbox
  v-model="item._selected"
  @change="toggleSelect(item.id)"
  style="position: absolute; top: 8px; left: 8px; z-index: 1;"
  @click.stop
/>
```

- [ ] **Step 3: 多选底部操作栏**

```vue
<el-affix v-if="isMultiSelect" position="bottom" :offset="0">
  <div style="background: #fff; padding: 12px 24px; border-top: 1px solid #e0e0e0; display: flex; align-items: center; gap: 16px; justify-content: center;">
    <span>已选择 {{ selectedIds.size }} 项</span>
    <el-button type="primary" @click="handleBatchRefine">批量精修</el-button>
    <el-button @click="handleBatchDownload">批量下载</el-button>
    <el-button @click="handleBatchMoveToCollection">移至集合</el-button>
    <el-button type="danger" @click="handleBatchDelete">批量删除</el-button>
    <el-button @click="selectedIds = new Set()">取消选择</el-button>
  </div>
</el-affix>
```

- [ ] **Step 4: 实现批量操作**

```typescript
function handleBatchRefine() {
  // 取第一张图精修（批量精修走批量生成页）
  const first = assetItems.value.find(item => selectedIds.value.has(item.id))
  if (!first) return
  const url = assetSrc(first)
  if (!url) return
  redoStore.setRedoData({
    mode: 'batch',
    imageUrl: url,
    prompt: first.prompt || '',
    inputMode: 'url',
  })
  selectedIds.value = new Set()
}

async function handleBatchDownload() {
  const ids = Array.from(selectedIds.value)
  // 调用现有 batchDownload API
  window.open('/api/v1/assets/batch-download?ids=' + ids.join(','), '_blank')
  selectedIds.value = new Set()
}

function handleBatchMoveToCollection() {
  // 打开集合选择弹窗
  showCollectionPicker.value = true
}

async function handleBatchDelete() {
  await ElMessageBox.confirm(`确定删除选中的 ${selectedIds.value.size} 项？`, '提示')
  // 调用现有 DeleteAssets API
  selectedIds.value = new Set()
}
```

- [ ] **Step 5: 「移至集合」选择弹窗**

```vue
<el-dialog v-model="showCollectionPicker" title="选择集合" width="400px">
  <el-radio-group v-model="targetCollectionId" direction="vertical" style="width: 100%;">
    <el-radio v-for="c in collections" :key="c.id" :value="c.id" style="margin-bottom: 8px;">
      {{ c.name }}
    </el-radio>
  </el-radio-group>
  <template #footer>
    <el-button @click="showCollectionPicker = false">取消</el-button>
    <el-button type="primary" @click="handleConfirmMoveToCollection">确定</el-button>
  </template>
</el-dialog>
```

- [ ] **Step 6: 验证构建**

```bash
cd frontend && pnpm build 2>&1 | tail -20
```
Expected: 构建成功。

- [ ] **Step 7: 提交**

```bash
GIT_MASTER=1 git add frontend/src/views/Assets.vue
GIT_MASTER=1 git commit -m "feat: 作品库精修入口 + 多选操作栏" -m "每张图片卡片新增精修按钮（redo store 跳转图生图），多选模式支持批量精修/下载/移动/删除。" -m "Ultraworked with [Sisyphus](https://github.com/code-yeongyu/oh-my-openagent)" -m "Co-authored-by: Sisyphus <clio-agent@sisyphuslabs.ai>"
```

---

### Task 5: 后端 — PromptTemplate GORM 模型 + CRUD + 导入导出 API

**Files:**
- Modify: `backend/internal/repository/gorm/models.go` — 添加 PromptTemplate 模型
- Modify: `backend/internal/repository/gorm/gorm.go` — AutoMigrate
- Create: `backend/internal/repository/gorm/template.go` — TemplateRepository
- Modify: `backend/internal/repository/interfaces.go` — 添加接口
- Create: `backend/internal/handler/template.go` — TemplateHandler
- Modify: `backend/cmd/server/main.go` — 注册路由

- [ ] **Step 1: 添加 GORM 模型**

在 `backend/internal/repository/gorm/models.go` 中添加：

```go
// PromptTemplate Prompt 模板/风格预设
type PromptTemplate struct {
	ID             uint      `gorm:"primaryKey"`
	Name           string    `gorm:"not null;size:200"`
	Type           string    `gorm:"default:template;size:20"` // template | preset
	Category       string    `gorm:"size:50"`                   // 人物/产品/背景/封面/海报/社媒/自定义
	Prompt         string    `gorm:"type:text"`
	NegativePrompt string    `gorm:"type:text"`
	Size           string    `gorm:"size:20"`
	Strength       float64   `gorm:"default:0.75"`
	Model          string    `gorm:"size:100"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
```

在 `OpenDB().AutoMigrate()` 中添加 `&PromptTemplate{}`。

- [ ] **Step 2: 编写 Repository**

`backend/internal/repository/gorm/template.go`:

```go
package gorm

import (
	"gorm.io/gorm"
)

type TemplateRepository struct {
	db *gorm.DB
}

func NewTemplateRepository(db *gorm.DB) *TemplateRepository {
	return &TemplateRepository{db: db}
}

func (r *TemplateRepository) List(category string) ([]PromptTemplate, error) {
	query := r.db.Order("updated_at desc")
	if category != "" {
		query = query.Where("category = ?", category)
	}
	var templates []PromptTemplate
	err := query.Find(&templates).Error
	return templates, err
}

func (r *TemplateRepository) GetByID(id uint) (*PromptTemplate, error) {
	var t PromptTemplate
	err := r.db.First(&t, id).Error
	return &t, err
}

func (r *TemplateRepository) Create(t *PromptTemplate) error {
	return r.db.Create(t).Error
}

func (r *TemplateRepository) Update(t *PromptTemplate) error {
	return r.db.Model(&PromptTemplate{}).Where("id = ?", t.ID).Updates(map[string]interface{}{
		"name": t.Name, "type": t.Type, "category": t.Category,
		"prompt": t.Prompt, "negative_prompt": t.NegativePrompt,
		"size": t.Size, "strength": t.Strength, "model": t.Model,
	}).Error
}

func (r *TemplateRepository) Delete(id uint) error {
	return r.db.Delete(&PromptTemplate{}, id).Error
}

func (r *TemplateRepository) Export() ([]PromptTemplate, error) {
	var templates []PromptTemplate
	err := r.db.Find(&templates).Error
	return templates, err
}

func (r *TemplateRepository) Import(templates []PromptTemplate) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i := range templates {
			templates[i].ID = 0 // 重置 ID 避免冲突
			if err := tx.Create(&templates[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
```

- [ ] **Step 3: 编写 HTTP Handler**

`backend/internal/handler/template.go` — 类似 collection.go 的 CRUD 模式，额外增加导出/导入端点。

- [ ] **Step 4: 注册路由**

```go
templateRepo := gormrepo.NewTemplateRepository(db)
templateHandler := handler.NewTemplateHandler(templateRepo)

api.GET("/templates", templateHandler.ListTemplates)
api.POST("/templates", templateHandler.CreateTemplate)
api.PUT("/templates/:id", templateHandler.UpdateTemplate)
api.DELETE("/templates/:id", templateHandler.DeleteTemplate)
api.POST("/templates/export", templateHandler.ExportTemplates)
api.POST("/templates/import", templateHandler.ImportTemplates)
api.POST("/history/:id/save-template", templateHandler.SaveFromHistory)
```

- [ ] **Step 5: 后端验证**

```bash
cd backend && go vet ./... && go build ./cmd/server
```
Expected: 无错误。

- [ ] **Step 6: 提交**

```bash
GIT_MASTER=1 git add backend/internal/repository/gorm/models.go backend/internal/repository/gorm/gorm.go backend/internal/repository/gorm/template.go backend/internal/repository/interfaces.go backend/internal/handler/template.go backend/cmd/server/main.go
GIT_MASTER=1 git commit -m "feat: Prompt 模板与风格预设 — CRUD + 导入导出 API" -m "新增 PromptTemplate 模型（template/preset 双类型）、TemplateRepository、TemplateHandler。支持分类过滤、JSON 导入导出、从历史记录保存为模板。" -m "Ultraworked with [Sisyphus](https://github.com/code-yeongyu/oh-my-openagent)" -m "Co-authored-by: Sisyphus <clio-agent@sisyphuslabs.ai>"
```

---

### Task 6: 前端 — Prompt 模板管理页面 + 生成页模板选择器

**Files:**
- Create: `frontend/src/api/templates.ts`
- Modify: `frontend/src/views/ImageToImage.vue` — 模板选择器 + 风格预设下拉
- Modify: `frontend/src/views/TextToImage.vue` — 模板选择器 + 风格预设下拉
- Modify: `frontend/src/App.vue` — 添加「模板管理」页面路由

**Interfaces:**
- Consumes: Template API from backend (Task 5)
- Produces: 生成页可一键选择模板/预设

- [ ] **Step 1: 创建模板 API 客户端**

`frontend/src/api/templates.ts`:

```typescript
import client from './client'

export interface PromptTemplate {
  id: number
  name: string
  type: 'template' | 'preset'
  category: string
  prompt: string
  negative_prompt: string
  size: string
  strength: number
  model: string
  created_at: string
  updated_at: string
}

export async function getTemplates(category?: string): Promise<PromptTemplate[]> {
  const params = category ? { category } : {}
  const res = await client.get('/api/v1/templates', { params })
  return res.data
}

export async function createTemplate(data: Partial<PromptTemplate>): Promise<PromptTemplate> {
  const res = await client.post('/api/v1/templates', data)
  return res.data
}

export async function updateTemplate(id: number, data: Partial<PromptTemplate>): Promise<void> {
  await client.put(`/api/v1/templates/${id}`, data)
}

export async function deleteTemplate(id: number): Promise<void> {
  await client.delete(`/api/v1/templates/${id}`)
}

export async function exportTemplates(): Promise<PromptTemplate[]> {
  const res = await client.post('/api/v1/templates/export')
  return res.data
}

export async function importTemplates(data: PromptTemplate[]): Promise<void> {
  await client.post('/api/v1/templates/import', data)
}
```

- [ ] **Step 2: 添加「模板管理」页面**

新增 `frontend/src/views/TemplateManager.vue` — 表格展示所有模板，支持 CRUD 操作：创建、编辑、删除、导入/导出 JSON。

在 `App.vue` 中添加 `activePage === 'templates'` 分支路由。

- [ ] **Step 3: 生成页添加模板选择器**

在 `ImageToImage.vue` 和 `TextToImage.vue` 的 prompt 输入框旁添加「从模板」按钮：

```vue
<el-button size="small" @click="showTemplatePicker = true">从模板</el-button>

<el-dialog v-model="showTemplatePicker" title="选择 Prompt 模板" width="600px">
  <el-table :data="templates" style="width: 100%" @row-click="applyTemplate">
    <el-table-column prop="name" label="名称" width="150" />
    <el-table-column prop="category" label="分类" width="100" />
    <el-table-column prop="prompt" label="Prompt" show-overflow-tooltip />
    <el-table-column prop="size" label="尺寸" width="80" />
  </el-table>
</el-dialog>
```

- [ ] **Step 4: 参数区添加风格预设下拉**

在生成页的参数区（size、strength 等）添加风格预设下拉：

```vue
<el-select v-model="selectedPreset" placeholder="风格预设" @change="applyPreset" clearable>
  <el-option
    v-for="p in presets"
    :key="p.id"
    :label="p.name"
    :value="p.id"
  />
</el-select>
```

```typescript
const presets = ref<PromptTemplate[]>([])

async function loadPresets() {
  const all = await getTemplates()
  presets.value = all.filter(t => t.type === 'preset')
}

function applyPreset(presetId: number) {
  const preset = presets.value.find(p => p.id === presetId)
  if (!preset) return
  if (preset.size) size.value = preset.size
  if (preset.strength) strength.value = preset.strength
  if (preset.negative_prompt) negativePrompt.value = preset.negative_prompt
  if (preset.model) model.value = preset.model
}
```

- [ ] **Step 5: 验证构建**

```bash
cd frontend && pnpm build 2>&1 | tail -20
```
Expected: 构建成功。

- [ ] **Step 6: 提交**

```bash
GIT_MASTER=1 git add frontend/src/api/templates.ts frontend/src/views/TemplateManager.vue frontend/src/views/ImageToImage.vue frontend/src/views/TextToImage.vue frontend/src/App.vue
GIT_MASTER=1 git commit -m "feat: Prompt 模板管理页面 + 生成页模板/预设选择器" -m "新增 TemplateManager 页面管理模板 CRUD，生成页 prompt 旁「从模板」按钮，参数区「风格预设」下拉。" -m "Ultraworked with [Sisyphus](https://github.com/code-yeongyu/oh-my-openagent)" -m "Co-authored-by: Sisyphus <clio-agent@sisyphuslabs.ai>"
```

---

### Task 7: Prompt 历史侧栏（生成页）

**Files:**
- Modify: `frontend/src/views/ImageToImage.vue` — prompt 输入框旁添加历史侧栏
- Modify: `frontend/src/views/TextToImage.vue` — prompt 输入框旁添加历史侧栏

**Interfaces:**
- Consumes: `getHistory()` from `src/api/history.ts`

- [ ] **Step 1: 添加 Prompt 历史侧栏**

在 prompt 输入框旁添加一个展开/收起的历史面板：

```vue
<el-button size="small" @click="showHistory = !showHistory">
  {{ showHistory ? '收起历史' : '历史 Prompt' }}
</el-button>

<el-collapse-transition>
  <div v-if="showHistory" style="margin-top: 8px; max-height: 300px; overflow-y: auto; border: 1px solid #e0e0e0; border-radius: 4px; padding: 8px;">
    <div
      v-for="h in historyList"
      :key="h.id"
      style="padding: 6px 8px; cursor: pointer; border-bottom: 1px solid #f0f0f0;"
      @click="applyHistory(h)"
    >
      <div style="font-size: 12px; color: #999;">{{ h.time }}</div>
      <div style="font-size: 13px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">{{ h.prompt }}</div>
    </div>
    <div v-if="historyList.length === 0" style="color: #999; text-align: center; padding: 16px;">
      暂无历史记录
    </div>
  </div>
</el-collapse-transition>
```

- [ ] **Step 2: 加载和选择历史**

```typescript
import { getHistory } from '../api/history'

const showHistory = ref(false)
const historyList = ref<any[]>([])

async function loadHistory() {
  try {
    const res = await getHistory({ page: 1, per_page: 20 })
    historyList.value = res.items || []
  } catch { /* 静默 */ }
}

function applyHistory(h: any) {
  prompt.value = h.prompt
  showHistory.value = false
}
```

- [ ] **Step 3: 验证构建**

```bash
cd frontend && pnpm build 2>&1 | tail -20
```
Expected: 构建成功。

- [ ] **Step 4: 提交**

```bash
GIT_MASTER=1 git add frontend/src/views/ImageToImage.vue frontend/src/views/TextToImage.vue
GIT_MASTER=1 git commit -m "feat: 生成页 Prompt 历史侧栏" -m "在 prompt 输入框旁添加可展开/收起的历史记录面板，点击历史 prompt 自动填入。" -m "Ultraworked with [Sisyphus](https://github.com/code-yeongyu/oh-my-openagent)" -m "Co-authored-by: Sisyphus <clio-agent@sisyphuslabs.ai>"
```
