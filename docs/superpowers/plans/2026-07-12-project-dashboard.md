# Project Dashboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add project progress management and file aggregation to creative projects — a dashboard page showing step progress, file grid, stats cards, and real-time task progress.

**Architecture:** Extend GORM models with `project_id` foreign keys (TaskRecord/History/Asset → Project), add 3 new API endpoints under `/api/v1/projects/:id/`, implement ProjectDashboard.vue with 4 sub-components (StepProgressBar, StatsCards, FileGrid, TaskProgressPanel), and wire step-progress tracking into ProjectEditor.

**Tech Stack:** Go 1.25 + Gin + GORM/SQLite · Vue 3 + Element Plus + TypeScript 6 · SSE

## Global Constraints

- All user-facing strings in Chinese (error messages, labels, placeholders)
- No auth middleware — API is local-only
- Use `gormrepo` package alias for GORM repo imports
- All `TableName()` methods return existing table names (no table renames)
- New fields use GORM AutoMigrate for schema changes (ALTER TABLE ADD COLUMN)
- TypeScript 6 `erasableSyntaxOnly` — no enums, use `as const` or union types
- Vue Composition API + `<script setup>` only
- SSE subscriber pattern via `TaskQueue` channel (max 10 buffered events)

---

## File Structure

| File | Status | Responsibility |
|------|--------|---------------|
| `backend/internal/repository/gorm/models.go` | Modify | Add `ProjectID` to History/Asset/TaskRecord, `StepProgress` to Project |
| `backend/internal/model/types.go` | Modify | Add `ProjectID` to TaskRecord/Asset, add `ProjectFile`/`ProjectStats` response types |
| `backend/internal/repository/interfaces.go` | Modify | Add `GetRecordsByProjectID`, `GetByProjectID`, `ListByProjectID` to repos |
| `backend/internal/repository/gorm/history.go` | Modify | Implement `GetRecordsByProjectID` |
| `backend/internal/repository/gorm/asset.go` | Modify | Implement `GetByProjectID` |
| `backend/internal/repository/gorm/task.go` | Modify | Implement `ListByProjectID` |
| `backend/internal/repository/gorm/project.go` | Modify | Add `UpdateField` method to ProjectRepository |
| `backend/internal/handler/project.go` | Modify | Add `GetProjectFiles`, `GetProjectStats`, `UpdateStepProgress` handlers |
| `backend/internal/handler/history.go` | Modify | Add `GetHistoryRepo()` getter, add `SetTaskRepo()`/`GetTaskRepo()` globals |
| `backend/cmd/server/main.go` | Modify | Wire taskRepo global, register new routes |
| `frontend/src/types/index.ts` | Modify | Add `ProjectFile`, `ProjectStats` interfaces, `step_progress` to Project, `project_id` to HistoryRecord |
| `frontend/src/api/projects.ts` | Modify | Add `getProjectFiles`, `getProjectStats`, `updateStepProgress` |
| `frontend/src/router/index.ts` | Modify | Add `/projects/:id/dashboard` route |
| `frontend/src/views/ProjectDashboard.vue` | Create | Main dashboard page — assembles all sub-components |
| `frontend/src/components/StepProgressBar.vue` | Create | 4-step progress indicator bar |
| `frontend/src/components/ProjectStatsCards.vue` | Create | Stats summary cards row |
| `frontend/src/components/ProjectFileGrid.vue` | Create | File thumbnail grid with tab filters |
| `frontend/src/views/ProjectEditor.vue` | Modify | Call `updateStepProgress` on step transitions, add dashboard button |
| `frontend/src/views/ProjectList.vue` | Modify | Show file count + last activity time per project |

---

### Task 1: Backend Data Model — Add ProjectID + StepProgress fields

**Files:**
- Modify: `backend/internal/repository/gorm/models.go` — add fields to History, Asset, TaskRecord, Project
- Modify: `backend/internal/model/types.go` — add ProjectID to Asset, TaskRecord
- Modify: `frontend/src/types/index.ts` — add step_progress + project_id fields

**Interfaces:**
- Consumes: existing struct definitions
- Produces: `gorm.History.ProjectID`, `gorm.Asset.ProjectID`, `gorm.TaskRecord.ProjectID`, `gorm.Project.StepProgress`, `model.Asset.ProjectID`, `model.TaskRecord.ProjectID`

- [ ] **Step 1: Add ProjectID to gorm History**

```go
// In backend/internal/repository/gorm/models.go, after line 11 (Extra field):
type History struct {
	ID     int64   `gorm:"primaryKey"`
	Time   string  `gorm:"index"`
	Mode   string  `gorm:"index"`
	Prompt string
	Images string  // JSON array
	Extra  *string
	ProjectID int64 `json:"project_id" gorm:"column:project_id;index;default:0"`
}
```

- [ ] **Step 2: Add ProjectID to gorm Asset**

