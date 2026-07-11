package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/agnes-image-tool/backend/internal/repository/gorm"
)

type CollectionHandler struct {
	repo *gorm.CollectionRepository
}

func NewCollectionHandler(repo *gorm.CollectionRepository) *CollectionHandler {
	return &CollectionHandler{repo: repo}
}

// ListCollections GET /api/v1/collections
func (h *CollectionHandler) ListCollections(c *gin.Context) {
	collections, err := h.repo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询集合失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, collections)
}

// CreateCollection POST /api/v1/collections
func (h *CollectionHandler) CreateCollection(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	collection, err := h.repo.Create(req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建集合失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, collection)
}

// UpdateCollection PUT /api/v1/collections/:id
func (h *CollectionHandler) UpdateCollection(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	if err := h.repo.Update(uint(id), req.Name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新集合失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// DeleteCollection DELETE /api/v1/collections/:id
func (h *CollectionHandler) DeleteCollection(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.repo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除集合失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// AddAssetsToCollection POST /api/v1/collections/:id/assets
func (h *CollectionHandler) AddAssetsToCollection(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		AssetIDs []uint `json:"asset_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	if err := h.repo.AddAssets(uint(id), req.AssetIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "添加资产到集合失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "添加成功"})
}

// RemoveAssetsFromCollection DELETE /api/v1/collections/:id/assets
func (h *CollectionHandler) RemoveAssetsFromCollection(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		AssetIDs []uint `json:"asset_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	if err := h.repo.RemoveAssets(uint(id), req.AssetIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "从集合移除资产失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "移除成功"})
}
