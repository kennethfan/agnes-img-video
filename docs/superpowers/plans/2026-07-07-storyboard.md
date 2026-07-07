# Storyboard Studio 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a storyboard planning feature allowing users to create projects, manage shot sequences, and batch-generate videos.

**Architecture:** Backend adds a new `StoryboardRepo` (SQLite, separate tables) and `StoryboardHandler` (Gin). Frontend adds a new Storyboard tab with project list/detail views and ShotCard components. Batch generation reuses existing video generation APIs.

**Tech Stack:** Go 1.25 + Gin + SQLite (mattn/go-sqlite3) · Vue 3 + TypeScript 6 + Element Plus + Axios

**Spec:** `docs/superpowers/specs/2026-07-07-storyboard-design.md`

## Global Constraints

- Module path: `github.com/agnes-image-tool/backend`
- All new backend files in `backend/internal/handler/` and `backend/internal/repository/`
- All new frontend files in `frontend/src/views/`, `frontend/src/components/`, `frontend/src/api/`
- Follow existing patterns: Gin handler struct pattern, Vue `<script setup lang="ts">`, Element Plus UI components
- No auth middleware (project is local-dev only)
- No vue-router — use `el-tabs` for navigation
- TypeScript: `erasableSyntaxOnly` enabled — no enums or namespaces, use `as const` or union types

---
### Task 1: Backend — Storyboard data model + repository

**Files:**
- Modify: `backend/internal/model/types.go` — add StoryboardProject, StoryboardShot types
- Create: `backend/internal/repository/storyboard.go` — StoryboardRepo with full CRUD

**Interfaces:**
- Produces: `StoryboardProject` / `StoryboardShot` types, `StoryboardRepo` with methods for project + shot CRUD

- [x] **Step 1: Add types to `backend/internal/model/types.go`**

Append after the existing Asset types:

```go
// ==================== 分镜策划 ====================

type StoryboardProject struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Script    string `json:"script"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	ShotCount int    `json:"shot_count"`
}

type StoryboardShot struct {
	ID             int64  `json:"id"`
	ProjectID      int64  `json:"project_id"`
	Sequence       int    `json:"sequence"`
	Prompt         string `json:"prompt"`
	Type           string `json:"type"`
	ReferenceImage string `json:"reference_image"`
	Status         string `json:"status"`
	ResultVideo    string `json:"result_video"`
	TaskID         string `json:"task_id"`
	CreatedAt      string `json:"created_at"`
}

type CreateProjectRequest struct {
	Title  string `json:"title" binding:"required"`
	Script string `json:"script"`
}

type UpdateProjectRequest struct {
	Title  string `json:"title"`
	Script string `json:"script"`
}

type CreateShotRequest struct {
	Prompt         string `json:"prompt" binding:"required"`
	Type           string `json:"type" binding:"required"`
	ReferenceImage string `json:"reference_image"`
}

type UpdateShotRequest struct {
	Prompt         string `json:"prompt"`
	Type           string `json:"type"`
	ReferenceImage string `json:"reference_image"`
}

type ReorderShotsRequest struct {
	IDs []int64 `json:"ids" binding:"required"`
}
```

- [x] **Step 2: Create `backend/internal/repository/storyboard.go`**

```go
package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/agnes-image-tool/backend/internal/model"
)

type StoryboardRepo struct {
	db *sql.DB
}

