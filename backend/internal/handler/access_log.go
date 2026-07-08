package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/agnes-image-tool/backend/internal/repository"
)

type AccessLogHandler struct {
	repo *repository.AccessLogRepo
}

func NewAccessLogHandler(repo *repository.AccessLogRepo) *AccessLogHandler {
	return &AccessLogHandler{repo: repo}
}

// ListLogs 查询访问日志
// GET /api/v1/access-logs
func (h *AccessLogHandler) ListLogs(c *gin.Context) {
	q := repository.AccessLogQuery{
		Method:    c.Query("method"),
		Path:      c.Query("path"),
		Sort:      c.DefaultQuery("sort", "desc"),
		From:      c.Query("from"),
		To:        c.Query("to"),
	}

	if p, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil {
		q.Page = p
	} else {
		q.Page = 1
	}

	if ps, err := strconv.Atoi(c.DefaultQuery("page_size", "50")); err == nil {
		q.PageSize = ps
	} else {
		q.PageSize = 50
	}

	if sm, err := strconv.Atoi(c.DefaultQuery("status_min", "0")); err == nil {
		q.StatusMin = sm
	}

	if sm, err := strconv.Atoi(c.DefaultQuery("status_max", "0")); err == nil {
		q.StatusMax = sm
	}

	result, err := h.repo.Query(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询日志失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// DeleteLog 删除单条日志
// DELETE /api/v1/access-logs/:id
func (h *AccessLogHandler) DeleteLog(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的日志 ID"})
		return
	}

	if err := h.repo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除日志失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ClearLogs 清空全部日志
// DELETE /api/v1/access-logs
func (h *AccessLogHandler) ClearLogs(c *gin.Context) {
	if err := h.repo.ClearAll(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "清空日志失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