```go
// In same file, add after line 65 (GitHubURL field):
type Asset struct {
	ID          int64  `gorm:"primaryKey"`
	Mode        string `gorm:"index"`
	Prompt      string
	Type        string
	Time        string
	Favorite    bool
	OriginalURL string `gorm:"column:original_url"`
	LocalPath   string `gorm:"column:local_path"`
	GitHubURL   string `gorm:"column:github_url"`
	ProjectID   int64  `json:"project_id" gorm:"column:project_id;index;default:0"`
}
```

- [ ] **Step 3: Add ProjectID to gorm TaskRecord**

```go
// In same file, add after line 97 (CompletedAt field):
type TaskRecord struct {
	ID          int64   `gorm:"primaryKey"`
	Type        string  `gorm:"index"`
	Status      string  `gorm:"index"`
	Params      string  `gorm:"type:text"`
	Result      *string `gorm:"type:text"`
	Progress    int
	Error       *string `gorm:"type:text"`
	RetryCount  int     `gorm:"column:retry_count"`
	CreatedAt   string  `gorm:"column:created_at"`
	UpdatedAt   string  `gorm:"column:updated_at"`
	CompletedAt *string `gorm:"column:completed_at"`
	ProjectID   int64   `json:"project_id" gorm:"column:project_id;index;default:0"`
}
```

- [ ] **Step 4: Add StepProgress to gorm Project**

```go
// In same file, add after line 146 (Notes field):
type Project struct {
	ID        int64         `gorm:"primaryKey" json:"id"`
	Title     string        `gorm:"size:200" json:"title"`
	Brief     string        `gorm:"type:text" json:"brief"`
	AIResult  string        `gorm:"type:text" json:"ai_result"`
	Status    string        `gorm:"size:20;default:draft" json:"status"`
	CoverURL  string        `gorm:"type:text" json:"cover_url"`
	FinalURL  string        `gorm:"type:text" json:"final_url"`
	AssetIDs  string        `gorm:"type:text" json:"asset_ids"`
	Notes     string        `gorm:"type:text" json:"notes"`
	StepProgress string      `gorm:"type:text" json:"step_progress"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	Steps     []ProjectStep `gorm:"foreignKey:ProjectID" json:"steps"`
}
```

- [ ] **Step 5: Add ProjectID to model.Asset**

```go
// In backend/internal/model/types.go, add after line 201 (GitHubURL field):
type Asset struct {
	ID          int64  `json:"id" gorm:"primaryKey"`
	Mode        string `json:"mode" gorm:"index"`
	Prompt      string `json:"prompt"`
	Type        string `json:"type"`
	Time        string `json:"time"`
	Favorite    bool   `json:"favorite"`
	OriginalURL string `json:"original_url" gorm:"column:original_url"`
	LocalPath   string `json:"local_path" gorm:"column:local_path"`
	GitHubURL   string `json:"github_url" gorm:"column:github_url"`
	ProjectID   int64  `json:"project_id" gorm:"column:project_id;index;default:0"`
}
```

- [ ] **Step 6: Add ProjectID to model.TaskRecord**

```go
// In backend/internal/model/types.go, add after line 299 (CompletedAt field):
type TaskRecord struct {
	ID          int64  `json:"id"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Params      string `json:"params"`
	Result      string `json:"result,omitempty"`
	Progress    int    `json:"progress"`
	Error       string `json:"error,omitempty"`
	RetryCount  int    `json:"retry_count"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	CompletedAt string `json:"completed_at,omitempty"`
	ProjectID   int64  `json:"project_id"`
}
```

- [ ] **Step 7: Add frontend ProjectFile + ProjectStats types + update Project + update HistoryRecord**

```typescript
// In frontend/src/types/index.ts, add after the existing types:

export interface ProjectFile {
  id: number
  type: 'image' | 'video'
  source: 'history' | 'asset'
  url: string
  prompt: string
  step: string
  created_at: string
}

export interface ProjectStats {
  file_count: number
  optimized_count: number
  running_tasks: number
  last_activity: string
  step_progress: Record<string, string>
}

// Update Project interface — add step_progress after notes (line 217):
export interface Project {
  id: number
  title: string
  brief: string
  ai_result: string
  status: 'draft' | 'generating' | 'refining' | 'completed'
  cover_url: string
  final_url: string
  asset_ids: string
  notes: string
  step_progress: string  // JSON string {"ideate":"completed",...}
  created_at: string
  updated_at: string
  steps: ProjectStep[]
}

