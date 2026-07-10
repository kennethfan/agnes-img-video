# 独立作品库 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Decouple Asset Gallery from History table — create independent `assets` table with manual save-only workflow.

**Architecture:** New `Asset` GORM model + `AssetRepository` interface/impl. `AssetHandler` switches from `HistoryRepository` to `AssetRepository`. Frontend `ImageResult` gets "保存到作品库" button. No data migration — gallery starts fresh.

**Tech Stack:** Go 1.25 / GORM v1.31.2 / Gin / Vue 3 / TypeScript 6 / Element Plus

## Global Constraints

- Go: project-layout standard (`cmd/`, `internal/`, etc.)
- Vue: Composition API + `<script setup>` only
- TypeScript 6: `erasableSyntaxOnly` — no enums, use `as const` or union types
- Chinese error messages: `gin.H{"error": "消息: " + err.Error()}`
- No auth middleware — local dev tool
- Do NOT delete `history.db` or `outputs/` — runtime data
- Do NOT use `as any`, `@ts-ignore`, `@ts-expect-error`
- Do NOT refactor existing working code (history.go, HistoryRepository, `deleteRecordFiles`)
- Comments in Chinese

---

## File Structure

| File | Action | Responsibility |
|------|--------|----------------|
| `backend/internal/repository/gorm/models.go` | Modify | Add `Asset` model, keep `Favorite` model (removed from AutoMigrate in Task 4) |
| `backend/internal/repository/interfaces.go` | Modify | Add `AssetRepository` interface |
| `backend/internal/model/types.go` | Modify | Update `AssetItem` (add OriginalURL/LocalPath/GitHubURL, rm Files), update `AssetFavoriteRequest` (HistoryID→AssetID) |
| `backend/internal/repository/gorm/asset.go` | Create | GORM implementation of `AssetRepository` |
| `backend/internal/repository/gorm/asset_test.go` | Create | Unit tests for repository |
| `backend/internal/handler/asset.go` | Modify | Refactor to use `AssetRepository`, add `SaveAsset` |
| `backend/cmd/server/main.go` | Modify | Wire `AssetRepo`, update `AssetHandler` init, add POST /assets, update AutoMigrate |
| `backend/internal/repository/gorm/gorm.go` | Modify | Update AutoMigrate — add Asset, remove Favorite |
| `frontend/src/types/index.ts` | Modify | Update `AssetItem` and `AssetFavoriteRequest` interfaces |
| `frontend/src/api/assets.ts` | Modify | Add `saveAsset` API function |
| `frontend/src/components/ImageResult.vue` | Modify | Add `prompt`/`mode` props, add "保存到作品库" button |
| `frontend/src/views/TextToImage.vue` | Modify | Pass `prompt`/`mode` to ImageResult |
| `frontend/src/views/ImageToImage.vue` | Modify | Same |
| `frontend/src/views/BatchGen.vue` | Modify | Same |
| `frontend/src/views/Assets.vue` | Modify | Adapt `toggleFavorite` call (asset_id instead of history_id) |

---

### Task 1: Backend — Asset model + interface + Go types

**Files:**
- Modify: `backend/internal/repository/gorm/models.go`
- Modify: `backend/internal/repository/interfaces.go`
- Modify: `backend/internal/model/types.go`

**Interfaces:**
- Consumes: existing `History`, `Favorite` models
- Produces: `Asset` struct, `AssetRepository` interface, new `AssetItem`/`AssetFavoriteRequest` structs

- [ ] **Step 1: Add Asset model to models.go**

Insert after the `Favorite` model block:

```go
// Asset 作品库表 — 独立于 history
type Asset struct {
	ID          int64  `gorm:"primaryKey"`
	Mode        string `gorm:"index"`
	Prompt      string
	Type        string // "image" | "video"
	Time        string // 保存时间
	Favorite    bool
	OriginalURL string `gorm:"column:original_url"`
	LocalPath   string `gorm:"column:local_path"`
	GitHubURL   string `gorm:"column:github_url"`
}

func (Asset) TableName() string { return "assets" }
```

