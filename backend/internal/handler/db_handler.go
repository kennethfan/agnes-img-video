package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ReplaceFunc 数据库替换函数，接收临时文件路径，返回错误信息
type ReplaceFunc func(tmpPath string) error

// DBHandler 数据库导出与恢复（支持 .db / .sql）
type DBHandler struct {
	dbPath      string
	replaceFunc ReplaceFunc
	getDB       func() *sql.DB    // 仅用于 .sql 导出（保持兼容）
	gormDB      *gorm.DB          // 新增：JSON 格式使用
}

func (h *DBHandler) SetGormDB(db *gorm.DB) { h.gormDB = db }

// ExportPayload JSON 导出/恢复负载
type ExportPayload struct {
	Version int                       `json:"version"`
	Driver  string                    `json:"driver"`
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

func NewDBHandler(dbPath string, replaceFunc ReplaceFunc, getDB func() *sql.DB, gormDB *gorm.DB) *DBHandler {
	return &DBHandler{dbPath: dbPath, replaceFunc: replaceFunc, getDB: getDB, gormDB: gormDB}
}

func (h *DBHandler) exportJSON() ([]byte, error) {
	payload := ExportPayload{
		Version: 1,
		Driver:  "gorm",
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

// ExportDB 导出数据库
// GET /api/v1/db/export?format=db|sql|json
func (h *DBHandler) ExportDB(c *gin.Context) {
	format := c.DefaultQuery("format", "db")

	switch format {
	case "sql":
		dump, err := h.dumpSQL()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "导出SQL失败: " + err.Error()})
			return
		}
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Transfer-Encoding", "binary")
		c.Header("Content-Disposition", `attachment; filename="history.sql"`)
		c.Data(http.StatusOK, "application/octet-stream", []byte(dump))
	case "json":
		data, err := h.exportJSON()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "导出JSON失败: " + err.Error()})
			return
		}
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Transfer-Encoding", "binary")
		c.Header("Content-Disposition", `attachment; filename="history.json"`)
		c.Data(http.StatusOK, "application/json", data)
	default:
		c.FileAttachment(h.dbPath, "history.db")
	}
}

func (h *DBHandler) restoreJSON(data []byte) error {
	var payload ExportPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("JSON 解析失败: %w", err)
	}
	if payload.Version != 1 {
		return fmt.Errorf("不支持的版本号: %d", payload.Version)
	}

	return h.gormDB.Transaction(func(tx *gorm.DB) error {
		// 反序清空（先删依赖表，再删主表）
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
				if err := tx.Table(table).Create(&row).Error; err != nil {
					return fmt.Errorf("恢复表 %s 失败: %w", table, err)
				}
			}
		}
		return nil
	})
}

// POST /api/v1/db/restore — 支持 .db/.sqlite/.sql 和 .json 文件
func (h *DBHandler) RestoreDB(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传文件"})
		return
	}

	tmpPath := h.dbPath + ".restore.tmp"
	if err := c.SaveUploadedFile(file, tmpPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存上传文件失败: " + err.Error()})
		return
	}
	defer os.Remove(tmpPath)

	if strings.HasSuffix(file.Filename, ".json") {
		content, err := os.ReadFile(tmpPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取JSON文件失败: " + err.Error()})
			return
		}
		if err := h.restoreJSON(content); err != nil {
			log.Printf("[DB] JSON 恢复失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "JSON恢复失败: " + err.Error()})
			return
		}
		log.Printf("[DB] JSON 恢复成功 (from: %s)", file.Filename)
		c.JSON(http.StatusOK, gin.H{"ok": true, "message": "JSON 恢复成功"})
		return
	}

	if strings.HasSuffix(file.Filename, ".sql") {
		content, err := os.ReadFile(tmpPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取SQL文件失败: " + err.Error()})
			return
		}
		if err := h.execSQL(string(content)); err != nil {
			log.Printf("[DB] SQL 执行失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "执行SQL失败: " + err.Error()})
			return
		}
		log.Printf("[DB] SQL 文件执行成功 (from: %s)", file.Filename)
		c.JSON(http.StatusOK, gin.H{"ok": true, "message": "SQL 执行成功"})
		return
	}

	if err := h.replaceFunc(tmpPath); err != nil {
		log.Printf("[DB] 恢复失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "恢复失败: " + err.Error()})
		return
	}

	log.Printf("[DB] 数据库恢复成功 (from: %s)", file.Filename)
	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "数据库恢复成功"})
}

