package gorm

import (
	"github.com/agnes-image-tool/backend/internal/model"
	"gorm.io/gorm"
)

type SettingsRepository struct {
	db *gorm.DB
}

func NewSettingsRepository(db *gorm.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

func (r *SettingsRepository) GetSettings() (*model.Settings, error) {
	s := &model.Settings{
		StorageTarget:   "local",
		LocalImageDir:   "images",
		LocalVideoDir:   "videos",
		GithubImagePath: "outputs/images",
		GithubVideoPath: "outputs/videos",
	}
	var settings []Setting
	if err := r.db.Find(&settings).Error; err != nil {
		return nil, err
	}
	for _, kv := range settings {
		switch kv.Key {
		case "storage_target":
			s.StorageTarget = kv.Value
		case "local_image_dir":
			s.LocalImageDir = kv.Value
		case "local_video_dir":
			s.LocalVideoDir = kv.Value
		case "github_image_path":
			s.GithubImagePath = kv.Value
		case "github_video_path":
			s.GithubVideoPath = kv.Value
		}
	}
	return s, nil
}

func (r *SettingsRepository) UpdateSettings(s *model.Settings) error {
	pairs := map[string]string{
		"storage_target":    s.StorageTarget,
		"local_image_dir":   s.LocalImageDir,
		"local_video_dir":   s.LocalVideoDir,
		"github_image_path": s.GithubImagePath,
		"github_video_path": s.GithubVideoPath,
	}
	for k, v := range pairs {
		if err := r.db.Where("key = ?", k).Assign(Setting{Value: v}).FirstOrCreate(&Setting{Key: k}).Error; err != nil {
			return err
		}
	}
	return nil
}
