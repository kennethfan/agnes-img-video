package handler

import (
	"database/sql"
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

func (h *StoryboardHandler) ListProjects(c *gin.Context) {
	projects, err := h.repo.ListProjects()
	if err != nil {
		log.Printf("[Storyboard] 获取项目列表失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取项目列表失败: " + err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目ID"})
		return
	}

	project, err := h.repo.GetProject(id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
			return
		}
		log.Printf("[Storyboard] 查询项目失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询项目失败"})
		return
	}

	shots, err := h.repo.ListShots(id)
	if err != nil {
		log.Printf("[Storyboard] 获取镜头列表失败: %v", err)
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建项目失败: " + err.Error()})
		return
	}

	project, err := h.repo.GetProject(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建成功但获取项目失败"})
		return
	}

	c.JSON(http.StatusCreated, project)
}

func (h *StoryboardHandler) UpdateProject(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目ID"})
		return
	}

	var req model.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	if err := h.repo.UpdateProject(id, req.Title, req.Script); err != nil {
		log.Printf("[Storyboard] 更新项目失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新项目失败: " + err.Error()})
		return
	}

	project, err := h.repo.GetProject(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新成功但获取项目失败"})
		return
	}

	c.JSON(http.StatusOK, project)
}

func (h *StoryboardHandler) DeleteProject(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目ID"})
		return
	}

	if err := h.repo.DeleteProject(id); err != nil {
		log.Printf("[Storyboard] 删除项目失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除项目失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *StoryboardHandler) DuplicateProject(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目ID"})
		return
	}

	newID, err := h.repo.DuplicateProject(id)
	if err != nil {
		log.Printf("[Storyboard] 复制项目失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "复制项目失败: " + err.Error()})
		return
	}

	project, err := h.repo.GetProject(newID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "复制成功但获取项目失败"})
		return
	}

	c.JSON(http.StatusCreated, project)
}

func (h *StoryboardHandler) CreateShot(c *gin.Context) {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目ID"})
		return
	}

	var req model.CreateShotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	shots, err := h.repo.ListShots(projectID)
	if err != nil {
		shots = []model.StoryboardShot{}
	}
	seq := len(shots) + 1

	shotID, err := h.repo.CreateShot(projectID, seq, req.Prompt, req.Type, req.ReferenceImage)
	if err != nil {
		log.Printf("[Storyboard] 创建镜头失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建镜头失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": shotID, "sequence": seq})
}

func (h *StoryboardHandler) UpdateShot(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的镜头ID"})
		return
	}

	var req model.UpdateShotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	if err := h.repo.UpdateShot(id, req.Prompt, req.Type, req.ReferenceImage); err != nil {
		log.Printf("[Storyboard] 更新镜头失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新镜头失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *StoryboardHandler) DeleteShot(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的镜头ID"})
		return
	}

	if err := h.repo.DeleteShot(id); err != nil {
		log.Printf("[Storyboard] 删除镜头失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除镜头失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *StoryboardHandler) ReorderShots(c *gin.Context) {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目ID"})
		return
	}

	var req model.ReorderShotsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	if err := h.repo.ReorderShots(req.IDs); err != nil {
		log.Printf("[Storyboard] 重排镜头失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "重排镜头失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
	_ = projectID
}

func (h *StoryboardHandler) GenerateShots(c *gin.Context) {
	projectID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的项目ID"})
		return
	}

	shots, err := h.repo.ListShots(projectID)
	if err != nil {
		log.Printf("[Storyboard] 获取镜头列表失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取镜头列表失败: " + err.Error()})
		return
	}

	pending := make([]model.StoryboardShot, 0)
	for _, s := range shots {
		if s.Status == "pending" {
			pending = append(pending, s)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total":   len(shots),
		"pending": len(pending),
		"shots":   pending,
	})
}