// dumpSQL 生成 SQL 格式完整转储（schema + 数据）
func (h *DBHandler) dumpSQL() (string, error) {
	db := h.getDB()
	var buf strings.Builder

	buf.WriteString("PRAGMA foreign_keys=OFF;\nBEGIN TRANSACTION;\n\n")

	// 1. Schema: CREATE TABLE / INDEX / VIEW / TRIGGER
		rows, err := db.Query(`SELECT sql FROM sqlite_master WHERE sql IS NOT NULL AND name NOT LIKE 'sqlite_%' ORDER BY type DESC, name`)
	if err != nil {
		return "", fmt.Errorf("查询 schema 失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var sql string
		if err := rows.Scan(&sql); err != nil {
			return "", fmt.Errorf("扫描 schema 失败: %w", err)
		}
		// 将 CREATE TABLE 改为 CREATE TABLE IF NOT EXISTS，使 SQL 恢复可重入
		if strings.HasPrefix(strings.TrimSpace(sql), "CREATE TABLE") {
			sql = strings.Replace(sql, "CREATE TABLE", "CREATE TABLE IF NOT EXISTS", 1)
		}
		buf.WriteString(sql)
		buf.WriteString(";\n")
	}
	buf.WriteString("\n")

	// 2. 数据：逐表导出 INSERT
	tables, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return "", fmt.Errorf("查询表列表失败: %w", err)
	}
	defer tables.Close()

	for tables.Next() {
		var name string
		if err := tables.Scan(&name); err != nil {
			return "", fmt.Errorf("扫描表名失败: %w", err)
		}

		colRows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%q)", name))
		if err != nil {
			return "", fmt.Errorf("查询表 %s 结构失败: %w", name, err)
		}

		var cols []string
		for colRows.Next() {
			var cid int
			var colName, colType string
			var notNull int
			var dflt, pk interface{}
			if err := colRows.Scan(&cid, &colName, &colType, &notNull, &dflt, &pk); err != nil {
				colRows.Close()
				return "", fmt.Errorf("扫描列信息失败: %w", err)
			}
			cols = append(cols, colName)
		}
		colRows.Close()

		dataRows, err := db.Query(fmt.Sprintf("SELECT * FROM %q", name))
		if err != nil {
			return "", fmt.Errorf("查询表 %s 数据失败: %w", name, err)
		}

		for dataRows.Next() {
			values := make([]interface{}, len(cols))
			ptrs := make([]interface{}, len(cols))
			for i := range values {
				ptrs[i] = &values[i]
			}
			if err := dataRows.Scan(ptrs...); err != nil {
				dataRows.Close()
				return "", fmt.Errorf("扫描数据行失败: %w", err)
			}

			var parts []string
			for _, v := range values {
				if v == nil {
					parts = append(parts, "NULL")
				} else {
					switch val := v.(type) {
					case []byte:
						parts = append(parts, fmt.Sprintf("X'%x'", val))
					case int64:
						parts = append(parts, strconv.FormatInt(val, 10))
					case float64:
						parts = append(parts, strconv.FormatFloat(val, 'g', -1, 64))
					case bool:
						if val {
							parts = append(parts, "1")
						} else {
							parts = append(parts, "0")
						}
					case string:
						parts = append(parts, fmt.Sprintf("'%s'", strings.ReplaceAll(val, "'", "''")))
					default:
						parts = append(parts, fmt.Sprintf("'%s'", strings.ReplaceAll(fmt.Sprintf("%v", v), "'", "''")))
					}
				}
			}

			quotedCols := make([]string, len(cols))
			for i, c := range cols {
				quotedCols[i] = fmt.Sprintf("%q", c)
			}

			buf.WriteString(fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);\n", name, strings.Join(quotedCols, ", "), strings.Join(parts, ", ")))
		}
		dataRows.Close()
		buf.WriteString("\n")
	}

	buf.WriteString("COMMIT;\n")
	return buf.String(), nil
}

// execSQL 执行 SQL 文件内容（逐条执行，识别字符串字面量，防止分号打断）
func (h *DBHandler) execSQL(sqlContent string) error {
	db := h.getDB()
	statements := splitSQLStmts(sqlContent)
	for _, stmt := range statements {
		// 兼容旧版导出：CREATE TABLE 无 IF NOT EXISTS 时自动补全
		trimmed := strings.TrimSpace(stmt)
		if strings.HasPrefix(trimmed, "CREATE TABLE") && !strings.Contains(trimmed, "IF NOT EXISTS") {
			stmt = strings.Replace(stmt, "CREATE TABLE", "CREATE TABLE IF NOT EXISTS", 1)
		}
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("执行SQL失败: %w", err)
		}
	}
	return nil
}

// splitSQLStmts 按分号分割 SQL 语句，但忽略字符串字面量内部的 ';'
func splitSQLStmts(sql string) []string {
	var stmts []string
	var cur strings.Builder
	inStr := false

	for i := 0; i < len(sql); i++ {
		ch := sql[i]

		// 转义单引号 '' 在字符串内部不切换 inStr
		if ch == '\'' && inStr && i+1 < len(sql) && sql[i+1] == '\'' {
			cur.WriteByte(ch)
			i++
			cur.WriteByte(sql[i])
			continue
		}

		if ch == '\'' {
			inStr = !inStr
			cur.WriteByte(ch)
			continue
		}

		if ch == ';' && !inStr {
			stmt := strings.TrimSpace(cur.String())
			if stmt != "" {
				stmts = append(stmts, stmt)
			}
			cur.Reset()
			continue
		}

		cur.WriteByte(ch)
	}

	if rem := strings.TrimSpace(cur.String()); rem != "" {
		stmts = append(stmts, rem)
	}

	return stmts
}
