package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"

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

	return &HistoryRepo{db: db}, nil
}

func (r *HistoryRepo) Close() error {
	return r.db.Close()
}

func (r *HistoryRepo) InsertRecord(prompt string, images []string, mode string, extra any) error {
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
		"INSERT INTO history (time, mode, prompt, images, extra) VALUES (datetime('now','localtime'), ?, ?, ?, ?)",
		mode, prompt, string(imagesJSON), extraJSON,
	)
	return err
}

func (r *HistoryRepo) GetRecords(limit int) ([]model.HistoryRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.Query(
		"SELECT time, mode, prompt, images, extra FROM history ORDER BY id DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []model.HistoryRecord
	for rows.Next() {
		var (
			time, mode, prompt string
			imagesJSON         string
			extraJSON          *string
		)
		if err := rows.Scan(&time, &mode, &prompt, &imagesJSON, &extraJSON); err != nil {
			return nil, err
		}

		var images []string
		json.Unmarshal([]byte(imagesJSON), &images)
		if images == nil {
			images = []string{}
		}

		rec := model.HistoryRecord{
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

func (r *HistoryRepo) ClearRecords() error {
	_, err := r.db.Exec("DELETE FROM history")
	return err
}

func (r *HistoryRepo) TrimRecords(max int) error {
	_, err := r.db.Exec(fmt.Sprintf(
		"DELETE FROM history WHERE id NOT IN (SELECT id FROM history ORDER BY id DESC LIMIT %d)", max,
	))
	return err
}
