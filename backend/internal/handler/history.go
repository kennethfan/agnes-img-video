package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/agnes-image-tool/backend/internal/model"
)

var (
	historyMu     sync.RWMutex
	historyMaxLen = 100
	historyPath   = "../history.json" // 相对于 backend/ 目录
)

// SetHistoryPath 设置历史记录文件路径（由 main.go 调用）
func SetHistoryPath(path string) {
	historyMu.Lock()
	defer historyMu.Unlock()
	historyPath = path
}

// GetHistoryPath 获取当前历史记录文件路径
func GetHistoryPath() string {
	historyMu.RLock()
	defer historyMu.RUnlock()
	return historyPath
}

// HistoryHandler 历史记录相关 handler
type HistoryHandler struct{}

func NewHistoryHandler() *HistoryHandler {
	return &HistoryHandler{}
}

// GetHistory 获取历史记录列表
// GET /api/v1/history
func (h *HistoryHandler) GetHistory(c *gin.Context) {
	records := loadHistory()
	if records == nil {
		records = []model.HistoryRecord{}
	}
	c.JSON(http.StatusOK, gin.H{"records": records})
}

// ClearHistory 清空历史记录
// DELETE /api/v1/history
func (h *HistoryHandler) ClearHistory(c *gin.Context) {
	historyMu.Lock()
	defer historyMu.Unlock()

	if err := os.WriteFile(historyPath, []byte("[]"), 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "清空历史记录失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ==================== 文件 I/O ====================

func loadHistory() []model.HistoryRecord {
	historyMu.RLock()
	path := historyPath
	historyMu.RUnlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var records []model.HistoryRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil
	}
	return records
}

func saveHistory(records []model.HistoryRecord) {
	historyMu.Lock()
	path := historyPath
	historyMu.Unlock()

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(path, data, 0644)
}

// saveHistoryRecord 插入一条新历史记录（供其他 handler 调用）
func saveHistoryRecord(prompt string, imagePaths []string, mode string, extra any) {
	records := loadHistory()
	if records == nil {
		records = make([]model.HistoryRecord, 0, historyMaxLen)
	}

	record := model.HistoryRecord{
		Time:   time.Now().Format("2006-01-02 15:04:05"),
		Mode:   mode,
		Prompt: prompt,
		Images: imagePaths,
	}
	if extra != nil {
		record.Extra = extra
	}

	records = append([]model.HistoryRecord{record}, records...)
	if len(records) > historyMaxLen {
		records = records[:historyMaxLen]
	}

	saveHistory(records)
}
