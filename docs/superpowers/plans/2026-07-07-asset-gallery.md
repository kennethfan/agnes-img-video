# Asset Gallery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a unified gallery view and asset management (favorites, batch download/delete) for all generated images and videos.

**Architecture:** Backend extends the existing SQLite history repository with a favorites table and a new `AssetHandler`. Frontend adds a new Assets gallery view as an `el-tab-pane` in App.vue with search, filter, grid display, detail drawer, and batch operations.

**Tech Stack:** Go 1.25 + Gin + SQLite (mattn/go-sqlite3) · Vue 3 + TypeScript 6 + Element Plus + Pinia + Axios

**Spec:** `docs/superpowers/specs/2026-07-07-asset-gallery-design.md`

## Global Constraints

- All new backend files in `backend/internal/handler/` and `backend/internal/repository/`
- All new frontend files in `frontend/src/views/`, `frontend/src/components/`, `frontend/src/api/`
- Follow existing patterns: Gin handler struct pattern, Vue `<script setup lang="ts">`, Element Plus UI components
- No auth middleware (project is local-dev only)
- No vue-router — use `el-tabs` for navigation
- TypeScript: `erasableSyntaxOnly` enabled — no enums or namespaces, use `as const` or union types
- Frontend uses Flush: `sync` for redo store watchers

---

### Task 1: Backend — Add favorites table and repository methods

**Files:**
- Modify: `backend/internal/repository/history.go` — add `favorites` table and CRUD
- Test: `backend/internal/repository/history_test.go` — add favorites tests

**Interfaces:**
- Consumes: existing `HistoryRepo` struct and `GetRecords` method
- Produces: `ToggleFavorite(historyID int64, favorite bool) error`, `GetFavoriteIDs() (map[int64]bool, error)`, `GetFavoritedRecordIDs() ([]int64, error)`

- [ ] **Step 1: Add favorites table creation to `NewHistoryRepo`**

In `backend/internal/repository/history.go`, add the `favorites` table to the schema migration after the `history` table:

```go
_, err = db.Exec(`
    CREATE TABLE IF NOT EXISTS favorites (
        history_id INTEGER PRIMARY KEY,
        created_at TEXT DEFAULT (datetime('now')),
        FOREIGN KEY (history_id) REFERENCES history(id) ON DELETE CASCADE
    )
`)
if err != nil {
    db.Close()
    return nil, fmt.Errorf("创建收藏表失败: %w", err)
}
```

- [ ] **Step 2: Add `ToggleFavorite` and `GetFavoriteIDs` methods**

Add these methods to `HistoryRepo`:

```go
// ToggleFavorite 收藏/取消收藏
func (r *HistoryRepo) ToggleFavorite(historyID int64, favorite bool) error {
    if favorite {
        _, err := r.db.Exec(
            "INSERT OR IGNORE INTO favorites (history_id) VALUES (?)", historyID,
        )
        return err
    }
    _, err := r.db.Exec("DELETE FROM favorites WHERE history_id = ?", historyID)
    return err
}

// GetFavoriteIDs 获取所有收藏的历史记录ID
func (r *HistoryRepo) GetFavoriteIDs() (map[int64]bool, error) {
    rows, err := r.db.Query("SELECT history_id FROM favorites")
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    favs := make(map[int64]bool)
    for rows.Next() {
        var id int64
        if err := rows.Scan(&id); err == nil {
            favs[id] = true
        }
    }
    return favs, rows.Err()
}
```

- [ ] **Step 3: Run existing tests to verify nothing is broken**

```bash
cd backend && go test ./internal/repository/ -v
```