// Update HistoryRecord — add project_id after extra (line 67):
export interface HistoryRecord {
  id: number
  time: string
  mode: string
  prompt: string
  images: string[]
  extra?: Record<string, unknown>
  project_id?: number
}
```

- [ ] **Step 8: Build verification**

```bash
cd backend && go build ./cmd/server
cd ../frontend && pnpm build
```
Expected: both exit 0

---

### Task 2: Backend API — Repository methods + Handler endpoints

**Files:**
- Modify: `backend/internal/repository/interfaces.go` — add query-by-projectID methods
- Modify: `backend/internal/repository/gorm/history.go` — implement `GetRecordsByProjectID`
- Modify: `backend/internal/repository/gorm/asset.go` — implement `GetByProjectID`
- Modify: `backend/internal/repository/gorm/task.go` — implement `ListByProjectID`
- Modify: `backend/internal/repository/gorm/project.go` — add `UpdateField` method
- Modify: `backend/internal/handler/project.go` — add 3 new handlers + response types
- Modify: `backend/internal/handler/history.go` — add `GetHistoryRepo()` + `SetTaskRepo()`/`GetTaskRepo()`
- Modify: `backend/cmd/server/main.go` — wire taskRepo global, register new routes

**Interfaces:**
- Consumes: `model.HistoryRecord`, `model.Asset`, `model.TaskRecord` (with ProjectID fields from Task 1)
- Produces: new repository methods, new handler methods, new routes

- [ ] **Step 1: Add repository interface methods to interfaces.go**

```go
// In backend/internal/repository/interfaces.go, add to HistoryRepository:
GetRecordsByProjectID(projectID int64) ([]model.HistoryRecord, error)

// Add to AssetRepository:
GetByProjectID(projectID int64) ([]model.Asset, error)

// Add to TaskRepository:
ListByProjectID(projectID int64) ([]*model.TaskRecord, error)

// Add to ProjectRepository (create if not exist — check first):
// If ProjectRepository interface already exists, just add:
UpdateField(id int64, field, value string) error
// If it doesn't exist, add the interface. But project.go uses concrete type,
// so we can skip the interface method and just add directly to the concrete type.
```

- [ ] **Step 2: Implement GetRecordsByProjectID in gorm/history.go**

```go
func (r *HistoryRepository) GetRecordsByProjectID(projectID int64) ([]model.HistoryRecord, error) {
	var records []History
	if err := r.db.Where("project_id = ?", projectID).Find(&records).Error; err != nil {
		return nil, err
	}
	result := make([]model.HistoryRecord, len(records))
	for i, h := range records {
		var images []string
		if h.Images != "" {
			json.Unmarshal([]byte(h.Images), &images)
		}
		var extra any
		if h.Extra != nil {
			json.Unmarshal([]byte(*h.Extra), &extra)
		} else {
			extra = nil
		}
		result[i] = model.HistoryRecord{
			ID:     h.ID,
			Time:   h.Time,
			Mode:   h.Mode,
			Prompt: h.Prompt,
			Images: images,
			Extra:  extra,
		}
	}
	return result, nil
}
```

Note: The gorm History struct in models.go is `History` (no package alias needed within the gorm package). Ensure `encoding/json` is already imported in `history.go`.

- [ ] **Step 3: Implement GetByProjectID in gorm/asset.go**

```go
func (r *AssetRepository) GetByProjectID(projectID int64) ([]model.Asset, error) {
	var assets []model.Asset
	if err := r.db.Model(&model.Asset{}).Where("project_id = ?", projectID).Find(&assets).Error; err != nil {
		return nil, err
	}
	return assets, nil
}
```

- [ ] **Step 4: Implement ListByProjectID in gorm/task.go**

```go
func (r *TaskRepository) ListByProjectID(projectID int64) ([]*model.TaskRecord, error) {
	var gormTasks []TaskRecord
	if err := r.db.Where("project_id = ?", projectID).Order("created_at DESC").Find(&gormTasks).Error; err != nil {
		return nil, err
	}
	result := make([]*model.TaskRecord, len(gormTasks))
	for i, t := range gormTasks {
		result[i] = toTaskRecord(&t)
	}
	return result, nil
}
```

- [ ] **Step 5: Implement UpdateField in gorm/project.go**

```go
func (r *ProjectRepository) UpdateField(id int64, field, value string) error {
	return r.db.Model(&Project{}).Where("id = ?", id).Update(field, value).Error
}
```

- [ ] **Step 6: Add GetHistoryRepo() + SetTaskRepo()/GetTaskRepo() globals in handler package**

```go
// In backend/internal/handler/history.go, add after SetHistoryRepo:
func GetHistoryRepo() repository.HistoryRepository {
	return historyRepo
}

// Add new globals:
var taskRepo repository.TaskRepository

func SetTaskRepo(repo repository.TaskRepository) {
	taskRepo = repo
}

func GetTaskRepo() repository.TaskRepository {
	return taskRepo
}
```

Also add `"github.com/agnes-image-tool/backend/internal/repository"` import if not already present.

- [ ] **Step 7: Add response types + new handlers in project.go**

Add after existing `createProjectRequest` struct (after line 29):

```go
type ProjectFileItem struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`      // "image" | "video"
	Source    string `json:"source"`    // "history" | "asset"
	URL       string `json:"url"`
	Prompt    string `json:"prompt"`
	Step      string `json:"step"`
	CreatedAt string `json:"created_at"`
}

type ProjectStatsResponse struct {
	FileCount      int               `json:"file_count"`
	OptimizedCount int               `json:"optimized_count"`
	RunningTasks   int               `json:"running_tasks"`
	LastActivity   string            `json:"last_activity"`
	StepProgress   map[string]string `json:"step_progress"`
}

type UpdateStepProgressRequest struct {
	Step   string `json:"step" binding:"required"`
	Status string `json:"status" binding:"required"`
}
```