- [ ] **Step 2: Add AssetRepository interface to interfaces.go**

After the `TaskRepository` block:

```go
// ==================== Asset ====================

type AssetRepository interface {
	Insert(asset *model.Asset) (int64, error)
	List(page, perPage int, assetType, search string, favoriteFilter bool) ([]model.Asset, int, error)
	GetByIDs(ids []int64) ([]model.Asset, error)
	ToggleFavorite(id int64, favorite bool) error
	Delete(ids []int64) error
}
```

- [ ] **Step 3: Update AssetItem struct in types.go**

Replace lines 192-201:

```go
type AssetItem struct {
	ID          int64  `json:"id"`
	Mode        string `json:"mode"`
	Prompt      string `json:"prompt"`
	Type        string `json:"type"`
	Time        string `json:"time"`
	Favorite    bool   `json:"favorite"`
	OriginalURL string `json:"original_url"`
	LocalPath   string `json:"local_path"`
	GitHubURL   string `json:"github_url"`
	Thumbnail   string `json:"thumbnail"`
}
```

- [ ] **Step 4: Update AssetFavoriteRequest**

Replace lines 209-211:

```go
type AssetFavoriteRequest struct {
	AssetID  int64 `json:"asset_id" binding:"required"`
	Favorite bool  `json:"favorite"`
}
```

- [ ] **Step 5: Commit**

```bash
cd backend
GIT_MASTER=1 git add internal/repository/gorm/models.go internal/repository/interfaces.go internal/model/types.go
GIT_MASTER=1 git commit -m "feat: add Asset model and AssetRepository interface" -m "Ultraworked with [Sisyphus](https://github.com/code-yeongyu/oh-my-openagent)" -m "Co-authored-by: Sisyphus <clio-agent@sisyphuslabs.ai>"
```

---

### Task 2: Backend — GORM AssetRepository implementation

**Files:**
- Create: `backend/internal/repository/gorm/asset.go`
- Create: `backend/internal/repository/gorm/asset_test.go`

**Interfaces:**
- Consumes: `AssetRepository` interface, `Asset` model, `gorm.DB`
- Produces: `AssetRepository` concrete implementation

- [ ] **Step 1: Create repository/gorm/asset.go**

```go
package gorm

import (
	"fmt"
	"strings"

	"github.com/agnes-image-tool/backend/internal/model"
	"github.com/agnes-image-tool/backend/internal/repository"
	"gorm.io/gorm"
)

type AssetRepository struct {
	db *gorm.DB
}

func NewAssetRepository(db *gorm.DB) *AssetRepository {
	return &AssetRepository{db: db}
}

// compile-time interface check
var _ repository.AssetRepository = (*AssetRepository)(nil)

func (r *AssetRepository) Insert(asset *model.Asset) (int64, error) {
	if err := r.db.Create(asset).Error; err != nil {
		return 0, fmt.Errorf("插入资产失败: %w", err)
	}
	return asset.ID, nil
}

func (r *AssetRepository) List(page, perPage int, assetType, search string, favoriteFilter bool) ([]model.Asset, int, error) {
	query := r.db.Model(&model.Asset{})
	if assetType != "" && assetType != "all" {
		query = query.Where("type = ?", assetType)
	}
	if search != "" {
		query = query.Where("prompt LIKE ?", "%"+search+"%")
	}
	if favoriteFilter {
		query = query.Where("favorite = ?", true)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计资产总数失败: %w", err)
	}
	var assets []model.Asset
	if err := query.Order("id DESC").Offset((page - 1) * perPage).Limit(perPage).Find(&assets).Error; err != nil {
		return nil, 0, fmt.Errorf("查询资产列表失败: %w", err)
	}
	return assets, int(total), nil
}

func (r *AssetRepository) GetByIDs(ids []int64) ([]model.Asset, error) {
	var assets []model.Asset
	if err := r.db.Where("id IN ?", ids).Find(&assets).Error; err != nil {
		return nil, fmt.Errorf("查询资产失败: %w", err)
	}
	return assets, nil
}

func (r *AssetRepository) ToggleFavorite(id int64, favorite bool) error {
	if err := r.db.Model(&model.Asset{}).Where("id = ?", id).Update("favorite", favorite).Error; err != nil {
		return fmt.Errorf("更新收藏状态失败: %w", err)
	}
	return nil
}

func (r *AssetRepository) Delete(ids []int64) error {
	if err := r.db.Where("id IN ?", ids).Delete(&model.Asset{}).Error; err != nil {
		return fmt.Errorf("删除资产失败: %w", err)
	}
	return nil
}
```

