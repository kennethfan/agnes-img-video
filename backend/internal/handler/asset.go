package handler

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/agnes-image-tool/backend/internal/model"
	"github.com/agnes-image-tool/backend/internal/repository"
)

type AssetHandler struct {
	repo *repository.HistoryRepo
}

func NewAssetHandler(repo *repository.HistoryRepo) *AssetHandler {
	return &AssetHandler{repo: repo}
}

func (h *AssetHandler) ListAssets(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	assetType := c.Query("type")
	search := c.Query("search")
	favoriteFilter := c.Query("favorite") == "true"

	favIDs, err := h.repo.GetFavoriteIDs()
	if err != nil {
		log.Printf("[Asset] 获取收藏列表失败: %v", err)
		favIDs = make(map[int64]bool)
	}

	if favoriteFilter && len(favIDs) == 0 {
		c.JSON(http.StatusOK, model.AssetListResponse{
			Items: []model.AssetItem{},
			Total: 0,
			Page:  page,
		})
		return
	}

	records, total, err := h.repo.GetRecordsPaginated(page, perPage, assetType, search, favIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询资产失败: " + err.Error()})
		return
	}

	videoModes := map[string]bool{
		"text2video":        true,
		"image2video":       true,
		"multi_image_video": true,
	}
	items := make([]model.AssetItem, 0, len(records))
	for _, rec := range records {
		// When favorite filter is active, skip non-favorited items
		if favoriteFilter && !favIDs[rec.ID] {
			continue
		}

		assetType := "image"
		if videoModes[rec.Mode] {
			assetType = "video"
		}

		thumbnail := ""
		if len(rec.Images) > 0 {
			thumbnail = rec.Images[0]
		}

		items = append(items, model.AssetItem{
			ID:        rec.ID,
			Mode:      rec.Mode,
			Prompt:    rec.Prompt,
			Files:     rec.Images,
			Thumbnail: thumbnail,
			Type:      assetType,
			Time:      rec.Time,
			Favorite:  favIDs[rec.ID],
		})
	}

	// When favorite filter is active, total reflects only favorited items
	if favoriteFilter {
		total = len(items)
	}

	c.JSON(http.StatusOK, model.AssetListResponse{
		Items: items,
		Total: total,
		Page:  page,
	})
}

func (h *AssetHandler) ToggleFavorite(c *gin.Context) {
	var req model.AssetFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	if err := h.repo.ToggleFavorite(req.HistoryID, req.Favorite); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新收藏失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AssetHandler) BatchDownload(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	records, err := h.repo.GetRecordsByIDs(req.IDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询记录失败: " + err.Error()})
		return
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for _, rec := range records {
		for i, img := range rec.Images {
			if img == "" {
				continue
			}
			if strings.HasPrefix(img, "http://") || strings.HasPrefix(img, "https://") {
				continue
			}

			data, err := os.ReadFile(img)
			if err != nil {
				fallback := filepath.Join("outputs", filepath.Base(img))
				data, err = os.ReadFile(fallback)
				if err != nil {
					continue
				}
			}

			ext := filepath.Ext(img)
			entryName := fmt.Sprintf("%s_%d_%d%s", rec.Mode, rec.ID, i, ext)
			f, err := zw.Create(entryName)
			if err != nil {
				continue
			}
			if _, err := io.Copy(f, bytes.NewReader(data)); err != nil {
				continue
			}
		}
	}

	if err := zw.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建压缩文件失败: " + err.Error()})
		return
	}

	c.Header("Content-Disposition", "attachment; filename=assets.zip")
	c.Data(http.StatusOK, "application/zip", buf.Bytes())
}

func (h *AssetHandler) DeleteAssets(c *gin.Context) {
	var req model.AssetDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	if req.DeleteFiles {
		records, err := h.repo.GetRecordsByIDs(req.IDs)
		if err == nil {
			for _, rec := range records {
				deleteRecordFiles(rec.Images)
			}
		}
	}

	if err := h.repo.DeleteRecords(req.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除记录失败: " + err.Error()})
		return
	}

	for _, id := range req.IDs {
		_ = h.repo.ToggleFavorite(id, false)
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
