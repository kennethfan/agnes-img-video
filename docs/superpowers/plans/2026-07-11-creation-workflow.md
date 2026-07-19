# 创作区项目式设计 — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将「创作」导航组重构为项目式 4 步闭环（创意简报 → 生成+对比 → 精修 → 定稿）

**Architecture:** 新增 Project + ProjectStep 两个 GORM 模型 & SQLite 表，后端 CRUD + AI 推荐端点，前端 ProjectList + ProjectEditor（4 步容器）+ 子组件。不动其他导航组。

**Tech Stack:** Go 1.25 · Gin · GORM · SQLite · Vue 3 · TypeScript 6 · Element Plus · Axios

## Global Constraints

- 不动现有「图片/视频/工具/作品/系统」导航组
- 所有新增 GORM 模型在 `backend/internal/repository/gorm/models.go` 中添加，并在 `OpenDB().AutoMigrate()` 注册
- 所有新增 API 路由在 `backend/cmd/server/main.go` 注册（`/api/v1/` 下）
- 前端使用 Composition API + `<script setup>`，TypeScript 6 `erasableSyntaxOnly`（无 enum）
- 后端错误返回中文：`gin.H{"error": "..."}`，日志前缀 `[Project]`
- 所有代码注释使用中文（项目规范）
- 提交信息使用 semantic style（`feat:` / `fix:` / `docs:`）
- 前端验证：`pnpm build`；后端验证：`go vet ./... && go build ./cmd/server`
- 模型 ID 类型使用 `int64`，与代码库现有所有模型一致

---

### Task 1: 后端 — Project + ProjectStep 模型、Repository、Handler、路由注册

**Files:**
- Modify: `backend/internal/repository/gorm/models.go` — 添加 Project + ProjectStep 模型
- Modify: `backend/internal/repository/gorm/gorm.go` — AutoMigrate
- Create: `backend/internal/repository/gorm/project.go` — ProjectRepository
- Create: `backend/internal/handler/project.go` — ProjectHandler
- Modify: `backend/cmd/server/main.go` — 注册路由

**Interfaces:**
- Produces: `ProjectRepository`（CRUD + step management）、`ProjectHandler`（12 个 Gin handlers）

- [ ] **Step 1: 添加 GORM 模型**

在 `backend/internal/repository/gorm/models.go` 末尾添加：

```go
// Project 创作项目
type Project struct {
	ID        int64     `gorm:"primaryKey"`
	Title     string    `gorm:"size:200"`
	Brief     string    `gorm:"type:text"`        // 原始创意简报
	AIResult  string    `gorm:"type:text"`        // AI 推荐结果 JSON
	Status    string    `gorm:"size:20;default:draft"` // draft | generating | refining | completed
	CoverURL  string    `gorm:"type:text"`        // 定稿封面图
	FinalURL  string    `gorm:"type:text"`        // 最终输出 URL
	AssetIDs  string    `gorm:"type:text"`        // 关联作品库资产 ID 列表 JSON
	Notes     string    `gorm:"type:text"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Steps     []ProjectStep `gorm:"foreignKey:ProjectID"`
}

func (Project) TableName() string { return "projects" }

// ProjectStep 项目步骤
type ProjectStep struct {
	ID        int64     `gorm:"primaryKey"`
	ProjectID int64     `gorm:"index"`
	StepType  string    `gorm:"size:20"`       // generate | refine | finalize
	Position  int                               // 步骤序号
	Input     string    `gorm:"type:text"`      // 输入参数 JSON
	Output    string    `gorm:"type:text"`      // 输出结果 JSON
	CreatedAt time.Time
}

func (ProjectStep) TableName() string { return "project_steps" }
```

- [ ] **Step 2: 在 AutoMigrate 注册**

`backend/internal/repository/gorm/gorm.go` 的 `db.AutoMigrate()` 中添加：
```go
&Project{}, &ProjectStep{},
```

- [ ] **Step 3: 编写 Repository**

创建 `backend/internal/repository/gorm/project.go`：

```go
package gorm

import (
	"encoding/json"
	"gorm.io/gorm"
)

type ProjectRepository struct {
	db *gorm.DB
}

