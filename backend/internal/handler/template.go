package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	gormrepo "github.com/agnes-image-tool/backend/internal/repository/gorm"
)

type TemplateHandler struct {
	repo *gormrepo.TemplateRepository
}

func NewTemplateHandler(repo *gormrepo.TemplateRepository) *TemplateHandler {
	return &TemplateHandler{repo: repo}
}

func (h *TemplateHandler) ListTemplates(c *gin.Context) {
	category := c.Query("category")
	templates, err := h.repo.List(category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询模板失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, templates)
}

func (h *TemplateHandler) CreateTemplate(c *gin.Context) {
	var req gormrepo.PromptTemplate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	if err := h.repo.Create(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建模板失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, req)
}

func (h *TemplateHandler) UpdateTemplate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req gormrepo.PromptTemplate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	req.ID = id
	if err := h.repo.Update(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新模板失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

func (h *TemplateHandler) DeleteTemplate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.repo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除模板失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func (h *TemplateHandler) ExportTemplates(c *gin.Context) {
	templates, err := h.repo.Export()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "导出模板失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, templates)
}

func (h *TemplateHandler) ImportTemplates(c *gin.Context) {
	var templates []gormrepo.PromptTemplate
	if err := c.ShouldBindJSON(&templates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	if err := h.repo.Import(templates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "导入模板失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "导入成功", "count": len(templates)})
}

func (h *TemplateHandler) SaveFromHistory(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	record, err := historyRepo.GetRecordsByIDs([]int64{id})
	if err != nil || len(record) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "历史记录不存在"})
		return
	}
	r := record[0]
	extra, ok := r.Extra.(map[string]interface{})
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "历史记录缺少扩展参数"})
		return
	}

	tpl := gormrepo.PromptTemplate{
		Name:           r.Prompt,
		Type:           "template",
		Prompt:         r.Prompt,
		Category:       "自定义",
	}
	if s, ok := extra["size"].(string); ok {
		tpl.Size = s
	}
	if s, ok := extra["negative_prompt"].(string); ok {
		tpl.NegativePrompt = s
	}
	if s, ok := extra["model"].(string); ok {
		tpl.Model = s
	}
	if v, ok := extra["strength"].(float64); ok {
		tpl.Strength = v
	}

	if err := h.repo.Create(&tpl); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存模板失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, tpl)
}