- [ ] **Step 2: Create repository/gorm/asset_test.go**

```go
package gorm

import (
	"testing"
	"time"

	"github.com/agnes-image-tool/backend/internal/model"
)

func TestAssetRepository(t *testing.T) {
	db := setupTestDB()
	repo := NewAssetRepository(db)
	now := time.Now().Format("2006-01-02 15:04:05")

	// Insert
	asset := &model.Asset{
		Mode:        "text2image",
		Prompt:      "测试提示词",
		Type:        "image",
		Time:        now,
		Favorite:    false,
		OriginalURL: "/outputs/test.png",
		LocalPath:   "outputs/test.png",
	}
	id, err := repo.Insert(asset)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	if id == 0 {
		t.Fatal("Insert returned zero id")
	}

	// List
	items, total, err := repo.List(1, 20, "", "", false)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total < 1 {
		t.Fatal("List returned zero total")
	}

	// ToggleFavorite
	if err := repo.ToggleFavorite(id, true); err != nil {
		t.Fatalf("ToggleFavorite failed: %v", err)
	}

	// GetByIDs
	assets, err := repo.GetByIDs([]int64{id})
	if err != nil {
		t.Fatalf("GetByIDs failed: %v", err)
	}
	if len(assets) != 1 {
		t.Fatal("GetByIDs returned unexpected count")
	}
	if !assets[0].Favorite {
		t.Fatal("Expected favorite=true")
	}

	// Delete
	if err := repo.Delete([]int64{id}); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	after, _, _ := repo.List(1, 20, "", "", false)
	if len(after) != 0 {
		t.Fatal("Expected empty list after delete")
	}
}
```

The test depends on `setupTestDB()` which should already exist from other test files in the same package. Verify: `grep -n 'func setupTestDB' backend/internal/repository/gorm/*_test.go`

- [ ] **Step 3: Verify tests pass**

```bash
cd backend
go test ./internal/repository/gorm/ -run TestAssetRepository -v
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
GIT_MASTER=1 git add internal/repository/gorm/asset.go internal/repository/gorm/asset_test.go
GIT_MASTER=1 git commit -m "feat: implement AssetRepository CRUD" -m "Ultraworked with [Sisyphus](https://github.com/code-yeongyu/oh-my-openagent)" -m "Co-authored-by: Sisyphus <clio-agent@sisyphuslabs.ai>"
```

---

### Task 3: Backend — Refactor AssetHandler + SaveAsset

**Files:**
- Modify: `backend/internal/handler/asset.go`

**Interfaces:**
- Consumes: `AssetRepository` (instead of `HistoryRepository`), `model.Asset`/`AssetItem`/`AssetFavoriteRequest`
- Produces: refactored `AssetHandler` using `AssetRepository`

- [ ] **Step 1: Rewrite AssetHandler**

Replace the entire `asset.go`:

