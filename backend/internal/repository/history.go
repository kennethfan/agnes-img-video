package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/agnes-image-tool/backend/internal/model"
	_ "github.com/mattn/go-sqlite3"
)

type HistoryRepo struct {
	db *sql.DB
}

func NewHistoryRepo(dbPath string) (*HistoryRepo, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// 创建表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS history (
			id     INTEGER PRIMARY KEY AUTOINCREMENT,
			time   TEXT NOT NULL,
			mode   TEXT NOT NULL,
			prompt TEXT NOT NULL,
			images TEXT NOT NULL DEFAULT '[]',
			extra  TEXT
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("创建历史记录表失败: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS favorites (
			history_id INTEGER PRIMARY KEY,
			created_at TEXT DEFAULT (datetime('now')),
			FOREIGN KEY (history_id) REFERENCES history(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("创建收藏表失败: %w", err)
	}

	return &HistoryRepo{db: db}, nil
}

func (r *HistoryRepo) Close() error {
	return r.db.Close()
}

func (r *HistoryRepo) DB() *sql.DB {
	return r.db
}

func (r *HistoryRepo) InsertRecord(prompt string, images []string, mode string, extra any) (int64, error) {
	imagesJSON, err := json.Marshal(images)
	if err != nil {
		return 0, fmt.Errorf("序列化图片列表失败: %w", err)
	}

	var extraJSON *string
	if extra != nil {
		b, err := json.Marshal(extra)
		if err != nil {
			return 0, fmt.Errorf("序列化 extra 失败: %w", err)
		}
		s := string(b)
		extraJSON = &s
	}

	res, err := r.db.Exec(
		"INSERT INTO history (time, mode, prompt, images, extra) VALUES (datetime('now','localtime'), ?, ?, ?, ?)",
		mode, prompt, string(imagesJSON), extraJSON,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *HistoryRepo) insertRecordAt(time string, prompt string, images []string, mode string, extra any) error {
	imagesJSON, err := json.Marshal(images)
	if err != nil {
		return fmt.Errorf("序列化图片列表失败: %w", err)
	}

	var extraJSON *string
	if extra != nil {
		b, err := json.Marshal(extra)
		if err != nil {
			return fmt.Errorf("序列化 extra 失败: %w", err)
		}
		s := string(b)
		extraJSON = &s
	}

	_, err = r.db.Exec(
		"INSERT INTO history (time, mode, prompt, images, extra) VALUES (?, ?, ?, ?, ?)",
		time, mode, prompt, string(imagesJSON), extraJSON,
	)
	return err
}

func (r *HistoryRepo) ImportFromJSON(jsonPath string) (int, error) {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil // 文件不存在，跳过
		}
		return 0, fmt.Errorf("读取 JSON 文件失败: %w", err)
	}

	var records []model.HistoryRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return 0, fmt.Errorf("解析 JSON 失败: %w", err)
	}

	if len(records) == 0 {
		return 0, nil
	}

	imported := 0
	for _, rec := range records {
		if rec.Time == "" {
			continue
		}
		if err := r.insertRecordAt(rec.Time, rec.Prompt, rec.Images, rec.Mode, rec.Extra); err != nil {
			log.Printf("[Migration] 导入记录失败 (time=%s, mode=%s): %v", rec.Time, rec.Mode, err)
			continue
		}
		imported++
	}

	// 重命名已导入的文件，防止二次导入
	if imported > 0 {
		backupPath := jsonPath + ".migrated"
		if err := os.Rename(jsonPath, backupPath); err != nil {
			log.Printf("[Migration] 备份文件失败: %v", err)
		}
	}

	return imported, nil
}

