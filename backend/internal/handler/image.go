package handler

import (
	"encoding/base64"
	"encoding/json"
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
	svc  *service.AgnesClient
	task *service.TaskQueue
}

func NewImageHandler(svc *service.AgnesClient, task *service.TaskQueue) *ImageHandler {
	return &ImageHandler{svc: svc, task: task}
}

// TextToImage 文生图（异步）
// POST /api/v1/images/text-to-image
func (h *ImageHandler) TextToImage(c *gin.Context) {
	var req model.TextToImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	params, _ := json.Marshal(map[string]any{
		"prompt":          req.Prompt,
		"size":            req.Size,
		"n":               req.N,
		"negative_prompt": req.NegativePrompt,
	})
	taskID, err := h.task.SubmitTask(string(model.TaskTypeTextToImage), string(params))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交任务失败: " + err.Error()})
		return
	}

	saveHistoryRecord(req.Prompt, []string{}, "text2image", map[string]any{"taskId": taskID}, req.ProjectID)
	c.JSON(http.StatusAccepted, model.TaskCreateResponse{TaskID: taskID})
}

// ImageToImage 图生图（异步）
// POST /api/v1/images/image-to-image
// Content-Type application/json: {"image_url": "https://...", "prompt": "...", "size": "...", "strength": 0.75}
// Content-Type multipart/form-data: image file + prompt + size + strength
func (h *ImageHandler) ImageToImage(c *gin.Context) {
	var imageValue string
	var prompt string
	size := "1024x1024"
	strength := 0.75
	negativePrompt := ""

	var projectID int64

	if c.Request.Header.Get("Content-Type") == "application/json" {
		var req struct {
			ImageURL       string  `json:"image_url"`
			Prompt         string  `json:"prompt" binding:"required"`
			Size           string  `json:"size"`
			Strength       float64 `json:"strength"`
			NegativePrompt string  `json:"negative_prompt"`
			ProjectID      int64   `json:"project_id"`
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
		projectID = req.ProjectID
	} else {
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
		projectID = parseInt64(c.PostForm("project_id"))
	}

	params, _ := json.Marshal(map[string]any{
		"prompt":          prompt,
		"size":            size,
		"image_value":     imageValue,
		"strength":        strength,
		"negative_prompt": negativePrompt,
	})
	taskID, err := h.task.SubmitTask(string(model.TaskTypeImageToImage), string(params))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交任务失败: " + err.Error()})
		return
	}

	saveHistoryRecord(prompt, []string{}, "image2image", map[string]any{
		"taskId":   taskID,
		"size":     size,
		"strength": strength,
	}, projectID)

	c.JSON(http.StatusAccepted, model.TaskCreateResponse{TaskID: taskID})
}

// BatchGenerate 批量文生图（异步）
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

	params, _ := json.Marshal(map[string]any{
		"prompts": req.Prompts,
		"size":    req.Size,
	})
	taskID, err := h.task.SubmitTask(string(model.TaskTypeBatch), string(params))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交任务失败: " + err.Error()})
		return
	}

	saveHistoryRecord(strings.Join(req.Prompts, "; "), []string{}, "batch", map[string]any{
		"taskId": taskID,
		"size":   req.Size,
	}, req.ProjectID)

	c.JSON(http.StatusAccepted, model.TaskCreateResponse{TaskID: taskID})
}

// ==================== 辅助函数 ====================

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	if f <= 0 {
		return 0.75
	}
	return f
}

func parseInt64(s string) int64 {
	var n int64
	fmt.Sscanf(s, "%d", &n)
	return n
}