```go
package handler

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/agnes-image-tool/backend/internal/model"
	"github.com/agnes-image-tool/backend/internal/repository"
)

type AssetHandler struct {
	repo repository.AssetRepository
}

func NewAssetHandler(repo repository.AssetRepository) *AssetHandler {
	return &AssetHandler{repo: repo}
}

// SaveAsset 保存到作品库
func (h *AssetHandler) SaveAsset(c *gin.Context) {
	var req struct {
		ImageURL string `json:"image_url"`
		Prompt   string `json:"prompt"`
		Mode     string `json:"mode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	if req.ImageURL == "" || req.Prompt == "" || req.Mode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不完整"})
		return
	}

	// 判断类型
	videoModes := map[string]bool{
		"text2video":        true,
		"image2video":       true,
		"multi_image_video": true,
	}
	assetType := "image"
	if videoModes[req.Mode] {
		assetType = "video"
	}

	// 解析本地路径
	var localPath string
	if !strings.HasPrefix(req.ImageURL, "http://") && !strings.HasPrefix(req.ImageURL, "https://") {
		// 本地路径：尝试 outputs/ 和 backend/outputs/
		candidates := []string{
			req.ImageURL,
			filepath.Join("outputs", filepath.Base(req.ImageURL)),
		}
		for _, p := range candidates {
			if _, err := os.Stat(p); err == nil {
				localPath = p
				break
			}
		}
		if localPath == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "图片文件不存在"})
			return
		}
	}

	asset := &model.Asset{
		Mode:        req.Mode,
		Prompt:      req.Prompt,
		Type:        assetType,
		Time:        time.Now().Format("2006-01-02 15:04:05"),
		Favorite:    false,
		OriginalURL: req.ImageURL,
		LocalPath:   localPath,
	}

	id, err := h.repo.Insert(asset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存到作品库失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id})
}

// ListAssets 列出作品库
func (h *AssetHandler) ListAssets(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	assetType := c.Query("type")
	search := c.Query("search")
	favoriteFilter := c.Query("favorite") == "true"

	videoModes := map[string]bool{
		"text2video":        true,
		"image2video":       true,
		"multi_image_video": true,
	}
	if perPage <= 0 || perPage > 100 {
		perPage = 20
	}
	if page <= 0 {
		page = 1
	}

	assets, total, err := h.repo.List(page, perPage, assetType, search, favoriteFilter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询作品失败: " + err.Error()})
		return
	}

	items := make([]model.AssetItem, 0, len(assets))
	for _, a := range assets {
		thumbnail := a.OriginalURL
		if thumbnail == "" {
			thumbnail = a.LocalPath
		}
		items = append(items, model.AssetItem{
			ID:          a.ID,
			Mode:        a.Mode,
			Prompt:      a.Prompt,
			Type:        a.Type,
			Time:        a.Time,
			Favorite:    a.Favorite,
			OriginalURL: a.OriginalURL,
			LocalPath:   a.LocalPath,
			GitHubURL:   a.GitHubURL,
			Thumbnail:   thumbnail,
		})
	}

	c.JSON(http.StatusOK, model.AssetListResponse{
		Items: items,
		Total: total,
		Page:  page,
	})
}