Add these imports to project.go if not present:
```go
"sort"
"github.com/agnes-image-tool/backend/internal/model"
"github.com/agnes-image-tool/backend/internal/repository"
```

Note: check if `model` and `repository` imports already exist. If `handler` package globals are used via package-level functions (no import needed since same package), the `repository` import is needed only if explicit.

- [ ] **Step 8: Implement GetProjectFiles handler**

```go
// GetProjectFiles 获取项目关联的所有文件
func (h *ProjectHandler) GetProjectFiles(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目 ID"})
		return
	}

	var files []ProjectFileItem

	// 从 History 中查询
	if histRepo := GetHistoryRepo(); histRepo != nil {
		records, err := histRepo.GetRecordsByProjectID(id)
		if err == nil {
			for _, r := range records {
				fileType := "image"
				if len(r.Mode) >= 5 && r.Mode[:5] == "video" {
					fileType = "video"
				}
				for _, img := range r.Images {
					files = append(files, ProjectFileItem{
						ID:        r.ID,
						Type:      fileType,
						Source:    "history",
						URL:       img,
						Prompt:    r.Prompt,
						Step:      "",
						CreatedAt: r.Time,
					})
				}
			}
		}
	}

	// 从 Asset 中查询
	if assetRepo := GetAssetRepo(); assetRepo != nil {
		assets, err := assetRepo.GetByProjectID(id)
		if err == nil {
			for _, a := range assets {
				url := a.OriginalURL
				if a.LocalPath != "" {
					url = a.LocalPath
				}
				fileType := a.Type
				if fileType == "" {
					fileType = "image"
				}
				files = append(files, ProjectFileItem{
					ID:        a.ID,
					Type:      fileType,
					Source:    "asset",
					URL:       url,
					Prompt:    a.Prompt,
					Step:      "",
					CreatedAt: a.Time,
				})
			}
		}
	}

	// 按时间排序（最新的在前）
	sort.Slice(files, func(i, j int) bool {
		return files[i].CreatedAt > files[j].CreatedAt
	})

	c.JSON(http.StatusOK, gin.H{"files": files})
}
```

- [ ] **Step 9: Implement GetProjectStats handler + parseStepProgress helper**

```go
// GetProjectStats 获取项目统计
func (h *ProjectHandler) GetProjectStats(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目 ID"})
		return
	}

	project, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
		return
	}

	resp := ProjectStatsResponse{
		StepProgress: parseStepProgress(project.StepProgress),
	}

	// 文件数
	if histRepo := GetHistoryRepo(); histRepo != nil {
		records, err := histRepo.GetRecordsByProjectID(id)
		if err == nil {
			for _, r := range records {
				resp.FileCount += len(r.Images)
			}
		}
	}
	if assetRepo := GetAssetRepo(); assetRepo != nil {
		assets, err := assetRepo.GetByProjectID(id)
		if err == nil {
			resp.FileCount += len(assets)
			resp.OptimizedCount = len(assets)
		}
	}

	// 运行中任务 + 最后活动时间
	if taskRepo := GetTaskRepo(); taskRepo != nil {
		tasks, err := taskRepo.ListByProjectID(id)
		if err == nil {
			for _, t := range tasks {
				if t.Status == "pending" || t.Status == "processing" {
					resp.RunningTasks++
				}
				if t.CreatedAt > resp.LastActivity {
					resp.LastActivity = t.CreatedAt
				}
			}
		}
	}

	c.JSON(http.StatusOK, resp)
}

// parseStepProgress 解析 StepProgress JSON 字符串
func parseStepProgress(s string) map[string]string {
	result := map[string]string{
		"ideate":   "pending",
		"generate": "pending",
		"refine":   "pending",
		"finalize": "pending",
	}
	if s == "" {
		return result
	}
	json.Unmarshal([]byte(s), &result)
	return result
}
```

- [ ] **Step 10: Implement UpdateStepProgress handler**

```go
// UpdateStepProgress 更新项目步骤进度
func (h *ProjectHandler) UpdateStepProgress(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目 ID"})
		return
	}

	var req UpdateStepProgressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	project, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
		return
	}

	progress := parseStepProgress(project.StepProgress)
	progress[req.Step] = req.Status

	progressJSON, _ := json.Marshal(progress)
	if err := h.repo.UpdateField(id, "step_progress", string(progressJSON)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新进度失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
```

- [ ] **Step 11: Register new routes + wire taskRepo in main.go**

```go
// In backend/cmd/server/main.go, after line 88 (taskQueue creation):
handler.SetTaskRepo(taskRepo)

// In the projects route group after the existing routes (after line 210):
projects.GET("/:id/files", projectHandler.GetProjectFiles)
projects.GET("/:id/stats", projectHandler.GetProjectStats)
projects.PUT("/:id/step-progress", projectHandler.UpdateStepProgress)
```

