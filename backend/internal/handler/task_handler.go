package handler

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/agnes-image-tool/backend/internal/service"
)

// TaskHandler 统一异步任务状态查询和 SSE 推送
type TaskHandler struct {
	task *service.TaskQueue
}

func NewTaskHandler(task *service.TaskQueue) *TaskHandler {
	return &TaskHandler{task: task}
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
