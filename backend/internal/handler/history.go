package handler

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/agnes-image-tool/backend/internal/model"
	"github.com/agnes-image-tool/backend/internal/repository"
	"github.com/agnes-image-tool/backend/internal/service"
)

var historyRepo *repository.HistoryRepo
var githubStorage *service.GithubStorage

func SetHistoryRepo(repo *repository.HistoryRepo) {
	historyRepo = repo
}

func SetGithubStorage(gs *service.GithubStorage) {
	githubStorage = gs
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
	deleteFiles := c.Query("delete_files") == "true"

	if deleteFiles {
		records, err := h.repo.GetRecords(100)
		if err == nil {
			for _, rec := range records {
				deleteRecordFiles(rec.Images)
			}
		}
	}

	if err := h.repo.ClearRecords(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "清空历史记录失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func deleteRecordFiles(images []string) {
	for _, img := range images {
		if img == "" {
			continue
		}
		if strings.HasPrefix(img, "http://") || strings.HasPrefix(img, "https://") {
			if githubStorage != nil && strings.Contains(img, "raw.githubusercontent.com") {
				if err := githubStorage.DeleteByURL(img); err != nil {
					log.Printf("[History] 删除 GitHub 文件失败: %v", err)
				}
			}
			continue
		}
		if err := os.Remove(img); err != nil && !os.IsNotExist(err) {
			log.Printf("[History] 删除本地文件失败: %s - %v", img, err)
		}
	}
}

func (h *HistoryHandler) DeleteHistory(c *gin.Context) {
	var req model.BatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	if err := h.repo.DeleteRecords(req.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除记录失败: " + err.Error()})
		return
	}

	if req.DeleteFiles {
		// 先查询这些记录的图片路径再删除
		for _, id := range req.IDs {
			// 用单条查询获取图片列表
			records, err := h.repo.GetRecords(100)
			if err != nil {
				continue
			}
			for _, rec := range records {
				if rec.ID == id {
					deleteRecordFiles(rec.Images)
					break
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *HistoryHandler) DeleteRecord(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的记录 ID"})
		return
	}

	deleteFiles := c.Query("delete_files") == "true"

	// 如果需要删除文件，先查出图片路径
	if deleteFiles {
		records, err := h.repo.GetRecords(100)
		if err == nil {
			for _, rec := range records {
				if rec.ID == id {
					deleteRecordFiles(rec.Images)
					break
				}
			}
		}
	}

	if err := h.repo.DeleteRecord(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除记录失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func saveHistoryRecord(prompt string, imagePaths []string, mode string, extra any) int64 {
	if historyRepo == nil {
		log.Printf("[History] repository 未初始化，跳过历史记录")
		return 0
	}
	id, err := historyRepo.InsertRecord(prompt, imagePaths, mode, extra)
	if err != nil {
		log.Printf("[History] 保存失败: %v", err)
		return 0
	}
	if err := historyRepo.TrimRecords(100); err != nil {
		log.Printf("[History] 清理旧记录失败: %v", err)
	}
	return id
}

func updateHistoryImages(id int64, images []string) {
	if historyRepo == nil || id <= 0 {
		return
	}
	if err := historyRepo.UpdateRecordImages(id, images); err != nil {
		log.Printf("[History] 更新图片失败: %v", err)
	}
}
