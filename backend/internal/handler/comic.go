package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/agnes-image-tool/backend/internal/service"
)

type ComicHandler struct {
	svc *service.AgnesClient
}

func NewComicHandler(svc *service.AgnesClient) *ComicHandler {
	return &ComicHandler{svc: svc}
}

// GeneratePrompts 生成漫画各格的画面提示词
// POST /api/v1/comic/generate-prompts
func (h *ComicHandler) GeneratePrompts(c *gin.Context) {
	var req struct {
		Theme      string `json:"theme" binding:"required"`
		Layout     string `json:"layout"`
		PanelCount int    `json:"panel_count" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	prompts, err := h.svc.GenerateComicPrompts(req.Theme, req.Layout, req.PanelCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"prompts": prompts})
}
