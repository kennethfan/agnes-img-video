package handler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/agnes-image-tool/backend/internal/model"
	"github.com/agnes-image-tool/backend/internal/service"
)

type VideoHandler struct {
	svc  *service.AgnesClient
	mgr  *service.TaskManager
}

func NewVideoHandler(svc *service.AgnesClient, mgr *service.TaskManager) *VideoHandler {
	return &VideoHandler{svc: svc, mgr: mgr}
}

// TextToVideo 文生视频
func (h *VideoHandler) TextToVideo(c *gin.Context) {
	var req model.VideoCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	opts := h.buildVideoOptions(req)
	opts.RecordType = "text2video"

	// 提交任务
	videoID, err := h.svc.SubmitVideoTask(h.svc.BuildVideoPayload(req.Prompt, opts))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交视频任务失败: " + err.Error()})
		return
	}

	// 创建任务管理器中的跟踪
	h.mgr.CreateTask(videoID, req.Prompt, opts)

	// 立即写入历史记录（待更新）
	saveHistoryRecord(req.Prompt, []string{}, "text2video", map[string]any{"taskId": videoID})

	log.Printf("[Video] 文生视频任务已创建: %s", videoID)
	c.JSON(http.StatusOK, model.VideoTaskResponse{TaskID: videoID})
}

// ImageToVideo 图生视频
// POST /api/v1/videos/image-to-video
// Content-Type application/json: {"image_url": "https://...", "prompt": "...", ...}
//   also supports {"image_urls": ["https://..."], ...} for compatibility
// Content-Type multipart/form-data: image file + prompt + ...
func (h *VideoHandler) ImageToVideo(c *gin.Context) {
	var req model.VideoCreateRequest
	// 尝试 JSON 绑定或表单
	if c.Request.Header.Get("Content-Type") == "application/json" {
		var jsonReq struct {
			model.VideoCreateRequest
			ImageURL string `json:"image_url"`
		}
		if err := c.ShouldBindJSON(&jsonReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
			return
		}
		req = jsonReq.VideoCreateRequest
		if jsonReq.ImageURL != "" && len(req.ImageURLs) == 0 {
			req.ImageURLs = []string{jsonReq.ImageURL}
		}
	} else {
		req.Prompt = c.PostForm("prompt")
		if req.Prompt == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "prompt 不能为空"})
			return
		}
	}

	opts := h.buildVideoOptions(req)
	opts.RecordType = "image2video"

	var videoID string
	var err error

	if len(req.ImageURLs) > 0 {
		// 直接使用提供的 URL
		opts.ImageURLs = req.ImageURLs
		payload := h.svc.BuildVideoPayload(req.Prompt, opts)
		payload["image"] = req.ImageURLs[0]
		videoID, err = h.svc.SubmitVideoTask(payload)
	} else {
		// multipart 上传图片
		file, err := c.FormFile("image")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请上传图片文件或提供图片 URL"})
			return
		}

		tmpDir := "tmp"
		os.MkdirAll(tmpDir, 0755)
		tmpPath := filepath.Join(tmpDir, fmt.Sprintf("video_upload_%d_%s", time.Now().UnixNano(), file.Filename))
		if err := c.SaveUploadedFile(file, tmpPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存上传文件失败: " + err.Error()})
			return
		}
		defer os.Remove(tmpPath)

		// 转换为 base64 URL
		imageData, err := os.ReadFile(tmpPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取图片失败: " + err.Error()})
			return
		}

		ext := strings.ToLower(filepath.Ext(file.Filename))
		mimeType := map[string]string{
			".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
			".gif": "image/gif", ".webp": "image/webp",
		}[ext]
		if mimeType == "" {
			mimeType = "image/png"
		}

		b64 := fmt.Sprintf("data:%s;base64,%s", mimeType, imageData)
		payload := h.svc.BuildVideoPayload(req.Prompt, opts)
		payload["image"] = b64
		videoID, err = h.svc.SubmitVideoTask(payload)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交视频任务失败: " + err.Error()})
		return
	}

	h.mgr.CreateTask(videoID, req.Prompt, opts)
	saveHistoryRecord(req.Prompt, []string{}, "image2video", map[string]any{"taskId": videoID})
	log.Printf("[Video] 图生视频任务已创建: %s", videoID)
	c.JSON(http.StatusOK, model.VideoTaskResponse{TaskID: videoID})
}

