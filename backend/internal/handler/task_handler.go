package handler

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/agnes-image-tool/backend/internal/model"
	"github.com/agnes-image-tool/backend/internal/service"
)

// TaskHandler 统一异步任务状态查询和 SSE 推送
type TaskHandler struct {
	task *service.TaskQueue
}

func NewTaskHandler(task *service.TaskQueue) *TaskHandler {
	return &TaskHandler{task: task}
}

// ListTasks 查询任务列表
// GET /api/v1/tasks
func (h *TaskHandler) ListTasks(c *gin.Context) {
	taskType := c.Query("type")
	status := c.Query("status")
	limit := 50
	offset := 0

	records, err := h.task.ListTasks(taskType, status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询任务列表失败: " + err.Error()})
		return
	}
	if records == nil {
		records = []*model.TaskRecord{}
	}
	c.JSON(http.StatusOK, gin.H{"records": records})
}

// GetTask 查询任务状态
// GET /api/v1/tasks/:id
func (h *TaskHandler) GetTask(c *gin.Context) {
	id := c.Param("id")
	rec, err := h.task.GetTask(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询任务失败: " + err.Error()})
		return
	}
	if rec == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	c.JSON(http.StatusOK, rec)
}

// StreamSSE 统一 SSE 进度推送
// GET /api/v1/tasks/:id/stream
func (h *TaskHandler) StreamSSE(c *gin.Context) {
	id := c.Param("id")

	rec, err := h.task.GetTask(id)
	if err != nil || rec == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	subID := fmt.Sprintf("sse_%d", time.Now().UnixNano())
	ch := h.task.Subscribe(id, subID)
	if ch == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "订阅失败"})
		return
	}
	defer h.task.Unsubscribe(id, subID)

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
					"result": event.Result,
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

// CancelTask 取消 pending 任务
// POST /api/v1/tasks/:id/cancel
func (h *TaskHandler) CancelTask(c *gin.Context) {
	id := c.Param("id")
	if err := h.task.CancelTask(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "任务已取消"})
}

// RetryTask 手动重试失败的任务
// POST /api/v1/tasks/:id/retry
func (h *TaskHandler) RetryTask(c *gin.Context) {
	id := c.Param("id")
	if err := h.task.RetryTask(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "任务已重新提交"})
}
