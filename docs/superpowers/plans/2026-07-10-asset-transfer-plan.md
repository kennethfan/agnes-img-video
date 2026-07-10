# Asset Transfer (转存) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract shared `storeFile()` method for server-side storage decision (local/GitHub/both) and add `POST /api/v1/assets/:id/transfer` endpoint.

**Architecture:** Extract the "download remote file → store per storage_target" logic from `AssetHandler.SaveAsset` into reusable `storeFile()`. Add new repository method `UpdateStoragePaths`. Add new handler `TransferAsset`. Wire up in routes. Update frontend `Assets.vue` drawer to call the new endpoint instead of direct GitHub upload.

**Tech Stack:** Go 1.25 + Gin + GORM + Vue 3 + TypeScript

## Global Constraints

- No new GORM models or DB migrations (DB schema already stable)
- `githubStorage` remains package-level global variable (not injected)
- Existing `UpdateGithubURL` method is preserved (not removed)
- Chinese error messages for all new handler responses
- Frontend uses Composition API + `<script setup>` only

---

### Task 1: Extract `storeFile()` from `SaveAsset`

**Files:**
- Modify: `backend/internal/handler/asset.go`

**Interfaces:**
- Produces: `func (h *AssetHandler) storeFile(imageURL string, assetType string) (localPath string, githubURL string, err error)` — reusable across SaveAsset and TransferAsset

- [ ] **Step 1: Extract `storeFile()` method**

Add a new method on `AssetHandler` that contains the remote URL download + storage logic (lines 79-146 of current SaveAsset). The method signature:

```go
// storeFile 下载远程文件并根据 storage_target 处理存储
// 返回 (localPath, githubURL, error)
func (h *AssetHandler) storeFile(imageURL string, assetType string) (string, string, error)
```

