package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/agnes-image-tool/backend/internal/model"
	"github.com/agnes-image-tool/backend/internal/service"
)

type IdeasHandler struct {
	svc *service.AgnesClient
}

func NewIdeasHandler(svc *service.AgnesClient) *IdeasHandler {
	return &IdeasHandler{svc: svc}
}

// ExpandIdea AI 完善点子
// POST /api/v1/ideas/expand
func (h *IdeasHandler) ExpandIdea(c *gin.Context) {
	var req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
		Tags    string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	if req.Title == "" && req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "标题和内容不能都为空"})
		return
	}

	result, err := h.svc.ExpandIdea(req.Title, req.Content, req.Tags)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI 完善失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.ExpandIdeaResponse{Result: result})
}
