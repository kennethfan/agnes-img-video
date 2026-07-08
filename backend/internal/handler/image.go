package handler

import (
	"encoding/base64"
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

	// 直接使用 API 返回的 URL，不自动下载到本地
	saveHistoryRecord(req.Prompt, urls, "text2image", nil)

	c.JSON(http.StatusOK, model.ImageResponse{Images: urls})
}

// ImageToImage 图生图
// POST /api/v1/images/image-to-image
// Content-Type application/json: {"image_url": "https://...", "prompt": "...", "size": "...", "strength": 0.75}
// Content-Type multipart/form-data: image file + prompt + size + strength
func (h *ImageHandler) ImageToImage(c *gin.Context) {
	var imageValue string
	var prompt string
	size := "1024x1024"
	strength := 0.75
	negativePrompt := ""
	n := 1

	if c.Request.Header.Get("Content-Type") == "application/json" {
		var req struct {
			ImageURL       string  `json:"image_url"`
			Prompt         string  `json:"prompt" binding:"required"`
			Size           string  `json:"size"`
			Strength       float64 `json:"strength"`
			NegativePrompt string  `json:"negative_prompt"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
			return
		}
		if req.ImageURL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "image_url 不能为空"})
			return
		}
		imageValue = req.ImageURL
		prompt = req.Prompt
		if req.Size != "" {
			size = req.Size
		}
		if req.Strength > 0 {
			strength = req.Strength
		}
		negativePrompt = req.NegativePrompt
	} else {
		// multipart/form-data：上传图片文件
		file, err := c.FormFile("image")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请上传图片文件"})
			return
		}

		tmpDir := "tmp"
		os.MkdirAll(tmpDir, 0755)
		tmpPath := filepath.Join(tmpDir, fmt.Sprintf("upload_%d_%s", time.Now().UnixNano(), file.Filename))
		if err := c.SaveUploadedFile(file, tmpPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存上传文件失败: " + err.Error()})
			return
		}
		defer os.Remove(tmpPath)

		// 读取图片并编码为 base64 Data URI
		imageData, err := os.ReadFile(tmpPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取图片失败: " + err.Error()})
			return
		}
		b64 := base64.StdEncoding.EncodeToString(imageData)
		ext := strings.ToLower(filepath.Ext(file.Filename))
		mimeType := map[string]string{
			".png": "image/png", ".jpg": "image/jpeg",
			".jpeg": "image/jpeg", ".gif": "image/gif",
			".webp": "image/webp",
		}[ext]
		if mimeType == "" {
			mimeType = "image/png"
		}
		imageValue = fmt.Sprintf("data:%s;base64,%s", mimeType, b64)

		prompt = c.PostForm("prompt")
		if s := c.PostForm("size"); s != "" {
			size = s
		}
		if s := c.PostForm("strength"); s != "" {
			strength = parseFloat(s)
		}
		negativePrompt = c.PostForm("negative_prompt")
	}

	urls, err := h.svc.ImageToImage(imageValue, prompt, size, n, strength, negativePrompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	saveHistoryRecord(prompt, urls, "image2image", map[string]any{
		"size":     size,
		"strength": strength,
	})

	c.JSON(http.StatusOK, model.ImageResponse{Images: urls})
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
			allImages = append(allImages, urls...)
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
