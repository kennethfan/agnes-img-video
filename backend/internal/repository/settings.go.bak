package repository

import (
	"database/sql"

	"github.com/agnes-image-tool/backend/internal/model"
)

// SettingsRepo 存储设置
type SettingsRepo struct {
	db *sql.DB
}

func NewSettingsRepo(db *sql.DB) *SettingsRepo {
	return &SettingsRepo{db: db}
}

func (r *SettingsRepo) InitTable() error {
	_, err := r.db.Exec(`CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`)
	return err
}

func (r *SettingsRepo) GetSettings() (*model.Settings, error) {
	s := &model.Settings{
		StorageTarget:   "local",
		LocalImageDir:   "images",
		LocalVideoDir:   "videos",
		GithubImagePath: "outputs/images",
		GithubVideoPath: "outputs/videos",
	}

	rows, err := r.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		switch key {
		case "storage_target":
			s.StorageTarget = value
		case "local_image_dir":
			s.LocalImageDir = value
		case "local_video_dir":
			s.LocalVideoDir = value
		case "github_image_path":
			s.GithubImagePath = value
		case "github_video_path":
			s.GithubVideoPath = value
		}
	}
	return s, nil
}

func (r *SettingsRepo) UpdateSettings(s *model.Settings) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	upsert := `INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`
	pairs := map[string]string{
		"storage_target":   s.StorageTarget,
		"local_image_dir":  s.LocalImageDir,
		"local_video_dir":  s.LocalVideoDir,
		"github_image_path": s.GithubImagePath,
		"github_video_path": s.GithubVideoPath,
	}
	for k, v := range pairs {
		if _, err := tx.Exec(upsert, k, v); err != nil {
			return err
		}
	}
	return tx.Commit()
}