- [ ] **Step 12: Build verification**

```bash
cd backend && go build ./cmd/server
```
Expected: exit 0

---

### Task 3: Frontend API Client + Types + Router

**Files:**
- Modify: `frontend/src/api/projects.ts` — add 3 new API functions
- Modify: `frontend/src/router/index.ts` — add dashboard route
- Modify: `frontend/src/App.vue` — import ProjectDashboard

**Interfaces:**
- Consumes: `ProjectFile`, `ProjectStats` types from Task 1
- Produces: `getProjectFiles(id)`, `getProjectStats(id)`, `updateStepProgress(id, step, status)`

- [ ] **Step 1: Add API functions to projects.ts**

```typescript
// Add after addStep function:
import type { ProjectFile, ProjectStats } from '../types'

export async function getProjectFiles(id: number): Promise<ProjectFile[]> {
  const res = await client.get(`/api/v1/projects/${id}/files`)
  return res.data.files
}

export async function getProjectStats(id: number): Promise<ProjectStats> {
  const res = await client.get(`/api/v1/projects/${id}/stats`)
  return res.data
}

export async function updateStepProgress(
  id: number,
  step: string,
  status: string
): Promise<void> {
  await client.put(`/api/v1/projects/${id}/step-progress`, { step, status })
}
```

- [ ] **Step 2: Add dashboard route to router/index.ts**

```typescript
// Add to routes array:
{
  path: '/projects/:id/dashboard',
  name: 'ProjectDashboard',
  component: () => import('../views/ProjectDashboard.vue')
}
```

- [ ] **Step 3: Import ProjectDashboard in App.vue**

```typescript
// Add to the imports section:
import ProjectDashboard from './views/ProjectDashboard.vue'
```

No changes needed if using Vue Router dynamic import — the component is lazy-loaded.

- [ ] **Step 4: Build verification**

```bash
cd frontend && pnpm build
```
Expected: exit 0 (pre-existing INEFFECTIVE_DYNAMIC_IMPORT warnings expected)

---

### Task 4: Frontend Dashboard Components

**Files:**
- Create: `frontend/src/components/StepProgressBar.vue` — 4-step progress indicator
- Create: `frontend/src/components/ProjectStatsCards.vue` — stats summary cards
- Create: `frontend/src/components/ProjectFileGrid.vue` — file thumbnail grid

**Interfaces:**
- Consumes: `Record<string, string>` for step_progress, `ProjectFile[]`, `ProjectStats`
- Produces: ready-to-use Vue components with defined props/emits

- [ ] **Step 1: Create StepProgressBar.vue**

```vue
<script setup lang="ts">
const props = defineProps<{
  steps: { key: string; label: string; status: string }[]
}>()

const statusIcon = (status: string): string => {
  if (status === 'completed') return 'el-icon-success'
  if (status === 'in_progress') return 'el-icon-loading'
  return 'el-icon-minus'
}

const statusColor = (status: string): string => {
  if (status === 'completed') return 'var(--el-color-success)'
  if (status === 'in_progress') return 'var(--el-color-primary)'
  return 'var(--el-color-info)'
}
</script>

<template>
  <div class="step-progress-bar">
    <div
      v-for="(step, index) in steps"
      :key="step.key"
      class="step-item"
      :class="{ active: step.status === 'in_progress', done: step.status === 'completed' }"
    >
      <div class="step-indicator" :style="{ borderColor: statusColor(step.status) }">
        <el-icon v-if="step.status === 'completed'" :color="statusColor(step.status)">
          <Check />
        </el-icon>
        <span v-else-if="step.status === 'in_progress'" class="step-num">{{ index + 1 }}</span>
        <span v-else class="step-num">{{ index + 1 }}</span>
      </div>
      <span class="step-label" :style="{ color: statusColor(step.status) }">{{ step.label }}</span>
      <div v-if="index < steps.length - 1" class="step-line" :class="{ done: step.status === 'completed' }" />
    </div>
  </div>
</template>

<style scoped>
.step-progress-bar {
  display: flex;
  align-items: center;
  gap: 0;
  padding: 16px 0;
}
.step-item {
  display: flex;
  align-items: center;
  position: relative;
  gap: 8px;
}
.step-indicator {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  border: 2px solid var(--el-color-info);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  font-weight: 600;
  background: #fff;
  flex-shrink: 0;
}
.step-item.done .step-indicator {
  background: var(--el-color-success);
  border-color: var(--el-color-success);
  color: #fff;
}
.step-item.active .step-indicator {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
}
.step-num {
  font-size: 13px;
  line-height: 1;
}
.step-label {
  font-size: 13px;
  white-space: nowrap;
}
.step-line {
  width: 60px;
  height: 2px;
  background: var(--el-color-info-light-5);
  margin: 0 12px;
  flex-shrink: 0;
}
.step-line.done {
  background: var(--el-color-success);
}
</style>
```

