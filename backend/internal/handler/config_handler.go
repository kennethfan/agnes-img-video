package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/agnes-image-tool/backend/internal/config"
	"github.com/agnes-image-tool/backend/internal/model"
)

type ConfigHandler struct {
	configPath string
}

func NewConfigHandler(cfgPath string) *ConfigHandler {
	return &ConfigHandler{configPath: cfgPath}
}

// GetConfig 获取当前配置
// GET /api/v1/config
func (h *ConfigHandler) GetConfig(c *gin.Context) {
	cfg := config.GetConfig()
	// 不返回 API Key / Github Token（安全原因）
	c.JSON(http.StatusOK, gin.H{
		"base_url":      cfg.BaseURL,
		"model":         cfg.Model,
		"github_repo":   cfg.GithubRepo,
		"github_branch": cfg.GithubBranch,
	})
}

// UpdateConfig 更新配置
// PUT /api/v1/config
func (h *ConfigHandler) UpdateConfig(c *gin.Context) {
	var req model.Config
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// 合并当前配置（只更新提供的字段）
	current := config.GetConfig()
	if req.APIKey != "" {
		current.APIKey = req.APIKey
	}
	if req.BaseURL != "" {
		current.BaseURL = req.BaseURL
	}
	if req.Model != "" {
		current.Model = req.Model
	}
	if req.GithubToken != "" {
		current.GithubToken = req.GithubToken
	}
	if req.GithubRepo != "" {
		current.GithubRepo = req.GithubRepo
	}
	if req.GithubBranch != "" {
		current.GithubBranch = req.GithubBranch
	}

	if err := config.SaveConfig(h.configPath, current); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存配置失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
