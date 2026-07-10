package gorm

import (
	"testing"

	"github.com/agnes-image-tool/backend/internal/model"
)

func TestSettingsGetDefaults(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&Setting{})
	repo := NewSettingsRepository(db)

	s, err := repo.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings failed: %v", err)
	}
	if s.StorageTarget != "" {
		t.Fatalf("expected empty default storage_target, got %q", s.StorageTarget)
	}
	if s.LocalImageDir != "images" {
		t.Fatalf("expected default local_image_dir 'images', got %q", s.LocalImageDir)
	}
	if s.LocalVideoDir != "videos" {
		t.Fatalf("expected default local_video_dir 'videos', got %q", s.LocalVideoDir)
	}
}

func TestSettingsUpdateAndGet(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&Setting{})
	repo := NewSettingsRepository(db)

	input := &model.Settings{
		StorageTarget:   "github",
		LocalImageDir:   "my_images",
		LocalVideoDir:   "my_videos",
		GithubImagePath: "custom/images",
		GithubVideoPath: "custom/videos",
	}
	if err := repo.UpdateSettings(input); err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}

	s, err := repo.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings failed: %v", err)
	}
	if s.StorageTarget != "github" {
		t.Fatalf("expected 'github', got %q", s.StorageTarget)
	}
	if s.GithubImagePath != "custom/images" {
		t.Fatalf("expected 'custom/images', got %q", s.GithubImagePath)
	}
}

func TestSettingsPartialUpdate(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&Setting{})
	repo := NewSettingsRepository(db)

	if err := repo.UpdateSettings(&model.Settings{
		StorageTarget: "s3",
	}); err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}

	s, _ := repo.GetSettings()
	if s.StorageTarget != "s3" {
		t.Fatalf("expected 's3', got %q", s.StorageTarget)
	}
	if s.LocalImageDir != "" {
		t.Fatalf("expected empty local_image_dir, got %q", s.LocalImageDir)
	}
}

func TestSettingsIdempotentUpdate(t *testing.T) {
	db := openDBMemory(t)
	db.AutoMigrate(&Setting{})
	repo := NewSettingsRepository(db)

	input := &model.Settings{StorageTarget: "local"}
	for range 3 {
		if err := repo.UpdateSettings(input); err != nil {
			t.Fatalf("repeated UpdateSettings failed: %v", err)
		}
	}

	s, _ := repo.GetSettings()
	if s.StorageTarget != "local" {
		t.Fatalf("expected 'local', got %q", s.StorageTarget)
	}
}