Implementation (extracted from SaveAsset's remote URL branch):

```go
func (h *AssetHandler) storeFile(imageURL string, assetType string) (string, string, error) {
	storageTarget := "local"
	if s, err := h.settingsRepo.GetSettings(); err == nil {
		storageTarget = s.StorageTarget
	}

	outputDir := "outputs"
	os.MkdirAll(outputDir, 0755)

	ext := filepath.Ext(imageURL)
	if ext == "" {
		ext = ".png"
		if assetType == "video" {
			ext = ".mp4"
		}
	}

	timestamp := time.Now().Format("20060102_150405_000000")
	filename := fmt.Sprintf("asset_%s%s", timestamp, ext)
	filePath := filepath.Join(outputDir, filename)

	resp, err := http.Get(imageURL)
	if err != nil {
		return "", "", fmt.Errorf("下载文件失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("下载文件失败: 上游返回 %s", http.StatusText(resp.StatusCode))
	}

	out, err := os.Create(filePath)
	if err != nil {
		return "", "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", "", fmt.Errorf("写入文件失败: %w", err)
	}

	var localPath string
	var githubURL string

	saveLocal := storageTarget == "local" || storageTarget == "both"
	uploadGithub := (storageTarget == "github" || storageTarget == "both") && githubStorage != nil

	if saveLocal {
		localPath = filePath
	}

	if uploadGithub {
		remotePath := fmt.Sprintf("images/%s", filename)
		uploadedURL, err := githubStorage.UploadFile(filePath, remotePath)
		if err != nil {
			log.Printf("[Asset] 上传到 GitHub 失败: %v", err)
		} else {
			githubURL = uploadedURL
		}
	}

	// 仅 GitHub 模式：上传后删除本地临时文件
	if storageTarget == "github" && githubURL != "" {
		os.Remove(filePath)
	}

	return localPath, githubURL, nil
}
```

Add the new imports `"errors"` and `"fmt"` (check if they're already imported; `fmt` is, `errors` may not be).

- [ ] **Step 2: Refactor `SaveAsset` to use `storeFile()`**

Replace the remote URL branch body (lines 78-146) with a call to `storeFile()`:

```go
// Old inline code (lines 78-146) replaced with:
localPath, githubURL, err := h.storeFile(req.ImageURL, assetType)
if err != nil {
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	return
}
```

Verify the full SaveAsset compiles correctly with this change.

- [ ] **Step 3: Verify build**

```bash
cd backend && go build ./...
go vet ./...
```

Expected: clean build, no errors.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/handler/asset.go
git commit -m "refactor(asset): extract storeFile() from SaveAsset for reuse"
```

---

### Task 2: Add `UpdateStoragePaths` to Repository

**Files:**
- Modify: `backend/internal/repository/interfaces.go`
- Modify: `backend/internal/repository/gorm/asset.go`

**Interfaces:**
- Consumes: `AssetRepository` interface (needs new method)
- Produces: `UpdateStoragePaths(id int64, localPath, githubURL string) error`

- [ ] **Step 1: Add method to interface**

In `backend/internal/repository/interfaces.go`, add to the `AssetRepository` interface:

```go
UpdateStoragePaths(id int64, localPath, githubURL string) error
```

Place it after `UpdateGithubURL` (line 111).

- [ ] **Step 2: Implement in GORM repository**

In `backend/internal/repository/gorm/asset.go`, add:

```go
func (r *AssetRepository) UpdateStoragePaths(id int64, localPath, githubURL string) error {
	if err := r.db.Model(&model.Asset{}).Where("id = ?", id).Updates(map[string]interface{}{
		"local_path": localPath,
		"github_url": githubURL,
	}).Error; err != nil {
		return fmt.Errorf("更新存储路径失败: %w", err)
	}
	return nil
}
```

- [ ] **Step 3: Verify build**

```bash
cd backend && go build ./...
go vet ./...
```

Expected: clean build (interface implementer `AssetRepository` compiles with the new method).

- [ ] **Step 4: Commit**

```bash
git add backend/internal/repository/interfaces.go backend/internal/repository/gorm/asset.go
git commit -m "feat(repo): add UpdateStoragePaths repository method"
```

---

### Task 3: Add `TransferAsset` Handler

**Files:**
- Modify: `backend/internal/handler/asset.go`

**Interfaces:**
- Consumes: `storeFile()`, `h.repo.GetByIDs()`, `h.repo.UpdateStoragePaths()`
- Produces: `TransferAsset(c *gin.Context)` — called by route `POST /api/v1/assets/:id/transfer`

- [ ] **Step 1: Add `TransferAsset` handler method**

Add after `SaveAsset` (before `ListAssets`):

```go
// TransferAsset 转存 — 根据当前存储设置补全 local_path / github_url
func (h *AssetHandler) TransferAsset(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: 无效的 ID"})
		return
	}

	assets, err := h.repo.GetByIDs([]int64{id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询作品失败: " + err.Error()})
		return
	}
	if len(assets) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "作品不存在"})
		return
	}
	asset := assets[0]

	// 如果 original_url 为空，无法转存
	if asset.OriginalURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "作品无原始链接，无法转存"})
		return
	}

	// 本地路径（非 URL）直接返回
	if !strings.HasPrefix(asset.OriginalURL, "http://") && !strings.HasPrefix(asset.OriginalURL, "https://") {
		c.JSON(http.StatusOK, gin.H{
			"id":         asset.ID,
			"local_path": asset.LocalPath,
			"github_url": asset.GitHubURL,
		})
		return
	}

	// 调用共享的 storeFile()
	localPath, githubURL, err := h.storeFile(asset.OriginalURL, asset.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "转存失败: " + err.Error()})
		return
	}

	// 更新数据库
	if err := h.repo.UpdateStoragePaths(asset.ID, localPath, githubURL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新存储路径失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         asset.ID,
		"local_path": localPath,
		"github_url": githubURL,
	})
}
```

- [ ] **Step 2: Verify build**

```bash
cd backend && go build ./...
go vet ./...
```

Expected: clean build.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/handler/asset.go
git commit -m "feat(asset): add TransferAsset handler for POST /assets/:id/transfer"
```

---

### Task 4: Register Route

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Add route registration**

In `backend/cmd/server/main.go`, after the existing asset routes (line 152), add:

```go
api.POST("/assets/:id/transfer", assetHandler.TransferAsset)
```

- [ ] **Step 2: Verify build**

```bash
cd backend && go build ./...
```

Expected: clean build.