// MultiImageVideo 多图视频 / 关键帧
func (h *VideoHandler) MultiImageVideo(c *gin.Context) {
	var req model.VideoCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	if len(req.ImageURLs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "至少需要一张图片 URL"})
		return
	}

	opts := h.buildVideoOptions(req)
	opts.RecordType = "multi_image_video"
	opts.ImageURLs = req.ImageURLs
	opts.Mode = req.Mode

	payload := h.svc.BuildVideoPayload(req.Prompt, opts)
	extraBody := map[string]any{
		"image": req.ImageURLs,
	}
	if req.Mode == "keyframes" {
		extraBody["mode"] = "keyframes"
	}
	payload["extra_body"] = extraBody

	videoID, err := h.svc.SubmitVideoTask(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交视频任务失败: " + err.Error()})
		return
	}

	h.mgr.CreateTask(videoID, req.Prompt, opts)
	saveHistoryRecord(req.Prompt, []string{}, "multi_image_video", map[string]any{"taskId": videoID})
	log.Printf("[Video] 多图视频任务已创建: %s", videoID)
	c.JSON(http.StatusOK, model.VideoTaskResponse{TaskID: videoID})
}

// GenerateScript 生成视频脚本
func (h *VideoHandler) GenerateScript(c *gin.Context) {
	var req model.ScriptGenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	if req.Topic == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "主题不能为空"})
		return
	}
	if req.Duration <= 0 {
		req.Duration = 30
	}

	script, err := h.svc.GenerateScript(req.Topic, req.Duration, req.Style, req.Language)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成脚本失败: " + err.Error()})
		return
	}

	// 保存到历史记录
	saveHistoryRecord(req.Topic, []string{}, "script_gen", map[string]any{
		"script":   script,
		"duration": req.Duration,
		"style":    req.Style,
		"language": req.Language,
	})

	c.JSON(http.StatusOK, model.ScriptGenResponse{Script: script})
}

// GetTaskStatus 查询任务状态
func (h *VideoHandler) GetTaskStatus(c *gin.Context) {
	taskID := c.Param("taskId")
	task := h.mgr.GetTask(taskID)
	if task == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	c.JSON(http.StatusOK, model.VideoStatus{
		Status:   task.Status,
		Progress: task.Progress,
		URL:      task.ResultURL,
		Error:    task.Error,
		Seconds:  task.Seconds,
	})
}

// StreamSSE SSE 实时推送视频进度
func (h *VideoHandler) StreamSSE(c *gin.Context) {
	taskID := c.Param("taskId")
	task := h.mgr.GetTask(taskID)
	if task == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	subID := fmt.Sprintf("sse_%d", time.Now().UnixNano())
	ch := h.mgr.Subscribe(taskID, subID)
	if ch == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "订阅失败"})
		return
	}
	defer h.mgr.Unsubscribe(taskID, subID)

	// 设置 SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-ch:
			if !ok {
				return false
			}

			switch event.Event {
			case "progress":
				c.SSEvent("progress", map[string]any{
					"progress": event.Progress,
					"status":   event.Status,
				})
			case "complete":
				c.SSEvent("complete", map[string]any{
					"url":     event.URL,
					"seconds": event.Seconds,
				})
			case "error":
				c.SSEvent("error", map[string]any{
					"error": event.Error,
				})
			}
			return true

		case <-c.Request.Context().Done():
			return false
		}
	})
}

// SetupVideoHistoryCallback 设置视频完成时自动保存历史记录
func SetupVideoHistoryCallback(tm *service.TaskManager, svc *service.AgnesClient) {
	tm.SetOnComplete(func(taskID, prompt, resultURL string, opts service.VideoOptions) {
		if resultURL == "" {
			return
		}
			// 直接使用 API 返回的 URL，不自动下载到本地
			paths := []string{resultURL}
		// 更新已有的待完成记录（通过 extra.taskId 匹配）
		if historyRepo != nil {
			if id, err := historyRepo.FindByTaskId(taskID); err == nil && id > 0 {
				updateHistoryImages(id, paths)
				log.Printf("[History] 视频任务 %s 历史已更新", taskID)
				return
			}
		}
		// 未找到待更新记录时，创建新记录
		recordType := opts.RecordType
		if recordType == "" {
			recordType = "video"
		}
		saveHistoryRecord(prompt, paths, recordType, nil)
		log.Printf("[History] 视频任务 %s 历史已保存", taskID)
	})
}

// buildVideoOptions 从请求构建 VideoOptions
func (h *VideoHandler) buildVideoOptions(req model.VideoCreateRequest) service.VideoOptions {
	opts := service.VideoOptions{
		Duration:        req.Duration,
		AspectRatio:     req.AspectRatio,
		FrameRate:       req.FrameRate,
		NegativePrompt:  req.NegativePrompt,
		Seed:            req.Seed,
		NumInferenceSteps: req.NumInferenceSteps,
		Width:           req.Width,
		Height:          req.Height,
		NumFrames:       req.NumFrames,
	}

	if opts.Duration <= 0 {
		opts.Duration = 5
	}
	if opts.AspectRatio == "" {
		opts.AspectRatio = "16:9"
	}
	if opts.FrameRate <= 0 {
		opts.FrameRate = 24
	}

	return opts
}