Note: Add `import { Check } from '@element-plus/icons-vue'` to the script section. The `el-icon` components need the icon registered.

- [ ] **Step 2: Create ProjectStatsCards.vue**

```vue
<script setup lang="ts">
import type { ProjectStats } from '../types'
import { Picture, VideoCamera, Clock, Warning } from '@element-plus/icons-vue'

defineProps<{ stats: ProjectStats | null }>()
</script>

<template>
  <div class="stats-cards">
    <el-card shadow="hover" class="stat-card">
      <el-icon :size="24" color="var(--el-color-primary)"><Picture /></el-icon>
      <div class="stat-info">
        <span class="stat-value">{{ stats?.file_count ?? '-' }}</span>
        <span class="stat-label">文件总数</span>
      </div>
    </el-card>
    <el-card shadow="hover" class="stat-card">
      <el-icon :size="24" color="var(--el-color-success)"><VideoCamera /></el-icon>
      <div class="stat-info">
        <span class="stat-value">{{ stats?.optimized_count ?? '-' }}</span>
        <span class="stat-label">优化次数</span>
      </div>
    </el-card>
    <el-card shadow="hover" class="stat-card">
      <el-icon :size="24" color="var(--el-color-warning)"><Warning /></el-icon>
      <div class="stat-info">
        <span class="stat-value">{{ stats?.running_tasks ?? '-' }}</span>
        <span class="stat-label">进行中任务</span>
      </div>
    </el-card>
    <el-card shadow="hover" class="stat-card">
      <el-icon :size="24" color="var(--el-color-info)"><Clock /></el-icon>
      <div class="stat-info">
        <span class="stat-value">{{ stats?.last_activity ? stats.last_activity.slice(0, 16).replace('T', ' ') : '-' }}</span>
        <span class="stat-label">最后活动</span>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.stats-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 20px;
}
.stat-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px;
}
.stat-card :deep(.el-card__body) {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
}
.stat-info {
  display: flex;
  flex-direction: column;
}
.stat-value {
  font-size: 22px;
  font-weight: 700;
  line-height: 1.2;
}
.stat-label {
  font-size: 12px;
  color: var(--el-color-info);
}
</style>
```

- [ ] **Step 3: Create ProjectFileGrid.vue**

```vue
<script setup lang="ts">
import { ref, computed } from 'vue'
import type { ProjectFile } from '../types'
import { Download, Eye } from '@element-plus/icons-vue'

const props = defineProps<{ files: ProjectFile[] }>()
const emit = defineEmits<{ preview: [file: ProjectFile] }>()

const activeTab = ref<'all' | 'image' | 'video'>('all')

const filteredFiles = computed(() => {
  if (activeTab.value === 'all') return props.files
  return props.files.filter(f => f.type === activeTab.value)
})

const stepLabel: Record<string, string> = {
  generate: '生成',
  refine: '优化',
  finalize: '定稿',
}

function getThumbnailUrl(file: ProjectFile): string {
  if (file.type === 'video') {
    return file.url.replace(/\.(mp4|webm)$/, '.jpg') || '/placeholder-video.png'
  }
  return file.url
}

function copyUrl(url: string) {
  navigator.clipboard.writeText(url).then(() => {
    ElMessage.success('链接已复制')
  }).catch(() => {})
}
</script>

<template>
  <div class="file-grid-panel">
    <div class="file-grid-header">
      <el-radio-group v-model="activeTab" size="small">
        <el-radio-button value="all">全部 ({{ files.length }})</el-radio-button>
        <el-radio-button value="image">图片 ({{ files.filter(f => f.type === 'image').length }})</el-radio-button>
        <el-radio-button value="video">视频 ({{ files.filter(f => f.type === 'video').length }})</el-radio-button>
      </el-radio-group>
    </div>

    <div v-if="filteredFiles.length === 0" class="file-grid-empty">
      <el-empty description="暂无文件" />
    </div>

    <div v-else class="file-grid">
      <div v-for="file in filteredFiles" :key="`${file.source}-${file.id}`" class="file-card">
        <div class="file-thumb" @click="emit('preview', file)">
          <img :src="getThumbnailUrl(file)" :alt="file.prompt" @error="(e: any) => e.target.src = '/placeholder.png'" />
          <span v-if="file.step" class="file-step-tag" :class="file.step">{{ stepLabel[file.step] || file.step }}</span>
          <span v-if="file.type === 'video'" class="file-type-badge">视频</span>
        </div>
        <div class="file-info">
          <p class="file-prompt" :title="file.prompt">{{ file.prompt }}</p>
          <div class="file-actions">
            <el-tooltip content="预览">
              <el-button size="small" circle :icon="Eye" @click="emit('preview', file)" />
            </el-tooltip>
            <el-tooltip content="复制链接">
              <el-button size="small" circle :icon="Download" @click="copyUrl(file.url)" />
            </el-tooltip>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.file-grid-panel {
  margin-top: 8px;
}
.file-grid-header {
  margin-bottom: 16px;
}
.file-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 16px;
}
.file-card {
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  overflow: hidden;
  transition: box-shadow 0.2s;
  background: #fff;
}
.file-card:hover {
  box-shadow: 0 2px 12px rgba(0,0,0,0.08);
}
.file-thumb {
  position: relative;
  aspect-ratio: 1;
  overflow: hidden;
  cursor: pointer;
  background: var(--el-color-info-light-9);
}
.file-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.file-step-tag {
  position: absolute;
  top: 6px;
  left: 6px;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  color: #fff;
  background: var(--el-color-primary);
}
.file-type-badge {
  position: absolute;
  top: 6px;
  right: 6px;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  color: #fff;
  background: var(--el-color-warning);
}
.file-info {
  padding: 8px 10px;
}
.file-prompt {
  font-size: 12px;
  line-height: 1.4;
  margin: 0 0 8px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  color: var(--el-text-color-secondary);
}
.file-actions {
  display: flex;
  gap: 4px;
  justify-content: flex-end;
}
.file-grid-empty {
  padding: 40px 0;
}
</style>
```