- [ ] **Step 3: Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "feat(route): register POST /api/v1/assets/:id/transfer"
```

---

### Task 5: Update Frontend — Add `transferAsset` API and Update Assets View

**Files:**
- Modify: `frontend/src/api/assets.ts`
- Modify: `frontend/src/views/Assets.vue`

**Interfaces:**
- Consumes: `POST /api/v1/assets/:id/transfer` (new backend endpoint)
- Produces: `transferAsset(id: number)` API wrapper

- [ ] **Step 1: Add API function**

In `frontend/src/api/assets.ts`, add:

```typescript
export async function transferAsset(id: number): Promise<{ id: number; local_path: string; github_url: string }> {
  const res = await client.post(`/api/v1/assets/${id}/transfer`)
  return res.data
}
```

- [ ] **Step 2: Update Assets.vue drawer**

In `frontend/src/views/Assets.vue`:

Replace the import of `uploadToGitHub` with the new `transferAsset`:

```typescript
// Remove line 6: import { uploadToGitHub } from '../api/github'
// Add to line 5 (getAssets import):
import { getAssets, toggleFavorite, batchDownload, deleteAssets, transferAsset } from '../api/assets'
```

Replace `handleUploadToGitHub` function (lines 86-98) with:

```typescript
async function handleTransfer() {
  if (!detailAsset.value) return
  uploadingUrl.value = detailAsset.value.original_url
  try {
    const result = await transferAsset(detailAsset.value.id)
    if (detailAsset.value) {
      detailAsset.value.local_path = result.local_path
      detailAsset.value.github_url = result.github_url
    }
    if (result.github_url) {
      ElMessage.success(`已转存到 GitHub: ${result.github_url}`)
    } else if (result.local_path) {
      ElMessage.success(`已保存到本地: ${result.local_path}`)
    } else {
      ElMessage.success('转存完成')
    }
  } catch (e: any) {
    ElMessage.error(e.message || '转存失败')
  } finally {
    uploadingUrl.value = ''
  }
}
```

Update the template:

Replace the old 转存 button (lines 248-253):
```html
<el-button
  :loading="uploadingUrl === (detailAsset.original_url || '')"
  @click="handleTransfer"
>
  转存
</el-button>
```

- [ ] **Step 3: Check unused imports**

If `uploadToGitHub` was only imported for `handleUploadToGitHub`, remove the import line `import { uploadToGitHub } from '../api/github'` from `Assets.vue`. Also check if `Search, Star, ...` are still used — they should be.

- [ ] **Step 4: Verify frontend build**

```bash
cd frontend && pnpm build
```

Expected: clean typecheck + Vite build.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/api/assets.ts frontend/src/views/Assets.vue
git commit -m "feat(frontend): update Assets view to use new transfer endpoint"
```

---

### Task 6: Integration Test

**Files:** (no changes — manual verification)

- [ ] **Step 1: Start backend**

```bash
cd backend && go run ./cmd/server &
```

Expected: server starts on :8080 with no errors.

- [ ] **Step 2: Test SaveAsset (both mode)**

```bash
curl -s -X POST http://localhost:8080/api/v1/assets \
  -H 'Content-Type: application/json' \
  -d '{"image_url":"https://www.w3schools.com/w3images/fjords.jpg","prompt":"测试both","mode":"text2image"}'
```

Expected: `{"id":<N>}`. Verify DB has both `local_path` and `github_url`:
```bash
sqlite3 history.db "SELECT id, local_path, github_url FROM assets WHERE id = <N>;"
```

- [ ] **Step 3: Test TransferAsset**

```bash
# Create an asset with remote URL only (no local_path/github_url)
curl -s -X POST http://localhost:8080/api/v1/assets \
  -H 'Content-Type: application/json' \
  -d '{"image_url":"https://www.w3schools.com/w3images/lighthouse.jpg","prompt":"测试transfer","mode":"text2image"}'
# Note the returned id, then:
sqlite3 history.db "UPDATE assets SET local_path='', github_url='' WHERE id = <id>;"
# Now transfer:
curl -s -X POST http://localhost:8080/api/v1/assets/<id>/transfer
```

Expected: returns `{"id":<id>,"local_path":"outputs/...","github_url":"https://..."}`.

- [ ] **Step 4: Test edge cases**

```bash
# Non-existent ID
curl -s -X POST http://localhost:8080/api/v1/assets/99999/transfer
# Expected: 404 "作品不存在"

# Invalid ID
curl -s -X POST http://localhost:8080/api/v1/assets/abc/transfer
# Expected: 400 "参数错误: 无效的 ID"

# Local path asset (no remote URL)
sqlite3 history.db "UPDATE assets SET original_url='outputs/test.png', local_path='outputs/test.png' WHERE id = 1;"
curl -s -X POST http://localhost:8080/api/v1/assets/1/transfer
# Expected: 200 with existing paths
```

- [ ] **Step 5: Stop server and commit test evidence**

```bash
pkill -f "go run"
git add -A
git status
```
