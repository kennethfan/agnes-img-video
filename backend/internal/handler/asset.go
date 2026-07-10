package handler

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/agnes-image-tool/backend/internal/model"
	"github.com/agnes-image-tool/backend/internal/repository"
)

type AssetHandler struct {
	repo repository.AssetRepository
}

func NewAssetHandler(repo repository.AssetRepository) *AssetHandler {
	return &AssetHandler{repo: repo}
}

// SaveAsset 保存到作品库
func (h *AssetHandler) SaveAsset(c *gin.Context) {
	var req struct {
		ImageURL string `json:"image_url"`
		Prompt   string `json:"prompt"`
		Mode     string `json:"mode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	if req.ImageURL == "" || req.Prompt == "" || req.Mode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不完整"})
		return
	}

	// 判断类型
	videoModes := map[string]bool{
		"text2video":        true,
		"image2video":       true,
		"multi_image_video": true,
	}
	assetType := "image"
	if videoModes[req.Mode] {
		assetType = "video"
	}

	// 解析本地路径
	var localPath string
	if !strings.HasPrefix(req.ImageURL, "http://") && !strings.HasPrefix(req.ImageURL, "https://") {
		// 本地路径：尝试 outputs/ 和 backend/outputs/
		candidates := []string{
			req.ImageURL,
			filepath.Join("outputs", filepath.Base(req.ImageURL)),
		}
		for _, p := range candidates {
			if _, err := os.Stat(p); err == nil {
				localPath = p
				break
			}
		}
		if localPath == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "图片文件不存在"})
			return
		}
	}

	asset := &model.Asset{
		Mode:        req.Mode,
		Prompt:      req.Prompt,
		Type:        assetType,
		Time:        time.Now().Format("2006-01-02 15:04:05"),
		Favorite:    false,
		OriginalURL: req.ImageURL,
		LocalPath:   localPath,
	}

	id, err := h.repo.Insert(asset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存到作品库失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id})
}

// ListAssets 列出作品库
func (h *AssetHandler) ListAssets(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	assetType := c.Query("type")
	search := c.Query("search")
	favoriteFilter := c.Query("favorite") == "true"

	if perPage <= 0 || perPage > 100 {
		perPage = 20
	}
	if page <= 0 {
		page = 1
	}

	assets, total, err := h.repo.List(page, perPage, assetType, search, favoriteFilter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询作品失败: " + err.Error()})
		return
	}

	items := make([]model.AssetItem, 0, len(assets))
	for _, a := range assets {
		thumbnail := a.OriginalURL
		if thumbnail == "" {
			thumbnail = a.LocalPath
		}
		items = append(items, model.AssetItem{
			ID:          a.ID,
			Mode:        a.Mode,
			Prompt:      a.Prompt,
			Type:        a.Type,
			Time:        a.Time,
			Favorite:    a.Favorite,
			OriginalURL: a.OriginalURL,
			LocalPath:   a.LocalPath,
			GitHubURL:   a.GitHubURL,
			Thumbnail:   thumbnail,
		})
	}

	c.JSON(http.StatusOK, model.AssetListResponse{
		Items: items,
		Total: total,
		Page:  page,
	})
}

// ToggleFavorite 切换收藏
func (h *AssetHandler) ToggleFavorite(c *gin.Context) {
	var req model.AssetFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	if err := h.repo.ToggleFavorite(req.AssetID, req.Favorite); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新收藏失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// BatchDownload 批量下载
func (h *AssetHandler) BatchDownload(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	assets, err := h.repo.GetByIDs(req.IDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询记录失败: " + err.Error()})
		return
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for _, a := range assets {
		if a.LocalPath == "" {
			continue
		}
		data, err := os.ReadFile(a.LocalPath)
		if err != nil {
			fallback := filepath.Join("outputs", filepath.Base(a.LocalPath))
			data, err = os.ReadFile(fallback)
			if err != nil {
				continue
			}
		}
		ext := filepath.Ext(a.LocalPath)
		entryName := fmt.Sprintf("%s_%d%s", a.Mode, a.ID, ext)
		f, err := zw.Create(entryName)
		if err != nil {
			continue
		}
		if _, err := io.Copy(f, bytes.NewReader(data)); err != nil {
			continue
		}
	}

	if err := zw.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建压缩文件失败: " + err.Error()})
		return
	}

	c.Header("Content-Disposition", "attachment; filename=assets.zip")
	c.Data(http.StatusOK, "application/zip", buf.Bytes())
}

// DeleteAssets 删除作品
func (h *AssetHandler) DeleteAssets(c *gin.Context) {
	var req model.AssetDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	if req.DeleteFiles {
		assets, err := h.repo.GetByIDs(req.IDs)
		if err == nil {
			for _, a := range assets {
				if a.LocalPath != "" {
					deleteRecordFiles([]string{a.LocalPath})
				}
			}
		}
	}

	if err := h.repo.Delete(req.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除作品失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