func NewProjectRepository(db *gorm.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// List 获取项目列表（按更新时间倒序）
func (r *ProjectRepository) List() ([]Project, error) {
	var projects []Project
	err := r.db.Order("updated_at desc").Find(&projects).Error
	return projects, err
}

// GetByID 获取项目详情（含步骤，按 position 排序）
func (r *ProjectRepository) GetByID(id int64) (*Project, error) {
	var project Project
	err := r.db.Preload("Steps", func(db *gorm.DB) *gorm.DB {
		return db.Order("position asc")
	}).First(&project, id).Error
	return &project, err
}

// Create 创建项目
func (r *ProjectRepository) Create(project *Project) error {
	return r.db.Create(project).Error
}

// Update 更新项目字段
func (r *ProjectRepository) Update(project *Project) error {
	return r.db.Model(&Project{}).Where("id = ?", project.ID).Updates(map[string]interface{}{
		"title":    project.Title,
		"brief":    project.Brief,
		"ai_result": project.AIResult,
		"status":   project.Status,
		"cover_url": project.CoverURL,
		"final_url": project.FinalURL,
		"asset_ids": project.AssetIDs,
		"notes":    project.Notes,
	}).Error
}

// Delete 删除项目（级联删步骤）
func (r *ProjectRepository) Delete(id int64) error {
	r.db.Where("project_id = ?", id).Delete(&ProjectStep{})
	return r.db.Delete(&Project{}, id).Error
}

// AddStep 添加步骤
func (r *ProjectRepository) AddStep(step *ProjectStep) error {
	return r.db.Create(step).Error
}

// UpdateStep 更新步骤
func (r *ProjectRepository) UpdateStep(step *ProjectStep) error {
	return r.db.Model(&ProjectStep{}).Where("id = ?", step.ID).Updates(map[string]interface{}{
		"input":  step.Input,
		"output": step.Output,
	}).Error
}

// DeleteStep 删除步骤
func (r *ProjectRepository) DeleteStep(stepID int64) error {
	return r.db.Delete(&ProjectStep{}, stepID).Error
}

// GetStepByID 获取单个步骤
func (r *ProjectRepository) GetStepByID(stepID int64) (*ProjectStep, error) {
	var step ProjectStep
	err := r.db.First(&step, stepID).Error
	return &step, err
}

// Duplicate 复制项目（含步骤，新项目状态为 draft）
func (r *ProjectRepository) Duplicate(id int64) (*Project, error) {
	orig, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}
	newProject := &Project{
		Title: orig.Title + " (副本)",
		Brief: orig.Brief,
		Status: "draft",
		Notes: orig.Notes,
	}
	if err := r.db.Create(newProject).Error; err != nil {
		return nil, err
	}
	// 复制步骤（重置 ID）
	for _, step := range orig.Steps {
		newStep := &ProjectStep{
			ProjectID: newProject.ID,
			StepType:  step.StepType,
			Position:  step.Position,
			Input:     step.Input,
			Output:    step.Output,
		}
		if err := r.db.Create(newStep).Error; err != nil {
			return nil, err
		}
	}
	return newProject, nil
}
```

- [ ] **Step 4: 编写 HTTP Handler**

创建 `backend/internal/handler/project.go`：

```go
package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	gormrepo "github.com/agnes-image-tool/backend/internal/repository/gorm"
	"github.com/agnes-image-tool/backend/internal/service"
)

type ProjectHandler struct {
	repo *gormrepo.ProjectRepository
	svc  *service.AgnesClient
}

func NewProjectHandler(repo *gormrepo.ProjectRepository, svc *service.AgnesClient) *ProjectHandler {
	return &ProjectHandler{repo: repo, svc: svc}
}

// ListProjects GET /api/v1/projects
func (h *ProjectHandler) ListProjects(c *gin.Context) {
	projects, err := h.repo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询项目列表失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, projects)
}

// CreateProject POST /api/v1/projects
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	var req struct {
		Title string `json:"title" binding:"required"`
		Brief string `json:"brief"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	project := &gormrepo.Project{
		Title:     req.Title,
		Brief:     req.Brief,
		Status:    "draft",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := h.repo.Create(project); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建项目失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, project)
}

// GetProject GET /api/v1/projects/:id
func (h *ProjectHandler) GetProject(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	project, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
		return
	}
	c.JSON(http.StatusOK, project)
}

// UpdateProject PUT /api/v1/projects/:id
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	project, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
		return
	}
	var req struct {
		Title    string `json:"title"`
		Brief    string `json:"brief"`
		AIResult string `json:"ai_result"`
		Status   string `json:"status"`
		CoverURL string `json:"cover_url"`
		FinalURL string `json:"final_url"`
		AssetIDs string `json:"asset_ids"`
		Notes    string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	if req.Title != "" { project.Title = req.Title }
	if req.Brief != "" { project.Brief = req.Brief }
	if req.AIResult != "" { project.AIResult = req.AIResult }
	if req.Status != "" { project.Status = req.Status }
	if req.CoverURL != "" { project.CoverURL = req.CoverURL }
	if req.FinalURL != "" { project.FinalURL = req.FinalURL }
	if req.AssetIDs != "" { project.AssetIDs = req.AssetIDs }
	if req.Notes != "" { project.Notes = req.Notes }
	if err := h.repo.Update(project); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新项目失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// DeleteProject DELETE /api/v1/projects/:id
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.repo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除项目失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// DuplicateProject POST /api/v1/projects/:id/duplicate
func (h *ProjectHandler) DuplicateProject(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	project, err := h.repo.Duplicate(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "复制项目失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, project)
}

// AddStep POST /api/v1/projects/:id/steps
func (h *ProjectHandler) AddStep(c *gin.Context) {
	projectID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		StepType string `json:"step_type" binding:"required"`
		Input    string `json:"input"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	// 获取当前最大 position
	project, _ := h.repo.GetByID(projectID)
	position := len(project.Steps) + 1
	step := &gormrepo.ProjectStep{
		ProjectID: projectID,
		StepType:  req.StepType,
		Position:  position,
		Input:     req.Input,
		Output:    "",
		CreatedAt: time.Now(),
	}
	if err := h.repo.AddStep(step); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "添加步骤失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, step)
}

