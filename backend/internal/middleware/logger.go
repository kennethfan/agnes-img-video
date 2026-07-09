package middleware

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/agnes-image-tool/backend/internal/repository"
)

var accessLogRepo repository.AccessLogRepository

func SetAccessLogRepo(repo repository.AccessLogRepository) {
	accessLogRepo = repo
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyLogWriter) Write(data []byte) (int, error) {
	// 只缓存 JSON 响应，限制 100KB 防止内存溢出
	if strings.Contains(w.Header().Get("Content-Type"), "application/json") && w.body.Len() < 100*1024 {
		w.body.Write(data)
	}
	return w.ResponseWriter.Write(data)
}

// AccessLogger 返回记录接口调用日志的中间件
func AccessLogger() gin.HandlerFunc {
	if accessLogRepo == nil {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		start := time.Now()

		// === 捕获请求体 ===
		reqBody := captureRequestBody(c)

		// === 捕获响应体 ===
		bw := &bodyLogWriter{
			body:           &bytes.Buffer{},
			ResponseWriter: c.Writer,
		}
		c.Writer = bw

		// === 处理请求 ===
		c.Next()

		// === 收集日志 ===
		durationMs := int(time.Since(start).Milliseconds())
		respBody := captureResponseBody(c, bw)

		if strings.Contains(c.Request.URL.Path, "/access-logs") || strings.Contains(c.Request.URL.Path, "/db/") {
			return
		}

		// === 异步写入数据库 ===
		record := &repository.AccessLogRecord{
			Timestamp:    start.Format(time.RFC3339),
			Method:       c.Request.Method,
			Path:         c.Request.URL.String(),
			Status:       c.Writer.Status(),
			DurationMs:   durationMs,
			ClientIP:     c.ClientIP(),
			UserAgent:    c.Request.UserAgent(),
			RequestBody:  reqBody,
			ResponseBody: respBody,
		}
		if len(c.Errors) > 0 {
			record.Error = c.Errors.ByType(gin.ErrorTypeAny).String()
		}

		go func() {
			if err := accessLogRepo.Insert(record); err != nil {
				log.Printf("[AccessLog] 写入日志失败: %v", err)
			}
		}()
	}
}

func captureRequestBody(c *gin.Context) string {
	contentType := c.Request.Header.Get("Content-Type")

	// multipart: 只记录字段摘要，跳过二进制
	if strings.Contains(contentType, "multipart/form-data") {
		return captureMultipartSummary(c)
	}

	// 其他: 读取 body，最多 100KB
	if c.Request.Body == nil {
		return ""
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 100*1024))
	if err != nil {
		return ""
	}
	// 恢复 body 供后续 handler 使用
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	// 限制显示长度
	s := string(body)
	if len(s) > 2000 {
		s = s[:2000] + "... [truncated]"
	}
	return s
}

func captureMultipartSummary(c *gin.Context) string {
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		return fmt.Sprintf("[multipart parse error: %v]", err)
	}
	if c.Request.MultipartForm == nil {
		return ""
	}

	var parts []string
	for key := range c.Request.MultipartForm.Value {
		vals := c.Request.MultipartForm.Value[key]
		if len(vals) == 1 {
			v := vals[0]
			if len(v) > 200 {
				v = v[:200] + "..."
			}
			parts = append(parts, fmt.Sprintf("%s=%s", key, v))
		} else {
			parts = append(parts, fmt.Sprintf("%s=[%d values]", key, len(vals)))
		}
	}
	for key, files := range c.Request.MultipartForm.File {
		var fileDescs []string
		for _, fh := range files {
			fileDescs = append(fileDescs, fmt.Sprintf("%s(%d bytes)", fh.Filename, fh.Size))
		}
		parts = append(parts, fmt.Sprintf("%s=[%s]", key, strings.Join(fileDescs, ", ")))
	}

	// 重新构造 body（Gin 消费后不会恢复）
	c.Request.Body = io.NopCloser(bytes.NewBuffer(nil))

	_ = c.Request.ParseMultipartForm(32 << 20) // 这个是多余的，但保留兼容
	// 实际上对于 multipart，我们无法完美恢复 body，但后续 handler 会自己再 ParseMultipartForm
	// Gin 的 FormFile 会调用 ParseMultipartForm，所以二次解析没问题

	summary := strings.Join(parts, "; ")
	if len(summary) > 2000 {
		summary = summary[:2000] + "... [truncated]"
	}
	return "[multipart] " + summary
}

func captureResponseBody(c *gin.Context, bw *bodyLogWriter) string {
	contentType := c.Writer.Header().Get("Content-Type")

	if strings.Contains(contentType, "text/event-stream") {
		return "[SSE stream]"
	}
	if strings.Contains(contentType, "image/") || strings.Contains(contentType, "video/") || strings.Contains(contentType, "application/octet-stream") {
		return fmt.Sprintf("[binary: %d bytes]", bw.body.Len())
	}
	if strings.Contains(contentType, "application/json") {
		s := bw.body.String()
		if len(s) > 2000 {
			s = s[:2000] + "... [truncated]"
		}
		return s
	}

	s := bw.body.String()
	if len(s) > 2000 {
		s = s[:2000] + "... [truncated]"
	}
	return s
}

// Make lint happy: ensure multipart.FileHeader is referenced
var _ = (*multipart.FileHeader)(nil)
