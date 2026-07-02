package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/agnes-image-tool/backend/internal/model"
	"github.com/agnes-image-tool/backend/internal/repository"
)

var historyRepo *repository.HistoryRepo

func SetHistoryRepo(repo *repository.HistoryRepo) {
	historyRepo = repo
}

type HistoryHandler struct {
	repo *repository.HistoryRepo
}

func NewHistoryHandler(repo *repository.HistoryRepo) *HistoryHandler {
	return &HistoryHandler{repo: repo}
}

func (h *HistoryHandler) GetHistory(c *gin.Context) {
	records, err := h.repo.GetRecords(100)
	if err != nil {
		log.Printf("[History] 读取失败: %v", err)
		records = []model.HistoryRecord{}
	}
	c.JSON(http.StatusOK, gin.H{"records": records})
}

func (h *HistoryHandler) ClearHistory(c *gin.Context) {
	if err := h.repo.ClearRecords(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "清空历史记录失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func saveHistoryRecord(prompt string, imagePaths []string, mode string, extra any) {
	if historyRepo == nil {
		log.Printf("[History] repository 未初始化，跳过历史记录")
		return
	}
	if err := historyRepo.InsertRecord(prompt, imagePaths, mode, extra); err != nil {
		log.Printf("[History] 保存失败: %v", err)
		return
	}
	if err := historyRepo.TrimRecords(100); err != nil {
		log.Printf("[History] 清理旧记录失败: %v", err)
	}
}