Note: Add `import { ElMessage } from 'element-plus'` to the script section.

- [ ] **Step 4: Build verification**

```bash
cd frontend && pnpm build
```
Expected: exit 0

---

### Task 5: Frontend ProjectDashboard Page

**Files:**
- Create: `frontend/src/views/ProjectDashboard.vue` — main dashboard page

**Interfaces:**
- Consumes: all 3 components from Task 4, API functions from Task 3
- Produces: full dashboard page viewable at `/projects/:id/dashboard`

- [ ] **Step 1: Create ProjectDashboard.vue**

```vue
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft } from '@element-plus/icons-vue'
import { getProject, getProjectFiles, getProjectStats } from '../api/projects'
import type { Project, ProjectFile, ProjectStats } from '../types'
import StepProgressBar from '../components/StepProgressBar.vue'
import ProjectStatsCards from '../components/ProjectStatsCards.vue'
import ProjectFileGrid from '../components/ProjectFileGrid.vue'

const route = useRoute()
const router = useRouter()
const projectId = Number(route.params.id)

const project = ref<Project | null>(null)
const files = ref<ProjectFile[]>([])
const stats = ref<ProjectStats | null>(null)
const loading = ref(true)

const stepConfig = [
  { key: 'ideate', label: '创意发想', status: 'pending' },
  { key: 'generate', label: '生成', status: 'pending' },
  { key: 'refine', label: '优化', status: 'pending' },
  { key: 'finalize', label: '定稿', status: 'pending' },
]

async function loadData() {
  loading.value = true
  try {
    const [p, f, s] = await Promise.all([
      getProject(projectId),
      getProjectFiles(projectId),
      getProjectStats(projectId),
    ])
    project.value = p
    files.value = f
    stats.value = s

    // Update step statuses from API response
    if (s.step_progress) {
      stepConfig.forEach(st => {
        st.status = s.step_progress[st.key] || 'pending'
      })
    }
  } catch (e: any) {
    ElMessage.error('加载仪表盘失败: ' + (e.message || ''))
  } finally {
    loading.value = false
  }
}

function goBack() {
  router.push(`/projects/${projectId}`)
}

function onPreview(file: ProjectFile) {
  window.open(file.url, '_blank')
}

onMounted(loadData)
</script>

<template>
  <div class="dashboard">
    <!-- 顶部导航 -->
    <div class="dashboard-header">
      <el-button text :icon="ArrowLeft" @click="goBack">返回编辑器</el-button>
      <h2 v-if="project" class="dashboard-title">{{ project.title }} — 仪表盘</h2>
      <el-tag v-if="project" :type="project.status === 'completed' ? 'success' : 'warning'">
        {{ project.status === 'completed' ? '已完成' : project.status === 'generating' ? '生成中' : '草稿' }}
      </el-tag>
    </div>

    <div v-loading="loading" class="dashboard-content">
      <!-- 步骤进度 -->
      <el-card shadow="never" class="section-card">
        <template #header><span>步骤进度</span></template>
        <StepProgressBar :steps="stepConfig" />
      </el-card>

      <!-- 统计卡片 -->
      <ProjectStatsCards :stats="stats" />

      <!-- 文件网格 -->
      <el-card shadow="never" class="section-card">
        <template #header><span>生成文件</span></template>
        <ProjectFileGrid :files="files" @preview="onPreview" />
      </el-card>
    </div>
  </div>
</template>

<style scoped>
.dashboard {
  padding: 20px;
  max-width: 1200px;
  margin: 0 auto;
}
.dashboard-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
}
.dashboard-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  flex: 1;
}
.dashboard-content {
  min-height: 400px;
}
.section-card {
  margin-bottom: 20px;
}
</style>
```

- [ ] **Step 2: Build verification**

```bash
cd frontend && pnpm build
```
Expected: exit 0

---

### Task 6: Integration — ProjectEditor + ProjectList Enhancement