func scanRecords(rows *sql.Rows) ([]model.HistoryRecord, error) {
	var records []model.HistoryRecord
	for rows.Next() {
		var (
			id                     int64
			time, mode, prompt     string
			imagesJSON             string
			extraJSON              *string
		)
		if err := rows.Scan(&id, &time, &mode, &prompt, &imagesJSON, &extraJSON); err != nil {
			return nil, err
		}

		var images []string
		json.Unmarshal([]byte(imagesJSON), &images)
		if images == nil {
			images = []string{}
		}

		rec := model.HistoryRecord{
			ID:     id,
			Time:   time,
			Mode:   mode,
			Prompt: prompt,
			Images: images,
		}
		if extraJSON != nil {
			var extra any
			if err := json.Unmarshal([]byte(*extraJSON), &extra); err == nil {
				rec.Extra = extra
			}
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

func (r *HistoryRepo) GetRecords(limit int) ([]model.HistoryRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.Query(
		"SELECT id, time, mode, prompt, images, extra FROM history ORDER BY id DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRecords(rows)
}

func (r *HistoryRepo) GetRecordsPaginated(page, perPage int, assetType, search string, favIDs map[int64]bool) ([]model.HistoryRecord, int, error) {
	var conditions []string
	var args []any

	if assetType != "" {
		switch assetType {
		case "image":
			conditions = append(conditions, "mode IN ('text2image','image2image','batch')")
		case "video":
			conditions = append(conditions, "mode IN ('text2video','image2video','multi_image_video')")
		}
	}

	if search != "" {
		conditions = append(conditions, "prompt LIKE ?")
		args = append(args, "%"+search+"%")
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM history" + whereClause
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []model.HistoryRecord{}, 0, nil
	}

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	dataQuery := "SELECT id, time, mode, prompt, images, extra FROM history" + whereClause + " ORDER BY id DESC LIMIT ? OFFSET ?"
	args = append(args, perPage, offset)
	rows, err := r.db.Query(dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	records, err := scanRecords(rows)
	if err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

func (r *HistoryRepo) GetRecordsByIDs(ids []int64) ([]model.HistoryRecord, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	q := fmt.Sprintf("SELECT id, time, mode, prompt, images, extra FROM history WHERE id IN (%s) ORDER BY id DESC", strings.Join(placeholders, ","))
	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRecords(rows)
}

func (r *HistoryRepo) DeleteRecord(id int64) error {
	_, err := r.db.Exec("DELETE FROM history WHERE id = ?", id)
	return err
}

func (r *HistoryRepo) DeleteRecords(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	q := fmt.Sprintf("DELETE FROM history WHERE id IN (%s)", strings.Join(placeholders, ","))
	_, err := r.db.Exec(q, args...)
	return err
}

func (r *HistoryRepo) ClearRecords() error {
	_, err := r.db.Exec("DELETE FROM history")
	return err
}

// PendingVideoInfo 待恢复的视频任务信息
type PendingVideoInfo struct {
	ID      int64
	TaskID  string
	Prompt  string
	Mode    string
}

// FindByTaskId 通过 extra.taskId 查找历史记录 ID
func (r *HistoryRepo) FindByTaskId(taskId string) (int64, error) {
	var id int64
	err := r.db.QueryRow(
		"SELECT id FROM history WHERE json_extract(extra, '$.taskId') = ? ORDER BY id DESC LIMIT 1",
		taskId,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// UpdateRecordImages 更新历史记录的图片列表
func (r *HistoryRepo) UpdateRecordImages(id int64, images []string) error {
	imagesJSON, err := json.Marshal(images)
	if err != nil {
		return fmt.Errorf("序列化图片列表失败: %w", err)
	}
	_, err = r.db.Exec("UPDATE history SET images = ? WHERE id = ?", string(imagesJSON), id)
	return err
}

// FindPendingVideos 查找等待完成的视频任务（images 为空且有 taskId）
func (r *HistoryRepo) FindPendingVideos() ([]PendingVideoInfo, error) {
	rows, err := r.db.Query(`
		SELECT id, mode, prompt, extra FROM history
		WHERE images = '[]' AND extra IS NOT NULL AND extra != ''
		ORDER BY id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []PendingVideoInfo
	for rows.Next() {
		var id int64
		var mode, prompt string
		var extraJSON string
		if err := rows.Scan(&id, &mode, &prompt, &extraJSON); err != nil {
			continue
		}
		// 只筛选视频类型的记录
		if !strings.HasSuffix(mode, "video") && mode != "video" {
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
		results = append(results, PendingVideoInfo{
			ID:     id,
			TaskID: taskID,
			Prompt: prompt,
			Mode:   mode,
		})
	}
	return results, rows.Err()
}

func (r *HistoryRepo) TrimRecords(max int) error {
	_, err := r.db.Exec(fmt.Sprintf(
		"DELETE FROM history WHERE id NOT IN (SELECT id FROM history ORDER BY id DESC LIMIT %d)", max,
	))
	return err
}

func (r *HistoryRepo) ToggleFavorite(historyID int64, favorite bool) error {
	if favorite {
		_, err := r.db.Exec("INSERT OR IGNORE INTO favorites (history_id) VALUES (?)", historyID)
		return err
	}
	_, err := r.db.Exec("DELETE FROM favorites WHERE history_id = ?", historyID)
	return err
}

func (r *HistoryRepo) GetFavoriteIDs() (map[int64]bool, error) {
	rows, err := r.db.Query("SELECT history_id FROM favorites")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result[id] = true
	}
	return result, rows.Err()
}