// UpdateStep PUT /api/v1/projects/:id/steps/:stepId
func (h *ProjectHandler) UpdateStep(c *gin.Context) {
	stepID, _ := strconv.ParseInt(c.Param("stepId"), 10, 64)
	var req struct {
		Input  string `json:"input"`
		Output string `json:"output"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	step, err := h.repo.GetStepByID(stepID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "步骤不存在"})
		return
	}
	if req.Input != "" { step.Input = req.Input }
	if req.Output != "" { step.Output = req.Output }
	if err := h.repo.UpdateStep(step); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新步骤失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// DeleteStep DELETE /api/v1/projects/:id/steps/:stepId
func (h *ProjectHandler) DeleteStep(c *gin.Context) {
	stepID, _ := strconv.ParseInt(c.Param("stepId"), 10, 64)
	if err := h.repo.DeleteStep(stepID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除步骤失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// AIRecommend POST /api/v1/projects/:id/ai-recommend
// 调用 chat API 对创意简报进行智能推荐：风格、尺寸、模型、prompt 扩写
func (h *ProjectHandler) AIRecommend(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	project, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
		return
	}
	if project.Brief == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先填写创意简报"})
		return
	}

	systemPrompt := `你是一个专业的 AI 图片创作助手。根据用户提供的创意描述，从以下四个维度给出建议：
1. enhanced_prompt: 将用户的简短描述扩写为详细的 AI 生图 prompt（中文）
2. style_suggestions: 推荐 3-5 种适合的风格，用中文逗号分隔
3. size_suggestion: 推荐合适的图片尺寸（如 "1:1"、"16:9"、"3:4"、"4:3"、"9:16"）
4. model_suggestion: 推荐图片模型

请以 JSON 格式返回，不要包含其他文字。`

	resp, err := h.svc.Chat(systemPrompt, "用户创意描述: "+project.Brief)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI 推荐失败: " + err.Error()})
		return
	}

	// 保存 AI 结果到项目
	project.AIResult = resp
	h.repo.Update(project)

	c.JSON(http.StatusOK, gin.H{
		"ai_result": resp,
	})
}

// FinalizeProject POST /api/v1/projects/:id/finalize
// 定稿：设置最终 URL，保存到作品库，更新状态为 completed
func (h *ProjectHandler) FinalizeProject(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	project, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
		return
	}
	var req struct {
		FinalURL string `json:"final_url" binding:"required"`
		Notes    string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	project.FinalURL = req.FinalURL
	project.CoverURL = req.FinalURL
	project.Status = "completed"
	if req.Notes != "" {
		project.Notes = req.Notes
	}
	if err := h.repo.Update(project); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "定稿失败: " + err.Error()})
		return
	}

	// 自动保存到作品库（如果有 assetRepo）
	assetRepo := GetAssetRepo()
	if assetRepo != nil {
		assetRepo.Create(project.Title, req.FinalURL, "", "project")
	}

	c.JSON(http.StatusOK, gin.H{"message": "定稿成功", "project": project})
}
```

注意：`GetAssetRepo()` 需要从 `asset.go` 中已有的 `assetRepo` 变量暴露出来。在 `backend/internal/handler/asset.go` 中添加：
```go
var assetRepo gormrepo.AssetRepositoryInterface

func SetAssetRepo(repo gormrepo.AssetRepositoryInterface) {
	assetRepo = repo
}

func GetAssetRepo() gormrepo.AssetRepositoryInterface {
	return assetRepo
}
```

或者如果 `SetAssetRepo` 已存在，只需添加 `GetAssetRepo`。同样在 `main.go` 中将 `NewProjectHandler` 传入 `svc`。

建立 `AssetRepositoryInterface` 接口（因为 handler 包只引用 gormrepo 包中定义的接口）。在 `backend/internal/repository/interfaces.go` 中添加：

```go
// 用于 handler 包引用的入参接口
type AssetRepositoryInterface interface {
	Create(prompt, imageURL, localPath, mode string) (*Asset, error)
}
```

- [ ] **Step 5: 注册路由**

在 `backend/cmd/server/main.go` 中：

添加 import `"encoding/json"` 和 `"time"` 不需要（已有 log, os 等）。

在 `main.go` 的初始化区添加：
```go
projectRepo := gormrepo.NewProjectRepository(gormDB)
projectHandler := handler.NewProjectHandler(projectRepo, svc)
```

在路由组 `api := r.Group("/api/v1")` 内添加：
```go
// 创作项目
api.GET("/projects", projectHandler.ListProjects)
api.POST("/projects", projectHandler.CreateProject)
api.GET("/projects/:id", projectHandler.GetProject)
api.PUT("/projects/:id", projectHandler.UpdateProject)
api.DELETE("/projects/:id", projectHandler.DeleteProject)
api.POST("/projects/:id/duplicate", projectHandler.DuplicateProject)
api.POST("/projects/:id/steps", projectHandler.AddStep)
api.PUT("/projects/:id/steps/:stepId", projectHandler.UpdateStep)
api.DELETE("/projects/:id/steps/:stepId", projectHandler.DeleteStep)
api.POST("/projects/:id/ai-recommend", projectHandler.AIRecommend)
api.POST("/projects/:id/finalize", projectHandler.FinalizeProject)
```

- [ ] **Step 6: 后端验证**

```bash
cd backend && go vet ./... && go build ./cmd/server
```
Expected: 无错误，编译成功。

- [ ] **Step 7: 提交**

```bash
GIT_MASTER=1 git add backend/internal/repository/gorm/models.go backend/internal/repository/gorm/gorm.go backend/internal/repository/gorm/project.go backend/internal/handler/project.go backend/cmd/server/main.go
GIT_MASTER=1 git commit -m "feat: 创作项目后端 — Project/ProjectStep CRUD API + AI 推荐 + 定稿" -m "新增 Project/ProjectStep GORM 模型、ProjectRepository（含复制/步骤管理）、ProjectHandler（12 个端点）。创意简报 AI 推荐复用 chat API。" -m "Ultraworked with [Sisyphus](https://github.com/code-yeongyu/oh-my-openagent)" -m "Co-authored-by: Sisyphus <clio-agent@sisyphuslabs.ai>"
```

---

### Task 2: 前端 — 项目 API 客户端 + 项目列表页

**Files:**
- Create: `frontend/src/api/projects.ts`
- Create: `frontend/src/views/ProjectList.vue`

**Interfaces:**
- Consumes: Backend Project API from Task 1
- Produces: `ProjectList.vue` — 项目入口页面，展示所有项目的卡片列表

- [ ] **Step 1: 创建项目 API 客户端**

`frontend/src/api/projects.ts`：

```typescript
import client from './client'

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
  created_at: string
  updated_at: string
  steps?: ProjectStep[]
}

export interface ProjectStep {
  id: number
  project_id: number
  step_type: 'generate' | 'refine' | 'finalize'
  position: number
  input: string
  output: string
  created_at: string
}

export async function getProjects(): Promise<Project[]> {
  const res = await client.get('/api/v1/projects')
  return res.data
}

export async function getProject(id: number): Promise<Project> {
  const res = await client.get(`/api/v1/projects/${id}`)
  return res.data
}

export async function createProject(data: { title: string; brief?: string }): Promise<Project> {
  const res = await client.post('/api/v1/projects', data)
  return res.data
}

export async function updateProject(id: number, data: Partial<Project>): Promise<void> {
  await client.put(`/api/v1/projects/${id}`, data)
}

export async function deleteProject(id: number): Promise<void> {
  await client.delete(`/api/v1/projects/${id}`)
}

export async function duplicateProject(id: number): Promise<Project> {
  const res = await client.post(`/api/v1/projects/${id}/duplicate`)
  return res.data
}

export async function addStep(projectId: number, data: { step_type: string; input?: string }): Promise<ProjectStep> {
  const res = await client.post(`/api/v1/projects/${projectId}/steps`, data)
  return res.data
}

export async function updateStep(projectId: number, stepId: number, data: { input?: string; output?: string }): Promise<void> {
  await client.put(`/api/v1/projects/${projectId}/steps/${stepId}`, data)
}

export async function deleteStep(projectId: number, stepId: number): Promise<void> {
  await client.delete(`/api/v1/projects/${projectId}/steps/${stepId}`)
}

export async function aiRecommend(projectId: number): Promise<{ ai_result: string }> {
  const res = await client.post(`/api/v1/projects/${projectId}/ai-recommend`)
  return res.data
}

export async function finalizeProject(projectId: number, data: { final_url: string; notes?: string }): Promise<void> {
  await client.post(`/api/v1/projects/${projectId}/finalize`, data)
}
```

- [ ] **Step 2: 创建项目列表页**

`frontend/src/views/ProjectList.vue`：

```vue
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRouter } from 'vue-router'
import { getProjects, createProject, deleteProject, duplicateProject, type Project } from '../api/projects'

const router = useRouter()
const projects = ref<Project[]>([])
const loading = ref(false)
const showCreateDialog = ref(false)
const newTitle = ref('')
const newBrief = ref('')

async function loadProjects() {
  loading.value = true
  try {
    projects.value = await getProjects()
  } catch (e: any) {
    ElMessage.error(e.message || '加载项目列表失败')
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  if (!newTitle.value.trim()) return
  try {
    const project = await createProject({ title: newTitle.value.trim(), brief: newBrief.value.trim() })
    ElMessage.success('项目创建成功')
    showCreateDialog.value = false
    newTitle.value = ''
    newBrief.value = ''
    router.push({ name: 'project_editor', params: { id: project.id } })
  } catch (e: any) {
    ElMessage.error(e.message || '创建失败')
  }
}

async function handleDelete(project: Project) {
  try {
    await ElMessageBox.confirm(`确定删除项目「${project.title}」？`, '提示')
    await deleteProject(project.id)
    ElMessage.success('已删除')
    await loadProjects()
  } catch { /* 取消或失败均静默 */ }
}

async function handleDuplicate(project: Project) {
  try {
    const dup = await duplicateProject(project.id)
    ElMessage.success('已复制')
    await loadProjects()
    router.push({ name: 'project_editor', params: { id: dup.id } })
  } catch (e: any) {
    ElMessage.error(e.message || '复制失败')
  }
}

function openEditor(project: Project) {
  router.push({ name: 'project_editor', params: { id: project.id } })
}

const statusLabels: Record<string, string> = {
  draft: '草稿',
  generating: '生成中',
  refining: '精修中',
  completed: '已完成',
}

onMounted(loadProjects)
</script>

<template>
  <div class="project-list">
    <div class="page-header">
      <h2>创作项目</h2>
      <el-button type="primary" @click="showCreateDialog = true">新建项目</el-button>
    </div>

    <!-- 空状态 -->
    <el-empty v-if="!loading && projects.length === 0" description="还没有创作项目，点击上方按钮创建一个" />

    <!-- 项目卡片网格 -->
    <div v-if="projects.length > 0" class="project-grid">
      <el-card
        v-for="p in projects"
        :key="p.id"
        :body-style="{ padding: '0' }"
        shadow="hover"
        class="project-card"
        @click="openEditor(p)"
      >
        <!-- 封面图 -->
        <div class="card-cover">
          <img v-if="p.cover_url" :src="p.cover_url" alt="" />
          <div v-else class="card-cover-placeholder">
            <span>{{ p.title.charAt(0) }}</span>
          </div>
          <el-tag :type="p.status === 'completed' ? 'success' : 'info'" class="card-status">
            {{ statusLabels[p.status] || p.status }}
          </el-tag>
        </div>
        <div class="card-body">
          <h3 class="card-title">{{ p.title }}</h3>
          <p v-if="p.brief" class="card-brief">{{ p.brief }}</p>
          <p class="card-time">{{ p.updated_at?.slice(0, 10) || '' }}</p>
        </div>
        <div class="card-actions" @click.stop>
          <el-button size="small" @click="openEditor(p)">编辑</el-button>
          <el-button size="small" @click="handleDuplicate(p)">复制</el-button>
          <el-button size="small" type="danger" @click="handleDelete(p)">删除</el-button>
        </div>
      </el-card>
    </div>

    <!-- 新建项目弹窗 -->
    <el-dialog v-model="showCreateDialog" title="新建创作项目" width="500px">
      <el-form label-width="80px">
        <el-form-item label="项目名称" required>
          <el-input v-model="newTitle" placeholder="例如：三月社媒封面系列" />
        </el-form-item>
        <el-form-item label="创意简报">
          <el-input v-model="newBrief" type="textarea" :rows="3" placeholder="一句话描述你的创意，例如：一只橘猫在樱花树下睡觉，日系动漫风格" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <el-button type="primary" @click="handleCreate" :disabled="!newTitle.trim()">创建并进入</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.project-list {
  max-width: 1200px;
}
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}
.page-header h2 {
  margin: 0;
  font-size: 20px;
}
.project-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}
.project-card {
  cursor: pointer;
  border-radius: 12px;
  overflow: hidden;
  transition: transform 0.15s;
}
.project-card:hover {
  transform: translateY(-2px);
}
.card-cover {
  height: 160px;
  background: #f5f5f5;
  position: relative;
  overflow: hidden;
}
.card-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.card-cover-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 48px;
  color: #ccc;
  background: linear-gradient(135deg, #f5f5f5, #e8e8e8);
}
.card-status {
  position: absolute;
  top: 8px;
  right: 8px;
}
.card-body {
  padding: 16px;
}
.card-title {
  margin: 0 0 4px;
  font-size: 15px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.card-brief {
  margin: 0 0 4px;
  font-size: 13px;
  color: #888;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.card-time {
  margin: 0;
  font-size: 12px;
  color: #bbb;
}
.card-actions {
  padding: 8px 16px 12px;
  display: flex;
  gap: 8px;
}
</style>
```

- [ ] **Step 3: 验证前端构建**

```bash
cd frontend && pnpm build 2>&1 | tail -20
```
Expected: 构建成功。

- [ ] **Step 4: 提交**

```bash
GIT_MASTER=1 git add frontend/src/api/projects.ts frontend/src/views/ProjectList.vue
GIT_MASTER=1 git commit -m "feat: 创作项目前端 — API 客户端 + 项目列表页" -m "创建 projects.ts API 客户端（12 个端点）、ProjectList.vue 卡片式列表页（新建/编辑/复制/删除）。" -m "Ultraworked with [Sisyphus](https://github.com/code-yeongyu/oh-my-openagent)" -m "Co-authored-by: Sisyphus <clio-agent@sisyphuslabs.ai>"
```

---

### Task 3: 前端 — 项目编辑器（4 步流程）

**Files:**
- Create: `frontend/src/views/ProjectEditor.vue` — 编辑器容器（含步骤导航）
- Create: `frontend/src/components/ProjectBrief.vue` — 步骤 1：创意简报 + AI 推荐
- Create: `frontend/src/components/ProjectGenerate.vue` — 步骤 2：生成+对比
- Create: `frontend/src/components/ProjectRefine.vue` — 步骤 3：精修
- Create: `frontend/src/components/ProjectFinalize.vue` — 步骤 4：定稿

**Interfaces:**
- Consumes: Project API from Task 2

- [ ] **Step 1: 创建 ProjectEditor（4 步容器）**

`frontend/src/views/ProjectEditor.vue`：

```vue
<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getProject, updateProject, type Project } from '../api/projects'
import ProjectBrief from '../components/ProjectBrief.vue'
import ProjectGenerate from '../components/ProjectGenerate.vue'
import ProjectRefine from '../components/ProjectRefine.vue'
import ProjectFinalize from '../components/ProjectFinalize.vue'

const route = useRoute()
const router = useRouter()
const project = ref<Project | null>(null)
const loading = ref(true)
const currentStep = ref(1)
const totalSteps = 4

const stepTitles = ['创意简报', '生成+对比', '精修', '定稿']

async function loadProject() {
  const id = Number(route.params.id)
  if (!id) {
    ElMessage.error('项目 ID 无效')
    router.push({ name: 'projects' })
    return
  }
  try {
    project.value = await getProject(id)
  } catch (e: any) {
    ElMessage.error(e.message || '加载项目失败')
    router.push({ name: 'projects' })
  } finally {
    loading.value = false
  }
}

function goToStep(step: number) {
  if (step >= 1 && step <= totalSteps) {
    currentStep.value = step
  }
}

async function handleSave(data: Partial<Project>) {
  if (!project.value) return
  try {
    await updateProject(project.value.id, data)
    project.value = { ...project.value, ...data }
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  }
}

onMounted(loadProject)
</script>

<template>
  <div class="project-editor" v-if="project">
    <!-- 顶部导航条 -->
    <div class="editor-header">
      <el-button text @click="router.push({ name: 'projects' })">← 返回项目列表</el-button>
      <h2 class="editor-title">{{ project.title }}</h2>
      <el-tag>{{ project.status === 'completed' ? '已完成' : '进行中' }}</el-tag>
    </div>

    <!-- 步骤进度条 -->
    <el-steps :active="currentStep - 1" align-center class="step-bar">
      <el-step v-for="(title, idx) in stepTitles" :key="idx" :title="title" />
    </el-steps>

    <!-- 步骤内容 -->
    <div class="step-content" v-if="!loading">
      <ProjectBrief
        v-if="currentStep === 1"
        :project="project"
        @save="handleSave"
        @next="goToStep(2)"
      />
      <ProjectGenerate
        v-else-if="currentStep === 2"
        :project="project"
        @save="handleSave"
        @prev="goToStep(1)"
        @next="goToStep(3)"
      />
      <ProjectRefine
        v-else-if="currentStep === 3"
        :project="project"
        @save="handleSave"
        @prev="goToStep(2)"
        @next="goToStep(4)"
      />
      <ProjectFinalize
        v-else-if="currentStep === 4"
        :project="project"
        @save="handleSave"
        @prev="goToStep(3)"
        @done="router.push({ name: 'projects' })"
      />
    </div>
  </div>
  <div v-else-if="loading" style="text-align:center;padding:40px">
    <el-skeleton :rows="3" animated />
  </div>
</template>

<style scoped>
.project-editor {
  max-width: 1000px;
}
.editor-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 24px;
}
.editor-title {
  flex: 1;
  margin: 0;
  font-size: 18px;
}
.step-bar {
  margin-bottom: 24px;
}
.step-content {
  min-height: 400px;
}
</style>
```

- [ ] **Step 2: 创建 ProjectBrief（创意简报 + AI 推荐）**

`frontend/src/components/ProjectBrief.vue`：

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { aiRecommend, updateProject, type Project } from '../api/projects'

const props = defineProps<{ project: Project }>()
const emit = defineEmits<{ save: [data: Partial<Project>]; next: [] }>()

const brief = ref(props.project.brief || '')
const aiResult = ref<any>(null)
const loadingAI = ref(false)

function parseAIResult(text: string) {
  try {
    return JSON.parse(text)
  } catch {
    return { enhanced_prompt: text, style_suggestions: '', size_suggestion: '', model_suggestion: '' }
  }
}

async function handleAIRecommend() {
  if (!brief.value.trim()) {
    ElMessage.warning('请先填写创意简报')
    return
  }
  loadingAI.value = true
  try {
    const result = await aiRecommend(props.project.id)
    aiResult.value = parseAIResult(result.ai_result)
  } catch (e: any) {
    ElMessage.error(e.message || 'AI 推荐失败')
  } finally {
    loadingAI.value = false
  }
}

async function handleConfirm() {
  await updateProject(props.project.id, {
    brief: brief.value,
    ai_result: aiResult.value ? JSON.stringify(aiResult.value) : '',
  })
  emit('save', { brief: brief.value })
  emit('next')
}
</script>

<template>
  <div class="brief-step">
    <h3>创意简报</h3>
    <p class="step-desc">用一句话描述你想要的画面，AI 会为你推荐风格、尺寸、模型和 prompt。</p>

    <el-input
      v-model="brief"
      type="textarea"
      :rows="3"
      placeholder="例如：一只橘猫在樱花树下睡觉，日系动漫风格"
    />

    <div style="margin: 12px 0;">
      <el-button type="primary" :loading="loadingAI" @click="handleAIRecommend">
        AI 智能推荐
      </el-button>
    </div>

    <!-- AI 推荐结果 -->
    <el-card v-if="aiResult" class="ai-result">
      <template #header><strong>AI 推荐</strong></template>
      <div class="ai-field"><label>扩写 Prompt：</label><p>{{ aiResult.enhanced_prompt }}</p></div>
      <div class="ai-field"><label>推荐风格：</label><span>{{ aiResult.style_suggestions }}</span></div>
      <div class="ai-field"><label>推荐尺寸：</label><span>{{ aiResult.size_suggestion }}</span></div>
      <div class="ai-field"><label>推荐模型：</label><span>{{ aiResult.model_suggestion }}</span></div>
    </el-card>

    <div class="step-actions">
      <el-button type="primary" @click="handleConfirm" :disabled="!brief.trim()">
        确认，进入生成
      </el-button>
    </div>
  </div>
</template>

<style scoped>
.brief-step { max-width: 700px; }
.step-desc { color: #888; font-size: 14px; margin-bottom: 16px; }
.ai-result { margin: 16px 0; }
.ai-field { margin-bottom: 8px; }
.ai-field label { font-weight: 500; font-size: 13px; color: #555; }
.ai-field p, .ai-field span { margin: 2px 0 0; font-size: 14px; }
.step-actions { margin-top: 24px; display: flex; gap: 8px; }
</style>
```

- [ ] **Step 3: 创建 ProjectGenerate（生成+对比）**

`frontend/src/components/ProjectGenerate.vue`：

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { addStep, updateStep, type Project } from '../api/projects'
import { images } from '../api/image'

const props = defineProps<{ project: Project }>()
const emit = defineEmits<{ save: [data: Partial<Project>]; prev: []; next: [] }>()

const prompt = ref('')
const size = ref('1:1')
const count = ref(4)
const generatedImages = ref<string[]>([])
const selectedImage = ref<string | null>(null)
const generating = ref(false)
const sizes = ['1:1', '16:9', '3:4', '4:3', '9:16']

// 从 AI 结果预填参数
function initFromAIResult() {
  if (!props.project.ai_result) return
  try {
    const ai = JSON.parse(props.project.ai_result)
    if (ai.enhanced_prompt) prompt.value = ai.enhanced_prompt
    if (ai.size_suggestion) size.value = ai.size_suggestion
  } catch { /* 忽略 */ }
}
initFromAIResult()

async function handleGenerate() {
  if (!prompt.value.trim()) {
    ElMessage.warning('请填写 prompt')
    return
  }
  generating.value = true
  try {
    const res = await images.textToImage({ prompt: prompt.value, size: size.value, n: count.value })
    const urls: string[] = res.images || []
    generatedImages.value = urls

    // 保存步骤
    const step = await addStep(props.project.id, {
      step_type: 'generate',
      input: JSON.stringify({ prompt: prompt.value, size: size.value, count: count.value }),
    })
    await updateStep(props.project.id, step.id, {
      output: JSON.stringify({ images: urls, selected: [] }),
    })
  } catch (e: any) {
    ElMessage.error(e.message || '生成失败')
  } finally {
    generating.value = false
  }
}

function toggleSelect(url: string) {
  selectedImage.value = selectedImage.value === url ? null : url
}

async function handleNext() {
  if (!selectedImage.value) {
    ElMessage.warning('请先选择一张候选图')
    return
  }
  emit('next')
}
</script>

<template>
  <div class="generate-step">
    <h3>生成+对比</h3>

    <el-form label-width="100px">
      <el-form-item label="Prompt">
        <el-input v-model="prompt" type="textarea" :rows="3" placeholder="输入图片描述" />
      </el-form-item>
      <el-form-item label="尺寸">
        <el-select v-model="size">
          <el-option v-for="s in sizes" :key="s" :label="s" :value="s" />
        </el-select>
      </el-form-item>
      <el-form-item label="生成数量">
        <el-radio-group v-model="count">
          <el-radio :value="2">2 张</el-radio>
          <el-radio :value="4">4 张</el-radio>
        </el-radio-group>
      </el-form-item>
    </el-form>

    <el-button type="primary" :loading="generating" @click="handleGenerate">
      生成
    </el-button>

    <!-- 结果展示 -->
    <div v-if="generatedImages.length > 0" class="result-grid">
      <div
        v-for="(url, idx) in generatedImages"
        :key="idx"
        class="result-card"
        :class="{ selected: selectedImage === url }"
        @click="toggleSelect(url)"
      >
        <el-image :src="url" fit="contain" style="width:100%;height:200px" />
        <div class="result-check" v-if="selectedImage === url">✓ 已选</div>
      </div>
    </div>

    <div class="step-actions">
      <el-button @click="emit('prev')">上一步</el-button>
      <el-button type="primary" @click="handleNext" :disabled="!selectedImage">
        选好了，去精修
      </el-button>
    </div>
  </div>
</template>

<style scoped>
.generate-step { max-width: 800px; }
.result-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 12px;
  margin-top: 16px;
}
.result-card {
  border: 2px solid #e0e0e0;
  border-radius: 8px;
  padding: 8px;
  cursor: pointer;
  transition: border-color 0.15s;
  position: relative;
}
.result-card:hover { border-color: #409eff; }
.result-card.selected { border-color: #67c23a; }
.result-check {
  position: absolute;
  top: 4px;
  right: 4px;
  background: #67c23a;
  color: white;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
}
.step-actions { margin-top: 24px; display: flex; gap: 8px; }
</style>
```

注意：这里的 `import { images } from '../api/image'` 需要确认 `image.ts` 的导出。如果 `image.ts` 导出的是独立函数而非命名空间，修改为：

```typescript
import { textToImage } from '../api/image'
// 使用时直接调用 textToImage({...})
```

- [ ] **Step 4: 创建 ProjectRefine（精修）**

`frontend/src/components/ProjectRefine.vue`：

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { addStep, updateStep, type Project } from '../api/projects'

const props = defineProps<{ project: Project }>()
const emit = defineEmits<{ save: [data: Partial<Project>]; prev: []; next: [] }>()

const imageUrl = ref('')
const prompt = ref('')
const strength = ref(0.75)
const newImages = ref<string[]>([])
const selectedFinal = ref<string | null>(null)
const refining = ref(false)

async function handleRefine() {
  if (!imageUrl.value) {
    ElMessage.warning('请先选择要精修的图片')
    return
  }
  refining.value = true
  try {
    // 调图生图 API
    const form = new FormData()
    form.append('image', imageUrl.value)
    form.append('prompt', prompt.value)
    form.append('strength', String(strength.value))
    const res = await fetch('/api/v1/images/image-to-image', {
      method: 'POST',
      body: form,
    })
    const data = await res.json()
    const urls: string[] = data.images || []
    newImages.value = urls

    const step = await addStep(props.project.id, {
      step_type: 'refine',
      input: JSON.stringify({ image_url: imageUrl.value, prompt: prompt.value, strength: strength.value }),
    })
    await updateStep(props.project.id, step.id, {
      output: JSON.stringify({ images: urls, selected: selectedFinal.value }),
    })
  } catch (e: any) {
    ElMessage.error(e.message || '精修失败')
  } finally {
    refining.value = false
  }
}

function pickAsFinal(url: string) {
  selectedFinal.value = url
}

async function handleNext() {
  if (!selectedFinal.value) {
    ElMessage.warning('请选择最终版本')
    return
  }
  emit('next')
}
</script>

<template>
  <div class="refine-step">
    <h3>精修</h3>
    <p class="step-desc">对上一步选中的图片进行迭代优化。</p>

    <el-form label-width="100px">
      <el-form-item label="图片 URL">
        <el-input v-model="imageUrl" placeholder="上一步选中的图片 URL （自动填入）" />
      </el-form-item>
      <el-form-item label="Prompt">
        <el-input v-model="prompt" type="textarea" :rows="2" placeholder="可修改 prompt" />
      </el-form-item>
      <el-form-item label="强度">
        <el-slider v-model="strength" :min="0" :max="1" :step="0.05" style="width: 300px" />
      </el-form-item>
    </el-form>

    <el-button type="primary" :loading="refining" @click="handleRefine">
      开始精修
    </el-button>

    <div v-if="newImages.length > 0" class="result-grid">
      <div
        v-for="(url, idx) in newImages"
        :key="idx"
        class="result-card"
        :class="{ selected: selectedFinal === url }"
        @click="pickAsFinal(url)"
      >
        <el-image :src="url" fit="contain" style="width:100%;height:200px" />
        <div class="result-check" v-if="selectedFinal === url">✓ 最终版</div>
      </div>
    </div>

    <div class="step-actions">
      <el-button @click="emit('prev')">上一步</el-button>
      <el-button type="primary" @click="handleNext" :disabled="!selectedFinal">
        选好了，去定稿
      </el-button>
    </div>
  </div>
</template>

<style scoped>
.refine-step { max-width: 800px; }
.step-desc { color: #888; font-size: 14px; margin-bottom: 16px; }
.result-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 12px;
  margin-top: 16px;
}
.result-card {
  border: 2px solid #e0e0e0;
  border-radius: 8px;
  padding: 8px;
  cursor: pointer;
  transition: border-color 0.15s;
  position: relative;
}
.result-card:hover { border-color: #409eff; }
.result-card.selected { border-color: #67c23a; }
.result-check {
  position: absolute;
  top: 4px;
  right: 4px;
  background: #67c23a;
  color: white;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
}
.step-actions { margin-top: 24px; display: flex; gap: 8px; }
</style>
```

- [ ] **Step 5: 创建 ProjectFinalize（定稿）**

`frontend/src/components/ProjectFinalize.vue`：

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { finalizeProject, type Project } from '../api/projects'

const props = defineProps<{ project: Project }>()
const emit = defineEmits<{ save: [data: Partial<Project>]; prev: []; done: [] }>()

const notes = ref('')
const finalizing = ref(false)

async function handleFinalize() {
  if (!props.project.final_url) {
    ElMessage.warning('还未选择最终版本，请先完成精修步骤')
    return
  }
  finalizing.value = true
  try {
    await finalizeProject(props.project.id, {
      final_url: props.project.final_url,
      notes: notes.value,
    })
    ElMessage.success('定稿成功！已保存到作品库')
    emit('done')
  } catch (e: any) {
    ElMessage.error(e.message || '定稿失败')
  } finally {
    finalizing.value = false
  }
}
</script>

<template>
  <div class="finalize-step">
    <h3>定稿</h3>
    <p class="step-desc">确认最终版本，自动保存到作品库。</p>

    <el-card v-if="project.final_url" class="final-preview">
      <el-image :src="project.final_url" fit="contain" style="width:100%;max-height:300px" />
    </el-card>

    <el-form label-width="80px" style="margin-top: 16px;">
      <el-form-item label="最终 URL">
        <el-input :model-value="project.final_url" disabled />
      </el-form-item>
      <el-form-item label="备注">
        <el-input v-model="notes" type="textarea" :rows="2" placeholder="添加备注信息（可选）" />
      </el-form-item>
    </el-form>

    <div class="step-actions">
      <el-button @click="emit('prev')">返回精修</el-button>
      <el-button type="success" :loading="finalizing" @click="handleFinalize">
        确认定稿
      </el-button>
    </div>
  </div>
</template>

<style scoped>
.finalize-step { max-width: 600px; }
.step-desc { color: #888; font-size: 14px; margin-bottom: 16px; }
.final-preview { margin-bottom: 16px; }
.step-actions { margin-top: 24px; display: flex; gap: 8px; }
</style>
```

- [ ] **Step 6: 验证前端构建**

```bash
cd frontend && pnpm build 2>&1 | tail -30
```
Expected: 构建成功。

- [ ] **Step 7: 提交**

```bash
GIT_MASTER=1 git add frontend/src/views/ProjectEditor.vue frontend/src/components/ProjectBrief.vue frontend/src/components/ProjectGenerate.vue frontend/src/components/ProjectRefine.vue frontend/src/components/ProjectFinalize.vue
GIT_MASTER=1 git commit -m "feat: 创作项目编辑器 — 4 步流程（简报/生成/精修/定稿）" -m "ProjectEditor 主容器 + 4 个子组件。创意简报 AI 推荐、生成+对比多图选择、精修迭代优化、定稿自动入库。" -m "Ultraworked with [Sisyphus](https://github.com/code-yeongyu/oh-my-openagent)" -m "Co-authored-by: Sisyphus <clio-agent@sisyphuslabs.ai>"
```

---

### Task 4: 前端 — 导航更新（NavSidebar + 路由 + App.vue）

**Files:**
- Modify: `frontend/src/components/NavSidebar.vue` — 替换「创作」组
- Modify: `frontend/src/App.vue` — 添加 ProjectList + ProjectEditor 路由
- Modify: `frontend/src/router/index.ts` — 注册新路由

- [ ] **Step 1: 修改 NavSidebar — 替换「创作」组**

将 `workflow` 组的内容改为单一的「创作项目」入口：

```typescript
{
  id: 'workflow',
  icon: '⚡',
  label: '创作',
  items: [
    { id: 'projects', label: '创作项目' },
  ],
},
```

- [ ] **Step 2: 修改 router/index.ts**

添加两个新路由：

```typescript
{ path: '/projects',         name: 'projects',       component: () => import('../views/ProjectList.vue') },
{ path: '/projects/:id',     name: 'project_editor', component: () => import('../views/ProjectEditor.vue') },
```

- [ ] **Step 3: 修改 App.vue**

添加 import：
```typescript
import ProjectList from './views/ProjectList.vue'
import ProjectEditor from './views/ProjectEditor.vue'
```

在模板的 `v-if` 链中添加：
```vue
<ProjectList v-else-if="activePage === 'projects'" />
<ProjectEditor v-else-if="activePage === 'project_editor'" />
```

- [ ] **Step 4: 验证前端构建**

```bash
cd frontend && pnpm build 2>&1 | tail -20
```
Expected: 构建成功。

- [ ] **Step 5: 提交**

```bash
GIT_MASTER=1 git add frontend/src/components/NavSidebar.vue frontend/src/router/index.ts frontend/src/App.vue
GIT_MASTER=1 git commit -m "feat: 创作区导航更新 — 替换创作组为项目式入口" -m "NavSidebar 创作组改为「创作项目」入口，注册 ProjectList + ProjectEditor 路由。" -m "Ultraworked with [Sisyphus](https://github.com/code-yeongyu/oh-my-openagent)" -m "Co-authored-by: Sisyphus <clio-agent@sisyphuslabs.ai>"
```

---

### 最终验证

```bash
# 后端
cd backend && go vet ./... && go build ./cmd/server
# 前端
cd frontend && pnpm build
```

Expected: 都通过。
