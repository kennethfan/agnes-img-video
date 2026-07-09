package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DBHandler 数据库备份恢复（JSON 格式）
type DBHandler struct {
	gormDB *gorm.DB
}

func NewDBHandler(gormDB *gorm.DB) *DBHandler {
	return &DBHandler{gormDB: gormDB}
}

// ExportPayload JSON 备份负载
type ExportPayload struct {
	Version int                       `json:"version"`
	Tables  map[string][]map[string]any `json:"tables"`
}

// exportTableOrder 导出顺序（按外键依赖排序）
var exportTableOrder = []string{
	"settings",
	"history",
	"favorites",
	"storyboard_projects",
	"storyboard_shots",
	"access_logs",
	"task_queue",
}

func (h *DBHandler) exportJSON() ([]byte, error) {
	payload := ExportPayload{
		Version: 1,
		Tables:  make(map[string][]map[string]any),
	}

	for _, table := range exportTableOrder {
		var rows []map[string]any
		if err := h.gormDB.Table(table).Find(&rows).Error; err != nil {
			return nil, fmt.Errorf("导出表 %s 失败: %w", table, err)
		}
		payload.Tables[table] = rows
	}

	return json.MarshalIndent(payload, "", "  ")
}

// ExportDB 导出数据库备份（JSON）
// GET /api/v1/db/export
func (h *DBHandler) ExportDB(c *gin.Context) {
	data, err := h.exportJSON()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "导出失败: " + err.Error()})
		return
	}
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", `attachment; filename="history.json"`)
	c.Data(http.StatusOK, "application/json", data)
}

func (h *DBHandler) restoreJSON(data []byte) error {
	var payload ExportPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("解析失败: %w", err)
	}
	if payload.Version != 1 {
		return fmt.Errorf("不支持的版本号: %d", payload.Version)
	}

	return h.gormDB.Transaction(func(tx *gorm.DB) error {
		// 反序清空
		for i := len(exportTableOrder) - 1; i >= 0; i-- {
			table := exportTableOrder[i]
			if err := tx.Exec(fmt.Sprintf("DELETE FROM %s", table)).Error; err != nil {
				return fmt.Errorf("清空表 %s 失败: %w", table, err)
			}
		}
		// 正序插入
		for _, table := range exportTableOrder {
			rows, ok := payload.Tables[table]
			if !ok || len(rows) == 0 {
				continue
			}
			for _, row := range rows {
				cols := make([]string, 0, len(row))
				vals := make([]any, 0, len(row))
				for k, v := range row {
					cols = append(cols, k)
					vals = append(vals, v)
				}
				placeholders := strings.Repeat("?, ", len(cols)-1) + "?"
				query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(cols, ", "), placeholders)
				if err := tx.Exec(query, vals...).Error; err != nil {
					return fmt.Errorf("恢复表 %s 失败: %w", table, err)
				}
			}
		}
		return nil
	})
}

// RestoreDB 恢复数据库备份
// POST /api/v1/db/restore — 仅接受 .json 文件
func (h *DBHandler) RestoreDB(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传文件"})
		return
	}

	if !strings.HasSuffix(file.Filename, ".json") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持 .json 文件"})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "打开文件失败: " + err.Error()})
		return
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败: " + err.Error()})
		return
	}
	if err := h.restoreJSON(content); err != nil {
		log.Printf("[DB] 恢复失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "恢复失败: " + err.Error()})
		return
	}
	log.Printf("[DB] 恢复成功 (from: %s)", file.Filename)
	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "恢复成功"})
}
