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
	"time"

	"github.com/gin-gonic/gin"

	"github.com/agnes-image-tool/backend/internal/model"
	"github.com/agnes-image-tool/backend/internal/repository"
)

type AssetHandler struct {
	repo         repository.AssetRepository
	settingsRepo repository.SettingsRepository
}

func NewAssetHandler(repo repository.AssetRepository, settingsRepo repository.SettingsRepository) *AssetHandler {
	return &AssetHandler{repo: repo, settingsRepo: settingsRepo}
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
		log.Printf("[Asset] 保存参数不完整: image_url=%q prompt=%q mode=%q", req.ImageURL, req.Prompt, req.Mode)
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

	// 解析本地路径 — 本地路径直接引用，远程 URL 则异步处理
	var id int64
	var err error
	if !strings.HasPrefix(req.ImageURL, "http://") && !strings.HasPrefix(req.ImageURL, "https://") {
		// 本地路径：尝试 outputs/ 和 backend/outputs/
		var localPath string
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

		asset := &model.Asset{
			Mode:        req.Mode,
			Prompt:      req.Prompt,
			Type:        assetType,
			Time:        time.Now().Format("2006-01-02 15:04:05"),
			Favorite:    false,
			OriginalURL: req.ImageURL,
			LocalPath:   localPath,
		}
		id, err = h.repo.Insert(asset)
	} else {
		// 远程 URL：先插入记录（local_path/github_url 为空），有存储目标再异步下载+上传
		asset := &model.Asset{
			Mode:        req.Mode,
			Prompt:      req.Prompt,
			Type:        assetType,
			Time:        time.Now().Format("2006-01-02 15:04:05"),
			Favorite:    false,
			OriginalURL: req.ImageURL,
		}
		id, err = h.repo.Insert(asset)
		if err == nil {
			// 没配存储目标时只存记录，不触发转存
			if s, e := h.settingsRepo.GetSettings(); e == nil && s.StorageTarget == "" {
				log.Printf("[Asset] 存储目标未配置，跳过转存 id=%d", id)
			} else {
				go h.processAssetStorage(id, req.ImageURL, assetType)
			}
		}
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存到作品库失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id})
}

// storeFile 下载远程文件并根据 storage_target 处理存储
// 返回 (localPath, githubURL, error)
func (h *AssetHandler) storeFile(imageURL string, assetType string) (string, string, error) {
	storageTarget := "local"
	if s, err := h.settingsRepo.GetSettings(); err == nil {
		storageTarget = s.StorageTarget
	}
	if storageTarget == "" {
		return "", "", fmt.Errorf("存储目标未配置，请在设置中至少选择一种存储方式（本地/GitHub）")
	}

	outputDir := "outputs"
	os.MkdirAll(outputDir, 0755)

	ext := filepath.Ext(imageURL)
	if ext == "" {
		ext = ".png"
		if assetType == "video" {
			ext = ".mp4"
		}
	}

	timestamp := time.Now().Format("20060102_150405_000000")
	filename := fmt.Sprintf("asset_%s%s", timestamp, ext)
	filePath := filepath.Join(outputDir, filename)

	resp, err := http.Get(imageURL)
	if err != nil {
		return "", "", fmt.Errorf("下载文件失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("下载文件失败: 上游返回 %s", http.StatusText(resp.StatusCode))
	}

	out, err := os.Create(filePath)
	if err != nil {
		return "", "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", "", fmt.Errorf("写入文件失败: %w", err)
	}

	var localPath string
	var githubURL string

	saveLocal := storageTarget == "local" || storageTarget == "both"
	uploadGithub := (storageTarget == "github" || storageTarget == "both") && githubStorage != nil

	if saveLocal {
		localPath = filePath
	}

	if uploadGithub {
		remotePath := fmt.Sprintf("images/%s", filename)
		uploadedURL, err := githubStorage.UploadFile(filePath, remotePath)
		if err != nil {
			log.Printf("[Asset] 上传到 GitHub 失败: %v", err)
		} else {
			githubURL = uploadedURL
		}
	}

	// 仅 GitHub 模式：上传后删除本地临时文件
	if storageTarget == "github" && githubURL != "" {
		os.Remove(filePath)
	}

	return localPath, githubURL, nil
}

// processAssetStorage 异步处理资产存储（下载+上传）
func (h *AssetHandler) processAssetStorage(id int64, imageURL string, assetType string) {
	localPath, githubURL, err := h.storeFile(imageURL, assetType)
	if err != nil {
		log.Printf("[Asset] 异步处理存储失败 id=%d: %v", id, err)
		return
	}
	if err := h.repo.UpdateStoragePaths(id, localPath, githubURL); err != nil {
		log.Printf("[Asset] 异步更新存储路径失败 id=%d: %v", id, err)
	}
}

// TransferAsset 转存 — 对已入库 asset 补全 local_path / github_url
func (h *AssetHandler) TransferAsset(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: 无效的 ID"})
		return
	}

	assets, err := h.repo.GetByIDs([]int64{id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询作品失败: " + err.Error()})
		return
	}
	if len(assets) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "作品不存在"})
		return
	}
	asset := assets[0]

	// 本地路径（非远程 URL）直接返回
	if !strings.HasPrefix(asset.OriginalURL, "http://") && !strings.HasPrefix(asset.OriginalURL, "https://") {
		c.JSON(http.StatusOK, gin.H{"message": "本地文件无需转存", "asset": asset})
		return
	}

	// 下载远程文件并根据 storage_target 处理
	localPath, githubURL, err := h.storeFile(asset.OriginalURL, asset.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 更新 asset 记录
	if err := h.repo.UpdateStoragePaths(id, localPath, githubURL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新存储路径失败: " + err.Error()})
		return
	}

	// 重新查询返回最新数据
	updated, _ := h.repo.GetByIDs([]int64{id})
	if len(updated) > 0 {
		c.JSON(http.StatusOK, gin.H{"asset": updated[0]})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": "转存完成"})
	}
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
