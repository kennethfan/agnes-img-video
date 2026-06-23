package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/agnes-image-tool/backend/internal/model"
	"github.com/agnes-image-tool/backend/internal/service"
)

type ImageHandler struct {
	svc *service.AgnesClient
}

func NewImageHandler(svc *service.AgnesClient) *ImageHandler {
	return &ImageHandler{svc: svc}
}

// TextToImage 文生图
// POST /api/v1/images/text-to-image
func (h *ImageHandler) TextToImage(c *gin.Context) {
	var req model.TextToImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	urls, err := h.svc.TextToImage(req.Prompt, req.Size, req.N, req.NegativePrompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 下载所有图片到本地
	localPaths := make([]string, 0, len(urls))
	for i, url := range urls {
		prefix := fmt.Sprintf("text2img_%d", i)
		localPath, err := h.svc.DownloadAndSave(url, prefix)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("下载图片 %d 失败: %v", i, err)})
			return
		}
		localPaths = append(localPaths, toRelPath(localPath))
	}

	// 保存历史
	saveHistoryRecord(req.Prompt, localPaths, "text2image", nil)

	c.JSON(http.StatusOK, model.ImageResponse{Images: localPaths})
}

// ImageToImage 图生图
// POST /api/v1/images/image-to-image
func (h *ImageHandler) ImageToImage(c *gin.Context) {
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传图片文件"})
		return
	}

	// 保存上传文件到临时目录
	tmpDir := "tmp"
	os.MkdirAll(tmpDir, 0755)
	tmpPath := filepath.Join(tmpDir, fmt.Sprintf("upload_%d_%s", time.Now().UnixNano(), file.Filename))
	if err := c.SaveUploadedFile(file, tmpPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存上传文件失败: " + err.Error()})
		return
	}
	defer os.Remove(tmpPath)

	prompt := c.PostForm("prompt")
	size := c.DefaultPostForm("size", "1024x1024")
	strength := c.DefaultPostForm("strength", "0.75")
	negativePrompt := c.PostForm("negative_prompt")

	n := 1 // 图生图默认 1 张

	urls, err := h.svc.ImageToImage(tmpPath, prompt, size, n, parseFloat(strength), negativePrompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	localPaths := make([]string, 0, len(urls))
	for i, url := range urls {
		prefix := fmt.Sprintf("img2img_%d", i)
		localPath, err := h.svc.DownloadAndSave(url, prefix)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("下载图片 %d 失败: %v", i, err)})
			return
		}
		localPaths = append(localPaths, toRelPath(localPath))
	}

	saveHistoryRecord(prompt, localPaths, "image2image", map[string]any{
		"size":     size,
		"strength": strength,
	})

	c.JSON(http.StatusOK, model.ImageResponse{Images: localPaths})
}

// BatchGenerate 批量文生图
// POST /api/v1/images/batch
func (h *ImageHandler) BatchGenerate(c *gin.Context) {
	var req model.BatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	if len(req.Prompts) > 20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "批量生成最多支持 20 个提示词"})
		return
	}

	allImages := make([]string, 0)
	for i, prompt := range req.Prompts {
		urls, err := h.svc.TextToImage(prompt, req.Size, 1, "")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("第 %d 个提示词失败: %v", i+1, err)})
			return
		}
		for _, url := range urls {
			localPath, err := h.svc.DownloadAndSave(url, fmt.Sprintf("batch_%d", i))
			if err != nil {
				continue
			}
			allImages = append(allImages, toRelPath(localPath))
		}
	}

	saveHistoryRecord(strings.Join(req.Prompts, "; "), allImages, "batch", map[string]any{
		"size": req.Size,
	})

	c.JSON(http.StatusOK, model.ImageResponse{Images: allImages})
}

// ==================== 辅助函数 ====================

func toRelPath(absPath string) string {
	// 将绝对路径转为相对路径（相对于项目根）
	// 如果路径包含 "outputs/"，返回 "outputs/xxx"
	if idx := strings.Index(absPath, "outputs/"); idx >= 0 {
		return absPath[idx:]
	}
	// 如果已经在 outputs/ 下
	if strings.HasPrefix(absPath, "outputs/") {
		return absPath
	}
	return absPath
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	if f <= 0 {
		return 0.75
	}
	return f
}
