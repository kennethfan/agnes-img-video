//go:build ignore

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

// 一次性迁移脚本：将 history 表中 pending 的视频任务迁移到 tasks 表
func main() {
	dbPath := "history.db"
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}
	defer db.Close()

	// 确保 tasks 表存在
	db.Exec(`CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY, type TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'pending',
		params TEXT NOT NULL, result TEXT, progress INTEGER NOT NULL DEFAULT 0,
		error TEXT, retry_count INTEGER NOT NULL DEFAULT 0,
		created_at TEXT, updated_at TEXT, completed_at TEXT
	)`)

	rows, err := db.Query(`
		SELECT id, mode, prompt, extra FROM history
		WHERE images = '[]' AND extra IS NOT NULL AND extra != ''
		ORDER BY id ASC
	`)
	if err != nil {
		log.Fatalf("查询失败: %v", err)
	}
	defer rows.Close()

	migrated := 0
	for rows.Next() {
		var id int64
		var mode, prompt, extraJSON string
		if err := rows.Scan(&id, &mode, &prompt, &extraJSON); err != nil {
			continue
		}
		var extra map[string]any
		if err := json.Unmarshal([]byte(extraJSON), &extra); err != nil {
			continue
		}
		taskID, _ := extra["taskId"].(string)
		if taskID == "" {
			continue
		}

		// 检查是否已迁移
		var count int
		db.QueryRow("SELECT COUNT(*) FROM tasks WHERE id = ?", "hist_"+taskID).Scan(&count)
		if count > 0 {
			continue
		}

		params, _ := json.Marshal(map[string]any{
			"taskId": taskID,
			"prompt": prompt,
		})
		_, err := db.Exec(
			"INSERT INTO tasks (id, type, status, params, progress, created_at, updated_at) VALUES (?, ?, 'pending', ?, 0, datetime('now','localtime'), datetime('now','localtime'))",
			"hist_"+taskID, mode, string(params),
		)
		if err != nil {
			log.Printf("迁移记录 %d 失败: %v", id, err)
			continue
		}
		migrated++
		fmt.Printf("已迁移: history_id=%d taskId=%s type=%s\n", id, taskID, mode)
	}

	fmt.Printf("迁移完成: 共 %d 条", migrated)
}