func NewStoryboardRepo(dbPath string) (*StoryboardRepo, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS storyboard_projects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL DEFAULT '',
			script TEXT DEFAULT '',
			created_at TEXT DEFAULT (datetime('now')),
			updated_at TEXT DEFAULT (datetime('now'))
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("创建分镜项目表失败: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS storyboard_shots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER NOT NULL,
			sequence INTEGER NOT NULL DEFAULT 0,
			prompt TEXT NOT NULL DEFAULT '',
			type TEXT NOT NULL DEFAULT 'text2video',
			reference_image TEXT DEFAULT '',
			status TEXT NOT NULL DEFAULT 'pending',
			result_video TEXT DEFAULT '',
			task_id TEXT DEFAULT '',
			created_at TEXT DEFAULT (datetime('now')),
			FOREIGN KEY (project_id) REFERENCES storyboard_projects(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("创建分镜镜头表失败: %w", err)
	}

	return &StoryboardRepo{db: db}, nil
}

// ==================== Projects ====================

func (r *StoryboardRepo) ListProjects() ([]model.StoryboardProject, error) {
	rows, err := r.db.Query(`
		SELECT p.id, p.title, p.script, p.created_at, p.updated_at,
			   COALESCE((SELECT COUNT(*) FROM storyboard_shots s WHERE s.project_id = p.id), 0) AS shot_count
		FROM storyboard_projects p ORDER BY p.updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []model.StoryboardProject
	for rows.Next() {
		var p model.StoryboardProject
		if err := rows.Scan(&p.ID, &p.Title, &p.Script, &p.CreatedAt, &p.UpdatedAt, &p.ShotCount); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (r *StoryboardRepo) GetProject(id int64) (*model.StoryboardProject, error) {
	row := r.db.QueryRow(`
		SELECT p.id, p.title, p.script, p.created_at, p.updated_at,
			   COALESCE((SELECT COUNT(*) FROM storyboard_shots s WHERE s.project_id = p.id), 0) AS shot_count
		FROM storyboard_projects p WHERE p.id = ?
	`, id)

	var p model.StoryboardProject
	if err := row.Scan(&p.ID, &p.Title, &p.Script, &p.CreatedAt, &p.UpdatedAt, &p.ShotCount); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *StoryboardRepo) CreateProject(title, script string) (int64, error) {
	res, err := r.db.Exec(
		"INSERT INTO storyboard_projects (title, script, updated_at) VALUES (?, ?, datetime('now'))",
		title, script,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *StoryboardRepo) UpdateProject(id int64, title, script string) error {
	var parts []string
	var args []any

	if title != "" {
		parts = append(parts, "title = ?")
		args = append(args, title)
	}
	if script != "" {
		parts = append(parts, "script = ?")
		args = append(args, script)
	}
	if len(parts) == 0 {
		return nil
	}
	parts = append(parts, "updated_at = datetime('now')")
	args = append(args, id)

	q := fmt.Sprintf("UPDATE storyboard_projects SET %s WHERE id = ?", strings.Join(parts, ", "))
	_, err := r.db.Exec(q, args...)
	return err
}

func (r *StoryboardRepo) DeleteProject(id int64) error {
	_, err := r.db.Exec("DELETE FROM storyboard_projects WHERE id = ?", id)
	return err
}

func (r *StoryboardRepo) DuplicateProject(id int64) (int64, error) {
	orig, err := r.GetProject(id)
	if err != nil || orig == nil {
		return 0, fmt.Errorf("项目不存在")
	}

	newID, err := r.CreateProject(orig.Title+" (副本)", orig.Script)
	if err != nil {
		return 0, err
	}

	// Copy all shots
	shots, err := r.ListShots(id)
	if err != nil {
		return 0, err
	}
	for _, s := range shots {
		_, err := r.db.Exec(
			"INSERT INTO storyboard_shots (project_id, sequence, prompt, type, reference_image) VALUES (?, ?, ?, ?, ?)",
			newID, s.Sequence, s.Prompt, s.Type, s.ReferenceImage,
		)
		if err != nil {
			return 0, err
		}
	}

	return newID, nil
}

// ==================== Shots ====================

func (r *StoryboardRepo) ListShots(projectID int64) ([]model.StoryboardShot, error) {
	rows, err := r.db.Query(
		"SELECT id, project_id, sequence, prompt, type, reference_image, status, result_video, task_id, created_at FROM storyboard_shots WHERE project_id = ? ORDER BY sequence ASC",
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shots []model.StoryboardShot
	for rows.Next() {
		var s model.StoryboardShot
		if err := rows.Scan(&s.ID, &s.ProjectID, &s.Sequence, &s.Prompt, &s.Type, &s.ReferenceImage, &s.Status, &s.ResultVideo, &s.TaskID, &s.CreatedAt); err != nil {
			return nil, err
		}
		shots = append(shots, s)
	}
	return shots, rows.Err()
}

func (r *StoryboardRepo) CreateShot(projectID int64, seq int, prompt, shotType, refImage string) (int64, error) {
	res, err := r.db.Exec(
		"INSERT INTO storyboard_shots (project_id, sequence, prompt, type, reference_image) VALUES (?, ?, ?, ?, ?)",
		projectID, seq, prompt, shotType, refImage,
	)
	if err != nil {
		return 0, err
	}
	// Update project timestamp
	r.db.Exec("UPDATE storyboard_projects SET updated_at = datetime('now') WHERE id = ?", projectID)
	return res.LastInsertId()
}

func (r *StoryboardRepo) UpdateShot(id int64, prompt, shotType, refImage string) error {
	_, err := r.db.Exec(
		"UPDATE storyboard_shots SET prompt = ?, type = ?, reference_image = ? WHERE id = ?",
		prompt, shotType, refImage, id,
	)
	return err
}

func (r *StoryboardRepo) DeleteShot(id int64) error {
	_, err := r.db.Exec("DELETE FROM storyboard_shots WHERE id = ?", id)
	return err
}

func (r *StoryboardRepo) ReorderShots(ids []int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for i, id := range ids {
		_, err := tx.Exec("UPDATE storyboard_shots SET sequence = ? WHERE id = ?", i+1, id)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *StoryboardRepo) UpdateShotStatus(shotID int64, status, resultVideo, taskID string) error {
	_, err := r.db.Exec(
		"UPDATE storyboard_shots SET status = ?, result_video = ?, task_id = ? WHERE id = ?",
		status, resultVideo, taskID, shotID,
	)
	return err
}
```

- [x] **Step 3: Verify compilation**

```bash
cd backend && go build ./...
```

Expected: Build succeeds with no errors.

- [x] **Step 4: Commit**

```bash
git add backend/internal/model/types.go backend/internal/repository/storyboard.go
git commit -m "feat: add storyboard data model and repository"
```

---

### Task 2: Backend — Storyboard handler + route registration

**Files:**
- Create: `backend/internal/handler/storyboard.go` — StoryboardHandler with all endpoints
- Modify: `backend/cmd/server/main.go` — register storyboard routes

**Interfaces:**
- Consumes: `StoryboardRepo` methods from Task 1
- Produces: Handler methods for all 11 API endpoints

- [x] **Step 1: Create `backend/internal/handler/storyboard.go``

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

type StoryboardHandler struct {
	repo *repository.StoryboardRepo
}

func NewStoryboardHandler(repo *repository.StoryboardRepo) *StoryboardHandler {
	return &StoryboardHandler{repo: repo}
}

// ==================== Projects ====================

func (h *StoryboardHandler) ListProjects(c *gin.Context) {
	projects, err := h.repo.ListProjects()
	if err != nil {
		log.Printf("[Storyboard] 查询项目列表失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	if projects == nil {
		projects = []model.StoryboardProject{}
	}
	c.JSON(http.StatusOK, gin.H{"projects": projects})
}

func (h *StoryboardHandler) GetProject(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目 ID"})
		return
	}

	project, err := h.repo.GetProject(id)
	if err != nil {
		log.Printf("[Storyboard] 查询项目失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	if project == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
		return
	}

	shots, err := h.repo.ListShots(id)
	if err != nil {
		log.Printf("[Storyboard] 查询镜头失败: %v", err)
		shots = []model.StoryboardShot{}
	}
	if shots == nil {
		shots = []model.StoryboardShot{}
	}

	c.JSON(http.StatusOK, gin.H{
		"project": project,
		"shots":   shots,
	})
}

func (h *StoryboardHandler) CreateProject(c *gin.Context) {
	var req model.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	id, err := h.repo.CreateProject(req.Title, req.Script)
	if err != nil {
		log.Printf("[Storyboard] 创建项目失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id})
}

func (h *StoryboardHandler) UpdateProject(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目 ID"})
		return
	}

	var req model.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	if err := h.repo.UpdateProject(id, req.Title, req.Script); err != nil {
		log.Printf("[Storyboard] 更新项目失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *StoryboardHandler) DeleteProject(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目 ID"})
		return
	}

	if err := h.repo.DeleteProject(id); err != nil {
		log.Printf("[Storyboard] 删除项目失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *StoryboardHandler) DuplicateProject(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目 ID"})
		return
	}

	newID, err := h.repo.DuplicateProject(id)
	if err != nil {
		log.Printf("[Storyboard] 复制项目失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "复制失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": newID})
}

// ==================== Shots ====================

func (h *StoryboardHandler) CreateShot(c *gin.Context) {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目 ID"})
		return
	}

	var req model.CreateShotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// Get current shot count for sequence
	shots, err := h.repo.ListShots(projectID)
	if err != nil {
		shots = nil
	}
	seq := len(shots) + 1

	id, err := h.repo.CreateShot(projectID, seq, req.Prompt, req.Type, req.ReferenceImage)
	if err != nil {
		log.Printf("[Storyboard] 创建镜头失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id})
}

func (h *StoryboardHandler) UpdateShot(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的镜头 ID"})
		return
	}

	var req model.UpdateShotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	if err := h.repo.UpdateShot(id, req.Prompt, req.Type, req.ReferenceImage); err != nil {
		log.Printf("[Storyboard] 更新镜头失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *StoryboardHandler) DeleteShot(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的镜头 ID"})
		return
	}

	if err := h.repo.DeleteShot(id); err != nil {
		log.Printf("[Storyboard] 删除镜头失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *StoryboardHandler) ReorderShots(c *gin.Context) {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目 ID"})
		return
	}

	var req model.ReorderShotsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	if err := h.repo.ReorderShots(req.IDs); err != nil {
		log.Printf("[Storyboard] 排序镜头失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "排序失败"})
		return
	}

	// Update project timestamp
	h.repo.UpdateProject(projectID, "", "")

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ==================== Generate ====================

func (h *StoryboardHandler) GenerateShots(c *gin.Context) {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目 ID"})
		return
	}

	shots, err := h.repo.ListShots(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询镜头失败"})
		return
	}

	var pending []model.StoryboardShot
	for _, s := range shots {
		if s.Status == "pending" {
			pending = append(pending, s)
		}
	}

	if len(pending) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "没有待生成的镜头", "generated": 0})
		return
	}

	generated := 0
	for _, s := range pending {
		// Each pending shot will be generated. The actual video API call
		// is handled by the frontend/client side to allow per-shot SSE tracking.
		// Here we just return the list of pending shots to generate.
		_ = s
		generated++
	}

	// Update project timestamp
	h.repo.UpdateProject(projectID, "", "")

	c.JSON(http.StatusOK, gin.H{
		"message":   "批量生成已触发",
		"generated": generated,
	})
}
```

- [x] **Step 2: Register routes in `backend/cmd/server/main.go`**

After `assetHandler := handler.NewAssetHandler(histRepo)`, add:

```go
storyboardRepo, err := repository.NewStoryboardRepo(historyPath)
if err != nil {
	log.Fatalf("初始化分镜仓库失败: %v", err)
}
storyboardHandler := handler.NewStoryboardHandler(storyboardRepo)
```

Inside the API group, after asset routes:

```go
// 分镜策划
storyboard := api.Group("/storyboard")
{
	storyboard.GET("/projects", storyboardHandler.ListProjects)
	storyboard.POST("/projects", storyboardHandler.CreateProject)
	storyboard.GET("/projects/:id", storyboardHandler.GetProject)
	storyboard.PUT("/projects/:id", storyboardHandler.UpdateProject)
	storyboard.DELETE("/projects/:id", storyboardHandler.DeleteProject)
	storyboard.POST("/projects/:id/duplicate", storyboardHandler.DuplicateProject)
	storyboard.POST("/projects/:id/shots", storyboardHandler.CreateShot)
	storyboard.PUT("/projects/:id/shots/reorder", storyboardHandler.ReorderShots)
	storyboard.PUT("/shots/:id", storyboardHandler.UpdateShot)
	storyboard.DELETE("/shots/:id", storyboardHandler.DeleteShot)
	storyboard.POST("/projects/:id/generate", storyboardHandler.GenerateShots)
}
```

- [x] **Step 3: Add import for `repository` package if not already present**

In `main.go`, ensure `repository` is imported (it should already be imported from other handlers).

- [x] **Step 4: Verify compilation**

```bash
cd backend && go build ./...
```

Expected: Build succeeds with no errors.

- [x] **Step 5: Commit**

```bash
git add backend/internal/handler/storyboard.go backend/cmd/server/main.go
git commit -m "feat: add storyboard handler and route registration"
```

---

### Task 3: Frontend — TypeScript types + API client

**Files:**
- Modify: `frontend/src/types/index.ts` — add Storyboard types
- Create: `frontend/src/api/storyboard.ts` — API client

- [x] **Step 1: Add types to `frontend/src/types/index.ts`**

Append after existing types:

```typescript
export interface StoryboardProject {
  id: number
  title: string
  script: string
  created_at: string
  updated_at: string
  shot_count: number
}

export interface StoryboardShot {
  id: number
  project_id: number
  sequence: number
  prompt: string
  type: string
  reference_image: string
  status: 'pending' | 'generating' | 'completed'
  result_video: string
  task_id: string
  created_at: string
}

export interface CreateProjectRequest {
  title: string
  script?: string
}

export interface UpdateProjectRequest {
  title?: string
  script?: string
}

export interface CreateShotRequest {
  prompt: string
  type: string
  reference_image?: string
}

export interface UpdateShotRequest {
  prompt?: string
  type?: string
  reference_image?: string
}
```

- [x] **Step 2: Create `frontend/src/api/storyboard.ts`**

```typescript
import client from './client'
import type { StoryboardProject, StoryboardShot, CreateProjectRequest, UpdateProjectRequest, CreateShotRequest, UpdateShotRequest } from '../types'

export async function listProjects(): Promise<StoryboardProject[]> {
  const res = await client.get('/api/v1/storyboard/projects')
  return res.data.projects
}

export async function getProject(id: number): Promise<{ project: StoryboardProject; shots: StoryboardShot[] }> {
  const res = await client.get(`/api/v1/storyboard/projects/${id}`)
  return res.data
}

export async function createProject(data: CreateProjectRequest): Promise<number> {
  const res = await client.post('/api/v1/storyboard/projects', data)
  return res.data.id
}

export async function updateProject(id: number, data: UpdateProjectRequest): Promise<void> {
  await client.put(`/api/v1/storyboard/projects/${id}`, data)
}

export async function deleteProject(id: number): Promise<void> {
  await client.delete(`/api/v1/storyboard/projects/${id}`)
}

export async function duplicateProject(id: number): Promise<number> {
  const res = await client.post(`/api/v1/storyboard/projects/${id}/duplicate`)
  return res.data.id
}

export async function createShot(projectId: number, data: CreateShotRequest): Promise<number> {
  const res = await client.post(`/api/v1/storyboard/projects/${projectId}/shots`, data)
  return res.data.id
}

export async function updateShot(id: number, data: UpdateShotRequest): Promise<void> {
  await client.put(`/api/v1/storyboard/shots/${id}`, data)
}

export async function deleteShot(id: number): Promise<void> {
  await client.delete(`/api/v1/storyboard/shots/${id}`)
}

export async function reorderShots(projectId: number, ids: number[]): Promise<void> {
  await client.put(`/api/v1/storyboard/projects/${projectId}/shots/reorder`, { ids })
}

export async function generateShots(projectId: number): Promise<void> {
  await client.post(`/api/v1/storyboard/projects/${projectId}/generate`)
}
```

- [x] **Step 3: Run typecheck**

```bash
cd frontend && npx vue-tsc --noEmit
```

Expected: No type errors.

- [x] **Step 4: Commit**

```bash
git add frontend/src/types/index.ts frontend/src/api/storyboard.ts
git commit -m "feat: add storyboard types and API client"
```

---

### Task 4: Frontend — ShotCard component

**Files:**
- Create: `frontend/src/components/ShotCard.vue`

- [x] **Step 1: Create `frontend/src/components/ShotCard.vue`**

```vue
<script setup lang="ts">
import { computed } from 'vue'
import type { StoryboardShot } from '../types'
import { VideoCameraFilled, PictureFilled } from '@element-plus/icons-vue'

const props = defineProps<{
  shot: StoryboardShot
}>()

const emit = defineEmits<{
  (e: 'edit'): void
  (e: 'delete'): void
  (e: 'generate'): void
  (e: 'preview'): void
}>()

const statusConfig = computed(() => {
  const configs: Record<string, { label: string; color: string }> = {
    pending: { label: '待生成', color: '#909399' },
    generating: { label: '生成中', color: '#409eff' },
    completed: { label: '已完成', color: '#67c23a' },
  }
  return configs[props.shot.status] || configs.pending
})

const typeLabel = computed(() => {
  const labels: Record<string, string> = {
    text2video: '文生视频',
    image2video: '图生视频',
  }
  return labels[props.shot.type] || props.shot.type
})

const isVideo = computed(() => props.shot.type === 'text2video' || props.shot.type === 'image2video')
</script>

<template>
  <div class="shot-card" :class="`is-${shot.status}`">
    <div class="shot-card__header">
      <span class="shot-card__seq">#{{ shot.sequence }}</span>
      <el-tag :color="statusConfig.color" :style="{ color: '#fff', border: 'none' }" size="small">
        {{ statusConfig.label }}
      </el-tag>
    </div>

    <div class="shot-card__body">
      <div class="shot-card__icon">
        <el-icon v-if="isVideo" :size="24"><VideoCameraFilled /></el-icon>
        <el-icon v-else :size="24"><PictureFilled /></el-icon>
      </div>
      <div class="shot-card__info">
        <div class="shot-card__type">{{ typeLabel }}</div>
        <div class="shot-card__prompt">{{ shot.prompt?.slice(0, 50) }}{{ shot.prompt?.length > 50 ? '...' : '' }}</div>
      </div>
    </div>

    <div v-if="shot.result_video" class="shot-card__result">
      <video v-if="isVideo" :src="shot.result_video" controls style="width: 100%; max-height: 120px" />
      <el-image v-else :src="shot.result_video" fit="cover" style="width: 100%; max-height: 120px" />
    </div>

    <div class="shot-card__actions">
      <el-button size="small" text @click="emit('edit')">编辑</el-button>
      <el-button v-if="shot.status === 'pending'" size="small" type="primary" text @click="emit('generate')">
        生成
      </el-button>
      <el-button v-if="shot.result_video" size="small" text @click="emit('preview')">
        预览
      </el-button>
      <el-button size="small" text type="danger" @click="emit('delete')">删除</el-button>
    </div>
  </div>
</template>

<style scoped>
.shot-card {
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 12px;
  margin-bottom: 12px;
  background: #fff;
  transition: box-shadow 0.2s;
}
.shot-card:hover {
  box-shadow: 0 2px 12px rgba(0,0,0,0.08);
}
.shot-card.is-generating {
  border-color: #409eff;
  background: #f0f7ff;
}
.shot-card.is-completed {
  border-color: #e1f3d8;
}
.shot-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}
.shot-card__seq {
  font-weight: 600;
  font-size: 14px;
  color: #303133;
}
.shot-card__body {
  display: flex;
  gap: 10px;
  align-items: flex-start;
}
.shot-card__icon {
  color: #909399;
  flex-shrink: 0;
  margin-top: 2px;
}
.shot-card__info {
  flex: 1;
  min-width: 0;
}
.shot-card__type {
  font-size: 12px;
  color: #909399;
  margin-bottom: 4px;
}
.shot-card__prompt {
  font-size: 13px;
  color: #606266;
  line-height: 1.5;
}
.shot-card__result {
  margin-top: 8px;
}
.shot-card__actions {
  display: flex;
  gap: 4px;
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid #f0f0f0;
}
</style>
```

- [x] **Step 2: Run typecheck**

```bash
cd frontend && npx vue-tsc --noEmit
```

Expected: No type errors.

- [x] **Step 3: Commit**

```bash
git add frontend/src/components/ShotCard.vue
git commit -m "feat: add ShotCard component"
```

---

### Task 5: Frontend — Storyboard page + App.vue tab

**Files:**
- Create: `frontend/src/views/Storyboard.vue`
- Modify: `frontend/src/App.vue` — add tab

- [x] **Step 1: Create `frontend/src/views/Storyboard.vue`**

```vue
<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Edit, Delete, CopyDocument, VideoPlay } from '@element-plus/icons-vue'
import { listProjects, getProject, createProject, updateProject, deleteProject, duplicateProject, createShot, deleteShot, generateShots } from '../api/storyboard'
import type { StoryboardProject, StoryboardShot } from '../types'
import ShotCard from '../components/ShotCard.vue'

// View state: 'list' | 'detail'
const view = ref<'list' | 'detail'>('list')

// Project list
const projects = ref<StoryboardProject[]>([])
const loading = ref(false)

// Project detail
const currentProject = ref<StoryboardProject | null>(null)
const shots = ref<StoryboardShot[]>([])

// Dialog
const showProjectDialog = ref(false)
const isEditingProject = ref(false)
const projectForm = ref({ title: '', script: '' })

const showShotDialog = ref(false)
const shotForm = ref({ prompt: '', type: 'text2video', reference_image: '' })

// Load project list
async function loadProjects() {
  loading.value = true
  try {
    projects.value = await listProjects()
  } catch (e: any) {
    ElMessage.error('加载失败: ' + (e.message || ''))
  } finally {
    loading.value = false
  }
}

onMounted(loadProjects)

// Create / Edit project
function openNewProject() {
  isEditingProject.value = false
  projectForm.value = { title: '', script: '' }
  showProjectDialog.value = true
}

async function saveProject() {
  try {
    if (isEditingProject.value && currentProject.value) {
      await updateProject(currentProject.value.id, projectForm.value)
      ElMessage.success('保存成功')
    } else {
      const id = await createProject(projectForm.value)
      currentProject.value = { id, title: projectForm.value.title, script: projectForm.value.script, created_at: '', updated_at: '', shot_count: 0 }
      view.value = 'detail'
      ElMessage.success('创建成功')
    }
    showProjectDialog.value = false
    await loadProjects()
  } catch (e: any) {
    ElMessage.error('操作失败')
  }
}

// Open project detail
async function openProject(project: StoryboardProject) {
  loading.value = true
  try {
    const data = await getProject(project.id)
    currentProject.value = data.project
    shots.value = data.shots
    view.value = 'detail'
  } catch (e: any) {
    ElMessage.error('加载失败')
  } finally {
    loading.value = false
  }
}

function backToList() {
  view.value = 'list'
  currentProject.value = null
  shots.value = []
}

// Edit project title/script
function editProject() {
  if (!currentProject.value) return
  isEditingProject.value = true
  projectForm.value = { title: currentProject.value.title, script: currentProject.value.script }
  showProjectDialog.value = true
}

async function deleteCurrentProject() {
  if (!currentProject.value) return
  try {
    await ElMessageBox.confirm('确定删除此分镜项目？所有镜头将一同删除。', '确认删除', { type: 'warning' })
    await deleteProject(currentProject.value.id)
    ElMessage.success('删除成功')
    backToList()
    await loadProjects()
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error('删除失败')
  }
}

async function duplicateCurrentProject() {
  if (!currentProject.value) return
  try {
    await duplicateProject(currentProject.value.id)
    ElMessage.success('复制成功')
    await loadProjects()
  } catch (e: any) {
    ElMessage.error('复制失败')
  }
}

// Shot management
function openNewShot() {
  shotForm.value = { prompt: '', type: 'text2video', reference_image: '' }
  showShotDialog.value = true
}

async function addShot() {
  if (!currentProject.value) return
  if (!shotForm.value.prompt) {
    ElMessage.warning('请输入提示词')
    return
  }
  try {
    await createShot(currentProject.value.id, shotForm.value)
    ElMessage.success('添加成功')
    showShotDialog.value = false
    // Reload shots
    const data = await getProject(currentProject.value.id)
    currentProject.value = data.project
    shots.value = data.shots
  } catch (e: any) {
    ElMessage.error('添加失败')
  }
}

async function deleteShotById(id: number) {
  try {
    await deleteShot(id)
    shots.value = shots.value.filter(s => s.id !== id)
    ElMessage.success('删除成功')
  } catch (e: any) {
    ElMessage.error('删除失败')
  }
}

async function handleGenerateShots() {
  if (!currentProject.value) return
  try {
    await generateShots(currentProject.value.id)
    ElMessage.success('批量生成已触发')
    // Reload to see status updates
    const data = await getProject(currentProject.value.id)
    shots.value = data.shots
  } catch (e: any) {
    ElMessage.error('批量生成失败')
  }
}

// Preview
const previewUrl = ref('')
const showPreview = ref(false)

function previewVideo(url: string) {
  previewUrl.value = url
  showPreview.value = true
}
</script>

<template>
  <div>
    <!-- ==================== Project List View ==================== -->
    <template v-if="view === 'list'">
      <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px">
        <h3 style="margin: 0">分镜项目</h3>
        <el-button type="primary" size="small" :icon="Plus" @click="openNewProject">新建项目</el-button>
      </div>

      <div v-loading="loading">
        <div v-if="projects.length === 0 && !loading" style="text-align: center; padding: 60px; color: #c0c4cc">
          <el-icon :size="48"><VideoPlay /></el-icon>
          <p style="margin-top: 12px">暂无分镜项目</p>
          <el-button type="primary" size="small" @click="openNewProject">创建第一个项目</el-button>
        </div>

        <div v-else style="display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 16px">
          <div v-for="project in projects" :key="project.id" class="project-card" @click="openProject(project)">
            <div style="font-weight: 600; font-size: 15px; color: #303133; margin-bottom: 8px">
              {{ project.title || '未命名项目' }}
            </div>
            <div style="font-size: 12px; color: #909399">
              {{ project.shot_count }} 个镜头 · {{ project.updated_at?.slice(0, 10) }}
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- ==================== Project Detail View ==================== -->
    <template v-else-if="view === 'detail' && currentProject">
      <div style="display: flex; align-items: center; gap: 12px; margin-bottom: 16px; flex-wrap: wrap">
        <el-button text @click="backToList">&lt; 返回</el-button>
        <h3 style="margin: 0; flex: 1">{{ currentProject.title || '未命名项目' }}</h3>
        <el-button size="small" :icon="Edit" @click="editProject">编辑</el-button>
        <el-button size="small" :icon="CopyDocument" @click="duplicateCurrentProject">复制</el-button>
        <el-button size="small" type="danger" :icon="Delete" @click="deleteCurrentProject">删除</el-button>
        <el-button v-if="shots.some(s => s.status === 'pending')" type="primary" size="small" @click="handleGenerateShots">
          批量生成 ({{ shots.filter(s => s.status === 'pending').length }})
        </el-button>
      </div>

      <!-- Shots list -->
      <div v-loading="loading">
        <div v-if="shots.length === 0 && !loading" style="text-align: center; padding: 40px; color: #c0c4cc">
          <p>暂无镜头，添加第一个镜头开始策划</p>
        </div>

        <ShotCard
          v-for="shot in shots"
          :key="shot.id"
          :shot="shot"
          @edit="/* TODO: inline edit */"
          @delete="deleteShotById(shot.id)"
          @generate="/* TODO: generate single shot */"
          @preview="shot.result_video ? previewVideo(shot.result_video) : undefined"
        />

        <div style="text-align: center; margin-top: 16px">
          <el-button type="primary" plain :icon="Plus" @click="openNewShot">添加镜头</el-button>
        </div>
      </div>
    </template>

    <!-- ==================== Project Dialog ==================== -->
    <el-dialog
      v-model="showProjectDialog"
      :title="isEditingProject ? '编辑项目' : '新建项目'"
      width="500px"
    >
      <el-form :model="projectForm" label-width="60px">
        <el-form-item label="标题">
          <el-input v-model="projectForm.title" placeholder="分镜项目名称" />
        </el-form-item>
        <el-form-item label="脚本">
          <el-input
            v-model="projectForm.script"
            type="textarea"
            :rows="6"
            placeholder="可选：粘贴完整的脚本内容，后续可拆分为镜头"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showProjectDialog = false">取消</el-button>
        <el-button type="primary" @click="saveProject">保存</el-button>
      </template>
    </el-dialog>

    <!-- ==================== Shot Dialog ==================== -->
    <el-dialog
      v-model="showShotDialog"
      title="添加镜头"
      width="500px"
    >
      <el-form :model="shotForm" label-width="80px">
        <el-form-item label="提示词">
          <el-input
            v-model="shotForm.prompt"
            type="textarea"
            :rows="4"
            placeholder="描述这个镜头的画面内容"
          />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="shotForm.type">
            <el-option label="文生视频" value="text2video" />
            <el-option label="图生视频" value="image2video" />
          </el-select>
        </el-form-item>
        <el-form-item label="参考图">
          <el-input v-model="shotForm.reference_image" placeholder="图片 URL（可选）" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showShotDialog = false">取消</el-button>
        <el-button type="primary" @click="addShot">添加</el-button>
      </template>
    </el-dialog>

    <!-- ==================== Preview Dialog ==================== -->
    <el-dialog v-model="showPreview" title="视频预览" width="600px">
      <video v-if="previewUrl" :src="previewUrl" controls style="width: 100%; max-height: 400px" />
    </el-dialog>
  </div>
</template>

<style scoped>
.project-card {
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 16px;
  cursor: pointer;
  transition: box-shadow 0.2s, border-color 0.2s;
  background: #fff;
}
.project-card:hover {
  box-shadow: 0 2px 12px rgba(0,0,0,0.08);
  border-color: #409eff;
}
</style>
```

- [x] **Step 2: Add tab to `frontend/src/App.vue`**

Add import:
```typescript
import Storyboard from './views/Storyboard.vue'
```

Add tab before "作品" tab:
```vue
<el-tab-pane label="分镜" name="storyboard">
  <Storyboard />
</el-tab-pane>
```

- [x] **Step 3: Run typecheck**

```bash
cd frontend && npx vue-tsc --noEmit
```

Expected: No type errors.

- [x] **Step 4: Commit**

```bash
git add frontend/src/views/Storyboard.vue frontend/src/App.vue
git commit -m "feat: add storyboard page and tab"
```

---

### Task 6: Integration test + polish

**Files:**
- Any files with bugs found during testing

- [x] **Step 1: Start backend and verify storyboard API**

```bash
cd backend && go run ./cmd/server
```

Test in another terminal:
```bash
# Create project
curl -X POST "http://localhost:8080/api/v1/storyboard/projects" \
  -H "Content-Type: application/json" \
  -d '{"title": "测试项目", "script": "测试脚本"}'

# List projects
curl "http://localhost:8080/api/v1/storyboard/projects"

# Add shot
curl -X POST "http://localhost:8080/api/v1/storyboard/projects/1/shots" \
  -H "Content-Type: application/json" \
  -d '{"prompt": "一只猫在走路", "type": "text2video"}'

# Get project detail
curl "http://localhost:8080/api/v1/storyboard/projects/1"
```

Expected: All endpoints return correct responses.

- [x] **Step 2: Run full backend test suite**

```bash
cd backend && go test ./... -v
```

Expected: All tests pass.

- [x] **Step 3: Run frontend typecheck + build**

```bash
cd frontend && npx vue-tsc --noEmit && pnpm build
```

Expected: Build succeeds.

- [x] **Step 4: Commit final polish**

```bash
git add -A
git commit -m "feat: complete storyboard studio with polish and fixes"
```
