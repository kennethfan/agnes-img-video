package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/agnes-image-tool/backend/internal/model"
	"github.com/agnes-image-tool/backend/internal/repository"
)

type SettingsHandler struct {
	repo *repository.SettingsRepo
}

func NewSettingsHandler(repo *repository.SettingsRepo) *SettingsHandler {
	return &SettingsHandler{repo: repo}
}

func (h *SettingsHandler) GetSettings(c *gin.Context) {
	s, err := h.repo.GetSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取设置失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, s)
}

func (h *SettingsHandler) UpdateSettings(c *gin.Context) {
	var s model.Settings
	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	if err := h.repo.UpdateSettings(&s); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存设置失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