**Files:**
- Modify: `frontend/src/views/ProjectEditor.vue` — add "查看仪表盘" button, call `updateStepProgress` on step transitions
- Modify: `frontend/src/views/ProjectList.vue` — show file count + last activity time per project

**Interfaces:**
- Consumes: `updateStepProgress` from projects.ts, `Project` with enhanced cards
- Produces: integrated feature visible in browser

- [ ] **Step 1: Add dashboard button + step-progress calls in ProjectEditor.vue**

Find the step transition logic. In `ProjectEditor.vue`, there's a `nextStep()` function that advances `currentStep`. After each step transition, call `updateStepProgress`.

```typescript
// In ProjectEditor.vue, after the existing imports, add:
import { updateStepProgress } from '../api/projects'

// The steps array is ['ideate', 'generate', 'refine', 'finalize']
// In the nextStep() function, add before advancing:
function nextStep() {
  const idx = steps.indexOf(currentStep.value)
  if (idx < steps.length - 1) {
    // Mark current step as completed before advancing
    updateStepProgress(props.projectId, currentStep.value, 'completed').catch(() => {})
    // Mark next step as in_progress
    const nextStepName = steps[idx + 1]
    updateStepProgress(props.projectId, nextStepName, 'in_progress').catch(() => {})
    currentStep.value = nextStepName
  }
}

// In the StepProgressBar area (or the header), add a dashboard link button:
// Add near the step header:
<el-button size="small" text @click="goDashboard">📊 查看仪表盘</el-button>

// Add the goDashboard function:
function goDashboard() {
  window.open(`/projects/${props.projectId}/dashboard`, '_blank')
}
```

Note: Use `import { ArrowRight, Picture, Edit, Check, ChatLineSquare } from '@element-plus/icons-vue'` — existing icons. The dashboard button text should be `查看仪表盘` (no emoji — use an el-icon instead):

```html
<el-button size="small" text @click="goDashboard">
  <el-icon><DataAnalysis /></el-icon> 查看仪表盘
</el-button>
```

Add `DataAnalysis` to the icon imports.

- [ ] **Step 2: Enhance ProjectList.vue with richer project cards**

Locate the project card rendering in `ProjectList.vue`. Currently it shows basic title/status. Enhance to show file count hint and last activity:

```typescript
// Add imports:
import { getProjectStats } from '../api/projects'
import type { ProjectStats } from '../types'

// Add stats mapping:
const projectStats = ref<Record<number, ProjectStats>>({})

// After loading projects, load stats for each:
async function loadStats() {
  for (const p of projects.value) {
    try {
      const s = await getProjectStats(p.id)
      projectStats.value[p.id] = s
    } catch {}
  }
}
```

Or simpler: modify the project card template to show a status indicator with step progress:

```html
<!-- Inside the project card template, after the status tag, add: -->
<div v-if="projectStats[project.id]" class="project-meta">
  <span class="meta-item">{{ projectStats[project.id].file_count }} 个文件</span>
  <span v-if="projectStats[project.id].last_activity" class="meta-item">
    {{ projectStats[project.id].last_activity.slice(0, 10) }}
  </span>
</div>
```

- [ ] **Step 3: Full build verification**

```bash
cd frontend && pnpm build
cd ../backend && go build ./cmd/server
```
Expected: both exit 0

---

## Self-Review (run after writing)

1. **Spec coverage:** Skim each section/requirement in the spec. Can you point to a task that implements it? List any gaps.
   - §1 Data model → Task 1
   - §2 GET /files → Task 2 (handler) + Task 3 (frontend API) + Task 5 (page)
   - §2 GET /stats → Task 2 + Task 3 + Task 5
   - §2 PUT /step-progress → Task 2 + Task 3 + Task 6 (ProjectEditor integration)
   - §3.1 Dashboard page → Task 5 + Task 4 (components)
   - §3.2 ProjectEditor integration → Task 6
   - §3.3 ProjectList enhancement → Task 6
   - §3.4 API client → Task 3
   - §3.5 Types → Task 1 (Step 7) + Task 3
   - §4 Route → Task 3 (router) + Task 5 (page)
   - §5 Status enum → Task 2 (parseStepProgress defaults)

2. **Placeholder scan:** Search for "TBD", "TODO", "Add appropriate error handling" patterns. None should exist.

3. **Type consistency:** Verify function signatures match across tasks:
   - `getProjectFiles(id)` → `ProjectFile[]` — consistent in Task 3 and Task 5
   - `getProjectStats(id)` → `ProjectStats` — consistent in Task 3, Task 5, Task 6
   - `updateStepProgress(id, step, status)` → `void` — consistent in Task 3 and Task 6
   - `GetRecordsByProjectID(projectID)` → `[]model.HistoryRecord` — consistent in Task 2
   - `GetByProjectID(projectID)` → `[]model.Asset` — consistent in Task 2
   - `ListByProjectID(projectID)` → `[]*model.TaskRecord` — consistent in Task 2