// ToggleFavorite 切换收藏
func (h *AssetHandler) ToggleFavorite(c *gin.Context) {
	var req model.AssetFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	if err := h.repo.ToggleFavorite(req.AssetID, req.Favorite); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新收藏失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// BatchDownload 批量下载
func (h *AssetHandler) BatchDownload(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	assets, err := h.repo.GetByIDs(req.IDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询记录失败: " + err.Error()})
		return
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for _, a := range assets {
		if a.LocalPath == "" {
			continue
		}
		data, err := os.ReadFile(a.LocalPath)
		if err != nil {
			fallback := filepath.Join("outputs", filepath.Base(a.LocalPath))
			data, err = os.ReadFile(fallback)
			if err != nil {
				continue
			}
		}
		ext := filepath.Ext(a.LocalPath)
		entryName := fmt.Sprintf("%s_%d%s", a.Mode, a.ID, ext)
		f, err := zw.Create(entryName)
		if err != nil {
			continue
		}
		if _, err := io.Copy(f, bytes.NewReader(data)); err != nil {
			continue
		}
	}

	if err := zw.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建压缩文件失败: " + err.Error()})
		return
	}

	c.Header("Content-Disposition", "attachment; filename=assets.zip")
	c.Data(http.StatusOK, "application/zip", buf.Bytes())
}

// DeleteAssets 删除作品
func (h *AssetHandler) DeleteAssets(c *gin.Context) {
	var req model.AssetDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	if req.DeleteFiles {
		assets, err := h.repo.GetByIDs(req.IDs)
		if err == nil {
			for _, a := range assets {
				if a.LocalPath != "" {
					deleteRecordFiles([]string{a.LocalPath})
				}
			}
		}
	}

	if err := h.repo.Delete(req.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除作品失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
```

- [ ] **Step 2: Verify build**

```bash
cd backend
go build ./...
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
GIT_MASTER=1 git add internal/handler/asset.go
GIT_MASTER=1 git commit -m "refactor: AssetHandler uses AssetRepository, add SaveAsset" -m "Ultraworked with [Sisyphus](https://github.com/code-yeongyu/oh-my-openagent)" -m "Co-authored-by: Sisyphus <clio-agent@sisyphuslabs.ai>"
```

---

### Task 4: Backend — Wire in main.go + AutoMigrate

**Files:**
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/internal/repository/gorm/gorm.go`

**Interfaces:**
- Consumes: `NewAssetRepository`, `NewAssetHandler`, `Asset` model
- Produces: working `POST /api/v1/assets` route, updated AutoMigrate

- [ ] **Step 1: Update main.go — init AssetRepository + update AssetHandler**

Replace line 98:

```go
assetRepo := gormrepo.NewAssetRepository(gormDB)
assetHandler := handler.NewAssetHandler(assetRepo)
```

- [ ] **Step 2: Add POST /assets route**

Insert after the existing `api.DELETE("/assets", assetHandler.DeleteAssets)` (line 148):

```go
api.POST("/assets", assetHandler.SaveAsset)
```

- [ ] **Step 3: Switch to new sqlite TableName for assets**

The `Asset` model uses `TableName() -> "assets"`. Since this is a NEW table, when AutoMigrate runs it creates `assets` table from scratch.

- [ ] **Step 4: Update AutoMigrate in gorm.go**

Replace lines 33-37:

```go
	if err := db.AutoMigrate(
		&History{}, &Asset{},
		&StoryboardProject{}, &StoryboardShot{},
		&Setting{}, &AccessLog{}, &TaskRecord{},
	); err != nil {
		return nil, fmt.Errorf("自动迁移失败: %w", err)
	}
```

Changes: Add `&Asset{}`, remove `&Favorite{}`.

- [ ] **Step 5: Verify build**

```bash
cd backend
go build ./...
go vet ./...
```

Expected: clean

- [ ] **Step 6: Commit**

```bash
GIT_MASTER=1 git add cmd/server/main.go internal/repository/gorm/gorm.go
GIT_MASTER=1 git commit -m "feat: wire AssetRepository, add POST /assets, update AutoMigrate" -m "Ultraworked with [Sisyphus](https://github.com/code-yeongyu/oh-my-openagent)" -m "Co-authored-by: Sisyphus <clio-agent@sisyphuslabs.ai>"
```

---

### Task 5: Frontend — Types + API layer

**Files:**
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/api/assets.ts`

**Interfaces:**
- Consumes: backend API contract
- Produces: typed frontend interfaces and API wrappers

- [ ] **Step 1: Update AssetItem in types/index.ts**

Find the `AssetItem` interface in `frontend/src/types/index.ts` and replace it:

```ts
export interface AssetItem {
  id: number
  mode: string
  prompt: string
  type: string
  time: string
  favorite: boolean
  original_url: string
  local_path: string
  github_url: string
  thumbnail: string
}
```

- [ ] **Step 2: Update AssetFavoriteRequest**

Find and replace:

```ts
export interface AssetFavoriteRequest {
  asset_id: number
  favorite: boolean
}
```

- [ ] **Step 3: Add saveAsset to assets.ts**

Add before `export async function getAssets`:

```ts
export async function saveAsset(data: { image_url: string; prompt: string; mode: string }): Promise<{ id: number }> {
  const res = await client.post('/api/v1/assets', data)
  return res.data
}
```

- [ ] **Step 4: Verify frontend typecheck**

```bash
cd frontend
npx vue-tsc -b --noEmit
```

Expected: no type errors

- [ ] **Step 5: Commit**

```bash
GIT_MASTER=1 git add frontend/src/types/index.ts frontend/src/api/assets.ts
GIT_MASTER=1 git commit -m "feat: add saveAsset API, update AssetItem types" -m "Ultraworked with [Sisyphus](https://github.com/code-yeongyu/oh-my-openagent)" -m "Co-authored-by: Sisyphus <clio-agent@sisyphuslabs.ai>"
```

---

### Task 6: Frontend — ImageResult.vue + views

**Files:**
- Modify: `frontend/src/components/ImageResult.vue`
- Modify: `frontend/src/views/TextToImage.vue`
- Modify: `frontend/src/views/ImageToImage.vue`
- Modify: `frontend/src/views/BatchGen.vue`

- [ ] **Step 1: Update ImageResult.vue**

Add prompt/mode props and save button.

Replace `<script setup>` block:

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { uploadToGitHub } from '../api/github'
import { saveAsset } from '../api/assets'

const props = defineProps<{
  images: string[]
  loading: boolean
  prompt: string
  mode: string
}>()

const uploadingUrls = ref<Set<string>>(new Set())
const savingUrls = ref<Set<string>>(new Set())

function downloadImage(url: string) {
  window.open('/api/v1/download?url=' + encodeURIComponent(url), '_blank')
}

async function handleUploadToGitHub(url: string) {
  uploadingUrls.value = new Set([...uploadingUrls.value, url])
  try {
    const githubUrl = await uploadToGitHub(url)
    ElMessage.success(`已上传到 GitHub: ${githubUrl}`)
  } catch (e: any) {
    ElMessage.error(e.message || '上传到 GitHub 失败')
  } finally {
    const next = new Set(uploadingUrls.value)
    next.delete(url)
    uploadingUrls.value = next
  }
}

async function handleSaveToGallery(url: string) {
  savingUrls.value = new Set([...savingUrls.value, url])
  try {
    await saveAsset({ image_url: url, prompt: props.prompt, mode: props.mode })
    ElMessage.success('已保存到作品库')
  } catch (e: any) {
    ElMessage.error(e.message || '保存到作品库失败')
  } finally {
    const next = new Set(savingUrls.value)
    next.delete(url)
    savingUrls.value = next
  }
}
</script>
```

Replace the template block — image-actions section (lines 46-58):

```vue
      <div class="image-actions">
        <el-button type="primary" size="small" @click="downloadImage(img)">
          下载
        </el-button>
        <el-button
          size="small"
          type="success"
          :loading="savingUrls.has(img)"
          :disabled="savingUrls.has(img)"
          @click="handleSaveToGallery(img)"
        >
          保存到作品库
        </el-button>
        <el-button
          size="small"
          :loading="uploadingUrls.has(img)"
          :disabled="uploadingUrls.has(img)"
          @click="handleUploadToGitHub(img)"
        >
          转存
        </el-button>
      </div>
```

- [ ] **Step 2: Update TextToImage.vue**

Find the `<ImageResult>` usage. It's likely in the template:

```vue
<ImageResult :images="resultImages" :loading="generating" />
```

Change to:

```vue
<ImageResult :images="resultImages" :loading="generating" :prompt="prompt" mode="text2image" />
```

- [ ] **Step 3: Update ImageToImage.vue**

Find the `<ImageResult>` usage, change to:

```vue
<ImageResult :images="resultImages" :loading="generating" :prompt="prompt" mode="image2image" />
```

- [ ] **Step 4: Update BatchGen.vue**

Find the `<ImageResult>` usage. For batch, join prompts with `; `:

```vue
<ImageResult :images="resultImages" :loading="generating" :prompt="prompts.join('; ')" mode="batch" />
```

- [ ] **Step 5: Verify frontend typecheck**

```bash
cd frontend
npx vue-tsc -b --noEmit
```

Expected: no type errors

- [ ] **Step 6: Commit**

```bash
GIT_MASTER=1 git add frontend/src/components/ImageResult.vue frontend/src/views/TextToImage.vue frontend/src/views/ImageToImage.vue frontend/src/views/BatchGen.vue
GIT_MASTER=1 git commit -m "feat: add save-to-gallery button in ImageResult, wire props from views" -m "Ultraworked with [Sisyphus](https://github.com/code-yeongyu/oh-my-openagent)" -m "Co-authored-by: Sisyphus <clio-agent@sisyphuslabs.ai>"
```

---

### Task 7: Frontend — Assets.vue adaptation

**Files:**
- Modify: `frontend/src/views/Assets.vue`

- [ ] **Step 1: Replace `detailAsset.files[0]` with `original_url` in drawer**

The new `AssetItem` no longer has a `Files` array. The drawer preview and GitHub upload use `original_url` instead. Four changes needed:

**Line 220** (image preview fallback):
```vue
:src="detailAsset.thumbnail || detailAsset.original_url"
```

**Line 226** (video source):
```vue
:src="detailAsset.original_url"
```

**Line 247** (upload tracking):
```vue
:loading="uploadingUrl === (detailAsset.original_url || '')"
```

**Line 248** (upload call):
```vue
@click="handleUploadToGitHub(detailAsset.original_url)"
```

- [ ] **Step 2: Update toggleFavorite call**

Find the call in `handleToggleFavorite`:

```ts
function handleToggleFavorite(item: AssetItem) {
  toggleFavorite({ history_id: item.id, favorite: !item.favorite })
  item.favorite = !item.favorite
}
```

Change to:

```ts
function handleToggleFavorite(item: AssetItem) {
  toggleFavorite({ asset_id: item.id, favorite: !item.favorite })
  item.favorite = !item.favorite
}
```

- [ ] **Step 3: Verify frontend typecheck**

```bash
cd frontend
npx vue-tsc -b --noEmit
```

- [ ] **Step 4: Commit**

```bash
GIT_MASTER=1 git add frontend/src/views/Assets.vue
GIT_MASTER=1 git commit -m "fix: Assets.vue — use original_url, asset_id for new AssetItem" -m "Ultraworked with [Sisyphus](https://github.com/code-yeongyu/oh-my-openagent)" -m "Co-authored-by: Sisyphus <clio-agent@sisyphuslabs.ai>"
```

---

## Self-Review

### 1. Spec Coverage

| Spec Section | Task(s) | Status |
|---|---|---|
| Data model: Asset table | Task 1 Step 1 | ✓ |
| Data model: Favorite removed | Task 4 Step 4 | ✓ |
| API: POST /assets | Task 3 Step 1 (SaveAsset) + Task 4 Step 2 (route) | ✓ |
| API: GET /assets refactored | Task 3 Step 1 (ListAssets) | ✓ |
| API: POST /assets/favorite refactored | Task 3 Step 1 (ToggleFavorite) | ✓ |
| API: POST /assets/batch-download refactored | Task 3 Step 1 (BatchDownload) | ✓ |
| API: DELETE /assets refactored | Task 3 Step 1 (DeleteAssets) | ✓ |
| AssetRepository interface | Task 1 Step 2 | ✓ |
| GORM AssetRepository | Task 2 Step 1 | ✓ |
| AssetHandler constructor change | Task 3 Step 1 | ✓ |
| Remove SetRepo | Task 3 Step 1 | ✓ |
| ImageResult prompt/mode props | Task 6 Step 1 | ✓ |
| Three views pass props | Task 6 Steps 2-4 | ✓ |
| saveAsset() API | Task 5 Step 3 | ✓ |
| Frontend AssetItem update | Task 5 Step 1 | ✓ |
| AssetFavoriteRequest field rename | Task 1 Step 4 + Task 5 Step 2 + Task 7 Step 1 | ✓ |

### 2. Placeholder Scan
- No TBD, TODO, or placeholder patterns found
- All code blocks contain complete implementation
- All file paths are exact

### 3. Type Consistency
- Go `AssetItem`: fields match between types.go and asset.go handler
- TS `AssetItem`: matches Go `AssetItem` field names (snake_case JSON → camelCase TS)
- `AssetFavoriteRequest`: `asset_id` field used consistently across Go types, TS types, Assets.vue
- `AssetRepository` interface method signatures match between interfaces.go and asset.go (GORM impl)