Expected: All existing tests PASS.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/repository/history.go
git commit -m "feat: add favorites table and repository methods"
```

---

### Task 2: Backend — Implement Assets GET API with pagination and filtering

**Files:**
- Create: `backend/internal/handler/asset.go` — `AssetHandler` with `ListAssets`
- Modify: `backend/internal/model/types.go` — add `AssetItem`, `AssetListResponse`, `AssetListQuery`
- Modify: `backend/cmd/server/main.go` — register new route
- Modify: `backend/internal/repository/history.go` — add `GetRecordsPaginated`

**Interfaces:**
- Consumes: `HistoryRepo` with `GetRecordsPaginated(page, perPage int, assetType, search string)`
- Produces: `GET /api/v1/assets?page=1&per_page=20&type=image|video|all&sort=newest|oldest&search=&favorite=true|false`

- [ ] **Step 1: Add paginated query method to `HistoryRepo`**

```go
// GetRecordsPaginated 分页查询历史记录
// type: "image", "video", or "all" (default)
// search: search in prompt field (LIKE %search%)
func (r *HistoryRepo) GetRecordsPaginated(page, perPage int, assetType, search string, favIDs map[int64]bool) ([]model.HistoryRecord, int, error) {
    if page < 1 {
        page = 1
    }
    if perPage < 1 || perPage > 100 {
        perPage = 20
    }

    // Build WHERE clause
    var conditions []string
    var args []any

    if assetType == "image" {
        conditions = append(conditions, "(mode IN ('text2image','image2image','batch'))")
    } else if assetType == "video" {
        conditions = append(conditions, "(mode IN ('text2video','image2video','multi_image_video'))")
    }
    // "all" — no filter on mode

    if search != "" {
        conditions = append(conditions, "prompt LIKE ?")
        args = append(args, "%"+search+"%")
    }

    whereClause := ""
    if len(conditions) > 0 {
        whereClause = " WHERE " + strings.Join(conditions, " AND ")
    }

    // Count total
    countQuery := "SELECT COUNT(*) FROM history" + whereClause
    var total int
    if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
        return nil, 0, err
    }

    // Query page
    offset := (page - 1) * perPage
    queryArgs := append(append([]any{}, args...), perPage, offset)
    dataQuery := fmt.Sprintf(
        "SELECT id, time, mode, prompt, images, extra FROM history%s ORDER BY id DESC LIMIT ? OFFSET ?",
        whereClause,
    )
    rows, err := r.db.Query(dataQuery, queryArgs...)
    if err != nil {
        return nil, 0, err
    }
    defer rows.Close()

    records, err := scanRecords(rows)
    return records, total, err
}
```

Add `scanRecords` helper to avoid duplication between `GetRecords` and `GetRecordsPaginated`:

```go
func scanRecords(rows *sql.Rows) ([]model.HistoryRecord, error) {
    var records []model.HistoryRecord
    for rows.Next() {
        var (
            id                int64
            time, mode, prompt string
            imagesJSON         string
            extraJSON          *string
        )
        if err := rows.Scan(&id, &time, &mode, &prompt, &imagesJSON, &extraJSON); err != nil {
            return nil, err
        }
        var images []string
        json.Unmarshal([]byte(imagesJSON), &images)
        if images == nil {
            images = []string{}
        }
        rec := model.HistoryRecord{
            ID:     id,
            Time:   time,
            Mode:   mode,
            Prompt: prompt,
            Images: images,
        }
        if extraJSON != nil {
            var extra any
            if err := json.Unmarshal([]byte(*extraJSON), &extra); err == nil {
                rec.Extra = extra
            }
        }
        records = append(records, rec)
    }
    return records, rows.Err()
}
```

Refactor `GetRecords` to use `scanRecords`:

```go
func (r *HistoryRepo) GetRecords(limit int) ([]model.HistoryRecord, error) {
    if limit <= 0 {
        limit = 100
    }
    rows, err := r.db.Query(
        "SELECT id, time, mode, prompt, images, extra FROM history ORDER BY id DESC LIMIT ?",
        limit,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    return scanRecords(rows)
}
```

- [ ] **Step 2: Add `AssetItem` and response types to `model/types.go`**

```go
// ==================== 资产管理 ====================

type AssetItem struct {
    ID        int64  `json:"id"`
    Mode      string `json:"mode"`
    Prompt    string `json:"prompt"`
    Files     []string `json:"files"`
    Thumbnail string `json:"thumbnail"`
    Type      string `json:"type"` // "image" or "video"
    Time      string `json:"time"`
    Favorite  bool   `json:"favorite"`
}

type AssetListResponse struct {
    Items []AssetItem `json:"items"`
    Total int         `json:"total"`
    Page  int         `json:"page"`
}

type AssetFavoriteRequest struct {
    HistoryID int64 `json:"history_id" binding:"required"`
    Favorite  bool  `json:"favorite"`
}

type AssetDeleteRequest struct {
    IDs         []int64 `json:"ids" binding:"required"`
    DeleteFiles bool    `json:"delete_files"`
}
```

- [ ] **Step 3: Create `AssetHandler` with `ListAssets`**

Create `backend/internal/handler/asset.go`:

```go
package handler

import (
    "log"
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"

    "github.com/agnes-image-tool/backend/internal/model"
    "github.com/agnes-image-tool/backend/internal/repository"
)

type AssetHandler struct {
    repo *repository.HistoryRepo
}

func NewAssetHandler(repo *repository.HistoryRepo) *AssetHandler {
    return &AssetHandler{repo: repo}
}

func (h *AssetHandler) ListAssets(c *gin.Context) {
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
    assetType := c.DefaultQuery("type", "all")
    search := c.Query("search")
    favoriteFilter := c.Query("favorite")

    if page < 1 {
        page = 1
    }
    if perPage < 1 || perPage > 100 {
        perPage = 20
    }

    // Get favorited IDs
    favIDs, err := h.repo.GetFavoriteIDs()
    if err != nil {
        log.Printf("[Asset] 获取收藏列表失败: %v", err)
        favIDs = make(map[int64]bool)
    }

    // If filtering by favorite, convert to specific IDs
    if favoriteFilter == "true" && len(favIDs) == 0 {
        c.JSON(http.StatusOK, model.AssetListResponse{
            Items: []model.AssetItem{},
            Total: 0,
            Page:  page,
        })
        return
    }

    records, total, err := h.repo.GetRecordsPaginated(page, perPage, assetType, search, favIDs)
    if err != nil {
        log.Printf("[Asset] 查询失败: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
        return
    }

    // Build AssetItem list
    items := make([]model.AssetItem, 0, len(records))
    for _, rec := range records {
        isVideo := rec.Mode == "text2video" || rec.Mode == "image2video" || rec.Mode == "multi_image_video"
        assetType := "image"
        if isVideo {
            assetType = "video"
        }

        // For favorite filter, skip non-favorited
        if favoriteFilter == "true" && !favIDs[rec.ID] {
            continue
        }

        thumbnail := ""
        if len(rec.Images) > 0 {
            thumbnail = rec.Images[0]
        }

        items = append(items, model.AssetItem{
            ID:        rec.ID,
            Mode:      rec.Mode,
            Prompt:    rec.Prompt,
            Files:     rec.Images,
            Thumbnail: thumbnail,
            Type:      assetType,
            Time:      rec.Time,
            Favorite:  favIDs[rec.ID],
        })
    }

    c.JSON(http.StatusOK, model.AssetListResponse{
        Items: items,
        Total: total,
        Page:  page,
    })
}
```

- [ ] **Step 4: Register route in `main.go`**

In `backend/cmd/server/main.go`, after `configHandler` initialization, add:

```go
assetHandler := handler.NewAssetHandler(histRepo)
```

And inside the API routes group, add:

```go
// 资产管理
api.GET("/assets", assetHandler.ListAssets)
api.POST("/assets/favorite", assetHandler.ToggleFavorite)
api.POST("/assets/batch-download", assetHandler.BatchDownload)
api.DELETE("/assets", assetHandler.DeleteAssets)
```

- [ ] **Step 5: Verify compilation**

```bash
cd backend && go build ./...
```

Expected: Build succeeds with no errors.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/handler/asset.go backend/internal/model/types.go \
       backend/internal/repository/history.go backend/cmd/server/main.go
git commit -m "feat: add asset list API with pagination and filtering"
```

---

### Task 3: Backend — Implement favorite toggle API

**Files:**
- Modify: `backend/internal/handler/asset.go` — add `ToggleFavorite`

- [ ] **Step 1: Add `ToggleFavorite` handler**

In `asset.go`:

```go
func (h *AssetHandler) ToggleFavorite(c *gin.Context) {
    var req model.AssetFavoriteRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
        return
    }

    if err := h.repo.ToggleFavorite(req.HistoryID, req.Favorite); err != nil {
        log.Printf("[Asset] 收藏操作失败: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"ok": true})
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd backend && go build ./...
```

Expected: Build succeeds.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/handler/asset.go
git commit -m "feat: add favorite toggle API"
```

---

### Task 4: Backend — Implement batch download zip API

**Files:**
- Modify: `backend/internal/handler/asset.go` — add `BatchDownload`
- Modify: `backend/internal/handler/history.go` — export or reuse `deleteRecordFiles` pattern

- [ ] **Step 1: Add `BatchDownload` handler**

```go
func (h *AssetHandler) BatchDownload(c *gin.Context) {
    var req struct {
        IDs []int64 `json:"ids" binding:"required"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
        return
    }

    if len(req.IDs) == 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "请选择要下载的文件"})
        return
    }

    // Collect file paths from history records
    records, err := h.repo.GetRecordsByIDs(req.IDs)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "查询记录失败"})
        return
    }

    // Create zip in memory
    buf := new(bytes.Buffer)
    zw := zip.NewWriter(buf)

    for _, rec := range records {
        for i, filePath := range rec.Images {
            if filePath == "" || strings.HasPrefix(filePath, "http") {
                continue // skip empty or remote URLs
            }

            // Resolve local file path — stored paths may be "outputs/xxx" or "../outputs/xxx"
            localPath := filePath
            data, err := os.ReadFile(localPath)
            if err != nil {
                // fallback: try bare filename in outputs/
                localPath = filepath.Join("outputs", filepath.Base(filePath))
                data, err = os.ReadFile(localPath)
                if err != nil {
                    log.Printf("[Asset] 读取文件失败 %s: %v", localPath, err)
                    continue
                }
            }

            ext := filepath.Ext(filePath)
            name := fmt.Sprintf("%s_%d_%d%s", rec.Mode, rec.ID, i, ext)
            f, err := zw.Create(name)
            if err != nil {
                continue
            }
            f.Write(data)
        }
    }

    if err := zw.Close(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "打包失败"})
        return
    }

    c.Header("Content-Type", "application/zip")
    c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="agnes-assets-%s.zip"`, time.Now().Format("20060102")))
    c.Data(http.StatusOK, "application/zip", buf.Bytes())
}
```

- [ ] **Step 2: Add `GetRecordsByIDs` to `HistoryRepo`**

In `backend/internal/repository/history.go`:

```go
// GetRecordsByIDs 根据ID列表查询历史记录
func (r *HistoryRepo) GetRecordsByIDs(ids []int64) ([]model.HistoryRecord, error) {
    if len(ids) == 0 {
        return nil, nil
    }
    placeholders := make([]string, len(ids))
    args := make([]any, len(ids))
    for i, id := range ids {
        placeholders[i] = "?"
        args[i] = id
    }
    q := fmt.Sprintf(
        "SELECT id, time, mode, prompt, images, extra FROM history WHERE id IN (%s) ORDER BY id DESC",
        strings.Join(placeholders, ","),
    )
    rows, err := r.db.Query(q, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    return scanRecords(rows)
}
```

- [ ] **Step 3: Add imports to `asset.go`**

Ensure these imports are added:

```go
import (
    "archive/zip"
    "bytes"
    "fmt"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
)
```

- [ ] **Step 4: Verify compilation**

```bash
cd backend && go build ./...
```

Expected: Build succeeds.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/handler/asset.go backend/internal/repository/history.go
git commit -m "feat: add batch download zip API"
```

---

### Task 5: Backend — Implement batch delete assets API

**Files:**
- Modify: `backend/internal/handler/asset.go` — add `DeleteAssets`

- [ ] **Step 1: Add `DeleteAssets` handler**

In `asset.go`:

```go
func (h *AssetHandler) DeleteAssets(c *gin.Context) {
    var req model.AssetDeleteRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
        return
    }

    if len(req.IDs) == 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "请选择要删除的记录"})
        return
    }

    // If delete_files, get file paths first
    if req.DeleteFiles {
        records, err := h.repo.GetRecordsByIDs(req.IDs)
        if err == nil {
            for _, rec := range records {
                deleteRecordFiles(rec.Images)
            }
        }
    }

    if err := h.repo.DeleteRecords(req.IDs); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "删除记录失败: " + err.Error()})
        return
    }

    // Also clean up favorites for these IDs
    for _, id := range req.IDs {
        h.repo.ToggleFavorite(id, false)
    }

    c.JSON(http.StatusOK, gin.H{"ok": true})
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd backend && go build ./...
```

Expected: Build succeeds.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/handler/asset.go
git commit -m "feat: add batch delete assets API"
```

---

### Task 6: Frontend — Add asset types and API client

**Files:**
- Modify: `frontend/src/types/index.ts` — add `AssetItem`, `AssetListResponse`
- Create: `frontend/src/api/assets.ts` — API client

- [ ] **Step 1: Add `AssetItem` and related types to `frontend/src/types/index.ts`**

```typescript
export interface AssetItem {
  id: number
  mode: string
  prompt: string
  files: string[]
  thumbnail: string
  type: 'image' | 'video'
  time: string
  favorite: boolean
}

export interface AssetListResponse {
  items: AssetItem[]
  total: number
  page: number
}

export interface AssetFavoriteRequest {
  history_id: number
  favorite: boolean
}

export interface AssetDeleteRequest {
  ids: number[]
  delete_files?: boolean
}
```

- [ ] **Step 2: Create `frontend/src/api/assets.ts`**

```typescript
import client from './client'
import type { AssetItem, AssetListResponse, AssetFavoriteRequest, AssetDeleteRequest } from '../types'

export interface AssetQuery {
  page?: number
  per_page?: number
  type?: 'image' | 'video' | 'all'
  sort?: 'newest' | 'oldest'
  search?: string
  favorite?: string
}

export async function getAssets(params: AssetQuery = {}): Promise<AssetListResponse> {
  const res = await client.get('/api/v1/assets', { params })
  return res.data
}

export async function toggleFavorite(data: AssetFavoriteRequest): Promise<void> {
  await client.post('/api/v1/assets/favorite', data)
}

export async function batchDownload(ids: number[]): Promise<Blob> {
  const res = await client.post('/api/v1/assets/batch-download', { ids }, {
    responseType: 'blob',
  })
  return res.data
}

export async function deleteAssets(data: AssetDeleteRequest): Promise<void> {
  await client.delete('/api/v1/assets', { data })
}
```

- [ ] **Step 3: Run frontend typecheck**

```bash
cd frontend && npx vue-tsc --noEmit
```

Expected: No type errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/types/index.ts frontend/src/api/assets.ts
git commit -m "feat: add asset types and API client"
```

---

### Task 7: Frontend — Create AssetCard component

**Files:**
- Create: `frontend/src/components/AssetCard.vue`

- [ ] **Step 1: Create `AssetCard.vue`**

```vue
<script setup lang="ts">
import { computed } from 'vue'
import type { AssetItem } from '../types'

const props = defineProps<{
  asset: AssetItem
  selected: boolean
}>()

const emit = defineEmits<{
  (e: 'toggle-favorite'): void
  (e: 'toggle-select'): void
  (e: 'click'): void
}>()

const modeLabel = computed(() => {
  const labels: Record<string, string> = {
    text2image: '文生图',
    image2image: '图生图',
    batch: '批量',
    text2video: '文生视频',
    image2video: '图生视频',
    multi_image_video: '多图视频',
  }
  return labels[props.asset.mode] || props.asset.mode
})

const modeColor = computed(() => {
  const colors: Record<string, string> = {
    text2image: '#409eff',
    image2image: '#67c23a',
    batch: '#e6a23c',
    text2video: '#f56c6c',
    image2video: '#f56c6c',
    multi_image_video: '#f56c6c',
  }
  return colors[props.asset.mode] || '#909399'
})

const isVideo = computed(() => props.asset.type === 'video')

function formatTime(timeStr: string): string {
  return timeStr?.slice(5, 16) || ''
}
</script>

<template>
  <div
    class="asset-card"
    :class="{ 'is-selected': selected }"
    @click="emit('click')"
  >
    <!-- Selection checkbox -->
    <div class="asset-card__select" @click.stop="emit('toggle-select')">
      <el-checkbox :model-value="selected" />
    </div>

    <!-- Favorite toggle -->
    <div class="asset-card__favorite" @click.stop="emit('toggle-favorite')">
      <el-icon :color="asset.favorite ? '#f56c6c' : '#c0c4cc'" :size="18">
        <StarFilled v-if="asset.favorite" />
        <Star v-else />
      </el-icon>
    </div>

    <!-- Thumbnail -->
    <div class="asset-card__thumb">
      <el-image
        v-if="!isVideo && asset.thumbnail"
        :src="asset.thumbnail"
        fit="cover"
        style="width: 100%; height: 100%"
      >
        <template #error>
          <div class="asset-card__placeholder">
            <el-icon :size="32"><PictureFilled /></el-icon>
          </div>
        </template>
      </el-image>
      <div v-else-if="isVideo" class="asset-card__placeholder">
        <el-icon :size="32"><VideoCameraFilled /></el-icon>
      </div>
      <div v-else class="asset-card__placeholder">
        <el-icon :size="32"><PictureFilled /></el-icon>
      </div>
    </div>

    <!-- Mode badge -->
    <div
      class="asset-card__badge"
      :style="{ background: modeColor }"
    >
      {{ modeLabel }}
    </div>

    <!-- Prompt -->
    <div class="asset-card__prompt">
      {{ asset.prompt?.slice(0, 40) }}{{ asset.prompt?.length > 40 ? '...' : '' }}
    </div>

    <!-- Time -->
    <div class="asset-card__time">
      {{ formatTime(asset.time) }}
    </div>
  </div>
</template>

<style scoped>
.asset-card {
  position: relative;
  border: 1px solid #ebeef5;
  border-radius: 6px;
  overflow: hidden;
  cursor: pointer;
  transition: box-shadow 0.2s, border-color 0.2s;
  background: #fff;
}
.asset-card:hover {
  box-shadow: 0 2px 12px rgba(0,0,0,0.1);
  border-color: #409eff;
}
.asset-card.is-selected {
  border-color: #409eff;
  box-shadow: 0 0 0 2px rgba(64,158,255,0.2);
}
.asset-card__select {
  position: absolute;
  top: 6px;
  left: 6px;
  z-index: 2;
}
.asset-card__favorite {
  position: absolute;
  top: 6px;
  right: 6px;
  z-index: 2;
  cursor: pointer;
  background: rgba(255,255,255,0.8);
  border-radius: 50%;
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.asset-card__thumb {
  width: 100%;
  aspect-ratio: 1;
  overflow: hidden;
  background: #f5f7fa;
}
.asset-card__placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #c0c4cc;
  background: #f5f7fa;
  min-height: 120px;
}
.asset-card__badge {
  position: absolute;
  top: 6px;
  left: 38px;
  color: #fff;
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 3px;
  line-height: 1.4;
}
.asset-card__prompt {
  padding: 6px 8px;
  font-size: 12px;
  color: #606266;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.asset-card__time {
  padding: 0 8px 6px;
  font-size: 11px;
  color: #c0c4cc;
}
</style>
```

- [ ] **Step 2: Run frontend typecheck**

```bash
cd frontend && npx vue-tsc --noEmit
```

Expected: No type errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/AssetCard.vue
git commit -m "feat: add AssetCard component"
```

---

### Task 8: Frontend — Create Assets gallery page view

**Files:**
- Create: `frontend/src/views/Assets.vue`
- Modify: `frontend/src/App.vue` — add tab

- [ ] **Step 1: Create `Assets.vue`**

```vue
<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Star, StarFilled, Download, Delete, PictureFilled, VideoCameraFilled } from '@element-plus/icons-vue'
import { getAssets, toggleFavorite, batchDownload, deleteAssets } from '../api/assets'
import type { AssetItem, AssetQuery } from '../types'
import AssetCard from '../components/AssetCard.vue'

const items = ref<AssetItem[]>([])
const total = ref(0)
const loading = ref(false)
const page = ref(1)
const perPage = ref(20)
const assetType = ref<'all' | 'image' | 'video'>('all')
const search = ref('')
const showFavorites = ref(false)
const selectionMode = ref(false)
const selectedIds = ref<Set<number>>(new Set())
const deleteFiles = ref(false)

// Detail drawer
const drawerVisible = ref(false)
const detailAsset = ref<AssetItem | null>(null)

const totalPages = computed(() => Math.ceil(total.value / perPage.value))

const sort = computed<'newest' | 'oldest'>(() => 'newest')

const queryParams = computed<AssetQuery>(() => ({
  page: page.value,
  per_page: perPage.value,
  type: assetType.value === 'all' ? undefined : assetType.value,
  search: search.value || undefined,
  favorite: showFavorites.value ? 'true' : undefined,
}))

async function loadAssets() {
  loading.value = true
  try {
    const res = await getAssets(queryParams.value)
    items.value = res.items
    total.value = res.total
  } catch (e: any) {
    ElMessage.error('加载作品失败: ' + (e.message || ''))
  } finally {
    loading.value = false
  }
}

onMounted(loadAssets)

function handleSearch() {
  page.value = 1
  loadAssets()
}

function handleTypeChange() {
  page.value = 1
  loadAssets()
}

function toggleFavoritesFilter() {
  showFavorites.value = !showFavorites.value
  page.value = 1
  loadAssets()
}

async function handleToggleFavorite(asset: AssetItem) {
  try {
    await toggleFavorite({ history_id: asset.id, favorite: !asset.favorite })
    asset.favorite = !asset.favorite
  } catch (e: any) {
    ElMessage.error('操作失败')
  }
}

function toggleSelection(id: number) {
  const next = new Set(selectedIds.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selectedIds.value = next
}

function openDetail(asset: AssetItem) {
  detailAsset.value = asset
  drawerVisible.value = true
}

async function handleBatchDownload() {
  const ids = Array.from(selectedIds.value)
  if (ids.length === 0) return
  try {
    const blob = await batchDownload(ids)
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `agnes-assets-${new Date().toISOString().slice(0, 10)}.zip`
    a.click()
    URL.revokeObjectURL(url)
  } catch (e: any) {
    ElMessage.error('下载失败: ' + (e.message || ''))
  }
}

async function handleBatchDelete() {
  const ids = Array.from(selectedIds.value)
  if (ids.length === 0) return
  try {
    await ElMessageBox.confirm(
      `确定删除选中的 ${ids.length} 条记录？`,
      '确认删除',
      { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' }
    )
    await deleteAssets({ ids, delete_files: deleteFiles.value })
    ElMessage.success('删除成功')
    selectedIds.value = new Set()
    await loadAssets()
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}
</script>

<template>
  <div>
    <!-- Toolbar -->
    <div style="display: flex; gap: 12px; margin-bottom: 16px; flex-wrap: wrap; align-items: center">
      <el-input
        v-model="search"
        placeholder="搜索提示词..."
        clearable
        style="width: 240px"
        size="small"
        @keyup.enter="handleSearch"
        @clear="handleSearch"
      >
        <template #prefix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>

      <el-select v-model="assetType" size="small" style="width: 120px" @change="handleTypeChange">
        <el-option label="全部" value="all" />
        <el-option label="图片" value="image" />
        <el-option label="视频" value="video" />
      </el-select>

      <el-button
        :type="showFavorites ? 'danger' : 'default'"
        size="small"
        :icon="showFavorites ? StarFilled : Star"
        @click="toggleFavoritesFilter"
      >
        {{ showFavorites ? '全部' : '仅收藏' }}
      </el-button>

      <div style="flex: 1" />

      <el-button
        size="small"
        :type="selectionMode ? 'primary' : 'default'"
        @click="selectionMode = !selectionMode"
      >
        {{ selectionMode ? '取消选择' : '选择' }}
      </el-button>
    </div>

    <!-- Batch action bar -->
    <div v-if="selectionMode && selectedIds.size > 0" style="margin-bottom: 12px; display: flex; gap: 12px; align-items: center">
      <span style="font-size: 13px; color: #606266">
        已选 {{ selectedIds.size }} 项
      </span>
      <el-button type="primary" size="small" :icon="Download" @click="handleBatchDownload">
        下载 ({{ selectedIds.size }})
      </el-button>
      <el-checkbox v-model="deleteFiles" style="margin-left: 8px">
        同时删除文件
      </el-checkbox>

      <div style="flex: 1" />
      <el-button type="danger" size="small" :icon="Delete" @click="handleBatchDelete">
        删除 ({{ selectedIds.size }})
      </el-button>
    </div>

    <!-- Grid -->
    <div v-loading="loading" :style="{ minHeight: loading ? '200px' : 'auto' }">
      <div v-if="items.length === 0 && !loading" style="text-align: center; padding: 60px; color: #c0c4cc">
        <el-icon :size="48"><PictureFilled /></el-icon>
        <p style="margin-top: 12px">暂无作品</p>
      </div>

      <div v-else style="display: grid; grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); gap: 16px">
        <AssetCard
          v-for="asset in items"
          :key="asset.id"
          :asset="asset"
          :selected="selectedIds.has(asset.id)"
          @toggle-favorite="handleToggleFavorite(asset)"
          @toggle-select="toggleSelection(asset.id)"
          @click="selectionMode ? toggleSelection(asset.id) : openDetail(asset)"
        />
      </div>
    </div>

    <!-- Pagination -->
    <div v-if="total > perPage" style="text-align: center; margin-top: 24px">
      <el-pagination
        v-model:current-page="page"
        :page-size="perPage"
        :total="total"
        layout="prev, pager, next"
        @current-change="loadAssets"
      />
    </div>

    <!-- Detail drawer -->
    <el-drawer
      v-model="drawerVisible"
      :title="detailAsset?.prompt?.slice(0, 50) || '作品详情'"
      size="500px"
    >
      <template v-if="detailAsset">
        <div v-if="detailAsset.type === 'image' && detailAsset.thumbnail">
          <el-image
            :src="detailAsset.thumbnail"
            fit="contain"
            style="width: 100%; max-height: 400px"
          />
        </div>
        <video
          v-else-if="detailAsset.type === 'video' && detailAsset.thumbnail"
          :src="detailAsset.thumbnail"
          controls
          style="width: 100%; max-height: 400px"
        />

        <el-divider />

        <p style="font-size: 14px; color: #303133; white-space: pre-wrap">{{ detailAsset.prompt }}</p>
        <p style="font-size: 12px; color: #909399; margin-top: 8px">
          模式: {{ detailAsset.mode }} · {{ detailAsset.time }}
        </p>

        <div style="margin-top: 16px; display: flex; gap: 12px">
          <el-button
            :type="detailAsset.favorite ? 'danger' : 'default'"
            size="small"
            @click="handleToggleFavorite(detailAsset!)"
          >
            {{ detailAsset.favorite ? '取消收藏' : '收藏' }}
          </el-button>
          <el-button size="small" @click="drawerVisible = false">
            关闭
          </el-button>
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<style scoped>
:deep(.el-pagination) {
  justify-content: center;
}
</style>
```

- [ ] **Step 2: Add tab to `App.vue`**

In `frontend/src/App.vue`, add imports:

```typescript
import Assets from './views/Assets.vue'
```

Add tab before 历史记录:

```vue
<el-tab-pane label="作品" name="assets">
  <Assets />
</el-tab-pane>
```

- [ ] **Step 3: Run frontend typecheck**

```bash
cd frontend && npx vue-tsc --noEmit
```

Expected: No type errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/views/Assets.vue frontend/src/App.vue
git commit -m "feat: add assets gallery view with search, filter, and detail drawer"
```

---

### Task 9: Integration test and polish

**Files:**
- Modify: any files with bugs found during testing

- [ ] **Step 1: Start backend and verify assets API**

```bash
cd backend && go run ./cmd/server
```

Test in another terminal:

```bash
# List assets
curl "http://localhost:8080/api/v1/assets?page=1&per_page=5"

# Toggle favorite
curl -X POST "http://localhost:8080/api/v1/assets/favorite" \
  -H "Content-Type: application/json" \
  -d '{"history_id": 1, "favorite": true}'

# Delete assets
curl -X DELETE "http://localhost:8080/api/v1/assets" \
  -H "Content-Type: application/json" \
  -d '{"ids": [1], "delete_files": false}'
```

Expected: All endpoints return correct responses.

- [ ] **Step 2: Start frontend and verify gallery page**

```bash
cd frontend && pnpm dev
```

Verify:
1. "作品" tab appears and loads assets
2. Search/filter work
3. Favorite toggle works
4. Batch selection + batch download works
5. Detail drawer opens on click
6. Pagination works when enough records exist

- [ ] **Step 3: Run full backend test suite**

```bash
cd backend && go test ./... -v
```

Expected: All tests pass.

- [ ] **Step 4: Run frontend typecheck**

```bash
cd frontend && npx vue-tsc --noEmit && pnpm build
```

Expected: Build succeeds.

- [ ] **Step 5: Commit final polish**

```bash
git add -A
git commit -m "feat: complete asset gallery with polish and fixes"
```
